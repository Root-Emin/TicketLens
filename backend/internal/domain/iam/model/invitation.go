package model

import (
	"time"

	"github.com/google/uuid"
)

// InvitationStatus is the state an invitation presents to an administrator.
//
// Derived rather than stored: the timestamps on the row are the facts, and a
// status column would be a second copy of them that can disagree — an expired
// invitation would need a background job to relabel it, and would sit there
// claiming to be pending until that job ran.
type InvitationStatus string

const (
	InvitationPending  InvitationStatus = "pending"
	InvitationAccepted InvitationStatus = "accepted"
	InvitationRevoked  InvitationStatus = "revoked"
	InvitationExpired  InvitationStatus = "expired"
)

// Invitation is a pending offer of staff membership in one organization.
//
// It records what the person becomes when they accept — the role, and where
// they sit — so that acceptance needs no further decision from an administrator.
// It is not a membership record: membership is user_roles.
type Invitation struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	Email          string     `json:"email"`
	RoleID         uuid.UUID  `json:"role_id"`
	DepartmentID   *uuid.UUID `json:"department_id,omitempty"`

	// TokenHash is the SHA-256 of the token that was mailed out. The token
	// itself is never stored, so it exists only in the creation response and in
	// the recipient's inbox.
	TokenHash string `json:"-"`

	InvitedBy  *uuid.UUID `json:"invited_by,omitempty"`
	ExpiresAt  time.Time  `json:"expires_at"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// Status reports the invitation's state at the given moment.
//
// Order matters: an invitation that was accepted and then ran past its expiry is
// accepted, not expired. What already happened outranks what merely lapsed.
func (i *Invitation) Status(now time.Time) InvitationStatus {
	switch {
	case i.AcceptedAt != nil:
		return InvitationAccepted
	case i.RevokedAt != nil:
		return InvitationRevoked
	case !now.Before(i.ExpiresAt):
		return InvitationExpired
	default:
		return InvitationPending
	}
}

// IsRedeemable reports whether this invitation can still be accepted.
//
// Callers must not branch on which of the failing conditions applied when
// answering an unauthenticated caller: telling somebody that a token is expired
// rather than unknown confirms it was once real, which is enough to probe for
// live invitations.
func (i *Invitation) IsRedeemable(now time.Time) bool {
	return i.Status(now) == InvitationPending
}
