package usecase

import (
	"context"
	"errors"
	"strings"

	"github.com/Root-Emin/TicketLens/internal/application/triage/dto"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/repository"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
	"github.com/google/uuid"
)

// CreateDepartmentUseCase handles department creation.
type CreateDepartmentUseCase struct {
	departmentRepo repository.DepartmentRepository
}

// NewCreateDepartmentUseCase creates a new CreateDepartmentUseCase.
func NewCreateDepartmentUseCase(departmentRepo repository.DepartmentRepository) *CreateDepartmentUseCase {
	return &CreateDepartmentUseCase{departmentRepo: departmentRepo}
}

// Execute creates a department. New departments are never the default one:
// an organization keeps the single default it was created with.
func (uc *CreateDepartmentUseCase) Execute(ctx context.Context, orgID uuid.UUID, req dto.CreateDepartmentRequest) (*dto.DepartmentInfo, error) {
	name := strings.TrimSpace(req.Name)

	existing, err := uc.departmentRepo.GetByName(ctx, orgID, name)
	if err != nil && !errors.Is(err, domainErr.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, domainErr.New(domainErr.ErrAlreadyExists, "department name already taken in this organization", nil)
	}

	department := &model.Department{
		OrganizationID: orgID,
		Name:           name,
		Description:    strings.TrimSpace(req.Description),
		IsDefault:      false,
	}

	if req.Category != nil && *req.Category != "" {
		category := model.Category(*req.Category)
		if !model.ValidCategory(category) {
			return nil, domainErr.New(domainErr.ErrValidation, "unknown category: "+*req.Category, nil)
		}
		// One department per category keeps the classifier mapping deterministic.
		taken, err := uc.departmentRepo.GetByCategory(ctx, orgID, category)
		if err != nil && !errors.Is(err, domainErr.ErrNotFound) {
			return nil, err
		}
		if taken != nil {
			return nil, domainErr.New(domainErr.ErrAlreadyExists,
				"another department already handles category "+*req.Category, nil)
		}
		department.Category = &category
	}

	if err := uc.departmentRepo.Create(ctx, department); err != nil {
		return nil, err
	}

	return &dto.DepartmentInfo{
		ID:          department.ID,
		Name:        department.Name,
		Description: department.Description,
		Category:    department.Category,
		IsDefault:   department.IsDefault,
		TicketCount: 0,
	}, nil
}
