// Command seed fills a development database with a realistic demo tenant.
//
// It is deliberately separate from the server and from migrations: running it
// never changes schema, and the server never runs it. Re-running is safe — the
// demo organization is identified by a fixed slug and rebuilt from scratch.
//
// Analyses are produced by the real AnalyzeTicketUseCase over the real stub
// classifier, so seeded predictions always match what the running code would
// produce. Nothing is faked into ai_analyses by hand.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	iamModel "github.com/Root-Emin/TicketLens/internal/domain/iam/model"
	triageModel "github.com/Root-Emin/TicketLens/internal/domain/triage/model"

	iamDTO "github.com/Root-Emin/TicketLens/internal/application/iam/dto"
	iamUC "github.com/Root-Emin/TicketLens/internal/application/iam/usecase"
	tenantDTO "github.com/Root-Emin/TicketLens/internal/application/tenant/dto"
	tenantUC "github.com/Root-Emin/TicketLens/internal/application/tenant/usecase"
	triageDTO "github.com/Root-Emin/TicketLens/internal/application/triage/dto"
	triageUC "github.com/Root-Emin/TicketLens/internal/application/triage/usecase"

	infraAuth "github.com/Root-Emin/TicketLens/internal/infrastructure/auth"
	stubClassifier "github.com/Root-Emin/TicketLens/internal/infrastructure/classifier/stub"
	pgAudit "github.com/Root-Emin/TicketLens/internal/infrastructure/postgres/audit"
	pgIam "github.com/Root-Emin/TicketLens/internal/infrastructure/postgres/iam"
	pgTenant "github.com/Root-Emin/TicketLens/internal/infrastructure/postgres/tenant"
	pgTriage "github.com/Root-Emin/TicketLens/internal/infrastructure/postgres/triage"

	"github.com/Root-Emin/TicketLens/internal/shared/cache"
	"github.com/Root-Emin/TicketLens/internal/shared/config"
	"github.com/Root-Emin/TicketLens/internal/shared/database"
	"github.com/Root-Emin/TicketLens/internal/shared/events"
	"github.com/Root-Emin/TicketLens/internal/shared/middleware"
)

const (
	demoOrgName   = "Demo Software Inc."
	demoOrgSlug   = "demo"
	demoUserEmail = "demo@ticketlens.dev"
	demoPassword  = "Demo1234!"

	// The support agent. Signing in as the admin shows the staff panel with `*`
	// behind it, which hides every permission mistake: the admin passes checks a
	// real agent would fail. This account holds the `agent` role and nothing
	// else, so the panel can be exercised as the person who actually works the
	// queue.
	demoAgentEmail    = "agent@ticketlens.dev"
	demoAgentFullName = "Selin Aydın"

	// seedRandSeed keeps the run reproducible: the same database always ends up
	// with the same dates, threads and overrides.
	seedRandSeed = 20260801
)

// noopBus swallows domain events.
//
// The seed must not emit ticket.created: a server running against the same
// Kafka topic would consume it and analyze the ticket a second time, producing
// duplicate analyses. The seed calls the analyzer itself, in-process.
type noopBus struct{}

