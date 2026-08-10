package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jrsn-dev/fluuidpay/ecommerce-service/internal/domain"
)

type EcommerceRepository interface {
	SaveOrder(ctx context.Context, order *domain.Order) error
	GetProduct(ctx context.Context, id string) (*domain.Product, error)
	GetCustomer(ctx context.Context, id string) (*domain.Customer, error)
	UpdateOrderStatus(ctx context.Context, orderID, status string) error
	CreateCustomer(ctx context.Context, customer *domain.Customer) error
	CreateProduct(ctx context.Context, product *domain.Product) error
	ListProducts(ctx context.Context) ([]domain.Product, error)
}

type CheckoutUseCase struct {
	repo       EcommerceRepository
	paymentURL string
}

func NewCheckoutUseCase(repo EcommerceRepository, paymentURL string) *CheckoutUseCase {
	return &CheckoutUseCase{
		repo:       repo,
		paymentURL: paymentURL,
	}
}

type CheckoutRequest struct {
	CustomerID string             `json:"customer_id"`
	Items      []domain.OrderItem `json:"items"`
	CardToken  string             `json:"card_token"` // Em produção seria um token PCI
}

type PaymentRequest struct {
	OrderID     string     `json:"order_id"`
	AmountMinor int64      `json:"amount_minor"`
	Currency    string     `json:"currency"`
	CardToken   string     `json:"card_token"`
	TaxContext  TaxContext `json:"tax_context"`
	Destination Address    `json:"destination"`
}

type TaxContext struct {
	ProductType  string `json:"product_type"`
	CustomerType string `json:"customer_type"`
}

type Address struct {
	CountryCode string `json:"country_code"`
	StateCode   string `json:"state_code"`
}

func (uc *CheckoutUseCase) Execute(ctx context.Context, req CheckoutRequest) (*domain.Order, error) {
	customer, err := uc.repo.GetCustomer(ctx, req.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	order := &domain.Order{
		ID:            uuid.New().String(),
		CustomerID:    customer.ID,
		Status:        "PENDING",
		PaymentMethod: "credit_card",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	var productType string

	// Validate items and get current prices
	for _, item := range req.Items {
		product, err := uc.repo.GetProduct(ctx, item.ProductID)
		if err != nil {
			return nil, fmt.Errorf("failed to get product %s: %w", item.ProductID, err)
		}
		order.Items = append(order.Items, domain.OrderItem{
			ProductID:  product.ID,
			Quantity:   item.Quantity,
			PriceMinor: product.PriceMinor,
		})
		productType = product.Type // Simplification: using the type of the last product
	}

	order.CalculateTotal()

	// 1. Save Order as PENDING
	if err := uc.repo.SaveOrder(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	// 2. Call Payment Service
	paymentReq := PaymentRequest{
		OrderID:     order.ID,
		AmountMinor: order.TotalMinor,
		Currency:    "BRL",
		CardToken:   req.CardToken,
		TaxContext: TaxContext{
			ProductType:  productType,
			CustomerType: customer.Type,
		},
		Destination: Address{
			CountryCode: "BR",
			StateCode:   "SP", // Hardcoded for demo
		},
	}

	payload, _ := json.Marshal(paymentReq)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", uc.paymentURL+"/v1/payments", bytes.NewBuffer(payload))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Idempotency-Key", order.ID) // Order ID is unique

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(httpReq)
	
	if err != nil || (resp != nil && resp.StatusCode >= 400) {
		// Se falhou imediatamente, atualiza para REJECTED
		uc.repo.UpdateOrderStatus(ctx, order.ID, "REJECTED")
		order.Status = "REJECTED"
		return order, fmt.Errorf("payment failed to initiate")
	}
	defer resp.Body.Close()

	// Sucesso na criação! O pagamento está PENDING/PROCESSING.
	// O status do pedido será atualizado pelo Consumer do RabbitMQ depois.
	return order, nil
}
