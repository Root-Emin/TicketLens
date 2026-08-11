package usecase

import (
	"context"

	"github.com/Root-Emin/TicketLens/internal/domain/iam/repository"
	"github.com/google/uuid"
)

// RevokeInvitationUseCase withdraws a pending invitation.
type RevokeInvitationUseCase struct {
	invitations repository.InvitationRepository
}

// NewRevokeInvitationUseCase creates a new RevokeInvitationUseCase.
func NewRevokeInvitationUseCase(invitations repository.InvitationRepository) *RevokeInvitationUseCase {
	return &RevokeInvitationUseCase{invitations: invitations}
}

// Execute revokes an invitation belonging to the caller's organization.
//
// orgID is part of the update rather than a check performed beforehand, so an id
// from another tenant matches no row and answers not-found — never 403, which
// would confirm the invitation exists somewhere.
//
// Revoking is also the way to re-invite an address: one live invitation per
// address is enforced by a partial unique index, so an outstanding one has to be
// withdrawn before a fresh link can be issued.
//
// Already-accepted invitations are refused rather than silently succeeding.
// Revoking one would read as withdrawing the membership, which it does not do —
// the person keeps their role until it is removed directly.
func (uc *RevokeInvitationUseCase) Execute(ctx context.Context, orgID, invitationID uuid.UUID) error {
	return uc.invitations.Revoke(ctx, orgID, invitationID)
}
