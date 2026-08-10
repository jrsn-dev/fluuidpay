package messaging

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Topology defines the RabbitMQ exchanges, queues, and bindings.
type Topology struct {
	Exchange       string
	DeadLetterExch string
	MainQueue      string
	RetryQueues    []string
	DLQ            string
	RoutingKey     string
	RetryDelaysMs  []int32
}

// DefaultTopology returns the standard topology for the payment service.
func DefaultTopology() Topology {
	return Topology{
		Exchange:       "payment.events",
		DeadLetterExch: "payment.dlx",
		MainQueue:      "payment.processed",
		RetryQueues: []string{
			"payment.processed.retry.1",
			"payment.processed.retry.2",
			"payment.processed.retry.3",
		},
		DLQ:           "payment.processed.dlq",
		RoutingKey:    "payment.processed",
		RetryDelaysMs: []int32{10_000, 60_000, 300_000},
	}
}

// Declare creates all exchanges, queues, and bindings.
// This operation is idempotent and safe to run on startup.
func Declare(ch *amqp.Channel, t Topology) error {
	// Main topic exchange for payment events
	if err := ch.ExchangeDeclare(t.Exchange, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange %s: %w", t.Exchange, err)
	}

	// Dead letter exchange for retry routing
	if err := ch.ExchangeDeclare(t.DeadLetterExch, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dlx %s: %w", t.DeadLetterExch, err)
	}

	// Main consumption queue with DLQ fallback
	if _, err := ch.QueueDeclare(t.MainQueue, true, false, false, false, amqp.Table{
		"x-dead-letter-exchange":    t.DeadLetterExch,
		"x-dead-letter-routing-key": "payment.processed.dlq",
	}); err != nil {
		return fmt.Errorf("declare main queue %s: %w", t.MainQueue, err)
	}

	if err := ch.QueueBind(t.MainQueue, t.RoutingKey, t.Exchange, false, nil); err != nil {
		return fmt.Errorf("bind main queue: %w", err)
	}

	// Retry queues with TTL — messages wait here then route back to main queue
	for i, queue := range t.RetryQueues {
		if i >= len(t.RetryDelaysMs) {
			return fmt.Errorf("retry queue %q has no configured delay", queue)
		}

		if _, err := ch.QueueDeclare(queue, true, false, false, false, amqp.Table{
			"x-message-ttl":             t.RetryDelaysMs[i],
			"x-dead-letter-exchange":    t.Exchange,
			"x-dead-letter-routing-key": t.RoutingKey,
		}); err != nil {
			return fmt.Errorf("declare retry queue %s: %w", queue, err)
		}

		retryRoutingKey := fmt.Sprintf("payment.retry.%d", i+1)
		if err := ch.QueueBind(queue, retryRoutingKey, t.DeadLetterExch, false, nil); err != nil {
			return fmt.Errorf("bind retry queue %s: %w", queue, err)
		}
	}

	// Dead letter queue — final destination for unprocessable messages
	if _, err := ch.QueueDeclare(t.DLQ, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dlq %s: %w", t.DLQ, err)
	}

	if err := ch.QueueBind(t.DLQ, "payment.processed.dlq", t.DeadLetterExch, false, nil); err != nil {
		return fmt.Errorf("bind dlq: %w", err)
	}

	return nil
}
