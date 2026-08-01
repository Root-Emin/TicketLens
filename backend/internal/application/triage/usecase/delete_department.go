package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// DeleteDepartmentUseCase handles department removal.
type DeleteDepartmentUseCase struct {
	departmentRepo repository.DepartmentRepository
	ticketRepo     repository.TicketRepository
}

// NewDeleteDepartmentUseCase creates a new DeleteDepartmentUseCase.
func NewDeleteDepartmentUseCase(
	departmentRepo repository.DepartmentRepository,
	ticketRepo repository.TicketRepository,
) *DeleteDepartmentUseCase {
	return &DeleteDepartmentUseCase{departmentRepo: departmentRepo, ticketRepo: ticketRepo}
}

// Execute deletes a department after moving its tickets to the organization's
// default department. The default department itself cannot be deleted.
func (uc *DeleteDepartmentUseCase) Execute(ctx context.Context, orgID, departmentID uuid.UUID) error {
	department, err := uc.departmentRepo.GetByID(ctx, orgID, departmentID)
	if err != nil {
		return err
	}

	if !department.IsDeletable() {
		return domainErr.New(domainErr.ErrConflict, "the default department cannot be deleted", nil)
	}

	fallback, err := uc.departmentRepo.GetDefault(ctx, orgID)
	if err != nil {
		return err
	}

	// Tickets move first: the tickets.department_id foreign key has no ON DELETE
	// action, so the delete below would fail while rows still point here.
	if _, err := uc.ticketRepo.ReassignDepartment(ctx, orgID, departmentID, fallback.ID); err != nil {
		return err
	}

	return uc.departmentRepo.Delete(ctx, orgID, departmentID)
}
