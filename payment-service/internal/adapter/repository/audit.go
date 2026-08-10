package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fluuid/payment-service/internal/domain"
	"github.com/fluuid/payment-service/internal/platform"
)

type AuditRepo struct {
	pool *pgxpool.Pool
}

func NewAuditRepo(pool *pgxpool.Pool) *AuditRepo {
	return &AuditRepo{pool: pool}
}

func (r *AuditRepo) Insert(ctx context.Context, log *domain.AuditLog) error {
	query := `
		INSERT INTO audit_logs (id, entity_id, entity_type, event_type, actor, old_state, new_state, reason, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	
	var executor interface {
		Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	}
	if tx, ok := platform.TxFromContext(ctx); ok {
		executor = tx
	} else {
		executor = r.pool
	}
	
	_, err := executor.Exec(ctx, query,
		log.ID, log.EntityID, log.EntityType, log.EventType,
		log.Actor, log.OldState, log.NewState, log.Reason, log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	
	return nil
}
