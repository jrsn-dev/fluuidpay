package domain

import (
	"errors"
	"time"
)

var (
	ErrCustomerNotFound = errors.New("customer not found")
	ErrProductNotFound  = errors.New("product not found")
	ErrOrderNotFound    = errors.New("order not found")
	ErrInvalidOrder     = errors.New("invalid order data")
)

type Customer struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Document  string    `json:"document"` // CPF/CNPJ
	Type      string    `json:"type"`     // "individual", "company"
	CreatedAt time.Time `json:"created_at"`
}

type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	PriceMinor  int64     `json:"price_minor"`
	Type        string    `json:"type"` // "physical_goods", "digital_service"
	CreatedAt   time.Time `json:"created_at"`
}

type OrderItem struct {
	ProductID  string `json:"product_id"`
	Quantity   int    `json:"quantity"`
	PriceMinor int64  `json:"price_minor"` // Snapshot of price at purchase
}

type Order struct {
	ID            string      `json:"id"`
	CustomerID    string      `json:"customer_id"`
	Items         []OrderItem `json:"items"`
	TotalMinor    int64       `json:"total_minor"`
	Status        string      `json:"status"` // PENDING, PAID, REJECTED
	PaymentID     string      `json:"payment_id"`
	PaymentMethod string      `json:"payment_method"` // Token
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

func (o *Order) CalculateTotal() {
	var total int64 = 0
	for _, item := range o.Items {
		total += item.PriceMinor * int64(item.Quantity)
	}
	o.TotalMinor = total
}
