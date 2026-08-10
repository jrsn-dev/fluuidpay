package platform

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// RedisClient wraps the go-redis client.
type RedisClient struct {
	Client *redis.Client
}

// NewRedisClient creates a new Redis connection.
func NewRedisClient(cfg RedisConfig) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Verify connectivity
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return &RedisClient{Client: client}, nil
}

// Close releases the Redis connection.
func (r *RedisClient) Close() error {
	if r.Client != nil {
		return r.Client.Close()
	}
	return nil
}

// HealthCheck verifies Redis is reachable.
func (r *RedisClient) HealthCheck(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}
