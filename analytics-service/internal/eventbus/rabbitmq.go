package eventbus

import (
	"analytics-service/internal/metrics"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"analytics-service/internal/domain/analytics"
	"github.com/rabbitmq/amqp091-go"
)

type EventConsumer interface {
	Consume(ctx context.Context, handler func(analytics.PasteViewedEvent) error) error
	Close() error
}

type RabbitMQConsumer struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
	queue   string
}

func NewRabbitMQConsumer(conn *amqp091.Connection, queue string) (*RabbitMQConsumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	if _, err := ch.QueueDeclare(
		queue,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,
	); err != nil {
		ch.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	if err := ch.QueueBind(
		queue,
		"paste.viewed",    // routing key
		"pastebin_events", // exchange
		false,
		nil,
	); err != nil {
		ch.Close()
		return nil, fmt.Errorf("failed to bind queue: %w", err)
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
		"",    // consumer tag
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
			if msg.RoutingKey != "paste.viewed" {
				log.Printf("Skipping unrelated event: %s", msg.RoutingKey)
				_ = msg.Ack(false)
				continue
			}

			var event analytics.PasteViewedEvent
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				log.Printf("Failed to unmarshal event: %v", err)
				_ = msg.Nack(false, true)
				continue
			}

			if err := handler(event); err != nil {
				log.Printf("Handler error: %v", err)
				_ = msg.Nack(false, true)
				continue
			}

			metrics.PasteEventDuration.Observe(float64(time.Since(event.ViewedAt)))

			if err := msg.Ack(false); err != nil {
				return fmt.Errorf("failed to ack message: %w", err)
			}
		}
	}
}

func (c *RabbitMQConsumer) Close() error {
	return c.channel.Close()
}
