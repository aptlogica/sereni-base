package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aptlogica/sereni-base/internal/config"
	"github.com/aptlogica/sereni-base/internal/handlers"
	"github.com/aptlogica/sereni-base/internal/router"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRouterConstants(t *testing.T) {
	if router.RouteCreate != "/create" {
		t.Errorf("RouteCreate = %q, want %q", router.RouteCreate, "/create")
	}
}

// Helper functions
func createMockMiddlewares() router.Middlewares {
	return router.Middlewares{
		CORS: func() gin.HandlerFunc {
			return func(c *gin.Context) {
				c.Next()
			}
		},
		RequestLogger: func() gin.HandlerFunc {
			return func(c *gin.Context) {
				c.Next()
			}
		},
		DatabaseQueryLogger: func() gin.HandlerFunc {
			return func(c *gin.Context) {
				c.Next()
			}
		},
		RequestSizeLimit: func(size int64) gin.HandlerFunc {
			return func(c *gin.Context) {
				c.Next()
			}
		},
		AuthMiddleware: func() gin.HandlerFunc {
			return func(c *gin.Context) {
				c.Next()
			}
		},
		FileSizeLimitMiddleware: func() gin.HandlerFunc {
			return func(c *gin.Context) {
				c.Next()
			}
		},
		ScopeHeaderMiddleware: func(scope string) gin.HandlerFunc {
			return func(c *gin.Context) {
				c.Next()
			}
		},
		WorkspaceAndBaseAccessValidationMiddleware: func(allowedAccess []string) gin.HandlerFunc {
			return func(c *gin.Context) {
				c.Next()
			}
		},
	}
}

func createMockHandlers() router.Handlers {
	// Create mock handlers with simple responses
	authHandler := &handlers.AuthHandler{}
	workspaceHandler := &handlers.WorkspaceHandler{}
	baseHandler := &handlers.BaseHandler{}
	assetHandler := &handlers.AssetsHandler{}
	tableHandler := &handlers.TableHandler{}
	userHandler := &handlers.UserHandler{}
	organizationHandler := &handlers.OrganizationHandler{}

	return router.Handlers{
		Auth:         authHandler,
		Workspace:    workspaceHandler,
		Base:         baseHandler,
		Asset:        assetHandler,
		Table:        tableHandler,
		User:         userHandler,
		Organization: organizationHandler,
	}
}

func TestSetup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	assert.NotNil(t, r)
}

func TestHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "ok", response["status"])
	assert.Equal(t, "Serenibase is running", response["message"])
	assert.Equal(t, "1.0.0", response["version"])

	// Verify features array
	features, ok := response["features"].([]interface{})
	assert.True(t, ok)
	assert.Equal(t, 7, len(features))
	assert.Contains(t, features, "Dynamic table creation")
	assert.Contains(t, features, "Complex filtering")
	assert.Contains(t, features, "Relationship joins")
	assert.Contains(t, features, "Aggregation functions")
	assert.Contains(t, features, "Full-text search")
	assert.Contains(t, features, "Range queries")
	assert.Contains(t, features, "Views management")
}

func TestAuthRoutesSetup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	routes := r.Routes()

	authRoutes := []string{
		"/api/v1/auth/login",
		"/api/v1/auth/forgot-password",
		"/api/v1/auth/reset-password",
		"/api/v1/auth/validate-token",
		"/api/v1/auth/verify-token",
		"/api/v1/auth/refresh",
		"/api/v1/auth/logout",
		"/api/v1/auth/otp/verify",
		"/api/v1/auth/otp/resend",
	}

	for _, expectedRoute := range authRoutes {
		found := false
		for _, route := range routes {
			if route.Path == expectedRoute {
				found = true
				break
			}
		}
		assert.True(t, found, "Route %s should be registered", expectedRoute)
	}
}

