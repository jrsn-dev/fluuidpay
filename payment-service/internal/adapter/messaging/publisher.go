package messaging

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/fluuid/payment-service/internal/domain"
)

// AMQPChannel abstracts the amqp091-go Channel for testing.
type AMQPChannel interface {
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Confirm(noWait bool) error
	NotifyPublish(confirm chan amqp.Confirmation) chan amqp.Confirmation
}

// OutboxPublisher polls the outbox table and publishes events to RabbitMQ.
type OutboxPublisher struct {
	outboxRepo domain.OutboxRepository
	channel    AMQPChannel
	topology   Topology
	logger     *slog.Logger
	interval   time.Duration
	batchSize  int
}

// NewOutboxPublisher creates a new outbox publisher.
func NewOutboxPublisher(
	outboxRepo domain.OutboxRepository,
	channel AMQPChannel,
	topology Topology,
	logger *slog.Logger,
) *OutboxPublisher {
	return &OutboxPublisher{
		outboxRepo: outboxRepo,
		channel:    channel,
		topology:   topology,
		logger:     logger,
		interval:   2 * time.Second,
		batchSize:  50,
	}
}

// Run starts the polling loop. Blocks until context is cancelled.
func (p *OutboxPublisher) Run(ctx context.Context) {
	p.logger.Info("outbox publisher started", "interval", p.interval)

	// Enable publisher confirms
	if err := p.channel.Confirm(false); err != nil {
		p.logger.Error("failed to enable publisher confirms", "error", err)
		return
	}

	backoff := p.interval
	timer := time.NewTimer(backoff)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("outbox publisher stopped")
			return
		case <-timer.C:
			if err := p.publishBatch(ctx); err != nil {
				// Exponential backoff
				backoff *= 2
				if backoff > 30*time.Second {
					backoff = 30 * time.Second
				}
				p.logger.Warn("outbox publisher error, backing off", "backoff", backoff)
			} else {
				// Reset backoff
				backoff = p.interval
			}

			// Add jitter (±10%)
			jitter := time.Duration(rand.Intn(int(backoff)/5)) - backoff/10
			timer.Reset(backoff + jitter)
		}
	}
}

// publishBatch fetches pending events and publishes them.
func (p *OutboxPublisher) publishBatch(ctx context.Context) error {
	entries, err := p.outboxRepo.FetchPending(ctx, p.batchSize)
	if err != nil {
		p.logger.Error("failed to fetch pending outbox events", "error", err)
		return err
	}

	var batchErr error
	for _, entry := range entries {
		if err := p.publishOne(ctx, entry); err != nil {
			p.logger.Error("failed to publish outbox event",
				"event_id", entry.ID,
				"aggregate_id", entry.AggregateID,
				"error", err,
			)
			batchErr = err
			// Mark as retryable failure
			if markErr := p.outboxRepo.MarkFailed(ctx, entry.ID, err, true); markErr != nil {
				p.logger.Error("failed to mark outbox as failed", "error", markErr)
			}
			continue
		}

		// Mark as published
		if err := p.outboxRepo.MarkPublished(ctx, entry.ID); err != nil {
			p.logger.Error("failed to mark outbox as published", "error", err)
			batchErr = err
		} else {
			p.logger.Info("outbox event published",
				"event_id", entry.ID,
				"event_type", entry.EventType,
				"aggregate_id", entry.AggregateID,
			)
		}
	}
	return batchErr
}

// publishOne publishes a single event with publisher confirms.
func (p *OutboxPublisher) publishOne(ctx context.Context, entry *domain.OutboxEntry) error {
	confirms := p.channel.NotifyPublish(make(chan amqp.Confirmation, 1))

	err := p.channel.PublishWithContext(ctx, p.topology.Exchange, p.topology.RoutingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         entry.Payload,
		DeliveryMode: amqp.Persistent,
		MessageId:    entry.ID,
		Timestamp:    time.Now().UTC(),
		Headers: amqp.Table{
			"event_type":       string(entry.EventType),
			"aggregate_id":     entry.AggregateID,
			"schema_version":   1,
			"x-correlation-id": entry.CorrelationID,
		},
	})
	if err != nil {
		return err
	}

	// Wait for broker confirmation
	select {
	case confirmation := <-confirms:
		if !confirmation.Ack {
			return domain.ErrEventPublishFailed
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return domain.ErrEventPublishFailed
	}
}
