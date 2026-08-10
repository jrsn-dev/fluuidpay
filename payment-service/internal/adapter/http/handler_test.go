package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httphandler "github.com/fluuid/payment-service/internal/adapter/http"
	"github.com/fluuid/payment-service/internal/domain"
	"github.com/fluuid/payment-service/internal/usecase"
)

// --- Mocks ---
type mockPaymentRepo struct {
	payment *domain.Payment
	err     error
}
func (m *mockPaymentRepo) Create(ctx context.Context, p *domain.Payment) error { return m.err }
func (m *mockPaymentRepo) GetByID(ctx context.Context, id string) (*domain.Payment, error) { return m.payment, m.err }
func (m *mockPaymentRepo) GetByOrderID(ctx context.Context, id string) ([]*domain.Payment, error) { return nil, nil }
func (m *mockPaymentRepo) UpdateStatus(ctx context.Context, p *domain.Payment) error { return m.err }
func (m *mockPaymentRepo) ListPendingForReconciliation(ctx context.Context, d time.Duration) ([]*domain.Payment, error) { return nil, nil }

type mockIdempotency struct{}
func (m *mockIdempotency) Get(ctx context.Context, key string) (*domain.IdempotencyRecord, error) { return nil, nil }
func (m *mockIdempotency) Reserve(ctx context.Context, key, hash string, ttl time.Duration) (bool, error) { return true, nil }
func (m *mockIdempotency) Complete(ctx context.Context, key string, response []byte, code int, ttl time.Duration) error { return nil }

type mockOutbox struct{}
func (m *mockOutbox) Insert(ctx context.Context, e *domain.OutboxEntry) error { return nil }
func (m *mockOutbox) FetchPending(ctx context.Context, limit int) ([]*domain.OutboxEntry, error) { return nil, nil }
func (m *mockOutbox) MarkPublished(ctx context.Context, id string) error { return nil }
func (m *mockOutbox) MarkFailed(ctx context.Context, id string, err error, retry bool) error { return nil }

type mockTxManager struct{}
func (m *mockTxManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }

type mockGateway struct{}
func (m *mockGateway) ProcessCharge(ctx context.Context, req domain.GatewayChargeRequest) (*domain.GatewayChargeResponse, error) {
	return &domain.GatewayChargeResponse{Status: domain.PaymentApproved, ProviderTransactionID: "g-123"}, nil
}
func (m *mockGateway) QueryCharge(ctx context.Context, id string) (*domain.GatewayChargeResponse, error) { return nil, nil }
func (m *mockGateway) VoidCharge(ctx context.Context, id string) error { return nil }
func (m *mockGateway) RefundCharge(ctx context.Context, id string, amount int64) error { return nil }

type mockTaxCalc struct{}
func (m *mockTaxCalc) Calculate(ctx context.Context, in domain.TaxCalculationInput) (*domain.TaxDetails, error) {
	return &domain.TaxDetails{TotalTaxMinor: 100}, nil
}

type mockAuditRepo struct{}

func (m *mockAuditRepo) Insert(ctx context.Context, log *domain.AuditLog) error {
	return nil
}

// setupTestHandler initializes a router with mock dependencies.
func setupTestHandler(repo domain.PaymentRepository) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	idem := &mockIdempotency{}
	outbox := &mockOutbox{}
	audit := &mockAuditRepo{}
	txm := &mockTxManager{}
	gw := &mockGateway{}
	tax := &mockTaxCalc{}

	createUC := usecase.NewCreatePaymentUseCase(repo, idem, outbox, audit, txm, gw, tax, logger)
	getUC := usecase.NewGetPaymentUseCase(repo)
	cancelUC := usecase.NewCancelPaymentUseCase(repo, outbox, txm, gw, logger)
	webhookUC := usecase.NewProcessWebhookUseCase(repo, outbox, audit, txm, "secret", logger)

	handler, _ := httphandler.NewHandler(createUC, getUC, cancelUC, webhookUC, logger, "")
	return handler
}

// --- Tests ---

func TestCreatePaymentHandler_Success(t *testing.T) {
	repo := &mockPaymentRepo{}
	handler := setupTestHandler(repo)

	reqBody := `{"user_id":"123e4567-e89b-12d3-a456-426614174000","order_id":"o1","amount_minor":1000,"currency":"BRL","card_token":"tok_123456789","destination":{"country_code":"BR","state_code":"SP"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/payments", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem-123")
	req.Header.Set("Authorization", "Bearer valid_token")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201 Created, got %d", rec.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)

	if resp["status"] != "APPROVED" {
		t.Errorf("expected status APPROVED, got %v", resp["status"])
	}
	if resp["transaction_id"] == "" {
		t.Error("expected non-empty transaction_id")
	}
}

func TestCreatePaymentHandler_MissingIdempotencyKey(t *testing.T) {
	repo := &mockPaymentRepo{}
	handler := setupTestHandler(repo)

	reqBody := `{"user_id":"u1","order_id":"o1","amount_minor":1000,"currency":"BRL","card_token":"tok_123456789"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/payments", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid_token")
	// Missing Idempotency-Key

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", rec.Code)
	}
}

func TestGetPaymentHandler_Success(t *testing.T) {
	payment, _ := domain.NewPayment("u1", "o1", domain.Money{AmountMinor: 100, Currency: "BRL"}, "hash", "SP", nil)
	repo := &mockPaymentRepo{payment: payment}
	handler := setupTestHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/v1/payments/"+payment.ID, nil)
	req.Header.Set("Authorization", "Bearer valid_token")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rec.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["transaction_id"] != payment.ID {
		t.Errorf("expected transaction_id %s, got %v", payment.ID, resp["transaction_id"])
	}
}

func TestGetPaymentHandler_NotFound(t *testing.T) {
	repo := &mockPaymentRepo{err: domain.ErrPaymentNotFound}
	handler := setupTestHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/v1/payments/invalid-id", nil)
	req.Header.Set("Authorization", "Bearer valid_token")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found, got %d", rec.Code)
	}
}

func TestCancelPaymentHandler_Success(t *testing.T) {
	payment, _ := domain.NewPayment("u1", "o1", domain.Money{AmountMinor: 100, Currency: "BRL"}, "hash", "SP", nil)
	_ = payment.MarkPending()
	_ = payment.Approve("prov")
	repo := &mockPaymentRepo{payment: payment}
	handler := setupTestHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/v1/payments/"+payment.ID+"/cancel", nil)
	req.Header.Set("Authorization", "Bearer valid_token")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rec.Code)
	}
}

func TestWebhookHandler_InvalidSignature(t *testing.T) {
	repo := &mockPaymentRepo{}
	handler := setupTestHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/payment-gateway", bytes.NewBufferString(`{}`))
	req.Header.Set("X-Gateway-Signature", "wrong")
	// Webhooks typically don't have authorization headers, but let's see. In our router, webhooks are under /v1 but outside the mockAuthMiddleware. Let's check router setup.
	
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for invalid webhook signature, got %d", rec.Code)
	}
}