func (noopBus) Publish(context.Context, string, events.Event) error { return nil }
func (noopBus) Subscribe(string, events.Handler)                    {}
func (noopBus) Close() error                                        { return nil }

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "seed failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	db, err := database.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer db.Close()

	rng := rand.New(rand.NewSource(seedRandSeed)) // #nosec G404 -- reproducibility, not security

	fmt.Println("🌱 Preparing TicketLens demo data...")

	// --- Repositories & services -------------------------------------------
	userRepo := pgIam.NewUserRepo(db)
	roleRepo := pgIam.NewRoleRepo(db)
	orgRepo := pgTenant.NewOrgRepo(db)
	auditRepo := pgAudit.NewAuditRepo(db)
	departmentRepo := pgTriage.NewDepartmentRepo(db)
	customerRepo := pgTriage.NewCustomerRepo(db)
	ticketRepo := pgTriage.NewTicketRepo(db)
	messageRepo := pgTriage.NewTicketMessageRepo(db)
	analysisRepo := pgTriage.NewAIAnalysisRepo(db)

	jwtService := infraAuth.NewJWTService(cfg.JWT)
	bus := noopBus{}

	registerUC := iamUC.NewRegisterUseCase(userRepo, jwtService, bus)
	createOrgUC := tenantUC.NewCreateOrgUseCase(orgRepo, roleRepo, departmentRepo, bus)
	createDepartmentUC := triageUC.NewCreateDepartmentUseCase(departmentRepo)
	createCustomerUC := triageUC.NewCreateCustomerUseCase(customerRepo)
	txManager := database.NewTxManager(db)
	createTicketUC := triageUC.NewCreateTicketUseCase(
		ticketRepo, messageRepo, customerRepo, departmentRepo, analysisRepo, userRepo, txManager, bus)
	createMessageUC := triageUC.NewCreateMessageUseCase(ticketRepo, messageRepo)
	analyzeUC := triageUC.NewAnalyzeTicketUseCase(
		ticketRepo, messageRepo, departmentRepo, customerRepo, analysisRepo, userRepo,
		stubClassifier.New(), cfg.Classifier.ReviewThreshold, bus)
	updateTicketUC := triageUC.NewUpdateTicketUseCase(
		ticketRepo, analysisRepo, departmentRepo, customerRepo, messageRepo, userRepo, auditRepo, bus)

	// --- 1. Template roles --------------------------------------------------
	fmt.Println("  roles")
	if err := ensureTemplateRoles(ctx, db); err != nil {
		return err
	}
	// Permission checks are cached in Redis for 15 minutes. After repairing
	// grants, drop those keys so a re-login (or even a still-valid token) sees
	// the new matrix immediately.
	if n, err := flushPermissionCache(ctx, cfg); err != nil {
		fmt.Printf("  ⚠ could not clear permission cache: %v\n", err)
	} else if n > 0 {
		fmt.Printf("  ✓ cleared %d cached permission entries\n", n)
	}

	// --- 2. Wipe any previous demo tenant -----------------------------------
	removed, err := resetDemoOrg(ctx, db)
	if err != nil {
		return err
	}
	if removed {
		fmt.Println("  ↺ previous demo organization cleaned up")
	}

	// --- 3. Demo admin ------------------------------------------------------
	adminID, err := ensureDemoUser(ctx, userRepo, registerUC)
	if err != nil {
		return err
	}

	// CreateOrgUseCase reads the creator from the context, exactly as the HTTP
	// middleware would supply it.
	actorCtx := context.WithValue(ctx, middleware.ContextKeyUserID, adminID)

	// --- 4. Organization ----------------------------------------------------
	org, err := createOrgUC.Execute(actorCtx, tenantDTO.CreateOrgRequest{
		Name: demoOrgName,
		Slug: demoOrgSlug,
	})
	if err != nil {
		return fmt.Errorf("create demo organization: %w", err)
	}
	fmt.Printf("  ✓ organization: %s (%s)\n", org.Name, org.ID)

	// --- 5. Departments -----------------------------------------------------
	// Deliberately fewer departments than the taxonomy has categories: six
	// categories stay unmapped so the fallback path and unmapped_count stay
	// visible in the demo.
	extraDepartments := []struct {
		name     string
		category string
	}{
		{"Technical Support", string(triageModel.CategoryTechnicalIssue)},
		{"Integration Support", string(triageModel.CategoryIntegration)},
		{"Payment Operations", string(triageModel.CategoryPaymentOps)},
		{"Customer Success", string(triageModel.CategoryHowTo)},
	}
	for _, d := range extraDepartments {
		category := d.category
		if _, err := createDepartmentUC.Execute(ctx, org.ID, triageDTO.CreateDepartmentRequest{
			Name:     d.name,
			Category: &category,
		}); err != nil {
			return fmt.Errorf("create department %s: %w", d.name, err)
		}
	}
	fmt.Printf("  ✓ departments: 1 default + %d categorized\n", len(extraDepartments))

	departments, err := departmentRepo.ListByOrg(ctx, org.ID)
	if err != nil {
		return err
	}

	// --- 6. Customers -------------------------------------------------------
	customerIDs := make([]uuid.UUID, 0, len(demoCustomers))
	for _, c := range demoCustomers {
		created, err := createCustomerUC.Execute(ctx, org.ID, triageDTO.CreateCustomerRequest{
			Email:    c.Email,
			FullName: c.FullName,
			Company:  c.Company,
		})
		if err != nil {
			return fmt.Errorf("create customer %s: %w", c.Email, err)
		}
		customerIDs = append(customerIDs, created.ID)
	}
	fmt.Printf("  ✓ customers: %d\n", len(customerIDs))

	// Give the first few customers a login, so the portal has something to sign
	// into. Without this the customer panel demos against an empty account: the
	// seeded customers are records an agent could have typed in, with no way to
	// authenticate as one of them.
	portalLogins, err := seedPortalLogins(ctx, db, userRepo, registerUC, org.ID,
		demoCustomers[:portalLoginCount], customerIDs[:portalLoginCount])
	if err != nil {
		return err
	}
	fmt.Printf("  ✓ portal logins: %d\n", len(portalLogins))

	agentEmail, err := seedAgentLogin(ctx, db, userRepo, registerUC, org.ID)
	if err != nil {
		return err
	}
	fmt.Printf("  ✓ agent login: %s\n", agentEmail)

	// The rest of the support team, placed on departments. Returned grouped by
	// department so the ticket pass below can hand work to somebody who is
	// actually on the team the ticket was routed to.
	agentsByDepartment, err := seedSupportTeam(ctx, db, userRepo, registerUC, org.ID, departments)
	if err != nil {
		return err
	}
	fmt.Printf("  ✓ support team: %d agents across %d departments\n",
		len(demoAgents)+1, len(agentsByDepartment))

	// --- 7. Tickets, analyses, threads --------------------------------------
	seeded, err := seedTickets(ctx, seedDeps{
		db:              db,
		orgID:           org.ID,
		adminID:         adminID,
		customerIDs:     customerIDs,
		departments:     departments,
		createTicketUC:  createTicketUC,
		createMessageUC: createMessageUC,
		analyzeUC:       analyzeUC,
		updateTicketUC:  updateTicketUC,
		rng:             rng,
		logger:          logger,
	})
	if err != nil {
		return err
	}

	fmt.Printf("  ✓ tickets: %d (analyzed: %d, unmapped: %d)\n",
		seeded.total, seeded.analyzed, seeded.unmapped)
	fmt.Printf("  ✓ overrides: %d priority, %d department\n",
		seeded.priorityOverrides, seeded.departmentOverrides)
	fmt.Printf("  ✓ messages: %d\n", seeded.messages)

	// Assignment runs last, over tickets that already exist and have already
	// been analyzed: the classifier decides a department, and only then can the
	// work go to somebody on that department.
	assignedTickets, err := assignTicketsToAgents(ctx, db, org.ID, agentsByDepartment, rng)
	if err != nil {
		return err
	}
	fmt.Printf("  ✓ assigned tickets: %d\n", assignedTickets)

	fmt.Println()
	fmt.Printf("✅ Ready. Every account below uses the password %s\n", demoPassword)
	fmt.Printf("   org: %s (slug: %s)\n", org.Name, demoOrgSlug)
	fmt.Println()
	fmt.Printf("   owner    (role: admin)    %s\n", demoUserEmail)
	fmt.Printf("   staff    (role: agent)    %s\n", agentEmail)
	fmt.Println("   customer (role: customer)")
	for _, email := range portalLogins {
		fmt.Printf("                             %s\n", email)
	}
	fmt.Println()
	fmt.Println("   Sign in again if a browser tab still holds a pre-seed JWT —")
	fmt.Println("   demo re-create mints a new organization id.")
	return nil
}

