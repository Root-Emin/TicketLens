package repository

import (
	"context"

	"github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	"github.com/google/uuid"
)

// CustomerRepository defines the interface for customer persistence.
type CustomerRepository interface {
	Create(ctx context.Context, customer *model.Customer) error
	GetByID(ctx context.Context, orgID, id uuid.UUID) (*model.Customer, error)
	GetByEmail(ctx context.Context, orgID uuid.UUID, email string) (*model.Customer, error)
	// GetByUserID resolves the customer a signed-in account belongs to. This is
	// how a portal request learns whose tickets it may see; it must never be
	// replaced by a lookup on anything the client supplies.
	GetByUserID(ctx context.Context, orgID, userID uuid.UUID) (*model.Customer, error)
	// ListByOrg returns customers for an organization. When query is non-empty it
	// matches a substring of either email or full name.
	ListByOrg(ctx context.Context, orgID uuid.UUID, query string, offset, limit int) ([]*model.Customer, int, error)
	// ListByIDs returns the named customers keyed by ID, so a ticket listing can
	// resolve its customers in one round trip instead of one per row.
	ListByIDs(ctx context.Context, orgID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]*model.Customer, error)
	Update(ctx context.Context, customer *model.Customer) error
	Delete(ctx context.Context, orgID, id uuid.UUID) error
}
