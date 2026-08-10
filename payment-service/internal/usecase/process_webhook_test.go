package usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/fluuid/payment-service/internal/domain"
	"github.com/fluuid/payment-service/internal/usecase"
)

func TestProcessWebhookUseCase_ValidSignature_Approved(t *testing.T) {
	payment, _ := domain.NewPayment("user1", "order1", domain.Money{AmountMinor: 100, Currency: "BRL"}, "hash", "SP", nil)
	_ = payment.MarkPending() // Must be in pending to approve

	repo := &mockPaymentRepo{
		payments: map[string]*domain.Payment{
			payment.ID: payment,
		},
	}
	outb := &mockOutboxRepo{}
	audit := &mockAuditRepo{}
	txm := &mockTxManager{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	uc := usecase.NewProcessWebhookUseCase(repo, outb, audit, txm, "", logger) // Empty secret bypasses signature validation

	payload := usecase.WebhookPayload{
		ProviderTransactionID: "prov-1",
		Status:                "approved",
		OccurredAt:            time.Now(),
	}
	payloadBytes, _ := json.Marshal(payload)

	err := uc.Execute(context.Background(), payloadBytes, "any-sig", payment.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if payment.Status != domain.PaymentApproved {
		t.Errorf("expected APPROVED, got %s", payment.Status)
	}

	if len(outb.entries) != 1 {
		t.Fatalf("expected 1 outbox entry, got %d", len(outb.entries))
	}
	if outb.entries[0].EventType != domain.EventPaymentApproved {
		t.Errorf("expected PaymentApproved event, got %s", outb.entries[0].EventType)
	}
}

func TestProcessWebhookUseCase_InvalidSignature(t *testing.T) {
	repo := &mockPaymentRepo{}
	outb := &mockOutboxRepo{}
	audit := &mockAuditRepo{}
	txm := &mockTxManager{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	uc := usecase.NewProcessWebhookUseCase(repo, outb, audit, txm, "my-secret", logger)

	err := uc.Execute(context.Background(), []byte("{}"), "invalid-sig", "txn-1")
	if !errors.Is(err, usecase.ErrInvalidWebhookSignature) {
		t.Errorf("expected ErrInvalidWebhookSignature, got %v", err)
	}
}