// flushPermissionCache deletes user:…:org:…:permissions keys so role matrix
// repairs take effect without waiting out the 15-minute RBAC TTL.
func flushPermissionCache(ctx context.Context, cfg *config.Config) (int, error) {
	client, err := cache.NewRedisClient(ctx, cfg.Redis)
	if err != nil {
		// Redis is optional for local dev; skip quietly when not up.
		return 0, nil
	}
	defer client.Close()

	var cursor uint64
	var deleted int
	for {
		keys, next, err := client.Scan(ctx, cursor, "user:*:org:*:permissions", 200).Result()
		if err != nil {
			return deleted, err
		}
		if len(keys) > 0 {
			n, err := client.Del(ctx, keys...).Result()
			if err != nil {
				return deleted, err
			}
			deleted += int(n)
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return deleted, nil
}

// templateRoleDefs is the product's RBAC matrix.
//
// Kept in lockstep with frontend ROLE_PERMISSIONS
// (frontend/src/lib/auth/permissions.ts) and api-contract.md §2.
//
// Who can do what on workforce:
//
//	admin / org_admin  — full departments + staff (manage + assign)
//	viewer             — read-only (departments via ticket:read or *:read, staff via user:read)
//	agent              — queue work only: ticket:read lists departments for routing,
//	                     but no department:manage and no user:read/user:write
//	customer           — portal only
func templateRoleDefs() []struct {
	name        string
	description string
	permissions []string
} {
	return []struct {
		name        string
		description string
		permissions []string
	}{
		{"admin", "Full system administrator", []string{"*"}},

		// Contract "Owner". Not a second superuser: org-scoped operations without
		// a bare "*". user:* covers the staff roster (GET /staff, PUT assignment);
		// department:manage and the ticket/stats grants cover the owner workspace.
		{"org_admin", "Organization administrator", []string{
			"org:*",
			"app:*",
			"user:*",
			"ticket:create",
			"ticket:read",
			"ticket:update",
			"ticket:delete",
			"ticket:assign",
			"message:create",
			"department:manage",
			"customer:manage",
			"analysis:read",
			"stats:read",
		}},

		{"app_admin", "Application administrator", []string{"app:*", "endpoint:*"}},
		{"developer", "Developer with read/write access", []string{"endpoint:read", "endpoint:write"}},
		{"viewer", "Read-only access", []string{"*:read"}},

		// Support staff. Spelled out rather than "ticket:*" so ticket:read_own /
		// ticket:reopen_own never land here by accident. No department:manage:
		// agents route work from the queue, they do not rewrite the org chart.
		// No user:read / user:write: no colleague directory or reassignment API.
		{"agent", "Support agent", []string{
			"ticket:create",
			"ticket:read",
			"ticket:update",
			"ticket:assign",
			"message:create",
			"customer:manage",
			"analysis:read",
		}},

		// The portal role.
		//
		// Note what is absent. No ticket:read — that is the organization-wide
		// read, and its absence is exactly what narrows this account to its own
		// tickets (see usecase.ResolveScopeUseCase, which tests for it rather
		// than for a role name). No ticket:update either: a customer who could
		// set priority would make triage meaningless, so reopening gets its own
		// narrow grant.
		{"customer", "Portal customer", []string{
			"ticket:create",
			"ticket:read_own",
			"ticket:reopen_own",
			"message:create",
		}},
	}
}

// ensureTemplateRoles creates (or repairs) the template roles every new
// organization is cloned from, and re-applies the same permission set to every
// org-scoped role of the same name.
//
// Re-running seed must fix grants, not only add them: permission rows used to
// be insert-only, so a removed grant (for example agent + department:manage)
// would stick forever and a missing org_admin triage grant would never appear
// on orgs created before the matrix was expanded.
func ensureTemplateRoles(ctx context.Context, db *pgxpool.Pool) error {
	for _, r := range templateRoleDefs() {
		// Upsert the template so RETURNING always yields a real id.
		var templateID uuid.UUID
		err := db.QueryRow(ctx, `
			INSERT INTO roles (id, scope_type, scope_id, name, description, created_at, updated_at)
			VALUES ($1, 'organization', $2, $3, $4, NOW(), NOW())
			ON CONFLICT (scope_type, scope_id, name)
			DO UPDATE SET description = EXCLUDED.description, updated_at = NOW()
			RETURNING id
		`, uuid.New(), iamModel.TemplateScopeID, r.name, r.description).Scan(&templateID)
		if err != nil {
			return fmt.Errorf("seed role %s: %w", r.name, err)
		}

		// Every role with this name — template plus each organization's clone —
		// gets the same description and exact permission set.
		rows, err := db.Query(ctx, `
			SELECT id FROM roles WHERE name = $1
		`, r.name)
		if err != nil {
			return fmt.Errorf("list roles named %s: %w", r.name, err)
		}

		var roleIDs []uuid.UUID
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return fmt.Errorf("scan role %s: %w", r.name, err)
			}
			roleIDs = append(roleIDs, id)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate roles named %s: %w", r.name, err)
		}

		for _, roleID := range roleIDs {
			if _, err := db.Exec(ctx, `
				UPDATE roles SET description = $2, updated_at = NOW() WHERE id = $1
			`, roleID, r.description); err != nil {
				return fmt.Errorf("update role %s (%s): %w", r.name, roleID, err)
			}
			if err := replaceRolePermissions(ctx, db, roleID, r.permissions); err != nil {
				return fmt.Errorf("sync permissions for role %s (%s): %w", r.name, roleID, err)
			}
		}

		_ = templateID // ensured above; clones already covered by the name query
		fmt.Printf("  ✓ role %s (%d instances, %d permissions)\n",
			r.name, len(roleIDs), len(r.permissions))
	}
	return nil
}

