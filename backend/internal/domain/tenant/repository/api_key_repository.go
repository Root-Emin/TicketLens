package repository

import (
	"context"

	"github.com/Root-Emin/TicketLens/internal/domain/tenant/model"
	"github.com/google/uuid"
)

// APIKeyRepository defines the interface for API key persistence.
type APIKeyRepository interface {
	Create(ctx context.Context, key *model.AppAPIKey) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.AppAPIKey, error)
	GetByHash(ctx context.Context, hash string) (*model.AppAPIKey, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	ListByApp(ctx context.Context, appID uuid.UUID) ([]*model.AppAPIKey, error)
}
