package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	iamModel "github.com/Root-Emin/TicketLens/internal/domain/iam/model"
	iamPort "github.com/Root-Emin/TicketLens/internal/domain/iam/port"
	iamRepo "github.com/Root-Emin/TicketLens/internal/domain/iam/repository"
	tenantModel "github.com/Root-Emin/TicketLens/internal/domain/tenant/model"
	triageModel "github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	triageRepo "github.com/Root-Emin/TicketLens/internal/domain/triage/repository"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
	"github.com/Root-Emin/TicketLens/internal/shared/events"
)

// In-memory doubles for the invitation flow.
//
// Hand-written and stateful, like the triage fakes: what these tests assert is
// what ended up stored after a redemption — one user, one role assignment, a
// spent invitation — and expectation-based mocks would describe the calls made
// instead of the state reached.

func notFound(what string) error {
	return domainErr.New(domainErr.ErrNotFound, what+" not found", nil)
}

// ------------------------------------------------------------- invitations

type fakeInvitationRepo struct {
	byID map[uuid.UUID]*iamModel.Invitation

	// createErr and acceptErr inject failure at the two points the acceptance
	// transaction can break, so a test can assert nothing partial survives.
	createErr error
}

var _ iamRepo.InvitationRepository = (*fakeInvitationRepo)(nil)

func newFakeInvitationRepo(invitations ...*iamModel.Invitation) *fakeInvitationRepo {
	r := &fakeInvitationRepo{byID: make(map[uuid.UUID]*iamModel.Invitation, len(invitations))}
	for _, inv := range invitations {
		r.byID[inv.ID] = inv
	}
	return r
}

func (r *fakeInvitationRepo) Create(_ context.Context, inv *iamModel.Invitation) error {
	if r.createErr != nil {
		return r.createErr
	}
	if inv.ID == uuid.Nil {
		inv.ID = uuid.New()
	}
	inv.CreatedAt = time.Now().UTC()
	r.byID[inv.ID] = inv
	return nil
}

func (r *fakeInvitationRepo) GetByTokenHash(_ context.Context, hash string) (*iamModel.Invitation, error) {
	for _, inv := range r.byID {
		if inv.TokenHash == hash {
			return inv, nil
		}
	}
	return nil, notFound("invitation")
}

func (r *fakeInvitationRepo) GetByID(_ context.Context, orgID, id uuid.UUID) (*iamModel.Invitation, error) {
	inv, ok := r.byID[id]
	if !ok || inv.OrganizationID != orgID {
		return nil, notFound("invitation")
	}
	return inv, nil
}

func (r *fakeInvitationRepo) FindLiveByEmail(_ context.Context, orgID uuid.UUID, email string) (*iamModel.Invitation, error) {
	for _, inv := range r.byID {
		if inv.OrganizationID == orgID && inv.Email == email &&
			inv.AcceptedAt == nil && inv.RevokedAt == nil {
			return inv, nil
		}
	}
	return nil, notFound("invitation")
}

func (r *fakeInvitationRepo) ListByOrg(_ context.Context, orgID uuid.UUID, offset, limit int) ([]*iamModel.Invitation, int, error) {
	var all []*iamModel.Invitation
	for _, inv := range r.byID {
		if inv.OrganizationID == orgID {
			all = append(all, inv)
		}
	}
	total := len(all)
	if offset >= total {
		return nil, total, nil
	}
	end := min(offset+limit, total)
	return all[offset:end], total, nil
}

// MarkAccepted mirrors the real UPDATE ... WHERE accepted_at IS NULL: spending
// an already-spent invitation affects no row and reports not-found. The tests
// for single use depend on this being modelled faithfully.
func (r *fakeInvitationRepo) MarkAccepted(_ context.Context, id uuid.UUID) error {
	inv, ok := r.byID[id]
	if !ok || inv.AcceptedAt != nil || inv.RevokedAt != nil {
		return notFound("invitation")
	}
	now := time.Now().UTC()
	inv.AcceptedAt = &now
	return nil
}

func (r *fakeInvitationRepo) Revoke(_ context.Context, orgID, id uuid.UUID) error {
	inv, ok := r.byID[id]
	if !ok || inv.OrganizationID != orgID || inv.AcceptedAt != nil || inv.RevokedAt != nil {
		return notFound("invitation")
	}
	now := time.Now().UTC()
	inv.RevokedAt = &now
	return nil
}

// ------------------------------------------------------------------- users

