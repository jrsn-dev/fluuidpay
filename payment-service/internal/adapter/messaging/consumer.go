package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/fluuid/payment-service/internal/domain"
)

const (
	headerRetryCount    = "x-retry-count"
	headerOriginalKey   = "x-original-routing-key"
	headerFirstFailedAt = "x-first-failed-at"
	headerLastError     = "x-last-error"
	maxErrorHeaderBytes = 512
)

// PaymentProcessedEvent matches the event contract from the SDD.
type PaymentProcessedEvent struct {
	EventID       string          `json:"event_id"`
	EventType     string          `json:"event_type"`
	SchemaVersion int             `json:"schema_version"`
	OccurredAt    time.Time       `json:"occurred_at"`
	TransactionID string          `json:"transaction_id"`
	OrderID       string          `json:"order_id"`
	UserID        string          `json:"user_id"`
	Status        string          `json:"status"`
	AmountMinor   int64           `json:"amount_minor"`
	Currency      string          `json:"currency"`
	Taxes         json.RawMessage `json:"taxes,omitempty"`
}

// Consumer processes payment events from RabbitMQ with retry and DLQ support.
type Consumer struct {
	ch           *amqp.Channel
	processed    domain.ProcessedEventStore
	topology     Topology
	logger       *slog.Logger
	maxRetries   int
	retryRoutes  []string
	consumerName string
	handler      func(ctx context.Context, event PaymentProcessedEvent) error
}

// ConsumerConfig holds configuration for the consumer.
type ConsumerConfig struct {
	Channel      *amqp.Channel
	Processed    domain.ProcessedEventStore
	Topology     Topology
	Logger       *slog.Logger
	MaxRetries   int
	ConsumerName string
	Handler      func(ctx context.Context, event PaymentProcessedEvent) error
}

// NewConsumer creates a new idempotent consumer.
func NewConsumer(cfg ConsumerConfig) *Consumer {
	retryRoutes := make([]string, len(cfg.Topology.RetryQueues))
	for i := range cfg.Topology.RetryQueues {
		retryRoutes[i] = fmt.Sprintf("payment.retry.%d", i+1)
	}

	return &Consumer{
		ch:           cfg.Channel,
		processed:    cfg.Processed,
		topology:     cfg.Topology,
		logger:       cfg.Logger,
		maxRetries:   cfg.MaxRetries,
		retryRoutes:  retryRoutes,
		consumerName: cfg.ConsumerName,
		handler:      cfg.Handler,
	}
}

// Run starts consuming messages. Blocks until context is cancelled.
func (c *Consumer) Run(ctx context.Context) error {
	// Set prefetch
	if err := c.ch.Qos(20, 0, false); err != nil {
		return fmt.Errorf("set qos: %w", err)
	}

	deliveries, err := c.ch.Consume(
		c.topology.MainQueue,
		c.consumerName,
		false, // autoAck=false — we ACK manually
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	c.logger.Info("consumer started", "queue", c.topology.MainQueue, "consumer", c.consumerName)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("consumer stopping")
			return nil
		case msg, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("delivery channel closed")
			}
			c.processOne(ctx, msg)
		}
	}
}

// processOne handles a single message delivery.
func (c *Consumer) processOne(ctx context.Context, msg amqp.Delivery) {
	err := c.handleMessage(ctx, msg)
	if err == nil {
		if ackErr := msg.Ack(false); ackErr != nil {
			c.logger.Error("ack failed", "error", ackErr)
		}
		return
	}

	// Check if it's a permanent error
	var permanent *domain.PermanentError
	if errors.As(err, &permanent) {
		c.toDLQ(msg, err)
		return
	}

	// Check retry count
	retryCount := headerInt(msg.Headers, headerRetryCount)
	if retryCount >= c.maxRetries {
		c.toDLQ(msg, fmt.Errorf("retry limit reached: %w", err))
		return
	}

	// Send to retry queue
	if retryCount < len(c.retryRoutes) {
		route := c.retryRoutes[retryCount]
		if pubErr := publishRetry(c.ch, msg, c.topology.DeadLetterExch, route, err); pubErr != nil {
			c.logger.Error("publish retry failed", "error", pubErr, "event_id", msg.MessageId)
			_ = msg.Nack(false, true)
			return
		}
		if ackErr := msg.Ack(false); ackErr != nil {
			c.logger.Error("ack after retry publish failed", "error", ackErr)
		}
	} else {
		c.toDLQ(msg, err)
	}
}

