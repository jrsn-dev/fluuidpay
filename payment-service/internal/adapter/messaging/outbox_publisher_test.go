package messaging

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/fluuid/payment-service/internal/domain"
)

type mockChannel struct {
	published int
}

func (m *mockChannel) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	m.published++
	return nil
}
func (m *mockChannel) Confirm(noWait bool) error { return nil }
func (m *mockChannel) NotifyPublish(confirm chan amqp.Confirmation) chan amqp.Confirmation {
	go func() {
		// Mock a successful confirmation
		confirm <- amqp.Confirmation{Ack: true}
	}()
	return confirm
}

type mockOutboxRepo struct {
	entries []*domain.OutboxEntry
	marked  int
}

func (m *mockOutboxRepo) Insert(ctx context.Context, e *domain.OutboxEntry) error { return nil }
func (m *mockOutboxRepo) FetchPending(ctx context.Context, limit int) ([]*domain.OutboxEntry, error) {
	return m.entries, nil
}
func (m *mockOutboxRepo) MarkPublished(ctx context.Context, id string) error {
	m.marked++
	return nil
}
func (m *mockOutboxRepo) MarkFailed(ctx context.Context, id string, err error, retry bool) error {
	return nil
}

func TestOutboxPublisher_PublishPending(t *testing.T) {
	repo := &mockOutboxRepo{
		entries: []*domain.OutboxEntry{
			{ID: "evt1", EventType: domain.EventPaymentPending, Payload: []byte("{}")},
			{ID: "evt2", EventType: domain.EventPaymentApproved, Payload: []byte("{}")},
		},
	}
	ch := &mockChannel{}
	topology := DefaultTopology()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	pub := NewOutboxPublisher(repo, ch, topology, logger)

	// Since Run is a blocking loop, we can just call processBatch directly for testing
	// Wait, processBatch is private. We can test it by using a short context timeout on Run.
	pub.publishBatch(context.Background())

	if ch.published != 2 {
		t.Errorf("expected 2 messages published, got %d", ch.published)
	}

	if repo.marked != 2 {
		t.Errorf("expected 2 events marked as published, got %d", repo.marked)
	}
}

type mockChannelFail struct{}

func (m *mockChannelFail) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	return errors.New("amqp error")
}
func (m *mockChannelFail) Confirm(noWait bool) error { return nil }
func (m *mockChannelFail) NotifyPublish(confirm chan amqp.Confirmation) chan amqp.Confirmation { return confirm }

func TestOutboxPublisher_PublishFailure(t *testing.T) {
	repo := &mockOutboxRepo{
		entries: []*domain.OutboxEntry{
			{ID: "evt1", EventType: domain.EventPaymentPending, Payload: []byte("{}")},
		},
	}
	ch := &mockChannelFail{}
	topology := DefaultTopology()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	pub := NewOutboxPublisher(repo, ch, topology, logger)

	pub.publishBatch(context.Background())

	// MarkPublished should NOT be called if publish fails
	if repo.marked != 0 {
		t.Errorf("expected 0 events marked as published, got %d", repo.marked)
	}
}
