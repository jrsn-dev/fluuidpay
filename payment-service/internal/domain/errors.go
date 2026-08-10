package domain

import "errors"

// Domain errors — payment processing.
var (
	// ErrInvalidTransition indicates an illegal state transition was attempted.
	ErrInvalidTransition = errors.New("invalid payment state transition")

	// ErrPaymentNotFound indicates the requested payment does not exist.
	ErrPaymentNotFound = errors.New("payment not found")

	// ErrIdempotencyConflict indicates the same idempotency key was used with a different payload.
	ErrIdempotencyConflict = errors.New("idempotency key conflict: different payload")

	// ErrIdempotencyKeyMissing indicates the required Idempotency-Key header is absent.
	ErrIdempotencyKeyMissing = errors.New("idempotency key is required")

	// ErrOperationInProgress indicates the idempotency key is currently being processed.
	ErrOperationInProgress = errors.New("operation already in progress for this idempotency key")
)

// Domain errors — monetary values.
var (
	// ErrInvalidAmount indicates the payment amount is invalid (zero or negative).
	ErrInvalidAmount = errors.New("payment amount must be positive")

	// ErrNegativeAmount indicates a negative monetary value was provided.
	ErrNegativeAmount = errors.New("monetary amount cannot be negative")

	// ErrInvalidCurrency indicates an unsupported or malformed currency code.
	ErrInvalidCurrency = errors.New("invalid currency code")

	// ErrCurrencyMismatch indicates an operation between different currencies.
	ErrCurrencyMismatch = errors.New("currency mismatch")

	// ErrAmountOverflow indicates the resulting amount exceeds int64 capacity.
	ErrAmountOverflow = errors.New("amount overflow")

	// ErrInsufficientAmount indicates the subtraction would result in a negative value.
	ErrInsufficientAmount = errors.New("insufficient amount")
)

// Domain errors — validation.
var (
	// ErrInvalidCardToken indicates the card token format is invalid.
	ErrInvalidCardToken = errors.New("invalid card token")

	// ErrInvalidUserID indicates the user ID is missing or malformed.
	ErrInvalidUserID = errors.New("invalid user ID")

	// ErrInvalidOrderID indicates the order ID is missing or malformed.
	ErrInvalidOrderID = errors.New("invalid order ID")

	// ErrInvalidStateCode indicates the state/destination code is invalid.
	ErrInvalidStateCode = errors.New("invalid state code")
)

// Domain errors — gateway.
var (
	// ErrGatewayUnavailable indicates the payment gateway is temporarily unreachable.
	ErrGatewayUnavailable = errors.New("payment gateway unavailable")

	// ErrGatewayTimeout indicates the gateway did not respond within the configured timeout.
	ErrGatewayTimeout = errors.New("payment gateway timeout")

	// ErrGatewayRejected indicates the gateway rejected the charge.
	ErrGatewayRejected = errors.New("payment rejected by gateway")
)

// Domain errors — tax.
var (
	// ErrTaxCalculationFailed indicates the tax engine failed to compute taxes.
	ErrTaxCalculationFailed = errors.New("tax calculation failed")

	// ErrTaxRuleNotFound indicates no tax rule matches the given context.
	ErrTaxRuleNotFound = errors.New("tax rule not found for the given context")

	// ErrTaxRuleVersionUnsupported indicates the requested tax rule version is not available.
	ErrTaxRuleVersionUnsupported = errors.New("unsupported tax rule version")
)

// Domain errors — messaging.
var (
	// ErrEventPublishFailed indicates the event could not be published to the broker.
	ErrEventPublishFailed = errors.New("event publish failed")

	// ErrEventAlreadyProcessed indicates the event was already consumed and applied.
	ErrEventAlreadyProcessed = errors.New("event already processed")
)

// RetryableError wraps an error that should be retried.
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string { return e.Err.Error() }
func (e *RetryableError) Unwrap() error { return e.Err }

// PermanentError wraps an error that should not be retried.
type PermanentError struct {
	Err error
}

func (e *PermanentError) Error() string { return e.Err.Error() }
func (e *PermanentError) Unwrap() error { return e.Err }
