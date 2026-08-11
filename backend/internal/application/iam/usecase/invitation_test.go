package usecase

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Root-Emin/TicketLens/internal/application/iam/dto"
	iamModel "github.com/Root-Emin/TicketLens/internal/domain/iam/model"
	tenantModel "github.com/Root-Emin/TicketLens/internal/domain/tenant/model"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
)

/*
	Staff onboarding.

	Two properties carry most of the weight here.

	The first is that a token is a bearer credential for creating an account, so
	every way it can fail — never existed, already spent, ran out of time — has
	to be indistinguishable from outside. Anything else lets somebody sort real
	tokens from noise and narrow a brute force to the live ones.

	The second is that redemption is atomic. A half-finished acceptance leaves a
	login that belongs to no organization, cannot be invited again because the
	address is now taken, and can still sign in — a state with no route back that
	does not involve the database.
*/

type inviteFixture struct {
	orgID        uuid.UUID
	roleID       uuid.UUID
	departmentID uuid.UUID

	invitations *fakeInvitationRepo
	users       *fakeUserStore
	roles       *fakeRoleStore
	orgs        *fakeOrgStore
	staff       *fakeStaffStore
	tx          *fakeTx
	rbac        *fakeRBAC
	bus         *fakeInviteBus
	mailer      *recordingMailer
}

func newInviteFixture(users ...*iamModel.User) *inviteFixture {
	orgID := uuid.New()
	roleID := uuid.New()

	return &inviteFixture{
		orgID:        orgID,
		roleID:       roleID,
		departmentID: uuid.New(),

		invitations: newFakeInvitationRepo(),
		users:       newFakeUserStore(users...),
		roles: newFakeRoleStore(&iamModel.Role{
			ID:        roleID,
			Name:      "agent",
			ScopeType: iamModel.ScopeTypeOrganization,
			ScopeID:   orgID,
		}),
		orgs:   &fakeOrgStore{org: &tenantModel.Organization{ID: orgID, Name: "Acme Support"}},
		staff:  newFakeStaffStore(),
		tx:     &fakeTx{},
		rbac:   &fakeRBAC{},
		bus:    &fakeInviteBus{},
		mailer: &recordingMailer{},
	}
}

func (f *inviteFixture) createUC() *CreateInvitationUseCase {
	return NewCreateInvitationUseCase(
		f.invitations, f.users, f.roles, f.orgs, f.mailer,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		"https://support.example.com", 7*24*time.Hour,
	)
}

func (f *inviteFixture) acceptUC() *AcceptInvitationUseCase {
	return NewAcceptInvitationUseCase(
		f.invitations, f.users, f.roles, f.orgs, f.staff,
		fakeAuth{}, f.rbac, f.tx, f.bus,
	)
}

// seed puts a live invitation in the store and returns its raw token, which is
// otherwise unrecoverable — only the hash is kept.
func (f *inviteFixture) seed(email string, departmentID *uuid.UUID) string {
	token, err := generateInvitationToken()
	if err != nil {
		panic(err)
	}
	inv := &iamModel.Invitation{
		ID:             uuid.New(),
		OrganizationID: f.orgID,
		Email:          email,
		RoleID:         f.roleID,
		DepartmentID:   departmentID,
		TokenHash:      HashInvitationToken(token),
		ExpiresAt:      time.Now().UTC().Add(24 * time.Hour),
	}
	f.invitations.byID[inv.ID] = inv
	return token
}

// ------------------------------------------------------------------ create

func TestCreateInvitation_RejectsRoleFromAnotherOrganization(t *testing.T) {
	f := newInviteFixture()

	// A role that exists, but belongs to somebody else's tenant. Without the
	// scope check this is accepted and grants membership there — the ids being
	// unguessable is not an authorization check.
	foreign := &iamModel.Role{
		ID:        uuid.New(),
		Name:      "admin",
		ScopeType: iamModel.ScopeTypeOrganization,
		ScopeID:   uuid.New(),
	}
	f.roles.roles[foreign.ID] = foreign

	_, err := f.createUC().Execute(context.Background(), f.orgID, dto.CreateInvitationRequest{
		Email:  "mallory@example.com",
		RoleID: foreign.ID,
	})

	if !errors.Is(err, domainErr.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for a foreign role, got %v", err)
	}
	if len(f.invitations.byID) != 0 {
		t.Fatalf("no invitation should have been created, found %d", len(f.invitations.byID))
	}
}

