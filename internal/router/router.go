// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package router

import (
	"net/http"

	"github.com/aptlogica/sereni-base/internal/config"
	"github.com/aptlogica/sereni-base/internal/handlers"
	"github.com/aptlogica/sereni-base/internal/middleware"
	"github.com/aptlogica/sereni-base/internal/utils/response"
	responseConstants "github.com/aptlogica/sereni-base/internal/utils/response/constants"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

const (
	RouteCreate = "/create"
)

type Middlewares struct {
	CORS                                       func() gin.HandlerFunc
	RequestLogger                              func() gin.HandlerFunc
	DatabaseQueryLogger                        func() gin.HandlerFunc
	RequestSizeLimit                           func(int64) gin.HandlerFunc
	AuthMiddleware                             func() gin.HandlerFunc
	FileSizeLimitMiddleware                    func() gin.HandlerFunc
	ScopeHeaderMiddleware                      func(scope string) gin.HandlerFunc
	WorkspaceAndBaseAccessValidationMiddleware func(allowedAccess []string) gin.HandlerFunc
}

type Handlers struct {
	Auth         *handlers.AuthHandler
	Workspace    *handlers.WorkspaceHandler
	Base         *handlers.BaseHandler
	Asset        *handlers.AssetsHandler
	Table        *handlers.TableHandler
	User         *handlers.UserHandler
	Organization *handlers.OrganizationHandler
}

func Setup(cfg *config.Config,
	handlerGroups Handlers,
	middlewareGroups Middlewares,
) *gin.Engine {

	r := gin.Default()

	// Global middleware
	r.Use(middlewareGroups.CORS())
	r.Use(middleware.RequestID())
	r.Use(middlewareGroups.RequestLogger())
	r.Use(middlewareGroups.DatabaseQueryLogger())
	r.Use(gin.Recovery())
	r.MaxMultipartMemory = 100 << 20 // 100MB

	r.Static("/assets", "./assets")

	// Swagger UI (serves at /swagger/index.html)
	r.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL("/swagger/doc.json"),
	))

	// API routes
	api := r.Group("/api/v1")

	// Health check endpoint
	api.GET("/health", func(c *gin.Context) {
		response.SendSuccess(c, responseConstants.CoreSuccess.HealthCheck, gin.H{
			"status":  "ok",
			"message": "Serenibase is running",
			"version": cfg.Server.Version,
			"features": []string{
				"Dynamic table creation",
				"Complex filtering",
				"Relationship joins",
				"Aggregation functions",
				"Full-text search",
				"Range queries",
				"Views management",
			},
		})
	})

	api.GET("/health/live", handlerGroups.Auth.HealthLive)

	// Public Auth Routes
	setupAuthRoutes(api, handlerGroups)

	// Protected Routes
	private := api.Group("")
	private.Use(middlewareGroups.AuthMiddleware())
	{
		setupUserRoutes(private, handlerGroups)
		setupOrganizationRoutes(private, handlerGroups)
		setupWorkspaceRoutes(private, handlerGroups)
		setupBaseRoutes(private, handlerGroups)
		setupTableRoutes(private, handlerGroups)
		setupColumnRoutes(private, handlerGroups)
		setupRowRoutes(private, handlerGroups, middlewareGroups)
		setupViewRoutes(private, handlerGroups)
		setupAssetRoutes(private, handlerGroups, middlewareGroups)
	}

	return r
}

// setupAuthRoutes configures public authentication endpoints
func setupAuthRoutes(api *gin.RouterGroup, handlers Handlers) {
	auth := api.Group("/auth")
	{
		auth.POST("/login", handlers.Auth.LoginUser)
		auth.POST("/forgot-password", handlers.Auth.ForgotPassword)
		auth.POST("/reset-password", handlers.Auth.ResetPassword)
		auth.POST("/validate-token", handlers.Auth.ValidateToken)
		auth.POST("/verify-token", handlers.Auth.VerifyToken)
		auth.POST("/refresh", handlers.Auth.RefreshToken)
		auth.POST("/logout", handlers.Auth.Logout)

		otp := auth.Group("/otp")
		{
			otp.POST("/verify", handlers.Auth.VerifyEmail)
			otp.POST("/resend", handlers.Auth.ResendOTP)
		}
	}
}

