package platform

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQ wraps an AMQP connection and channel.
type RabbitMQ struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
}

// NewRabbitMQ creates a new RabbitMQ connection with a channel.
func NewRabbitMQ(cfg RabbitMQConfig) (*RabbitMQ, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("connect to rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("open rabbitmq channel: %w", err)
	}

	return &RabbitMQ{Conn: conn, Channel: ch}, nil
}

// Close releases the RabbitMQ connection and channel.
func (r *RabbitMQ) Close() {
	if r.Channel != nil {
		r.Channel.Close()
	}
	if r.Conn != nil {
		r.Conn.Close()
	}
}

// HealthCheck verifies the RabbitMQ connection is alive.
func (r *RabbitMQ) HealthCheck() error {
	if r.Conn == nil || r.Conn.IsClosed() {
		return fmt.Errorf("rabbitmq connection is closed")
	}
	return nil
}
