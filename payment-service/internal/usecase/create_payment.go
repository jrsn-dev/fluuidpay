package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/fluuid/payment-service/internal/domain"
)

// CreatePaymentInput holds the input for creating a payment.
type CreatePaymentInput struct {
	UserID         string                `validate:"required,uuid"`
	OrderID        string                `validate:"required,max=64"`
	AmountMinor    int64                 `validate:"required,gt=0"`
	Currency       string                `validate:"required,len=3"`
	CardToken      string                `validate:"required,min=10"`
	IdempotencyKey string                `validate:"required,max=128"`
	Capture        bool                  `validate:"-"`
	Destination    domain.TaxDestination `validate:"required"`
	TaxContext     domain.TaxContext     `validate:"required"`
	Metadata       map[string]string     `validate:"dive,keys,max=64,endkeys,max=256"`
	CorrelationID  string                `validate:"required"`
}

// CreatePaymentOutput holds the result of creating a payment.
type CreatePaymentOutput struct {
	TransactionID         string               `json:"transaction_id"`
	OrderID               string               `json:"order_id"`
	UserID                string               `json:"user_id"`
	Status                domain.PaymentStatus  `json:"status"`
	AmountMinor           int64                `json:"amount_minor"`
	Currency              string               `json:"currency"`
	ProviderTransactionID string               `json:"provider_transaction_id,omitempty"`
	Taxes                 *domain.TaxDetails   `json:"taxes,omitempty"`
	Failure               *domain.PaymentFailure `json:"failure,omitempty"`
	CreatedAt             time.Time            `json:"created_at"`
	UpdatedAt             time.Time            `json:"updated_at"`
	Idempotent            bool                 `json:"-"` // true if this is a cached response
	HTTPCode              int                  `json:"-"` // suggested HTTP status code
}

// CreatePaymentUseCase orchestrates the payment creation flow:
// validate → idempotency check → tax calculation → gateway charge → persist → outbox.
type CreatePaymentUseCase struct {
	paymentRepo domain.PaymentRepository
	idempotency domain.IdempotencyStore
	outboxRepo  domain.OutboxRepository
	auditRepo   domain.AuditRepository
	txManager   domain.TxManager
	gateway     domain.PaymentGateway
	taxCalc     domain.TaxCalculator
	logger      *slog.Logger
	idempotencyTTL time.Duration
}

// NewCreatePaymentUseCase creates the use case with all dependencies injected.
func NewCreatePaymentUseCase(
	paymentRepo domain.PaymentRepository,
	idempotency domain.IdempotencyStore,
	outboxRepo domain.OutboxRepository,
	auditRepo domain.AuditRepository,
	txManager domain.TxManager,
	gateway domain.PaymentGateway,
	taxCalc domain.TaxCalculator,
	logger *slog.Logger,
) *CreatePaymentUseCase {
	return &CreatePaymentUseCase{
		paymentRepo:    paymentRepo,
		idempotency:    idempotency,
		outboxRepo:     outboxRepo,
		auditRepo:      auditRepo,
		txManager:      txManager,
		gateway:        gateway,
		taxCalc:        taxCalc,
		logger:         logger,
		idempotencyTTL: 24 * time.Hour,
	}
}

