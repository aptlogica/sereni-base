package router

import (
	"serenibase/internal/config"
	"serenibase/internal/handlers"

	// "serenibase/internal/providers/email"
	// "serenibase/internal/providers/otp"

	appConstant "serenibase/internal/constant"

	"github.com/gin-gonic/gin"
)

type Middlewares struct {
	CORS                                       func() gin.HandlerFunc
	RequestLogger                              func() gin.HandlerFunc
	DatabaseQueryLogger                        func() gin.HandlerFunc
	RequestSizeLimit                           func(int64) gin.HandlerFunc
	AuthMiddleware                             func() gin.HandlerFunc
	TenantSchemaMiddleware                     func() gin.HandlerFunc
	FileSizeLimitMiddleware                    func() gin.HandlerFunc
	ScopeHeaderMiddleware                      func(scope string) gin.HandlerFunc
	WorkspaceAndBaseAccessValidationMiddleware func(allowedAccess []string) gin.HandlerFunc
}

type Handlers struct {
	Auth      *handlers.AuthHandler
	Workspace *handlers.WorkspaceHandler
	Base      *handlers.BaseHandler
	Asset     *handlers.AssetsHandler
	Table     *handlers.TableHandler
	User      *handlers.UserHandler
	Tenant    *handlers.TenantHandler
}

