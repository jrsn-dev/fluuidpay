package usecase_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/fluuid/payment-service/internal/domain"
	"github.com/fluuid/payment-service/internal/usecase"
)

// --- Mocks ---

type mockPaymentRepo struct {
	payments map[string]*domain.Payment
	created  []*domain.Payment
}

func (m *mockPaymentRepo) Create(ctx context.Context, p *domain.Payment) error {
	m.payments[p.ID] = p
	m.created = append(m.created, p)
	return nil
}
func (m *mockPaymentRepo) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	if p, ok := m.payments[id]; ok {
		return p, nil
	}
	return nil, domain.ErrPaymentNotFound
}
func (m *mockPaymentRepo) GetByOrderID(ctx context.Context, orderID string) ([]*domain.Payment, error) {
	return nil, nil
}
func (m *mockPaymentRepo) UpdateStatus(ctx context.Context, p *domain.Payment) error {
	m.payments[p.ID] = p
	return nil
}
func (m *mockPaymentRepo) ListPendingForReconciliation(ctx context.Context, d time.Duration) ([]*domain.Payment, error) {
	return nil, nil
}

type mockAuditRepo struct{}

func (m *mockAuditRepo) Insert(ctx context.Context, log *domain.AuditLog) error {
	return nil
}

type mockIdempotency struct {
	records map[string]*domain.IdempotencyRecord
}

func (m *mockIdempotency) Get(ctx context.Context, key string) (*domain.IdempotencyRecord, error) {
	if r, ok := m.records[key]; ok {
		return r, nil
	}
	return nil, nil
}
func (m *mockIdempotency) Reserve(ctx context.Context, key, hash string, ttl time.Duration) (bool, error) {
	if _, ok := m.records[key]; ok {
		return false, nil
	}
	m.records[key] = &domain.IdempotencyRecord{
		Key:         key,
		PayloadHash: hash,
		Status:      domain.IdempotencyProcessing,
	}
	return true, nil
}
func (m *mockIdempotency) Complete(ctx context.Context, key string, response []byte, code int, ttl time.Duration) error {
	if r, ok := m.records[key]; ok {
		r.Status = domain.IdempotencyCompleted
		r.Response = response
		r.HTTPCode = code
	}
	return nil
}

type mockOutboxRepo struct {
	entries []*domain.OutboxEntry
}

func (m *mockOutboxRepo) Insert(ctx context.Context, e *domain.OutboxEntry) error {
	m.entries = append(m.entries, e)
	return nil
}
func (m *mockOutboxRepo) FetchPending(ctx context.Context, limit int) ([]*domain.OutboxEntry, error) {
	return nil, nil
}
func (m *mockOutboxRepo) MarkPublished(ctx context.Context, id string) error { return nil }
func (m *mockOutboxRepo) MarkFailed(ctx context.Context, id string, err error, r bool) error {
	return nil
}

type mockTxManager struct{}

