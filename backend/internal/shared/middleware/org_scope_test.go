package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	iamModel "github.com/Root-Emin/TicketLens/internal/domain/iam/model"
	tenantModel "github.com/Root-Emin/TicketLens/internal/domain/tenant/model"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
)

// These guards are the only thing standing between a tenant and another
// tenant's data on every path-addressed route, so each one is tested against
// the attack it exists to stop: naming somebody else's id.

// scopeTestApp is the app owned by ownerOrgID in the stub repository.
type stubAppRepo struct {
	app *tenantModel.App
	err error
}

func (r *stubAppRepo) Create(context.Context, *tenantModel.App) error { return nil }
func (r *stubAppRepo) GetByID(_ context.Context, id uuid.UUID) (*tenantModel.App, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.app == nil || r.app.ID != id {
		return nil, domainErr.New(domainErr.ErrNotFound, "app not found", nil)
	}
	return r.app, nil
}
func (r *stubAppRepo) GetBySlug(context.Context, uuid.UUID, string) (*tenantModel.App, error) {
	return nil, domainErr.New(domainErr.ErrNotFound, "app not found", nil)
}
func (r *stubAppRepo) Update(context.Context, *tenantModel.App) error { return nil }
func (r *stubAppRepo) Delete(context.Context, uuid.UUID) error        { return nil }
func (r *stubAppRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*tenantModel.App, int, error) {
	return nil, 0, nil
}

// stubRoleRepo answers membership questions from a fixed set of assignments.
type stubRoleRepo struct {
	// memberships maps "userID/orgID" to whether the user holds a role there.
	memberships map[string]bool
	err         error
}

func (r *stubRoleRepo) GetUserRoles(_ context.Context, userID, orgID uuid.UUID) ([]*iamModel.UserRole, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.memberships[userID.String()+"/"+orgID.String()] {
		return []*iamModel.UserRole{{UserID: userID, OrganizationID: orgID}}, nil
	}
	return nil, nil
}

func (r *stubRoleRepo) GetUserRoleNames(context.Context, uuid.UUID, uuid.UUID) ([]string, error) {
	return nil, nil
}

func (r *stubRoleRepo) Create(context.Context, *iamModel.Role) error { return nil }
func (r *stubRoleRepo) GetByID(context.Context, uuid.UUID) (*iamModel.Role, error) {
	return nil, domainErr.New(domainErr.ErrNotFound, "role not found", nil)
}
func (r *stubRoleRepo) ListByScope(context.Context, iamModel.ScopeType, uuid.UUID) ([]*iamModel.Role, error) {
	return nil, nil
}
func (r *stubRoleRepo) Update(context.Context, *iamModel.Role) error                   { return nil }
func (r *stubRoleRepo) Delete(context.Context, uuid.UUID) error                        { return nil }
func (r *stubRoleRepo) AddPermission(context.Context, uuid.UUID, string) error         { return nil }
func (r *stubRoleRepo) RemovePermission(context.Context, uuid.UUID, string) error      { return nil }
func (r *stubRoleRepo) GetPermissions(context.Context, uuid.UUID) ([]string, error)    { return nil, nil }
func (r *stubRoleRepo) AssignRoleToUser(context.Context, *iamModel.UserRole) error     { return nil }
func (r *stubRoleRepo) RemoveRoleFromUser(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (r *stubRoleRepo) GetUserPermissions(context.Context, uuid.UUID, uuid.UUID) ([]string, error) {
	return nil, nil
}
func (r *stubRoleRepo) GetPrimaryOrganization(context.Context, uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, nil
}

// serveGuarded runs a request through guard on a route carrying param, and
// reports the status plus whether the protected handler ran at all.
func serveGuarded(
	t *testing.T,
	guard func(http.Handler) http.Handler,
	pattern, path string,
	ctxValues map[contextKey]any,
) (int, bool) {
	t.Helper()

	reached := false
	r := chi.NewRouter()
	r.With(guard).Get(pattern, func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, path, nil)
	ctx := req.Context()
	for k, v := range ctxValues {
		ctx = context.WithValue(ctx, k, v)
	}

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req.WithContext(ctx))
	return rec.Code, reached
}

func TestRequireOrgFromPath(t *testing.T) {
	callerOrgID := uuid.New()
	otherOrgID := uuid.New()

	tests := []struct {
		name       string
		path       string
		claim      any
		wantStatus int
		wantReach  bool
	}{
		{
			name:       "own organization is admitted",
			path:       "/orgs/" + callerOrgID.String(),
			claim:      callerOrgID,
			wantStatus: http.StatusOK,
			wantReach:  true,
		},
		{
			name:  "another organization looks like it does not exist",
			path:  "/orgs/" + otherOrgID.String(),
			claim: callerOrgID,
			// 404 rather than 403: a 403 would confirm the id is real.
			wantStatus: http.StatusNotFound,
			wantReach:  false,
		},
		{
			name:       "no organization claim is rejected",
			path:       "/orgs/" + callerOrgID.String(),
			claim:      nil,
			wantStatus: http.StatusNotFound,
			wantReach:  false,
		},
		{
			name:       "nil organization claim is rejected",
			path:       "/orgs/" + uuid.Nil.String(),
			claim:      uuid.Nil,
			wantStatus: http.StatusNotFound,
			wantReach:  false,
		},
		{
			name:       "malformed id is a bad request",
			path:       "/orgs/not-a-uuid",
			claim:      callerOrgID,
			wantStatus: http.StatusBadRequest,
			wantReach:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctxValues := map[contextKey]any{}
			if tc.claim != nil {
				ctxValues[ContextKeyOrganizationID] = tc.claim
			}

			status, reached := serveGuarded(t, RequireOrgFromPath("orgId"),
				"/orgs/{orgId}", tc.path, ctxValues)

			assert.Equal(t, tc.wantStatus, status)
			assert.Equal(t, tc.wantReach, reached)
		})
	}
}

