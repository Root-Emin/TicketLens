package router

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	// Handlers
	apimgmtHandler "github.com/Root-Emin/TicketLens/internal/infrastructure/http/handler/apimanagement"
	auditHandler "github.com/Root-Emin/TicketLens/internal/infrastructure/http/handler/audit"
	"github.com/Root-Emin/TicketLens/internal/infrastructure/http/handler/health"
	iamHandler "github.com/Root-Emin/TicketLens/internal/infrastructure/http/handler/iam"
	realtimeHandler "github.com/Root-Emin/TicketLens/internal/infrastructure/http/handler/realtime"
	tenantHandler "github.com/Root-Emin/TicketLens/internal/infrastructure/http/handler/tenant"
	triageHandler "github.com/Root-Emin/TicketLens/internal/infrastructure/http/handler/triage"

	// Services & middleware
	iamService "github.com/Root-Emin/TicketLens/internal/domain/iam/service"
	"github.com/Root-Emin/TicketLens/internal/gateway"
	"github.com/Root-Emin/TicketLens/internal/shared/middleware"

	// Repositories (for tenant resolver and scope middleware)
	apimgmtRepo "github.com/Root-Emin/TicketLens/internal/domain/apimanagement/repository"
	iamRepo "github.com/Root-Emin/TicketLens/internal/domain/iam/repository"
	tenantRepo "github.com/Root-Emin/TicketLens/internal/domain/tenant/repository"
)

func maybeRequirePermission(rbac iamService.RBACService, permission string) func(http.Handler) http.Handler {
	if rbac == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	return middleware.RequirePermission(rbac, permission)
}

func maybeRequireAnyPermission(rbac iamService.RBACService, permissions ...string) func(http.Handler) http.Handler {
	if rbac == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	return middleware.RequireAnyPermission(rbac, permissions...)
}

// Dependencies holds all injected dependencies for the router.
type Dependencies struct {
	Logger *slog.Logger
	DB     *pgxpool.Pool
	Redis  *redis.Client

	CORSAllowedOrigins []string
	MaxBodyBytes       int64

	// Services
	AuthService iamService.AuthService
	RBACService iamService.RBACService

	// Handlers
	IAMHandler      *iamHandler.Handler
	TenantHandler   *tenantHandler.Handler
	APIMgmtHandler  *apimgmtHandler.Handler
	AuditHandler    *auditHandler.Handler
	RealtimeHandler *realtimeHandler.Handler
	TriageHandler   *triageHandler.Handler

	// Gateway
	GatewayPipeline *gateway.Pipeline

	// Repos needed for middleware
	OrgRepo       tenantRepo.OrgRepository
	WorkspaceRepo tenantRepo.WorkspaceRepository
	AppRepo       tenantRepo.AppRepository
	APIKeyRepo    tenantRepo.APIKeyRepository
	RoleRepo      iamRepo.RoleRepository
	EndpointRepo  apimgmtRepo.EndpointRepository
}

// requireOrgFromPath guards a path-addressed organization, degrading to a no-op
// when auth is disabled (no AuthService) since there is no claim to compare.
func maybeRequireOrgFromPath(deps Dependencies, param string) func(http.Handler) http.Handler {
	if deps.AuthService == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	return middleware.RequireOrgFromPath(param)
}

func maybeRequireAppInOrg(deps Dependencies, param string) func(http.Handler) http.Handler {
	if deps.AuthService == nil || deps.AppRepo == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	return middleware.RequireAppInOrg(deps.AppRepo, param)
}

func maybeRequireUserInOrg(deps Dependencies, param string) func(http.Handler) http.Handler {
	if deps.AuthService == nil || deps.RoleRepo == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	return middleware.RequireUserInOrg(deps.RoleRepo, param)
}

// maybeRequireKeyInApp keeps an API key id from being paired with somebody
// else's app id in the path.
func maybeRequireKeyInApp(deps Dependencies) func(http.Handler) http.Handler {
	if deps.AuthService == nil || deps.APIKeyRepo == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	return middleware.RequireChildOfPathResource(
		func(ctx context.Context, keyID uuid.UUID) (uuid.UUID, error) {
			key, err := deps.APIKeyRepo.GetByID(ctx, keyID)
			if err != nil {
				return uuid.Nil, err
			}
			return key.AppID, nil
		},
		"appId", "keyId", "api key",
	)
}