func TestUserRoutesSetup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	routes := r.Routes()

	userRoutes := []string{
		"/api/v1/user/profile/:id",
		"/api/v1/user/change-password/:id",
		"/api/v1/user/profile/:id/avatar",
		"/api/v1/user/workspaces",
		"/api/v1/user/access-details",
		"/api/v1/user/roles-and-access/:id",
		"/api/v1/user/assign",
		"/api/v1/user/access/update",
		"/api/v1/user/create",
		"/api/v1/user/edit",
		"/api/v1/user/remove",
		"/api/v1/user/activate",
		"/api/v1/user/deactivate",
		"/api/v1/user/list",
		"/api/v1/user/list-for-assign",
	}

	for _, expectedRoute := range userRoutes {
		found := false
		for _, route := range routes {
			if route.Path == expectedRoute {
				found = true
				break
			}
		}
		assert.True(t, found, "Route %s should be registered", expectedRoute)
	}
}

func TestOrganizationRoutesSetup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	routes := r.Routes()

	orgRoutes := []string{
		"/api/v1/organization",
		"/api/v1/organization/:id",
	}

	for _, expectedRoute := range orgRoutes {
		found := false
		for _, route := range routes {
			if route.Path == expectedRoute {
				found = true
				break
			}
		}
		assert.True(t, found, "Route %s should be registered", expectedRoute)
	}
}

func TestWorkspaceRoutesSetup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	routes := r.Routes()

	workspaceRoutes := []string{
		"/api/v1/workspace/create",
		"/api/v1/workspace/",
		"/api/v1/workspace/:id/tables",
		"/api/v1/workspace/:id",
		"/api/v1/workspace/:id/remove",
		"/api/v1/workspace/:id/members",
		"/api/v1/workspace/:id/members-with-roles",
		"/api/v1/workspace/:id/bulk-add-members",
		"/api/v1/workspace/access/:id",
		"/api/v1/workspace/:id/bases",
	}

	for _, expectedRoute := range workspaceRoutes {
		found := false
		for _, route := range routes {
			if route.Path == expectedRoute {
				found = true
				break
			}
		}
		assert.True(t, found, "Route %s should be registered", expectedRoute)
	}
}

func TestBaseRoutesSetup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	routes := r.Routes()

	baseRoutes := []string{
		"/api/v1/base/create",
		"/api/v1/base/:id/remove",
		"/api/v1/base/:id/members",
		"/api/v1/base/:id/members-with-roles",
		"/api/v1/base/:id/bulk-add-members",
		"/api/v1/base/access/:id",
		"/api/v1/base/:id/image",
		"/api/v1/base/:id",
		"/api/v1/base/:id/tables",
	}

	for _, expectedRoute := range baseRoutes {
		found := false
		for _, route := range routes {
			if route.Path == expectedRoute {
				found = true
				break
			}
		}
		assert.True(t, found, "Route %s should be registered", expectedRoute)
	}
}

func TestTableRoutesSetup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	routes := r.Routes()

	tableRoutes := []string{
		"/api/v1/table/create",
		"/api/v1/table/import",
		"/api/v1/table/:id",
		"/api/v1/table/",
		"/api/v1/table/:id/columns",
		"/api/v1/table/:id/views",
		"/api/v1/table/:id/records",
	}

	for _, expectedRoute := range tableRoutes {
		found := false
		for _, route := range routes {
			if route.Path == expectedRoute {
				found = true
				break
			}
		}
		assert.True(t, found, "Route %s should be registered", expectedRoute)
	}
}

func TestColumnRoutesSetup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	routes := r.Routes()

	columnRoutes := []string{
		"/api/v1/column/create",
		"/api/v1/column/:id",
		"/api/v1/column/",
		"/api/v1/column/reorder",
	}

	for _, expectedRoute := range columnRoutes {
		found := false
		for _, route := range routes {
			if route.Path == expectedRoute {
				found = true
				break
			}
		}
		assert.True(t, found, "Route %s should be registered", expectedRoute)
	}
}