// handleMessage processes a single event with idempotency check.
func (c *Consumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	var event PaymentProcessedEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		return &domain.PermanentError{Err: fmt.Errorf("decode event: %w", err)}
	}

	if event.EventID == "" {
		return &domain.PermanentError{Err: errors.New("event_id is required")}
	}

	// Check if already processed (idempotency)
	already, err := c.processed.AlreadyProcessed(ctx, event.EventID)
	if err != nil {
		return &domain.RetryableError{Err: fmt.Errorf("check processed: %w", err)}
	}
	if already {
		c.logger.Info("event already processed, skipping", "event_id", event.EventID)
		return nil
	}

	// Process the event
	if err := c.handler(ctx, event); err != nil {
		return err
	}

	// Mark as processed
	if err := c.processed.MarkProcessed(ctx, event.EventID, c.consumerName); err != nil {
		return &domain.RetryableError{Err: fmt.Errorf("mark processed: %w", err)}
	}

	c.logger.Info("event processed",
		"event_id", event.EventID,
		"transaction_id", event.TransactionID,
		"status", event.Status,
	)

	return nil
}

// toDLQ sends a message to the dead letter queue.
func (c *Consumer) toDLQ(msg amqp.Delivery, cause error) {
	c.logger.Warn("sending message to DLQ",
		"event_id", msg.MessageId,
		"cause", cause.Error(),
		"retry_count", headerInt(msg.Headers, headerRetryCount),
	)

	if err := publishDLQ(c.ch, msg, c.topology.DeadLetterExch, cause); err != nil {
		c.logger.Error("publish dlq failed", "error", err, "event_id", msg.MessageId)
		_ = msg.Nack(false, true)
		return
	}
	if err := msg.Ack(false); err != nil {
		c.logger.Error("ack after dlq publish failed", "error", err)
	}
}

// --- Helper functions ---

func headerInt(headers amqp.Table, key string) int {
	if headers == nil {
		return 0
	}
	switch n := headers[key].(type) {
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	default:
		return 0
	}
}

func cloneHeaders(in amqp.Table) amqp.Table {
	out := make(amqp.Table, len(in)+4)
	for k, v := range in {
		out[k] = v
	}
	return out
}

func truncate(s string) string {
	if len(s) <= maxErrorHeaderBytes {
		return s
	}
	return s[:maxErrorHeaderBytes]
}

func publishRetry(ch *amqp.Channel, msg amqp.Delivery, exchange, routingKey string, err error) error {
	headers := cloneHeaders(msg.Headers)
	headers[headerRetryCount] = headerInt(headers, headerRetryCount) + 1
	headers[headerOriginalKey] = msg.RoutingKey
	headers[headerLastError] = truncate(err.Error())
	if _, ok := headers[headerFirstFailedAt]; !ok {
		headers[headerFirstFailedAt] = time.Now().UTC().Format(time.RFC3339)
	}

	return ch.PublishWithContext(context.Background(), exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  msg.ContentType,
		Body:         msg.Body,
		Headers:      headers,
		DeliveryMode: amqp.Persistent,
		MessageId:    msg.MessageId,
		Timestamp:    time.Now().UTC(),
	})
}

func publishDLQ(ch *amqp.Channel, msg amqp.Delivery, exchange string, err error) error {
	headers := cloneHeaders(msg.Headers)
	headers[headerLastError] = truncate(err.Error())
	headers["x-dead-lettered-at"] = time.Now().UTC().Format(time.RFC3339)

	return ch.PublishWithContext(context.Background(), exchange, "payment.processed.dlq", false, false, amqp.Publishing{
		ContentType:  msg.ContentType,
		Body:         msg.Body,
		Headers:      headers,
		DeliveryMode: amqp.Persistent,
		MessageId:    msg.MessageId,
		Timestamp:    time.Now().UTC(),
	})
}
