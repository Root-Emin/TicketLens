package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/Root-Emin/TicketLens/internal/application/iam/dto"
	iamEvent "github.com/Root-Emin/TicketLens/internal/domain/iam/event"
	"github.com/Root-Emin/TicketLens/internal/domain/iam/model"
	"github.com/Root-Emin/TicketLens/internal/domain/iam/repository"
	"github.com/Root-Emin/TicketLens/internal/domain/iam/service"
	tenantRepo "github.com/Root-Emin/TicketLens/internal/domain/tenant/repository"
	triageRepo "github.com/Root-Emin/TicketLens/internal/domain/triage/repository"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
	"github.com/Root-Emin/TicketLens/internal/shared/events"
)

// TxManager runs a unit of work inside one database transaction.
//
// Declared here rather than imported from the triage context: this is the
// consumer, the contract is one method wide, and importing triage's copy would
// tie account creation to the ticketing context for no reason. The postgres
// implementation in shared/database satisfies both.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// AcceptInvitationUseCase turns a valid invitation token into staff membership.
type AcceptInvitationUseCase struct {
	invitations repository.InvitationRepository
	users       repository.UserRepository
	roles       repository.RoleRepository
	orgs        tenantRepo.OrgRepository
	staff       triageRepo.StaffRepository
	auth        service.AuthService
	rbac        service.RBACService
	tx          TxManager
	eventBus    events.EventBus
}

// NewAcceptInvitationUseCase creates a new AcceptInvitationUseCase.
func NewAcceptInvitationUseCase(
	invitations repository.InvitationRepository,
	users repository.UserRepository,
	roles repository.RoleRepository,
	orgs tenantRepo.OrgRepository,
	staff triageRepo.StaffRepository,
	auth service.AuthService,
	rbac service.RBACService,
	tx TxManager,
	eventBus events.EventBus,
) *AcceptInvitationUseCase {
	return &AcceptInvitationUseCase{
		invitations: invitations,
		users:       users,
		roles:       roles,
		orgs:        orgs,
		staff:       staff,
		auth:        auth,
		rbac:        rbac,
		tx:          tx,
		eventBus:    eventBus,
	}
}

// errInvalidToken is the single answer every failed redemption gives.
//
// One error for "no such token", "already used" and "expired" on purpose. The
// caller is unauthenticated and holds only a string; telling them which of those
// applies distinguishes a token that was never real from one that was, which is
// exactly what somebody probing for live invitations wants to learn.
func errInvalidToken() error {
	return domainErr.New(domainErr.ErrNotFound, "this invitation link is not valid", nil)
}

// Preview reports what an invitation offers, without accepting it.
//
// Public: whoever holds the token may not be the intended recipient, so this
// returns only what the acceptance screen has to render — never ids, the
// inviter, or anything about the organization's other members.
func (uc *AcceptInvitationUseCase) Preview(ctx context.Context, token string) (*dto.InvitationPreview, error) {
	inv, err := uc.resolve(ctx, token)
	if err != nil {
		return nil, err
	}

	role, err := uc.roles.GetByID(ctx, inv.RoleID)
	if err != nil {
		return nil, err
	}

	orgName := ""
	if org, err := uc.orgs.GetByID(ctx, inv.OrganizationID); err == nil && org != nil {
		orgName = org.Name
	}

	// A lookup failure is reported as "no account", which is the safe way to be
	// wrong: the screen then asks for a password, and Execute ignores it if an
	// account does turn out to exist. The reverse would leave somebody with no
	// account and no way to set one.
	hasAccount := false
	if existing, err := uc.users.GetByEmail(ctx, inv.Email); err == nil && existing != nil {
		hasAccount = true
	}

	return &dto.InvitationPreview{
		Email:            inv.Email,
		OrganizationName: orgName,
		RoleName:         role.Name,
		ExpiresAt:        inv.ExpiresAt,
		HasAccount:       hasAccount,
	}, nil
}

