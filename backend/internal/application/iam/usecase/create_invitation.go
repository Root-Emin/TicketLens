package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/Root-Emin/TicketLens/internal/application/iam/dto"
	"github.com/Root-Emin/TicketLens/internal/domain/iam/model"
	"github.com/Root-Emin/TicketLens/internal/domain/iam/port"
	"github.com/Root-Emin/TicketLens/internal/domain/iam/repository"
	tenantRepo "github.com/Root-Emin/TicketLens/internal/domain/tenant/repository"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
	"github.com/Root-Emin/TicketLens/internal/shared/middleware"
	"github.com/google/uuid"
)

// tokenPrefix marks an invitation token in a log or a support conversation, the
// way mf_ marks an API key. It is part of the token: the hash covers it too.
const tokenPrefix = "inv_"

// CreateInvitationUseCase invites somebody to join an organization as staff.
type CreateInvitationUseCase struct {
	invitations repository.InvitationRepository
	users       repository.UserRepository
	roles       repository.RoleRepository
	orgs        tenantRepo.OrgRepository
	mailer      port.Mailer
	log         *slog.Logger

	appURL string
	ttl    time.Duration
}

// NewCreateInvitationUseCase creates a new CreateInvitationUseCase.
func NewCreateInvitationUseCase(
	invitations repository.InvitationRepository,
	users repository.UserRepository,
	roles repository.RoleRepository,
	orgs tenantRepo.OrgRepository,
	mailer port.Mailer,
	log *slog.Logger,
	appURL string,
	ttl time.Duration,
) *CreateInvitationUseCase {
	return &CreateInvitationUseCase{
		invitations: invitations,
		users:       users,
		roles:       roles,
		orgs:        orgs,
		mailer:      mailer,
		log:         log,
		appURL:      strings.TrimRight(appURL, "/"),
		ttl:         ttl,
	}
}

// Execute creates an invitation and mails the link to the invitee.
//
// orgID comes from the caller's token, never from the request: the invitation
// grants a role inside that organization, so letting a body choose the tenant
// would let anyone holding user:write in one organization staff another.
func (uc *CreateInvitationUseCase) Execute(ctx context.Context, orgID uuid.UUID, req dto.CreateInvitationRequest) (*dto.InvitationResponse, error) {
	// Addresses are compared, stored and mailed in one normalised form.
	// Otherwise "Ada@x.com" and "ada@x.com" are two different live invitations
	// as far as the unique index is concerned, and the second one silently
	// duplicates the first.
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// The role must belong to this organization. Roles are per-organization
	// clones (see tenant CreateOrgUseCase.provisionRoles), so a role id from
	// another tenant is a well-formed value that would otherwise be accepted and
	// grant membership there. Answering not-found rather than forbidden keeps
	// this from confirming that the id exists somewhere.
	role, err := uc.roles.GetByID(ctx, req.RoleID)
	if err != nil {
		return nil, err
	}
	if role.ScopeType != model.ScopeTypeOrganization || role.ScopeID != orgID {
		return nil, domainErr.New(domainErr.ErrNotFound, "role not found", nil)
	}

	// Somebody who already has an account and a role here does not need an
	// invitation, and sending one would imply they are not yet a member.
	if existing, err := uc.users.GetByEmail(ctx, email); err == nil && existing != nil {
		names, err := uc.roles.GetUserRoleNames(ctx, existing.ID, orgID)
		if err != nil {
			return nil, err
		}
		if len(names) > 0 {
			return nil, domainErr.New(domainErr.ErrAlreadyExists,
				"this person is already a member of the organization", nil)
		}
	} else if err != nil && !errors.Is(err, domainErr.ErrNotFound) {
		// A lookup failure must not be read as "no such user": that would let a
		// transient fault turn into a second invitation for a current member.
		return nil, err
	}

	// One live invitation per address. The partial unique index enforces this;
	// checking first turns a constraint violation into an answer the caller can
	// act on, and names the fix.
	if live, err := uc.invitations.FindLiveByEmail(ctx, orgID, email); err == nil && live != nil {
		return nil, domainErr.New(domainErr.ErrAlreadyExists,
			"an invitation for this address is already outstanding; revoke it to send a new one", nil)
	} else if err != nil && !errors.Is(err, domainErr.ErrNotFound) {
		return nil, err
	}

	token, err := generateInvitationToken()
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to generate invitation token", err)
	}

	invitedBy, _ := middleware.UserIDFromContext(ctx)
	inv := &model.Invitation{
		OrganizationID: orgID,
		Email:          email,
		RoleID:         req.RoleID,
		DepartmentID:   req.DepartmentID,
		TokenHash:      HashInvitationToken(token),
		ExpiresAt:      time.Now().UTC().Add(uc.ttl),
	}
	if invitedBy != uuid.Nil {
		inv.InvitedBy = &invitedBy
	}

	if err := uc.invitations.Create(ctx, inv); err != nil {
		return nil, err
	}

	acceptURL := uc.acceptURL(token)
	uc.sendInvitationEmail(ctx, orgID, inv, role.Name, acceptURL)

	return &dto.InvitationResponse{
		ID:           inv.ID,
		Email:        inv.Email,
		RoleID:       inv.RoleID,
		RoleName:     role.Name,
		DepartmentID: inv.DepartmentID,
		Status:       string(inv.Status(time.Now().UTC())),
		ExpiresAt:    inv.ExpiresAt,
		CreatedAt:    inv.CreatedAt,
		AcceptURL:    acceptURL,
	}, nil
}