// setupUserRoutes configures user management endpoints
func setupUserRoutes(private *gin.RouterGroup, handlers Handlers) {
	user := private.Group("/user")
	{
		// User profile endpoints
		user.GET("/profile/:id", handlers.User.GetUserProfileByID)
		user.PATCH("/profile/:id", handlers.User.UpdateUserProfile)
		user.POST("/change-password/:id", handlers.Auth.UpdatePassword)
		user.POST("/profile/:id/avatar", handlers.User.AddAvatar)
		user.DELETE("/profile/:id/avatar", handlers.User.RemoveAvatar)
		user.GET("/workspaces", handlers.User.GetWorkspaces)
		user.GET("/access-details", handlers.User.GetUserAccessDetails)
		user.GET("/roles-and-access/:id", handlers.User.GetUserRolesAndAccess)
		user.POST("/assign", handlers.Auth.AssignUserToWorkspace)
		user.PUT("/access/update", handlers.Auth.UpdateUserAccess)

		// Admin user management endpoints
		user.POST(RouteCreate, handlers.Auth.AddUser)
		user.POST("/edit", handlers.Auth.EditUser)
		user.POST("/remove", handlers.Auth.RemoveUser)
		user.POST("/activate", handlers.Auth.ActivateUser)
		user.POST("/deactivate", handlers.Auth.DeactivateUser)
		user.GET("/list", handlers.Auth.GetUsers)
		user.GET("/list-for-assign", handlers.Auth.GetActiveUsersForAssign)
	}
}

// setupOrganizationRoutes configures organization management endpoints
func setupOrganizationRoutes(private *gin.RouterGroup, handlers Handlers) {
	organization := private.Group("/organization")
	{
		organization.GET("", handlers.Organization.GetAllOrganizations)
		organization.PUT("/:id", handlers.Organization.UpdateOrganization)
	}
}

// setupWorkspaceRoutes configures workspace management endpoints
func setupWorkspaceRoutes(private *gin.RouterGroup, handlers Handlers) {
	workspace := private.Group("/workspace")
	{
		// Admin operations
		workspace.POST(RouteCreate, handlers.Workspace.CreateWorkspace)
		workspace.GET("/", handlers.Workspace.GetAllWorkspaces)
		workspace.GET("/:id/tables", handlers.Workspace.GetTablesByWorkspaceId)
		workspace.PUT("/:id", handlers.Workspace.UpdateWorkspace)
		workspace.DELETE("/:id", handlers.Workspace.DeleteWorkspace)

		// Full access operations
		workspace.POST("/:id/remove", handlers.Auth.RemoveUserFromWorkspace)
		workspace.GET("/:id/members", handlers.Auth.GetWorkspaceMembers)
		workspace.GET("/:id/members-with-roles", handlers.Auth.GetWorkspaceMembersWithRole)
		workspace.POST("/:id/bulk-add-members", handlers.Workspace.BulkAddMembers)
		workspace.DELETE("/access/:id", handlers.Auth.RemoveAccessMemberByID)
		// All access operations
		workspace.GET("/:id/bases", handlers.Workspace.GetBasesByWorkspaceId)
		workspace.GET("/:id", handlers.Workspace.GetWorkspaceByID)
	}
}

// setupBaseRoutes configures base management endpointsmembers-with-roles
func setupBaseRoutes(private *gin.RouterGroup, handlers Handlers) {
	base := private.Group("/base")
	{
		// Admin operations
		base.POST(RouteCreate, handlers.Base.CreateBase)

		// Full access operations - member management with specific routes before dynamic :id routes
		base.POST("/:id/remove", handlers.Auth.RemoveUserFromBase)
		base.GET("/:id/members", handlers.Auth.GetBaseMembers)
		base.GET("/:id/members-with-roles", handlers.Auth.GetBaseMembersWithRole)
		base.POST("/:id/bulk-add-members", handlers.Workspace.BulkAddBaseMembers)
		base.DELETE("/access/:id", handlers.Auth.RemoveAccessMemberByID)

		// Image operations
		base.POST("/:id/image", handlers.Base.AddBaseImage)
		base.DELETE("/:id/image", handlers.Base.RemoveBaseImage)

		// Base CRUD operations
		base.PUT("/:id", handlers.Base.UpdateBase)
		base.DELETE("/:id", handlers.Base.DeleteBase)

		// All access operations
		base.GET("/:id", handlers.Base.GetBaseByID)
		base.GET("/:id/tables", handlers.Base.GetTablesByBaseId)
	}
}

