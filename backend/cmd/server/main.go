package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	// Infrastructure
	iamPort "github.com/Root-Emin/TicketLens/internal/domain/iam/port"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/port"
	infraAuth "github.com/Root-Emin/TicketLens/internal/infrastructure/auth"
	httpClassifier "github.com/Root-Emin/TicketLens/internal/infrastructure/classifier/http"
	stubClassifier "github.com/Root-Emin/TicketLens/internal/infrastructure/classifier/stub"
	apimgmtHandler "github.com/Root-Emin/TicketLens/internal/infrastructure/http/handler/apimanagement"
	auditHandler "github.com/Root-Emin/TicketLens/internal/infrastructure/http/handler/audit"
	iamHandler "github.com/Root-Emin/TicketLens/internal/infrastructure/http/handler/iam"
	realtimeHandler "github.com/Root-Emin/TicketLens/internal/infrastructure/http/handler/realtime"
	tenantHandler "github.com/Root-Emin/TicketLens/internal/infrastructure/http/handler/tenant"
	triageHandler "github.com/Root-Emin/TicketLens/internal/infrastructure/http/handler/triage"
	"github.com/Root-Emin/TicketLens/internal/infrastructure/http/router"
	infraKafka "github.com/Root-Emin/TicketLens/internal/infrastructure/kafka"
	"github.com/Root-Emin/TicketLens/internal/infrastructure/mail"
	"github.com/Root-Emin/TicketLens/internal/infrastructure/postgres"
	pgApimgmt "github.com/Root-Emin/TicketLens/internal/infrastructure/postgres/apimanagement"
	pgAudit "github.com/Root-Emin/TicketLens/internal/infrastructure/postgres/audit"
	pgIam "github.com/Root-Emin/TicketLens/internal/infrastructure/postgres/iam"
	pgTenant "github.com/Root-Emin/TicketLens/internal/infrastructure/postgres/tenant"
	pgTriage "github.com/Root-Emin/TicketLens/internal/infrastructure/postgres/triage"
	infraTriage "github.com/Root-Emin/TicketLens/internal/infrastructure/triage"
	infraWS "github.com/Root-Emin/TicketLens/internal/infrastructure/websocket"

	// Application use cases
	apimgmtUC "github.com/Root-Emin/TicketLens/internal/application/apimanagement/usecase"
	iamUC "github.com/Root-Emin/TicketLens/internal/application/iam/usecase"
	realtimeUC "github.com/Root-Emin/TicketLens/internal/application/realtime/usecase"
	tenantUC "github.com/Root-Emin/TicketLens/internal/application/tenant/usecase"
	triageUC "github.com/Root-Emin/TicketLens/internal/application/triage/usecase"

	// Gateway
	"github.com/Root-Emin/TicketLens/internal/gateway"
	gatewayInterceptors "github.com/Root-Emin/TicketLens/internal/infrastructure/gateway/interceptors"

	// Shared
	"github.com/Root-Emin/TicketLens/internal/shared/cache"
	"github.com/Root-Emin/TicketLens/internal/shared/config"
	"github.com/Root-Emin/TicketLens/internal/shared/database"
	"github.com/Root-Emin/TicketLens/internal/shared/events"
	"github.com/Root-Emin/TicketLens/internal/shared/logger"
	"github.com/Root-Emin/TicketLens/internal/shared/telemetry"
	"github.com/Root-Emin/TicketLens/internal/shared/version"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	log := logger.New(cfg.Log.Level, cfg.Log.Format)
	slog.SetDefault(log)

	log.Info("starting ticketlens",
		"host", cfg.Server.Host,
		"port", cfg.Server.Port,
		"env", cfg.Env,
	)

	// Refuse to serve traffic with development defaults outside development.
	// Returning here aborts startup: a deployment signing tokens with a public
	// secret is worse than a deployment that does not start.
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	if cfg.JWT.Secret == config.InsecureJWTSecret {
		log.Warn("JWT_SECRET is unset; authentication uses a known default value",
			"env", cfg.Env)
	}

	for _, w := range cfg.Warnings() {
		log.Warn(w)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize OpenTelemetry
	otelShutdown, err := telemetry.Setup(ctx, version.ServiceName, version.Version)
	if err != nil {
		log.Warn("opentelemetry setup failed", "error", err)
	} else {
		defer func() { _ = otelShutdown(context.Background()) }()
		log.Info("opentelemetry initialized")
	}

	// Initialize PostgreSQL.
	//
	// Development tolerates its absence so the UI can be worked on with no
	// infrastructure running. Anywhere else a database-less process is not a
	// degraded server, it is a server that answers every request with a 500
	// while its readiness probe has to lie about it — so refuse to start.
	db, err := database.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		if !cfg.IsDevelopment() {
			return fmt.Errorf("connect to postgres (required for APP_ENV=%s): %w", cfg.Env, err)
		}
		log.Warn("postgres unavailable, running without database", "error", err)
		db = nil
	} else {
		defer db.Close()
		log.Info("connected to postgres")
		if err := prepareSchema(context.Background(), cfg, db, log); err != nil {
			return err
		}
	}

	// Initialize Redis
	redisClient, err := cache.NewRedisClient(ctx, cfg.Redis)
	if err != nil {
		log.Warn("redis unavailable, running without cache", "error", err)
		redisClient = nil
	} else {
		defer redisClient.Close()
		log.Info("connected to redis")
	}

	// Initialize event bus (Kafka or in-process)
	eventBus := initEventBus(ctx, cfg, log)
	defer func() { _ = eventBus.Close() }()

	// Build dependencies
	deps := buildDependencies(log, cfg, db, redisClient, eventBus)

	// Start consuming only now: Consumer.Start creates one reader per SUBSCRIBED
	// topic, and every Subscribe call happens inside buildDependencies. Starting
	// earlier would iterate an empty handler map and silently consume nothing.
	if kafkaBus, ok := eventBus.(*infraKafka.Bus); ok {
		kafkaBus.Start(context.Background())
		log.Info("kafka consumers started")
	}

	// Build router
	r := router.New(deps)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", addr)
		serverErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
	case sig := <-shutdown:
		log.Info("shutdown signal received", "signal", sig)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			_ = srv.Close()
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}
		log.Info("server stopped gracefully")
	}

	return nil
}

