package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sony/gobreaker"

	"github.com/fluuid/payment-service/internal/adapter/gateway"
	httphandler "github.com/fluuid/payment-service/internal/adapter/http"
	"github.com/fluuid/payment-service/internal/adapter/messaging"
	"github.com/fluuid/payment-service/internal/adapter/repository"
	"github.com/fluuid/payment-service/internal/adapter/tax"
	"github.com/fluuid/payment-service/internal/platform"
	"github.com/fluuid/payment-service/internal/usecase"
)

func main() {
	// Structured logger
	logger := platform.NewLogger()
	slog.SetDefault(logger)

	cfg := platform.LoadConfig()

	logger.Info("starting payment service",
		"host", cfg.Server.Host,
		"port", cfg.Server.Port,
	)

	// --- Initialize infrastructure ---

	// Database
	db, err := platform.NewDatabase(cfg.Database)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database connected")

	// Redis
	rdb, err := platform.NewRedisClient(cfg.Redis)
	if err != nil {
		logger.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()
	logger.Info("redis connected")

	// RabbitMQ
	rmq, err := platform.NewRabbitMQ(cfg.RabbitMQ)
	if err != nil {
		logger.Error("failed to connect to rabbitmq", "error", err)
		os.Exit(1)
	}
	defer rmq.Close()
	logger.Info("rabbitmq connected")

	// Declare RabbitMQ topology
	topology := messaging.DefaultTopology()
	if err := messaging.Declare(rmq.Channel, topology); err != nil {
		logger.Error("failed to declare messaging topology", "error", err)
		os.Exit(1)
	}
	logger.Info("messaging topology declared")

	// --- Build dependency graph ---

	// Repositories
	txManager := platform.NewTxManager(db.Pool)
	paymentRepo := repository.NewPaymentRepo(db.Pool)
	idempotencyRepo := repository.NewIdempotencyRepo(db.Pool, rdb.Client, cfg.Redis.IdempotencyTTL)
	outboxRepo := repository.NewOutboxRepo(db.Pool)
	auditRepo := repository.NewAuditRepo(db.Pool)

	// Adapters
	taxCalc, err := tax.NewCalculator(cfg.Tax.RulesPath)
	if err != nil {
		logger.Error("failed to initialize tax calculator", "error", err)
		os.Exit(1)
	}
	gwMock := gateway.NewMockGateway()
	
	cbSettings := gobreaker.Settings{
		Name:        "PaymentGateway",
		MaxRequests: 3,
		Interval:    5 * time.Second,
		Timeout:     10 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 5 && failureRatio >= 0.2 // Open if >= 5 reqs and >= 20% fail
		},
	}
	gw := gateway.NewCircuitBreakerGateway(gwMock, cbSettings)

	// Use cases
	createPaymentUC := usecase.NewCreatePaymentUseCase(
		paymentRepo, idempotencyRepo, outboxRepo, auditRepo, txManager,
		gw, taxCalc, logger,
	)
	getPaymentUC := usecase.NewGetPaymentUseCase(paymentRepo)
	cancelPaymentUC := usecase.NewCancelPaymentUseCase(
		paymentRepo, outboxRepo, txManager, gw, logger,
	)

	processWebhookUC := usecase.NewProcessWebhookUseCase(
		paymentRepo, outboxRepo, auditRepo, txManager, "secret", logger,
	)

	// HTTP handler
	handler, err := httphandler.NewHandler(createPaymentUC, getPaymentUC, cancelPaymentUC, processWebhookUC, logger, cfg.Server.KeycloakJWKSURL)
	if err != nil {
		logger.Error("failed to initialize http handler", "error", err)
		os.Exit(1)
	}

	// --- HTTP Server ---
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in goroutine
	go func() {
		logger.Info("http server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	// --- Outbox publisher goroutine ---
	pubCtx, pubCancel := context.WithCancel(context.Background())
	defer pubCancel()

	outboxPublisher := messaging.NewOutboxPublisher(outboxRepo, rmq.Channel, topology, logger)
	go outboxPublisher.Run(pubCtx)

	// --- Graceful shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info("shutdown signal received", "signal", sig.String())

	// Stop outbox publisher
	pubCancel()

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("http server shutdown error", "error", err)
	}

	logger.Info("closing database and messaging connections...")
	rmq.Close()
	if err := rdb.Close(); err != nil {
		logger.Error("failed to close redis", "error", err)
	}
	db.Close()

	logger.Info("payment service stopped")
}