// sendInvitationEmail delivers the invitation, and does not fail the operation
// if it cannot.
//
// The invitation already exists at this point, and destroying it because a relay
// was unreachable would be worse than the missed email: the administrator holds
// the same link in the response and can pass it on directly. The failure is
// logged so a systematically broken relay is visible rather than merely quiet.
func (uc *CreateInvitationUseCase) sendInvitationEmail(ctx context.Context, orgID uuid.UUID, inv *model.Invitation, roleName, acceptURL string) {
	orgName := "your team"
	if org, err := uc.orgs.GetByID(ctx, orgID); err == nil && org != nil {
		orgName = org.Name
	}

	body := fmt.Sprintf(`You have been invited to join %s on TicketLens as %s.

Open this link to choose a password and activate your account:

%s

The link works once and expires on %s.

If you were not expecting this invitation you can ignore this message — the
account is not created until the link is opened.`,
		orgName, roleName, acceptURL, inv.ExpiresAt.Format("2 January 2006"))

	if err := uc.mailer.Send(ctx, port.Message{
		To:      inv.Email,
		Subject: fmt.Sprintf("You have been invited to %s on TicketLens", orgName),
		Body:    body,
	}); err != nil {
		uc.log.Error("invitation email could not be sent; the link is still valid",
			"invitation_id", inv.ID, "organization_id", orgID, "error", err)
	}
}

// acceptURL builds the link the invitee opens. It points at the frontend, not at
// this API: the recipient needs a screen to choose a password on, and the page
// posts the token back to /auth/invitations/{token}/accept itself.
func (uc *CreateInvitationUseCase) acceptURL(token string) string {
	return fmt.Sprintf("%s/invite/%s", uc.appURL, url.PathEscape(token))
}

// generateInvitationToken returns a fresh token with 256 bits of entropy. Same
// construction as the API keys in tenant/usecase/manage_api_keys.go — this value
// is a bearer credential for one account creation and has to be unguessable.
func generateInvitationToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return tokenPrefix + hex.EncodeToString(b), nil
}

// HashInvitationToken is what the database stores. Exported so the acceptance
// path hashes the presented token exactly the same way; a second implementation
// that disagreed would make every invitation unredeemable.
//
// Plain SHA-256, no salt or work factor, and that is correct here: the input is
// 256 random bits, so there is no dictionary to slow down.
func HashInvitationToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