// migrationTimeout bounds the schema step. It is deliberately not the ten
// second budget the rest of startup shares: that one sizes a TCP dial, while
// this one has to cover DDL over a table that may already hold real data.
const migrationTimeout = 2 * time.Minute

// prepareSchema reconciles the database schema with the binary's embedded
// migrations, in one of two modes.
//
// With DB_AUTO_MIGRATE on (the development default) the server applies pending
// migrations itself, so a fresh checkout serves without a separate step. With
// it off — every deployment — migrations are their own deploy stage (cmd/migrate)
// and the server only checks. Finding work left to do there means the deploy
// skipped that stage, and the running binary expects columns the database does
// not have; failing the boot surfaces that immediately instead of turning it
// into a scattering of query errors under live traffic.
func prepareSchema(ctx context.Context, cfg *config.Config, db *pgxpool.Pool, log *slog.Logger) error {
	ctx, cancel := context.WithTimeout(ctx, migrationTimeout)
	defer cancel()

	if cfg.Database.AutoMigrate {
		if err := postgres.MigrateUp(ctx, db, log); err != nil {
			return fmt.Errorf("run migrations: %w", err)
		}
		return nil
	}

	pending, err := postgres.PendingMigrations(ctx, db)
	if err != nil {
		return fmt.Errorf("check pending migrations: %w", err)
	}
	if len(pending) > 0 {
		return fmt.Errorf(
			"database has %d pending migration(s), starting with %s: run the migrate binary before the server (DB_AUTO_MIGRATE is off)",
			len(pending), pending[0])
	}

	log.Info("database schema is up to date", "auto_migrate", false)
	return nil
}

