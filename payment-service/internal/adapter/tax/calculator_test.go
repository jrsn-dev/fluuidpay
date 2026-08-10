package tax_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fluuid/payment-service/internal/adapter/tax"
	"github.com/fluuid/payment-service/internal/domain"
)

func TestTaxCalculator_Calculate_ValidRules(t *testing.T) {
	// Create a temporary YAML file with test rules
	yamlContent := []byte(`
version: "1.0"
rules:
  - id: "rule-1"
    country_code: "BR"
    state_code: "SP"
    ibs_rate: 0.10
    cbs_rate: 0.05
    description: "Standard rate for SP"
`)
	tmpDir := t.TempDir()
	rulesPath := filepath.Join(tmpDir, "rules.yaml")
	if err := os.WriteFile(rulesPath, yamlContent, 0644); err != nil {
		t.Fatalf("failed to write mock rules file: %v", err)
	}

	calc, err := tax.NewCalculator(rulesPath)
	if err != nil {
		t.Fatalf("failed to create calculator: %v", err)
	}

	input := domain.TaxCalculationInput{
		AmountMinor: 1000,
		CountryCode: "BR",
		StateCode:   "SP",
	}

	details, err := calc.Calculate(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if details.TotalTaxMinor != 150 { // 1000 * (0.10 + 0.05) = 150
		t.Errorf("expected 150 total tax, got %d", details.TotalTaxMinor)
	}
	if len(details.Breakdown) != 2 {
		t.Errorf("expected 2 breakdown items, got %d", len(details.Breakdown))
	}
}

func TestTaxCalculator_Calculate_RuleNotFound(t *testing.T) {
	yamlContent := []byte(`
version: "1.0"
rules:
  - id: "rule-1"
    country_code: "BR"
    state_code: "SP"
    ibs_rate: 0.10
    cbs_rate: 0.05
`)
	tmpDir := t.TempDir()
	rulesPath := filepath.Join(tmpDir, "rules.yaml")
	os.WriteFile(rulesPath, yamlContent, 0644)

	calc, err := tax.NewCalculator(rulesPath)
	if err != nil {
		t.Fatalf("failed to create calculator: %v", err)
	}

	input := domain.TaxCalculationInput{
		AmountMinor: 1000,
		CountryCode: "BR",
		StateCode:   "RJ", // RJ is not in the rules
	}

	details, err := calc.Calculate(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error (fallback to default), got %v", err)
	}

	if details.TotalTaxMinor != 0 {
		t.Errorf("expected 0 total tax using empty default, got %d", details.TotalTaxMinor)
	}
}
