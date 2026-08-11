package usecase

import (
	"context"
	"time"

	"github.com/Root-Emin/TicketLens/internal/application/iam/dto"
	iamEvent "github.com/Root-Emin/TicketLens/internal/domain/iam/event"
	"github.com/Root-Emin/TicketLens/internal/domain/iam/model"
	"github.com/Root-Emin/TicketLens/internal/domain/iam/repository"
	"github.com/Root-Emin/TicketLens/internal/domain/iam/service"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
	"github.com/Root-Emin/TicketLens/internal/shared/events"
	"github.com/google/uuid"
)

// AssignRoleUseCase handles role assignment.
type AssignRoleUseCase struct {
	roleRepo repository.RoleRepository
	rbac     service.RBACService
	eventBus events.EventBus
}

// ErrRoleNotInOrg is returned when the requested role does not belong to the
// caller's organization. Not-found rather than forbidden, per the API contract:
// a 403 would confirm the id names a real role in some other tenant.
var ErrRoleNotInOrg = domainErr.New(domainErr.ErrNotFound, "role not found", nil)

// NewAssignRoleUseCase creates a new AssignRoleUseCase.
func NewAssignRoleUseCase(roleRepo repository.RoleRepository, rbac service.RBACService, eventBus events.EventBus) *AssignRoleUseCase {
	return &AssignRoleUseCase{roleRepo: roleRepo, rbac: rbac, eventBus: eventBus}
}

// Execute assigns a role to a user within the caller's organization.
//
// orgID is the caller's, taken from the token by the handler — the
// organization_id in the request body is ignored. It used to be trusted, which
// meant a caller holding user:write anywhere could write a user_roles row into
// any tenant whose role id they could obtain; the ids being unguessable is not
// an authorization check. The API contract already states the rule: the
// organization comes from the claims, "never from a request body".
//
// The role is then checked to belong to that organization. Roles are
// per-organization clones (tenant CreateOrgUseCase.provisionRoles), so without
// this a role id from elsewhere is a well-formed value that grants membership
// there.
func (uc *AssignRoleUseCase) Execute(ctx context.Context, orgID uuid.UUID, req dto.AssignRoleRequest) error {
	role, err := uc.roleRepo.GetByID(ctx, req.RoleID)
	if err != nil {
		return err
	}
	if role.ScopeType != model.ScopeTypeOrganization || role.ScopeID != orgID {
		return ErrRoleNotInOrg
	}

	userRole := &model.UserRole{
		UserID:         req.UserID,
		RoleID:         req.RoleID,
		OrganizationID: orgID,
		AppID:          req.AppID,
	}

	if err := uc.roleRepo.AssignRoleToUser(ctx, userRole); err != nil {
		return err
	}

	// Invalidate permission cache
	_ = uc.rbac.InvalidateCache(ctx, req.UserID, orgID)

	// Publish domain event to Kafka
	_ = uc.eventBus.Publish(ctx, events.TopicIAM, iamEvent.RoleAssigned{
		UserID:         req.UserID,
		RoleID:         req.RoleID,
		OrganizationID: orgID,
		Timestamp:      time.Now().UTC(),
	})

	return nil
}