func (m *mockTxManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

type mockGateway struct {
	resp *domain.GatewayChargeResponse
	err  error
}

func (m *mockGateway) ProcessCharge(ctx context.Context, req domain.GatewayChargeRequest) (*domain.GatewayChargeResponse, error) {
	return m.resp, m.err
}
func (m *mockGateway) QueryCharge(ctx context.Context, id string) (*domain.GatewayChargeResponse, error) {
	return nil, nil
}
func (m *mockGateway) VoidCharge(ctx context.Context, id string) error { return nil }
func (m *mockGateway) RefundCharge(ctx context.Context, id string, amount int64) error { return nil }

type mockTaxCalc struct {
	details *domain.TaxDetails
	err     error
}

func (m *mockTaxCalc) Calculate(ctx context.Context, in domain.TaxCalculationInput) (*domain.TaxDetails, error) {
	return m.details, m.err
}

// --- Tests ---

func TestCreatePaymentUseCase_Success(t *testing.T) {
	repo := &mockPaymentRepo{payments: make(map[string]*domain.Payment)}
	idem := &mockIdempotency{records: make(map[string]*domain.IdempotencyRecord)}
	outb := &mockOutboxRepo{}
	txm := &mockTxManager{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	gw := &mockGateway{
		resp: &domain.GatewayChargeResponse{
			ProviderTransactionID: "provider-123",
			Status:                domain.PaymentApproved,
		},
	}
	tax := &mockTaxCalc{
		details: &domain.TaxDetails{TotalTaxMinor: 100},
	}

	uc := usecase.NewCreatePaymentUseCase(repo, idem, outb, &mockAuditRepo{}, txm, gw, tax, logger)

	input := usecase.CreatePaymentInput{
		UserID:         "123e4567-e89b-12d3-a456-426614174000",
		OrderID:        "order1",
		AmountMinor:    1000,
		Currency:       "BRL",
		CardToken:      "tok_123456789",
		IdempotencyKey: "idem-key-1",
		Capture:        true,
		Destination: domain.TaxDestination{
			CountryCode:   "BR",
			StateCode:     "SP",
		},
		CorrelationID: "corr-1",
	}

	out, err := uc.Execute(context.Background(), input)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if out == nil {
		t.Fatal("expected output, got nil")
	}

	if out.Status != domain.PaymentApproved {
		t.Errorf("expected status %s, got %s", domain.PaymentApproved, out.Status)
	}

	if len(repo.created) != 1 {
		t.Fatalf("expected 1 payment persisted, got %d", len(repo.created))
	}

	if repo.created[0].Status != domain.PaymentApproved {
		t.Errorf("expected persisted payment to be APPROVED, got %s", repo.created[0].Status)
	}

	if len(outb.entries) != 1 {
		t.Fatalf("expected 1 outbox entry, got %d", len(outb.entries))
	}

	if outb.entries[0].EventType != domain.EventPaymentApproved {
		t.Errorf("expected outbox event PaymentApproved, got %s", outb.entries[0].EventType)
	}

	// Idempotency check
	rec, _ := idem.Get(context.Background(), "idem-key-1")
	if rec == nil || rec.Status != domain.IdempotencyCompleted {
		t.Error("expected idempotency key to be completed")
	}
}

func TestCreatePaymentUseCase_IdempotencyConflict(t *testing.T) {
	repo := &mockPaymentRepo{payments: make(map[string]*domain.Payment)}
	idem := &mockIdempotency{records: make(map[string]*domain.IdempotencyRecord)}
	outb := &mockOutboxRepo{}
	txm := &mockTxManager{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	gw := &mockGateway{}
	tax := &mockTaxCalc{}

	// Pre-fill idempotency with DIFFERENT hash
	idem.records["idem-key-1"] = &domain.IdempotencyRecord{
		Key:         "idem-key-1",
		PayloadHash: "different-hash",
		Status:      domain.IdempotencyCompleted,
	}

	uc := usecase.NewCreatePaymentUseCase(repo, idem, outb, &mockAuditRepo{}, txm, gw, tax, logger)

	input := usecase.CreatePaymentInput{
		UserID:         "123e4567-e89b-12d3-a456-426614174000",
		OrderID:        "order1",
		AmountMinor:    1000,
		Currency:       "BRL",
		CardToken:      "tok_123456789",
		IdempotencyKey: "idem-key-1",
		Destination:    domain.TaxDestination{StateCode: "SP"},
		CorrelationID:  "corr-1",
	}

	_, err := uc.Execute(context.Background(), input)

	if !errors.Is(err, domain.ErrIdempotencyConflict) {
		t.Errorf("expected ErrIdempotencyConflict, got %v", err)
	}
}

func TestCreatePaymentUseCase_IdempotencyCached(t *testing.T) {
	repo := &mockPaymentRepo{payments: make(map[string]*domain.Payment)}
	idem := &mockIdempotency{records: make(map[string]*domain.IdempotencyRecord)}
	outb := &mockOutboxRepo{}
	txm := &mockTxManager{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	gw := &mockGateway{}
	tax := &mockTaxCalc{}

	uc := usecase.NewCreatePaymentUseCase(repo, idem, outb, &mockAuditRepo{}, txm, gw, tax, logger)

	input := usecase.CreatePaymentInput{
		UserID:         "123e4567-e89b-12d3-a456-426614174000",
		OrderID:        "order1",
		AmountMinor:    1000,
		Currency:       "BRL",
		CardToken:      "tok_123456789",
		IdempotencyKey: "idem-key-1",
		Destination:    domain.TaxDestination{StateCode: "SP"},
		CorrelationID:  "corr-2",
	}

	// Calculate correct hash for the mock by calling Execute and failing at Gateway (or pre-calculate)
	// We'll just let it fail at gateway to cache it? No, we need to mock it properly.
	// Actually, the easiest way to test cached response is to run it once, then run again.

	tax.details = &domain.TaxDetails{}
	gw.resp = &domain.GatewayChargeResponse{Status: domain.PaymentApproved}

	out1, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("first execution failed: %v", err)
	}
	if out1.Idempotent {
		t.Error("first execution should not be marked as idempotent")
	}

	// Run exact same request again
	out2, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("second execution failed: %v", err)
	}

	if !out2.Idempotent {
		t.Error("second execution should be marked as idempotent")
	}
	if out1.TransactionID != out2.TransactionID {
		t.Error("transaction IDs should match")
	}

	// Ensure we only persisted one payment and one outbox event
	if len(repo.created) != 1 {
		t.Errorf("expected 1 payment persisted, got %d", len(repo.created))
	}
	if len(outb.entries) != 1 {
		t.Errorf("expected 1 outbox entry, got %d", len(outb.entries))
	}
}

func TestCreatePaymentUseCase_GatewayRejection(t *testing.T) {
	repo := &mockPaymentRepo{payments: make(map[string]*domain.Payment)}
	idem := &mockIdempotency{records: make(map[string]*domain.IdempotencyRecord)}
	outb := &mockOutboxRepo{}
	txm := &mockTxManager{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	gw := &mockGateway{
		resp: &domain.GatewayChargeResponse{
			ProviderTransactionID: "provider-123",
			Status:                domain.PaymentRejected,
			ProviderCode:          "insufficient_funds",
		},
	}
	tax := &mockTaxCalc{details: &domain.TaxDetails{}}

	uc := usecase.NewCreatePaymentUseCase(repo, idem, outb, &mockAuditRepo{}, txm, gw, tax, logger)

	input := usecase.CreatePaymentInput{
		UserID:         "123e4567-e89b-12d3-a456-426614174000",
		OrderID:        "order1",
		AmountMinor:    1000,
		Currency:       "BRL",
		CardToken:      "tok_123456789",
		IdempotencyKey: "idem-key-1",
		Destination:    domain.TaxDestination{StateCode: "SP"},
		CorrelationID:  "corr-3",
	}

	out, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if out.Status != domain.PaymentRejected {
		t.Errorf("expected REJECTED, got %s", out.Status)
	}
	if out.HTTPCode != 402 {
		t.Errorf("expected HTTP 402, got %d", out.HTTPCode)
	}
	if out.Failure == nil || out.Failure.Code != "insufficient_funds" {
		t.Errorf("expected failure code insufficient_funds")
	}

	// Check outbox for rejection event
	if len(outb.entries) != 1 {
		t.Fatalf("expected 1 outbox entry, got %d", len(outb.entries))
	}
	if outb.entries[0].EventType != domain.EventPaymentRejected {
		t.Errorf("expected outbox event PaymentRejected, got %s", outb.entries[0].EventType)
	}

	// Ensure the cached idempotency response returns 402
	rec, _ := idem.Get(context.Background(), "idem-key-1")
	if rec.HTTPCode != 402 {
		t.Errorf("expected cached HTTP code 402, got %d", rec.HTTPCode)
	}
}

func TestCreatePaymentUseCase_GatewayTimeout(t *testing.T) {
	repo := &mockPaymentRepo{payments: make(map[string]*domain.Payment)}
	idem := &mockIdempotency{records: make(map[string]*domain.IdempotencyRecord)}
	outb := &mockOutboxRepo{}
	txm := &mockTxManager{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	gw := &mockGateway{
		err: domain.ErrGatewayTimeout,
	}
	tax := &mockTaxCalc{details: &domain.TaxDetails{}}

	uc := usecase.NewCreatePaymentUseCase(repo, idem, outb, &mockAuditRepo{}, txm, gw, tax, logger)

	input := usecase.CreatePaymentInput{
		UserID:         "123e4567-e89b-12d3-a456-426614174000",
		OrderID:        "order1",
		AmountMinor:    1000,
		Currency:       "BRL",
		CardToken:      "tok_123456789",
		IdempotencyKey: "idem-key-1",
		Destination:    domain.TaxDestination{StateCode: "SP"},
		CorrelationID:  "corr-4",
	}

	out, err := uc.Execute(context.Background(), input)

	if out != nil {
		t.Error("expected output to be nil on gateway error")
	}
	if !errors.Is(err, domain.ErrGatewayTimeout) {
		t.Errorf("expected ErrGatewayTimeout, got %v", err)
	}

	// Ensure payment is persisted as PENDING for reconciliation
	if len(repo.created) != 1 {
		t.Fatalf("expected 1 payment persisted as PENDING, got %d", len(repo.created))
	}
	if repo.created[0].Status != domain.PaymentPending {
		t.Errorf("expected status PENDING, got %s", repo.created[0].Status)
	}

	// No outbox event should be created yet since it's uncertain
	if len(outb.entries) != 0 {
		t.Errorf("expected 0 outbox entries, got %d", len(outb.entries))
	}
}
