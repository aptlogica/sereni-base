package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aptlogica/sereni-base/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// corsRouter serves the public API path and an app path behind the CORS middleware, with the app's
// origins limited to the base-ui
func corsRouter(handled *bool) *gin.Engine {
	setupCORSConfig("http://localhost:5050", "GET,POST,PUT,DELETE,OPTIONS,PATCH", "Content-Type,Authorization", true)
	_, r := gin.CreateTestContext(httptest.NewRecorder())
	r.Use(middleware.CORS())
	handler := func(c *gin.Context) {
		*handled = true
		c.Status(http.StatusOK)
	}
	r.GET("/api/v1/ws_abcdefghij/bases", handler)
	r.GET("/api/v1/workspace/w1", handler)
	return r
}

func TestCORS_PublicAPIAllowsAnyOrigin(t *testing.T) {
	handled := false
	r := corsRouter(&handled)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ws_abcdefghij/bases", nil)
	req.Header.Set("Origin", "https://my-app.example")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, handled)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "X-Request-Id, Retry-After, Idempotent-Replayed", w.Header().Get("Access-Control-Expose-Headers"))
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Credentials"), "no credentials with a wildcard origin")
}

func TestCORS_PublicAPIPreflight(t *testing.T) {
	handled := false
	r := corsRouter(&handled)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/ws_abcdefghij/bases", nil)
	req.Header.Set("Origin", "https://my-app.example")
	req.Header.Set("Access-Control-Request-Method", "PATCH")
	req.Header.Set("Access-Control-Request-Headers", "content-type,idempotency-key,x-client-version")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.False(t, handled, "a preflight is answered by the middleware")
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "PATCH")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "QUERY", "records/query also takes QUERY")
	assert.Equal(t, "content-type,idempotency-key,x-client-version", w.Header().Get("Access-Control-Allow-Headers"),
		"the headers the client asked for are allowed")
	assert.Equal(t, "600", w.Header().Get("Access-Control-Max-Age"))
}

func TestCORS_AppRoutesKeepTheirOriginList(t *testing.T) {
	handled := false
	r := corsRouter(&handled)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/w1", nil)
	req.Header.Set("Origin", "https://my-app.example")
	r.ServeHTTP(w, req)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"), "other origins can't call the app's API")

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/workspace/w1", nil)
	req.Header.Set("Origin", "http://localhost:5050")
	r.ServeHTTP(w, req)
	assert.Equal(t, "http://localhost:5050", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
}
