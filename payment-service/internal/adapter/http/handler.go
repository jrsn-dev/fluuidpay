package http

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/fluuid/payment-service/internal/domain"
	"github.com/fluuid/payment-service/internal/platform"
	"github.com/fluuid/payment-service/internal/usecase"
)

// Handler holds HTTP handlers and their dependencies.
type Handler struct {
	createPayment  *usecase.CreatePaymentUseCase
	getPayment     *usecase.GetPaymentUseCase
	cancelPayment  *usecase.CancelPaymentUseCase
	processWebhook *usecase.ProcessWebhookUseCase
	logger         *slog.Logger
}
func (h *Handler) docsHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("docsHandler accessed")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8">
    <title>Swagger UI</title>
    <link rel="stylesheet" type="text/css" href="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.10.3/swagger-ui.min.css" >
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.10.3/swagger-ui-bundle.js"> </script>
    <script>
      window.onload = function() {
        SwaggerUIBundle({
          url: "/openapi.yaml",
          dom_id: '#swagger-ui',
          presets: [
            SwaggerUIBundle.presets.apis,
            SwaggerUIBundle.SwaggerUIStandalonePreset
          ],
        })
      }
    </script>
  </body>
</html>`))
}

// NewHandler creates a new HTTP handler.
func NewHandler(
	createPayment *usecase.CreatePaymentUseCase,
	getPayment *usecase.GetPaymentUseCase,
	cancelPayment *usecase.CancelPaymentUseCase,
	processWebhook *usecase.ProcessWebhookUseCase,
	logger *slog.Logger,
	jwksURL string,
) (http.Handler, error) {
	h := &Handler{
		createPayment:  createPayment,
		getPayment:     getPayment,
		cancelPayment:  cancelPayment,
		processWebhook: processWebhook,
		logger:         logger,
	}
	authMiddleware, err := RequireAuth(jwksURL)
	if err != nil {
		return nil, err
	}

	r := chi.NewRouter()

	// Middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(correlationIDMiddleware)
	r.Use(middleware.Recoverer)
	r.Use(requestLogger(logger))
	r.Use(platform.MetricsMiddleware)
	r.Use(middleware.Timeout(30 * time.Second))

	// Health and observability endpoints
	r.Get("/health", h.healthCheck)
	r.Get("/ready", h.readinessCheck)
	r.Handle("/metrics", platform.MetricsHandler())

	r.Get("/api/docs", h.docsHandler)
	r.Get("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "openapi-payment-ibs-cbs.yaml")
	})

	// Payment API v1
	r.Route("/v1", func(r chi.Router) {
		r.Route("/payments", func(r chi.Router) {
			r.Use(authMiddleware) // Secure only the payments endpoint
			r.Post("/", h.createPaymentHandler)
			r.Get("/{transactionId}", h.getPaymentHandler)
			r.Post("/{transactionId}/cancel", h.cancelPaymentHandler)
		})

		r.Post("/webhooks/payment-gateway", h.webhookHandler)
	})

	return r, nil
}

// createPaymentHandler handles POST /v1/payments.
func (h *Handler) createPaymentHandler(w http.ResponseWriter, r *http.Request) {
	// Extract Idempotency-Key header
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		writeError(w, http.StatusBadRequest, "idempotency_key_missing",
			"Idempotency-Key header is required", r)
		return
	}

	correlationID := r.Header.Get("X-Correlation-Id")
	if correlationID == "" {
		correlationID = uuid.NewString()
	}

	// Parse request body
	var req CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request",
			"Invalid request body: "+err.Error(), r)
		return
	}

	// Extract UserID from JWT token (in context)
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "User ID not found in token", r)
		return
	}

	// Override user ID from request body with the authenticated user's ID
	req.UserID = userID

	// Default capture to true if not specified
	capture := true
	if req.Capture != nil {
		capture = *req.Capture
	}

	// Build use case input
	input := usecase.CreatePaymentInput{
		UserID:         req.UserID,
		OrderID:        req.OrderID,
		AmountMinor:    req.AmountMinor,
		Currency:       req.Currency,
		CardToken:      req.CardToken,
		IdempotencyKey: idempotencyKey,
		Capture:        capture,
		Destination: domain.TaxDestination{
			CountryCode: req.Destination.CountryCode,
			StateCode:   req.Destination.StateCode,
			CityCode:    req.Destination.CityCode,
			PostalCode:  req.Destination.PostalCode,
		},
		Metadata:      req.Metadata,
		CorrelationID: correlationID,
	}
	if req.TaxContext != nil {
		input.TaxContext = domain.TaxContext{
			ProductType:    req.TaxContext.ProductType,
			CustomerType:   req.TaxContext.CustomerType,
			TaxRegime:      req.TaxContext.TaxRegime,
			TaxRuleVersion: req.TaxContext.TaxRuleVersion,
		}
	}

	// Execute
	output, err := h.createPayment.Execute(r.Context(), input)
	if err != nil {
		h.handleUseCaseError(w, r, err)
		return
	}

	// Set response headers
	if !output.Idempotent {
		w.Header().Set("Location", "/v1/payments/"+output.TransactionID)
	}
	w.Header().Set("X-Correlation-Id", correlationID)

	writeJSON(w, output.HTTPCode, output)
}

// getPaymentHandler handles GET /v1/payments/{transactionId}.
func (h *Handler) getPaymentHandler(w http.ResponseWriter, r *http.Request) {
	transactionID := chi.URLParam(r, "transactionId")
	if transactionID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request",
			"Transaction ID is required", r)
		return
	}

	output, err := h.getPayment.Execute(r.Context(), transactionID)
	if err != nil {
		h.handleUseCaseError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, output)
}

// cancelPaymentHandler handles POST /v1/payments/{transactionId}/cancel.
func (h *Handler) cancelPaymentHandler(w http.ResponseWriter, r *http.Request) {
	transactionID := chi.URLParam(r, "transactionId")
	idempotencyKey := r.Header.Get("Idempotency-Key")
	correlationID := r.Header.Get("X-Correlation-Id")

	var req CancelPaymentRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request",
				"Invalid request body", r)
			return
		}
	}

	input := usecase.CancelPaymentInput{
		TransactionID:  transactionID,
		IdempotencyKey: idempotencyKey,
		Reason:         req.Reason,
		AmountMinor:    req.AmountMinor,
		CorrelationID:  correlationID,
	}

	output, err := h.cancelPayment.Execute(r.Context(), input)
	if err != nil {
		h.handleUseCaseError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, output)
}

// webhookHandler handles POST /v1/webhooks/payment-gateway.
func (h *Handler) webhookHandler(w http.ResponseWriter, r *http.Request) {
	signature := r.Header.Get("X-Gateway-Signature")
	transactionID := r.Header.Get("X-Transaction-Id") // Some gateways send it in header, some in body

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Cannot read body", r)
		return
	}
	defer r.Body.Close()

	if err := h.processWebhook.Execute(r.Context(), body, signature, transactionID); err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidWebhookSignature):
			writeError(w, http.StatusUnauthorized, "invalid_signature", "Invalid signature", r)
		case errors.Is(err, usecase.ErrInvalidWebhookPayload):
			writeError(w, http.StatusBadRequest, "invalid_payload", "Invalid payload", r)
		default:
			h.logger.Error("webhook processing failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "Internal error", r)
		}
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// healthCheck handles GET /health.
func (h *Handler) healthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// readinessCheck handles GET /ready.
func (h *Handler) readinessCheck(w http.ResponseWriter, r *http.Request) {
	// TODO: Check DB, Redis, RabbitMQ connectivity
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ready",
	})
}

// handleUseCaseError maps domain errors to HTTP responses.
func (h *Handler) handleUseCaseError(w http.ResponseWriter, r *http.Request, err error) {
	var vErrs validator.ValidationErrors

	switch {
	case errors.Is(err, domain.ErrPaymentNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Payment not found", r)
	case errors.Is(err, domain.ErrIdempotencyConflict):
		writeError(w, http.StatusConflict, "idempotency_conflict",
			"Idempotency key used with different payload", r)
	case errors.Is(err, domain.ErrOperationInProgress):
		w.Header().Set("Retry-After", "5")
		writeError(w, http.StatusAccepted, "processing",
			"Operation is in progress", r)
	case errors.Is(err, domain.ErrIdempotencyKeyMissing):
		writeError(w, http.StatusBadRequest, "idempotency_key_missing",
			"Idempotency-Key header is required", r)
	case errors.Is(err, domain.ErrInvalidAmount),
		errors.Is(err, domain.ErrInvalidCurrency),
		errors.Is(err, domain.ErrInvalidCardToken),
		errors.Is(err, domain.ErrInvalidUserID),
		errors.Is(err, domain.ErrInvalidOrderID),
		errors.Is(err, domain.ErrInvalidStateCode):
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error(), r)
	case errors.As(err, &vErrs):
		writeError(w, http.StatusBadRequest, "invalid_request", "Validation failed: "+vErrs.Error(), r)
	case errors.Is(err, domain.ErrInvalidTransition):
		writeError(w, http.StatusConflict, "invalid_state",
			"Payment state does not allow this operation", r)
	case errors.Is(err, domain.ErrGatewayUnavailable),
		errors.Is(err, domain.ErrGatewayTimeout):
		w.Header().Set("Retry-After", "30")
		writeError(w, http.StatusServiceUnavailable, "gateway_unavailable",
			"Payment gateway is temporarily unavailable", r)
	case errors.Is(err, domain.ErrGatewayRejected):
		writeError(w, http.StatusPaymentRequired, "payment_rejected",
			"Payment was rejected by the gateway", r)
	case errors.Is(err, domain.ErrTaxCalculationFailed),
		errors.Is(err, domain.ErrTaxRuleNotFound):
		writeError(w, http.StatusUnprocessableEntity, "tax_error",
			"Tax calculation failed", r)
	default:
		h.logger.Error("unhandled error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error",
			"An internal error occurred", r)
	}
}

// --- Request/Response DTOs ---

// CreatePaymentRequest is the HTTP request body for creating a payment.
type CreatePaymentRequest struct {
	UserID      string            `json:"user_id"`
	OrderID     string            `json:"order_id"`
	AmountMinor int64             `json:"amount_minor"`
	Currency    string            `json:"currency"`
	CardToken   string            `json:"card_token"`
	Capture     *bool             `json:"capture,omitempty"`
	Destination DestinationDTO    `json:"destination"`
	TaxContext  *TaxContextDTO    `json:"tax_context,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// DestinationDTO holds the tax destination in the request.
