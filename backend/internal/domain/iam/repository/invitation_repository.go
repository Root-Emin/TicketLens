package repository

import (
	"context"

	"github.com/Root-Emin/TicketLens/internal/domain/iam/model"
	"github.com/google/uuid"
)

// InvitationRepository defines the interface for staff invitation persistence.
type InvitationRepository interface {
	Create(ctx context.Context, inv *model.Invitation) error

	// GetByTokenHash resolves an invitation from the hash of the token its
	// recipient holds. It is the only lookup available to an unauthenticated
	// caller, so it takes no organization: the token itself names the tenant.
	//
	// Returns errors.ErrNotFound for an unknown hash. Callers must give the same
	// answer for that as for an invitation that is merely expired or spent —
	// distinguishing them tells a prober which tokens were once real.
	GetByTokenHash(ctx context.Context, tokenHash string) (*model.Invitation, error)

	// GetByID reads one invitation within an organization. The organization is a
	// parameter rather than something the caller filters afterwards so a
	// mismatched pair cannot return a row belonging to another tenant.
	GetByID(ctx context.Context, orgID, id uuid.UUID) (*model.Invitation, error)

	// ListByOrg returns one page of invitations, newest first. Includes spent and
	// expired ones: an administrator asking who was invited wants the history,
	// and filtering is a presentation decision.
	ListByOrg(ctx context.Context, orgID uuid.UUID, offset, limit int) ([]*model.Invitation, int, error)

	// FindLiveByEmail returns the outstanding invitation for an address in this
	// organization, if any. Used to turn the partial unique index into a clear
	// error instead of a constraint violation.
	//
	// "Live" means neither accepted nor revoked; it may still be expired, since
	// the index cannot express time. Returns errors.ErrNotFound when there is none.
	FindLiveByEmail(ctx context.Context, orgID uuid.UUID, email string) (*model.Invitation, error)

	// MarkAccepted spends an invitation. Implementations must only succeed when
	// the row is still unaccepted and unrevoked, so that two acceptances racing
	// on the same token cannot both create an account.
	MarkAccepted(ctx context.Context, id uuid.UUID) error

	// Revoke withdraws a pending invitation, scoped to its organization.
	Revoke(ctx context.Context, orgID, id uuid.UUID) error
}