// Execute runs the payment creation flow.
func (uc *CreatePaymentUseCase) Execute(ctx context.Context, input CreatePaymentInput) (*CreatePaymentOutput, error) {
	log := uc.logger.With(
		"correlation_id", input.CorrelationID,
		"order_id", input.OrderID,
		"idempotency_key", input.IdempotencyKey,
	)

	// Step 1: Validate input
	if err := uc.validate(input); err != nil {
		return nil, err
	}

	// Step 2: Compute canonical payload hash
	payloadHash := uc.computePayloadHash(input)

	// Step 3: Check idempotency
	existing, err := uc.idempotency.Get(ctx, input.IdempotencyKey)
	if err != nil {
		log.Error("idempotency lookup failed", "error", err)
		return nil, fmt.Errorf("idempotency check: %w", err)
	}

	if existing != nil {
		return uc.handleExistingIdempotency(existing, payloadHash)
	}

	// Step 4: Reserve idempotency key
	reserved, err := uc.idempotency.Reserve(ctx, input.IdempotencyKey, payloadHash, uc.idempotencyTTL)
	if err != nil {
		return nil, fmt.Errorf("reserve idempotency key: %w", err)
	}
	if !reserved {
		// Race condition — another request just reserved it
		return nil, domain.ErrOperationInProgress
	}

	log.Info("idempotency key reserved, processing payment")

	// Step 5: Calculate taxes
	taxInput := domain.TaxCalculationInput{
		AmountMinor:      input.AmountMinor,
		Currency:         input.Currency,
		CountryCode:      input.Destination.CountryCode,
		StateCode:        input.Destination.StateCode,
		CityCode:         input.Destination.CityCode,
		ProductType:      input.TaxContext.ProductType,
		CustomerType:     input.TaxContext.CustomerType,
		TaxRegime:        input.TaxContext.TaxRegime,
		EffectiveAt:      time.Now().UTC(),
		RequestedVersion: input.TaxContext.TaxRuleVersion,
	}

	taxes, err := uc.taxCalc.Calculate(ctx, taxInput)
	if err != nil {
		log.Error("tax calculation failed", "error", err)
		return nil, fmt.Errorf("calculate taxes: %w", err)
	}

	// Step 6: Create payment entity
	cardTokenHash := hashString(input.CardToken)
	payment, err := domain.NewPayment(
		input.UserID, input.OrderID,
		domain.Money{AmountMinor: input.AmountMinor, Currency: input.Currency},
		cardTokenHash, input.Destination.StateCode, input.Metadata,
	)
	if err != nil {
		return nil, err
	}
	payment.SetTaxes(*taxes)

	// Step 7: Transition to PENDING and charge gateway
	if err := payment.MarkPending(); err != nil {
		return nil, err
	}

	gatewayReq := domain.GatewayChargeRequest{
		IdempotencyKey: input.IdempotencyKey,
		CardToken:      input.CardToken, // Token only, never PAN/CVV
		AmountMinor:    input.AmountMinor,
		Currency:       input.Currency,
		OrderID:        input.OrderID,
		Capture:        input.Capture,
	}

	gatewayResp, err := uc.gateway.ProcessCharge(ctx, gatewayReq)

	// Step 8: Handle gateway result
	if err != nil {
		return uc.handleGatewayError(ctx, payment, taxes, input, err, log)
	}

	// Apply gateway response to payment
	switch gatewayResp.Status {
	case domain.PaymentApproved:
		if err := payment.Approve(gatewayResp.ProviderTransactionID); err != nil {
			return nil, err
		}
	case domain.PaymentAuthorized:
		if err := payment.Authorize(gatewayResp.ProviderTransactionID); err != nil {
			return nil, err
		}
	case domain.PaymentRejected:
		if err := payment.Reject(domain.PaymentFailure{
			Code:     gatewayResp.ProviderCode,
			Category: domain.FailureCategoryGateway,
			Message:  gatewayResp.Message,
		}); err != nil {
			return nil, err
		}
	case domain.PaymentPending:
		// Stay in PENDING — gateway hasn't confirmed yet
	}

	// Step 9: Persist atomically (payment + tax + outbox)
	eventType := domain.EventTypeFromStatus(payment.Status)
	event := domain.NewPaymentEvent(payment, eventType, input.CorrelationID)
	eventPayload, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal event: %w", err)
	}

	outboxEntry := &domain.OutboxEntry{
		ID:            uuid.NewString(),
		AggregateID:   payment.ID,
		EventType:     eventType,
		Payload:       eventPayload,
		CorrelationID: input.CorrelationID,
		Status:        domain.OutboxPending,
		CreatedAt:     time.Now().UTC(),
	}

	auditLog := &domain.AuditLog{
		ID:         uuid.NewString(),
		EntityID:   payment.ID,
		EntityType: "payment",
		EventType:  "CREATE",
		Actor:      input.UserID, // Typically from JWT, mapped to input
		NewState:   fmt.Sprintf(`{"status":"%s"}`, payment.Status),
		Reason:     "Initial payment creation",
		CreatedAt:  time.Now().UTC(),
	}

	err = uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if err := uc.paymentRepo.Create(txCtx, payment); err != nil {
			return fmt.Errorf("persist payment: %w", err)
		}
		if err := uc.outboxRepo.Insert(txCtx, outboxEntry); err != nil {
			return fmt.Errorf("insert outbox: %w", err)
		}
		if err := uc.auditRepo.Insert(txCtx, auditLog); err != nil {
			return fmt.Errorf("insert audit: %w", err)
		}
		return nil
	})
	if err != nil {
		log.Error("transaction failed", "error", err)
		return nil, err
	}

	// Step 10: Complete idempotency key
	output := uc.buildOutput(payment, taxes)
	responseData, _ := json.Marshal(output)
	if err := uc.idempotency.Complete(ctx, input.IdempotencyKey, responseData, output.HTTPCode, uc.idempotencyTTL); err != nil {
		log.Warn("failed to complete idempotency key", "error", err)
		// Non-fatal — payment is already persisted
	}

	log.Info("payment processed",
		"transaction_id", payment.ID,
		"status", payment.Status,
	)

	return output, nil
}

