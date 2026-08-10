package domain

import (
	"time"

	"github.com/google/uuid"
)

// PaymentStatus represents the state of a payment in the state machine.
type PaymentStatus string

const (
	PaymentCreated           PaymentStatus = "CREATED"
	PaymentPending           PaymentStatus = "PENDING"
	PaymentAuthorized        PaymentStatus = "AUTHORIZED"
	PaymentCaptured          PaymentStatus = "CAPTURED"
	PaymentApproved          PaymentStatus = "APPROVED"
	PaymentRejected          PaymentStatus = "REJECTED"
	PaymentCancelled         PaymentStatus = "CANCELLED"
	PaymentRefunded          PaymentStatus = "REFUNDED"
	PaymentPartiallyRefunded PaymentStatus = "PARTIALLY_REFUNDED"
)

// validTransitions defines the state machine for payment lifecycle.
// Key: current state → Value: set of allowed next states.
var validTransitions = map[PaymentStatus]map[PaymentStatus]bool{
	PaymentCreated: {
		PaymentPending: true,
	},
	PaymentPending: {
		PaymentAuthorized: true,
		PaymentApproved:   true, // For auto-capture flows
		PaymentRejected:   true,
	},
	PaymentAuthorized: {
		PaymentCaptured:  true,
		PaymentCancelled: true,
	},
	PaymentCaptured: {
		PaymentRefunded:          true,
		PaymentPartiallyRefunded: true,
	},
	PaymentApproved: {
		PaymentRefunded:          true,
		PaymentPartiallyRefunded: true,
		PaymentCancelled:         true,
	},
	PaymentPartiallyRefunded: {
		PaymentRefunded:          true,
		PaymentPartiallyRefunded: true, // Additional partial refunds
	},
}

// Payment is the core aggregate of the payment domain.
type Payment struct {
	ID                    string
	OrderID               string
	UserID                string
	Amount                Money
	Status                PaymentStatus
	CardTokenHash         string // Never store the token itself, only a hash for auditing
	ProviderTransactionID string
	StateCode             string // Destination state for tax calculation
	Metadata              map[string]string
	Taxes                 *TaxDetails
	Failure               *PaymentFailure
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// PaymentFailure captures details about why a payment was rejected.
type PaymentFailure struct {
	Code     string
	Category FailureCategory
	Message  string
}

// FailureCategory classifies the type of payment failure.
type FailureCategory string

const (
	FailureCategoryValidation FailureCategory = "validation"
	FailureCategoryGateway    FailureCategory = "gateway"
	FailureCategoryTimeout    FailureCategory = "timeout"
	FailureCategoryFraud      FailureCategory = "fraud"
	FailureCategoryTax        FailureCategory = "tax"
	FailureCategoryInternal   FailureCategory = "internal"
)

// NewPayment creates a new Payment in CREATED state.
func NewPayment(userID, orderID string, amount Money, cardTokenHash, stateCode string, metadata map[string]string) (*Payment, error) {
	if userID == "" {
		return nil, ErrInvalidUserID
	}
	if orderID == "" {
		return nil, ErrInvalidOrderID
	}
	if !amount.IsPositive() {
		return nil, ErrInvalidAmount
	}
	if cardTokenHash == "" {
		return nil, ErrInvalidCardToken
	}
	if stateCode == "" {
		return nil, ErrInvalidStateCode
	}

	now := time.Now().UTC()
	return &Payment{
		ID:            uuid.NewString(),
		OrderID:       orderID,
		UserID:        userID,
		Amount:        amount,
		Status:        PaymentCreated,
		CardTokenHash: cardTokenHash,
		StateCode:     stateCode,
		Metadata:      metadata,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

// CanTransitionTo checks if the state machine allows moving to the target status.
func (p *Payment) CanTransitionTo(target PaymentStatus) bool {
	if allowed, ok := validTransitions[p.Status]; ok {
		return allowed[target]
	}
	return false
}

// TransitionTo attempts to move the payment to a new status.
// Returns ErrInvalidTransition if the transition is not allowed.
func (p *Payment) TransitionTo(target PaymentStatus) error {
	if !p.CanTransitionTo(target) {
		return ErrInvalidTransition
	}
	p.Status = target
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// GetMetadata returns a defensive copy of the payment metadata.
// This prevents accidental mutation of the internal map by external packages.
func (p *Payment) GetMetadata() map[string]string {
	if p.Metadata == nil {
		return nil
	}
	copy := make(map[string]string, len(p.Metadata))
	for k, v := range p.Metadata {
		copy[k] = v
	}
	return copy
}

// MarkPending transitions to PENDING and records the gateway attempt.
func (p *Payment) MarkPending() error {
	return p.TransitionTo(PaymentPending)
}

// Approve transitions to APPROVED (auto-capture) and records provider ID.
func (p *Payment) Approve(providerTransactionID string) error {
	if err := p.TransitionTo(PaymentApproved); err != nil {
		return err
	}
	p.ProviderTransactionID = providerTransactionID
	return nil
}

// Authorize transitions to AUTHORIZED (separate capture).
func (p *Payment) Authorize(providerTransactionID string) error {
	if err := p.TransitionTo(PaymentAuthorized); err != nil {
		return err
	}
	p.ProviderTransactionID = providerTransactionID
	return nil
}

// Reject transitions to REJECTED with failure details.
func (p *Payment) Reject(failure PaymentFailure) error {
	if err := p.TransitionTo(PaymentRejected); err != nil {
		return err
	}
	p.Failure = &failure
	return nil
}

// Cancel transitions to CANCELLED.
func (p *Payment) Cancel() error {
	return p.TransitionTo(PaymentCancelled)
}

// Capture transitions from AUTHORIZED to CAPTURED.
func (p *Payment) Capture() error {
	return p.TransitionTo(PaymentCaptured)
}

// Refund transitions to REFUNDED.
func (p *Payment) Refund() error {
	return p.TransitionTo(PaymentRefunded)
}

// PartialRefund transitions to PARTIALLY_REFUNDED.
func (p *Payment) PartialRefund() error {
	return p.TransitionTo(PaymentPartiallyRefunded)
}

// SetTaxes attaches tax calculation results to the payment.
func (p *Payment) SetTaxes(taxes TaxDetails) {
	p.Taxes = &taxes
	p.UpdatedAt = time.Now().UTC()
}

// IsFinal returns true if the payment is in a terminal state.
func (p *Payment) IsFinal() bool {
	switch p.Status {
	case PaymentRejected, PaymentCancelled, PaymentRefunded:
		return true
	default:
		return false
	}
}
