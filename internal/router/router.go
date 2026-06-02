// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package router

import (
	"net/http"

	"github.com/aptlogica/sereni-base/internal/config"
	appConstant "github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/handlers"
	"github.com/aptlogica/sereni-base/internal/middleware"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
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
	AccessMemberService                        interfaces.AccessMemberService
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
		setupUserRoutes(private, handlerGroups, middlewareGroups)
		setupOrganizationRoutes(private, handlerGroups, middlewareGroups)
		setupWorkspaceRoutes(private, handlerGroups, middlewareGroups)
		setupBaseRoutes(private, handlerGroups, middlewareGroups)
		setupTableRoutes(private, handlerGroups, middlewareGroups)
		setupColumnRoutes(private, handlerGroups, middlewareGroups)
		setupRowRoutes(private, handlerGroups, middlewareGroups)
		setupViewRoutes(private, handlerGroups, middlewareGroups)
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
func setupUserRoutes(private *gin.RouterGroup, handlers Handlers, middlewares Middlewares) {
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

		// Member assignment endpoints (owner, co-owner, maintainer)
		user.POST("/assign",
			middleware.NewRoleGuard(
				[]string{appConstant.RBACRoleNames.Owner, appConstant.RBACRoleNames.CoOwner, appConstant.RBACRoleNames.WorkspaceMaintainer},
				middlewares.AccessMemberService, "").Middleware(),
			handlers.Auth.AssignUserToWorkspace)
		user.PUT("/access/update",
			middleware.NewRoleGuard(
				[]string{appConstant.RBACRoleNames.Owner, appConstant.RBACRoleNames.CoOwner, appConstant.RBACRoleNames.WorkspaceMaintainer},
				middlewares.AccessMemberService, "").Middleware(),
			handlers.Auth.UpdateUserAccess)

		// System-level admin user management endpoints (owner, co-owner only)
		user.POST(RouteCreate,
			middleware.NewRoleGuard(
				[]string{appConstant.RBACRoleNames.Owner, appConstant.RBACRoleNames.CoOwner},
				middlewares.AccessMemberService, "").Middleware(),
			handlers.Auth.AddUser)
		user.POST("/edit",
			middleware.NewRoleGuard(
				[]string{appConstant.RBACRoleNames.Owner, appConstant.RBACRoleNames.CoOwner},
				middlewares.AccessMemberService, "").Middleware(),
			handlers.Auth.EditUser)
		user.POST("/remove",
			middleware.NewRoleGuard(
				[]string{appConstant.RBACRoleNames.Owner, appConstant.RBACRoleNames.CoOwner},
				middlewares.AccessMemberService, "").Middleware(),
			handlers.Auth.RemoveUser)
		user.POST("/activate",
			middleware.NewRoleGuard(
				[]string{appConstant.RBACRoleNames.Owner, appConstant.RBACRoleNames.CoOwner},
				middlewares.AccessMemberService, "").Middleware(),
			handlers.Auth.ActivateUser)
		user.POST("/deactivate",
			middleware.NewRoleGuard(
				[]string{appConstant.RBACRoleNames.Owner, appConstant.RBACRoleNames.CoOwner},
				middlewares.AccessMemberService, "").Middleware(),
			handlers.Auth.DeactivateUser)
		user.GET("/list",
			middleware.NewRoleGuard(
				[]string{appConstant.RBACRoleNames.Owner, appConstant.RBACRoleNames.CoOwner, appConstant.RBACRoleNames.WorkspaceMaintainer},
				middlewares.AccessMemberService, "").Middleware(),
			handlers.Auth.GetUsers)
		user.GET("/list-for-assign",
			middleware.NewRoleGuard(
				[]string{appConstant.RBACRoleNames.Owner, appConstant.RBACRoleNames.CoOwner, appConstant.RBACRoleNames.WorkspaceMaintainer},
				middlewares.AccessMemberService, "").Middleware(),
			handlers.Auth.GetActiveUsersForAssign)
	}
}

// setupOrganizationRoutes configures organization management endpoints
func setupOrganizationRoutes(private *gin.RouterGroup, handlers Handlers, middlewares Middlewares) {
	organization := private.Group("/organization")
	{
		organization.GET("",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Settings, appConstant.ActionCodes.Read, middlewares.AccessMemberService).Middleware(),
			handlers.Organization.GetAllOrganizations)
		organization.PUT("/:id",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Settings, appConstant.ActionCodes.Update, middlewares.AccessMemberService).Middleware(),
			handlers.Organization.UpdateOrganization)
	}
}

