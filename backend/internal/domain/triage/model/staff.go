package model

import (
	"time"

	"github.com/google/uuid"
)

/*
	The support roster.

	StaffMember is a read model, not an aggregate: it is one row of "who works
	here and on which team", assembled from a user account, that account's
	membership of an organization, and the staff_departments row that places
	them (migration 00021). Nothing mutates a StaffMember — the only writable
	fact on it is the department, and that has its own repository verb.

	Kept in triage rather than iam because a "staff member" is a triage concept.
	iam knows about users, roles and organizations; it has no opinion about
	support teams, and giving dto.UserInfo a department would push a department
	into /me and every other IAM response that has no business carrying one.
*/

// StaffMember is one person on an organization's support roster.
type StaffMember struct {
	UserID         uuid.UUID
	OrganizationID uuid.UUID

	Email     string
	FirstName string
	LastName  string
	// Status is the account status from users.status — whether they can sign
	// in. Not a presence or an availability: nothing in the schema records
	// either.
	Status string

	// DepartmentID is nil when the person is on the roster but on no team.
	// That is a normal state, not a broken row: a newly granted account has no
	// assignment, and deleting a department returns its people to it.
	DepartmentID   *uuid.UUID
	DepartmentName string

	// CreatedAt is the account's creation time, which is the closest thing the
	// schema has to a joining date.
	CreatedAt time.Time
}

// FullName renders the display name, falling back to the email — which every
// account has, while first and last name are nullable.
func (s *StaffMember) FullName() string {
	switch {
	case s.FirstName != "" && s.LastName != "":
		return s.FirstName + " " + s.LastName
	case s.FirstName != "":
		return s.FirstName
	case s.LastName != "":
		return s.LastName
	default:
		return s.Email
	}
}

// Assigned reports whether this person is placed on a team.
func (s *StaffMember) Assigned() bool {
	return s.DepartmentID != nil
}
