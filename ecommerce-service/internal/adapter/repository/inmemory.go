package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/jrsn-dev/fluuidpay/ecommerce-service/internal/domain"
)

// InMemoryRepo is a fast implementation for the MVP
type InMemoryRepo struct {
	mu        sync.RWMutex
	customers map[string]*domain.Customer
	products  map[string]*domain.Product
	orders    map[string]*domain.Order
}

func NewInMemoryRepo() *InMemoryRepo {
	return &InMemoryRepo{
		customers: make(map[string]*domain.Customer),
		products:  make(map[string]*domain.Product),
		orders:    make(map[string]*domain.Order),
	}
}

func (r *InMemoryRepo) SaveOrder(ctx context.Context, order *domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[order.ID] = order
	return nil
}

func (r *InMemoryRepo) GetProduct(ctx context.Context, id string) (*domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.products[id]
	if !ok {
		return nil, domain.ErrProductNotFound
	}
	return p, nil
}

func (r *InMemoryRepo) GetCustomer(ctx context.Context, id string) (*domain.Customer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.customers[id]
	if !ok {
		return nil, domain.ErrCustomerNotFound
	}
	return c, nil
}

func (r *InMemoryRepo) UpdateOrderStatus(ctx context.Context, orderID, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	o, ok := r.orders[orderID]
	if !ok {
		return domain.ErrOrderNotFound
	}
	o.Status = status
	fmt.Printf("Order %s updated to status %s\n", orderID, status)
	return nil
}

func (r *InMemoryRepo) CreateCustomer(ctx context.Context, customer *domain.Customer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.customers[customer.ID] = customer
	return nil
}

func (r *InMemoryRepo) CreateProduct(ctx context.Context, product *domain.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.products[product.ID] = product
	return nil
}

func (r *InMemoryRepo) ListProducts(ctx context.Context) ([]domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []domain.Product
	for _, p := range r.products {
		list = append(list, *p)
	}
	return list, nil
}

// Seed adds some dummy data
func (r *InMemoryRepo) Seed() {
	r.CreateProduct(context.Background(), &domain.Product{
		ID:         "prod_1",
		Name:       "E-book Golang Clean Architecture",
		PriceMinor: 4990,
		Type:       "digital_service",
	})
	r.CreateCustomer(context.Background(), &domain.Customer{
		ID:       "cust_1",
		Name:     "John Doe",
		Email:    "john@example.com",
		Document: "12345678909",
		Type:     "individual",
	})
}