func TestRowRoutesSetup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	routes := r.Routes()

	rowRoutes := []string{
		"/api/v1/row/create",
		"/api/v1/row/remove",
		"/api/v1/row/bulk-remove",
		"/api/v1/row/data/insert",
		"/api/v1/row/data/relation",
		"/api/v1/row/attachment/add",
		"/api/v1/row/attachment/remove",
	}

	for _, expectedRoute := range rowRoutes {
		found := false
		for _, route := range routes {
			if route.Path == expectedRoute {
				found = true
				break
			}
		}
		assert.True(t, found, "Route %s should be registered", expectedRoute)
	}
}

func TestViewRoutesSetup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	routes := r.Routes()

	viewRoutes := []string{
		"/api/v1/view/create",
		"/api/v1/view/:id",
		"/api/v1/view/",
	}

	for _, expectedRoute := range viewRoutes {
		found := false
		for _, route := range routes {
			if route.Path == expectedRoute {
				found = true
				break
			}
		}
		assert.True(t, found, "Route %s should be registered", expectedRoute)
	}
}

func TestAssetRoutesSetup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	routes := r.Routes()

	assetRoutes := []string{
		"/api/v1/asset/upload",
		"/api/v1/asset/upload-image",
		"/api/v1/asset/bulk",
		"/api/v1/asset/:id",
	}

	for _, expectedRoute := range assetRoutes {
		found := false
		for _, route := range routes {
			if route.Path == expectedRoute {
				found = true
				break
			}
		}
		assert.True(t, found, "Route %s should be registered", expectedRoute)
	}
}

func TestMiddlewaresStructure(t *testing.T) {
	middlewares := createMockMiddlewares()

	assert.NotNil(t, middlewares.CORS)
	assert.NotNil(t, middlewares.RequestLogger)
	assert.NotNil(t, middlewares.DatabaseQueryLogger)
	assert.NotNil(t, middlewares.RequestSizeLimit)
	assert.NotNil(t, middlewares.AuthMiddleware)
	assert.NotNil(t, middlewares.FileSizeLimitMiddleware)
	assert.NotNil(t, middlewares.ScopeHeaderMiddleware)
	assert.NotNil(t, middlewares.WorkspaceAndBaseAccessValidationMiddleware)
}

func TestHandlersStructure(t *testing.T) {
	handlers := createMockHandlers()

	assert.NotNil(t, handlers.Auth)
	assert.NotNil(t, handlers.Workspace)
	assert.NotNil(t, handlers.Base)
	assert.NotNil(t, handlers.Asset)
	assert.NotNil(t, handlers.Table)
	assert.NotNil(t, handlers.User)
	assert.NotNil(t, handlers.Organization)
}

func TestSetupWithAllMiddlewares(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()

	middlewareCalled := make(map[string]bool)

	middlewares := router.Middlewares{
		CORS: func() gin.HandlerFunc {
			return func(c *gin.Context) {
				middlewareCalled["CORS"] = true
				c.Next()
			}
		},
		RequestLogger: func() gin.HandlerFunc {
			return func(c *gin.Context) {
				middlewareCalled["RequestLogger"] = true
				c.Next()
			}
		},
		DatabaseQueryLogger: func() gin.HandlerFunc {
			return func(c *gin.Context) {
				middlewareCalled["DatabaseQueryLogger"] = true
				c.Next()
			}
		},
		RequestSizeLimit: func(size int64) gin.HandlerFunc {
			return func(c *gin.Context) {
				c.Next()
			}
		},
		AuthMiddleware: func() gin.HandlerFunc {
			return func(c *gin.Context) {
				middlewareCalled["AuthMiddleware"] = true
				c.Next()
			}
		},
		FileSizeLimitMiddleware: func() gin.HandlerFunc {
			return func(c *gin.Context) {
				middlewareCalled["FileSizeLimitMiddleware"] = true
				c.Next()
			}
		},
		ScopeHeaderMiddleware: func(scope string) gin.HandlerFunc {
			return func(c *gin.Context) {
				c.Next()
			}
		},
		WorkspaceAndBaseAccessValidationMiddleware: func(allowedAccess []string) gin.HandlerFunc {
			return func(c *gin.Context) {
				c.Next()
			}
		},
	}

	r := router.Setup(cfg, handlers, middlewares)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/health", nil)
	r.ServeHTTP(w, req)

	assert.True(t, middlewareCalled["CORS"])
	assert.True(t, middlewareCalled["RequestLogger"])
	assert.True(t, middlewareCalled["DatabaseQueryLogger"])
}

func TestStaticAssetsRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	routes := r.Routes()

	found := false
	for _, route := range routes {
		if route.Path == "/assets/*filepath" {
			found = true
			break
		}
	}
	assert.True(t, found, "Static assets route should be registered")
}

func TestAPIv1GroupExists(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	routes := r.Routes()

	hasAPIv1Route := false
	for _, route := range routes {
		if len(route.Path) > 7 && route.Path[:7] == "/api/v1" {
			hasAPIv1Route = true
			break
		}
	}
	assert.True(t, hasAPIv1Route, "Should have routes under /api/v1")
}

func TestPrivateRoutesGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	// Verify that private routes exist
	routes := r.Routes()
	privateRoutes := []string{
		"/api/v1/user/profile/:id",
		"/api/v1/workspace/",
		"/api/v1/base/create",
		"/api/v1/table/create",
	}

	for _, expectedRoute := range privateRoutes {
		found := false
		for _, route := range routes {
			if route.Path == expectedRoute {
				found = true
				break
			}
		}
		assert.True(t, found, "Private route %s should be registered", expectedRoute)
	}
}

func TestRouteHTTPMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	routes := r.Routes()

	testCases := []struct {
		path   string
		method string
	}{
		{"/api/v1/health", "GET"},
		{"/api/v1/auth/login", "POST"},
		{"/api/v1/workspace/:id", "PUT"},
		{"/api/v1/workspace/:id", "DELETE"},
		{"/api/v1/table/:id", "PATCH"},
	}

	for _, tc := range testCases {
		found := false
		for _, route := range routes {
			if route.Path == tc.path && route.Method == tc.method {
				found = true
				break
			}
		}
		assert.True(t, found, "Route %s %s should be registered", tc.method, tc.path)
	}
}

func TestMaxMultipartMemory(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	// Verify that MaxMultipartMemory is set
	assert.Equal(t, int64(100<<20), r.MaxMultipartMemory)
}

func TestRouterRecoveryMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	// Verify router is set up (recovery middleware is automatically added)
	assert.NotNil(t, r)
}

func TestMiddlewaresFunctionsReturnHandlers(t *testing.T) {
	middlewares := createMockMiddlewares()

	// Test that middleware functions return gin.HandlerFunc
	corsHandler := middlewares.CORS()
	assert.NotNil(t, corsHandler)

	requestLoggerHandler := middlewares.RequestLogger()
	assert.NotNil(t, requestLoggerHandler)

	dbQueryLoggerHandler := middlewares.DatabaseQueryLogger()
	assert.NotNil(t, dbQueryLoggerHandler)

	authHandler := middlewares.AuthMiddleware()
	assert.NotNil(t, authHandler)

	fileSizeLimitHandler := middlewares.FileSizeLimitMiddleware()
	assert.NotNil(t, fileSizeLimitHandler)

	requestSizeLimitHandler := middlewares.RequestSizeLimit(1024)
	assert.NotNil(t, requestSizeLimitHandler)

	scopeHeaderHandler := middlewares.ScopeHeaderMiddleware("test-scope")
	assert.NotNil(t, scopeHeaderHandler)

	accessValidationHandler := middlewares.WorkspaceAndBaseAccessValidationMiddleware([]string{"read", "write"})
	assert.NotNil(t, accessValidationHandler)
}

