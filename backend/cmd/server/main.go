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

	// Initialize PostgreSQL
	db, err := database.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		log.Warn("postgres unavailable, running without database", "error", err)
		db = nil
	} else {
		defer db.Close()
		log.Info("connected to postgres")
		// Apply pending migrations so a fresh checkout does not require a
		// separate make migrate / start.sh infra step before serving.
		if migErr := postgres.MigrateUp(ctx, db, log); migErr != nil {
			return fmt.Errorf("run migrations: %w", migErr)
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
	ticketRepo := pgTriage.NewTicketRepo(db)
	ticketMessageRepo := pgTriage.NewTicketMessageRepo(db)
	aiAnalysisRepo := pgTriage.NewAIAnalysisRepo(db)
	statsRepo := pgTriage.NewStatsRepo(db)
	txManager := database.NewTxManager(db)

	// --- Services ---
	jwtService := infraAuth.NewJWTService(cfg.JWT)
	rbacService := infraAuth.NewRBACService(roleRepo, redisClient)

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
	assignRoleUC := iamUC.NewAssignRoleUseCase(roleRepo, rbacService, eventBus)
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
	deps.IAMHandler = iamHandler.NewHandler(registerUC, loginUC, assignRoleUC, userRepo)
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
		ListDepartments:  triageUC.NewListDepartmentsUseCase(departmentRepo, ticketRepo),
		CreateDepartment: triageUC.NewCreateDepartmentUseCase(departmentRepo),
		UpdateDepartment: triageUC.NewUpdateDepartmentUseCase(departmentRepo, ticketRepo),
		DeleteDepartment: triageUC.NewDeleteDepartmentUseCase(departmentRepo, ticketRepo),

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
