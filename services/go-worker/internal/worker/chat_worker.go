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

type ChatWorker struct {
	repo   *database.Repository
	redis  *redis.Client
	logger *logging.Logger
}

type ChatPayload struct {
	AppToken   string `json:"app_token"`
	ChatNumber int    `json:"chat_number"`
}

func NewChatWorker(db *database.Database, logger *logging.Logger) *ChatWorker {
	return &ChatWorker{
		repo:   database.NewRepository(db.MySqlDB),
		redis:  db.RedisDB,
		logger: logger.WithPrefix("ChatWorker"),
	}
}

func (w *ChatWorker) HandleMessage(delivery amqp.Delivery) error {
	var payload ChatPayload
	if err := queue.ParseMessageBody(delivery, &payload); err != nil {
		w.logger.Error("Failed to parse message: %v", err)
		return nil
	}

	w.logger.Info("Processing: app_token=%s, chat_number=%d",
		payload.AppToken, payload.ChatNumber)

	if payload.AppToken == "" || payload.ChatNumber == 0 {
		w.logger.Error("Missing required fields")
		return nil
	}

	application, err := w.repo.FindApplicationByToken(payload.AppToken)
	if err != nil {
		w.logger.Error("Application not found: %s - %v", payload.AppToken, err)
		return nil
	}

	existingChat, err := w.repo.FindChatByApplicationAndNumber(application.ID, payload.ChatNumber)
	if err == nil {
		w.logger.Info("Chat already exists: id=%d", existingChat.ID)
		return nil
	}

	chat := &model.Chat{
		ApplicationID: application.ID,
		Number:        payload.ChatNumber,
		MessagesCount: 0,
	}

	if err := w.repo.CreateChat(chat); err != nil {
		if isDuplicateError(err) {
			w.logger.Error("Chat already exists (race condition)")
			return nil
		}
		w.logger.Error("Failed to create chat: %v", err)
		return err
	}

	w.logger.Info("Chat created: id=%d, number=%d, app=%s",
		chat.ID, chat.Number, payload.AppToken)

	// Increment delta counter for reconciliation
	w.incrementCounter(fmt.Sprintf("delta:app:%d:chats", application.ID))

	return nil
}

func (w *ChatWorker) incrementCounter(key string) {
	ctx := context.Background()
	if err := w.redis.Incr(ctx, key).Err(); err != nil {
		w.logger.Error("Failed to increment counter %s: %v", key, err)
	}
}