// Execute redeems an invitation: it creates or joins the account, grants the
// role, places the person on their team, and spends the invitation — all or
// nothing.
func (uc *AcceptInvitationUseCase) Execute(ctx context.Context, token string, req dto.AcceptInvitationRequest) (*dto.UserInfo, error) {
	inv, err := uc.resolve(ctx, token)
	if err != nil {
		return nil, err
	}

	var user *model.User

	// One transaction over every write. A partial acceptance is the failure mode
	// worth engineering against: a user row with no role is an account that can
	// sign in, belongs to no organization, and cannot be invited again because
	// the address is taken — the same defect provisionRoles documents in the
	// tenant context.
	err = uc.tx.WithinTx(ctx, func(txCtx context.Context) error {
		var err error
		user, err = uc.resolveAccount(txCtx, inv, req)
		if err != nil {
			return err
		}

		if err := uc.roles.AssignRoleToUser(txCtx, &model.UserRole{
			UserID:         user.ID,
			RoleID:         inv.RoleID,
			OrganizationID: inv.OrganizationID,
		}); err != nil {
			return err
		}

		// Optional: no department on the invitation means the person joins the
		// roster unassigned, which staff_departments models as the absence of a
		// row. Nothing to write in that case.
		if inv.DepartmentID != nil {
			if err := uc.staff.SetDepartment(txCtx, inv.OrganizationID, user.ID, inv.DepartmentID); err != nil {
				return err
			}
		}

		// Last, and guarded by its own WHERE clause: two acceptances racing on
		// one token both reach here, and only the first updates a row. The loser
		// gets not-found and its whole transaction rolls back, so exactly one
		// account is created.
		if err := uc.invitations.MarkAccepted(txCtx, inv.ID); err != nil {
			if errors.Is(err, domainErr.ErrNotFound) {
				return errInvalidToken()
			}
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Outside the transaction: neither is part of the atomic unit, and a Redis
	// or broker hiccup must not undo a completed acceptance. The cache entry
	// matters for an existing account joining a second organization — a fresh
	// user has nothing cached.
	_ = uc.rbac.InvalidateCache(ctx, user.ID, inv.OrganizationID)
	_ = uc.eventBus.Publish(ctx, events.TopicIAM, iamEvent.RoleAssigned{
		UserID:         user.ID,
		RoleID:         inv.RoleID,
		OrganizationID: inv.OrganizationID,
		Timestamp:      time.Now().UTC(),
	})

	return &dto.UserInfo{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Status:    string(user.Status),
		CreatedAt: user.CreatedAt,
	}, nil
}

// resolveAccount returns the account the invitation attaches the role to,
// creating it if the address is new to the platform.
//
// An existing account is reused rather than duplicated, and its password is left
// alone. Users are global while roles are per-organization, so somebody already
// working in one tenant who is invited to a second must end up as one person
// with two memberships. Setting the password from this request instead would
// hand whoever holds the invitation link a way to overwrite the credentials of
// an existing account by inviting its address.
func (uc *AcceptInvitationUseCase) resolveAccount(ctx context.Context, inv *model.Invitation, req dto.AcceptInvitationRequest) (*model.User, error) {
	existing, err := uc.users.GetByEmail(ctx, inv.Email)
	if err != nil && !errors.Is(err, domainErr.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		if !existing.IsActive() {
			return nil, domainErr.New(domainErr.ErrForbidden, "account is not active", nil)
		}
		return existing, nil
	}

	// Required only on this branch. The DTO cannot express "required unless the
	// address already has an account", and enforcing it there would reject the
	// existing-account request that legitimately sends none of these.
	if req.FirstName == "" || req.LastName == "" || req.Password == "" {
		return nil, domainErr.New(domainErr.ErrValidation,
			"first_name, last_name and password are required to create an account", nil)
	}

	hash, err := uc.auth.HashPassword(req.Password)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to hash password", err)
	}

	user := &model.User{
		Email:        inv.Email,
		PasswordHash: hash,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Status:       model.UserStatusActive,
	}
	if err := uc.users.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// resolve looks the token up and checks it is still redeemable, collapsing every
// failure into the same answer. See errInvalidToken.
func (uc *AcceptInvitationUseCase) resolve(ctx context.Context, token string) (*model.Invitation, error) {
	inv, err := uc.invitations.GetByTokenHash(ctx, HashInvitationToken(token))
	if err != nil {
		if errors.Is(err, domainErr.ErrNotFound) {
			return nil, errInvalidToken()
		}
		return nil, err
	}
	if !inv.IsRedeemable(time.Now().UTC()) {
		return nil, errInvalidToken()
	}
	return inv, nil
}