// replaceRolePermissions makes the role's grants exactly the desired set:
// missing rows are inserted, extras are deleted. Additive-only seed left
// obsolete permissions (and missing new ones) stuck across re-runs.
func replaceRolePermissions(ctx context.Context, db *pgxpool.Pool, roleID uuid.UUID, desired []string) error {
	rows, err := db.Query(ctx, `
		SELECT permission FROM role_permissions WHERE role_id = $1
	`, roleID)
	if err != nil {
		return err
	}
	current := make(map[string]struct{})
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			rows.Close()
			return err
		}
		current[p] = struct{}{}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	want := make(map[string]struct{}, len(desired))
	for _, p := range desired {
		want[p] = struct{}{}
		if _, ok := current[p]; ok {
			continue
		}
		if _, err := db.Exec(ctx, `
			INSERT INTO role_permissions (role_id, permission, created_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (role_id, permission) DO NOTHING
		`, roleID, p); err != nil {
			return err
		}
	}

	for p := range current {
		if _, ok := want[p]; ok {
			continue
		}
		if _, err := db.Exec(ctx, `
			DELETE FROM role_permissions WHERE role_id = $1 AND permission = $2
		`, roleID, p); err != nil {
			return err
		}
	}
	return nil
}

// resetDemoOrg removes a previously seeded demo tenant so a re-run does not
// double the data. Deleting the organization cascades to departments,
// customers, tickets, messages, analyses and role assignments; the per-org
// roles have no foreign key to it and are removed explicitly.
func resetDemoOrg(ctx context.Context, db *pgxpool.Pool) (bool, error) {
	var orgID uuid.UUID
	err := db.QueryRow(ctx, `SELECT id FROM organizations WHERE slug = $1`, demoOrgSlug).Scan(&orgID)
	if err != nil {
		return false, nil // nothing seeded yet
	}

	if _, err := db.Exec(ctx,
		`DELETE FROM roles WHERE scope_type = 'organization' AND scope_id = $1`, orgID); err != nil {
		return false, fmt.Errorf("delete demo roles: %w", err)
	}
	if _, err := db.Exec(ctx, `DELETE FROM organizations WHERE id = $1`, orgID); err != nil {
		return false, fmt.Errorf("delete demo organization: %w", err)
	}
	return true, nil
}

