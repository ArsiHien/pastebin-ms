package shared

import (
	"log"

	"github.com/rabbitmq/amqp091-go"
)

// Logger wraps standard logging
type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

func (l *Logger) Infof(format string, args ...interface{}) {
	log.Printf("[INFO] "+format, args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	log.Printf("[ERROR] "+format, args...)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	log.Fatalf("[FATAL] "+format, args...)
}

// NewRabbitMQConn creates a RabbitMQ connection
func NewRabbitMQConn(uri string) (*amqp091.Connection, error) {
	return amqp091.Dial(uri)
}
