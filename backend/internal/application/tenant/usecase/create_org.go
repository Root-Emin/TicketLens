package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/Root-Emin/TicketLens/internal/application/tenant/dto"
	iamModel "github.com/Root-Emin/TicketLens/internal/domain/iam/model"
	iamRepo "github.com/Root-Emin/TicketLens/internal/domain/iam/repository"
	tenantEvent "github.com/Root-Emin/TicketLens/internal/domain/tenant/event"
	"github.com/Root-Emin/TicketLens/internal/domain/tenant/model"
	"github.com/Root-Emin/TicketLens/internal/domain/tenant/repository"
	triageModel "github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	triageRepo "github.com/Root-Emin/TicketLens/internal/domain/triage/repository"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
	"github.com/Root-Emin/TicketLens/internal/shared/events"
	"github.com/Root-Emin/TicketLens/internal/shared/middleware"
	"github.com/google/uuid"
)

// ownerRoleName is the seeded role granted to whoever creates an organization.
// It carries the "*" permission, scoped to that organization through
// user_roles.organization_id.
const ownerRoleName = "admin"

// defaultDepartmentName is the department every organization starts with. It is
// the fallback owner of tickets whose department is deleted, the department new
// tickets sit in until the classifier picks one, and where predictions land
// when the organization has no department for the predicted category.
//
// It deliberately carries no category: categories route predictions, and the
// default department is the destination for everything that routes nowhere.
const defaultDepartmentName = "General"

// CreateOrgUseCase handles organization creation.
type CreateOrgUseCase struct {
	orgRepo        repository.OrgRepository
	roleRepo       iamRepo.RoleRepository
	departmentRepo triageRepo.DepartmentRepository
	eventBus       events.EventBus
}

// NewCreateOrgUseCase creates a new CreateOrgUseCase.
func NewCreateOrgUseCase(
	orgRepo repository.OrgRepository,
	roleRepo iamRepo.RoleRepository,
	departmentRepo triageRepo.DepartmentRepository,
	eventBus events.EventBus,
) *CreateOrgUseCase {
	return &CreateOrgUseCase{
		orgRepo:        orgRepo,
		roleRepo:       roleRepo,
		departmentRepo: departmentRepo,
		eventBus:       eventBus,
	}
}

// Execute creates a new organization.
func (uc *CreateOrgUseCase) Execute(ctx context.Context, req dto.CreateOrgRequest) (*dto.OrgInfo, error) {
	// Check if slug is taken. A lookup error other than not-found is a real
	// failure; treating it as "slug free" would let a transient fault create a
	// second organization on a slug that is meant to be unique.
	existing, err := uc.orgRepo.GetBySlug(ctx, req.Slug)
	if err != nil && !errors.Is(err, domainErr.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, domainErr.New(domainErr.ErrAlreadyExists, "organization slug already taken", nil)
	}

	org := &model.Organization{
		Name:   req.Name,
		Slug:   req.Slug,
		Status: model.OrgStatusActive,
	}

	if err := uc.orgRepo.Create(ctx, org); err != nil {
		return nil, err
	}

	// Give the new organization its own copy of the seeded roles and make the
	// creator its owner. Without this the creator holds no permissions at all
	// and cannot administer the organization they just created.
	createdBy, _ := middleware.UserIDFromContext(ctx)
	if err := uc.provisionRoles(ctx, org.ID, createdBy); err != nil {
		return nil, err
	}

	// Every organization needs exactly one default department from the start:
	// tickets land there until the classifier assigns one, and deleting any
	// other department moves its tickets here.
	if err := uc.departmentRepo.Create(ctx, &triageModel.Department{
		OrganizationID: org.ID,
		Name:           defaultDepartmentName,
		Description:    "Default department for unclassified requests",
		Category:       nil,
		IsDefault:      true,
	}); err != nil {
		return nil, err
	}

	// Publish domain event to Kafka
	_ = uc.eventBus.Publish(ctx, events.TopicTenant, tenantEvent.OrganizationCreated{
		OrganizationID: org.ID,
		Name:           org.Name,
		CreatedBy:      createdBy,
		Timestamp:      time.Now().UTC(),
	})

	return &dto.OrgInfo{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		Status:    string(org.Status),
		CreatedAt: org.CreatedAt,
	}, nil
}

// provisionRoles clones the seeded template roles (scope_id = TemplateScopeID)
// into the new organization, permissions included, and grants the creator the
// owner role.
//
// NOTE: this runs outside a transaction — the repository interfaces have no
// transaction handle — so a failure part-way leaves the organization created
// with only some of its roles. The error is returned rather than swallowed so
// the caller sees it.
func (uc *CreateOrgUseCase) provisionRoles(ctx context.Context, orgID, ownerID uuid.UUID) error {
	templates, err := uc.roleRepo.ListByScope(ctx, iamModel.ScopeTypeOrganization, iamModel.TemplateScopeID)
	if err != nil {
		return err
	}
	if len(templates) == 0 {
		return domainErr.New(domainErr.ErrInternal,
			"no template roles found; run scripts/seed.go before creating organizations", nil)
	}

	var ownerRoleID uuid.UUID
	for _, tpl := range templates {
		permissions, err := uc.roleRepo.GetPermissions(ctx, tpl.ID)
		if err != nil {
			return err
		}

		role := &iamModel.Role{
			ScopeType:   iamModel.ScopeTypeOrganization,
			ScopeID:     orgID,
			Name:        tpl.Name,
			Description: tpl.Description,
		}
		if err := uc.roleRepo.Create(ctx, role); err != nil {
			return err
		}

		for _, permission := range permissions {
			if err := uc.roleRepo.AddPermission(ctx, role.ID, permission); err != nil {
				return err
			}
		}

		if role.Name == ownerRoleName {
			ownerRoleID = role.ID
		}
	}

	// An unauthenticated caller cannot be made owner; the organization is still
	// usable once somebody is assigned a role in it.
	if ownerID == uuid.Nil || ownerRoleID == uuid.Nil {
		return nil
	}

	return uc.roleRepo.AssignRoleToUser(ctx, &iamModel.UserRole{
		UserID:         ownerID,
		RoleID:         ownerRoleID,
		OrganizationID: orgID,
	})
}
