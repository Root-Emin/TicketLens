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
	// UpdatePassword replaces one user's password hash.
	//
	// Separate from Update, which deliberately does not touch password_hash: a
	// caller that loaded a user, changed a name and saved it would otherwise
	// rewrite the credential as a side effect, and one that built a User from a
	// request DTO would blank it. Password changes are rare, dangerous and
	// worth their own verb.
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]*model.User, int, error)

	// ListByOrganization returns the users assigned to a role in the given
	// organization. Callers serving HTTP must prefer it over List: List spans
	// the whole platform and would disclose accounts from other tenants.
	ListByOrganization(ctx context.Context, orgID uuid.UUID, offset, limit int) ([]*model.User, int, error)
}
