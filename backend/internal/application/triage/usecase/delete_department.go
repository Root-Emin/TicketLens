package usecase

import (
	"context"

	"github.com/Root-Emin/TicketLens/internal/domain/triage/repository"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
	"github.com/google/uuid"
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

	/*
		Staff are deliberately NOT moved with the tickets.

		staff_departments cascades on delete (migration 00021), so the people on
		this team come back unassigned and surface in the roster's Unassigned
		bucket. That asymmetry is the point: a ticket has to belong somewhere or
		it falls out of every queue, whereas a person placed on a team they were
		never chosen for is a staffing decision made silently by a delete button.
		Tickets get a safe default; people get a decision.
	*/
	return uc.departmentRepo.Delete(ctx, orgID, departmentID)
}
