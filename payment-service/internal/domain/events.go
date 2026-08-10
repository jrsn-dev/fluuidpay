package domain

import "time"

// EventType represents the kind of domain event.
type EventType string

const (
	EventPaymentPending           EventType = "PaymentPending"
	EventPaymentApproved          EventType = "PaymentApproved"
	EventPaymentRejected          EventType = "PaymentRejected"
	EventPaymentCancelled         EventType = "PaymentCancelled"
	EventPaymentCaptured          EventType = "PaymentCaptured"
	EventPaymentRefunded          EventType = "PaymentRefunded"
	EventPaymentPartiallyRefunded EventType = "PaymentPartiallyRefunded"
)

// PaymentEvent is the domain event envelope published when a payment transitions.
type PaymentEvent struct {
	EventID       string    `json:"event_id"`
	EventType     EventType `json:"event_type"`
	SchemaVersion int       `json:"schema_version"`
	Producer      string    `json:"producer"`
	OccurredAt    time.Time `json:"occurred_at"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	CausationID   string    `json:"causation_id,omitempty"`
	Data          EventData `json:"data"`
}

// EventData holds the payment-specific data inside the event envelope.
type EventData struct {
	TransactionID string      `json:"transaction_id"`
	OrderID       string      `json:"order_id"`
	UserID        string      `json:"user_id"`
	Status        string      `json:"status"`
	AmountMinor   int64       `json:"amount_minor"`
	Currency      string      `json:"currency"`
	Taxes         *TaxDetails `json:"taxes,omitempty"`
}

// OutboxEntry represents a pending event in the transactional outbox table.
type OutboxEntry struct {
	ID            string
	AggregateID   string
	EventType     EventType
	Payload       []byte
	CorrelationID string
	Status        OutboxStatus
	AttemptCount  int
	NextAttemptAt time.Time
	LastError     string
	LockedAt      *time.Time
	CreatedAt     time.Time
	PublishedAt   *time.Time
}

// OutboxStatus represents the lifecycle of an outbox event.
type OutboxStatus string

const (
	OutboxPending          OutboxStatus = "PENDING"
	OutboxPublishing       OutboxStatus = "PUBLISHING"
	OutboxPublished        OutboxStatus = "PUBLISHED"
	OutboxFailedRetryable  OutboxStatus = "FAILED_RETRYABLE"
	OutboxFailedPermanent  OutboxStatus = "FAILED_PERMANENT"
)

// NewPaymentEvent creates a domain event from a Payment entity.
func NewPaymentEvent(payment *Payment, eventType EventType, correlationID string) PaymentEvent {
	event := PaymentEvent{
		EventID:       payment.ID + "-" + string(eventType),
		EventType:     eventType,
		SchemaVersion: 1,
		Producer:      "payment-service",
		OccurredAt:    time.Now().UTC(),
		CorrelationID: correlationID,
		Data: EventData{
			TransactionID: payment.ID,
			OrderID:       payment.OrderID,
			UserID:        payment.UserID,
			Status:        string(payment.Status),
			AmountMinor:   payment.Amount.AmountMinor,
			Currency:      payment.Amount.Currency,
		},
	}
	if payment.Taxes != nil {
		event.Data.Taxes = payment.Taxes
	}
	return event
}

// EventTypeFromStatus maps a PaymentStatus to the corresponding EventType.
func EventTypeFromStatus(status PaymentStatus) EventType {
	switch status {
	case PaymentPending:
		return EventPaymentPending
	case PaymentApproved:
		return EventPaymentApproved
	case PaymentRejected:
		return EventPaymentRejected
	case PaymentCancelled:
		return EventPaymentCancelled
	case PaymentCaptured:
		return EventPaymentCaptured
	case PaymentRefunded:
		return EventPaymentRefunded
	case PaymentPartiallyRefunded:
		return EventPaymentPartiallyRefunded
	default:
		return EventType("Unknown")
	}
}
