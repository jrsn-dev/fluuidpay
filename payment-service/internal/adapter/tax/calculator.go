package tax

import (
	"context"
	"fmt"
	"math"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/fluuid/payment-service/internal/domain"
)

// Calculator implements domain.TaxCalculator using config-driven IBS/CBS rules.
type Calculator struct {
	rules   TaxRulesConfig
	version string
}

// TaxRulesConfig holds the tax rules loaded from YAML.
type TaxRulesConfig struct {
	Version       string     `yaml:"version"`
	EffectiveFrom string     `yaml:"effective_from"`
	EffectiveTo   string     `yaml:"effective_to"`
	Rules         []TaxRule  `yaml:"rules"`
	Default       TaxDefault `yaml:"default"`
}

// TaxRule defines IBS/CBS rates for a specific state/product combination.
type TaxRule struct {
	StateCode   string  `yaml:"state_code"`
	ProductType string  `yaml:"product_type"`
	IBSRate     float64 `yaml:"ibs_rate"`
	CBSRate     float64 `yaml:"cbs_rate"`
	LegalBasis  string  `yaml:"legal_basis,omitempty"`
}

// TaxDefault defines fallback rates when no specific rule matches.
type TaxDefault struct {
	IBSRate float64 `yaml:"ibs_rate"`
	CBSRate float64 `yaml:"cbs_rate"`
}

// NewCalculator creates a tax calculator from a YAML config file.
func NewCalculator(rulesPath string) (*Calculator, error) {
	data, err := os.ReadFile(rulesPath)
	if err != nil {
		return nil, fmt.Errorf("read tax rules: %w", err)
	}

	var config TaxRulesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse tax rules: %w", err)
	}

	if config.Version == "" {
		return nil, fmt.Errorf("tax rules version is required")
	}

	return &Calculator{
		rules:   config,
		version: config.Version,
	}, nil
}

// Calculate computes IBS/CBS taxes based on the transaction context.
func (c *Calculator) Calculate(ctx context.Context, input domain.TaxCalculationInput) (*domain.TaxDetails, error) {
	// Find matching rule
	ibsRate, cbsRate, legalBasis := c.findRates(input.StateCode, input.ProductType)

	// Calculate tax amounts (rounding to nearest integer — minor unit)
	ibsAmount := int64(math.Round(float64(input.AmountMinor) * ibsRate))
	cbsAmount := int64(math.Round(float64(input.AmountMinor) * cbsRate))
	totalTax := ibsAmount + cbsAmount

	// Determine jurisdiction
	jurisdiction := input.CountryCode
	if input.StateCode != "" {
		jurisdiction = input.CountryCode + "-" + input.StateCode
	}

	version := c.version
	if input.RequestedVersion != "" {
		if input.RequestedVersion != c.version {
			return nil, fmt.Errorf("%w: requested %s, available %s",
				domain.ErrTaxRuleVersionUnsupported, input.RequestedVersion, c.version)
		}
		version = input.RequestedVersion
	}

	details := &domain.TaxDetails{
		IBSAmountMinor:  ibsAmount,
		CBSAmountMinor:  cbsAmount,
		TotalTaxMinor:   totalTax,
		BaseAmountMinor: input.AmountMinor,
		Currency:        input.Currency,
		RuleVersion:     version,
		Jurisdiction:    jurisdiction,
		CalculatedAt:    time.Now().UTC(),
		Breakdown: []domain.TaxBreakdown{
			{
				TaxType:    "IBS",
				BaseAmount: input.AmountMinor,
				Rate:       ibsRate,
				TaxAmount:  ibsAmount,
				LegalBasis: legalBasis,
			},
			{
				TaxType:    "CBS",
				BaseAmount: input.AmountMinor,
				Rate:       cbsRate,
				TaxAmount:  cbsAmount,
				LegalBasis: legalBasis,
			},
		},
	}

	return details, nil
}

// findRates looks up the IBS/CBS rates for the given state and product type.
// Falls back to default rates if no specific rule matches.
func (c *Calculator) findRates(stateCode, productType string) (ibsRate, cbsRate float64, legalBasis string) {
	// Try exact match: state + product
	for _, rule := range c.rules.Rules {
		if rule.StateCode == stateCode && rule.ProductType == productType {
			return rule.IBSRate, rule.CBSRate, rule.LegalBasis
		}
	}

	// Try state-only match
	for _, rule := range c.rules.Rules {
		if rule.StateCode == stateCode && rule.ProductType == "" {
			return rule.IBSRate, rule.CBSRate, rule.LegalBasis
		}
	}

	// Use defaults
	return c.rules.Default.IBSRate, c.rules.Default.CBSRate, ""
}
