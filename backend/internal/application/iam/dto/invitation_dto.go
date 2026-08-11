package dto

import (
	"time"

	"github.com/google/uuid"
)

// CreateInvitationRequest is the input for POST /invitations.
//
// No organization_id: the tenant comes from the caller's token, as it does for
// every other resource in this API. A body field would let a caller holding
// user:write in one organization invite staff into another.
type CreateInvitationRequest struct {
	Email  string    `json:"email" validate:"required,email"`
	RoleID uuid.UUID `json:"role_id" validate:"required"`
	// Optional. When set, the invitee lands on that team the moment they accept
	// instead of sitting in the roster's Unassigned bucket.
	DepartmentID *uuid.UUID `json:"department_id,omitempty"`
}

// AcceptInvitationRequest is the input for accepting an invitation.
//
// Deliberately not an email: the address is fixed by the invitation, and taking
// one here would let the holder of a token open an account under any address in
// the inviting organization.
//
// All three fields are optional at this layer and required by the use case only
// when an account has to be created. An invitee whose address already has a
// TicketLens account is joined to the organization rather than duplicated, and
// their existing password is left alone — so asking them for a name and a
// password would be asking for values that are discarded. The use case knows
// which branch it is in and enforces the rule there; a `required` tag here would
// enforce it in the wrong place, for both.
type AcceptInvitationRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Password  string `json:"password" validate:"omitempty,min=8"`
}

// RoleInfo is one assignable role.
//
// No permissions: the screens that read this are choosing a role by name, and
// shipping the grant list would invite a client to make an access decision from
// a cached copy of it.
type RoleInfo struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

// InvitationResponse is one invitation as an administrator sees it.
type InvitationResponse struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	RoleID       uuid.UUID  `json:"role_id"`
	RoleName     string     `json:"role_name"`
	DepartmentID *uuid.UUID `json:"department_id,omitempty"`
	Status       string     `json:"status"`
	ExpiresAt    time.Time  `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`

	// InvitedBy is who sent it, for the administrator's list. A display name
	// rather than an id: the id would need a second lookup to render, and the
	// only question the column answers is "who did this".
	//
	// Empty when the inviter's account has since been deleted (invited_by is
	// ON DELETE SET NULL), which the UI renders as a dash rather than hiding
	// the row.
	InvitedBy string `json:"invited_by,omitempty"`

	// AcceptURL is populated only in the response to creating an invitation,
	// because that is the only moment the raw token exists — it is stored hashed
	// and cannot be recovered afterwards. Returning it means an administrator
	// always has a link to pass on by hand, so onboarding does not depend on
	// mail being delivered.
	AcceptURL string `json:"accept_url,omitempty"`
}

// InvitationPreview is what the acceptance screen may show before anyone signs
// in. It answers "what am I being asked to join?" and nothing more: no ids, no
// inviter, nothing about the organization's other members. Whoever holds the
// token is unauthenticated and may not be the intended recipient.
type InvitationPreview struct {
	Email            string    `json:"email"`
	OrganizationName string    `json:"organization_name"`
	RoleName         string    `json:"role_name"`
	ExpiresAt        time.Time `json:"expires_at"`

	// HasAccount says the address already has a TicketLens login, so accepting
	// joins that account instead of creating one and the screen must not ask for
	// a password — it would be discarded, and the person would then be unable to
	// sign in with what they just typed.
	//
	// This discloses only whether the address the invitation already names has an
	// account. The holder of the token was told that address by the invitation
	// itself, so it is not an oracle for arbitrary addresses.
	HasAccount bool `json:"has_account"`
}

// There is no list envelope type here on purpose: the IAM handlers page with
// shared/pagination (FromRequest + NewResult), as GET /users already does, so
// the list usecase returns the rows and the total and the handler wraps them the
// same way every other IAM list does.
