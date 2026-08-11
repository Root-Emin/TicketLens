package usecase

import (
	"context"

	"github.com/Root-Emin/TicketLens/internal/application/iam/dto"
	"github.com/Root-Emin/TicketLens/internal/domain/iam/model"
	"github.com/Root-Emin/TicketLens/internal/domain/iam/repository"
	"github.com/google/uuid"
)

// customerRoleName is the seeded role a portal account holds. Excluded from the
// assignable list below.
const customerRoleName = "customer"

// ListRolesUseCase reads the roles an organization can assign to staff.
type ListRolesUseCase struct {
	roles repository.RoleRepository
}

// NewListRolesUseCase creates a new ListRolesUseCase.
func NewListRolesUseCase(roles repository.RoleRepository) *ListRolesUseCase {
	return &ListRolesUseCase{roles: roles}
}

// Execute returns the organization's staff roles.
//
// This endpoint exists because nothing else could name a role. Inviting
// somebody or assigning a role takes a role id, and the ids are per-organization
// clones created when the organization was — so a client had no way to obtain
// one, and the admin panel said as much in a comment rather than offering the
// action.
//
// The customer role is filtered out here rather than in the caller. It is a real
// role and it is assignable in principle, but the screens that consume this list
// are for staffing a support team: offering it there invites somebody into the
// organization as a portal account, which puts them in no roster and looks like
// a bug from every screen an administrator can see. Deciding it once, server
// side, keeps two future clients from disagreeing about it.
func (uc *ListRolesUseCase) Execute(ctx context.Context, orgID uuid.UUID) ([]dto.RoleInfo, error) {
	roles, err := uc.roles.ListByScope(ctx, model.ScopeTypeOrganization, orgID)
	if err != nil {
		return nil, err
	}

	out := make([]dto.RoleInfo, 0, len(roles))
	for _, role := range roles {
		if role.Name == customerRoleName {
			continue
		}
		out = append(out, dto.RoleInfo{
			ID:          role.ID,
			Name:        role.Name,
			Description: role.Description,
		})
	}
	return out, nil
}
