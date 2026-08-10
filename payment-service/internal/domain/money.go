package domain

import (
	"fmt"
	"strings"
)

// Money represents a monetary value in minor units (e.g., centavos for BRL).
// Using int64 avoids floating-point precision issues in financial calculations.
type Money struct {
	AmountMinor int64  // Value in the smallest currency unit (e.g., cents)
	Currency    string // ISO 4217 currency code (e.g., "BRL", "USD")
}

// NewMoney creates a new Money value object with validation.
func NewMoney(amountMinor int64, currency string) (Money, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		return Money{}, ErrInvalidCurrency
	}
	if len(currency) != 3 {
		return Money{}, ErrInvalidCurrency
	}
	if amountMinor < 0 {
		return Money{}, ErrNegativeAmount
	}
	return Money{
		AmountMinor: amountMinor,
		Currency:    currency,
	}, nil
}

// Add adds two Money values. Both must have the same currency.
func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("%w: cannot add %s to %s", ErrCurrencyMismatch, other.Currency, m.Currency)
	}
	result := m.AmountMinor + other.AmountMinor
	// Overflow check
	if (other.AmountMinor > 0 && result < m.AmountMinor) || (other.AmountMinor < 0 && result > m.AmountMinor) {
		return Money{}, ErrAmountOverflow
	}
	return Money{AmountMinor: result, Currency: m.Currency}, nil
}

// Subtract subtracts other from m. Both must have the same currency.
func (m Money) Subtract(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("%w: cannot subtract %s from %s", ErrCurrencyMismatch, other.Currency, m.Currency)
	}
	result := m.AmountMinor - other.AmountMinor
	if result < 0 {
		return Money{}, ErrInsufficientAmount
	}
	return Money{AmountMinor: result, Currency: m.Currency}, nil
}

// IsZero returns true if the amount is zero.
func (m Money) IsZero() bool {
	return m.AmountMinor == 0
}

// IsPositive returns true if the amount is greater than zero.
func (m Money) IsPositive() bool {
	return m.AmountMinor > 0
}

// Equals returns true if both amount and currency match.
func (m Money) Equals(other Money) bool {
	return m.AmountMinor == other.AmountMinor && m.Currency == other.Currency
}

// String returns a human-readable representation (e.g., "BRL 199.90").
func (m Money) String() string {
	major := m.AmountMinor / 100
	minor := m.AmountMinor % 100
	return fmt.Sprintf("%s %d.%02d", m.Currency, major, minor)
}

// SupportedCurrencies lists the currencies currently supported by the system.
var SupportedCurrencies = map[string]bool{
	"BRL": true,
	"USD": true,
}

// IsSupportedCurrency checks if the currency is supported.
func IsSupportedCurrency(code string) bool {
	return SupportedCurrencies[strings.ToUpper(code)]
}
