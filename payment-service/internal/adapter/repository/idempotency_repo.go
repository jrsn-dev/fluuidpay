package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/fluuid/payment-service/internal/domain"
)

// IdempotencyRepo implements domain.IdempotencyStore using Redis (hot) + PostgreSQL (cold).
type IdempotencyRepo struct {
	pool   *pgxpool.Pool
	rdb    *redis.Client
	ttl    time.Duration
}

// NewIdempotencyRepo creates a new idempotency store.
func NewIdempotencyRepo(pool *pgxpool.Pool, rdb *redis.Client, ttl time.Duration) *IdempotencyRepo {
	return &IdempotencyRepo{pool: pool, rdb: rdb, ttl: ttl}
}

// redisKey generates the Redis key for an idempotency record.
func (r *IdempotencyRepo) redisKey(key string) string {
	return "idem:" + key
}

// Get retrieves the stored record for an idempotency key.
// Checks Redis first (hot), falls back to PostgreSQL (cold).
func (r *IdempotencyRepo) Get(ctx context.Context, key string) (*domain.IdempotencyRecord, error) {
	// Try Redis first
	data, err := r.rdb.Get(ctx, r.redisKey(key)).Bytes()
	if err == nil {
		var record domain.IdempotencyRecord
		if err := json.Unmarshal(data, &record); err == nil {
			return &record, nil
		}
	} else if err != redis.Nil {
		// Redis error — fall through to PostgreSQL
	}

	// Fall back to PostgreSQL
	query := `
		SELECT idempotency_key, payload_hash, status, response, http_code,
		       transaction_id, created_at, completed_at
		FROM idempotency_records
		WHERE idempotency_key = $1 AND operation = 'create_payment'
		  AND expires_at > NOW()
	`

	var record domain.IdempotencyRecord
	var txnID *string
	err = r.pool.QueryRow(ctx, query, key).Scan(
		&record.Key, &record.PayloadHash, &record.Status, &record.Response,
		&record.HTTPCode, &txnID, &record.CreatedAt, &record.CompletedAt,
	)
	if err != nil {
		// If no row found, return nil (key does not exist)
		return nil, nil
	}

	if txnID != nil {
		record.TransactionID = *txnID
	}

	// Re-cache in Redis for next lookup
	if data, err := json.Marshal(record); err == nil {
		r.rdb.Set(ctx, r.redisKey(key), data, r.ttl)
	}

	return &record, nil
}

// Reserve attempts to acquire the idempotency key atomically.
// Uses Redis SET NX for speed + PostgreSQL INSERT for durability.
func (r *IdempotencyRepo) Reserve(ctx context.Context, key, payloadHash string, ttl time.Duration) (bool, error) {
	// Try Redis SET NX first (fast path)
	record := domain.IdempotencyRecord{
		Key:         key,
		PayloadHash: payloadHash,
		Status:      domain.IdempotencyProcessing,
		CreatedAt:   time.Now().UTC(),
	}

	data, err := json.Marshal(record)
	if err != nil {
		return false, fmt.Errorf("marshal idempotency record: %w", err)
	}

	set, err := r.rdb.SetNX(ctx, r.redisKey(key), data, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("redis set nx: %w", err)
	}
	if !set {
		// Key already exists in Redis — not reserved
		return false, nil
	}

	// Also insert into PostgreSQL for durability
	query := `
		INSERT INTO idempotency_records (idempotency_key, operation, payload_hash, status, expires_at)
		VALUES ($1, 'create_payment', $2, 'PROCESSING', NOW() + $3::interval)
		ON CONFLICT (idempotency_key, operation) DO NOTHING
	`
	_, err = r.pool.Exec(ctx, query, key, payloadHash, fmt.Sprintf("%d seconds", int(ttl.Seconds())))
	if err != nil {
		// Clean up Redis if PG insert fails
		r.rdb.Del(ctx, r.redisKey(key))
		return false, fmt.Errorf("insert idempotency record: %w", err)
	}

	return true, nil
}

// Complete marks the idempotency key as completed with the final response.
func (r *IdempotencyRepo) Complete(ctx context.Context, key string, response []byte, httpCode int, ttl time.Duration) error {
	now := time.Now().UTC()

	// Update PostgreSQL
	query := `
		UPDATE idempotency_records
		SET status = 'COMPLETED', response = $2, http_code = $3, completed_at = $4
		WHERE idempotency_key = $1 AND operation = 'create_payment'
	`
	_, err := r.pool.Exec(ctx, query, key, response, httpCode, now)
	if err != nil {
		return fmt.Errorf("complete idempotency record: %w", err)
	}

	// Update Redis cache
	record := domain.IdempotencyRecord{
		Key:         key,
		Status:      domain.IdempotencyCompleted,
		Response:    response,
		HTTPCode:    httpCode,
		CreatedAt:   now,
		CompletedAt: &now,
	}
	data, err := json.Marshal(record)
	if err == nil {
		r.rdb.Set(ctx, r.redisKey(key), data, ttl)
	}

	return nil
}
