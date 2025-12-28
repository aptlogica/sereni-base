// @title           GoPostgREST
// @version         1.0
// @description     GoPostgREST is a RESTful API for PostgreSQL.
// @host            localhost:8080
// @BasePath        /api/v1
package app

import (
	"fmt"
	"net/http"
	"serenibase/internal/config"
	"serenibase/internal/handlers"
	"serenibase/internal/middleware"
	"serenibase/internal/providers/logger"
	"serenibase/internal/router"
	"serenibase/internal/scripts"
	"serenibase/internal/services"
	"serenibase/internal/services/interfaces"

	"serenibase/internal/providers/antivirus"
	"serenibase/internal/providers/auth"
	"serenibase/internal/providers/email"
	"serenibase/internal/providers/otp"
	"serenibase/internal/providers/storage"

	"time"

	// _ "serenibase/docs"

	"godbgrest/pkg"
	dbConfig "godbgrest/pkg/config"

	"github.com/gin-gonic/gin"
)

type App struct {
	config       *config.Config
	server       *http.Server
	authProvider auth.AuthProvider
}

func New(cfg *config.Config) (*App, error) {

	logger.Init(cfg)

	dbCfg, err := dbConfig.Load()

	// Initialize database service for repository
	dbService := pkg.NewDatabaseService()

	db, err := dbService.Connect(dbCfg)
	if err != nil {
		fmt.Printf("failed to connect to db: %v", err)
		return nil, err
	}

	// Initialize services
	if err := dbService.InitServices(db); err != nil {
		fmt.Printf("failed to init services: %v", err)
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
	relationshipService := services.NewRelationshipService(dbService)
	userResetTokenService := services.NewUserResetTokenService(dbService)
	resourceService := services.NewResourceService(dbService)
	actionService := services.NewActionService(dbService)
	permissionService := services.NewPermissionService(dbService)
	rolePermissionService := services.NewRolePermissionService(dbService)
	accessMemberService := services.NewAccessMemberService(dbService)
	accessRoleService := services.NewAccessRoleService(dbService)

	assetManagementService := services.NewAssetManagementService(
		dbService,
		assetService,
		storageProvider,
		antivirusProvider,
	)

	tableManagementService := services.NewTableManagementService(
		dbCfg.Database.Driver,
		dbService,
		modelService,
		columnService,
		viewService,
		relationshipService,
		assetManagementService,
	)

	rbacManagementService := services.NewRBACManagementService(
		dbService,
		accessRoleService,
		resourceService,
		actionService,
		permissionService,
		rolePermissionService,
		accessMemberService,
		baseService,
	)

	importService := services.NewImportService(tableManagementService)

	baseManagementService := services.NewBaseManagementService(
		dbService,
		baseService,
		modelService,
	)

	// Create workspaceManagementService without authManagementService first (will be set later)
	var workspaceManagementService interfaces.WorkspaceManagementService

	userManagementService := services.NewUserManagementService(
		dbService,
		userService,
		assetManagementService,
		userResetTokenService,
		nil, // Placeholder for workspaceManagementService (will be set after)
		rbacManagementService,
		authProvider,
	)

	authService := services.NewAuthManagementService(
		cfg.TemporaryAddedUserPassword,
		dbService,
		userManagementService,
		nil, // Placeholder for workspaceManagementService (will be set after)
		userResetTokenService,
		rbacManagementService,
		otpProvider,
		emailTemplateService,
		emailProvider,
		authProvider,
	)

	// Now create workspaceManagementService with authService
	workspaceManagementService = services.NewWorkspaceManagementService(
		dbService,
		workspaceService,
		workspaceMemberService,
		baseManagementService,
		tableManagementService,
		rbacManagementService,
	)

	organizationService := services.NewOrganizationService(dbService)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	workspaceHandler := handlers.NewWorkspaceHandler(workspaceManagementService, authService)
	baseHandler := handlers.NewBaseHandler(baseManagementService)
	assetHandler := handlers.NewAssetsHandler(assetManagementService)
	tableHandler := handlers.NewTableHandler(tableManagementService, importService)
	userHandler := handlers.NewUserHandler(userManagementService)
	organizationHandler := handlers.NewOrganizationHandler(organizationService)

	handlerGroups := router.Handlers{
		Auth:         authHandler,
		Workspace:    workspaceHandler,
		Base:         baseHandler,
		Asset:        assetHandler,
		Table:        tableHandler,
		User:         userHandler,
		Organization: organizationHandler,
	}

	middlewareGroups := router.Middlewares{
		CORS:                    middleware.CORS,
		RequestLogger:           middleware.RequestLogger,
		DatabaseQueryLogger:     middleware.DatabaseQueryLogger,
		RequestSizeLimit:        middleware.RequestSizeLimit,
		AuthMiddleware:          func() gin.HandlerFunc { return middleware.AuthMiddleware(authProvider) },
		FileSizeLimitMiddleware: middleware.FileSizeLimitMiddleware,
		ScopeHeaderMiddleware:   func(scope string) gin.HandlerFunc { return middleware.ScopeHeaderMiddleware(scope) },
		WorkspaceAndBaseAccessValidationMiddleware: func(allowedAccess []string) gin.HandlerFunc {
			return middleware.WorkspaceAndBaseAccessValidationMiddleware(workspaceMemberService, allowedAccess)
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
		authProvider: authProvider,
	}, nil
}

func (a *App) Run() error {
	// Run any pre-server logic, like migrations or initializations
	// conn = database.NewDb(&cfg.Database)
	dbCfg, err := dbConfig.Load()
	if err != nil {
		fmt.Printf("Failed to load database configuration: %v\n", err)
	}

	fmt.Println(dbCfg)

	dbService := pkg.NewDatabaseService()

	db, err := dbService.Connect(dbCfg)
	if err != nil {
		fmt.Printf("failed to connect to db: %v", err)
		return err
	}

	// Initialize services
	if err := dbService.InitServices(db); err != nil {
		fmt.Printf("failed to init services: %v", err)
	}

	runBeforeServer(dbService, a.authProvider, a.config)

	fmt.Printf("🚀 Serenibase server starting on %s\n", a.server.Addr)
	// fmt.Printf("📚 API Documentation available at http://%s/api/v1/health\n", a.server.Addr)
	// fmt.Printf("🔍 Database introspection: GET /api/v1/schema/tables\n")
	// fmt.Printf("⚡ Advanced querying: POST /api/v1/{table}/query\n")
	// fmt.Printf("🛠️  DDL operations: POST /api/v1/ddl/tables\n")
	// fmt.Printf("📊 Analytics: GET /api/v1/analytics/database\n")
	// fmt.Printf("📈 Metrics: GET /metrics\n")
	// fmt.Printf("💾 Export data: GET /api/v1/export/{table}/csv\n")
	// fmt.Printf("🔧 Health checks: GET /health, /ready, /live\n")

	// Start the server
	return a.server.ListenAndServe()
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
}
