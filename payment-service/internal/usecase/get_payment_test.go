package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/fluuid/payment-service/internal/domain"
	"github.com/fluuid/payment-service/internal/usecase"
)

func TestGetPaymentUseCase_Success(t *testing.T) {
	payment, _ := domain.NewPayment("user1", "order1", domain.Money{AmountMinor: 100, Currency: "BRL"}, "hash", "SP", nil)
	repo := &mockPaymentRepo{
		payments: map[string]*domain.Payment{
			payment.ID: payment,
		},
	}
	uc := usecase.NewGetPaymentUseCase(repo)

	out, err := uc.Execute(context.Background(), payment.ID)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if out == nil {
		t.Fatal("expected output")
	}
	if out.TransactionID != payment.ID {
		t.Errorf("expected transaction ID %s, got %s", payment.ID, out.TransactionID)
	}
	if out.HTTPCode != 200 {
		t.Errorf("expected HTTP 200, got %d", out.HTTPCode)
	}
}

func TestGetPaymentUseCase_NotFound(t *testing.T) {
	repo := &mockPaymentRepo{payments: make(map[string]*domain.Payment)}
	uc := usecase.NewGetPaymentUseCase(repo)

	_, err := uc.Execute(context.Background(), "unknown-id")

	if !errors.Is(err, domain.ErrPaymentNotFound) {
		t.Errorf("expected ErrPaymentNotFound, got %v", err)
	}
}
