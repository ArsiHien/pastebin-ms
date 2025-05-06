package worker

import (
	"context"
	"encoding/json"
	"github.com/ArsiHien/pastebin-ms/create-service/internal/domain/paste"
	"github.com/ArsiHien/pastebin-ms/create-service/internal/metrics"
	amqp "github.com/rabbitmq/amqp091-go"
	"strings"
	"time"
)

type MySQLSaveWorker struct {
	channel   *amqp.Channel
	repo      paste.Repository
	queueName string
}

func NewMySQLSaveWorker(conn *amqp.Connection, repo paste.Repository) (*MySQLSaveWorker, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	q, err := ch.QueueDeclare(
		"paste_save_queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	err = ch.QueueBind(
		q.Name,
		"paste.save",
		"pastebin_events",
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return &MySQLSaveWorker{
		channel:   ch,
		repo:      repo,
		queueName: q.Name,
	}, nil
}

func (w *MySQLSaveWorker) Start() error {
	msgs, err := w.channel.Consume(
		w.queueName,
		"mysql-save-consumer",
		false, // Manual ack
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {
			w.handleMessage(msg)
		}
	}()

	return nil
}

func (w *MySQLSaveWorker) handleMessage(msg amqp.Delivery) {

	phaseStart := time.Now()
	var p paste.Paste
	if err := json.Unmarshal(msg.Body, &p); err != nil {
		if err := msg.Nack(false, false); err != nil {

		}
		return
	}

	_, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := w.repo.Save(&p); err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") && strings.Contains(err.Error(), "uni_pastes_url") {
			if err := msg.Ack(false); err != nil {
			}
			return
		}
		if err := msg.Nack(false, true); err != nil {
		}
		return
	}
	if err := msg.Ack(false); err != nil {
	}

	metrics.CreateRequestDuration.WithLabelValues("mysql_save").Observe(time.Since(phaseStart).Seconds())
}

func (w *MySQLSaveWorker) Stop() error {
	if err := w.channel.Close(); err != nil {
		return err
	}
	return nil
}
