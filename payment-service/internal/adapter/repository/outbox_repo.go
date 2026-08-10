package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fluuid/payment-service/internal/domain"
)

// OutboxRepo implements domain.OutboxRepository using PostgreSQL.
type OutboxRepo struct {
	pool *pgxpool.Pool
}

// NewOutboxRepo creates a new outbox repository.
func NewOutboxRepo(pool *pgxpool.Pool) *OutboxRepo {
	return &OutboxRepo{pool: pool}
}

// Insert creates a new outbox entry within the current transaction.
func (r *OutboxRepo) Insert(ctx context.Context, entry *domain.OutboxEntry) error {
	query := `
		INSERT INTO outbox_events (id, aggregate_id, event_type, payload, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, query,
		entry.ID,
		entry.AggregateID,
		string(entry.EventType),
		entry.Payload,
		string(domain.OutboxPending),
		entry.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}
	return nil
}

// FetchPending retrieves unprocessed events ready for publishing.
// Uses SELECT FOR UPDATE SKIP LOCKED to prevent concurrent publishers
// from processing the same event.
func (r *OutboxRepo) FetchPending(ctx context.Context, limit int) ([]*domain.OutboxEntry, error) {
	query := `
		UPDATE outbox_events
		SET status = 'PUBLISHING', locked_at = NOW()
		WHERE id IN (
			SELECT id FROM outbox_events
			WHERE status IN ('PENDING', 'FAILED_RETRYABLE')
			  AND next_attempt_at <= NOW()
			  AND attempt_count < max_attempts
			ORDER BY created_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, aggregate_id, event_type, payload, status,
		          attempt_count, next_attempt_at, last_error,
		          locked_at, created_at, published_at
	`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch pending outbox events: %w", err)
	}
	defer rows.Close()

	var entries []*domain.OutboxEntry
	for rows.Next() {
		var e domain.OutboxEntry
		var eventType, status string
		var payload json.RawMessage

		if err := rows.Scan(
			&e.ID, &e.AggregateID, &eventType, &payload, &status,
			&e.AttemptCount, &e.NextAttemptAt, &e.LastError,
			&e.LockedAt, &e.CreatedAt, &e.PublishedAt,
		); err != nil {
			return nil, fmt.Errorf("scan outbox event: %w", err)
		}

		e.EventType = domain.EventType(eventType)
		e.Status = domain.OutboxStatus(status)
		e.Payload = payload
		entries = append(entries, &e)
	}

	return entries, nil
}

// MarkPublished updates an outbox entry as successfully published.
func (r *OutboxRepo) MarkPublished(ctx context.Context, id string) error {
	query := `
		UPDATE outbox_events
		SET status = 'PUBLISHED', published_at = NOW(), locked_at = NULL
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("mark outbox published: %w", err)
	}
	return nil
}

// MarkFailed updates an outbox entry with failure information.
func (r *OutboxRepo) MarkFailed(ctx context.Context, id string, failErr error, retryable bool) error {
	status := domain.OutboxFailedPermanent
	if retryable {
		status = domain.OutboxFailedRetryable
	}

	// Exponential backoff: 2^attempts * 5 seconds
	query := `
		UPDATE outbox_events
		SET status = $2,
		    last_error = $3,
		    attempt_count = attempt_count + 1,
		    next_attempt_at = NOW() + (POWER(2, attempt_count) * INTERVAL '5 seconds'),
		    locked_at = NULL
		WHERE id = $1
	`
	errMsg := ""
	if failErr != nil {
		errMsg = failErr.Error()
		if len(errMsg) > 512 {
			errMsg = errMsg[:512]
		}
	}

	_, err := r.pool.Exec(ctx, query, id, string(status), errMsg)
	if err != nil {
		return fmt.Errorf("mark outbox failed: %w", err)
	}
	return nil
}

// CountPending returns the number of pending outbox events (for monitoring).
func (r *OutboxRepo) CountPending(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM outbox_events
		WHERE status IN ('PENDING', 'FAILED_RETRYABLE', 'PUBLISHING')
	`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count pending outbox: %w", err)
	}
	return count, nil
}

// OldestPendingAge returns the age of the oldest pending event (for monitoring).
func (r *OutboxRepo) OldestPendingAge(ctx context.Context) (time.Duration, error) {
	var oldest *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT MIN(created_at) FROM outbox_events
		WHERE status IN ('PENDING', 'FAILED_RETRYABLE')
	`).Scan(&oldest)
	if err != nil || oldest == nil {
		return 0, err
	}
	return time.Since(*oldest), nil
}
