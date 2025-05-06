package eventbus

import (
	"context"
	"encoding/json"
	"github.com/ArsiHien/pastebin-ms/create-service/internal/domain/paste"
	amqp "github.com/rabbitmq/amqp091-go"
	"time"
)

type RabbitMQPublisher struct {
	channel *amqp.Channel
}

func NewRabbitMQPublisher(conn *amqp.Connection) (*RabbitMQPublisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	err = ch.ExchangeDeclare("pastebin_events", "topic", true, false, false, false, nil)
	if err != nil {
		return nil, err
	}
	return &RabbitMQPublisher{channel: ch}, nil
}

func (p *RabbitMQPublisher) PublishPasteCreated(paste *paste.Paste) error {
	type PasteMessage struct {
		ID         string    `json:"id"`
		URL        string    `json:"url"`
		Content    string    `json:"content"`
		CreatedAt  time.Time `json:"created_at"`
		PolicyType string    `json:"policy_type"`
		Duration   string    `json:"duration"`
	}

	message := PasteMessage{
		ID:         paste.ID,
		URL:        paste.URL,
		Content:    paste.Content,
		CreatedAt:  paste.CreatedAt,
		PolicyType: string(paste.ExpirationPolicy.Type),
		Duration:   paste.ExpirationPolicy.Duration,
	}

	body, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return p.channel.Publish(
		"pastebin_events", "paste.created", false, false,
		amqp.Publishing{ContentType: "application/json", Body: body},
	)
}

func (p *RabbitMQPublisher) PublishPasteSave(ctx context.Context, pasteData []byte) error {
	err := p.channel.PublishWithContext(
		ctx,
		"pastebin_events",
		"paste.save",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        pasteData,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func (p *RabbitMQPublisher) Close() error {
	return p.channel.Close()
}