// maybeRequireEndpointInApp does the same for endpoint ids, which also carry
// their policies.
func maybeRequireEndpointInApp(deps Dependencies) func(http.Handler) http.Handler {
	if deps.AuthService == nil || deps.EndpointRepo == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	return middleware.RequireChildOfPathResource(
		func(ctx context.Context, endpointID uuid.UUID) (uuid.UUID, error) {
			endpoint, err := deps.EndpointRepo.GetByID(ctx, endpointID)
			if err != nil {
				return uuid.Nil, err
			}
			return endpoint.AppID, nil
		},
		"appId", "endpointId", "endpoint",
	)
}

// New creates the root Chi router with all middleware and routes.
func New(deps Dependencies) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Logging(deps.Logger))
	r.Use(middleware.Recoverer(deps.Logger))
	if deps.MaxBodyBytes > 0 {
		r.Use(middleware.MaxBodyBytes(deps.MaxBodyBytes))
	}
	r.Use(cors.Handler(middleware.CORSOptions(deps.CORSAllowedOrigins)))

	// Health endpoints
	healthHandler := health.NewHandler(deps.DB, deps.Redis)
	r.Get("/health/live", healthHandler.Liveness)
	r.Get("/health/ready", healthHandler.Readiness)

	// Prometheus metrics
	r.Handle("/metrics", promhttp.Handler())

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public auth routes (no JWT required)
		r.Route("/auth", func(r chi.Router) {
			if deps.IAMHandler != nil {
				r.Post("/register", deps.IAMHandler.Register)
				r.Post("/login", deps.IAMHandler.Login)
			}
		})

		// Protected routes (require JWT)
		r.Group(func(r chi.Router) {
			if deps.AuthService != nil {
				r.Use(middleware.JWTAuth(deps.AuthService))
			}

			// Tenant resolution middleware (with workspace support)
			if deps.OrgRepo != nil {
				// Note: WorkspaceRepo can be nil - workspace resolution is optional
				r.Use(middleware.TenantResolverWithWorkspace(deps.OrgRepo, deps.WorkspaceRepo))
			}

			// Gateway pipeline (rate limiting, permission enforcement for managed endpoints)
			// Must be applied before specific routes so it can handle dynamic endpoints.
			// chi requires every Use() to precede the first route on this mux, so this
			// stays above /ws. WebSocket upgrades are not proxied: shouldSkipPipeline
			// short-circuits the /api/v1/ws prefix.
			if deps.GatewayPipeline != nil {
				r.Use(deps.GatewayPipeline.Enforce)
			}

			// WebSocket endpoint
			if deps.RealtimeHandler != nil {
				r.Get("/ws", deps.RealtimeHandler.Connect)
			}

			// User routes
			if deps.IAMHandler != nil {
				r.Get("/me", deps.IAMHandler.GetMe)
				r.With(maybeRequirePermission(deps.RBACService, "user:read")).Route("/users", func(r chi.Router) {
					r.Get("/", deps.IAMHandler.ListUsers)
					// A single user is only readable through a membership in the
					// caller's organization; the handler looks users up by id
					// alone and would otherwise serve any account on the platform.
					r.With(maybeRequireUserInOrg(deps, "id")).Get("/{id}", deps.IAMHandler.GetUser)
				})
				r.With(maybeRequirePermission(deps.RBACService, "user:write")).Post("/roles/assign", deps.IAMHandler.AssignRole)
			}

			// Organization routes
			if deps.TenantHandler != nil {
				r.Route("/organizations", func(r chi.Router) {
					// Self-signup: any authenticated user may create an organization
					// and becomes its owner. Requiring org:write here would be
					// unsatisfiable — a user with no organization has no permissions.
					r.Post("/", deps.TenantHandler.CreateOrg)
					r.With(maybeRequirePermission(deps.RBACService, "org:read")).Get("/", deps.TenantHandler.ListOrgs)
					r.Route("/{orgId}", func(r chi.Router) {
						// A permission is granted within an organization, so it
						// cannot by itself justify reading the organization named
						// in the path. Everything below is the caller's own org.
						r.Use(maybeRequireOrgFromPath(deps, "orgId"))

						r.With(maybeRequirePermission(deps.RBACService, "org:read")).Get("/", deps.TenantHandler.GetOrg)

						// Apps under organization
						r.Route("/apps", func(r chi.Router) {
							r.With(maybeRequirePermission(deps.RBACService, "app:write")).Post("/", deps.TenantHandler.CreateApp)
							r.With(maybeRequirePermission(deps.RBACService, "app:read")).Get("/", deps.TenantHandler.ListApps)
							r.Route("/{appId}", func(r chi.Router) {
								// Guards the API key and endpoint subtrees too:
								// those handlers resolve by app id alone.
								r.Use(maybeRequireAppInOrg(deps, "appId"))

								r.With(maybeRequirePermission(deps.RBACService, "app:read")).Get("/", deps.TenantHandler.GetApp)

								// API keys under app
								r.Route("/keys", func(r chi.Router) {
									r.With(maybeRequirePermission(deps.RBACService, "app:write")).Post("/", deps.TenantHandler.CreateAPIKey)
									r.With(maybeRequirePermission(deps.RBACService, "app:read")).Get("/", deps.TenantHandler.ListAPIKeys)
									// RevokeAPIKey resolves the key by id alone, so the
									// key must be confirmed to belong to this app.
									r.With(maybeRequirePermission(deps.RBACService, "app:write")).
										With(maybeRequireKeyInApp(deps)).
										Delete("/{keyId}", deps.TenantHandler.RevokeAPIKey)
								})

								// Endpoints under app
								if deps.APIMgmtHandler != nil {
									r.Route("/endpoints", func(r chi.Router) {
										r.With(maybeRequirePermission(deps.RBACService, "endpoint:write")).Post("/", deps.APIMgmtHandler.DefineEndpoint)
										r.With(maybeRequirePermission(deps.RBACService, "endpoint:read")).Get("/", deps.APIMgmtHandler.ListEndpoints)
										r.Route("/{endpointId}", func(r chi.Router) {
											// Every handler below resolves the endpoint
											// (or its policy) by endpoint id alone.
											r.Use(maybeRequireEndpointInApp(deps))

											r.With(maybeRequirePermission(deps.RBACService, "endpoint:read")).Get("/", deps.APIMgmtHandler.GetEndpoint)
											r.With(maybeRequirePermission(deps.RBACService, "endpoint:write")).Post("/retire", deps.APIMgmtHandler.RetireEndpoint)
											r.With(maybeRequirePermission(deps.RBACService, "endpoint:write")).Post("/activate", deps.APIMgmtHandler.ActivateEndpoint)
											r.With(maybeRequirePermission(deps.RBACService, "endpoint:write")).Put("/policy", deps.APIMgmtHandler.UpdatePolicy)
											r.With(maybeRequirePermission(deps.RBACService, "endpoint:read")).Get("/policy", deps.APIMgmtHandler.GetPolicy)
										})
									})
								}
							})
						})

						// Workspaces under organization
						r.Route("/workspaces", func(r chi.Router) {
							r.With(maybeRequirePermission(deps.RBACService, "org:write")).Post("/", deps.TenantHandler.CreateWorkspace)
							r.With(maybeRequirePermission(deps.RBACService, "org:read")).Get("/", deps.TenantHandler.ListWorkspaces)
							r.Route("/{workspaceId}", func(r chi.Router) {
								r.With(maybeRequirePermission(deps.RBACService, "org:write")).Put("/", deps.TenantHandler.UpdateWorkspace)
							})
						})

						// Audit logs under organization
						if deps.AuditHandler != nil {
							r.With(maybeRequirePermission(deps.RBACService, "org:read")).Get("/audit-logs", deps.AuditHandler.ListByOrg)
						}
					})
				})
			}

			// Audit logs by user
			if deps.AuditHandler != nil {
				r.With(maybeRequirePermission(deps.RBACService, "org:read")).
					With(maybeRequireUserInOrg(deps, "userId")).
					Get("/users/{userId}/audit-logs", deps.AuditHandler.ListByUser)
			}

			// Ticketing (triage). Every route resolves its organization from the
			// JWT claims, never from the path, so these are flat rather than
			// nested under /organizations/{orgId}.
			if deps.TriageHandler != nil {
				r.Route("/departments", func(r chi.Router) {
					r.With(maybeRequireAnyPermission(deps.RBACService, "department:manage", "ticket:read")).
						Get("/", deps.TriageHandler.ListDepartments)
					r.With(maybeRequirePermission(deps.RBACService, "department:manage")).
						Post("/", deps.TriageHandler.CreateDepartment)
					r.With(maybeRequirePermission(deps.RBACService, "department:manage")).
						Patch("/{departmentId}", deps.TriageHandler.UpdateDepartment)
					r.With(maybeRequirePermission(deps.RBACService, "department:manage")).
						Delete("/{departmentId}", deps.TriageHandler.DeleteDepartment)
				})

				r.Route("/customers", func(r chi.Router) {
					r.Use(maybeRequirePermission(deps.RBACService, "customer:manage"))
					r.Get("/", deps.TriageHandler.ListCustomers)
					r.Post("/", deps.TriageHandler.CreateCustomer)
					r.Get("/{customerId}", deps.TriageHandler.GetCustomer)
				})

				r.Route("/tickets", func(r chi.Router) {
					r.With(maybeRequirePermission(deps.RBACService, "ticket:create")).
						Post("/", deps.TriageHandler.CreateTicket)
					r.With(maybeRequireAnyPermission(deps.RBACService, "ticket:read", "ticket:read_own")).
						Get("/", deps.TriageHandler.ListTickets)

					r.Route("/{ticketId}", func(r chi.Router) {
						r.With(maybeRequireAnyPermission(deps.RBACService, "ticket:read", "ticket:read_own")).
							Get("/", deps.TriageHandler.GetTicket)
						r.With(maybeRequirePermission(deps.RBACService, "ticket:update")).
							Patch("/", deps.TriageHandler.UpdateTicket)
						r.With(maybeRequirePermission(deps.RBACService, "ticket:assign")).
							Post("/assign", deps.TriageHandler.AssignTicket)

						r.With(maybeRequireAnyPermission(deps.RBACService, "ticket:read", "ticket:read_own")).
							Get("/messages", deps.TriageHandler.ListMessages)
						r.With(maybeRequirePermission(deps.RBACService, "message:create")).
							Post("/messages", deps.TriageHandler.CreateMessage)

						r.With(maybeRequirePermission(deps.RBACService, "analysis:read")).
							Get("/analyses", deps.TriageHandler.ListAnalyses)
						// Re-running classification needs both permissions.
						r.With(maybeRequirePermission(deps.RBACService, "analysis:read")).
							With(maybeRequirePermission(deps.RBACService, "ticket:update")).
							Post("/analyses", deps.TriageHandler.RerunAnalysis)
					})
				})

				r.With(maybeRequirePermission(deps.RBACService, "stats:read")).
					Get("/stats/overview", deps.TriageHandler.StatsOverview)
			}

			// Catch-all handler for managed endpoints (must be last in the group)
			// This allows the gateway pipeline to handle dynamic endpoints like /api/v1/products
			// The gateway middleware will validate and return responses for managed endpoints
			r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
				// Gateway middleware should have already handled this if it's a managed endpoint
				// If we reach here, it means no endpoint was found, return 404
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"endpoint not found","code":404,"message":"No endpoint registered for this path. Define the endpoint first using POST /api/v1/organizations/{orgId}/apps/{appId}/endpoints"}`))
			})
		})
	})

	// Catch-all handler for managed endpoints (must be after all specific routes)
	// This allows the gateway pipeline to handle dynamic endpoints like /api/v1/products
	// The gateway middleware will validate and return responses for managed endpoints
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		// If this is an API v1 path, let the gateway handle it (if it hasn't already)
		// Otherwise return 404
		if !strings.HasPrefix(r.URL.Path, "/api/v1") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found","code":404}`))
			return
		}

		// For /api/v1 paths, check if gateway pipeline already handled it
		// If not, return 404 (gateway would have returned response if endpoint existed)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"endpoint not found","code":404,"message":"No endpoint registered for this path. Define the endpoint first."}`))
	})

	return r
}