// setupWorkspaceRoutes configures workspace management endpoints
func setupWorkspaceRoutes(private *gin.RouterGroup, handlers Handlers, middlewares Middlewares) {
	workspace := private.Group("/workspace")
	{
		// Admin operations with permission-based guards
		workspace.POST(RouteCreate,
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Workspace, appConstant.ActionCodes.Create, middlewares.AccessMemberService).Middleware(),
			handlers.Workspace.CreateWorkspace)
		workspace.GET("/",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Workspace, appConstant.ActionCodes.Read, middlewares.AccessMemberService).Middleware(),
			handlers.Workspace.GetAllWorkspaces)
		workspace.GET("/:id/tables",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Workspace, appConstant.ActionCodes.Read, middlewares.AccessMemberService).Middleware(),
			handlers.Workspace.GetTablesByWorkspaceId)
		workspace.PUT("/:id",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Workspace, appConstant.ActionCodes.Update, middlewares.AccessMemberService).Middleware(),
			handlers.Workspace.UpdateWorkspace)
		workspace.DELETE("/:id",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Workspace, appConstant.ActionCodes.Delete, middlewares.AccessMemberService).Middleware(),
			handlers.Workspace.DeleteWorkspace)

		// Full access operations
		workspace.POST("/:id/remove",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Members, appConstant.ActionCodes.Manage, middlewares.AccessMemberService).Middleware(),
			handlers.Auth.RemoveUserFromWorkspace)
		workspace.GET("/:id/members",
			middleware.NewRoleGuard(
				[]string{appConstant.RBACRoleNames.Owner, appConstant.RBACRoleNames.CoOwner, appConstant.RBACRoleNames.WorkspaceMaintainer, appConstant.RBACRoleNames.WorkspaceMaintainerRO},
				middlewares.AccessMemberService, "").Middleware(),
			handlers.Auth.GetWorkspaceMembers)
		workspace.GET("/:id/members-with-roles",
			middleware.NewRoleGuard(
				[]string{appConstant.RBACRoleNames.Owner, appConstant.RBACRoleNames.CoOwner, appConstant.RBACRoleNames.WorkspaceMaintainer, appConstant.RBACRoleNames.WorkspaceMaintainerRO},
				middlewares.AccessMemberService, "").Middleware(),
			handlers.Auth.GetWorkspaceMembersWithRole)
		workspace.POST("/:id/bulk-add-members",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Members, appConstant.ActionCodes.Invite, middlewares.AccessMemberService).Middleware(),
			handlers.Workspace.BulkAddMembers)
		workspace.DELETE("/access/:id",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Members, appConstant.ActionCodes.Manage, middlewares.AccessMemberService).Middleware(),
			handlers.Auth.RemoveAccessMemberByID)
		// All access operations
		workspace.GET("/:id/bases", handlers.Workspace.GetBasesByWorkspaceId)
		workspace.GET("/:id", handlers.Workspace.GetWorkspaceByID)
	}
}

// setupBaseRoutes configures base management endpoints
func setupBaseRoutes(private *gin.RouterGroup, handlers Handlers, middlewares Middlewares) {
	base := private.Group("/base")
	{
		// Admin operations with permission-based guards
		base.POST(RouteCreate,
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Base, appConstant.ActionCodes.Create, middlewares.AccessMemberService).Middleware(),
			handlers.Base.CreateBase)

		// Full access operations - member management with specific routes before dynamic :id routes
		base.POST("/:id/remove",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Members, appConstant.ActionCodes.Manage, middlewares.AccessMemberService).Middleware(),
			handlers.Auth.RemoveUserFromBase)
		base.GET("/:id/members",
			middleware.NewRoleGuard(
				[]string{appConstant.RBACRoleNames.Owner, appConstant.RBACRoleNames.CoOwner, appConstant.RBACRoleNames.WorkspaceMaintainer, appConstant.RBACRoleNames.WorkspaceMaintainerRO, appConstant.RBACRoleNames.BaseMember, appConstant.RBACRoleNames.BaseMemberReadOnly},
				middlewares.AccessMemberService, "").Middleware(),
			handlers.Auth.GetBaseMembers)
		base.GET("/:id/members-with-roles",
			middleware.NewRoleGuard(
				[]string{appConstant.RBACRoleNames.Owner, appConstant.RBACRoleNames.CoOwner, appConstant.RBACRoleNames.WorkspaceMaintainer, appConstant.RBACRoleNames.WorkspaceMaintainerRO, appConstant.RBACRoleNames.BaseMember, appConstant.RBACRoleNames.BaseMemberReadOnly},
				middlewares.AccessMemberService, "").Middleware(),
			handlers.Auth.GetBaseMembersWithRole)
		base.POST("/:id/bulk-add-members",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Members, appConstant.ActionCodes.Invite, middlewares.AccessMemberService).Middleware(),
			handlers.Workspace.BulkAddBaseMembers)
		base.DELETE("/access/:id",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Members, appConstant.ActionCodes.Manage, middlewares.AccessMemberService).Middleware(),
			handlers.Auth.RemoveAccessMemberByID)

		// Image operations
		base.POST("/:id/image",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Base, appConstant.ActionCodes.Update, middlewares.AccessMemberService).Middleware(),
			handlers.Base.AddBaseImage)
		base.DELETE("/:id/image",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Base, appConstant.ActionCodes.Update, middlewares.AccessMemberService).Middleware(),
			handlers.Base.RemoveBaseImage)

		// Base CRUD operations
		base.PUT("/:id",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Base, appConstant.ActionCodes.Update, middlewares.AccessMemberService).Middleware(),
			handlers.Base.UpdateBase)
		base.DELETE("/:id",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Base, appConstant.ActionCodes.Delete, middlewares.AccessMemberService).Middleware(),
			handlers.Base.DeleteBase)

		// All access operations
		base.GET("/:id", handlers.Base.GetBaseByID)
		base.GET("/:id/tables", handlers.Base.GetTablesByBaseId)
	}
}

