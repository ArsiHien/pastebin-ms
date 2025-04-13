package eventbus

import (
	"encoding/json"
	"github.com/ArsiHien/pastebin-ms/create-service/internal/domain/paste"
	amqp "github.com/rabbitmq/amqp091-go"
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
	body, _ := json.Marshal(paste)
	return p.channel.Publish(
		"pastebin_events", "paste.created", false, false,
		amqp.Publishing{ContentType: "application/json", Body: body},
	)
}

func (p *RabbitMQPublisher) Close() error {
	return p.channel.Close()
}
