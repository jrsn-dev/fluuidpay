package domain

import (
	"context"
	"time"
)

// PaymentRepository persists and retrieves payments from the durable store.
type PaymentRepository interface {
	// Create persists a new payment. Must be called within a transaction.
	Create(ctx context.Context, payment *Payment) error

	// GetByID retrieves a payment by its transaction ID.
	GetByID(ctx context.Context, id string) (*Payment, error)

	// GetByOrderID retrieves all payments for a given order.
	GetByOrderID(ctx context.Context, orderID string) ([]*Payment, error)

	// UpdateStatus updates the status and related fields of a payment.
	UpdateStatus(ctx context.Context, payment *Payment) error

	// ListPendingForReconciliation returns payments stuck in non-final states
	// older than the given threshold, for the reconciliation worker.
	ListPendingForReconciliation(ctx context.Context, olderThan time.Duration) ([]*Payment, error)
}

// IdempotencyStore provides idempotency key management.
// The implementation should use Redis for fast lookup and PostgreSQL for durability.
type IdempotencyStore interface {
	// Get retrieves the stored response for an idempotency key.
	// Returns nil, nil if the key does not exist.
	Get(ctx context.Context, key string) (*IdempotencyRecord, error)

	// Reserve attempts to acquire the idempotency key with atomic SET NX.
	// Returns true if the key was successfully reserved (new operation).
	// Returns false if the key already exists (duplicate).
	Reserve(ctx context.Context, key string, payloadHash string, ttl time.Duration) (bool, error)

	// Complete marks the idempotency key as completed with the final response.
	Complete(ctx context.Context, key string, response []byte, httpCode int, ttl time.Duration) error
}

// IdempotencyRecord holds the state of an idempotency key.
type IdempotencyRecord struct {
	Key           string
	PayloadHash   string
	Status        IdempotencyStatus
	Response      []byte
	HTTPCode      int
	TransactionID string
	CreatedAt     time.Time
	CompletedAt   *time.Time
}

// IdempotencyStatus represents the lifecycle of an idempotency record.
type IdempotencyStatus string

const (
	IdempotencyProcessing IdempotencyStatus = "PROCESSING"
	IdempotencyCompleted  IdempotencyStatus = "COMPLETED"
)

// PaymentGateway defines the interface for external payment providers.
// Implementations must never log or return raw PAN/CVV data.
type PaymentGateway interface {
	// ProcessCharge sends a charge request to the payment provider.
	// The card_token is a secure token issued by the provider's tokenization service.
	ProcessCharge(ctx context.Context, req GatewayChargeRequest) (*GatewayChargeResponse, error)

	// QueryCharge checks the status of a previously submitted charge.
	// Used by the reconciliation worker for uncertain/timeout scenarios.
	QueryCharge(ctx context.Context, providerTransactionID string) (*GatewayChargeResponse, error)

	// VoidCharge cancels an authorized but not captured charge.
	VoidCharge(ctx context.Context, providerTransactionID string) error

	// RefundCharge initiates a refund for a captured charge.
	RefundCharge(ctx context.Context, providerTransactionID string, amountMinor int64) error
}

// GatewayChargeRequest is the data sent to the payment gateway.
type GatewayChargeRequest struct {
	IdempotencyKey string
	CardToken      string
	AmountMinor    int64
	Currency       string
	OrderID        string
	Capture        bool // true for auto-capture, false for authorize-only
}

// GatewayChargeResponse is the sanitized response from the payment gateway.
// Must never contain raw card data.
type GatewayChargeResponse struct {
	ProviderTransactionID string
	Status                PaymentStatus
	ProviderCode          string // Sanitized provider code (e.g., "do_not_honor")
	Message               string // Sanitized message for the client
}

// TaxCalculator computes taxes for a transaction.
type TaxCalculator interface {
	// Calculate computes IBS/CBS taxes based on the transaction context.
	Calculate(ctx context.Context, input TaxCalculationInput) (*TaxDetails, error)
}

// EventPublisher publishes domain events to the message broker.
type EventPublisher interface {
	// Publish sends an event to the broker with confirmation.
	Publish(ctx context.Context, event PaymentEvent) error
}

// OutboxRepository manages the transactional outbox for reliable event publishing.
type OutboxRepository interface {
	// Insert creates a new outbox entry within the current transaction.
	Insert(ctx context.Context, entry *OutboxEntry) error

	// FetchPending retrieves unprocessed events ready for publishing.
	// Locks the rows to prevent concurrent publishers from processing the same event.
	FetchPending(ctx context.Context, limit int) ([]*OutboxEntry, error)

	// MarkPublished updates an outbox entry as successfully published.
	MarkPublished(ctx context.Context, id string) error

	// MarkFailed updates an outbox entry with failure information.
	MarkFailed(ctx context.Context, id string, err error, retryable bool) error
}

// ProcessedEventStore tracks which events have already been consumed,
// enabling idempotent message processing.
type ProcessedEventStore interface {
	// AlreadyProcessed checks if the event was already consumed.
	AlreadyProcessed(ctx context.Context, eventID string) (bool, error)

	// MarkProcessed records the event as successfully consumed.
	MarkProcessed(ctx context.Context, eventID, consumer string) error
}

// TxManager provides transaction management for use cases that need
// to persist payment + tax + outbox atomically.
type TxManager interface {
	// WithTx executes the given function within a database transaction.
	// If fn returns nil, the transaction is committed; otherwise it is rolled back.
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
