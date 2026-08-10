package gateway

import (
	"context"

	"github.com/sony/gobreaker"

	"github.com/fluuid/payment-service/internal/domain"
)

// CircuitBreakerGateway wraps a domain.PaymentGateway with a circuit breaker.
type CircuitBreakerGateway struct {
	next domain.PaymentGateway
	cb   *gobreaker.CircuitBreaker
}

// NewCircuitBreakerGateway creates a new decorated gateway.
func NewCircuitBreakerGateway(next domain.PaymentGateway, st gobreaker.Settings) *CircuitBreakerGateway {
	return &CircuitBreakerGateway{
		next: next,
		cb:   gobreaker.NewCircuitBreaker(st),
	}
}

func (g *CircuitBreakerGateway) ProcessCharge(ctx context.Context, req domain.GatewayChargeRequest) (*domain.GatewayChargeResponse, error) {
	result, err := g.cb.Execute(func() (interface{}, error) {
		return g.next.ProcessCharge(ctx, req)
	})
	if err != nil {
		if err == gobreaker.ErrOpenState || err == gobreaker.ErrTooManyRequests {
			return nil, domain.ErrGatewayUnavailable
		}
		return nil, err
	}
	return result.(*domain.GatewayChargeResponse), nil
}

func (g *CircuitBreakerGateway) QueryCharge(ctx context.Context, id string) (*domain.GatewayChargeResponse, error) {
	result, err := g.cb.Execute(func() (interface{}, error) {
		return g.next.QueryCharge(ctx, id)
	})
	if err != nil {
		if err == gobreaker.ErrOpenState || err == gobreaker.ErrTooManyRequests {
			return nil, domain.ErrGatewayUnavailable
		}
		return nil, err
	}
	return result.(*domain.GatewayChargeResponse), nil
}

func (g *CircuitBreakerGateway) VoidCharge(ctx context.Context, id string) error {
	_, err := g.cb.Execute(func() (interface{}, error) {
		return nil, g.next.VoidCharge(ctx, id)
	})
	if err != nil {
		if err == gobreaker.ErrOpenState || err == gobreaker.ErrTooManyRequests {
			return domain.ErrGatewayUnavailable
		}
		return err
	}
	return nil
}

func (g *CircuitBreakerGateway) RefundCharge(ctx context.Context, id string, amount int64) error {
	_, err := g.cb.Execute(func() (interface{}, error) {
		return nil, g.next.RefundCharge(ctx, id, amount)
	})
	if err != nil {
		if err == gobreaker.ErrOpenState || err == gobreaker.ErrTooManyRequests {
			return domain.ErrGatewayUnavailable
		}
		return err
	}
	return nil
}
