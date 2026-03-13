// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

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

	"serenibase/internal/providers/antivirus"
	"serenibase/internal/providers/auth"
	"serenibase/internal/providers/email"
	"serenibase/internal/providers/otp"
	"serenibase/internal/providers/storage"

	"time"

	// _ "serenibase/docs"

	"go-postgres-rest/pkg"
	dbConfig "go-postgres-rest/pkg/config"

	"github.com/gin-gonic/gin"
)

type App struct {
	config       *config.Config
	server       *http.Server
	router       *gin.Engine
	authProvider auth.AuthProvider
	dbService    *pkg.DatabaseService
}

func New(cfg *config.Config) (*App, error) {

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
		AuthMiddleware:          func() gin.HandlerFunc { return middleware.AuthMiddleware(authProvider, userManagementService) },
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
		router:       r,
		authProvider: authProvider,
		dbService:    dbService,
	}, nil
}

func (a *App) Run() error {
	// Run any pre-server logic, like migrations or initializations
	if a.dbService == nil {
		return fmt.Errorf("database service not initialized")
	}

	runBeforeServer(a.dbService, a.authProvider, a.config)

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

// Router exposes the Gin engine so that callers can modify or augment routes.
func (a *App) Router() *gin.Engine {
	return a.router
}