// setupTableRoutes configures table management endpoints
func setupTableRoutes(private *gin.RouterGroup, handlers Handlers, middlewares Middlewares) {
	table := private.Group("/table")
	{
		// Write operations require table.create permission
		table.POST(RouteCreate,
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Table, appConstant.ActionCodes.Create, middlewares.AccessMemberService).Middleware(),
			handlers.Table.CreateTable)
		table.POST("/import",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Table, appConstant.ActionCodes.Create, middlewares.AccessMemberService).Middleware(),
			handlers.Table.ImportTableWithConfig)
		table.PATCH("/:id",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Table, appConstant.ActionCodes.Update, middlewares.AccessMemberService).Middleware(),
			handlers.Table.UpdateTable)

		table.GET("/:id",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Table, appConstant.ActionCodes.Read, middlewares.AccessMemberService).Middleware(),
			handlers.Table.GetTableByID)
		table.GET("/",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Table, appConstant.ActionCodes.Read, middlewares.AccessMemberService).Middleware(),
			handlers.Table.GetAllTables)
		table.GET("/:id/columns",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Table, appConstant.ActionCodes.Read, middlewares.AccessMemberService).Middleware(),
			handlers.Table.GetColumnsByTable)
		table.GET("/:id/views",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Table, appConstant.ActionCodes.Read, middlewares.AccessMemberService).Middleware(),
			handlers.Table.GetViewsByModelID)
		table.GET("/:id/records",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Records, appConstant.ActionCodes.Read, middlewares.AccessMemberService).Middleware(),
			handlers.Table.GetAllRecords)

		// Delete requires table.delete permission
		table.DELETE("/:id",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Table, appConstant.ActionCodes.Delete, middlewares.AccessMemberService).Middleware(),
			handlers.Table.DeleteTable)
		
		// ai table create and add sample data
		table.POST("/ai", handlers.Table.PreviewAiTable)      // preview AI schema (no create)
		// table.POST("/ai/apply", handlers.Table.ApplyAiTable) // create from edited AI schema
	}
}

// setupColumnRoutes configures column management endpoints
func setupColumnRoutes(private *gin.RouterGroup, handlers Handlers, middlewares Middlewares) {
	column := private.Group("/column")
	{
		// Write operations require table.create permission
		column.POST(RouteCreate,
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Table, appConstant.ActionCodes.Create, middlewares.AccessMemberService).Middleware(),
			handlers.Table.AddColumn)

		// Read operations
		column.GET("/:id",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Table, appConstant.ActionCodes.Read, middlewares.AccessMemberService).Middleware(),
			handlers.Table.GetColumnById)
		column.GET("/",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Table, appConstant.ActionCodes.Read, middlewares.AccessMemberService).Middleware(),
			handlers.Table.GetAllColumns)

		// Update requires table.update permission
		column.PATCH("/:id",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Table, appConstant.ActionCodes.Update, middlewares.AccessMemberService).Middleware(),
			handlers.Table.UpdateColumn)

		// Delete requires table.delete permission
		column.DELETE("/:id",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Table, appConstant.ActionCodes.Delete, middlewares.AccessMemberService).Middleware(),
			handlers.Table.DeleteColumn)
		column.POST("/reorder",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Table, appConstant.ActionCodes.Update, middlewares.AccessMemberService).Middleware(),
			handlers.Table.ReorderColumn)
		column.POST("/bulk-update",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Table, appConstant.ActionCodes.Update, middlewares.AccessMemberService).Middleware(),
			handlers.Table.BulkUpdateColumns)
		column.POST("/reset",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Table, appConstant.ActionCodes.Update, middlewares.AccessMemberService).Middleware(),
			handlers.Table.ResetColumnValues)
	}
}

