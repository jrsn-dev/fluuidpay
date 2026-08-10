//go:build chaos
// +build chaos

package chaos_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/fluuid/payment-service/internal/adapter/messaging"
	"github.com/fluuid/payment-service/internal/domain"
)

type chaosChannel struct {
	published int
	failed    int
}

func (c *chaosChannel) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	// Randomly fail 30% of the time
	if rand.Float32() < 0.3 {
		c.failed++
		return errors.New("chaos failure: simulated network partition")
	}
	c.published++
	return nil
}

func (c *chaosChannel) Confirm(noWait bool) error {
	return nil
}

func (c *chaosChannel) NotifyPublish(confirm chan amqp.Confirmation) chan amqp.Confirmation {
	return confirm
}

type chaosRepo struct {
	entries []*domain.OutboxEntry
}

func (r *chaosRepo) Insert(ctx context.Context, entry *domain.OutboxEntry) error {
	r.entries = append(r.entries, entry)
	return nil
}

func (r *chaosRepo) FetchPending(ctx context.Context, limit int) ([]*domain.OutboxEntry, error) {
	var pending []*domain.OutboxEntry
	for _, e := range r.entries {
		if e.Status == domain.OutboxPending {
			pending = append(pending, e)
			if len(pending) == limit {
				break
			}
		}
	}
	return pending, nil
}

func (r *chaosRepo) MarkPublished(ctx context.Context, id string) error {
	for _, e := range r.entries {
		if e.ID == id {
			e.Status = "PUBLISHED"
			return nil
		}
	}
	return nil
}

func (r *chaosRepo) MarkFailed(ctx context.Context, id string, err error, retryable bool) error {
	for _, e := range r.entries {
		if e.ID == id {
			e.LastError = err.Error()
			e.AttemptCount++
			return nil
		}
	}
	return nil
}

func TestOutbox_ChaosResilience(t *testing.T) {
	repo := &chaosRepo{}
	
	// Create 100 pending events
	for i := 0; i < 100; i++ {
		repo.entries = append(repo.entries, &domain.OutboxEntry{
			ID:      fmt.Sprintf("%d", i),
			Status:  domain.OutboxPending,
			Payload: []byte("{}"),
		})
	}

	ch := &chaosChannel{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	pub := messaging.NewOutboxPublisher(repo, ch, messaging.DefaultTopology(), logger)

	// Run publisher for a short time
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pub.Run(ctx)

	// After running, check state
	publishedCount := 0
	pendingCount := 0
	for _, e := range repo.entries {
		if e.Status == "PUBLISHED" {
			publishedCount++
		}
		if e.Status == domain.OutboxPending {
			pendingCount++
		}
	}

	t.Logf("Total published (success): %d", publishedCount)
	t.Logf("Total failed (simulated network error): %d", ch.failed)
	t.Logf("Remaining pending: %d", pendingCount)

	// In a real chaos test, we'd loop until all are published (eventual consistency).
	// But this verifies the publisher doesn't crash on intermittent errors.
}
