package iam

import (
	"context"
	"errors"
	"time"

	"github.com/Root-Emin/TicketLens/internal/domain/iam/model"
	"github.com/Root-Emin/TicketLens/internal/domain/iam/repository"
	"github.com/Root-Emin/TicketLens/internal/shared/database"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// invitationColumns is the full row, in the order every scan below expects.
const invitationColumns = `id, organization_id, email, role_id, department_id,
	token_hash, invited_by, expires_at, accepted_at, revoked_at, created_at`

// InvitationRepo implements repository.InvitationRepository with PostgreSQL.
type InvitationRepo struct {
	db *pgxpool.Pool
}

// Verify interface compliance at compile time.
var _ repository.InvitationRepository = (*InvitationRepo)(nil)

// NewInvitationRepo creates a new InvitationRepo.
func NewInvitationRepo(db *pgxpool.Pool) *InvitationRepo {
	return &InvitationRepo{db: db}
}

func (r *InvitationRepo) Create(ctx context.Context, inv *model.Invitation) error {
	if inv.ID == uuid.Nil {
		inv.ID = uuid.New()
	}
	inv.CreatedAt = time.Now().UTC()

	_, err := database.Querier(ctx, r.db).Exec(ctx,
		`INSERT INTO invitations (id, organization_id, email, role_id, department_id,
			token_hash, invited_by, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		inv.ID, inv.OrganizationID, inv.Email, inv.RoleID, inv.DepartmentID,
		inv.TokenHash, inv.InvitedBy, inv.ExpiresAt, inv.CreatedAt,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to create invitation", err)
	}
	return nil
}

func (r *InvitationRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*model.Invitation, error) {
	row := database.Querier(ctx, r.db).QueryRow(ctx,
		`SELECT `+invitationColumns+` FROM invitations WHERE token_hash=$1`, tokenHash,
	)
	return scanInvitation(row, "invitation not found", "failed to get invitation")
}

func (r *InvitationRepo) GetByID(ctx context.Context, orgID, id uuid.UUID) (*model.Invitation, error) {
	row := database.Querier(ctx, r.db).QueryRow(ctx,
		`SELECT `+invitationColumns+` FROM invitations WHERE id=$1 AND organization_id=$2`, id, orgID,
	)
	return scanInvitation(row, "invitation not found", "failed to get invitation")
}

func (r *InvitationRepo) FindLiveByEmail(ctx context.Context, orgID uuid.UUID, email string) (*model.Invitation, error) {
	// Matches the partial unique index: not accepted, not revoked. Expiry is
	// left out here too, so an expired row is still found and still blocks a
	// duplicate — the caller decides what to do about it.
	row := database.Querier(ctx, r.db).QueryRow(ctx,
		`SELECT `+invitationColumns+` FROM invitations
		 WHERE organization_id=$1 AND email=$2
		   AND accepted_at IS NULL AND revoked_at IS NULL`, orgID, email,
	)
	return scanInvitation(row, "no live invitation", "failed to find invitation")
}

func (r *InvitationRepo) ListByOrg(ctx context.Context, orgID uuid.UUID, offset, limit int) ([]*model.Invitation, int, error) {
	var total int
	if err := database.Querier(ctx, r.db).QueryRow(ctx,
		`SELECT COUNT(*) FROM invitations WHERE organization_id=$1`, orgID,
	).Scan(&total); err != nil {
		return nil, 0, domainErr.New(domainErr.ErrInternal, "failed to count invitations", err)
	}

	rows, err := database.Querier(ctx, r.db).Query(ctx,
		`SELECT `+invitationColumns+` FROM invitations
		 WHERE organization_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		orgID, limit, offset,
	)
	if err != nil {
		return nil, 0, domainErr.New(domainErr.ErrInternal, "failed to list invitations", err)
	}
	defer rows.Close()

	var out []*model.Invitation
	for rows.Next() {
		inv, err := scanInvitation(rows, "invitation not found", "failed to scan invitation")
		if err != nil {
			return nil, 0, err
		}
		out = append(out, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, domainErr.New(domainErr.ErrInternal, "failed to list invitations", err)
	}
	return out, total, nil
}

func (r *InvitationRepo) MarkAccepted(ctx context.Context, id uuid.UUID) error {
	// The WHERE clause carries the guard, not a preceding read: two acceptances
	// arriving together would both pass a check-then-write, and the loser here
	// updates no rows and is told so. That makes the token single-use even under
	// a race, which is the whole point of spending it.
	tag, err := database.Querier(ctx, r.db).Exec(ctx,
		`UPDATE invitations SET accepted_at=NOW()
		 WHERE id=$1 AND accepted_at IS NULL AND revoked_at IS NULL`, id,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to accept invitation", err)
	}
	if tag.RowsAffected() == 0 {
		return domainErr.New(domainErr.ErrNotFound, "invitation not found", nil)
	}
	return nil
}

func (r *InvitationRepo) Revoke(ctx context.Context, orgID, id uuid.UUID) error {
	// Revoking an already-accepted invitation is refused rather than ignored:
	// it would suggest the membership had been withdrawn, which it has not.
	// Removing the person is a separate action on their role.
	tag, err := database.Querier(ctx, r.db).Exec(ctx,
		`UPDATE invitations SET revoked_at=NOW()
		 WHERE id=$1 AND organization_id=$2 AND accepted_at IS NULL AND revoked_at IS NULL`,
		id, orgID,
	)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to revoke invitation", err)
	}
	if tag.RowsAffected() == 0 {
		return domainErr.New(domainErr.ErrNotFound, "invitation not found", nil)
	}
	return nil
}

// scanRow is satisfied by both pgx.Row and pgx.Rows, so the column list and the
// scan order are written once for single reads and for list iteration alike.
type scanRow interface {
	Scan(dest ...any) error
}

func scanInvitation(row scanRow, notFoundMsg, failMsg string) (*model.Invitation, error) {
	var inv model.Invitation
	err := row.Scan(
		&inv.ID, &inv.OrganizationID, &inv.Email, &inv.RoleID, &inv.DepartmentID,
		&inv.TokenHash, &inv.InvitedBy, &inv.ExpiresAt, &inv.AcceptedAt, &inv.RevokedAt, &inv.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErr.New(domainErr.ErrNotFound, notFoundMsg, nil)
		}
		return nil, domainErr.New(domainErr.ErrInternal, failMsg, err)
	}
	return &inv, nil
}
