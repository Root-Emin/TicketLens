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
	if err := ensureTemplateRoles(ctx, db); err != nil {
		return err
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

	fmt.Println()
	fmt.Println("✅ Ready. Demo login:")
	fmt.Printf("   email    : %s\n", demoUserEmail)
	fmt.Printf("   password : %s\n", demoPassword)
	fmt.Printf("   org      : %s (slug: %s)\n", org.Name, demoOrgSlug)
	return nil
}

// ensureTemplateRoles creates the scope-less roles every new organization is
// cloned from. It is idempotent: an existing role is left in place and its ID
// is reused for the permission rows.
func ensureTemplateRoles(ctx context.Context, db *pgxpool.Pool) error {
	roles := []struct {
		name        string
		description string
		permissions []string
	}{
		{"admin", "Full system administrator", []string{"*"}},
		{"org_admin", "Organization administrator", []string{"org:*", "app:*", "user:*"}},
		{"app_admin", "Application administrator", []string{"app:*", "endpoint:*"}},
		{"developer", "Developer with read/write access", []string{"endpoint:read", "endpoint:write"}},
		{"viewer", "Read-only access", []string{"*:read"}},
	}

	for _, r := range roles {
		var roleID uuid.UUID
		// DO UPDATE (rather than DO NOTHING) so RETURNING yields the existing row
		// on a repeat run; otherwise the permission inserts below would reference
		// an id that was never stored.
		err := db.QueryRow(ctx, `
			INSERT INTO roles (id, scope_type, scope_id, name, description, created_at, updated_at)
			VALUES ($1, 'organization', $2, $3, $4, NOW(), NOW())
			ON CONFLICT (scope_type, scope_id, name)
			DO UPDATE SET description = EXCLUDED.description, updated_at = NOW()
			RETURNING id
		`, uuid.New(), iamModel.TemplateScopeID, r.name, r.description).Scan(&roleID)
		if err != nil {
			return fmt.Errorf("seed role %s: %w", r.name, err)
		}

		for _, permission := range r.permissions {
			if _, err := db.Exec(ctx, `
				INSERT INTO role_permissions (role_id, permission, created_at)
				VALUES ($1, $2, NOW())
				ON CONFLICT (role_id, permission) DO NOTHING
			`, roleID, permission); err != nil {
				return fmt.Errorf("seed permission %s: %w", permission, err)
			}
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
