package eventbus

import (
	"context"
	"encoding/json"
	"retrieval-service/shared"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/mongo"
	"retrieval-service/internal/domain/paste"
)

type PasteMessage struct {
	ID         string    `json:"id"`
	URL        string    `json:"url"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
	PolicyType string    `json:"policy_type"`
	Duration   string    `json:"duration"`
}

type RabbitMQConsumer struct {
	channel     *amqp.Channel
	collection  *mongo.Collection
	logger      *shared.Logger
	consumerTag string
}

func NewRabbitMQConsumer(conn *amqp.Connection, db *mongo.Database, logger *shared.Logger) (*RabbitMQConsumer, error) {
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

	collection := db.Collection("pastes")

	return &RabbitMQConsumer{
		channel:     ch,
		collection:  collection,
		logger:      logger,
		consumerTag: "paste-creation-consumer",
	}, nil
}

func (c *RabbitMQConsumer) Start() error {
	q, err := c.channel.QueueDeclare(
		"paste_creation_queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	err = c.channel.QueueBind(
		q.Name,
		"paste.created",
		"pastebin_events",
		false,
		nil,
	)
	if err != nil {
		return err
	}

	msgs, err := c.channel.Consume(
		q.Name,
		c.consumerTag,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for d := range msgs {
			c.handleMessage(d)
		}
	}()

	return nil
}

func (c *RabbitMQConsumer) handleMessage(delivery amqp.Delivery) {
	var message PasteMessage
	if err := json.Unmarshal(delivery.Body, &message); err != nil {
		c.logger.Errorf("Failed to unmarshal paste message: %v", err)
		delivery.Nack(false, false)
		return
	}

	// Convert message to paste domain model
	expPolicy := paste.ExpirationPolicy{
		Type: paste.ExpirationPolicyType(message.PolicyType),
	}

	if message.PolicyType == string(paste.TimedExpiration) {
		expPolicy.Duration = message.Duration
	} else if message.PolicyType == string(paste.BurnAfterReadExpiration) {
		expPolicy.IsRead = false
	}

	newPaste := paste.Paste{
		URL:              message.URL,
		Content:          message.Content,
		CreatedAt:        message.CreatedAt,
		ExpirationPolicy: expPolicy,
	}

	// Save to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.collection.InsertOne(ctx, newPaste)
	if err != nil {
		c.logger.Errorf("Failed to save paste to database: %v", err)
		err := delivery.Nack(false, true)
		if err != nil {
			return
		} // Requeue for retry
		return
	}

	c.logger.Infof("Saved paste with URL: %s", message.URL)
	err = delivery.Ack(false)
	if err != nil {
		return
	}
}

func (c *RabbitMQConsumer) Stop() error {
	if c.channel != nil {
		if err := c.channel.Cancel(c.consumerTag, false); err != nil {
			return err
		}
		return c.channel.Close()
	}
	return nil
}
