package queue

import (
	"context"
	"encoding/json"
	"go-chat/internal/config"
	"go-chat/internal/logging"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type QueueType string

const (
	ChatsQueue    QueueType = "chats_queue"
	MessagesQueue QueueType = "messages_queue"
)

type AMQP struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewAMQP(logger *logging.Logger, cfg *config.Config) (*AMQP, error) {
	logger.Info("Connecting to AMQP")
	conn, err := amqp.Dial(cfg.AmqpURL)
	if err != nil {
		logger.Error("Failed to connect to AMQP: %v", err)
		return nil, err
	}
	logger.Info("Connected to AMQP successfully")

	logger.Info("Opening AMQP channel")
	channel, err := conn.Channel()
	if err != nil {
		logger.Error("Failed to open AMQP channel: %v", err)
		return nil, err
	}

	queues := []QueueType{ChatsQueue, MessagesQueue}
	for _, queueType := range queues {
		logger.Info("Declaring AMQP '%s' queue", queueType)
		_, err = channel.QueueDeclare(
			string(queueType),
			true,
			false,
			false,
			false,
			nil,
		)

		if err != nil {
			logger.Error("Failed to declare AMQP queue: %v", err)
			return nil, err
		}
		logger.Info("AMQP '%s' queue declared successfully", queueType)
	}

	return &AMQP{
		conn:    conn,
		channel: channel,
	}, nil
}

func (a *AMQP) PublishMessage(payload interface{}, queueType QueueType) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return a.channel.PublishWithContext(
		ctx,
		"",
		string(queueType),
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}
