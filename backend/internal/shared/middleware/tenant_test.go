package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	tenantModel "github.com/Root-Emin/TicketLens/internal/domain/tenant/model"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
)

// stubOrgRepo resolves one slug, for the subdomain branch.
type stubOrgRepo struct {
	slug string
	org  *tenantModel.Organization
}

func (r *stubOrgRepo) Create(context.Context, *tenantModel.Organization) error { return nil }
func (r *stubOrgRepo) GetByID(context.Context, uuid.UUID) (*tenantModel.Organization, error) {
	return nil, domainErr.New(domainErr.ErrNotFound, "organization not found", nil)
}
func (r *stubOrgRepo) GetBySlug(_ context.Context, slug string) (*tenantModel.Organization, error) {
	if r.org != nil && slug == r.slug {
		return r.org, nil
	}
	return nil, domainErr.New(domainErr.ErrNotFound, "organization not found", nil)
}
func (r *stubOrgRepo) Update(context.Context, *tenantModel.Organization) error { return nil }
func (r *stubOrgRepo) Delete(context.Context, uuid.UUID) error                 { return nil }
func (r *stubOrgRepo) List(context.Context, int, int) ([]*tenantModel.Organization, int, error) {
	return nil, 0, nil
}

// resolveTenant runs the resolver and reports the status plus the tenant the
// downstream handler ended up seeing.
func resolveTenant(t *testing.T, claimOrgID uuid.UUID, header string) (int, uuid.UUID) {
	t.Helper()

	var resolved uuid.UUID
	handler := TenantResolver(&stubOrgRepo{})(http.HandlerFunc(
		func(_ http.ResponseWriter, r *http.Request) {
			if id, ok := TenantIDFromContext(r.Context()); ok {
				resolved = id
			}
		}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets", nil)
	if header != "" {
		req.Header.Set("X-Organization-ID", header)
	}
	if claimOrgID != uuid.Nil {
		req = req.WithContext(context.WithValue(req.Context(), ContextKeyOrganizationID, claimOrgID))
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec.Code, resolved
}

func TestTenantResolver_TokenIsAuthoritative(t *testing.T) {
	claimOrgID := uuid.New()

	status, resolved := resolveTenant(t, claimOrgID, "")

	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, claimOrgID, resolved)
}

func TestTenantResolver_HeaderCannotImpersonateAnotherTenant(t *testing.T) {
	// The header used to be consulted first, so naming another organization's id
	// was enough to operate inside it with your own token.
	claimOrgID := uuid.New()
	victimOrgID := uuid.New()

	status, resolved := resolveTenant(t, claimOrgID, victimOrgID.String())

	assert.Equal(t, http.StatusForbidden, status)
	assert.Equal(t, uuid.Nil, resolved, "the request must not reach the handler at all")
}

func TestTenantResolver_MatchingHeaderIsAccepted(t *testing.T) {
	// Clients that send the header explicitly keep working, as long as it agrees
	// with the token.
	claimOrgID := uuid.New()

	status, resolved := resolveTenant(t, claimOrgID, claimOrgID.String())

	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, claimOrgID, resolved)
}

func TestTenantResolver_MalformedHeaderIsRejected(t *testing.T) {
	status, _ := resolveTenant(t, uuid.New(), "not-a-uuid")

	assert.Equal(t, http.StatusBadRequest, status)
}

func TestTenantResolver_HeaderAloneResolvesNothing(t *testing.T) {
	// With no token there is nothing for the header to agree with, so it must not
	// establish a tenant on its own.
	someOrgID := uuid.New()

	status, resolved := resolveTenant(t, uuid.Nil, someOrgID.String())

	assert.Equal(t, http.StatusOK, status, "resolution is optional; RequireTenant decides")
	assert.Equal(t, uuid.Nil, resolved, "no tenant may be derived from the header alone")
}

func TestRequireTenant_RejectsUnresolvedTenant(t *testing.T) {
	reached := false
	handler := RequireTenant(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		reached = true
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/tickets", nil))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.False(t, reached)
}

func TestRequireTenant_AdmitsResolvedTenant(t *testing.T) {
	orgID := uuid.New()
	reached := false
	handler := RequireTenant(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		reached = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets", nil)
	req = req.WithContext(context.WithValue(req.Context(), ContextKeyTenantID, orgID))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, reached)
}