func TestRequireAppInOrg(t *testing.T) {
	callerOrgID := uuid.New()
	foreignOrgID := uuid.New()
	appID := uuid.New()

	ownApp := &tenantModel.App{ID: appID, OrganizationID: callerOrgID}
	foreignApp := &tenantModel.App{ID: appID, OrganizationID: foreignOrgID}

	tests := []struct {
		name       string
		repo       *stubAppRepo
		wantStatus int
		wantReach  bool
	}{
		{
			name:       "app in the caller's organization is admitted",
			repo:       &stubAppRepo{app: ownApp},
			wantStatus: http.StatusOK,
			wantReach:  true,
		},
		{
			name: "app owned by another organization is hidden",
			// This is the case that let any holder of app:write mint or revoke
			// API keys for another tenant's app.
			repo:       &stubAppRepo{app: foreignApp},
			wantStatus: http.StatusNotFound,
			wantReach:  false,
		},
		{
			name:       "unknown app is hidden",
			repo:       &stubAppRepo{},
			wantStatus: http.StatusNotFound,
			wantReach:  false,
		},
		{
			name:       "a lookup failure denies rather than admits",
			repo:       &stubAppRepo{err: domainErr.New(domainErr.ErrInternal, "connection reset", nil)},
			wantStatus: http.StatusNotFound,
			wantReach:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, reached := serveGuarded(t, RequireAppInOrg(tc.repo, "appId"),
				"/apps/{appId}", "/apps/"+appID.String(),
				map[contextKey]any{ContextKeyOrganizationID: callerOrgID})

			assert.Equal(t, tc.wantStatus, status)
			assert.Equal(t, tc.wantReach, reached)
		})
	}
}

func TestRequireChildOfPathResource(t *testing.T) {
	appID := uuid.New()
	foreignAppID := uuid.New()
	keyID := uuid.New()

	resolveTo := func(parentID uuid.UUID, err error) ParentResolver {
		return func(context.Context, uuid.UUID) (uuid.UUID, error) { return parentID, err }
	}

	tests := []struct {
		name       string
		resolve    ParentResolver
		path       string
		wantStatus int
		wantReach  bool
	}{
		{
			name:       "key belonging to the app in the path is admitted",
			resolve:    resolveTo(appID, nil),
			path:       "/apps/" + appID.String() + "/keys/" + keyID.String(),
			wantStatus: http.StatusOK,
			wantReach:  true,
		},
		{
			name: "key belonging to another app is hidden",
			// The attack: name an app you own, pair it with somebody else's key
			// id, and the handler would revoke their credential.
			resolve:    resolveTo(foreignAppID, nil),
			path:       "/apps/" + appID.String() + "/keys/" + keyID.String(),
			wantStatus: http.StatusNotFound,
			wantReach:  false,
		},
		{
			name:       "unresolvable child is hidden",
			resolve:    resolveTo(uuid.Nil, domainErr.New(domainErr.ErrNotFound, "api key not found", nil)),
			path:       "/apps/" + appID.String() + "/keys/" + keyID.String(),
			wantStatus: http.StatusNotFound,
			wantReach:  false,
		},
		{
			name:       "malformed child id is a bad request",
			resolve:    resolveTo(appID, nil),
			path:       "/apps/" + appID.String() + "/keys/nope",
			wantStatus: http.StatusBadRequest,
			wantReach:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			guard := RequireChildOfPathResource(tc.resolve, "appId", "keyId", "api key")

			status, reached := serveGuarded(t, guard,
				"/apps/{appId}/keys/{keyId}", tc.path, nil)

			assert.Equal(t, tc.wantStatus, status)
			assert.Equal(t, tc.wantReach, reached)
		})
	}
}

func TestRequireUserInOrg(t *testing.T) {
	callerOrgID := uuid.New()
	callerID := uuid.New()
	colleagueID := uuid.New()
	outsiderID := uuid.New()

	repo := &stubRoleRepo{memberships: map[string]bool{
		colleagueID.String() + "/" + callerOrgID.String(): true,
	}}

	tests := []struct {
		name       string
		targetID   uuid.UUID
		repo       *stubRoleRepo
		wantStatus int
		wantReach  bool
	}{
		{
			name:       "a colleague is admitted",
			targetID:   colleagueID,
			repo:       repo,
			wantStatus: http.StatusOK,
			wantReach:  true,
		},
		{
			name: "reading your own record needs no membership row",
			// Nothing writes organization_users, and a user who has just
			// registered holds no role yet; they must still read themselves.
			targetID:   callerID,
			repo:       repo,
			wantStatus: http.StatusOK,
			wantReach:  true,
		},
		{
			name:       "a user from another organization is hidden",
			targetID:   outsiderID,
			repo:       repo,
			wantStatus: http.StatusNotFound,
			wantReach:  false,
		},
		{
			name:     "a lookup failure denies rather than admits",
			targetID: colleagueID,
			repo: &stubRoleRepo{
				memberships: repo.memberships,
				err:         domainErr.New(domainErr.ErrInternal, "connection reset", nil),
			},
			wantStatus: http.StatusNotFound,
			wantReach:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, reached := serveGuarded(t, RequireUserInOrg(tc.repo, "userId"),
				"/users/{userId}", "/users/"+tc.targetID.String(),
				map[contextKey]any{
					ContextKeyOrganizationID: callerOrgID,
					ContextKeyUserID:         callerID,
				})

			assert.Equal(t, tc.wantStatus, status)
			assert.Equal(t, tc.wantReach, reached)
		})
	}
}