// initMailer selects the outbound mail adapter.
//
// No SMTP host means the logging mailer, which writes each message — invitation
// link included — to the server log instead of delivering it. That keeps local
// development and the test suite free of a mail server, mirrors how an empty
// CLASSIFIER_URL selects the keyword stub, and makes the absence visible rather
// than turning every invitation into a silent no-op. config.Warnings says so
// again at startup for any non-development environment.
func initMailer(cfg *config.Config, log *slog.Logger) iamPort.Mailer {
	if !cfg.Mail.Enabled() {
		log.Info("no SMTP host configured; invitation emails will be written to this log")
		return mail.NewLogMailer(log)
	}

	log.Info("smtp mailer initialized", "host", cfg.Mail.Host, "port", cfg.Mail.Port, "from", cfg.Mail.From)
	return mail.NewSMTPMailer(mail.SMTPConfig{
		Host:     cfg.Mail.Host,
		Port:     cfg.Mail.Port,
		Username: cfg.Mail.Username,
		Password: cfg.Mail.Password,
		From:     cfg.Mail.From,
		Timeout:  cfg.Mail.Timeout,
	})
}

// initEventBus creates either a Kafka bus or an in-process bus based on config.
func initEventBus(ctx context.Context, cfg *config.Config, log *slog.Logger) events.EventBus {
	if !cfg.Kafka.Enabled {
		log.Info("using in-process event bus (set KAFKA_ENABLED=true to use Kafka)")
		return events.NewInProcessBus(log, 256)
	}

	log.Info("initializing kafka event bus",
		"brokers", cfg.Kafka.Brokers,
		"group_id", cfg.Kafka.GroupID,
	)

	// Ensure topics exist
	if len(cfg.Kafka.Brokers) > 0 {
		if err := infraKafka.EnsureTopics(
			ctx,
			cfg.Kafka.Brokers[0],
			infraKafka.DefaultTopics(),
			cfg.Kafka.NumPartitions,
			cfg.Kafka.ReplicationFactor,
			log,
		); err != nil {
			log.Warn("failed to ensure kafka topics, falling back to in-process bus", "error", err)
			return events.NewInProcessBus(log, 256)
		}
	}

	kafkaBus := infraKafka.NewBus(cfg.Kafka.Brokers, cfg.Kafka.GroupID, log)

	// Consumption deliberately does NOT start here: subscriptions are registered
	// later, in buildDependencies. run() starts the readers once they exist.
	log.Info("kafka event bus initialized")
	return kafkaBus
}

