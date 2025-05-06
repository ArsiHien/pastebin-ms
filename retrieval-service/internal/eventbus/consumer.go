package eventbus

import (
	"context"
	"encoding/json"
	"retrieval-service/internal/cache"
	"retrieval-service/internal/domain/paste"
	"retrieval-service/internal/metrics"
	"retrieval-service/shared"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/mongo"
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
	cache       cache.PasteCache
	logger      *shared.Logger
	consumerTag string
}

func NewRabbitMQConsumer(conn *amqp.Connection, db *mongo.Database, cache cache.PasteCache, logger *shared.Logger) (*RabbitMQConsumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		logger.Errorf("Failed to open RabbitMQ channel", "error", err.Error())
		return nil, err
	}

	err = ch.ExchangeDeclare(
		"pastebin_events",
		"topic",
		true,  // Durable
		false, // Auto-deleted
		false, // Internal
		false, // No-wait
		nil,   // Arguments
	)
	if err != nil {
		if closeErr := ch.Close(); closeErr != nil {
			logger.Errorf("Failed to close channel after exchange declare error", "error", closeErr.Error())
		}
		logger.Errorf("Failed to declare exchange", "error", err.Error())
		return nil, err
	}

	collection := db.Collection("pastes")

	return &RabbitMQConsumer{
		channel:     ch,
		collection:  collection,
		cache:       cache,
		logger:      logger,
		consumerTag: "paste-creation-consumer",
	}, nil
}

func (c *RabbitMQConsumer) Start() error {
	q, err := c.channel.QueueDeclare(
		"paste_creation_queue",
		true,  // Durable
		false, // Auto-deleted
		false, // Exclusive
		false, // No-wait
		nil,   // Arguments
	)
	if err != nil {
		c.logger.Errorf("Failed to declare queue", "error", err.Error())
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
		c.logger.Errorf("Failed to bind queue", "error", err.Error())
		return err
	}

	msgs, err := c.channel.Consume(
		q.Name,
		c.consumerTag,
		false, // Auto-ack
		false, // Exclusive
		false, // No-local
		false, // No-wait
		nil,   // Args
	)
	if err != nil {
		c.logger.Errorf("Failed to start consuming messages", "error", err.Error())
		return err
	}

	go func() {
		for d := range msgs {
			c.handleMessage(d)
		}
		c.logger.Infof("Stopped consuming messages due to channel close")
	}()

	c.logger.Infof("Started consuming messages from queue", "queue", q.Name)
	return nil
}

func (c *RabbitMQConsumer) handleMessage(delivery amqp.Delivery) {
	logger := c.logger.With("messageID", delivery.MessageId)

	// Giai đoạn 1: Xử lý message
	logger.Infof("Received message", "body", string(delivery.Body))

	var message PasteMessage
	if err := json.Unmarshal(delivery.Body, &message); err != nil {
		logger.Errorf("Failed to unmarshal paste message", "error", err.Error(), "body", string(delivery.Body))
		if nackErr := delivery.Nack(false, false); nackErr != nil {
			logger.Errorf("Failed to nack message", "error", nackErr.Error())
		}
		return
	}
	logger.Infof("Parsed paste created event", "url", message.URL)

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

	// Giai đoạn 2: Lưu paste vào MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Kiểm tra dữ liệu trước khi lưu
	if newPaste.URL == "" || newPaste.Content == "" {
		logger.Errorf("Invalid paste data", "paste", newPaste)
		if nackErr := delivery.Nack(false, false); nackErr != nil {
			logger.Errorf("Failed to nack message", "error", nackErr.Error())
		}
		return
	}

	phaseStart := time.Now()
	_, err := c.collection.InsertOne(ctx, newPaste)
	if err != nil {
		logger.Errorf("Failed to save paste to database", "error", err.Error(), "paste", newPaste)
		if nackErr := delivery.Nack(false, true); nackErr != nil {
			logger.Errorf("Failed to nack message", "error", nackErr.Error())
		}
		return
	}
	logger.Infof("Successfully saved paste to database", "url", newPaste.URL)
	metrics.PasteProcessingDuration.WithLabelValues("mongo_save").Observe(time.Since(phaseStart).Seconds())

	// Giai đoạn 3: Lưu paste vào Redis cache
	phaseStart = time.Now()
	if err := c.cache.Set(&newPaste); err != nil {
		logger.Errorf("Failed to save paste to Redis cache", "error", err.Error(), "url", newPaste.URL)
		// Không nack vì MongoDB đã lưu thành công
	} else {
		logger.Infof("Successfully saved paste to Redis cache", "url", newPaste.URL)
	}
	metrics.PasteProcessingDuration.WithLabelValues("cache_save").Observe(time.Since(phaseStart).Seconds())

	if !newPaste.CreatedAt.IsZero() {
		duration := time.Since(newPaste.CreatedAt).Seconds()
		metrics.PasteProcessingDuration.WithLabelValues("total").Observe(duration)
		logger.Infof("Latency from CreatedAt to processing complete", "durationSeconds", duration)
	}

	// Ack message
	if err := delivery.Ack(false); err != nil {
		logger.Errorf("Failed to acknowledge message", "error", err.Error())
	}
}

func (c *RabbitMQConsumer) Stop() error {
	if c.channel != nil {
		if err := c.channel.Cancel(c.consumerTag, false); err != nil {
			c.logger.Errorf("Failed to cancel consumer", "error", err.Error())
			return err
		}
		if err := c.channel.Close(); err != nil {
			c.logger.Errorf("Failed to close RabbitMQ channel", "error", err.Error())
			return err
		}
	}
	c.logger.Infof("Stopped RabbitMQ consumer")
	return nil
}