func Setup(cfg *config.Config,
	handlerGroups Handlers,
	middlewareGroups Middlewares,
) *gin.Engine {

	r := gin.Default()

	// Global middleware
	// r.Use(middleware.CORS())
	// Use Gin's built-in CORS middleware as a replacement
	// import "github.com/gin-contrib/cors" at the top if not already imported
	r.Use(middlewareGroups.CORS())
	r.Use(middlewareGroups.RequestLogger())
	r.Use(middlewareGroups.DatabaseQueryLogger())
	r.Use(gin.Recovery())
	r.MaxMultipartMemory = 100 << 20 // 100MB

	r.Static("/assets", "./assets")

	// API routes
	api := r.Group("/api/v1")

	// Health check endpoint
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Serenibase is running",
			"version": "1.0.0",
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

	auth := api.Group("/auth")
	{
		auth.POST("/register", handlerGroups.Auth.RegisterUser)
		auth.POST("/login", handlerGroups.Auth.LoginUser)
		auth.POST("/refresh", handlerGroups.Auth.RefreshToken)
		auth.POST("/forgot-password", handlerGroups.Auth.ForgotPassword)
		auth.POST("/reset-password", handlerGroups.Auth.ResetPassword)
		auth.POST("/validate-token", handlerGroups.Auth.ValidateToken)
		auth.POST("/verify-token", handlerGroups.Auth.VerifyToken)

		// Logout with middleware if possible (handlerGroups.Auth.Logout)
		// Assuming AuthMiddleware is available in middlewareGroups
		// auth.POST("/logout", middlewareGroups.AuthMiddleware(), handlerGroups.Auth.Logout)
		auth.POST("/logout", handlerGroups.Auth.Logout)

		otp := auth.Group("/otp")
		{
			otp.POST("/verify", handlerGroups.Auth.VerifyEmail)
			otp.POST("/resend", handlerGroups.Auth.ResendOTP)
		}
	}

	private := api.Group("")
	private.Use(middlewareGroups.AuthMiddleware())
	{
		user := private.Group("/user")
		user.Use(middlewareGroups.TenantSchemaMiddleware())

		user.GET("profile/:id", handlerGroups.User.GetUserProfileByID)
		user.PATCH("profile/:id", handlerGroups.User.UpdateUserProfile)
		user.POST("change-password/:id", handlerGroups.Auth.UpdatePassword)
		user.POST("profile/:id/avatar", handlerGroups.User.AddAvatar)
		user.DELETE("profile/:id/avatar", handlerGroups.User.RemoveAvatar)
		user.GET("workspaces", handlerGroups.User.GetWorkspaces)
		user.GET("access-details", handlerGroups.User.GetUserAccessDetails)
		user.POST("assign", handlerGroups.Auth.AssignUserToWorkspace) // only admin can do

		tm := private.Group("") // Group of all api's that require tenant schema
		tm.Use(middlewareGroups.TenantSchemaMiddleware())
		{
			// -------- WORKSPACE SCOPED --------
			workspace := tm.Group("/workspace")

			// Admin-only
			adminWorkspace := workspace.Group("")
			adminWorkspace.Use(middlewareGroups.WorkspaceAndBaseAccessValidationMiddleware([]string{}))
			{
				adminWorkspace.POST("/create", handlerGroups.Workspace.CreateWorkspace)           // admin only
				adminWorkspace.GET("/", handlerGroups.Workspace.GetAllWorkspaces)                 // admin only
				adminWorkspace.GET("/:id/tables", handlerGroups.Workspace.GetTablesByWorkspaceId) // admin only
				adminWorkspace.PUT("/:id", handlerGroups.Workspace.UpdateWorkspace)               // admin only
				adminWorkspace.DELETE("/:id", handlerGroups.Workspace.DeleteWorkspace)            // admin only
			}

			// Full access
			fullAccessWorkspace := workspace.Group("")
			fullAccessWorkspace.Use(middlewareGroups.WorkspaceAndBaseAccessValidationMiddleware([]string{appConstant.AccessNames.FullAccess}))
			{
				// fullAccessWorkspace.POST("/:id/invite", handlerGroups.Auth.InviteUser)                   // fullAccess
				fullAccessWorkspace.POST("/:id/invite", handlerGroups.Auth.AddMultipleMembers)      // fullAccess
				fullAccessWorkspace.POST("/:id/remove", handlerGroups.Auth.RemoveUserFromWorkspace) // fullAccess
				fullAccessWorkspace.GET("/:id/members", handlerGroups.Auth.GetWorkspaceMembers)     // fullAccess
			}

			// All access
			allWorkspace := workspace.Group("")
			allWorkspace.Use(middlewareGroups.WorkspaceAndBaseAccessValidationMiddleware([]string{appConstant.AccessNames.LimitedAccess, appConstant.AccessNames.FullAccess}))
			{
				allWorkspace.GET("/:id/bases", handlerGroups.Workspace.GetBasesByWorkspaceId) // all
				allWorkspace.GET("/:id", handlerGroups.Workspace.GetWorkspaceByID)            // all
			}

			// -------- BASE SCOPED --------
			base := tm.Group("/base")
			// Full access
			fullAccessBase := base.Group("")
			fullAccessBase.Use(middlewareGroups.WorkspaceAndBaseAccessValidationMiddleware([]string{appConstant.AccessNames.FullAccess}))
			{
				fullAccessBase.POST("/create", handlerGroups.Base.CreateBase) // fullAccess
				fullAccessBase.PUT("/:id", handlerGroups.Base.UpdateBase)     // fullAccess
				fullAccessBase.DELETE("/:id", handlerGroups.Base.DeleteBase)  // fullAccess
			}
			// All access
			allBase := base.Group("")
			allBase.Use(middlewareGroups.WorkspaceAndBaseAccessValidationMiddleware([]string{appConstant.AccessNames.LimitedAccess, appConstant.AccessNames.FullAccess}))
			{
				allBase.GET("/:id", handlerGroups.Base.GetBaseByID)              // all
				allBase.GET("/:id/tables", handlerGroups.Base.GetTablesByBaseId) // all
				allBase.GET("/:id/members", handlerGroups.Auth.GetBaseMembers)   // all
			}

			// -------- TABLE SCOPED --------
			table := tm.Group("/table")
			table.Use(middlewareGroups.ScopeHeaderMiddleware("base"))
			table.Use(middlewareGroups.ScopeHeaderMiddleware("base")).Use(middlewareGroups.WorkspaceAndBaseAccessValidationMiddleware([]string{appConstant.AccessNames.LimitedAccess, appConstant.AccessNames.FullAccess}))

			// All access
			{
				table.POST("/create", handlerGroups.Table.CreateTable)           // all
				table.POST("/import", handlerGroups.Table.ImportTable)           // all
				table.PATCH("/:id", handlerGroups.Table.UpdateTable)             // all
				table.GET("/:id", handlerGroups.Table.GetTableByID)              // all
				table.GET("/", handlerGroups.Table.GetAllTables)                 // all
				table.GET("/:id/columns", handlerGroups.Table.GetColumnsByTable) // all
				table.GET("/:id/views", handlerGroups.Table.GetViewsByModelID)   // all
				table.GET("/:id/records", handlerGroups.Table.GetAllRecords)     // all
				table.DELETE("/:id", handlerGroups.Table.DeleteTable)            // all
			}

			// -------- COLUMN SCOPED --------
			column := tm.Group("/column")
			column.Use(middlewareGroups.ScopeHeaderMiddleware("base"))
			column.Use(middlewareGroups.ScopeHeaderMiddleware("base")).Use(middlewareGroups.WorkspaceAndBaseAccessValidationMiddleware([]string{appConstant.AccessNames.LimitedAccess, appConstant.AccessNames.FullAccess}))

			// All access
			{
				column.POST("/create", handlerGroups.Table.AddColumn)      // all
				column.GET("/:id", handlerGroups.Table.GetColumnById)      // all
				column.GET("/", handlerGroups.Table.GetAllColumns)         // all
				column.PATCH("/:id", handlerGroups.Table.UpdateColumn)     // all
				column.DELETE("/:id", handlerGroups.Table.DeleteColumn)    // all
				column.POST("/reorder", handlerGroups.Table.ReorderColumn) // all
			}

			// -------- ROW SCOPED --------
			row := tm.Group("/row")
			row.Use(middlewareGroups.ScopeHeaderMiddleware("base"))
			row.Use(middlewareGroups.ScopeHeaderMiddleware("base")).Use(middlewareGroups.WorkspaceAndBaseAccessValidationMiddleware([]string{appConstant.AccessNames.LimitedAccess, appConstant.AccessNames.FullAccess}))

			// All access
			{
				row.POST("/create", handlerGroups.Table.CreateRow)                    // all
				row.POST("/remove", handlerGroups.Table.DeleteRow)                    // all
				row.POST("/data/insert", handlerGroups.Table.InsertRowData)           // all
				row.POST("/data/relation", handlerGroups.Table.InsertRowDataForLinks) // all

				am := row.Group("")
				am.Use(middlewareGroups.FileSizeLimitMiddleware())            // all
				am.POST("/attachment/add", handlerGroups.Table.AddAttachment) // all

				row.POST("/attachment/remove", handlerGroups.Table.RemoveAttachments) // all
			}

			// -------- VIEW SCOPED --------
			view := tm.Group("/view")
			view.Use(middlewareGroups.ScopeHeaderMiddleware("base"))
			view.Use(middlewareGroups.ScopeHeaderMiddleware("base")).Use(middlewareGroups.WorkspaceAndBaseAccessValidationMiddleware([]string{appConstant.AccessNames.LimitedAccess, appConstant.AccessNames.FullAccess}))

			// All access
			{
				view.POST("/create", handlerGroups.Table.CreateView) // all
				view.GET("/:id", handlerGroups.Table.GetViewByID)    // all
				view.GET("/", handlerGroups.Table.GetAllViews)       // all
				view.PATCH("/:id", handlerGroups.Table.UpdateView)   // all
				view.DELETE("/:id", handlerGroups.Table.DeleteView)  // all
			}

			// -------- ASSET SCOPED --------
			asset := tm.Group("asset")
			asset.Use(middlewareGroups.ScopeHeaderMiddleware("base")).Use(middlewareGroups.WorkspaceAndBaseAccessValidationMiddleware([]string{appConstant.AccessNames.LimitedAccess, appConstant.AccessNames.FullAccess}))
			{
				am := asset.Group("")
				am.Use(middlewareGroups.FileSizeLimitMiddleware())
				am.POST("/upload", handlerGroups.Asset.Upload)
				am.POST("/upload-image", handlerGroups.Asset.UploadImage)

				asset.POST("/bulk", handlerGroups.Asset.GetBulkAssets)    // all
				asset.PATCH("/:id", handlerGroups.Asset.UpdateAssetByID)  // all
				asset.DELETE("/:id", handlerGroups.Asset.DeleteAssetByID) // all
			}
		}

		// only admin can do
		tenant := private.Group("/tenant")
		adminTenant := tenant.Group("")
		adminTenant.Use(middlewareGroups.TenantSchemaMiddleware())
		adminTenant.Use(middlewareGroups.WorkspaceAndBaseAccessValidationMiddleware([]string{}))
		{
			adminTenant.POST("user/create", handlerGroups.Auth.AddUser)            // only admin
			adminTenant.POST("user/remove", handlerGroups.Auth.RemoveUser)         // only admin
			adminTenant.POST("user/activate", handlerGroups.Auth.ActivateUser)     // only admin
			adminTenant.POST("user/deactivate", handlerGroups.Auth.DeactivateUser) // only admin
			adminTenant.GET("info", handlerGroups.Tenant.GetTenantInfo)            // only admin
			adminTenant.PATCH("info", handlerGroups.Tenant.UpdateTenantInfo)       // only admin
		}

		adminAndWorkspaceTenant := tenant.Group("")
		adminAndWorkspaceTenant.Use(middlewareGroups.TenantSchemaMiddleware())
		adminAndWorkspaceTenant.Use(middlewareGroups.WorkspaceAndBaseAccessValidationMiddleware([]string{appConstant.AccessNames.FullAccess}))
		adminAndWorkspaceTenant.GET("users", handlerGroups.Auth.GetUsers)
	}

	return r
}
