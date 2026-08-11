package repository

import (
	"context"

	"github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	"github.com/google/uuid"
)

// StaffFilter narrows a roster listing.
type StaffFilter struct {
	// DepartmentID limits the list to one team. Ignored when nil.
	DepartmentID *uuid.UUID
	// Unassigned limits the list to people on no team. Takes precedence over
	// DepartmentID, mirroring how ?assignee_id=unassigned beats a UUID on the
	// ticket queue.
	Unassigned bool
	// Query matches name or email, case-insensitively. Empty matches all.
	Query string
}

// StaffRepository reads and writes the support roster.
//
// Every method takes an organization ID. The roster is the one place in the
// product where a leak would hand somebody a complete list of another tenant's
// employees and their email addresses, so there is deliberately no unscoped
// variant to reach for by accident — contrast UserRepository.List, which spans
// the platform and needs a comment warning callers off it.
type StaffRepository interface {
	// ListByOrg returns the organization's support staff, newest account first.
	//
	// "Support staff" excludes portal customers. Both are users holding a role
	// in the organization and GET /users cannot tell them apart, but a customer
	// with a login has a customers row pointing at their account (migration
	// 00020) and a colleague does not — so the exclusion is a fact of the
	// database rather than a guess about email addresses.
	ListByOrg(ctx context.Context, orgID uuid.UUID, filter StaffFilter, offset, limit int) ([]*model.StaffMember, int, error)

	// GetByUser returns one roster entry, or ErrNotFound when the user is not
	// staff in this organization — which includes the case where they are one of
	// its customers.
	GetByUser(ctx context.Context, orgID, userID uuid.UUID) (*model.StaffMember, error)

	// SetDepartment places a person on a team, or removes them from one when
	// departmentID is nil. Idempotent: assigning somebody to the team they are
	// already on succeeds and touches only updated_at.
	SetDepartment(ctx context.Context, orgID, userID uuid.UUID, departmentID *uuid.UUID) error

	// CountByDepartment returns how many staff each department holds, keyed by
	// department ID. Departments with nobody on them are absent rather than
	// present with a zero, so a caller merging this into a department list must
	// default the missing ones.
	CountByDepartment(ctx context.Context, orgID uuid.UUID) (map[uuid.UUID]int, error)
}
