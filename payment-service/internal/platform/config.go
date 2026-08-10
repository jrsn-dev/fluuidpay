package platform

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	RabbitMQ RabbitMQConfig
	Gateway  GatewayConfig
	Tax      TaxConfig
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host            string
	Port            int
	KeycloakJWKSURL string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// DSN returns the PostgreSQL connection string.
func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.SSLMode,
	)
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addr            string
	Password        string
	DB              int
	IdempotencyTTL  time.Duration
}

// RabbitMQConfig holds RabbitMQ connection settings.
type RabbitMQConfig struct {
	URL              string
	Exchange         string
	DeadLetterExch   string
	MainQueue        string
	DLQ              string
	RoutingKey       string
	RetryQueues      []string
	RetryDelaysMs    []int32
	MaxRetries       int
	PrefetchCount    int
	ConsumerName     string
}

// GatewayConfig holds payment gateway settings.
type GatewayConfig struct {
	BaseURL        string
	APIKey         string
	Timeout        time.Duration
	MaxRetries     int
	RetryBaseDelay time.Duration
}

// TaxConfig holds tax calculation settings.
type TaxConfig struct {
	RulesPath      string
	DefaultVersion string
}

// LoadConfig reads configuration from environment variables with sensible defaults.
func LoadConfig() Config {
	return Config{
		Server: ServerConfig{
			Host:            envOrDefault("SERVER_HOST", "0.0.0.0"),
			Port:            envIntOrDefault("SERVER_PORT", 8080),
			KeycloakJWKSURL: envOrDefault("KEYCLOAK_JWKS_URL", ""),
			ReadTimeout:     envDurationOrDefault("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    envDurationOrDefault("SERVER_WRITE_TIMEOUT", 15*time.Second),
			ShutdownTimeout: envDurationOrDefault("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			Host:            envOrDefault("DB_HOST", "localhost"),
			Port:            envIntOrDefault("DB_PORT", 5432),
			User:            envOrDefault("DB_USER", "payment"),
			Password:        envOrDefault("DB_PASSWORD", "payment"),
			Database:        envOrDefault("DB_NAME", "payment_service"),
			SSLMode:         envOrDefault("DB_SSLMODE", "disable"),
			MaxOpenConns:    envIntOrDefault("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    envIntOrDefault("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: envDurationOrDefault("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Addr:           envOrDefault("REDIS_ADDR", "localhost:6379"),
			Password:       envOrDefault("REDIS_PASSWORD", ""),
			DB:             envIntOrDefault("REDIS_DB", 0),
			IdempotencyTTL: envDurationOrDefault("REDIS_IDEMPOTENCY_TTL", 24*time.Hour),
		},
		RabbitMQ: RabbitMQConfig{
			URL:            envOrDefault("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
			Exchange:       envOrDefault("RABBITMQ_EXCHANGE", "payment.events"),
			DeadLetterExch: envOrDefault("RABBITMQ_DLX", "payment.dlx"),
			MainQueue:      envOrDefault("RABBITMQ_MAIN_QUEUE", "payment.processed"),
			DLQ:            envOrDefault("RABBITMQ_DLQ", "payment.processed.dlq"),
			RoutingKey:     envOrDefault("RABBITMQ_ROUTING_KEY", "payment.processed"),
			RetryQueues: []string{
				"payment.processed.retry.1",
				"payment.processed.retry.2",
				"payment.processed.retry.3",
			},
			RetryDelaysMs:  []int32{10_000, 60_000, 300_000},
			MaxRetries:     envIntOrDefault("RABBITMQ_MAX_RETRIES", 3),
			PrefetchCount:  envIntOrDefault("RABBITMQ_PREFETCH", 20),
			ConsumerName:   envOrDefault("RABBITMQ_CONSUMER_NAME", "cart-payment-consumer"),
		},
		Gateway: GatewayConfig{
			BaseURL:        envOrDefault("GATEWAY_BASE_URL", "https://sandbox-api.example.com"),
			APIKey:         envOrDefault("GATEWAY_API_KEY", ""),
			Timeout:        envDurationOrDefault("GATEWAY_TIMEOUT", 10*time.Second),
			MaxRetries:     envIntOrDefault("GATEWAY_MAX_RETRIES", 3),
			RetryBaseDelay: envDurationOrDefault("GATEWAY_RETRY_BASE_DELAY", 500*time.Millisecond),
		},
		Tax: TaxConfig{
			RulesPath:      envOrDefault("TAX_RULES_PATH", "config/tax_rules.yaml"),
			DefaultVersion: envOrDefault("TAX_DEFAULT_VERSION", "2026-01"),
		},
	}
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func envIntOrDefault(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}

func envDurationOrDefault(key string, defaultVal time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultVal
}