type DestinationDTO struct {
	CountryCode string `json:"country_code"`
	StateCode   string `json:"state_code"`
	CityCode    string `json:"city_code,omitempty"`
	PostalCode  string `json:"postal_code,omitempty"`
}

// TaxContextDTO holds tax context in the request.
type TaxContextDTO struct {
	ProductType    string `json:"product_type,omitempty"`
	CustomerType   string `json:"customer_type,omitempty"`
	TaxRegime      string `json:"tax_regime,omitempty"`
	TaxRuleVersion string `json:"tax_rule_version,omitempty"`
}

// CancelPaymentRequest is the HTTP request body for cancelling a payment.
type CancelPaymentRequest struct {
	Reason      string `json:"reason,omitempty"`
	AmountMinor *int64 `json:"amount_minor,omitempty"`
}

// ErrorResponse is the standard error format matching the OpenAPI spec.
type ErrorResponse struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlation_id"`
}

// --- Helper functions ---

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string, r *http.Request) {
	correlationID := r.Header.Get("X-Correlation-Id")
	writeJSON(w, status, ErrorResponse{
		Code:          code,
		Message:       message,
		CorrelationID: correlationID,
	})
}

// --- Middlewares ---

func correlationIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Correlation-Id") == "" {
			r.Header.Set("X-Correlation-Id", uuid.NewString())
		}
		w.Header().Set("X-Correlation-Id", r.Header.Get("X-Correlation-Id"))
		next.ServeHTTP(w, r)
	})
}

func requestLogger(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap writer to capture status code
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration_ms", time.Since(start).Milliseconds(),
				"correlation_id", r.Header.Get("X-Correlation-Id"),
			)
		})
	}
}

// mockAuthMiddleware is a placeholder for actual authentication (e.g., JWT validation).
func mockAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Example: Require Authorization header for API paths
		if r.Header.Get("Authorization") == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Authorization header is required", r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
