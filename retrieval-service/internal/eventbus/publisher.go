package eventbus

import (
	"encoding/json"
	amqp "github.com/rabbitmq/amqp091-go"
	"retrieval-service/internal/domain/paste"
)

type RabbitMQPublisher struct {
	channel *amqp.Channel
}

func NewRabbitMQPublisher(conn *amqp.Connection) (*RabbitMQPublisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	err = ch.ExchangeDeclare(
		"pastebin_events",
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		err := ch.Close()
		if err != nil {
			return nil, err
		}
		return nil, err
	}

	return &RabbitMQPublisher{channel: ch}, nil
}

func (p *RabbitMQPublisher) PublishPasteViewedEvent(event paste.ViewedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.channel.Publish(
		"pastebin_events",
		"paste.viewed",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

func (p *RabbitMQPublisher) PublishBurnAfterReadPasteViewedEvent(
	event paste.BurnAfterReadPasteViewedEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.channel.Publish(
		"pastebin_events",
		"paste.burn_after_read_paste_viewed",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

func (p *RabbitMQPublisher) Close() error {
	return p.channel.Close()
}
