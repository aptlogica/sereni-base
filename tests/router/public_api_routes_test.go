package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/aptlogica/sereni-base/internal/config"
	"github.com/aptlogica/sereni-base/internal/middleware"
	"github.com/aptlogica/sereni-base/internal/platform"
	"github.com/aptlogica/sereni-base/internal/router"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func publicAPIRouter(t *testing.T, openAccess bool) (*gin.Engine, sqlmock.Sqlmock) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	cfg := &config.Config{}
	cfg.PublicAPI.OpenAccess = openAccess
	handlers := createMockHandlers()
	handlers.PublicAPI = platform.NewAPI(db, "public", nil, 600)
	return router.Setup(cfg, handlers, createMockMiddlewares()), mock
}

func serve(r *gin.Engine, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

func TestPublicAPIRoutesSitNextToExistingRoutes(t *testing.T) {
	r, mock := publicAPIRouter(t, true)
	assert.True(t, r.UseRawPath, "URL-encoded names must reach the handlers")

	// Existing /api/v1 routes still win
	w := serve(r, "/api/v1/health")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Serenibase is running")

	// Any other first segment is a workspace public ID, looked up by the resolver
	mock.ExpectQuery(`FROM "public".workspace_api`).WithArgs("ws_abcdefghij").
		WillReturnRows(sqlmock.NewRows([]string{"workspace_id", "schema_version"}))
	w = serve(r, "/api/v1/ws_abcdefghij/bases")
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"code":"not_found"`)
	assert.NoError(t, mock.ExpectationsWereMet(), "the request reached the workspace resolver")

	// A first segment that is not a public ID is a 404 without a database lookup
	w = serve(r, "/api/v1/auth/bases")
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Unknown paths under a workspace get the public API's envelope; other unknown paths keep gin's 404
	w = serve(r, "/api/v1/ws_abcdefghij/nothing")
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"error":{"code":"not_found"`)
	w = serve(r, "/nothing")
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NotContains(t, w.Body.String(), `"error"`)
}

func TestPublicAPIPreflightFromAnyOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	if config.AppConfig == nil {
		config.AppConfig = &config.Config{}
	}
	cfg := &config.Config{}
	cfg.PublicAPI.OpenAccess = true
	handlers := createMockHandlers()
	handlers.PublicAPI = platform.NewAPI(db, "public", nil, 600)
	middlewares := createMockMiddlewares()
	middlewares.CORS = middleware.CORS
	r := router.Setup(cfg, handlers, middlewares)

	// There is no OPTIONS route: the preflight is answered by the CORS middleware before the workspace lookup
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/ws_abcdefghij/bases/Main/tables/Orders/records", nil)
	req.Header.Set("Origin", "https://my-app.example")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "content-type")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.NoError(t, mock.ExpectationsWereMet(), "no database lookup for a preflight")
}

func TestPublicAPIRoutesOnlyWithOpenAccess(t *testing.T) {
	r, mock := publicAPIRouter(t, false)
	for _, route := range r.Routes() {
		assert.NotContains(t, route.Path, ":ws", "no public API route without PUBLIC_API_OPEN_ACCESS")
	}
	w := serve(r, "/api/v1/ws_abcdefghij/bases")
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet(), "no workspace lookup")
}
