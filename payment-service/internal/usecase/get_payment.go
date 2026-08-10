package usecase

import (
	"context"

	"github.com/fluuid/payment-service/internal/domain"
)

// GetPaymentUseCase retrieves a payment by its transaction ID.
type GetPaymentUseCase struct {
	paymentRepo domain.PaymentRepository
}

// NewGetPaymentUseCase creates a new GetPaymentUseCase.
func NewGetPaymentUseCase(paymentRepo domain.PaymentRepository) *GetPaymentUseCase {
	return &GetPaymentUseCase{paymentRepo: paymentRepo}
}

// Execute retrieves a payment by ID.
func (uc *GetPaymentUseCase) Execute(ctx context.Context, transactionID string) (*CreatePaymentOutput, error) {
	payment, err := uc.paymentRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}

	output := &CreatePaymentOutput{
		TransactionID:         payment.ID,
		OrderID:               payment.OrderID,
		UserID:                payment.UserID,
		Status:                payment.Status,
		AmountMinor:           payment.Amount.AmountMinor,
		Currency:              payment.Amount.Currency,
		ProviderTransactionID: payment.ProviderTransactionID,
		Taxes:                 payment.Taxes,
		Failure:               payment.Failure,
		CreatedAt:             payment.CreatedAt,
		UpdatedAt:             payment.UpdatedAt,
		HTTPCode:              200,
	}

	return output, nil
}
