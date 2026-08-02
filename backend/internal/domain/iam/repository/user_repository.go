package repository

import (
	"context"

	"github.com/Root-Emin/TicketLens/internal/domain/iam/model"
	"github.com/google/uuid"
)

// UserRepository defines the interface for user persistence.
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]*model.User, int, error)

	// ListByOrganization returns the users assigned to a role in the given
	// organization. Callers serving HTTP must prefer it over List: List spans
	// the whole platform and would disclose accounts from other tenants.
	ListByOrganization(ctx context.Context, orgID uuid.UUID, offset, limit int) ([]*model.User, int, error)
}
