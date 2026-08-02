package middleware

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	iamRepo "github.com/Root-Emin/TicketLens/internal/domain/iam/repository"
	tenantRepo "github.com/Root-Emin/TicketLens/internal/domain/tenant/repository"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
	"github.com/Root-Emin/TicketLens/internal/shared/response"
)

// Routes that carry a resource id in the path need more than a permission
// check: a permission is granted *within* an organization, so `org:read` on
// your own organization must not read someone else's. RBAC alone cannot see
// that, because it never looks at the path. The guards below close that gap by
// comparing the addressed resource against the caller's organization claim.
//
// A mismatch answers 404, never 403. The api-contract requires it: a 403 would
// confirm that the id exists, which is itself a leak.

// notFound hides both "does not exist" and "belongs to someone else" behind one
// answer, so a caller cannot probe for ids.
func notFound(w http.ResponseWriter, resource string) {
	response.Error(w, domainErr.New(domainErr.ErrNotFound, resource+" not found", nil))
}

// callerOrg returns the organization from the JWT claim.
//
// It reads the claim rather than the resolved tenant on purpose: the tenant may
// have been influenced by a request header, and this is exactly the comparison
// that header must not be able to satisfy.
func callerOrg(r *http.Request) (uuid.UUID, bool) {
	orgID, ok := r.Context().Value(ContextKeyOrganizationID).(uuid.UUID)
	return orgID, ok && orgID != uuid.Nil
}

// RequireOrgFromPath admits the request only when the organization named in the
// path is the caller's own.
func RequireOrgFromPath(param string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pathOrgID, err := uuid.Parse(chi.URLParam(r, param))
			if err != nil {
				response.Error(w, domainErr.New(domainErr.ErrBadRequest, "invalid organization id", nil))
				return
			}

			claimOrgID, ok := callerOrg(r)
			if !ok || claimOrgID != pathOrgID {
				notFound(w, "organization")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAppInOrg admits the request only when the addressed app belongs to the
// caller's organization.
//
// This guards the whole app subtree, API keys included. Without it, `app:write`
// in any organization was enough to mint or revoke keys for any app on the
// platform, because those handlers look the app up by id alone.
func RequireAppInOrg(appRepo tenantRepo.AppRepository, param string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			appID, err := uuid.Parse(chi.URLParam(r, param))
			if err != nil {
				response.Error(w, domainErr.New(domainErr.ErrBadRequest, "invalid app id", nil))
				return
			}

			claimOrgID, ok := callerOrg(r)
			if !ok {
				notFound(w, "app")
				return
			}

			app, err := appRepo.GetByID(r.Context(), appID)
			if err != nil {
				// A lookup failure is reported as not-found either way, so the
				// caller learns nothing from the distinction.
				notFound(w, "app")
				return
			}
			if app.OrganizationID != claimOrgID {
				notFound(w, "app")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ParentResolver reports which parent resource a child belongs to.
type ParentResolver func(ctx context.Context, childID uuid.UUID) (uuid.UUID, error)

// RequireChildOfPathResource admits the request only when the child named in the
// path really belongs to the parent named in the path.
//
// Guarding the parent is not enough on its own. Handlers under
// /apps/{appId}/keys/{keyId} resolve the key by its own id and never compare it
// to {appId}, so naming an app you own alongside somebody else's key id was
// enough to revoke their credential. The same shape covers endpoints and their
// policies.
//
// It takes a resolver rather than a repository so this stays independent of
// every bounded context whose ids appear in a URL.
func RequireChildOfPathResource(resolve ParentResolver, parentParam, childParam, resourceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			childID, err := uuid.Parse(chi.URLParam(r, childParam))
			if err != nil {
				response.Error(w, domainErr.New(domainErr.ErrBadRequest, "invalid "+resourceName+" id", nil))
				return
			}

			parentID, err := uuid.Parse(chi.URLParam(r, parentParam))
			if err != nil {
				response.Error(w, domainErr.New(domainErr.ErrBadRequest, "invalid parent id", nil))
				return
			}

			actualParentID, err := resolve(r.Context(), childID)
			if err != nil || actualParentID != parentID {
				notFound(w, resourceName)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireUserInOrg admits the request only when the addressed user is a member
// of the caller's organization, so user-scoped reads cannot enumerate accounts
// from other tenants.
//
// Membership is read from role assignments rather than organization_users:
// nothing writes that table yet, and role assignments are what CreateOrgUseCase
// actually creates. AssignTicketUseCase already decides membership the same way.
func RequireUserInOrg(roleRepo iamRepo.RoleRepository, param string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			targetUserID, err := uuid.Parse(chi.URLParam(r, param))
			if err != nil {
				response.Error(w, domainErr.New(domainErr.ErrBadRequest, "invalid user id", nil))
				return
			}

			claimOrgID, ok := callerOrg(r)
			if !ok {
				notFound(w, "user")
				return
			}

			// Reading your own record never depends on a membership row.
			if callerID, ok := UserIDFromContext(r.Context()); ok && callerID == targetUserID {
				next.ServeHTTP(w, r)
				return
			}

			roles, err := roleRepo.GetUserRoles(r.Context(), targetUserID, claimOrgID)
			if err != nil || len(roles) == 0 {
				notFound(w, "user")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