func buildDependencies(
	log *slog.Logger,
	cfg *config.Config,
	db *pgxpool.Pool,
	redisClient *redis.Client,
	eventBus events.EventBus,
) router.Dependencies {
	deps := router.Dependencies{
		Logger:             log,
		DB:                 db,
		Redis:              redisClient,
		CORSAllowedOrigins: cfg.Server.CORSAllowedOrigins,
		MaxBodyBytes:       cfg.Server.MaxBodyBytes,

		AllowPublicRegistration: cfg.Onboarding.AllowPublicRegistration,
	}

	if db == nil {
		log.Warn("database not available, API endpoints will not work")
		return deps
	}

	// --- Repositories ---
	userRepo := pgIam.NewUserRepo(db)
	roleRepo := pgIam.NewRoleRepo(db)
	orgRepo := pgTenant.NewOrgRepo(db)
	workspaceRepo := pgTenant.NewWorkspaceRepository(db)
	appRepo := pgTenant.NewAppRepo(db)
	apiKeyRepo := pgTenant.NewAPIKeyRepo(db)
	endpointRepo := pgApimgmt.NewEndpointRepo(db)
	policyRepo := pgApimgmt.NewPolicyRepo(db)
	auditRepo := pgAudit.NewAuditRepo(db)
	departmentRepo := pgTriage.NewDepartmentRepo(db)
	customerRepo := pgTriage.NewCustomerRepo(db)
	staffRepo := pgTriage.NewStaffRepo(db)
	ticketRepo := pgTriage.NewTicketRepo(db)
	ticketMessageRepo := pgTriage.NewTicketMessageRepo(db)
	aiAnalysisRepo := pgTriage.NewAIAnalysisRepo(db)
	statsRepo := pgTriage.NewStatsRepo(db)
	invitationRepo := pgIam.NewInvitationRepo(db)
	txManager := database.NewTxManager(db)

	// --- Services ---
	jwtService := infraAuth.NewJWTService(cfg.JWT)
	rbacService := infraAuth.NewRBACService(roleRepo, redisClient)
	mailer := initMailer(cfg, log)

	deps.AuthService = jwtService
	deps.RBACService = rbacService
	deps.OrgRepo = orgRepo
	deps.WorkspaceRepo = workspaceRepo
	// Needed by the path-scope guards: an addressed app or user must belong to
	// the caller's organization before any handler resolves it by id.
	deps.AppRepo = appRepo
	deps.RoleRepo = roleRepo
	// Child ids in a path must be confirmed against their parent: an api key or
	// endpoint id is otherwise usable with any app id the caller owns.
	deps.APIKeyRepo = apiKeyRepo
	deps.EndpointRepo = endpointRepo

	// --- Use cases (with event bus for domain event publishing) ---
	registerUC := iamUC.NewRegisterUseCase(userRepo, jwtService, eventBus)
	loginUC := iamUC.NewLoginUseCase(userRepo, roleRepo, jwtService)
	changePasswordUC := iamUC.NewChangePasswordUseCase(userRepo, jwtService)
	assignRoleUC := iamUC.NewAssignRoleUseCase(roleRepo, rbacService, eventBus)
	createInvitationUC := iamUC.NewCreateInvitationUseCase(
		invitationRepo, userRepo, roleRepo, orgRepo, mailer, log,
		cfg.Onboarding.PublicAppURL, cfg.Onboarding.InvitationTTL)
	acceptInvitationUC := iamUC.NewAcceptInvitationUseCase(
		invitationRepo, userRepo, roleRepo, orgRepo, staffRepo,
		jwtService, rbacService, txManager, eventBus)
	listInvitationsUC := iamUC.NewListInvitationsUseCase(invitationRepo, roleRepo, userRepo)
	revokeInvitationUC := iamUC.NewRevokeInvitationUseCase(invitationRepo)
	listRolesUC := iamUC.NewListRolesUseCase(roleRepo)
	createOrgUC := tenantUC.NewCreateOrgUseCase(orgRepo, roleRepo, departmentRepo, eventBus)
	createWorkspaceUC := tenantUC.NewCreateWorkspaceUseCase(workspaceRepo, orgRepo, eventBus)
	listWorkspacesUC := tenantUC.NewListWorkspacesUseCase(workspaceRepo)
	updateWorkspaceUC := tenantUC.NewUpdateWorkspaceUseCase(workspaceRepo)
	createAppUC := tenantUC.NewCreateAppUseCase(appRepo, orgRepo, eventBus)
	manageKeysUC := tenantUC.NewManageAPIKeysUseCase(apiKeyRepo)
	defineEndpointUC := apimgmtUC.NewDefineEndpointUseCase(endpointRepo, eventBus)
	updatePolicyUC := apimgmtUC.NewUpdatePolicyUseCase(policyRepo)
	retireEndpointUC := apimgmtUC.NewRetireEndpointUseCase(endpointRepo, eventBus)
	activateEndpointUC := apimgmtUC.NewActivateEndpointUseCase(endpointRepo, eventBus)

	// --- Register sample Kafka consumers ---
	// Log all IAM events
	eventBus.Subscribe(events.TopicIAM, func(ctx context.Context, event events.Event) error {
		log.Info("iam event received", "event", event)
		return nil
	})
	// Log all tenant events
	eventBus.Subscribe(events.TopicTenant, func(ctx context.Context, event events.Event) error {
		log.Info("tenant event received", "event", event)
		return nil
	})
	// Log all API management events
	eventBus.Subscribe(events.TopicAPIManagement, func(ctx context.Context, event events.Event) error {
		log.Info("api-management event received", "event", event)
		return nil
	})

	// --- Handlers ---
	deps.IAMHandler = iamHandler.NewHandler(
		registerUC, loginUC, assignRoleUC, changePasswordUC,
		createInvitationUC, acceptInvitationUC, listInvitationsUC, revokeInvitationUC,
		listRolesUC,
		userRepo)
	deps.TenantHandler = tenantHandler.NewHandler(
		createOrgUC,
		createAppUC,
		manageKeysUC,
		createWorkspaceUC,
		listWorkspacesUC,
		updateWorkspaceUC,
		orgRepo,
		appRepo,
	)
	deps.APIMgmtHandler = apimgmtHandler.NewHandler(defineEndpointUC, updatePolicyUC, retireEndpointUC, activateEndpointUC, endpointRepo, policyRepo)
	deps.AuditHandler = auditHandler.NewHandler(auditRepo)

	// --- Triage (ticketing) ---
	// CLASSIFIER_URL selects the Python inference service; empty keeps the
	// in-process keyword stub so local demos work with no extra container.
	ticketClassifier := newClassifier(cfg, log)
	analyzeTicketUC := triageUC.NewAnalyzeTicketUseCase(
		ticketRepo, ticketMessageRepo, departmentRepo, customerRepo, aiAnalysisRepo, userRepo,
		ticketClassifier, cfg.Classifier.ReviewThreshold, eventBus,
	)

	deps.TriageHandler = triageHandler.NewHandler(triageHandler.Config{
		ListDepartments:  triageUC.NewListDepartmentsUseCase(departmentRepo, ticketRepo, staffRepo),
		CreateDepartment: triageUC.NewCreateDepartmentUseCase(departmentRepo),
		UpdateDepartment: triageUC.NewUpdateDepartmentUseCase(departmentRepo, ticketRepo),
		DeleteDepartment: triageUC.NewDeleteDepartmentUseCase(departmentRepo, ticketRepo),

		ListStaff:             triageUC.NewListStaffUseCase(staffRepo),
		AssignStaffDepartment: triageUC.NewAssignStaffDepartmentUseCase(staffRepo, departmentRepo),

		ListCustomers:  triageUC.NewListCustomersUseCase(customerRepo),
		CreateCustomer: triageUC.NewCreateCustomerUseCase(customerRepo),
		GetCustomer: triageUC.NewGetCustomerUseCase(
			customerRepo, ticketRepo, departmentRepo, ticketMessageRepo, aiAnalysisRepo, userRepo),

		CreateTicket: triageUC.NewCreateTicketUseCase(
			ticketRepo, ticketMessageRepo, customerRepo, departmentRepo, aiAnalysisRepo, userRepo, txManager, eventBus),
		ListTickets: triageUC.NewListTicketsUseCase(
			ticketRepo, departmentRepo, customerRepo, ticketMessageRepo, aiAnalysisRepo, userRepo),
		GetTicket: triageUC.NewGetTicketUseCase(
			ticketRepo, departmentRepo, customerRepo, ticketMessageRepo, aiAnalysisRepo, userRepo),
		UpdateTicket: triageUC.NewUpdateTicketUseCase(
			ticketRepo, aiAnalysisRepo, departmentRepo, customerRepo, ticketMessageRepo, userRepo, auditRepo, eventBus),
		AssignTicket: triageUC.NewAssignTicketUseCase(
			ticketRepo, roleRepo, departmentRepo, customerRepo, ticketMessageRepo, aiAnalysisRepo, userRepo, eventBus),

		ListMessages:  triageUC.NewListMessagesUseCase(ticketRepo, ticketMessageRepo),
		CreateMessage: triageUC.NewCreateMessageUseCase(ticketRepo, ticketMessageRepo),
		ListAnalyses:  triageUC.NewListAnalysesUseCase(ticketRepo, aiAnalysisRepo),
		AnalyzeTicket: analyzeTicketUC,

		StatsOverview: triageUC.NewStatsOverviewUseCase(statsRepo),
		TicketSummary: triageUC.NewTicketSummaryUseCase(statsRepo),

		// Resolves per request whether the caller reads the whole queue or
		// only their own tickets. Every ticket route depends on it.
		ResolveScope: triageUC.NewResolveScopeUseCase(rbacService, customerRepo),

		ReviewThreshold: cfg.Classifier.ReviewThreshold,
	})

	// Classify newly created tickets off the event bus. The consumer is
	// idempotent, so at-least-once delivery cannot double-analyze a ticket.
	infraTriage.NewTicketConsumer(analyzeTicketUC, log).Register(eventBus)

	// --- WebSocket real-time hub ---
	wsHub := infraWS.NewHub(log, cfg.WebSocket.MaxConnections)
	eventBridge := infraWS.NewEventBridge(wsHub, appRepo, log)
	eventBridge.Register(eventBus)

	validateConnectUC := realtimeUC.NewValidateConnectUseCase(appRepo, rbacService)
	wsUpgrader := infraWS.NewUpgrader(infraWS.UpgraderConfig{
		ReadBufferSize:  cfg.WebSocket.ReadBufferSize,
		WriteBufferSize: cfg.WebSocket.WriteBufferSize,
		AllowedOrigins:  cfg.Server.CORSAllowedOrigins,
	})
	deps.RealtimeHandler = realtimeHandler.NewHandler(realtimeHandler.Config{
		ValidateUC:   validateConnectUC,
		AuthService:  jwtService,
		Hub:          wsHub,
		Upgrader:     wsUpgrader,
		PingInterval: cfg.WebSocket.PingIntervalSec,
		Logger:       log,
		Enabled:      cfg.WebSocket.Enabled,
	})

	// --- Gateway pipeline with interceptors ---
	// Create interceptor chain: schema validation, PII masking, request/response transformers
	piiMasker := gatewayInterceptors.NewPIIMasker(
		[]string{"password", "password_hash", "api_key", "secret", "token", "ssn", "credit_card"},
		"***",
	)
	schemaValidator := gatewayInterceptors.NewSchemaValidator()

	// Create dynamic handler resolver for routing requests to backend service handlers
	// This supports:
	// 1. Registered handlers (if you register specific handlers)
	// 2. HTTP proxy to external services (if backend_service is a URL or configured)
	// 3. Generic dynamic database handler (automatically performs CRUD operations)
	backendRegistry := gateway.NewBackendRegistry()
	dynamicResolver := gateway.NewDynamicHandlerResolver(backendRegistry, log, db)

	// Optional: Register service configurations for HTTP proxying
	// Example:
	// dynamicResolver.RegisterServiceConfig("product-service", gateway.ServiceConfig{
	//     BaseURL: "https://api.example.com/products",
	//     Headers: map[string]string{"Authorization": "Bearer token"},
	// })

	// Optional: Register specific handlers for services that need custom logic
	// Example:
	// productHandler := handlers.NewProductHandler(...)
	// backendRegistry.Register("product-service", productHandler)

	// Wire interceptors into gateway pipeline with dynamic resolver
	deps.GatewayPipeline = gateway.NewPipeline(
		endpointRepo,
		policyRepo,
		rbacService,
		redisClient,
		log,
		dynamicResolver, // Dynamic handler resolver (supports registered handlers, HTTP proxy, and generic handling)
		schemaValidator, // Schema validation interceptor
		piiMasker,       // PII masking interceptor
	)

	return deps
}

// newClassifier picks the HTTP model service when CLASSIFIER_URL is set,
// otherwise the deterministic keyword stub.
func newClassifier(cfg *config.Config, log *slog.Logger) port.Classifier {
	if cfg.Classifier.URL == "" {
		log.Info("classifier: using in-process stub")
		return stubClassifier.New()
	}
	log.Info("classifier: using HTTP adapter",
		"url", cfg.Classifier.URL,
		"timeout", cfg.Classifier.Timeout,
		"max_retries", cfg.Classifier.MaxRetries,
		"fallback_to_stub", cfg.Classifier.FallbackToStub,
	)
	return httpClassifier.New(httpClassifier.Config{
		URL:            cfg.Classifier.URL,
		Timeout:        cfg.Classifier.Timeout,
		MaxRetries:     cfg.Classifier.MaxRetries,
		FallbackToStub: cfg.Classifier.FallbackToStub,
	}, log)
}