// portalLoginCount is how many of the seeded customers get an account they can
// sign into. Three is enough to demonstrate that one customer cannot see
// another's tickets, which is the property worth demonstrating; giving all
// fifteen a login would only add rows.
const portalLoginCount = 3

// seedPortalLogins gives demo customers an account to sign in with.
//
// Three things have to line up before "my tickets" can answer anything, and all
// three happen here:
//
//  1. a users row, so there is something to authenticate;
//  2. the `customer` role in this organization, so the token carries
//     ticket:read_own and *not* ticket:read — the absence is what narrows the
//     scope;
//  3. customers.user_id, the link that says which customer this login is.
//
// Skipping the third leaves an account that authenticates fine and then reads
// as "not linked to a customer record" on every portal call.
func seedPortalLogins(
	ctx context.Context,
	db *pgxpool.Pool,
	userRepo *pgIam.UserRepo,
	registerUC *iamUC.RegisterUseCase,
	orgID uuid.UUID,
	customers []customerSeed,
	customerIDs []uuid.UUID,
) ([]string, error) {
	customerRoleID, err := roleIDByName(ctx, db, orgID, "customer")
	if err != nil {
		return nil, err
	}

	emails := make([]string, 0, len(customers))
	for i, c := range customers {
		userID, err := ensureUser(ctx, userRepo, registerUC, c.Email, c.FullName)
		if err != nil {
			return nil, err
		}

		if err := grantRole(ctx, db, userID, customerRoleID, orgID); err != nil {
			return nil, fmt.Errorf("assign customer role to %s: %w", c.Email, err)
		}

		if _, err := db.Exec(ctx,
			`UPDATE customers SET user_id = $1 WHERE id = $2 AND organization_id = $3`,
			userID, customerIDs[i], orgID,
		); err != nil {
			return nil, fmt.Errorf("link customer %s to login: %w", c.Email, err)
		}

		emails = append(emails, c.Email)
	}
	return emails, nil
}

