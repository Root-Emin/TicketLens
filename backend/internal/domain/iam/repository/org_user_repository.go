package repository

import (
	"context"

	"github.com/Root-Emin/TicketLens/internal/domain/iam/model"
	"github.com/google/uuid"
)

// OrgUserRepository defines the interface for organization-user membership persistence.
type OrgUserRepository interface {
	Add(ctx context.Context, orgUser *model.OrganizationUser) error
	Remove(ctx context.Context, orgID, userID uuid.UUID) error
	GetByOrgAndUser(ctx context.Context, orgID, userID uuid.UUID) (*model.OrganizationUser, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID, offset, limit int) ([]*model.OrganizationUser, int, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*model.OrganizationUser, error)
}
