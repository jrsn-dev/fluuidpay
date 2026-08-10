package domain_test

import (
	"errors"
	"testing"

	"github.com/fluuid/payment-service/internal/domain"
)

func TestNewMoney_Valid(t *testing.T) {
	m, err := domain.NewMoney(19990, "BRL")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if m.AmountMinor != 19990 {
		t.Errorf("expected amount 19990, got %d", m.AmountMinor)
	}
	if m.Currency != "BRL" {
		t.Errorf("expected currency BRL, got %s", m.Currency)
	}
}

func TestNewMoney_TrimsAndUppercases(t *testing.T) {
	m, err := domain.NewMoney(100, " brl ")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if m.Currency != "BRL" {
		t.Errorf("expected currency BRL, got %s", m.Currency)
	}
}

func TestNewMoney_InvalidCurrency(t *testing.T) {
	_, err := domain.NewMoney(100, "")
	if !errors.Is(err, domain.ErrInvalidCurrency) {
		t.Errorf("expected ErrInvalidCurrency, got: %v", err)
	}

	_, err = domain.NewMoney(100, "AB")
	if !errors.Is(err, domain.ErrInvalidCurrency) {
		t.Errorf("expected ErrInvalidCurrency for 2-char code, got: %v", err)
	}
}

func TestNewMoney_NegativeAmount(t *testing.T) {
	_, err := domain.NewMoney(-100, "BRL")
	if !errors.Is(err, domain.ErrNegativeAmount) {
		t.Errorf("expected ErrNegativeAmount, got: %v", err)
	}
}

func TestMoney_Add(t *testing.T) {
	a := domain.Money{AmountMinor: 1000, Currency: "BRL"}
	b := domain.Money{AmountMinor: 500, Currency: "BRL"}

	result, err := a.Add(b)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if result.AmountMinor != 1500 {
		t.Errorf("expected 1500, got %d", result.AmountMinor)
	}
}

func TestMoney_Add_CurrencyMismatch(t *testing.T) {
	a := domain.Money{AmountMinor: 1000, Currency: "BRL"}
	b := domain.Money{AmountMinor: 500, Currency: "USD"}

	_, err := a.Add(b)
	if !errors.Is(err, domain.ErrCurrencyMismatch) {
		t.Errorf("expected ErrCurrencyMismatch, got: %v", err)
	}
}

func TestMoney_Subtract(t *testing.T) {
	a := domain.Money{AmountMinor: 1000, Currency: "BRL"}
	b := domain.Money{AmountMinor: 400, Currency: "BRL"}

	result, err := a.Subtract(b)
	if err != nil {
		t.Fatalf("Subtract failed: %v", err)
	}
	if result.AmountMinor != 600 {
		t.Errorf("expected 600, got %d", result.AmountMinor)
	}
}

func TestMoney_Subtract_Insufficient(t *testing.T) {
	a := domain.Money{AmountMinor: 100, Currency: "BRL"}
	b := domain.Money{AmountMinor: 200, Currency: "BRL"}

	_, err := a.Subtract(b)
	if !errors.Is(err, domain.ErrInsufficientAmount) {
		t.Errorf("expected ErrInsufficientAmount, got: %v", err)
	}
}

func TestMoney_IsZero(t *testing.T) {
	zero := domain.Money{AmountMinor: 0, Currency: "BRL"}
	nonZero := domain.Money{AmountMinor: 1, Currency: "BRL"}

	if !zero.IsZero() {
		t.Error("expected IsZero() to be true for 0")
	}
	if nonZero.IsZero() {
		t.Error("expected IsZero() to be false for 1")
	}
}

func TestMoney_Equals(t *testing.T) {
	a := domain.Money{AmountMinor: 1000, Currency: "BRL"}
	b := domain.Money{AmountMinor: 1000, Currency: "BRL"}
	c := domain.Money{AmountMinor: 1000, Currency: "USD"}
	d := domain.Money{AmountMinor: 999, Currency: "BRL"}

	if !a.Equals(b) {
		t.Error("expected equal money to be equal")
	}
	if a.Equals(c) {
		t.Error("expected different currencies to not be equal")
	}
	if a.Equals(d) {
		t.Error("expected different amounts to not be equal")
	}
}

func TestMoney_String(t *testing.T) {
	m := domain.Money{AmountMinor: 19990, Currency: "BRL"}
	s := m.String()
	expected := "BRL 199.90"
	if s != expected {
		t.Errorf("expected %q, got %q", expected, s)
	}
}

func TestIsSupportedCurrency(t *testing.T) {
	if !domain.IsSupportedCurrency("BRL") {
		t.Error("BRL should be supported")
	}
	if !domain.IsSupportedCurrency("USD") {
		t.Error("USD should be supported")
	}
	if domain.IsSupportedCurrency("XYZ") {
		t.Error("XYZ should not be supported")
	}
}
