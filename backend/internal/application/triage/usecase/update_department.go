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

// UpdateDepartmentUseCase handles department updates.
type UpdateDepartmentUseCase struct {
	departmentRepo repository.DepartmentRepository
	ticketRepo     repository.TicketRepository
}

// NewUpdateDepartmentUseCase creates a new UpdateDepartmentUseCase.
func NewUpdateDepartmentUseCase(
	departmentRepo repository.DepartmentRepository,
	ticketRepo repository.TicketRepository,
) *UpdateDepartmentUseCase {
	return &UpdateDepartmentUseCase{departmentRepo: departmentRepo, ticketRepo: ticketRepo}
}

// Execute patches a department's name and/or description. is_default is not
// editable: the default department is fixed at organization creation.
func (uc *UpdateDepartmentUseCase) Execute(ctx context.Context, orgID, departmentID uuid.UUID, req dto.UpdateDepartmentRequest) (*dto.DepartmentInfo, error) {
	department, err := uc.departmentRepo.GetByID(ctx, orgID, departmentID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, domainErr.New(domainErr.ErrValidation, "name cannot be empty", nil)
		}
		existing, err := uc.departmentRepo.GetByName(ctx, orgID, name)
		if err != nil && !errors.Is(err, domainErr.ErrNotFound) {
			return nil, err
		}
		if existing != nil && existing.ID != departmentID {
			return nil, domainErr.New(domainErr.ErrAlreadyExists, "department name already taken in this organization", nil)
		}
		department.Name = name
	}

	if req.Description != nil {
		department.Description = strings.TrimSpace(*req.Description)
	}

	// An empty string clears the category; omitting the field leaves it alone.
	if req.Category != nil {
		if *req.Category == "" {
			department.Category = nil
		} else {
			category := model.Category(*req.Category)
			if !model.ValidCategory(category) {
				return nil, domainErr.New(domainErr.ErrValidation, "unknown category: "+*req.Category, nil)
			}
			taken, err := uc.departmentRepo.GetByCategory(ctx, orgID, category)
			if err != nil && !errors.Is(err, domainErr.ErrNotFound) {
				return nil, err
			}
			if taken != nil && taken.ID != departmentID {
				return nil, domainErr.New(domainErr.ErrAlreadyExists,
					"another department already handles category "+*req.Category, nil)
			}
			department.Category = &category
		}
	}

	if err := uc.departmentRepo.Update(ctx, department); err != nil {
		return nil, err
	}

	counts, err := uc.ticketRepo.CountByDepartment(ctx, orgID)
	if err != nil {
		return nil, err
	}

	return &dto.DepartmentInfo{
		ID:          department.ID,
		Name:        department.Name,
		Description: department.Description,
		Category:    department.Category,
		IsDefault:   department.IsDefault,
		TicketCount: counts[department.ID],
	}, nil
}
