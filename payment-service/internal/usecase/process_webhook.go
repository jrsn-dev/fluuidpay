package usecase

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/fluuid/payment-service/internal/domain"
)

var (
	ErrInvalidWebhookSignature = errors.New("invalid webhook signature")
	ErrInvalidWebhookPayload   = errors.New("invalid webhook payload")
)

// WebhookPayload represents the data coming from the payment provider.
type WebhookPayload struct {
	ProviderTransactionID string    `json:"provider_transaction_id"`
	Status                string    `json:"status"` // e.g., "approved", "rejected", "refunded"
	ReasonCode            string    `json:"reason_code,omitempty"`
	Message               string    `json:"message,omitempty"`
	OccurredAt            time.Time `json:"occurred_at"`
}

// ProcessWebhookUseCase handles incoming webhooks from the payment provider.
type ProcessWebhookUseCase struct {
	paymentRepo domain.PaymentRepository
	outboxRepo  domain.OutboxRepository
	auditRepo   domain.AuditRepository
	txManager   domain.TxManager
	secretKey   string
	logger      *slog.Logger
}

// NewProcessWebhookUseCase creates a new ProcessWebhookUseCase.
func NewProcessWebhookUseCase(
	paymentRepo domain.PaymentRepository,
	outboxRepo domain.OutboxRepository,
	auditRepo domain.AuditRepository,
	txManager domain.TxManager,
	secretKey string,
	logger *slog.Logger,
) *ProcessWebhookUseCase {
	return &ProcessWebhookUseCase{
		paymentRepo: paymentRepo,
		outboxRepo:  outboxRepo,
		auditRepo:   auditRepo,
		txManager:   txManager,
		secretKey:   secretKey,
		logger:      logger,
	}
}

// Execute validates and processes the webhook payload.
func (uc *ProcessWebhookUseCase) Execute(ctx context.Context, payloadBytes []byte, signature, transactionID string) error {
	log := uc.logger.With(
		"transaction_id", transactionID,
	)

	// 1. Validate signature
	if !uc.validateSignature(payloadBytes, signature) {
		log.Warn("invalid webhook signature")
		return ErrInvalidWebhookSignature
	}

	// 2. Parse payload
	var payload WebhookPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		log.Warn("invalid webhook payload", "error", err)
		return ErrInvalidWebhookPayload
	}

	// 3. Retrieve payment
	payment, err := uc.paymentRepo.GetByID(ctx, transactionID)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			log.Warn("webhook received for unknown payment")
			return nil // Ack to provider, we can't do anything
		}
		return err
	}

	// 4. Determine state transition based on provider status
	var transitionErr error
	var eventType domain.EventType
	switch payload.Status {
	case "approved":
		// Only transition if it's currently PENDING or AUTHORIZED
		if payment.Status == domain.PaymentPending || payment.Status == domain.PaymentAuthorized {
			transitionErr = payment.Approve(payload.ProviderTransactionID)
			eventType = domain.EventPaymentApproved
		} else if payment.Status == domain.PaymentApproved || payment.Status == domain.PaymentCaptured {
			// Already handled, just ack
			return nil
		}
	case "rejected":
		if payment.Status == domain.PaymentPending {
			transitionErr = payment.Reject(domain.PaymentFailure{
				Code:     payload.ReasonCode,
				Category: domain.FailureCategoryGateway,
				Message:  payload.Message,
			})
			eventType = domain.EventPaymentRejected
		} else if payment.Status == domain.PaymentRejected {
			return nil
		}
	default:
		log.Info("unhandled webhook status", "status", payload.Status)
		return nil
	}

	if transitionErr != nil {
		log.Warn("invalid state transition from webhook", "error", transitionErr, "status", payment.Status)
		return nil // Ack to avoid retries for invalid transitions
	}

	// 5. Create outbox event
	correlationID := uuid.NewString()
	event := domain.NewPaymentEvent(payment, eventType, correlationID)
	eventPayload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	outboxEntry := &domain.OutboxEntry{
		ID:            uuid.NewString(),
		AggregateID:   payment.ID,
		EventType:     eventType,
		Payload:       eventPayload,
		CorrelationID: correlationID,
		Status:        domain.OutboxPending,
		CreatedAt:     time.Now().UTC(),
	}

	auditLog := &domain.AuditLog{
		ID:         uuid.NewString(),
		EntityID:   payment.ID,
		EntityType: "payment",
		EventType:  "WEBHOOK_UPDATE",
		Actor:      "system-webhook",
		NewState:   fmt.Sprintf(`{"status":"%s"}`, payment.Status),
		Reason:     "Status updated via webhook: " + payload.Status,
		CreatedAt:  time.Now().UTC(),
	}

	// 6. Persist atomically
	err = uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if err := uc.paymentRepo.UpdateStatus(txCtx, payment); err != nil {
			return fmt.Errorf("update payment status: %w", err)
		}
		if err := uc.outboxRepo.Insert(txCtx, outboxEntry); err != nil {
			return fmt.Errorf("insert outbox entry: %w", err)
		}
		if err := uc.auditRepo.Insert(txCtx, auditLog); err != nil {
			return fmt.Errorf("insert audit: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	log.Info("webhook processed successfully", "new_status", payment.Status)
	return nil
}

func (uc *ProcessWebhookUseCase) validateSignature(payload []byte, signature string) bool {
	if uc.secretKey == "" {
		return true // Allow bypass if no secret configured (for local dev)
	}

	mac := hmac.New(sha256.New, []byte(uc.secretKey))
	mac.Write(payload)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedMAC))
}
