package domain_test

import (
	"testing"

	"github.com/fluuid/payment-service/internal/domain"
)

func TestPaymentStateTransitions(t *testing.T) {
	tests := []struct {
		name       string
		from       domain.PaymentStatus
		to         domain.PaymentStatus
		shouldPass bool
	}{
		{"CREATED → PENDING", domain.PaymentCreated, domain.PaymentPending, true},
		{"PENDING → APPROVED", domain.PaymentPending, domain.PaymentApproved, true},
		{"PENDING → AUTHORIZED", domain.PaymentPending, domain.PaymentAuthorized, true},
		{"PENDING → REJECTED", domain.PaymentPending, domain.PaymentRejected, true},
		{"AUTHORIZED → CAPTURED", domain.PaymentAuthorized, domain.PaymentCaptured, true},
		{"AUTHORIZED → CANCELLED", domain.PaymentAuthorized, domain.PaymentCancelled, true},
		{"CAPTURED → REFUNDED", domain.PaymentCaptured, domain.PaymentRefunded, true},
		{"CAPTURED → PARTIALLY_REFUNDED", domain.PaymentCaptured, domain.PaymentPartiallyRefunded, true},
		{"APPROVED → CANCELLED", domain.PaymentApproved, domain.PaymentCancelled, true},
		{"APPROVED → REFUNDED", domain.PaymentApproved, domain.PaymentRefunded, true},

		// Invalid transitions
		{"CREATED → APPROVED (invalid)", domain.PaymentCreated, domain.PaymentApproved, false},
		{"CREATED → REJECTED (invalid)", domain.PaymentCreated, domain.PaymentRejected, false},
		{"REJECTED → APPROVED (invalid)", domain.PaymentRejected, domain.PaymentApproved, false},
		{"CANCELLED → APPROVED (invalid)", domain.PaymentCancelled, domain.PaymentApproved, false},
		{"REFUNDED → PENDING (invalid)", domain.PaymentRefunded, domain.PaymentPending, false},
		{"PENDING → CAPTURED (invalid)", domain.PaymentPending, domain.PaymentCaptured, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment := createTestPayment(t)
			// Force the payment to the 'from' state for testing
			setPaymentStatus(payment, tt.from)

			err := payment.TransitionTo(tt.to)
			if tt.shouldPass && err != nil {
				t.Errorf("expected transition %s → %s to succeed, got error: %v", tt.from, tt.to, err)
			}
			if !tt.shouldPass && err == nil {
				t.Errorf("expected transition %s → %s to fail, but it succeeded", tt.from, tt.to)
			}
		})
	}
}

func TestNewPayment_Valid(t *testing.T) {
	money := domain.Money{AmountMinor: 19990, Currency: "BRL"}
	p, err := domain.NewPayment("user-1", "order-1", money, "hash123", "SP", nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if p.Status != domain.PaymentCreated {
		t.Errorf("expected status CREATED, got %s", p.Status)
	}
	if p.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestNewPayment_InvalidInputs(t *testing.T) {
	money := domain.Money{AmountMinor: 19990, Currency: "BRL"}
	tests := []struct {
		name      string
		userID    string
		orderID   string
		amount    domain.Money
		tokenHash string
		stateCode string
		wantErr   error
	}{
		{"empty user_id", "", "order-1", money, "hash", "SP", domain.ErrInvalidUserID},
		{"empty order_id", "user-1", "", money, "hash", "SP", domain.ErrInvalidOrderID},
		{"zero amount", "user-1", "order-1", domain.Money{AmountMinor: 0, Currency: "BRL"}, "hash", "SP", domain.ErrInvalidAmount},
		{"empty token hash", "user-1", "order-1", money, "", "SP", domain.ErrInvalidCardToken},
		{"empty state code", "user-1", "order-1", money, "hash", "", domain.ErrInvalidStateCode},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewPayment(tt.userID, tt.orderID, tt.amount, tt.tokenHash, tt.stateCode, nil)
			if err == nil {
				t.Errorf("expected error %v, got nil", tt.wantErr)
			}
		})
	}
}

func TestPayment_IsFinal(t *testing.T) {
	tests := []struct {
		status  domain.PaymentStatus
		isFinal bool
	}{
		{domain.PaymentCreated, false},
		{domain.PaymentPending, false},
		{domain.PaymentAuthorized, false},
		{domain.PaymentApproved, false},
		{domain.PaymentRejected, true},
		{domain.PaymentCancelled, true},
		{domain.PaymentRefunded, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			p := createTestPayment(t)
			setPaymentStatus(p, tt.status)
			if p.IsFinal() != tt.isFinal {
				t.Errorf("expected IsFinal()=%v for status %s", tt.isFinal, tt.status)
			}
		})
	}
}

func TestPayment_ApproveFlow(t *testing.T) {
	p := createTestPayment(t)

	if err := p.MarkPending(); err != nil {
		t.Fatalf("MarkPending failed: %v", err)
	}
	if p.Status != domain.PaymentPending {
		t.Errorf("expected PENDING, got %s", p.Status)
	}

	if err := p.Approve("provider-123"); err != nil {
		t.Fatalf("Approve failed: %v", err)
	}
	if p.Status != domain.PaymentApproved {
		t.Errorf("expected APPROVED, got %s", p.Status)
	}
	if p.ProviderTransactionID != "provider-123" {
		t.Errorf("expected provider ID 'provider-123', got '%s'", p.ProviderTransactionID)
	}
}

func TestPayment_RejectFlow(t *testing.T) {
	p := createTestPayment(t)
	_ = p.MarkPending()

	failure := domain.PaymentFailure{
		Code:     "do_not_honor",
		Category: domain.FailureCategoryGateway,
		Message:  "Transaction not authorized",
	}
	if err := p.Reject(failure); err != nil {
		t.Fatalf("Reject failed: %v", err)
	}
	if p.Status != domain.PaymentRejected {
		t.Errorf("expected REJECTED, got %s", p.Status)
	}
	if p.Failure == nil {
		t.Fatal("expected failure details to be set")
	}
	if p.Failure.Code != "do_not_honor" {
		t.Errorf("expected failure code 'do_not_honor', got '%s'", p.Failure.Code)
	}
}

// --- Helpers ---

func createTestPayment(t *testing.T) *domain.Payment {
	t.Helper()
	money := domain.Money{AmountMinor: 19990, Currency: "BRL"}
	p, err := domain.NewPayment("user-1", "order-1", money, "tokenhash123", "SP", nil)
	if err != nil {
		t.Fatalf("failed to create test payment: %v", err)
	}
	return p
}

func setPaymentStatus(p *domain.Payment, status domain.PaymentStatus) {
	p.Status = status
}