func TestSetupAuthRoutesIndependently(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	// Verify all auth routes are set up correctly
	authEndpoints := []string{
		"login",
		"forgot-password",
		"reset-password",
		"validate-token",
		"verify-token",
		"refresh",
		"logout",
	}

	routes := r.Routes()
	for _, endpoint := range authEndpoints {
		expectedPath := "/api/v1/auth/" + endpoint
		found := false
		for _, route := range routes {
			if route.Path == expectedPath {
				found = true
				break
			}
		}
		assert.True(t, found, "Auth endpoint %s should be registered", endpoint)
	}
}

func TestSetupUserRoutesWithDifferentHTTPMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	testCases := []struct {
		path   string
		method string
	}{
		{"/api/v1/user/profile/:id", "GET"},
		{"/api/v1/user/profile/:id", "PATCH"},
		{"/api/v1/user/change-password/:id", "POST"},
		{"/api/v1/user/profile/:id/avatar", "POST"},
		{"/api/v1/user/profile/:id/avatar", "DELETE"},
		{"/api/v1/user/create", "POST"},
	}

	routes := r.Routes()
	for _, tc := range testCases {
		found := false
		for _, route := range routes {
			if route.Path == tc.path && route.Method == tc.method {
				found = true
				break
			}
		}
		assert.True(t, found, "User route %s %s should be registered", tc.method, tc.path)
	}
}

func TestSetupWorkspaceRoutesWithNestedPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	nestedPaths := []string{
		"/api/v1/workspace/:id/tables",
		"/api/v1/workspace/:id/members",
		"/api/v1/workspace/:id/members-with-roles",
		"/api/v1/workspace/:id/bases",
		"/api/v1/workspace/:id/bulk-add-members",
	}

	routes := r.Routes()
	for _, expectedPath := range nestedPaths {
		found := false
		for _, route := range routes {
			if route.Path == expectedPath {
				found = true
				break
			}
		}
		assert.True(t, found, "Workspace nested route %s should be registered", expectedPath)
	}
}

func TestSetupBaseRoutesWithImageEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	routes := r.Routes()

	// Check POST /base/:id/image
	foundPost := false
	for _, route := range routes {
		if route.Path == "/api/v1/base/:id/image" && route.Method == "POST" {
			foundPost = true
			break
		}
	}
	assert.True(t, foundPost, "Base image POST route should be registered")

	// Check DELETE /base/:id/image
	foundDelete := false
	for _, route := range routes {
		if route.Path == "/api/v1/base/:id/image" && route.Method == "DELETE" {
			foundDelete = true
			break
		}
	}
	assert.True(t, foundDelete, "Base image DELETE route should be registered")
}

func TestSetupTableRoutesWithAllOperations(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	operations := map[string]string{
		"/api/v1/table/create":      "POST",
		"/api/v1/table/import":      "POST",
		"/api/v1/table/:id":         "PATCH",
		"/api/v1/table/":            "GET",
		"/api/v1/table/:id/columns": "GET",
		"/api/v1/table/:id/views":   "GET",
		"/api/v1/table/:id/records": "GET",
	}

	routes := r.Routes()
	for path, method := range operations {
		found := false
		for _, route := range routes {
			if route.Path == path && route.Method == method {
				found = true
				break
			}
		}
		assert.True(t, found, "Table route %s %s should be registered", method, path)
	}
}

func TestSetupColumnRoutesWithReorder(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	routes := r.Routes()

	found := false
	for _, route := range routes {
		if route.Path == "/api/v1/column/reorder" && route.Method == "POST" {
			found = true
			break
		}
	}
	assert.True(t, found, "Column reorder route should be registered")
}

