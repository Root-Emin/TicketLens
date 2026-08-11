package dto

import (
	"time"

	"github.com/google/uuid"
)

// StaffInfo is the public representation of one person on the support roster.
//
// Deliberately not iam's UserInfo with a field bolted on. UserInfo is what /me
// and GET /users return, and neither has any business carrying a department —
// a customer signing into the portal would be handed a field about support
// teams. This is the triage context's own view of a person, and it exists
// because triage is the context that has an opinion about teams.
type StaffInfo struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	// FullName is precomputed rather than left to the client, which would
	// otherwise each reimplement the fall back to the email address.
	FullName string `json:"full_name"`
	Status   string `json:"status"`
	// Department is null when the person is on the roster but on no team.
	//
	// The same DepartmentRef a ticket carries (ticket_dto.go), so a client can
	// compare a person's team against a ticket's department without mapping
	// between two shapes that mean the same thing.
	Department *DepartmentRef `json:"department"`
	CreatedAt  time.Time      `json:"created_at"`
}

// AssignStaffDepartmentRequest is the input for
// PUT /staff/{userId}/department.
//
// DepartmentID is a nullable pointer, and null is a meaningful value rather
// than an omission: it is how somebody is taken off a team. A PUT is used
// instead of a PATCH for exactly that reason — the request replaces the
// assignment wholesale, so "no department" can be stated rather than implied by
// leaving a field out.
type AssignStaffDepartmentRequest struct {
	DepartmentID *uuid.UUID `json:"department_id"`
}