type fakeUserStore struct {
	byID    map[uuid.UUID]*iamModel.User
	byEmail map[string]*iamModel.User
	creates int
}

var _ iamRepo.UserRepository = (*fakeUserStore)(nil)

func newFakeUserStore(users ...*iamModel.User) *fakeUserStore {
	s := &fakeUserStore{
		byID:    make(map[uuid.UUID]*iamModel.User),
		byEmail: make(map[string]*iamModel.User),
	}
	for _, u := range users {
		s.byID[u.ID] = u
		s.byEmail[strings.ToLower(u.Email)] = u
	}
	return s
}

func (s *fakeUserStore) Create(_ context.Context, u *iamModel.User) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	u.CreatedAt = time.Now().UTC()
	s.creates++
	s.byID[u.ID] = u
	s.byEmail[strings.ToLower(u.Email)] = u
	return nil
}

func (s *fakeUserStore) GetByID(_ context.Context, id uuid.UUID) (*iamModel.User, error) {
	if u, ok := s.byID[id]; ok {
		return u, nil
	}
	return nil, notFound("user")
}

func (s *fakeUserStore) GetByEmail(_ context.Context, email string) (*iamModel.User, error) {
	if u, ok := s.byEmail[strings.ToLower(email)]; ok {
		return u, nil
	}
	return nil, notFound("user")
}

func (s *fakeUserStore) Update(context.Context, *iamModel.User) error { return nil }
func (s *fakeUserStore) UpdatePassword(_ context.Context, id uuid.UUID, hash string) error {
	u, ok := s.byID[id]
	if !ok {
		return notFound("user")
	}
	u.PasswordHash = hash
	return nil
}
func (s *fakeUserStore) Delete(context.Context, uuid.UUID) error { return nil }
func (s *fakeUserStore) List(context.Context, int, int) ([]*iamModel.User, int, error) {
	return nil, 0, nil
}
func (s *fakeUserStore) ListByOrganization(context.Context, uuid.UUID, int, int) ([]*iamModel.User, int, error) {
	return nil, 0, nil
}

// ------------------------------------------------------------------- roles

type fakeRoleStore struct {
	roles map[uuid.UUID]*iamModel.Role
	// assignments is what a redemption is ultimately judged by: the user_roles
	// row is what makes somebody a member.
	assignments []*iamModel.UserRole
	// roleNames answers GetUserRoleNames, i.e. "is this person already a member".
	roleNames map[uuid.UUID][]string
	assignErr error
}

var _ iamRepo.RoleRepository = (*fakeRoleStore)(nil)

func newFakeRoleStore(roles ...*iamModel.Role) *fakeRoleStore {
	s := &fakeRoleStore{
		roles:     make(map[uuid.UUID]*iamModel.Role, len(roles)),
		roleNames: make(map[uuid.UUID][]string),
	}
	for _, r := range roles {
		s.roles[r.ID] = r
	}
	return s
}

func (s *fakeRoleStore) GetByID(_ context.Context, id uuid.UUID) (*iamModel.Role, error) {
	if r, ok := s.roles[id]; ok {
		return r, nil
	}
	return nil, notFound("role")
}

func (s *fakeRoleStore) AssignRoleToUser(_ context.Context, ur *iamModel.UserRole) error {
	if s.assignErr != nil {
		return s.assignErr
	}
	s.assignments = append(s.assignments, ur)
	return nil
}

func (s *fakeRoleStore) GetUserRoleNames(_ context.Context, userID, _ uuid.UUID) ([]string, error) {
	return s.roleNames[userID], nil
}

func (s *fakeRoleStore) Create(context.Context, *iamModel.Role) error { return nil }

func (s *fakeRoleStore) ListByScope(_ context.Context, scopeType iamModel.ScopeType, scopeID uuid.UUID) ([]*iamModel.Role, error) {
	var out []*iamModel.Role
	for _, r := range s.roles {
		if r.ScopeType == scopeType && r.ScopeID == scopeID {
			out = append(out, r)
		}
	}
	return out, nil
}
func (s *fakeRoleStore) Update(context.Context, *iamModel.Role) error           { return nil }
func (s *fakeRoleStore) Delete(context.Context, uuid.UUID) error                { return nil }
func (s *fakeRoleStore) AddPermission(context.Context, uuid.UUID, string) error { return nil }
func (s *fakeRoleStore) RemovePermission(context.Context, uuid.UUID, string) error {
	return nil
}
func (s *fakeRoleStore) GetPermissions(context.Context, uuid.UUID) ([]string, error) { return nil, nil }
func (s *fakeRoleStore) RemoveRoleFromUser(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}
func (s *fakeRoleStore) GetUserRoles(context.Context, uuid.UUID, uuid.UUID) ([]*iamModel.UserRole, error) {
	return nil, nil
}
func (s *fakeRoleStore) GetUserPermissions(context.Context, uuid.UUID, uuid.UUID) ([]string, error) {
	return nil, nil
}
func (s *fakeRoleStore) GetPrimaryOrganization(context.Context, uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, nil
}

