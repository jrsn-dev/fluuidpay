//go:build chaos
// +build chaos

package tax_test

import (
	"context"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"github.com/fluuid/payment-service/internal/adapter/tax"
	"github.com/fluuid/payment-service/internal/domain"
)

func TestCalculator_Properties(t *testing.T) {
	tmpPath := "test_property_rules.yaml"

	calc, err := tax.NewCalculator(tmpPath)
	if err != nil {
		t.Skip("Skipping property test, needs rules.yaml setup")
	}

	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 1000 // Test 1000 random values

	properties := gopter.NewProperties(parameters)

	properties.Property("TotalTaxMinor should always equal IBS + CBS", prop.ForAll(
		func(amountMinor int64) bool {
			// Skip negative amounts as domain doesn't allow them, but let's test absolute values
			if amountMinor < 0 {
				amountMinor = -amountMinor
			}

			input := domain.TaxCalculationInput{
				AmountMinor: amountMinor,
				Currency:    "BRL",
				CountryCode: "BR",
				StateCode:   "SP",
			}

			details, err := calc.Calculate(context.Background(), input)
			if err != nil {
				return false
			}

			// Invariant: TotalTaxMinor == IBSAmountMinor + CBSAmountMinor
			return details.TotalTaxMinor == (details.IBSAmountMinor + details.CBSAmountMinor)
		},
		gen.Int64(),
	))

	properties.TestingRun(t)
}
