package repository

import (
	"context"

	"github.com/Root-Emin/TicketLens/internal/domain/apimanagement/model"
	"github.com/google/uuid"
)

// PolicyRepository defines the interface for endpoint policy persistence.
type PolicyRepository interface {
	Create(ctx context.Context, policy *model.EndpointPolicy) error
	GetByEndpointID(ctx context.Context, endpointID uuid.UUID) (*model.EndpointPolicy, error)
	Update(ctx context.Context, policy *model.EndpointPolicy) error
	Delete(ctx context.Context, id uuid.UUID) error
}
