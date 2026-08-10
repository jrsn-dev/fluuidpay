package platform

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Database wraps a PostgreSQL connection pool.
type Database struct {
	Pool *pgxpool.Pool
}

// NewDatabase creates a new PostgreSQL connection pool.
func NewDatabase(cfg DatabaseConfig) (*Database, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	poolCfg.MaxConns = int32(cfg.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.MaxIdleConns)
	poolCfg.MaxConnLifetime = cfg.ConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	// Verify connectivity
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Database{Pool: pool}, nil
}

// Close releases all database connections.
func (d *Database) Close() {
	if d.Pool != nil {
		d.Pool.Close()
	}
}

// HealthCheck verifies the database is reachable.
func (d *Database) HealthCheck(ctx context.Context) error {
	return d.Pool.Ping(ctx)
}

// TxManagerImpl implements domain.TxManager using pgx transactions.
type TxManagerImpl struct {
	pool *pgxpool.Pool
}

// NewTxManager creates a new transaction manager.
func NewTxManager(pool *pgxpool.Pool) *TxManagerImpl {
	return &TxManagerImpl{pool: pool}
}

// WithTx executes fn within a database transaction.
// If fn returns nil, the transaction is committed; otherwise it is rolled back.
func (tm *TxManagerImpl) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := tm.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// Store the tx in context so repositories can access it
	txCtx := context.WithValue(ctx, txKey{}, tx)

	if err := fn(txCtx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// txKey is the context key for the current transaction.
type txKey struct{}

// TxFromContext extracts the current transaction from context, if any.
func TxFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	return tx, ok
}
