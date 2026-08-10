package usecase_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/fluuid/payment-service/internal/domain"
	"github.com/fluuid/payment-service/internal/usecase"
)

func TestCancelPaymentUseCase_Success(t *testing.T) {
	payment, _ := domain.NewPayment("user1", "order1", domain.Money{AmountMinor: 100, Currency: "BRL"}, "hash", "SP", nil)
	_ = payment.MarkPending()
	_ = payment.Authorize("provider-123")

	repo := &mockPaymentRepo{
		payments: map[string]*domain.Payment{
			payment.ID: payment,
		},
	}
	outb := &mockOutboxRepo{}
	txm := &mockTxManager{}
	gw := &mockGateway{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	uc := usecase.NewCancelPaymentUseCase(repo, outb, txm, gw, logger)

	input := usecase.CancelPaymentInput{
		TransactionID: payment.ID,
		CorrelationID: "corr-1",
	}

	out, err := uc.Execute(context.Background(), input)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if out.Status != domain.PaymentCancelled {
		t.Errorf("expected CANCELLED, got %s", out.Status)
	}

	if len(outb.entries) != 1 {
		t.Fatalf("expected 1 outbox entry, got %d", len(outb.entries))
	}
	if outb.entries[0].EventType != domain.EventPaymentCancelled {
		t.Errorf("expected event PaymentCancelled, got %s", outb.entries[0].EventType)
	}
}

func TestCancelPaymentUseCase_InvalidState(t *testing.T) {
	payment, _ := domain.NewPayment("user1", "order1", domain.Money{AmountMinor: 100, Currency: "BRL"}, "hash", "SP", nil)
	_ = payment.MarkPending()
	_ = payment.Approve("provider-123")
	_ = payment.Refund() // Now in REFUNDED state

	repo := &mockPaymentRepo{
		payments: map[string]*domain.Payment{
			payment.ID: payment,
		},
	}
	outb := &mockOutboxRepo{}
	txm := &mockTxManager{}
	gw := &mockGateway{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	uc := usecase.NewCancelPaymentUseCase(repo, outb, txm, gw, logger)

	input := usecase.CancelPaymentInput{
		TransactionID: payment.ID,
	}

	_, err := uc.Execute(context.Background(), input)

	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Errorf("expected ErrInvalidTransition, got %v", err)
	}
}
