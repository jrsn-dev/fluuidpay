package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	ecommercehttp "github.com/jrsn-dev/fluuidpay/ecommerce-service/internal/adapter/http"
	"github.com/jrsn-dev/fluuidpay/ecommerce-service/internal/adapter/messaging"
	"github.com/jrsn-dev/fluuidpay/ecommerce-service/internal/adapter/repository"
	"github.com/jrsn-dev/fluuidpay/ecommerce-service/internal/usecase"
)

func main() {
	// Initialize Repository (In-Memory for MVP)
	repo := repository.NewInMemoryRepo()
	repo.Seed() // Add dummy products and customers

	paymentURL := os.Getenv("PAYMENT_SERVICE_URL")
	if paymentURL == "" {
		paymentURL = "http://localhost:8080" // Default for local dev without docker
	}

	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	if rabbitmqURL == "" {
		rabbitmqURL = "amqp://guest:guest@localhost:5672/"
	}

	// Initialize Use Cases
	checkoutUC := usecase.NewCheckoutUseCase(repo, paymentURL)

	// Initialize HTTP Handler
	router := ecommercehttp.NewHandler(checkoutUC, repo)

	// Start RabbitMQ Consumer
	consumer, err := messaging.NewConsumer(rabbitmqURL, repo)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		log.Println("Starting RabbitMQ Consumer...")
		if err := consumer.Start(ctx); err != nil {
			log.Printf("Consumer error: %v", err)
		}
	}()

	// Start HTTP Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082" // Payment Service uses 8080, Keycloak uses 8081
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		log.Printf("Ecommerce Service listening on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gracefully...")
	cancel() // Stop consumer

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Ecommerce Service exited properly")
}
