package domain

import (
	"context"
	"time"
)

// AuditLog represents a record in the audit trail.
type AuditLog struct {
	ID         string
	EntityID   string
	EntityType string
	EventType  string
	Actor      string
	OldState   string
	NewState   string
	Reason     string
	CreatedAt  time.Time
}

// AuditRepository defines the interface for persisting audit logs.
type AuditRepository interface {
	// Insert creates a new audit log entry within the current transaction.
	Insert(ctx context.Context, log *AuditLog) error
}
