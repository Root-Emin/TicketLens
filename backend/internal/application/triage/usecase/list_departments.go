package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/application/triage/dto"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/repository"
)

// ListDepartmentsUseCase lists an organization's departments with ticket counts.
type ListDepartmentsUseCase struct {
	departmentRepo repository.DepartmentRepository
	ticketRepo     repository.TicketRepository
}

// NewListDepartmentsUseCase creates a new ListDepartmentsUseCase.
func NewListDepartmentsUseCase(
	departmentRepo repository.DepartmentRepository,
	ticketRepo repository.TicketRepository,
) *ListDepartmentsUseCase {
	return &ListDepartmentsUseCase{departmentRepo: departmentRepo, ticketRepo: ticketRepo}
}

// Execute returns every department of the organization, default first.
func (uc *ListDepartmentsUseCase) Execute(ctx context.Context, orgID uuid.UUID) ([]dto.DepartmentInfo, error) {
	departments, err := uc.departmentRepo.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// One grouped query rather than a count per department.
	counts, err := uc.ticketRepo.CountByDepartment(ctx, orgID)
	if err != nil {
		return nil, err
	}

	out := make([]dto.DepartmentInfo, 0, len(departments))
	for _, d := range departments {
		out = append(out, dto.DepartmentInfo{
			ID:          d.ID,
			Name:        d.Name,
			Description: d.Description,
			Category:    d.Category,
			IsDefault:   d.IsDefault,
			TicketCount: counts[d.ID],
		})
	}
	return out, nil
}
