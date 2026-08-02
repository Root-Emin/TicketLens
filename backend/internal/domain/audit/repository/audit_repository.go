package repository

import (
	"context"

	"github.com/Root-Emin/TicketLens/internal/domain/audit/model"
	"github.com/google/uuid"
)

// AuditRepository defines the interface for audit log persistence.
type AuditRepository interface {
	Create(ctx context.Context, log *model.AuditLog) error
	ListByOrg(ctx context.Context, orgID uuid.UUID, offset, limit int) ([]*model.AuditLog, int, error)
	ListByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*model.AuditLog, int, error)
	ListByResource(ctx context.Context, resourceType, resourceID string, offset, limit int) ([]*model.AuditLog, int, error)
}
