package usecase

import (
	"context"
	"time"

	"github.com/Root-Emin/TicketLens/internal/application/iam/dto"
	"github.com/Root-Emin/TicketLens/internal/domain/iam/repository"
	"github.com/google/uuid"
)

// ListInvitationsUseCase reads an organization's invitations.
type ListInvitationsUseCase struct {
	invitations repository.InvitationRepository
	roles       repository.RoleRepository
	users       repository.UserRepository
}

// NewListInvitationsUseCase creates a new ListInvitationsUseCase.
func NewListInvitationsUseCase(
	invitations repository.InvitationRepository,
	roles repository.RoleRepository,
	users repository.UserRepository,
) *ListInvitationsUseCase {
	return &ListInvitationsUseCase{invitations: invitations, roles: roles, users: users}
}

// Execute returns one page of invitations, newest first, with the total.
//
// Spent and expired invitations are included: an administrator asking who was
// invited wants the history, and hiding it would make a re-invitation that the
// unique index refuses look inexplicable.
//
// No AcceptURL is ever set here. Only the hash is stored, so the link genuinely
// cannot be reproduced after creation — a list that could hand out working
// tokens would make every invitation replayable by anyone with user:read.
func (uc *ListInvitationsUseCase) Execute(ctx context.Context, orgID uuid.UUID, offset, limit int) ([]dto.InvitationResponse, int, error) {
	invitations, total, err := uc.invitations.ListByOrg(ctx, orgID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	// Roles and inviters both repeat heavily across a page — most invitations
	// are for "agent", and most were sent by the same administrator — so each is
	// resolved once and reused rather than looked up per row.
	roleNames := make(map[uuid.UUID]string)
	inviterNames := make(map[uuid.UUID]string)
	now := time.Now().UTC()

	out := make([]dto.InvitationResponse, 0, len(invitations))
	for _, inv := range invitations {
		name, ok := roleNames[inv.RoleID]
		if !ok {
			// A role deleted out from under a spent invitation should not blank
			// the whole list; the row is still worth showing.
			if role, err := uc.roles.GetByID(ctx, inv.RoleID); err == nil && role != nil {
				name = role.Name
			}
			roleNames[inv.RoleID] = name
		}

		out = append(out, dto.InvitationResponse{
			ID:           inv.ID,
			Email:        inv.Email,
			RoleID:       inv.RoleID,
			RoleName:     name,
			DepartmentID: inv.DepartmentID,
			Status:       string(inv.Status(now)),
			ExpiresAt:    inv.ExpiresAt,
			CreatedAt:    inv.CreatedAt,
			InvitedBy:    uc.inviterName(ctx, inv.InvitedBy, inviterNames),
		})
	}

	return out, total, nil
}

// inviterName resolves who sent an invitation, memoised across the page.
//
// Returns "" both for an invitation with no inviter recorded and for one whose
// inviter has since been deleted — invited_by is ON DELETE SET NULL, so the two
// are the same row state and the list renders a dash for either.
func (uc *ListInvitationsUseCase) inviterName(ctx context.Context, id *uuid.UUID, cache map[uuid.UUID]string) string {
	if id == nil {
		return ""
	}
	if name, ok := cache[*id]; ok {
		return name
	}

	name := ""
	if user, err := uc.users.GetByID(ctx, *id); err == nil && user != nil {
		name = user.FullName()
	}
	cache[*id] = name
	return name
}