func TestCreateInvitation_NormalisesEmailSoDuplicatesCannotSlipThrough(t *testing.T) {
	f := newInviteFixture()
	uc := f.createUC()

	first, err := uc.Execute(context.Background(), f.orgID, dto.CreateInvitationRequest{
		Email: "Ada@Example.com", RoleID: f.roleID,
	})
	if err != nil {
		t.Fatalf("first invitation failed: %v", err)
	}
	if first.Email != "ada@example.com" {
		t.Fatalf("email not normalised: %q", first.Email)
	}

	// Same address, different capitalisation. Compared in the stored form, this
	// is the same person and must collide rather than create a second live link.
	_, err = uc.Execute(context.Background(), f.orgID, dto.CreateInvitationRequest{
		Email: "ADA@example.com", RoleID: f.roleID,
	})
	if !errors.Is(err, domainErr.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists for a duplicate address, got %v", err)
	}
}

func TestCreateInvitation_RefusesSomebodyWhoIsAlreadyAMember(t *testing.T) {
	member := &iamModel.User{ID: uuid.New(), Email: "colleague@example.com", Status: iamModel.UserStatusActive}
	f := newInviteFixture(member)
	f.roles.roleNames[member.ID] = []string{"agent"}

	_, err := f.createUC().Execute(context.Background(), f.orgID, dto.CreateInvitationRequest{
		Email: "colleague@example.com", RoleID: f.roleID,
	})

	if !errors.Is(err, domainErr.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists for a current member, got %v", err)
	}
}

func TestCreateInvitation_SurvivesAMailerThatCannotDeliver(t *testing.T) {
	f := newInviteFixture()
	f.mailer.sendErr = errors.New("connection refused")

	result, err := f.createUC().Execute(context.Background(), f.orgID, dto.CreateInvitationRequest{
		Email: "ada@example.com", RoleID: f.roleID,
	})

	// The invitation is the record; the email is a convenience. Failing here
	// would destroy a valid invitation because a relay was down, when the
	// administrator is holding a working link in this very response.
	if err != nil {
		t.Fatalf("a mail failure must not fail the invitation: %v", err)
	}
	if result.AcceptURL == "" {
		t.Fatal("accept_url must be returned so the link can be passed on by hand")
	}
	if len(f.invitations.byID) != 1 {
		t.Fatalf("expected the invitation to be stored, found %d", len(f.invitations.byID))
	}
}