// roleIDByName looks up one of the organization's roles. The roles were cloned
// from the templates when the org was created, so this is where a seeded
// account and its permissions meet.
func roleIDByName(
	ctx context.Context,
	db *pgxpool.Pool,
	orgID uuid.UUID,
	name string,
) (uuid.UUID, error) {
	var roleID uuid.UUID
	if err := db.QueryRow(ctx,
		`SELECT id FROM roles WHERE scope_type = 'organization' AND scope_id = $1 AND name = $2`,
		orgID, name,
	).Scan(&roleID); err != nil {
		return uuid.Nil, fmt.Errorf("find %s role: %w", name, err)
	}
	return roleID, nil
}

// grantRole assigns a role to a user, ignoring a grant that is already there so
// a re-run stays idempotent.
func grantRole(
	ctx context.Context,
	db *pgxpool.Pool,
	userID, roleID, orgID uuid.UUID,
) error {
	_, err := db.Exec(ctx, `
		INSERT INTO user_roles (id, user_id, role_id, organization_id, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (user_id, role_id, organization_id, app_id) DO NOTHING
	`, uuid.New(), userID, roleID, orgID)
	return err
}

// seedAgentLogin gives the demo tenant one support-agent account.
//
// Unlike the customers this account has no `customers` row: an agent is staff,
// not somebody who raises tickets, so nothing links it to a customer record and
// ResolveScope leaves it reading the whole queue.
func seedAgentLogin(
	ctx context.Context,
	db *pgxpool.Pool,
	userRepo *pgIam.UserRepo,
	registerUC *iamUC.RegisterUseCase,
	orgID uuid.UUID,
) (string, error) {
	agentRoleID, err := roleIDByName(ctx, db, orgID, "agent")
	if err != nil {
		return "", err
	}

	userID, err := ensureUser(ctx, userRepo, registerUC, demoAgentEmail, demoAgentFullName)
	if err != nil {
		return "", err
	}

	if err := grantRole(ctx, db, userID, agentRoleID, orgID); err != nil {
		return "", fmt.Errorf("assign agent role to %s: %w", demoAgentEmail, err)
	}
	return demoAgentEmail, nil
}

// ensureUser registers a seeded account, reusing one that already exists so a
// re-run is idempotent. Names are split off the seed's full name, which is the
// only place a first and last name exist.
func ensureUser(
	ctx context.Context,
	userRepo *pgIam.UserRepo,
	registerUC *iamUC.RegisterUseCase,
	email, fullName string,
) (uuid.UUID, error) {
	if existing, err := userRepo.GetByEmail(ctx, email); err == nil && existing != nil {
		return existing.ID, nil
	}

	first, last, _ := strings.Cut(fullName, " ")
	if last == "" {
		last = first
	}

	created, err := registerUC.Execute(ctx, iamDTO.RegisterRequest{
		Email:     email,
		Password:  demoPassword,
		FirstName: first,
		LastName:  last,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("create login %s: %w", email, err)
	}
	return created.ID, nil
}

// ensureDemoUser returns the demo admin, registering them on first run.
func ensureDemoUser(ctx context.Context, userRepo *pgIam.UserRepo, registerUC *iamUC.RegisterUseCase) (uuid.UUID, error) {
	if existing, err := userRepo.GetByEmail(ctx, demoUserEmail); err == nil && existing != nil {
		return existing.ID, nil
	}

	created, err := registerUC.Execute(ctx, iamDTO.RegisterRequest{
		Email:     demoUserEmail,
		Password:  demoPassword,
		FirstName: "Demo",
		LastName:  "Admin",
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("create demo user: %w", err)
	}
	return created.ID, nil
}

func init() {
	log.SetFlags(0)
}
