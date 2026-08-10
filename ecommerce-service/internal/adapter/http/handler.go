package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jrsn-dev/fluuidpay/ecommerce-service/internal/usecase"
)

type Handler struct {
	checkoutUC *usecase.CheckoutUseCase
	repo       usecase.EcommerceRepository
}

func NewHandler(checkoutUC *usecase.CheckoutUseCase, repo usecase.EcommerceRepository) *chi.Mux {
	h := &Handler{
		checkoutUC: checkoutUC,
		repo:       repo,
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/v1/products", h.listProducts)
	r.Post("/v1/orders", h.createOrder)

	return r
}

func (h *Handler) listProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.repo.ListProducts(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	var req usecase.CheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	order, err := h.checkoutUC.Execute(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}
