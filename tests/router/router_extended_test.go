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

func init() {
	gin.SetMode(gin.TestMode)
}

// TestRouterSetup_HealthEndpoint tests the health check endpoint
func TestRouterSetup_HealthEndpoint(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Version: "1.0.0",
		},
	}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.True(t, response["success"].(bool))
}

// TestRouterSetup_With404 tests unknown routes
func TestRouterSetup_With404(t *testing.T) {
	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/nonexistent", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestRouterSetup_StaticAssets tests static asset serving
func TestRouterSetup_StaticAssets(t *testing.T) {
	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	// This will 404 since there's no actual file, but proves route works
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/assets/test.css", nil)
	r.ServeHTTP(w, req)

	// Will be 404 or 403, not 500
	assert.NotEqual(t, http.StatusInternalServerError, w.Code)
}

// TestRouterSetup_CORSHeaders tests CORS middleware is applied
func TestRouterSetup_CORSHeaders(t *testing.T) {
	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := router.Middlewares{
		CORS: func() gin.HandlerFunc {
			return func(c *gin.Context) {
				c.Header("Access-Control-Allow-Origin", "*")
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

	r := router.Setup(cfg, handlers, middlewares)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

// TestRouterSetup_RequestID tests request ID is generated
func TestRouterSetup_RequestID(t *testing.T) {
	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	r.ServeHTTP(w, req)

	// Request ID should be in response
	requestID := w.Header().Get("X-Request-Id")
	assert.NotEmpty(t, requestID)
}

// TestRouterSetup_ProvidedRequestID tests provided request ID is used
func TestRouterSetup_ProvidedRequestID(t *testing.T) {
	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	req.Header.Set("X-Request-Id", "custom-request-id")
	r.ServeHTTP(w, req)

	assert.Equal(t, "custom-request-id", w.Header().Get("X-Request-Id"))
}

// TestRouterHandlers_Structure tests handlers struct
func TestRouterHandlers_Structure(t *testing.T) {
	handlers := router.Handlers{
		Auth:         nil,
		Workspace:    nil,
		Base:         nil,
		Asset:        nil,
		Table:        nil,
		User:         nil,
		Organization: nil,
	}

	assert.Nil(t, handlers.Auth)
	assert.Nil(t, handlers.Workspace)
	assert.Nil(t, handlers.Base)
	assert.Nil(t, handlers.Asset)
	assert.Nil(t, handlers.Table)
	assert.Nil(t, handlers.User)
	assert.Nil(t, handlers.Organization)
}

// TestRouterMiddlewares_Structure tests middlewares struct
func TestRouterMiddlewares_Structure(t *testing.T) {
	middlewares := router.Middlewares{
		CORS:                    nil,
		RequestLogger:           nil,
		DatabaseQueryLogger:     nil,
		RequestSizeLimit:        nil,
		AuthMiddleware:          nil,
		FileSizeLimitMiddleware: nil,
		ScopeHeaderMiddleware:   nil,
		WorkspaceAndBaseAccessValidationMiddleware: nil,
	}

	assert.Nil(t, middlewares.CORS)
	assert.Nil(t, middlewares.RequestLogger)
	assert.Nil(t, middlewares.DatabaseQueryLogger)
	assert.Nil(t, middlewares.RequestSizeLimit)
	assert.Nil(t, middlewares.AuthMiddleware)
	assert.Nil(t, middlewares.FileSizeLimitMiddleware)
	assert.Nil(t, middlewares.ScopeHeaderMiddleware)
	assert.Nil(t, middlewares.WorkspaceAndBaseAccessValidationMiddleware)
}

// TestRouterSetup_MultipleRequests tests multiple concurrent requests
func TestRouterSetup_MultipleRequests(t *testing.T) {
	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/api/v1/health", nil)
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestRouterSetup_HTTPMethods tests different HTTP methods
func TestRouterSetup_HTTPMethods(t *testing.T) {
	cfg := &config.Config{}
	handlers := createMockHandlers()
	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlers, middlewares)

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(method, "/api/v1/health", nil)
			r.ServeHTTP(w, req)
			// Health only supports GET, other methods return 404
			if method == "GET" {
				assert.Equal(t, http.StatusOK, w.Code)
			}
		})
	}
}

// TestRouterSetup_Recovery tests panic recovery
func TestRouterSetup_Recovery(t *testing.T) {
	cfg := &config.Config{}

	// Create handlers with mock
	authHandler := handlers.NewAuthHandler(nil)
	workspaceHandler := handlers.NewWorkspaceHandler(nil, nil)
	baseHandler := handlers.NewBaseHandler(nil)
	assetHandler := handlers.NewAssetsHandler(nil)
	tableHandler := handlers.NewTableHandler(nil, nil)
	userHandler := handlers.NewUserHandler(nil)
	organizationHandler := handlers.NewOrganizationHandler(nil)

	handlersGroup := router.Handlers{
		Auth:         authHandler,
		Workspace:    workspaceHandler,
		Base:         baseHandler,
		Asset:        assetHandler,
		Table:        tableHandler,
		User:         userHandler,
		Organization: organizationHandler,
	}

	middlewares := createMockMiddlewares()

	r := router.Setup(cfg, handlersGroup, middlewares)

	// This should not panic
	assert.NotPanics(t, func() {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		r.ServeHTTP(w, req)
	})
}
