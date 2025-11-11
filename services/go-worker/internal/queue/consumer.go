package queue

import (
	"encoding/json"
	"go-worker/internal/logging"

	amqp "github.com/rabbitmq/amqp091-go"
)

type MessageHandler func(delivery amqp.Delivery) error

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	logger  *logging.Logger
}

func NewConsumer(amqpConn *AMQP, logger *logging.Logger) *Consumer {
	return &Consumer{
		conn:    amqpConn.conn,
		channel: amqpConn.channel,
		logger:  logger,
	}
}

func (c *Consumer) ConsumeQueue(queueName string, handler MessageHandler) error {
	c.logger.Info("Starting consumer for queue: %s", queueName)

	_, err := c.channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		c.logger.Error("Failed to declare queue %s: %v", queueName, err)
		return err
	}

	err = c.channel.Qos(
		1,
		0,
		false,
	)
	if err != nil {
		c.logger.Error("Failed to set QoS: %v", err)
		return err
	}

	msgs, err := c.channel.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		c.logger.Error("Failed to register consumer: %v", err)
		return err
	}

	c.logger.Info("Consumer started for queue: %s", queueName)

	go func() {
		for msg := range msgs {
			c.logger.Info("[%s] Received message", queueName)
			err := handler(msg)
			if err != nil {
				c.logger.Error("[%s] Error processing message: %v", queueName, err)
				msg.Nack(false, true)
			} else {
				msg.Ack(false)
				c.logger.Info("[%s] Message processed successfully", queueName)
			}
		}
	}()

	return nil
}

func (c *Consumer) Close() error {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}

	return nil
}

func ParseMessageBody(delivery amqp.Delivery, target interface{}) error {
	return json.Unmarshal(delivery.Body, target)
}
