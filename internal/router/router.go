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

	// authRateLimitPerMinute caps login/reset/OTP attempts per client IP.
	authRateLimitPerMinute = 20
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
	ScopeLookups                               middleware.ScopeLookups
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
	credentialLimiter := middleware.RateLimiter(authRateLimitPerMinute)
	{
		auth.POST("/login", credentialLimiter, handlers.Auth.LoginUser)
		auth.POST("/forgot-password", credentialLimiter, handlers.Auth.ForgotPassword)
		auth.POST("/reset-password", credentialLimiter, handlers.Auth.ResetPassword)
		auth.POST("/validate-token", handlers.Auth.ValidateToken)
		auth.POST("/verify-token", handlers.Auth.VerifyToken)
		auth.POST("/refresh", handlers.Auth.RefreshToken)
		auth.POST("/logout", handlers.Auth.Logout)

		otp := auth.Group("/otp")
		{
			otp.POST("/verify", credentialLimiter, handlers.Auth.VerifyEmail)
			otp.POST("/resend", credentialLimiter, handlers.Auth.ResendOTP)
		}
	}
}

// setupUserRoutes configures user management endpoints
func setupUserRoutes(private *gin.RouterGroup, handlers Handlers, middlewares Middlewares) {
	user := private.Group("/user")
	// :id is a user ID: only that user may modify it; owner/co-owner may also read it
	selfOnly := middleware.NewSelfOrRoleGuard(nil, middlewares.AccessMemberService)
	selfOrAdmin := middleware.NewSelfOrRoleGuard(
		[]string{appConstant.RBACRoleNames.Owner, appConstant.RBACRoleNames.CoOwner},
		middlewares.AccessMemberService)
	{
		// User profile endpoints
		user.GET("/profile/:id", selfOrAdmin, handlers.User.GetUserProfileByID)
		user.PATCH("/profile/:id", selfOnly, handlers.User.UpdateUserProfile)
		user.POST("/change-password/:id", selfOnly, handlers.Auth.UpdatePassword)
		user.POST("/profile/:id/avatar", selfOnly, handlers.User.AddAvatar)
		user.DELETE("/profile/:id/avatar", selfOnly, handlers.User.RemoveAvatar)
		user.GET("/workspaces", handlers.User.GetWorkspaces)
		user.GET("/access-details", handlers.User.GetUserAccessDetails)
		user.GET("/roles-and-access/:id", selfOrAdmin, handlers.User.GetUserRolesAndAccess)

		// Member assignment endpoints (owner, co-owner, maintainer)
		// Members:Invite must be held in every workspace the body grants access in
		user.POST("/assign",
			objectGuard(middlewares, appConstant.ResourceCodes.Members, appConstant.ActionCodes.Invite, middlewares.ScopeLookups.MembershipBody()),
			handlers.Auth.AssignUserToWorkspace)
		user.PUT("/access/update",
			objectGuard(middlewares, appConstant.ResourceCodes.Members, appConstant.ActionCodes.Invite, middlewares.ScopeLookups.MembershipBody()),
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

// objectGuard requires resourceCode/actionCode on a membership that covers every scope target
// resolve returns, binding the permission to the workspace or base the request acts on.
func objectGuard(m Middlewares, resourceCode, actionCode string, resolve middleware.TargetResolver) gin.HandlerFunc {
	return middleware.NewObjectPermissionGuard(resourceCode, actionCode, m.AccessMemberService, resolve)
}

// adminOnly restricts tenant-wide listings to owner/co-owner.
func adminOnly(m Middlewares) gin.HandlerFunc {
	return middleware.NewRoleGuard(
		[]string{appConstant.RBACRoleNames.Owner, appConstant.RBACRoleNames.CoOwner},
		m.AccessMemberService, "").Middleware()
}

// setupWorkspaceRoutes configures workspace management endpoints
func setupWorkspaceRoutes(private *gin.RouterGroup, handlers Handlers, middlewares Middlewares) {
	workspace := private.Group("/workspace")
	lk := middlewares.ScopeLookups
	// Read-only membership check: members of the workspace or of one of its bases
	wsAccess := middleware.NewScopeAccessGuard(appConstant.ScopeLevels.Workspace, middlewares.AccessMemberService, nil)
	wsParam := lk.WorkspaceParam("id")
	{
		workspace.POST(RouteCreate,
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Workspace, appConstant.ActionCodes.Create, middlewares.AccessMemberService).Middleware(),
			handlers.Workspace.CreateWorkspace)
		// Lists every workspace in the tenant; other roles use /user/workspaces
		workspace.GET("/", adminOnly(middlewares), handlers.Workspace.GetAllWorkspaces)
		workspace.GET("/:id/tables",
			objectGuard(middlewares, appConstant.ResourceCodes.Workspace, appConstant.ActionCodes.Read, wsParam),
			handlers.Workspace.GetTablesByWorkspaceId)
		workspace.PUT("/:id",
			objectGuard(middlewares, appConstant.ResourceCodes.Workspace, appConstant.ActionCodes.Update, wsParam),
			handlers.Workspace.UpdateWorkspace)
		workspace.DELETE("/:id",
			objectGuard(middlewares, appConstant.ResourceCodes.Workspace, appConstant.ActionCodes.Delete, wsParam),
			handlers.Workspace.DeleteWorkspace)

		// Member management
		workspace.POST("/:id/remove",
			objectGuard(middlewares, appConstant.ResourceCodes.Members, appConstant.ActionCodes.Manage, wsParam),
			handlers.Auth.RemoveUserFromWorkspace)
		workspace.GET("/:id/members",
			objectGuard(middlewares, appConstant.ResourceCodes.Members, appConstant.ActionCodes.Read, wsParam),
			handlers.Auth.GetWorkspaceMembers)
		workspace.GET("/:id/members-with-roles",
			objectGuard(middlewares, appConstant.ResourceCodes.Members, appConstant.ActionCodes.Read, wsParam),
			handlers.Auth.GetWorkspaceMembersWithRole)
		workspace.POST("/:id/bulk-add-members",
			objectGuard(middlewares, appConstant.ResourceCodes.Members, appConstant.ActionCodes.Invite, wsParam),
			handlers.Workspace.BulkAddMembers)
		workspace.DELETE("/access/:id",
			objectGuard(middlewares, appConstant.ResourceCodes.Members, appConstant.ActionCodes.Manage,
				lk.AccessMemberParam("id", appConstant.ScopeLevels.Workspace)),
			handlers.Auth.RemoveAccessMemberByID)
		// All access operations
		workspace.GET("/:id/bases", wsAccess, handlers.Workspace.GetBasesByWorkspaceId)
		workspace.GET("/:id", wsAccess, handlers.Workspace.GetWorkspaceByID)
	}
}

// setupBaseRoutes configures base management endpoints
func setupBaseRoutes(private *gin.RouterGroup, handlers Handlers, middlewares Middlewares) {
	base := private.Group("/base")
	lk := middlewares.ScopeLookups
	// Read-only membership check: members of the base or of its workspace
	baseAccess := middleware.NewScopeAccessGuard(appConstant.ScopeLevels.Base, middlewares.AccessMemberService, lk.BaseWorkspace)
	baseParam := lk.BaseParam("id")
	{
		// Base:Create must be granted in the workspace named by workspace_id
		base.POST(RouteCreate,
			objectGuard(middlewares, appConstant.ResourceCodes.Base, appConstant.ActionCodes.Create, lk.WorkspaceForm("workspace_id")),
			handlers.Base.CreateBase)

		// Member management - specific routes before dynamic :id routes
		base.POST("/:id/remove",
			objectGuard(middlewares, appConstant.ResourceCodes.Members, appConstant.ActionCodes.Manage, baseParam),
			handlers.Auth.RemoveUserFromBase)
		base.GET("/:id/members",
			baseAccess,
			middleware.NewRoleGuard(
				[]string{appConstant.RBACRoleNames.Owner, appConstant.RBACRoleNames.CoOwner, appConstant.RBACRoleNames.WorkspaceMaintainer, appConstant.RBACRoleNames.WorkspaceMaintainerRO, appConstant.RBACRoleNames.BaseMember, appConstant.RBACRoleNames.BaseMemberReadOnly},
				middlewares.AccessMemberService, "").Middleware(),
			handlers.Auth.GetBaseMembers)
		base.GET("/:id/members-with-roles",
			baseAccess,
			middleware.NewRoleGuard(
				[]string{appConstant.RBACRoleNames.Owner, appConstant.RBACRoleNames.CoOwner, appConstant.RBACRoleNames.WorkspaceMaintainer, appConstant.RBACRoleNames.WorkspaceMaintainerRO, appConstant.RBACRoleNames.BaseMember, appConstant.RBACRoleNames.BaseMemberReadOnly},
				middlewares.AccessMemberService, "").Middleware(),
			handlers.Auth.GetBaseMembersWithRole)
		base.POST("/:id/bulk-add-members",
			objectGuard(middlewares, appConstant.ResourceCodes.Members, appConstant.ActionCodes.Invite, baseParam),
			handlers.Workspace.BulkAddBaseMembers)
		base.DELETE("/access/:id",
			objectGuard(middlewares, appConstant.ResourceCodes.Members, appConstant.ActionCodes.Manage,
				lk.AccessMemberParam("id", appConstant.ScopeLevels.Base)),
			handlers.Auth.RemoveAccessMemberByID)

		// Image operations
		base.POST("/:id/image",
			objectGuard(middlewares, appConstant.ResourceCodes.Base, appConstant.ActionCodes.Update, baseParam),
			handlers.Base.AddBaseImage)
		base.DELETE("/:id/image",
			objectGuard(middlewares, appConstant.ResourceCodes.Base, appConstant.ActionCodes.Update, baseParam),
			handlers.Base.RemoveBaseImage)

		// Base CRUD operations
		base.PUT("/:id",
			objectGuard(middlewares, appConstant.ResourceCodes.Base, appConstant.ActionCodes.Update, baseParam),
			handlers.Base.UpdateBase)
		base.DELETE("/:id",
			objectGuard(middlewares, appConstant.ResourceCodes.Base, appConstant.ActionCodes.Delete, baseParam),
			handlers.Base.DeleteBase)

		// All access operations
		base.GET("/:id", baseAccess, handlers.Base.GetBaseByID)
		base.GET("/:id/tables", baseAccess, handlers.Base.GetTablesByBaseId)
	}
}

// setupTableRoutes configures table management endpoints
func setupTableRoutes(private *gin.RouterGroup, handlers Handlers, middlewares Middlewares) {
	table := private.Group("/table")
	lk := middlewares.ScopeLookups
	modelParam := lk.ModelParam("id")
	{
		table.POST(RouteCreate,
			objectGuard(middlewares, appConstant.ResourceCodes.Table, appConstant.ActionCodes.Create, lk.TableCreateBody()),
			handlers.Table.CreateTable)
		table.POST("/import",
			objectGuard(middlewares, appConstant.ResourceCodes.Table, appConstant.ActionCodes.Create, lk.TableImportForm()),
			handlers.Table.ImportTableWithConfig)
		table.PATCH("/:id",
			objectGuard(middlewares, appConstant.ResourceCodes.Table, appConstant.ActionCodes.Update, modelParam),
			handlers.Table.UpdateTable)

		table.GET("/:id",
			objectGuard(middlewares, appConstant.ResourceCodes.Table, appConstant.ActionCodes.Read, modelParam),
			handlers.Table.GetTableByID)
		// Lists every table in the tenant
		table.GET("/", adminOnly(middlewares), handlers.Table.GetAllTables)
		table.GET("/:id/columns",
			objectGuard(middlewares, appConstant.ResourceCodes.Table, appConstant.ActionCodes.Read, modelParam),
			handlers.Table.GetColumnsByTable)
		table.GET("/:id/views",
			objectGuard(middlewares, appConstant.ResourceCodes.Table, appConstant.ActionCodes.Read, modelParam),
			handlers.Table.GetViewsByModelID)
		table.GET("/:id/records",
			objectGuard(middlewares, appConstant.ResourceCodes.Records, appConstant.ActionCodes.Read, modelParam),
			handlers.Table.GetAllRecords)

		table.DELETE("/:id",
			objectGuard(middlewares, appConstant.ResourceCodes.Table, appConstant.ActionCodes.Delete, modelParam),
			handlers.Table.DeleteTable)
	}
}

// setupColumnRoutes configures column management endpoints
func setupColumnRoutes(private *gin.RouterGroup, handlers Handlers, middlewares Middlewares) {
	column := private.Group("/column")
	lk := middlewares.ScopeLookups
	columnParam := lk.ColumnParam("id")
	// Data-enhancement operations take model_id (and column_id) in the JSON body
	updateByBody := objectGuard(middlewares, appConstant.ResourceCodes.Table, appConstant.ActionCodes.Update, lk.ModelBody())
	{
		column.POST(RouteCreate,
			objectGuard(middlewares, appConstant.ResourceCodes.Table, appConstant.ActionCodes.Create, lk.ModelBody()),
			handlers.Table.AddColumn)

		column.GET("/:id",
			objectGuard(middlewares, appConstant.ResourceCodes.Table, appConstant.ActionCodes.Read, columnParam),
			handlers.Table.GetColumnById)
		// Lists every column in the tenant
		column.GET("/", adminOnly(middlewares), handlers.Table.GetAllColumns)

		column.PATCH("/:id",
			objectGuard(middlewares, appConstant.ResourceCodes.Table, appConstant.ActionCodes.Update, columnParam),
			handlers.Table.UpdateColumn)
		column.DELETE("/:id",
			objectGuard(middlewares, appConstant.ResourceCodes.Table, appConstant.ActionCodes.Delete, columnParam),
			handlers.Table.DeleteColumn)
		column.POST("/reorder",
			objectGuard(middlewares, appConstant.ResourceCodes.Table, appConstant.ActionCodes.Update,
				lk.ColumnPairBody("source_column_id", "target_column_id")),
			handlers.Table.ReorderColumn)
		column.POST("/bulk-update", updateByBody, handlers.Table.BulkUpdateColumns)
		column.POST("/reset", updateByBody, handlers.Table.ResetColumnValues)
		column.POST("/trim-whitespace", updateByBody, handlers.Table.TrimWhitespace)
		column.POST("/find-replace", updateByBody, handlers.Table.FindReplace)
		column.POST("/case-normalize", updateByBody, handlers.Table.CaseNormalization)
		column.POST("/remove-special-characters", updateByBody, handlers.Table.RemoveSpecialCharacters)
		column.POST("/split", updateByBody, handlers.Table.ColumnSplit)
		column.POST("/remove-formatting", updateByBody, handlers.Table.RemoveFormatting)
		column.POST("/remove-duplicates", updateByBody, handlers.Table.RemoveDuplicates)
		column.POST("/merge-columns", updateByBody, handlers.Table.MergeColumns)
		column.POST("/extract-substring", updateByBody, handlers.Table.ExtractSubstring)
	}
}

// setupRowRoutes configures row management endpoints
func setupRowRoutes(private *gin.RouterGroup, handlers Handlers, middlewares Middlewares) {
	row := private.Group("/row")
	lk := middlewares.ScopeLookups
	records := func(action string) gin.HandlerFunc {
		return objectGuard(middlewares, appConstant.ResourceCodes.Records, action, lk.ModelBody())
	}
	{
		row.POST(RouteCreate, records(appConstant.ActionCodes.Create), handlers.Table.CreateRow)
		row.PATCH("/update", records(appConstant.ActionCodes.Update), handlers.Table.UpdateRow)

		row.POST("/remove", records(appConstant.ActionCodes.Delete), handlers.Table.DeleteRow)
		row.POST("/bulk-remove", records(appConstant.ActionCodes.Delete), handlers.Table.BulkDeleteRows)

		row.POST("/data/insert", records(appConstant.ActionCodes.Create), handlers.Table.InsertRowData)
		row.POST("/data/relation", records(appConstant.ActionCodes.Create), handlers.Table.InsertRowDataForLinks)

		// Attachment endpoints with file size limit; add is multipart, so model_id is a form field
		am := row.Group("")
		am.Use(middlewares.FileSizeLimitMiddleware())
		am.POST("/attachment/add",
			objectGuard(middlewares, appConstant.ResourceCodes.Records, appConstant.ActionCodes.Create, lk.ModelForm()),
			handlers.Table.AddAttachment)
		row.POST("/attachment/update", records(appConstant.ActionCodes.Create), handlers.Table.UpdateAttachment)
		row.POST("/attachment/remove", records(appConstant.ActionCodes.Delete), handlers.Table.RemoveAttachments)
	}
}

// setupViewRoutes configures view management endpoints
func setupViewRoutes(private *gin.RouterGroup, handlers Handlers, middlewares Middlewares) {
	view := private.Group("/view")
	lk := middlewares.ScopeLookups
	viewParam := lk.ViewParam("id")
	{
		view.POST(RouteCreate,
			objectGuard(middlewares, appConstant.ResourceCodes.Views, appConstant.ActionCodes.Create, lk.ModelBody()),
			handlers.Table.CreateView)

		view.GET("/:id",
			objectGuard(middlewares, appConstant.ResourceCodes.Views, appConstant.ActionCodes.Read, viewParam),
			handlers.Table.GetViewByID)
		// Lists views across the tenant; the handler drops tables the caller cannot access
		view.GET("/",
			middleware.NewPermissionGuard(appConstant.ResourceCodes.Views, appConstant.ActionCodes.Read, middlewares.AccessMemberService).Middleware(),
			middleware.NewModelAccessFilter(middlewares.AccessMemberService, lk),
			handlers.Table.GetAllViews)

		view.PATCH("/:id",
			objectGuard(middlewares, appConstant.ResourceCodes.Views, appConstant.ActionCodes.Update, viewParam),
			handlers.Table.UpdateView)
		view.DELETE("/:id",
			objectGuard(middlewares, appConstant.ResourceCodes.Views, appConstant.ActionCodes.Delete, viewParam),
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

		// Assets carry no link to a workspace or base, so direct access by ID is limited to
		// owner/co-owner. Attachments are managed through the /row/attachment routes.
		asset.POST("/bulk", adminOnly(middlewares), handlers.Asset.GetBulkAssets)
		asset.PATCH("/:id", adminOnly(middlewares), handlers.Asset.UpdateAssetByID)
		asset.DELETE("/:id", adminOnly(middlewares), handlers.Asset.DeleteAssetByID)
	}
}
