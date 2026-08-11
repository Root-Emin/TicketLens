package usecase

import (
	"context"

	"github.com/Root-Emin/TicketLens/internal/application/triage/dto"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/repository"
	"github.com/google/uuid"
)

// ListDepartmentsUseCase lists an organization's departments with their ticket
// and staff counts.
type ListDepartmentsUseCase struct {
	departmentRepo repository.DepartmentRepository
	ticketRepo     repository.TicketRepository
	staffRepo      repository.StaffRepository
}

// NewListDepartmentsUseCase creates a new ListDepartmentsUseCase.
func NewListDepartmentsUseCase(
	departmentRepo repository.DepartmentRepository,
	ticketRepo repository.TicketRepository,
	staffRepo repository.StaffRepository,
) *ListDepartmentsUseCase {
	return &ListDepartmentsUseCase{
		departmentRepo: departmentRepo,
		ticketRepo:     ticketRepo,
		staffRepo:      staffRepo,
	}
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

	// Likewise for headcount. Nil-tolerant so that a caller wiring this use
	// case without a staff repository still gets departments and tickets rather
	// than a panic — the counts then read zero, which is what an organization
	// with nobody assigned looks like anyway.
	staffCounts := map[uuid.UUID]int{}
	if uc.staffRepo != nil {
		staffCounts, err = uc.staffRepo.CountByDepartment(ctx, orgID)
		if err != nil {
			return nil, err
		}
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
			StaffCount:  staffCounts[d.ID],
		})
	}
	return out, nil
}