func TestCreateInvitation_StoresOnlyTheHashAndReturnsTheLinkOnce(t *testing.T) {
	f := newInviteFixture()

	result, err := f.createUC().Execute(context.Background(), f.orgID, dto.CreateInvitationRequest{
		Email: "ada@example.com", RoleID: f.roleID,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	var stored *iamModel.Invitation
	for _, inv := range f.invitations.byID {
		stored = inv
	}

	// The raw token appears in the URL and in the mail, never in the store. A
	// database disclosure must not hand over usable invitations.
	token := result.AcceptURL[strings.LastIndex(result.AcceptURL, "/")+1:]
	if stored.TokenHash == token {
		t.Fatal("the raw token was stored instead of its hash")
	}
	if stored.TokenHash != HashInvitationToken(token) {
		t.Fatal("stored hash does not match the issued token")
	}
	if !strings.HasPrefix(token, tokenPrefix) {
		t.Fatalf("token is missing the %q prefix: %q", tokenPrefix, token)
	}

	if len(f.mailer.sent) != 1 {
		t.Fatalf("expected one email, got %d", len(f.mailer.sent))
	}
	if !strings.Contains(f.mailer.sent[0].Body, result.AcceptURL) {
		t.Fatal("the email does not carry the acceptance link")
	}
	if !strings.Contains(f.mailer.sent[0].Body, "Acme Support") {
		t.Fatal("the email does not name the organization")
	}
}

// ------------------------------------------------------------------ accept

func TestAcceptInvitation_CreatesTheAccountWithRoleAndDepartment(t *testing.T) {
	f := newInviteFixture()
	token := f.seed("ada@example.com", &f.departmentID)

	user, err := f.acceptUC().Execute(context.Background(), token, dto.AcceptInvitationRequest{
		FirstName: "Ada", LastName: "Lovelace", Password: "Secret12345",
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if user.Email != "ada@example.com" {
		t.Fatalf("account created under the wrong address: %q", user.Email)
	}
	if f.users.creates != 1 {
		t.Fatalf("expected exactly one account, got %d", f.users.creates)
	}

	// The role assignment is what membership actually is — see the note in
	// migration 00021 on why it is not organization_users.
	if len(f.roles.assignments) != 1 {
		t.Fatalf("expected one role assignment, got %d", len(f.roles.assignments))
	}
	got := f.roles.assignments[0]
	if got.OrganizationID != f.orgID || got.RoleID != f.roleID {
		t.Fatalf("role assigned into the wrong place: %+v", got)
	}

	// Placed on the team the invitation named, so the new hire is not sitting in
	// the roster's Unassigned bucket waiting for an administrator.
	placed, ok := f.staff.departments[user.ID]
	if !ok || placed == nil || *placed != f.departmentID {
		t.Fatalf("expected placement in department %s, got %v", f.departmentID, placed)
	}
}

func TestAcceptInvitation_LeavesNoDepartmentRowWhenNoneWasOffered(t *testing.T) {
	f := newInviteFixture()
	token := f.seed("ada@example.com", nil)

	user, err := f.acceptUC().Execute(context.Background(), token, dto.AcceptInvitationRequest{
		FirstName: "Ada", LastName: "Lovelace", Password: "Secret12345",
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	// Absence of a row is how staff_departments models "unassigned"; writing a
	// nil placement would be a different, misleading fact.
	if _, ok := f.staff.departments[user.ID]; ok {
		t.Fatal("no department was offered, so nothing should have been written")
	}
}

func TestAcceptInvitation_IsSingleUse(t *testing.T) {
	f := newInviteFixture()
	token := f.seed("ada@example.com", nil)
	uc := f.acceptUC()
	req := dto.AcceptInvitationRequest{FirstName: "Ada", LastName: "Lovelace", Password: "Secret12345"}

	if _, err := uc.Execute(context.Background(), token, req); err != nil {
		t.Fatalf("first acceptance failed: %v", err)
	}

	_, err := uc.Execute(context.Background(), token, req)
	if !errors.Is(err, domainErr.ErrNotFound) {
		t.Fatalf("expected a spent token to be refused, got %v", err)
	}
	if f.users.creates != 1 {
		t.Fatalf("a second account was created: %d", f.users.creates)
	}
}

// The three failure modes must be one answer. Asserting on the message, not
// only the code, because a helpful "this link has expired" is exactly the leak:
// it confirms the token was once real.
func TestAcceptInvitation_FailuresAreIndistinguishable(t *testing.T) {
	req := dto.AcceptInvitationRequest{FirstName: "A", LastName: "B", Password: "Secret12345"}

	unknown := newInviteFixture()
	_, unknownErr := unknown.acceptUC().Execute(context.Background(), "inv_deadbeef", req)

	expired := newInviteFixture()
	expiredToken := expired.seed("ada@example.com", nil)
	for _, inv := range expired.invitations.byID {
		inv.ExpiresAt = time.Now().UTC().Add(-time.Hour)
	}
	_, expiredErr := expired.acceptUC().Execute(context.Background(), expiredToken, req)

	revoked := newInviteFixture()
	revokedToken := revoked.seed("ada@example.com", nil)
	for _, inv := range revoked.invitations.byID {
		_ = revoked.invitations.Revoke(context.Background(), revoked.orgID, inv.ID)
	}
	_, revokedErr := revoked.acceptUC().Execute(context.Background(), revokedToken, req)

	for name, err := range map[string]error{
		"unknown": unknownErr, "expired": expiredErr, "revoked": revokedErr,
	} {
		if err == nil {
			t.Fatalf("%s token was accepted", name)
		}
		if !errors.Is(err, domainErr.ErrNotFound) {
			t.Fatalf("%s token: expected ErrNotFound, got %v", name, err)
		}
	}
	if unknownErr.Error() != expiredErr.Error() || unknownErr.Error() != revokedErr.Error() {
		t.Fatalf("failure messages differ and leak which token was real:\n unknown: %v\n expired: %v\n revoked: %v",
			unknownErr, expiredErr, revokedErr)
	}
}

func TestAcceptInvitation_JoinsAnExistingAccountWithoutTouchingItsPassword(t *testing.T) {
	existing := &iamModel.User{
		ID:           uuid.New(),
		Email:        "ada@example.com",
		PasswordHash: "hashed:TheirRealPassword",
		FirstName:    "Ada",
		LastName:     "Lovelace",
		Status:       iamModel.UserStatusActive,
	}
	f := newInviteFixture(existing)
	token := f.seed("ada@example.com", nil)

	user, err := f.acceptUC().Execute(context.Background(), token, dto.AcceptInvitationRequest{
		FirstName: "Attacker", LastName: "Name", Password: "NewPassword123",
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if user.ID != existing.ID {
		t.Fatal("a second account was created for an address that already had one")
	}
	if f.users.creates != 0 {
		t.Fatalf("expected no account creation, got %d", f.users.creates)
	}
	// Otherwise inviting a known address would be a password reset for it.
	if existing.PasswordHash != "hashed:TheirRealPassword" {
		t.Fatalf("the existing password was overwritten: %q", existing.PasswordHash)
	}
	if len(f.roles.assignments) != 1 {
		t.Fatalf("the existing account should still gain the role, got %d assignments", len(f.roles.assignments))
	}
}

func TestAcceptInvitation_RollsBackAndLeavesTheInvitationUnspent(t *testing.T) {
	f := newInviteFixture()
	token := f.seed("ada@example.com", nil)
	f.roles.assignErr = errors.New("constraint violation")

	_, err := f.acceptUC().Execute(context.Background(), token, dto.AcceptInvitationRequest{
		FirstName: "Ada", LastName: "Lovelace", Password: "Secret12345",
	})
	if err == nil {
		t.Fatal("expected the failure to surface")
	}
	if !f.tx.rolledBack {
		t.Fatal("the unit of work was not rolled back")
	}

	// The observable that matters: the invitation is still redeemable, so the
	// person can try again rather than being locked out by an address that is
	// taken by an account with no membership.
	for _, inv := range f.invitations.byID {
		if inv.AcceptedAt != nil {
			t.Fatal("the invitation was spent despite the failure")
		}
	}
}

func TestAcceptInvitation_PreviewTellsTheRecipientWhatTheyAreJoining(t *testing.T) {
	f := newInviteFixture()
	token := f.seed("ada@example.com", nil)

	preview, err := f.acceptUC().Preview(context.Background(), token)
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}

	if preview.OrganizationName != "Acme Support" || preview.RoleName != "agent" {
		t.Fatalf("preview does not describe the offer: %+v", preview)
	}
	if preview.Email != "ada@example.com" {
		t.Fatalf("preview shows the wrong address: %q", preview.Email)
	}

	// A preview must not spend the invitation — the recipient loads the page
	// before they fill the form in.
	for _, inv := range f.invitations.byID {
		if inv.AcceptedAt != nil {
			t.Fatal("previewing consumed the invitation")
		}
	}
}

// ------------------------------------------- invitees who already have a login

func TestAcceptInvitation_PreviewFlagsAnAddressThatAlreadyHasAnAccount(t *testing.T) {
	fresh := newInviteFixture()
	freshToken := fresh.seed("newcomer@example.com", nil)
	p, err := fresh.acceptUC().Preview(context.Background(), freshToken)
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	if p.HasAccount {
		t.Fatal("an unknown address must not be reported as having an account")
	}

	existing := &iamModel.User{
		ID: uuid.New(), Email: "ada@example.com",
		PasswordHash: "hashed:Theirs", Status: iamModel.UserStatusActive,
	}
	known := newInviteFixture(existing)
	knownToken := known.seed("ada@example.com", nil)
	p, err = known.acceptUC().Preview(context.Background(), knownToken)
	if err != nil {
		t.Fatalf("Preview returned error: %v", err)
	}
	// Without this the screen asks for a password that acceptance discards, and
	// the person cannot then sign in with what they just typed.
	if !p.HasAccount {
		t.Fatal("a known address must be flagged so the screen skips the password form")
	}
}

func TestAcceptInvitation_ExistingAccountNeedsNoPasswordOrName(t *testing.T) {
	existing := &iamModel.User{
		ID: uuid.New(), Email: "ada@example.com",
		PasswordHash: "hashed:Theirs", FirstName: "Ada", LastName: "Lovelace",
		Status: iamModel.UserStatusActive,
	}
	f := newInviteFixture(existing)
	token := f.seed("ada@example.com", nil)

	// An empty request: the screen showed no form because HasAccount was set.
	user, err := f.acceptUC().Execute(context.Background(), token, dto.AcceptInvitationRequest{})
	if err != nil {
		t.Fatalf("an existing account should join with no credentials supplied: %v", err)
	}

	if user.ID != existing.ID {
		t.Fatal("joined the wrong account")
	}
	if existing.PasswordHash != "hashed:Theirs" {
		t.Fatalf("existing password was touched: %q", existing.PasswordHash)
	}
	if len(f.roles.assignments) != 1 {
		t.Fatalf("expected the role to be granted, got %d assignments", len(f.roles.assignments))
	}
}

func TestAcceptInvitation_NewAccountStillRequiresNameAndPassword(t *testing.T) {
	f := newInviteFixture()
	token := f.seed("ada@example.com", nil)

	// The same empty request that is legitimate above must not silently create
	// an account with no password — the DTO cannot express the difference, so
	// the use case has to.
	_, err := f.acceptUC().Execute(context.Background(), token, dto.AcceptInvitationRequest{})
	if !errors.Is(err, domainErr.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
	if f.users.creates != 0 {
		t.Fatalf("an account was created without credentials: %d", f.users.creates)
	}
}

// -------------------------------------------------------------------- roles

func TestListRoles_OffersStaffRolesAndNotCustomer(t *testing.T) {
	f := newInviteFixture()
	for _, name := range []string{"admin", "viewer", customerRoleName} {
		id := uuid.New()
		f.roles.roles[id] = &iamModel.Role{
			ID: id, Name: name,
			ScopeType: iamModel.ScopeTypeOrganization, ScopeID: f.orgID,
		}
	}
	// Another tenant's role, which must not appear.
	foreign := uuid.New()
	f.roles.roles[foreign] = &iamModel.Role{
		ID: foreign, Name: "agent",
		ScopeType: iamModel.ScopeTypeOrganization, ScopeID: uuid.New(),
	}

	roles, err := NewListRolesUseCase(f.roles).Execute(context.Background(), f.orgID)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	got := map[string]bool{}
	for _, r := range roles {
		got[r.Name] = true
	}

	for _, want := range []string{"agent", "admin", "viewer"} {
		if !got[want] {
			t.Fatalf("expected %q in the assignable roles, got %v", want, got)
		}
	}
	// A portal role offered on a staff invitation form puts somebody in the
	// organization but on no roster, which reads as a bug from every screen.
	if got[customerRoleName] {
		t.Fatal("the customer role must not be offered as a staff role")
	}
	if len(roles) != 3 {
		t.Fatalf("expected exactly the 3 staff roles of this organization, got %d", len(roles))
	}
}

// ------------------------------------------------------------------ revoke

func TestRevokeInvitation_FreesTheAddressForANewInvitation(t *testing.T) {
	f := newInviteFixture()
	create := f.createUC()

	if _, err := create.Execute(context.Background(), f.orgID, dto.CreateInvitationRequest{
		Email: "ada@example.com", RoleID: f.roleID,
	}); err != nil {
		t.Fatalf("first invitation failed: %v", err)
	}

	var id uuid.UUID
	for _, inv := range f.invitations.byID {
		id = inv.ID
	}

	if err := NewRevokeInvitationUseCase(f.invitations).Execute(context.Background(), f.orgID, id); err != nil {
		t.Fatalf("revoke failed: %v", err)
	}

	// One live invitation per address is a partial unique index; revoking is the
	// documented way to reissue, so it has to actually clear the way.
	if _, err := create.Execute(context.Background(), f.orgID, dto.CreateInvitationRequest{
		Email: "ada@example.com", RoleID: f.roleID,
	}); err != nil {
		t.Fatalf("re-invitation after revoke failed: %v", err)
	}
}

func TestRevokeInvitation_CannotReachAnotherTenantsInvitation(t *testing.T) {
	f := newInviteFixture()
	f.seed("ada@example.com", nil)

	var id uuid.UUID
	for _, inv := range f.invitations.byID {
		id = inv.ID
	}

	// Not 403: that would confirm the id names a real invitation somewhere.
	err := NewRevokeInvitationUseCase(f.invitations).Execute(context.Background(), uuid.New(), id)
	if !errors.Is(err, domainErr.ErrNotFound) {
		t.Fatalf("expected ErrNotFound across tenants, got %v", err)
	}
}
