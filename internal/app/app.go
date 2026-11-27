// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

// @title           GoPostgREST
// @version         1.0
// @description     GoPostgREST is a RESTful API for PostgreSQL.
// @host            localhost:8080
// @BasePath        /api/v1
// @license.name    Apache 2.0
// @license.url     https://www.apache.org/licenses/LICENSE-2.0
package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/aptlogica/sereni-base/internal/config"
	appConstant "github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/handlers"
	"github.com/aptlogica/sereni-base/internal/middleware"
	"github.com/aptlogica/sereni-base/internal/platform"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
	"github.com/aptlogica/sereni-base/internal/providers/logquery"
	"github.com/aptlogica/sereni-base/internal/providers/logship"
	"github.com/aptlogica/sereni-base/internal/router"
	"github.com/aptlogica/sereni-base/internal/scripts"
	"github.com/aptlogica/sereni-base/internal/services"
	rbac "github.com/aptlogica/sereni-base/internal/services/rbac"

	"github.com/aptlogica/sereni-base/internal/providers/antivirus"
	"github.com/aptlogica/sereni-base/internal/providers/auth"
	"github.com/aptlogica/sereni-base/internal/providers/email"
	"github.com/aptlogica/sereni-base/internal/providers/otp"
	"github.com/aptlogica/sereni-base/internal/providers/storage"

	"time"

	docs "github.com/aptlogica/sereni-base/docs"

	"github.com/aptlogica/go-postgres-rest/pkg"
	dbConfig "github.com/aptlogica/go-postgres-rest/pkg/config"

	"github.com/gin-gonic/gin"
)

type App struct {
	config       *config.Config
	server       *http.Server
	router       *gin.Engine
	authProvider auth.AuthProvider
	dbService    *pkg.DatabaseService
	publicAPI    *platform.API
	logShipper   *logship.Client // sends API platform logs to sereni-log; nil when LOG_SERVICE_URL is empty
}