// setupTableRoutes configures table management endpoints
func setupTableRoutes(private *gin.RouterGroup, handlers Handlers) {
	table := private.Group("/table")
	{
		table.POST(RouteCreate, handlers.Table.CreateTable)
		table.POST("/import", handlers.Table.ImportTable)
		table.PATCH("/:id", handlers.Table.UpdateTable)
		table.GET("/:id", handlers.Table.GetTableByID)
		table.GET("/", handlers.Table.GetAllTables)
		table.GET("/:id/columns", handlers.Table.GetColumnsByTable)
		table.GET("/:id/views", handlers.Table.GetViewsByModelID)
		table.GET("/:id/records", handlers.Table.GetAllRecords)
		table.DELETE("/:id", handlers.Table.DeleteTable)
	}
}

// setupColumnRoutes configures column management endpoints
func setupColumnRoutes(private *gin.RouterGroup, handlers Handlers) {
	column := private.Group("/column")
	{
		column.POST(RouteCreate, handlers.Table.AddColumn)
		column.GET("/:id", handlers.Table.GetColumnById)
		column.GET("/", handlers.Table.GetAllColumns)
		column.PATCH("/:id", handlers.Table.UpdateColumn)
		column.DELETE("/:id", handlers.Table.DeleteColumn)
		column.POST("/reorder", handlers.Table.ReorderColumn)
		column.POST("/bulk-update", handlers.Table.BulkUpdateColumns)
		column.POST("/reset", handlers.Table.ResetColumnValues)
	}
}

// setupRowRoutes configures row management endpoints
func setupRowRoutes(private *gin.RouterGroup, handlers Handlers, middlewares Middlewares) {
	row := private.Group("/row")
	{
		row.POST(RouteCreate, handlers.Table.CreateRow)
		row.PATCH("/update", handlers.Table.UpdateRow)
		row.POST("/remove", handlers.Table.DeleteRow)
		row.POST("/bulk-remove", handlers.Table.BulkDeleteRows)
		row.POST("/data/insert", handlers.Table.InsertRowData)
		row.POST("/data/relation", handlers.Table.InsertRowDataForLinks)

		// Attachment endpoints with file size limit
		am := row.Group("")
		am.Use(middlewares.FileSizeLimitMiddleware())
		am.POST("/attachment/add", handlers.Table.AddAttachment)
		row.POST("/attachment/update", handlers.Table.UpdateAttachment)

		row.POST("/attachment/remove", handlers.Table.RemoveAttachments)
	}
}

// setupViewRoutes configures view management endpoints
func setupViewRoutes(private *gin.RouterGroup, handlers Handlers) {
	view := private.Group("/view")
	{
		view.POST(RouteCreate, handlers.Table.CreateView)
		view.GET("/:id", handlers.Table.GetViewByID)
		view.GET("/", handlers.Table.GetAllViews)
		view.PATCH("/:id", handlers.Table.UpdateView)
		view.DELETE("/:id", handlers.Table.DeleteView)
	}
}

// setupAssetRoutes configures asset management endpoints
func setupAssetRoutes(private *gin.RouterGroup, handlers Handlers, middlewares Middlewares) {
	asset := private.Group("/asset")
	{
		am := asset.Group("")
		am.Use(middlewares.FileSizeLimitMiddleware())
		am.POST("/upload", handlers.Asset.Upload)
		am.POST("/upload-image", handlers.Asset.UploadImage)

		asset.POST("/bulk", handlers.Asset.GetBulkAssets)
		asset.PATCH("/:id", handlers.Asset.UpdateAssetByID)
		asset.DELETE("/:id", handlers.Asset.DeleteAssetByID)
	}
}
