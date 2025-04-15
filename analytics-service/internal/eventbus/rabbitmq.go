package eventbus

import (
	"context"
	"encoding/json"
	"fmt"

	"analytics-service/internal/domain/analytics"
	"github.com/rabbitmq/amqp091-go"
)

// EventConsumer defines operations for consuming events
type EventConsumer interface {
	Consume(ctx context.Context, handler func(analytics.PasteViewedEvent) error) error
	Close() error
}

// RabbitMQConsumer implements EventConsumer with RabbitMQ
type RabbitMQConsumer struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
	queue   string
}

// NewRabbitMQConsumer creates a new RabbitMQConsumer
func NewRabbitMQConsumer(conn *amqp091.Connection, queue string) (*RabbitMQConsumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	_, err = ch.QueueDeclare(
		queue,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,
	)
	if err != nil {
		ch.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	return &RabbitMQConsumer{
		conn:    conn,
		channel: ch,
		queue:   queue,
	}, nil
}

func (c *RabbitMQConsumer) Consume(ctx context.Context, handler func(analytics.PasteViewedEvent) error) error {
	msgs, err := c.channel.Consume(
		c.queue,
		"",    // consumer
		false, // autoAck
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to consume messages: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-msgs:
			var event analytics.PasteViewedEvent
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				msg.Nack(false, true)
				continue
			}

			if err := handler(event); err != nil {
				msg.Nack(false, true)
				continue
			}

			msg.Ack(false)
		}
	}
}

func (c *RabbitMQConsumer) Close() error {
	if err := c.channel.Close(); err != nil {
		return fmt.Errorf("failed to close channel: %w", err)
	}
	return nil
}
