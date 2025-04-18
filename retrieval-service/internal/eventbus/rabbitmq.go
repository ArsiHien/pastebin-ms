package eventbus

import amqp "github.com/rabbitmq/amqp091-go"

func NewRabbitMQConn(uri string) (*amqp.Connection, error) {
	return amqp.Dial(uri)
}