// ------------------------------------------------------------ organizations

type fakeOrgStore struct{ org *tenantModel.Organization }

func (s *fakeOrgStore) GetByID(_ context.Context, id uuid.UUID) (*tenantModel.Organization, error) {
	if s.org != nil && s.org.ID == id {
		return s.org, nil
	}
	return nil, notFound("organization")
}
func (s *fakeOrgStore) Create(context.Context, *tenantModel.Organization) error { return nil }
func (s *fakeOrgStore) GetBySlug(context.Context, string) (*tenantModel.Organization, error) {
	return nil, notFound("organization")
}
func (s *fakeOrgStore) Update(context.Context, *tenantModel.Organization) error { return nil }
func (s *fakeOrgStore) Delete(context.Context, uuid.UUID) error                 { return nil }
func (s *fakeOrgStore) List(context.Context, int, int) ([]*tenantModel.Organization, int, error) {
	return nil, 0, nil
}

// ------------------------------------------------------------------- staff

type fakeStaffStore struct {
	// departments records the placement a redemption wrote, keyed by user.
	departments map[uuid.UUID]*uuid.UUID
	setErr      error
}

var _ triageRepo.StaffRepository = (*fakeStaffStore)(nil)

func newFakeStaffStore() *fakeStaffStore {
	return &fakeStaffStore{departments: make(map[uuid.UUID]*uuid.UUID)}
}

func (s *fakeStaffStore) SetDepartment(_ context.Context, _, userID uuid.UUID, departmentID *uuid.UUID) error {
	if s.setErr != nil {
		return s.setErr
	}
	s.departments[userID] = departmentID
	return nil
}

func (s *fakeStaffStore) ListByOrg(context.Context, uuid.UUID, triageRepo.StaffFilter, int, int) ([]*triageModel.StaffMember, int, error) {
	return nil, 0, nil
}
func (s *fakeStaffStore) GetByUser(context.Context, uuid.UUID, uuid.UUID) (*triageModel.StaffMember, error) {
	return nil, notFound("staff member")
}
func (s *fakeStaffStore) CountByDepartment(context.Context, uuid.UUID) (map[uuid.UUID]int, error) {
	return nil, nil
}

// -------------------------------------------------------------- supporting

// fakeTx runs the unit of work without a real transaction, and records whether
// it was rolled back. The in-memory stores cannot actually undo their writes, so
// the rollback tests assert on this flag plus the absence of the *final* write
// (the spent invitation), which is the observable a real rollback guarantees.
type fakeTx struct{ rolledBack bool }

func (t *fakeTx) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	if err := fn(ctx); err != nil {
		t.rolledBack = true
		return err
	}
	return nil
}

type fakeRBAC struct{ invalidated int }

func (r *fakeRBAC) HasPermission(context.Context, uuid.UUID, uuid.UUID, string) (bool, error) {
	return true, nil
}
func (r *fakeRBAC) HasAnyPermission(context.Context, uuid.UUID, uuid.UUID, []string) (bool, error) {
	return true, nil
}
func (r *fakeRBAC) GetUserPermissions(context.Context, uuid.UUID, uuid.UUID) ([]string, error) {
	return nil, nil
}
func (r *fakeRBAC) InvalidateCache(context.Context, uuid.UUID, uuid.UUID) error {
	r.invalidated++
	return nil
}

type fakeInviteBus struct{ published int }

func (b *fakeInviteBus) Publish(context.Context, string, events.Event) error {
	b.published++
	return nil
}
func (b *fakeInviteBus) Subscribe(string, events.Handler) {}
func (b *fakeInviteBus) Close() error                     { return nil }

// recordingMailer captures what would have been sent, and can be made to fail so
// a test can assert an undelivered invitation is still a valid one.
type recordingMailer struct {
	sent    []iamPort.Message
	sendErr error
}

var _ iamPort.Mailer = (*recordingMailer)(nil)

func (m *recordingMailer) Send(_ context.Context, msg iamPort.Message) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.sent = append(m.sent, msg)
	return nil
}