// setupRowRoutes configures row management endpoints
func setupRowRoutes(private *gin.RouterGroup, handlers Handlers, middlewares Middlewares) {
	row := private.Group("/row")
	{
		// Write operations require records.create permission
		row.POST(RouteCreate,
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Records, appConstant.ActionCodes.Create, middlewares.AccessMemberService).Middleware(),
			handlers.Table.CreateRow)
		row.PATCH("/update",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Records, appConstant.ActionCodes.Update, middlewares.AccessMemberService).Middleware(),
			handlers.Table.UpdateRow)
		// row.POST("/remove", handlers.Table.DeleteRow)
		// row.POST("/bulk-remove", handlers.Table.BulkDeleteRows)
		// row.POST("/data/insert", handlers.Table.InsertRowData)
		// row.POST("/data/relation", handlers.Table.InsertRowDataForLinks)

		// Delete operations require records.delete permission
		row.POST("/remove",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Records, appConstant.ActionCodes.Delete, middlewares.AccessMemberService).Middleware(),
			handlers.Table.DeleteRow)
		row.POST("/bulk-remove",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Records, appConstant.ActionCodes.Delete, middlewares.AccessMemberService).Middleware(),
			handlers.Table.BulkDeleteRows)

		// Data manipulation operations
		row.POST("/data/insert",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Records, appConstant.ActionCodes.Create, middlewares.AccessMemberService).Middleware(),
			handlers.Table.InsertRowData)
		row.POST("/data/relation",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Records, appConstant.ActionCodes.Create, middlewares.AccessMemberService).Middleware(),
			handlers.Table.InsertRowDataForLinks)

		// Attachment endpoints with file size limit (requires create permission)
		am := row.Group("")
		am.Use(middlewares.FileSizeLimitMiddleware())
		am.POST("/attachment/add",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Records, appConstant.ActionCodes.Create, middlewares.AccessMemberService).Middleware(),
			handlers.Table.AddAttachment)
		row.POST("/attachment/update",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Records, appConstant.ActionCodes.Create, middlewares.AccessMemberService).Middleware(),
			handlers.Table.UpdateAttachment)

		row.POST("/attachment/remove",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Records, appConstant.ActionCodes.Delete, middlewares.AccessMemberService).Middleware(),
			handlers.Table.RemoveAttachments)
	}
}

// setupViewRoutes configures view management endpoints
func setupViewRoutes(private *gin.RouterGroup, handlers Handlers, middlewares Middlewares) {
	view := private.Group("/view")
	{
		// Create requires views.create permission
		view.POST(RouteCreate,
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Views, appConstant.ActionCodes.Create, middlewares.AccessMemberService).Middleware(),
			handlers.Table.CreateView)

		// Read operations
		view.GET("/:id",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Views, appConstant.ActionCodes.Read, middlewares.AccessMemberService).Middleware(),
			handlers.Table.GetViewByID)
		view.GET("/",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Views, appConstant.ActionCodes.Read, middlewares.AccessMemberService).Middleware(),
			handlers.Table.GetAllViews)

		// Update requires views.update permission
		view.PATCH("/:id",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Views, appConstant.ActionCodes.Update, middlewares.AccessMemberService).Middleware(),
			handlers.Table.UpdateView)

		// Delete requires views.delete permission
		view.DELETE("/:id",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Views, appConstant.ActionCodes.Delete, middlewares.AccessMemberService).Middleware(),
			handlers.Table.DeleteView)
	}
}

// setupAssetRoutes configures asset management endpoints
func setupAssetRoutes(private *gin.RouterGroup, handlers Handlers, middlewares Middlewares) {
	asset := private.Group("/asset")
	{
		am := asset.Group("")
		am.Use(middlewares.FileSizeLimitMiddleware())
		// Upload requires records.create permission (for storing assets)
		am.POST("/upload",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Records, appConstant.ActionCodes.Create, middlewares.AccessMemberService).Middleware(),
			handlers.Asset.Upload)
		am.POST("/upload-image",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Records, appConstant.ActionCodes.Create, middlewares.AccessMemberService).Middleware(),
			handlers.Asset.UploadImage)

		// Read operations
		asset.POST("/bulk",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Records, appConstant.ActionCodes.Read, middlewares.AccessMemberService).Middleware(),
			handlers.Asset.GetBulkAssets)

		// Update requires records.create permission (for modifying assets)
		asset.PATCH("/:id",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Records, appConstant.ActionCodes.Create, middlewares.AccessMemberService).Middleware(),
			handlers.Asset.UpdateAssetByID)

		// Delete requires records.delete permission
		asset.DELETE("/:id",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Records, appConstant.ActionCodes.Delete, middlewares.AccessMemberService).Middleware(),
			handlers.Asset.DeleteAssetByID)
	}
}
