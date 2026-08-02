package middleware

import (
	"context"
	"net/http"
	"strings"

	tenantRepo "github.com/Root-Emin/TicketLens/internal/domain/tenant/repository"
	"github.com/Root-Emin/TicketLens/internal/shared/logger"
	"github.com/Root-Emin/TicketLens/internal/shared/response"
	"github.com/google/uuid"
)

const (
	ContextKeyTenantID    contextKey = "tenant_id"
	ContextKeyWorkspaceID contextKey = "workspace_id"
	ContextKeyAppID       contextKey = "tenant_app_id"
)

// TenantResolver resolves the tenant (organization) and optionally workspace from the request.
// Resolution order: JWT claims > subdomain. X-Organization-ID may only confirm
// the JWT claim, never replace it.
// Workspace resolution: X-Workspace-ID header > X-Workspace-Slug header (requires org context).
func TenantResolver(orgRepo tenantRepo.OrgRepository) func(http.Handler) http.Handler {
	return TenantResolverWithWorkspace(orgRepo, nil)
}

// TenantResolverWithWorkspace resolves tenant and workspace from the request.
//
// The JWT claim is authoritative. X-Organization-ID used to be checked first,
// which let any authenticated caller adopt another tenant's context simply by
// naming its id; the header is now only honored when it agrees with the token,
// where it serves as an explicitness aid rather than a privilege.
func TenantResolverWithWorkspace(orgRepo tenantRepo.OrgRepository, workspaceRepo tenantRepo.WorkspaceRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			var orgID uuid.UUID

			// 1. JWT claims (if auth middleware already ran)
			if claimOrgID, ok := ctx.Value(ContextKeyOrganizationID).(uuid.UUID); ok && claimOrgID != uuid.Nil {
				orgID = claimOrgID
			}

			// 2. An explicit header must agree with the token.
			if header := r.Header.Get("X-Organization-ID"); header != "" {
				parsed, err := uuid.Parse(header)
				if err != nil {
					response.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid X-Organization-ID"})
					return
				}
				if orgID != uuid.Nil && parsed != orgID {
					response.JSON(w, http.StatusForbidden,
						map[string]string{"error": "X-Organization-ID does not match the authenticated organization"})
					return
				}
			}

			// 3. Fall back to subdomain
			if orgID == uuid.Nil {
				host := r.Host
				parts := strings.Split(host, ".")
				if len(parts) > 2 {
					slug := parts[0]
					if orgRepo != nil {
						org, err := orgRepo.GetBySlug(ctx, slug)
						if err == nil && org != nil {
							orgID = org.ID
						}
					}
				}
			}

			if orgID != uuid.Nil {
				ctx = context.WithValue(ctx, ContextKeyTenantID, orgID)
				ctx = logger.ContextWithOrganizationID(ctx, orgID.String())

				// Resolve workspace if workspace repo is provided
				var workspaceID uuid.UUID
				if workspaceRepo != nil {
					// 1. Check explicit header
					if header := r.Header.Get("X-Workspace-ID"); header != "" {
						parsed, err := uuid.Parse(header)
						if err == nil {
							workspaceID = parsed
						}
					}

					// 2. Check workspace slug header (requires org context)
					if workspaceID == uuid.Nil {
						if slug := r.Header.Get("X-Workspace-Slug"); slug != "" {
							if ws, err := workspaceRepo.GetBySlug(ctx, orgID, slug); err == nil && ws != nil {
								workspaceID = ws.ID
							}
						}
					}

					if workspaceID != uuid.Nil {
						ctx = context.WithValue(ctx, ContextKeyWorkspaceID, workspaceID)
					}
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireTenant ensures a tenant ID is present in the context.
func RequireTenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := TenantIDFromContext(r.Context()); !ok {
			response.JSON(w, http.StatusBadRequest, map[string]string{"error": "tenant (organization) not resolved"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// TenantIDFromContext extracts the tenant ID from context.
func TenantIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(ContextKeyTenantID).(uuid.UUID)
	return id, ok && id != uuid.Nil
}

// WorkspaceIDFromContext extracts the workspace ID from context.
func WorkspaceIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(ContextKeyWorkspaceID).(uuid.UUID)
	return id, ok && id != uuid.Nil
}
