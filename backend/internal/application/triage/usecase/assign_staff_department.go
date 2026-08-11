package usecase

import (
	"context"

	"github.com/Root-Emin/TicketLens/internal/application/triage/dto"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/repository"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
	"github.com/google/uuid"
)

// AssignStaffDepartmentUseCase places a staff member on a team, or removes them
// from one.
type AssignStaffDepartmentUseCase struct {
	staffRepo      repository.StaffRepository
	departmentRepo repository.DepartmentRepository
}

// NewAssignStaffDepartmentUseCase creates a new AssignStaffDepartmentUseCase.
func NewAssignStaffDepartmentUseCase(
	staffRepo repository.StaffRepository,
	departmentRepo repository.DepartmentRepository,
) *AssignStaffDepartmentUseCase {
	return &AssignStaffDepartmentUseCase{staffRepo: staffRepo, departmentRepo: departmentRepo}
}

// Execute sets one person's department, returning their updated roster entry.
//
// Two checks precede the write, and both are about the organization rather than
// about validity:
//
//   - the user has to be staff *in this organization*. StaffRepo.GetByUser is
//     scoped, so this refuses a user id belonging to another tenant, and it
//     refuses one of this organization's own portal customers — a customer is
//     not somebody you put on the Payments team.
//   - the department has to belong to the same organization. Without this,
//     a caller could post another tenant's department id and quietly create a
//     cross-tenant assignment that no listing would ever surface, because every
//     read is scoped and would filter it straight back out.
//
// The foreign keys alone would not catch either: they check that the rows
// exist, not that they belong together.
func (uc *AssignStaffDepartmentUseCase) Execute(
	ctx context.Context,
	orgID, userID uuid.UUID,
	req dto.AssignStaffDepartmentRequest,
) (dto.StaffInfo, error) {
	if _, err := uc.staffRepo.GetByUser(ctx, orgID, userID); err != nil {
		return dto.StaffInfo{}, err
	}

	if req.DepartmentID != nil {
		if _, err := uc.departmentRepo.GetByID(ctx, orgID, *req.DepartmentID); err != nil {
			return dto.StaffInfo{}, err
		}
	}

	if err := uc.staffRepo.SetDepartment(ctx, orgID, userID, req.DepartmentID); err != nil {
		return dto.StaffInfo{}, err
	}

	// Re-read rather than patching the entry in memory: the response then
	// carries the department *name* from the same source the roster listing
	// uses, so a client that renders this reply cannot end up showing a
	// different label than a refresh would.
	updated, err := uc.staffRepo.GetByUser(ctx, orgID, userID)
	if err != nil {
		return dto.StaffInfo{}, domainErr.New(domainErr.ErrInternal,
			"assignment saved but could not be read back", err)
	}
	return toStaffInfo(updated), nil
}
