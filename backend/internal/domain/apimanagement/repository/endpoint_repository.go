package repository

import (
	"context"

	"github.com/Root-Emin/TicketLens/internal/domain/apimanagement/model"
	"github.com/google/uuid"
)

// EndpointRepository defines the interface for endpoint persistence.
type EndpointRepository interface {
	Create(ctx context.Context, endpoint *model.Endpoint) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Endpoint, error)
	GetByMethodPath(ctx context.Context, appID uuid.UUID, method, path, version string) (*model.Endpoint, error)
	Update(ctx context.Context, endpoint *model.Endpoint) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByApp(ctx context.Context, appID uuid.UUID, offset, limit int) ([]*model.Endpoint, int, error)
}