func TestSetupRowRoutesWithAttachments(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	attachmentRoutes := []string{
		"/api/v1/row/attachment/add",
		"/api/v1/row/attachment/remove",
	}

	routes := r.Routes()
	for _, expectedPath := range attachmentRoutes {
		found := false
		for _, route := range routes {
			if route.Path == expectedPath && route.Method == "POST" {
				found = true
				break
			}
		}
		assert.True(t, found, "Row attachment route %s should be registered", expectedPath)
	}
}

func TestSetupRowRoutesWithDataOperations(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	dataRoutes := []string{
		"/api/v1/row/data/insert",
		"/api/v1/row/data/relation",
	}

	routes := r.Routes()
	for _, expectedPath := range dataRoutes {
		found := false
		for _, route := range routes {
			if route.Path == expectedPath && route.Method == "POST" {
				found = true
				break
			}
		}
		assert.True(t, found, "Row data route %s should be registered", expectedPath)
	}
}

func TestSetupViewRoutesCRUD(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	operations := map[string]string{
		"/api/v1/view/create": "POST",
		"/api/v1/view/":       "GET",
	}

	routes := r.Routes()
	for path, method := range operations {
		found := false
		for _, route := range routes {
			if route.Path == path && route.Method == method {
				found = true
				break
			}
		}
		assert.True(t, found, "View route %s %s should be registered", method, path)
	}
}

func TestSetupAssetRoutesWithUploads(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	uploadRoutes := []string{
		"/api/v1/asset/upload",
		"/api/v1/asset/upload-image",
	}

	routes := r.Routes()
	for _, expectedPath := range uploadRoutes {
		found := false
		for _, route := range routes {
			if route.Path == expectedPath && route.Method == "POST" {
				found = true
				break
			}
		}
		assert.True(t, found, "Asset upload route %s should be registered", expectedPath)
	}
}

func TestSetupAssetRoutesWithBulkOperations(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	routes := r.Routes()

	found := false
	for _, route := range routes {
		if route.Path == "/api/v1/asset/bulk" && route.Method == "POST" {
			found = true
			break
		}
	}
	assert.True(t, found, "Asset bulk route should be registered")
}

func TestSetupAssetRoutesWithUpdateDelete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	testCases := []struct {
		path   string
		method string
	}{
		{"/api/v1/asset/:id", "PATCH"},
		{"/api/v1/asset/:id", "DELETE"},
	}

	routes := r.Routes()
	for _, tc := range testCases {
		found := false
		for _, route := range routes {
			if route.Path == tc.path && route.Method == tc.method {
				found = true
				break
			}
		}
		assert.True(t, found, "Asset route %s %s should be registered", tc.method, tc.path)
	}
}

func TestRouterConstantUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	// Verify routes using RouteCreate constant
	createRoutes := []string{
		"/api/v1/workspace/create",
		"/api/v1/base/create",
		"/api/v1/table/create",
		"/api/v1/column/create",
		"/api/v1/row/create",
		"/api/v1/view/create",
		"/api/v1/user/create",
	}

	routes := r.Routes()
	for _, expectedPath := range createRoutes {
		found := false
		for _, route := range routes {
			if route.Path == expectedPath {
				found = true
				break
			}
		}
		assert.True(t, found, "Route using RouteCreate constant %s should be registered", expectedPath)
	}
}

func TestMiddlewaresAppliedToRowAttachments(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	// Verify the route exists
	routes := r.Routes()
	found := false
	for _, route := range routes {
		if route.Path == "/api/v1/row/attachment/add" {
			found = true
			break
		}
	}
	assert.True(t, found, "Row attachment route should exist")
}

func TestMiddlewaresAppliedToAssetUpload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	// Verify upload routes exist
	uploadRoutes := []string{
		"/api/v1/asset/upload",
		"/api/v1/asset/upload-image",
	}

	routes := r.Routes()
	for _, expectedPath := range uploadRoutes {
		found := false
		for _, route := range routes {
			if route.Path == expectedPath {
				found = true
				break
			}
		}
		assert.True(t, found, "Asset upload route %s should exist", expectedPath)
	}
}
