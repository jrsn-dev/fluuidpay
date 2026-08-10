package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fluuid/payment-service/internal/domain"
)

// PaymentRepo implements domain.PaymentRepository using PostgreSQL.
type PaymentRepo struct {
	pool *pgxpool.Pool
}

// NewPaymentRepo creates a new payment repository.
func NewPaymentRepo(pool *pgxpool.Pool) *PaymentRepo {
	return &PaymentRepo{pool: pool}
}

// getConn returns the transaction from context if available, otherwise the pool.
func (r *PaymentRepo) getConn(ctx context.Context) interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
} {
	// Check for transaction in context (set by TxManager)
	if tx, ok := ctx.Value(txContextKey{}).(pgx.Tx); ok {
		return tx
	}
	return r.pool
}

type txContextKey struct{}

// Create persists a new payment.
func (r *PaymentRepo) Create(ctx context.Context, payment *domain.Payment) error {
	query := `
		INSERT INTO payments (
			id, order_id, user_id, amount_minor, currency, status,
			card_token_hash, provider_transaction_id, state_code, metadata,
			failure_code, failure_category, failure_message,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	var failCode, failCategory, failMessage *string
	if payment.Failure != nil {
		fc := payment.Failure.Code
		fcat := string(payment.Failure.Category)
		fm := payment.Failure.Message
		failCode = &fc
		failCategory = &fcat
		failMessage = &fm
	}

	_, err := r.pool.Exec(ctx, query,
		payment.ID,
		payment.OrderID,
		payment.UserID,
		payment.Amount.AmountMinor,
		payment.Amount.Currency,
		string(payment.Status),
		payment.CardTokenHash,
		nilIfEmpty(payment.ProviderTransactionID),
		payment.StateCode,
		payment.Metadata,
		failCode,
		failCategory,
		failMessage,
		payment.CreatedAt,
		payment.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert payment: %w", err)
	}
	return nil
}

// GetByID retrieves a payment by its transaction ID.
func (r *PaymentRepo) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	query := `
		SELECT id, order_id, user_id, amount_minor, currency, status,
		       card_token_hash, provider_transaction_id, state_code, metadata,
		       failure_code, failure_category, failure_message,
		       created_at, updated_at
		FROM payments
		WHERE id = $1
	`

	var p domain.Payment
	var amountMinor int64
	var currency string
	var status string
	var providerTxnID *string
	var failCode, failCategory, failMessage *string

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.OrderID, &p.UserID, &amountMinor, &currency, &status,
		&p.CardTokenHash, &providerTxnID, &p.StateCode, &p.Metadata,
		&failCode, &failCategory, &failMessage,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("query payment: %w", err)
	}

	p.Amount = domain.Money{AmountMinor: amountMinor, Currency: currency}
	p.Status = domain.PaymentStatus(status)
	if providerTxnID != nil {
		p.ProviderTransactionID = *providerTxnID
	}
	if failCode != nil {
		p.Failure = &domain.PaymentFailure{
			Code:     *failCode,
			Category: domain.FailureCategory(*failCategory),
			Message:  *failMessage,
		}
	}

	return &p, nil
}

// GetByOrderID retrieves all payments for a given order.
func (r *PaymentRepo) GetByOrderID(ctx context.Context, orderID string) ([]*domain.Payment, error) {
	query := `
		SELECT id, order_id, user_id, amount_minor, currency, status,
		       card_token_hash, provider_transaction_id, state_code, metadata,
		       failure_code, failure_category, failure_message,
		       created_at, updated_at
		FROM payments
		WHERE order_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("query payments by order: %w", err)
	}
	defer rows.Close()

	var payments []*domain.Payment
	for rows.Next() {
		var p domain.Payment
		var amountMinor int64
		var currency, status string
		var providerTxnID *string
		var failCode, failCategory, failMessage *string

		if err := rows.Scan(
			&p.ID, &p.OrderID, &p.UserID, &amountMinor, &currency, &status,
			&p.CardTokenHash, &providerTxnID, &p.StateCode, &p.Metadata,
			&failCode, &failCategory, &failMessage,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}

		p.Amount = domain.Money{AmountMinor: amountMinor, Currency: currency}
		p.Status = domain.PaymentStatus(status)
		if providerTxnID != nil {
			p.ProviderTransactionID = *providerTxnID
		}
		if failCode != nil {
			p.Failure = &domain.PaymentFailure{
				Code:     *failCode,
				Category: domain.FailureCategory(*failCategory),
				Message:  *failMessage,
			}
		}
		payments = append(payments, &p)
	}

	return payments, nil
}

// UpdateStatus updates payment status and related fields.
func (r *PaymentRepo) UpdateStatus(ctx context.Context, payment *domain.Payment) error {
	query := `
		UPDATE payments
		SET status = $2,
		    provider_transaction_id = $3,
		    failure_code = $4,
		    failure_category = $5,
		    failure_message = $6,
		    updated_at = $7
		WHERE id = $1
	`

	var failCode, failCategory, failMessage *string
	if payment.Failure != nil {
		fc := payment.Failure.Code
		fcat := string(payment.Failure.Category)
		fm := payment.Failure.Message
		failCode = &fc
		failCategory = &fcat
		failMessage = &fm
	}

	_, err := r.pool.Exec(ctx, query,
		payment.ID,
		string(payment.Status),
		nilIfEmpty(payment.ProviderTransactionID),
		failCode, failCategory, failMessage,
		payment.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update payment status: %w", err)
	}
	return nil
}

// ListPendingForReconciliation returns payments stuck in non-final states.
func (r *PaymentRepo) ListPendingForReconciliation(ctx context.Context, olderThan time.Duration) ([]*domain.Payment, error) {
	query := `
		SELECT id, order_id, user_id, amount_minor, currency, status,
		       card_token_hash, provider_transaction_id, state_code, metadata,
		       failure_code, failure_category, failure_message,
		       created_at, updated_at
		FROM payments
		WHERE status IN ('CREATED', 'PENDING', 'AUTHORIZED')
		  AND updated_at < $1
		ORDER BY updated_at ASC
		LIMIT 100
	`

	threshold := time.Now().UTC().Add(-olderThan)
	rows, err := r.pool.Query(ctx, query, threshold)
	if err != nil {
		return nil, fmt.Errorf("query pending payments: %w", err)
	}
	defer rows.Close()

	var payments []*domain.Payment
	for rows.Next() {
		var p domain.Payment
		var amountMinor int64
		var currency, status string
		var providerTxnID *string
		var failCode, failCategory, failMessage *string

		if err := rows.Scan(
			&p.ID, &p.OrderID, &p.UserID, &amountMinor, &currency, &status,
			&p.CardTokenHash, &providerTxnID, &p.StateCode, &p.Metadata,
			&failCode, &failCategory, &failMessage,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan pending payment: %w", err)
		}

		p.Amount = domain.Money{AmountMinor: amountMinor, Currency: currency}
		p.Status = domain.PaymentStatus(status)
		if providerTxnID != nil {
			p.ProviderTransactionID = *providerTxnID
		}
		if failCode != nil {
			p.Failure = &domain.PaymentFailure{
				Code:     *failCode,
				Category: domain.FailureCategory(*failCategory),
				Message:  *failMessage,
			}
		}
		payments = append(payments, &p)
	}

	return payments, nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