func New(cfg *config.Config) (*App, error) {
	// The public API has no authentication yet, so it must never be open in production
	if cfg.PublicAPI.OpenAccess && (cfg.Server.Env == "prod" || cfg.Server.Env == "production") {
		return nil, fmt.Errorf("PUBLIC_API_OPEN_ACCESS must be false when SERVER_ENV is %s", cfg.Server.Env)
	}

	// Keep Swagger runtime metadata aligned with active server config.
	host := cfg.Server.Host
	if host == "" || host == "0.0.0.0" {
		host = "localhost"
	}

	scheme := strings.ToLower(strings.TrimSpace(cfg.Server.Scheme))
	if scheme != "https" {
		scheme = "http"
	}

	docs.SwaggerInfo.Title = "SereniBase API"
	docs.SwaggerInfo.Description = "SereniBase REST API documentation"
	docs.SwaggerInfo.Version = cfg.Server.Version
	docs.SwaggerInfo.Host = fmt.Sprintf("%s:%s", host, cfg.Server.Port)
	docs.SwaggerInfo.BasePath = "/api/v1"
	docs.SwaggerInfo.Schemes = []string{scheme}

	logger.Init(cfg)

	dfConfig := dbConfig.Config{
		Database: dbConfig.DatabaseConfig{
			Host:         cfg.Database.Host,
			Driver:       cfg.Database.Driver,
			Port:         cfg.Database.Port,
			Username:     cfg.Database.Username,
			Password:     cfg.Database.Password,
			DatabaseName: cfg.Database.DatabaseName,
			SSLMode:      cfg.Database.SSLMode,
			MaxOpenConns: cfg.Database.MaxOpenConns,
			MaxIdleConns: cfg.Database.MaxIdleConns,
		},
	}

	// Initialize database service for repository
	dbService, err := pkg.NewDatabaseServiceWithInit(&dfConfig)
	if err != nil {
		fmt.Printf("failed to initialize database service: %v", err)
		return nil, err
	}

	// providers
	emailTemplateService := email.NewEmailTemplateService()
	emailProvider := email.NewService(cfg.Email, 100, emailTemplateService)
	emailProvider.Start(5)

	otpProvider := otp.NewService(5 * time.Minute)
	// otpProvider := otp.NewService(24 * time.Hour)

	authProvider, err := auth.NewAuthProvider(&cfg.Auth)
	if err != nil {
		fmt.Printf("failed to connect to auth provider: %v", err)
		return nil, err
	}

	storageProvider, err := storage.NewStorage(&cfg.Storage)
	if err != nil {
		fmt.Printf("failed to connect to storage provider: %v", err)
		return nil, err
	}

	antivirusProvider, err := antivirus.NewAntivirus(&cfg.Antivirus)
	if err != nil {
		fmt.Printf("failed to initialize antivirus provider: %v", err)
		return nil, err
	}

	// Initialize services
	userService := services.NewUserService(dbService)
	workspaceService := services.NewWorkspaceService(dbService)
	workspaceMemberService := services.NewWorkspaceMemberService(dbService)
	baseService := services.NewBaseService(dbService)
	assetService := services.NewAssetsService(dbService)
	modelService := services.NewModelService(dbService)
	columnService := services.NewColumnService(dbService)
	viewService := services.NewViewService(dbService)
	automationService := services.NewAutomationService(dbService, modelService)
	primaryKeyService := services.NewPrimaryKeyService(dbService, modelService, columnService)
	tableConstraintService := services.NewTableConstraintService(dbService, modelService, columnService)
	relationshipService := services.NewRelationshipService(dbService)
	userResetTokenService := services.NewUserResetTokenService(dbService)
	resourceService := services.NewCoreResourceService(dbService)
	actionService := services.NewCoreActionService(dbService)
	permissionService := services.NewRBACPermissionService(dbService)
	rolePermissionService := services.NewRolePermissionService(dbService)
	accessMemberService := services.NewAccessMemberService(dbService)
	accessRoleService := services.NewRBACAccessRoleService(dbService)

	assetManagementService := services.NewAssetManagementService(
		dbService,
		assetService,
		storageProvider,
		antivirusProvider,
	)

	tableManagementService := services.NewTableManagementService(
		dfConfig.Database.Driver,
		dbService,
		modelService,
		columnService,
		viewService,
		relationshipService,
		assetManagementService,
	)

	rbacManagementService := services.NewRBACManagementService(
		dbService,
		services.RBACManagementServiceDeps{
			RoleService:           accessRoleService,
			ResourceService:       resourceService,
			ActionService:         actionService,
			PermissionService:     permissionService,
			RolePermissionService: rolePermissionService,
			AccessMemberService:   accessMemberService,
			BaseService:           baseService,
		},
	)

	baseManagementService := services.NewBaseManagementService(
		dbService,
		baseService,
		tableManagementService,
		modelService,
		assetManagementService,
	)

	importService := services.NewImportService(tableManagementService, baseManagementService, antivirusProvider)

	// Create workspaceManagementService first (no circular dependency needed here)
	workspaceManagementService := services.NewWorkspaceManagementService(
		dbService,
		workspaceService,
		workspaceMemberService,
		baseManagementService,
		tableManagementService,
		rbacManagementService,
	)

	userManagementService := services.NewUserManagementService(
		dbService,
		userService,
		assetManagementService,
		userResetTokenService,
		workspaceManagementService, // Now pass the actual service
		rbacManagementService,
		authProvider,
	)

	authService := services.NewAuthManagementService(
		cfg.TemporaryAddedUserPassword,
		dbService,
		services.AuthManagementServiceDeps{
			UserManagementService:      userManagementService,
			WorkspaceManagementService: workspaceManagementService,
			UserResetTokenService:      userResetTokenService,
			RBACManagementService:      rbacManagementService,
		},
		services.AuthManagementProviderDeps{
			OTPProviderService:   otpProvider,
			EmailTemplateService: emailTemplateService,
			EmailProviderService: emailProvider,
			AuthProviderService:  authProvider,
		},
	)

	organizationService := services.NewOrganizationService(dbService)
	baseAPIService := platform.NewBaseAPIService(dbService)

	// Base public API, open without authentication: only built when PUBLIC_API_OPEN_ACCESS is on
	var publicAPI *platform.API
	var logShipper *logship.Client
	if cfg.PublicAPI.OpenAccess {
		recordWriter, ok := tableManagementService.(platform.RecordWriter)
		if !ok {
			return nil, fmt.Errorf("table management service does not support public API writes")
		}
		publicAPI = platform.NewAPI(dbService.DB, appConstant.MasterDatabase, recordWriter, cfg.PublicAPI.RateLimitPerMinute).
			WithSchemaServices(platform.SchemaServices{
				Tables:      tableManagementService,
				Constraints: tableConstraintService,
				PrimaryKeys: primaryKeyService,
			})
		var err error
		if logShipper, err = configurePublicAPILog(publicAPI, cfg); err != nil {
			return nil, err
		}
		publicAPI.StartBackground(context.Background())
	}

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, workspaceManagementService, baseManagementService)
	workspaceHandler := handlers.NewWorkspaceHandler(workspaceManagementService, authService)
	baseHandler := handlers.NewBaseHandler(baseManagementService)
	assetHandler := handlers.NewAssetsHandler(assetManagementService)
	tableHandler := handlers.NewTableHandler(tableManagementService, importService)
	userHandler := handlers.NewUserHandler(userManagementService)
	organizationHandler := handlers.NewOrganizationHandler(organizationService)
	automationHandler := handlers.NewAutomationHandler(automationService)
	tableConstraintHandler := handlers.NewTableConstraintHandler(tableConstraintService)
	primaryKeyHandler := handlers.NewPrimaryKeyHandler(primaryKeyService)
	baseAPIHandler := handlers.NewBaseAPIHandler(baseAPIService)
	// Reading the API logs only needs LOG_SERVICE_URL; it does not depend on PUBLIC_API_OPEN_ACCESS
	logQuery := logquery.New(logquery.Config{URL: cfg.PublicAPI.LogServiceURL, APIKey: cfg.PublicAPI.LogServiceAPIKey})
	baseAPILogsHandler := handlers.NewBaseAPILogsHandler(services.NewBaseAPILogService(baseAPIService, logQuery))

	handlerGroups := router.Handlers{
		Auth:            authHandler,
		Workspace:       workspaceHandler,
		Base:            baseHandler,
		Asset:           assetHandler,
		Table:           tableHandler,
		User:            userHandler,
		Organization:    organizationHandler,
		Automation:      automationHandler,
		TableConstraint: tableConstraintHandler,
		PrimaryKey:      primaryKeyHandler,
		BaseAPI:         baseAPIHandler,
		BaseAPILogs:     baseAPILogsHandler,
		PublicAPI:       publicAPI,
	}

	middlewareGroups := router.Middlewares{
		CORS:                    middleware.CORS,
		RequestLogger:           middleware.RequestLogger,
		DatabaseQueryLogger:     middleware.DatabaseQueryLogger,
		RequestSizeLimit:        middleware.RequestSizeLimit,
		AuthMiddleware:          func() gin.HandlerFunc { return middleware.AuthMiddleware(authProvider, userManagementService) },
		FileSizeLimitMiddleware: middleware.FileSizeLimitMiddleware,
		ScopeHeaderMiddleware:   func(scope string) gin.HandlerFunc { return middleware.ScopeHeaderMiddleware(scope) },
		WorkspaceAndBaseAccessValidationMiddleware: func(allowedAccess []string) gin.HandlerFunc {
			return middleware.WorkspaceAndBaseAccessValidationMiddleware(workspaceMemberService, allowedAccess)
		},
		AccessMemberService: accessMemberService,
		// Resolve tables, columns, views and access rows to the base/workspace that owns them,
		// for object-level authorization. Columns and views resolve through their table, whose
		// base is authoritative, rather than their own client-supplied base_id.
		ScopeLookups: middleware.ScopeLookups{
			BaseWorkspace: func(ctx context.Context, schema, baseID string) (string, error) {
				base, err := baseService.GetBaseByID(ctx, schema, baseID)
				return base.WorkspaceID, err
			},
			ModelBase: func(ctx context.Context, schema, modelID string) (string, error) {
				model, err := modelService.GetModelByID(ctx, schema, modelID)
				return model.BaseID.String(), err
			},
			ColumnModel: func(ctx context.Context, schema, columnID string) (string, error) {
				column, err := columnService.GetColumnByID(ctx, schema, columnID)
				return column.ModelID, err
			},
			ViewModel: func(ctx context.Context, schema, viewID string) (string, error) {
				view, err := viewService.GetViewByID(ctx, schema, viewID)
				return view.ModelID, err
			},
			AccessMember: func(_ context.Context, schema, id string) (dto.AccessMemberDTO, error) {
				return rbac.GetAccessMemberByID(dbService, schema, id)
			},
		},
	}

	r := router.Setup(cfg, handlerGroups, middlewareGroups)

	// Create server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		// MaxHeaderBytes: 100 << 20, // 100MB
	}

	return &App{
		config:       cfg,
		server:       server,
		router:       r,
		authProvider: authProvider,
		dbService:    dbService,
		publicAPI:    publicAPI,
		logShipper:   logShipper,
	}, nil
}

