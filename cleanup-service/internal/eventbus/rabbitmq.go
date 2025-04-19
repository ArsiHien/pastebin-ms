package eventbus

import (
	"cleanup-service/internal/domain/paste"
	"context"
	"encoding/json"
	"fmt"
	"github.com/rabbitmq/amqp091-go"
)

// EventConsumer defines operations for consuming events
type EventConsumer interface {
	Consume(ctx context.Context, handler func(event interface{}) error) error
	Close() error
}

// RabbitMQConsumer implements EventConsumer with RabbitMQ
type RabbitMQConsumer struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
}

func NewRabbitMQConsumer(conn *amqp091.Connection) (*RabbitMQConsumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	err = ch.ExchangeDeclare(
		"pastebin_events", // Update exchange name
		"topic",
		true,  // durable
		false, // autoDelete
		false, // internal
		false, // noWait
		nil,
	)
	if err != nil {
		err := ch.Close()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	q, err := ch.QueueDeclare(
		"cleanup.events",
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,
	)
	if err != nil {
		err := ch.Close()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Update routing keys to match the new publisher's routing keys
	for _, key := range []string{"paste.created", "paste.burn_after_read_paste_viewed"} {
		err = ch.QueueBind(
			q.Name,
			key,
			"pastebin_events", // Update exchange name
			false,
			nil,
		)
		if err != nil {
			err := ch.Close()
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("failed to bind queue: %w", err)
		}
	}

	return &RabbitMQConsumer{
		conn:    conn,
		channel: ch,
	}, nil
}

func (c *RabbitMQConsumer) Consume(ctx context.Context, handler func(event interface{}) error) error {
	msgs, err := c.channel.Consume(
		"cleanup.events",
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
			var event interface{}
			switch msg.RoutingKey {
			case "paste.created":
				var e paste.CreatedEvent
				if err := json.Unmarshal(msg.Body, &e); err != nil {
					err := msg.Nack(false, true)
					if err != nil {
						return err
					}
					continue
				}
				event = e
			case "paste.viewed":
				var e paste.ViewedEvent
				if err := json.Unmarshal(msg.Body, &e); err != nil {
					err := msg.Nack(false, true)
					if err != nil {
						return err
					}
					continue
				}
				event = e
			case "paste.burn_after_read_paste_viewed":
				var e paste.BurnAfterReadPasteViewedEvent
				if err := json.Unmarshal(msg.Body, &e); err != nil {
					err := msg.Nack(false, true)
					if err != nil {
						return err
					}
					continue
				}
				event = e
			default:
				err := msg.Nack(false, true)
				if err != nil {
					return err
				}
				continue
			}

			if err := handler(event); err != nil {
				err := msg.Nack(false, true)
				if err != nil {
					return err
				}
				continue
			}

			err := msg.Ack(false)
			if err != nil {
				return err
			}
		}
	}
}

func (c *RabbitMQConsumer) Close() error {
	if err := c.channel.Close(); err != nil {
		return fmt.Errorf("failed to close channel: %w", err)
	}
	return nil
}
