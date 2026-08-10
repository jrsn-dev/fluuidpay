package gateway

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"

	"github.com/fluuid/payment-service/internal/domain"
)

// MockGateway simulates a payment gateway for development and testing.
// In production, this would be replaced with a real provider adapter (e.g., Stripe, PagSeguro).
type MockGateway struct {
	timeout        time.Duration
	approvalRate   float64 // Probability of approval (0.0 to 1.0)
	simulateDelay  bool
	maxDelayMs     int
}

// MockGatewayOption configures the mock gateway.
type MockGatewayOption func(*MockGateway)

// WithApprovalRate sets the approval probability.
func WithApprovalRate(rate float64) MockGatewayOption {
	return func(g *MockGateway) {
		g.approvalRate = rate
	}
}

// WithSimulatedDelay enables random processing delay.
func WithSimulatedDelay(maxMs int) MockGatewayOption {
	return func(g *MockGateway) {
		g.simulateDelay = true
		g.maxDelayMs = maxMs
	}
}

// NewMockGateway creates a mock payment gateway.
func NewMockGateway(opts ...MockGatewayOption) *MockGateway {
	g := &MockGateway{
		timeout:      10 * time.Second,
		approvalRate: 0.9, // 90% approval rate by default
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// ProcessCharge simulates a charge request.
func (g *MockGateway) ProcessCharge(ctx context.Context, req domain.GatewayChargeRequest) (*domain.GatewayChargeResponse, error) {
	// Simulate processing delay
	if g.simulateDelay && g.maxDelayMs > 0 {
		delay := time.Duration(rand.Intn(g.maxDelayMs)) * time.Millisecond
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, domain.ErrGatewayTimeout
		}
	}

	// Check context cancellation
	if ctx.Err() != nil {
		return nil, domain.ErrGatewayTimeout
	}

	providerTxnID := "mock_" + uuid.NewString()[:12]

	// Simulate approval/rejection based on configured rate
	if rand.Float64() < g.approvalRate {
		status := domain.PaymentApproved
		if !req.Capture {
			status = domain.PaymentAuthorized
		}
		return &domain.GatewayChargeResponse{
			ProviderTransactionID: providerTxnID,
			Status:                status,
			ProviderCode:          "approved",
			Message:               "Transaction approved",
		}, nil
	}

	// Simulate rejection with common codes
	rejectionCodes := []struct {
		code    string
		message string
	}{
		{"do_not_honor", "Transaction not authorized"},
		{"insufficient_funds", "Insufficient funds"},
		{"expired_card", "Card expired"},
		{"invalid_transaction", "Invalid transaction"},
	}

	rejection := rejectionCodes[rand.Intn(len(rejectionCodes))]
	return &domain.GatewayChargeResponse{
		ProviderTransactionID: providerTxnID,
		Status:                domain.PaymentRejected,
		ProviderCode:          rejection.code,
		Message:               rejection.message,
	}, nil
}

// QueryCharge simulates querying a charge status.
func (g *MockGateway) QueryCharge(ctx context.Context, providerTransactionID string) (*domain.GatewayChargeResponse, error) {
	return &domain.GatewayChargeResponse{
		ProviderTransactionID: providerTransactionID,
		Status:                domain.PaymentApproved,
		ProviderCode:          "approved",
		Message:               "Transaction approved",
	}, nil
}

// VoidCharge simulates voiding a charge.
func (g *MockGateway) VoidCharge(ctx context.Context, providerTransactionID string) error {
	if providerTransactionID == "" {
		return fmt.Errorf("provider transaction ID is required")
	}
	return nil
}

// RefundCharge simulates refunding a charge.
func (g *MockGateway) RefundCharge(ctx context.Context, providerTransactionID string, amountMinor int64) error {
	if providerTransactionID == "" {
		return fmt.Errorf("provider transaction ID is required")
	}
	if amountMinor <= 0 {
		return fmt.Errorf("refund amount must be positive")
	}
	return nil
}
