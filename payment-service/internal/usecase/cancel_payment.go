package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/fluuid/payment-service/internal/domain"
)

// CancelPaymentInput holds the input for cancelling a payment.
type CancelPaymentInput struct {
	TransactionID  string
	IdempotencyKey string
	Reason         string
	AmountMinor    *int64 // nil = full cancel
	CorrelationID  string
}

// CancelPaymentUseCase handles payment cancellation.
type CancelPaymentUseCase struct {
	paymentRepo domain.PaymentRepository
	outboxRepo  domain.OutboxRepository
	txManager   domain.TxManager
	gateway     domain.PaymentGateway
	logger      *slog.Logger
}

// NewCancelPaymentUseCase creates a new CancelPaymentUseCase.
func NewCancelPaymentUseCase(
	paymentRepo domain.PaymentRepository,
	outboxRepo domain.OutboxRepository,
	txManager domain.TxManager,
	gateway domain.PaymentGateway,
	logger *slog.Logger,
) *CancelPaymentUseCase {
	return &CancelPaymentUseCase{
		paymentRepo: paymentRepo,
		outboxRepo:  outboxRepo,
		txManager:   txManager,
		gateway:     gateway,
		logger:      logger,
	}
}

// Execute cancels a payment if the state machine allows it.
func (uc *CancelPaymentUseCase) Execute(ctx context.Context, input CancelPaymentInput) (*CreatePaymentOutput, error) {
	log := uc.logger.With(
		"correlation_id", input.CorrelationID,
		"transaction_id", input.TransactionID,
	)

	// Retrieve the payment
	payment, err := uc.paymentRepo.GetByID(ctx, input.TransactionID)
	if err != nil {
		return nil, err
	}

	// Check if cancellation is allowed
	if !payment.CanTransitionTo(domain.PaymentCancelled) {
		return nil, fmt.Errorf("%w: cannot cancel payment in status %s",
			domain.ErrInvalidTransition, payment.Status)
	}

	// Call gateway to void/cancel
	if payment.ProviderTransactionID != "" {
		if err := uc.gateway.VoidCharge(ctx, payment.ProviderTransactionID); err != nil {
			log.Error("gateway void failed", "error", err)
			return nil, fmt.Errorf("void charge: %w", err)
		}
	}

	// Transition state
	if err := payment.Cancel(); err != nil {
		return nil, err
	}

	// Create cancellation event
	event := domain.NewPaymentEvent(payment, domain.EventPaymentCancelled, input.CorrelationID)
	eventPayload, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal cancel event: %w", err)
	}

	outboxEntry := &domain.OutboxEntry{
		ID:            uuid.NewString(),
		AggregateID:   payment.ID,
		EventType:     domain.EventPaymentCancelled,
		Payload:       eventPayload,
		CorrelationID: input.CorrelationID,
		Status:        domain.OutboxPending,
		CreatedAt:     time.Now().UTC(),
	}

	// Persist atomically
	err = uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if err := uc.paymentRepo.UpdateStatus(txCtx, payment); err != nil {
			return err
		}
		return uc.outboxRepo.Insert(txCtx, outboxEntry)
	})
	if err != nil {
		return nil, err
	}

	log.Info("payment cancelled", "transaction_id", payment.ID)

	return &CreatePaymentOutput{
		TransactionID:         payment.ID,
		OrderID:               payment.OrderID,
		UserID:                payment.UserID,
		Status:                payment.Status,
		AmountMinor:           payment.Amount.AmountMinor,
		Currency:              payment.Amount.Currency,
		ProviderTransactionID: payment.ProviderTransactionID,
		CreatedAt:             payment.CreatedAt,
		UpdatedAt:             payment.UpdatedAt,
		HTTPCode:              200,
	}, nil
}
