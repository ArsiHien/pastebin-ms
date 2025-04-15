package eventbus

import (
	"context"
	"encoding/json"
	"fmt"

	"cleanup-service/internal/domain/paste"
	"github.com/rabbitmq/amqp091-go"
)

// EventPublisher defines operations for publishing events
type EventPublisher interface {
	PublishPasteDeleted(ctx context.Context, event paste.PasteDeletedEvent) error
	Close() error
}

// EventConsumer defines operations for consuming events
type EventConsumer interface {
	Consume(ctx context.Context, handler func(event interface{}) error) error
	Close() error
}

// RabbitMQPublisher implements EventPublisher with RabbitMQ
type RabbitMQPublisher struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
}

func NewRabbitMQPublisher(conn *amqp091.Connection) (*RabbitMQPublisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	err = ch.ExchangeDeclare(
		"paste.events",
		"topic",
		true,  // durable
		false, // autoDelete
		false, // internal
		false, // noWait
		nil,
	)
	if err != nil {
		ch.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	return &RabbitMQPublisher{
		conn:    conn,
		channel: ch,
	}, nil
}

func (p *RabbitMQPublisher) PublishPasteDeleted(ctx context.Context, event paste.PasteDeletedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = p.channel.PublishWithContext(
		ctx,
		"paste.events",
		"paste.deleted",
		false, // mandatory
		false, // immediate
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

func (p *RabbitMQPublisher) Close() error {
	if err := p.channel.Close(); err != nil {
		return fmt.Errorf("failed to close channel: %w", err)
	}
	return nil
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
		"paste.events",
		"topic",
		true,  // durable
		false, // autoDelete
		false, // internal
		false, // noWait
		nil,
	)
	if err != nil {
		ch.Close()
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
		ch.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	for _, key := range []string{"paste.created", "paste.viewed"} {
		err = ch.QueueBind(
			q.Name,
			key,
			"paste.events",
			false,
			nil,
		)
		if err != nil {
			ch.Close()
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
				var e paste.PasteCreatedEvent
				if err := json.Unmarshal(msg.Body, &e); err != nil {
					msg.Nack(false, true)
					continue
				}
				event = e
			case "paste.viewed":
				var e paste.PasteViewedEvent
				if err := json.Unmarshal(msg.Body, &e); err != nil {
					msg.Nack(false, true)
					continue
				}
				event = e
			default:
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
