package worker

import (
	"context"
	"fmt"
	"go-worker/internal/database"
	"go-worker/internal/logging"
	"go-worker/internal/model"
	"go-worker/internal/queue"

	"github.com/go-redis/redis/v8"
	amqp "github.com/rabbitmq/amqp091-go"
)

type MessageWorker struct {
	repo   *database.Repository
	redis  *redis.Client
	amqp   *queue.AMQP
	logger *logging.Logger
}

type MessagePayload struct {
	AppToken      string `json:"app_token"`
	ChatNumber    int    `json:"chat_number"`
	MessageNumber int    `json:"message_number"`
	Content       string `json:"content"`
}

func NewMessageWorker(db *database.Database, amqp *queue.AMQP, logger *logging.Logger) *MessageWorker {
	return &MessageWorker{
		repo:   database.NewRepository(db.MySqlDB),
		redis:  db.RedisDB,
		amqp:   amqp,
		logger: logger.WithPrefix("MessageWorker"),
	}
}

func (w *MessageWorker) HandleMessage(delivery amqp.Delivery) error {
	var payload MessagePayload
	if err := queue.ParseMessageBody(delivery, &payload); err != nil {
		w.logger.Error("Failed to parse message: %v", err)
		return nil
	}

	w.logger.Info("Processing: app=%s, chat=%d, msg=%d",
		payload.AppToken, payload.ChatNumber, payload.MessageNumber)

	if payload.AppToken == "" || payload.ChatNumber == 0 ||
		payload.MessageNumber == 0 || payload.Content == "" {
		w.logger.Error("Missing required fields")
		return nil
	}

	application, err := w.repo.FindApplicationByToken(payload.AppToken)
	if err != nil {
		w.logger.Error("Application not found: %s - %v", payload.AppToken, err)
		return nil
	}

	chat, err := w.repo.FindChatByApplicationAndNumber(application.ID, payload.ChatNumber)
	if err != nil {
		w.logger.Error("Chat not found: app=%s, chat_number=%d - %v",
			payload.AppToken, payload.ChatNumber, err)
		return fmt.Errorf("chat not found") // Requeue - chat might be processing
	}

	existingMessage, err := w.repo.FindMessageByChatAndNumber(chat.ID, payload.MessageNumber)
	if err == nil {
		w.logger.Info("Message already exists: id=%d", existingMessage.ID)
		return nil
	}

	message := &model.Message{
		ChatID:  chat.ID,
		Number:  payload.MessageNumber,
		Content: payload.Content,
	}

	if err := w.repo.CreateMessage(message); err != nil {
		if isDuplicateError(err) {
			w.logger.Error("Message already exists (race condition)")
			return nil
		}
		w.logger.Error("Failed to create message: %v", err)
		return err
	}

	w.logger.Info("Message created: id=%d, number=%d, chat=%d",
		message.ID, message.Number, chat.ID)

	// Increment delta counter for reconciliation
	w.incrementCounter(fmt.Sprintf("delta:chat:%d:messages", chat.ID))

	// Queue for Elasticsearch indexing
	if err := w.queueForIndexing(message, chat, application); err != nil {
		w.logger.Error("Failed to queue for indexing: %v", err)
	}

	return nil
}

func (w *MessageWorker) incrementCounter(key string) {
	ctx := context.Background()
	if err := w.redis.Incr(ctx, key).Err(); err != nil {
		w.logger.Error("Failed to increment counter %s: %v", key, err)
	}
}

func (w *MessageWorker) queueForIndexing(message *model.Message, chat *model.Chat, app *model.Application) error {
	indexPayload := map[string]interface{}{
		"message_id":        message.ID,
		"application_id":    app.ID,
		"application_token": app.Token,
		"application_name":  app.Name,
		"chat_id":           chat.ID,
		"chat_number":       chat.Number,
		"message_number":    message.Number,
		"content":           message.Content,
		"created_at":        message.CreatedAt,
	}

	return w.amqp.Publish(queue.IndexingQueue, indexPayload)
}