// configurePublicAPILog sets up the access log of the public API from the PUBLIC_API_LOG_* and LOG_SERVICE_*
// settings. Every logged request goes to the local log; with LOG_SERVICE_URL set it is also queued for
// sereni-log, with bodies as PUBLIC_API_LOG_BODIES says. It returns the shipper to flush at shutdown, or nil.
func configurePublicAPILog(api *platform.API, cfg *config.Config) (*logship.Client, error) {
	pc := cfg.PublicAPI
	var logAll bool
	switch strings.ToLower(strings.TrimSpace(pc.LogMode)) {
	case "", "all":
		logAll = true
	case "errors":
	case "off":
		return nil, nil
	default:
		return nil, fmt.Errorf("PUBLIC_API_LOG_MODE must be all, errors or off, got %q", pc.LogMode)
	}
	sinks := platform.MultiSink{platform.ZerologSink{}}

	var shipper *logship.Client
	if url := strings.TrimSpace(pc.LogServiceURL); url != "" {
		bodies := strings.ToLower(strings.TrimSpace(pc.LogBodies))
		switch bodies {
		case "":
			bodies = platform.BodiesErrors
		case platform.BodiesOff, platform.BodiesErrors, platform.BodiesAll:
		default:
			return nil, fmt.Errorf("PUBLIC_API_LOG_BODIES must be off, errors or all, got %q", pc.LogBodies)
		}
		shipper = logship.New(logship.Config{
			URL:           url,
			APIKey:        pc.LogServiceAPIKey,
			QueueSize:     pc.LogQueueSize,
			BatchSize:     pc.LogBatchSize,
			FlushInterval: time.Duration(pc.LogFlushSeconds) * time.Second,
			OnDrop:        platform.LogShipDropped,
		})
		shipper.Start()
		sinks = append(sinks, platform.LogServiceSink{Shipper: shipper})
		api.WithLogBodies(bodies, pc.LogMaxBodyBytes, splitList(pc.LogRedactFields))
	}
	api.WithLogSink(sinks, cfg.Server.Env, logAll)
	return shipper, nil
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// Shutdown flushes the API platform logs still queued for sereni-log. It waits until ctx is done.
func (a *App) Shutdown(ctx context.Context) error {
	if a.logShipper == nil {
		return nil
	}
	return a.logShipper.Shutdown(ctx)
}

func (a *App) Run() error {
	// Run any pre-server logic, like migrations or initializations
	if a.dbService == nil {
		return fmt.Errorf("database service not initialized")
	}

	runBeforeServer(a.dbService, a.authProvider, a.config)

	// Idempotency keys of the public API are kept for 24 hours
	if a.publicAPI != nil {
		a.publicAPI.StartIdempotencyCleanup(time.Hour)
	}

	fmt.Printf("🚀 Serenibase server starting on %s\n", a.server.Addr)
	// fmt.Printf("📚 API Documentation available at http://%s/api/v1/health\n", a.server.Addr)
	// fmt.Printf("🔍 Database introspection: GET /api/v1/schema/tables\n")
	// fmt.Printf("⚡ Advanced querying: POST /api/v1/{table}/query\n")
	// fmt.Printf("🛠️  DDL operations: POST /api/v1/ddl/tables\n")
	// fmt.Printf("📊 Analytics: GET /api/v1/analytics/database\n")
	// fmt.Printf("📈 Metrics: GET /metrics\n")
	// fmt.Printf("💾 Export data: GET /api/v1/export/{table}/csv\n")
	// fmt.Printf("🔧 Health checks: GET /health, /ready, /live\n")

	// Start the server. On SIGINT or SIGTERM stop taking requests, then flush the logs still queued for
	// sereni-log, each with a short deadline.
	stop, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	serverErr := make(chan error, 1)
	go func() { serverErr <- a.server.ListenAndServe() }()

	select {
	case err := <-serverErr:
		a.flushLogs()
		return err
	case <-stop.Done():
		ctx, done := context.WithTimeout(context.Background(), 10*time.Second)
		defer done()
		if err := a.server.Shutdown(ctx); err != nil {
			logger.Get().Warn().Err(err).Msg("server did not stop cleanly")
		}
		a.flushLogs()
		return nil
	}
}

// flushLogs sends the API platform logs still queued for sereni-log, waiting up to 5 seconds
func (a *App) flushLogs() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.Shutdown(ctx); err != nil {
		logger.Get().Warn().Err(err).Msg("API logs were not all sent before shutdown")
	}
}

func runBeforeServer(repo *pkg.DatabaseService, authProvider auth.AuthProvider, cfg *config.Config) {
	fmt.Println("Running script before Gin server starts...")

	// Your custom logic like DB connection, migration, etc.
	scripts.CreateMasterSchema(repo) // Example: Create database schema

	if err := scripts.CreateDefaultRBAC(repo); err != nil {
		fmt.Printf("⚠ Warning: Default RBAC creation failed: %v\n", err)
	}
	// Register predefined owner from configuration
	if err := scripts.RegisterOwner(repo, authProvider, cfg); err != nil {
		fmt.Printf("⚠ Warning: Owner registration failed: %v\n", err)
	}
	// Test bases for the public API, only while it is open without authentication
	if cfg.PublicAPI.OpenAccess && cfg.PublicAPI.SeedWorkspaces != "" {
		scripts.SeedAPIBases(repo, cfg.PublicAPI.SeedWorkspaces)
	}
}

// Router exposes the Gin engine so that callers can modify or augment routes.
func (a *App) Router() *gin.Engine {
	return a.router
}