// validate checks all required input fields.
func (uc *CreatePaymentUseCase) validate(input CreatePaymentInput) error {
	if input.IdempotencyKey == "" {
		return domain.ErrIdempotencyKeyMissing
	}
	
	v := validator.New()
	if err := v.Struct(input); err != nil {
		return err // Handled by HTTP layer
	}

	if !domain.IsSupportedCurrency(input.Currency) {
		return domain.ErrInvalidCurrency
	}
	return nil
}

// computePayloadHash creates a canonical hash of the input for idempotency conflict detection.
func (uc *CreatePaymentUseCase) computePayloadHash(input CreatePaymentInput) string {
	// Canonical form: sorted fields excluding idempotency key itself
	canonical := map[string]any{
		"user_id":      input.UserID,
		"order_id":     input.OrderID,
		"amount_minor": input.AmountMinor,
		"currency":     input.Currency,
		"card_token":   input.CardToken,
		"state_code":   input.Destination.StateCode,
	}

	// Sort keys for deterministic serialization
	keys := make([]string, 0, len(canonical))
	for k := range canonical {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	h := sha256.New()
	for _, k := range keys {
		fmt.Fprintf(h, "%s=%v;", k, canonical[k])
	}
	return hex.EncodeToString(h.Sum(nil))
}

// handleExistingIdempotency processes a repeated request.
func (uc *CreatePaymentUseCase) handleExistingIdempotency(record *domain.IdempotencyRecord, payloadHash string) (*CreatePaymentOutput, error) {
	if record.Status == domain.IdempotencyProcessing {
		return nil, domain.ErrOperationInProgress
	}

	// Check payload hash matches
	if record.PayloadHash != "" && record.PayloadHash != payloadHash {
		return nil, domain.ErrIdempotencyConflict
	}

	// Return cached response
	if record.Response != nil {
		var output CreatePaymentOutput
		if err := json.Unmarshal(record.Response, &output); err == nil {
			output.Idempotent = true
			if record.HTTPCode != 0 {
				output.HTTPCode = record.HTTPCode
			} else {
				output.HTTPCode = 200 // Fallback
			}
			return &output, nil
		}
	}

	return nil, domain.ErrOperationInProgress
}

// handleGatewayError handles errors from the payment gateway.
func (uc *CreatePaymentUseCase) handleGatewayError(
	ctx context.Context,
	payment *domain.Payment,
	taxes *domain.TaxDetails,
	input CreatePaymentInput,
	gatewayErr error,
	log *slog.Logger,
) (*CreatePaymentOutput, error) {
	log.Error("gateway charge failed", "error", gatewayErr)

	// Persist payment in PENDING state for reconciliation
	err := uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		return uc.paymentRepo.Create(txCtx, payment)
	})
	if err != nil {
		log.Error("failed to persist pending payment", "error", err)
	}

	return nil, fmt.Errorf("gateway error: %w", gatewayErr)
}

// buildOutput creates the response from the payment entity.
func (uc *CreatePaymentUseCase) buildOutput(payment *domain.Payment, taxes *domain.TaxDetails) *CreatePaymentOutput {
	httpCode := 201
	switch payment.Status {
	case domain.PaymentPending:
		httpCode = 202
	case domain.PaymentRejected:
		httpCode = 402
	}

	return &CreatePaymentOutput{
		TransactionID:         payment.ID,
		OrderID:               payment.OrderID,
		UserID:                payment.UserID,
		Status:                payment.Status,
		AmountMinor:           payment.Amount.AmountMinor,
		Currency:              payment.Amount.Currency,
		ProviderTransactionID: payment.ProviderTransactionID,
		Taxes:                 taxes,
		Failure:               payment.Failure,
		CreatedAt:             payment.CreatedAt,
		UpdatedAt:             payment.UpdatedAt,
		HTTPCode:              httpCode,
	}
}

// hashString creates a SHA-256 hash of a string (used for card token hashing).
func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
