package domain

import "time"

// TaxDetails holds the result of a tax calculation attached to a payment.
type TaxDetails struct {
	IBSAmountMinor  int64          `json:"ibs_amount"`
	CBSAmountMinor  int64          `json:"cbs_amount"`
	TotalTaxMinor   int64          `json:"total_tax"`
	BaseAmountMinor int64          `json:"base_amount,omitempty"`
	Currency        string         `json:"currency"`
	RuleVersion     string         `json:"rule_version"`
	Jurisdiction    string         `json:"jurisdiction,omitempty"`
	CalculatedAt    time.Time      `json:"calculated_at"`
	Breakdown       []TaxBreakdown `json:"breakdown,omitempty"`
}

// TaxBreakdown provides granular detail on each tax component.
type TaxBreakdown struct {
	TaxType    string  `json:"tax_type"`    // "IBS" or "CBS"
	BaseAmount int64   `json:"base_amount"` // Base in minor units
	Rate       float64 `json:"rate"`        // e.g., 0.05 for 5%
	TaxAmount  int64   `json:"tax_amount"`  // Calculated tax in minor units
	LegalBasis string  `json:"legal_basis,omitempty"`
}

// TaxCalculationInput holds all context needed to compute taxes for a transaction.
type TaxCalculationInput struct {
	AmountMinor      int64     // Transaction amount in minor units
	Currency         string    // ISO 4217
	CountryCode      string    // ISO 3166-1 alpha-2 (e.g., "BR")
	StateCode        string    // e.g., "SP", "RJ"
	CityCode         string    // Municipal code when applicable
	ProductType      string    // e.g., "physical_goods", "digital_service"
	CustomerType     string    // "individual", "company", "government", "other"
	TaxRegime        string    // Specific tax regime if applicable
	EffectiveAt      time.Time // Determines which rule version applies
	RequestedVersion string    // Client-requested rule version (server may reject)
}

// TaxDestination holds geographic information for tax determination.
type TaxDestination struct {
	CountryCode string `json:"country_code"`
	StateCode   string `json:"state_code"`
	CityCode    string `json:"city_code,omitempty"`
	PostalCode  string `json:"postal_code,omitempty"`
}

// TaxContext holds business context that may affect tax calculation.
type TaxContext struct {
	ProductType    string `json:"product_type,omitempty"`
	CustomerType   string `json:"customer_type,omitempty"`
	TaxRegime      string `json:"tax_regime,omitempty"`
	TaxRuleVersion string `json:"tax_rule_version,omitempty"`
}
