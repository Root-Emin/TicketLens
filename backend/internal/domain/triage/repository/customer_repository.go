package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/model"
)

// CustomerRepository defines the interface for customer persistence.
type CustomerRepository interface {
	Create(ctx context.Context, customer *model.Customer) error
	GetByID(ctx context.Context, orgID, id uuid.UUID) (*model.Customer, error)
	GetByEmail(ctx context.Context, orgID uuid.UUID, email string) (*model.Customer, error)
	// ListByOrg returns customers for an organization. When query is non-empty it
	// matches a substring of either email or full name.
	ListByOrg(ctx context.Context, orgID uuid.UUID, query string, offset, limit int) ([]*model.Customer, int, error)
	// ListByIDs returns the named customers keyed by ID, so a ticket listing can
	// resolve its customers in one round trip instead of one per row.
	ListByIDs(ctx context.Context, orgID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]*model.Customer, error)
	Update(ctx context.Context, customer *model.Customer) error
	Delete(ctx context.Context, orgID, id uuid.UUID) error
}
