package providers_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appErrors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/config"
	"github.com/aptlogica/sereni-base/internal/providers/auth"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const jwtServiceURL = "http://localhost:8081"

func checkJWTServiceAvailable() bool {
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get(jwtServiceURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func newAuthCfg() *config.AuthConfig {
	return &config.AuthConfig{
		URL: jwtServiceURL,
		JWT: config.JWTConfig{
			Secret:             "test-secret",
			AccessTokenExpiry:  3600,
			RefreshTokenExpiry: 86400,
			Issuer:             "test-issuer",
		},
	}
}

func newTestAuthProvider(t *testing.T, handlers map[string]http.HandlerFunc) (auth.AuthProvider, func()) {
	t.Helper()

	mux := http.NewServeMux()
	for path, handler := range handlers {
		mux.HandleFunc(path, handler)
	}

	server := httptest.NewServer(mux)
	cfg := newAuthCfg()
	cfg.URL = server.URL

	provider, err := auth.NewAuthProvider(cfg)
	require.NoError(t, err)

	return provider, server.Close
}

func TestNewAuthProvider(t *testing.T) {
	provider, err := auth.NewAuthProvider(newAuthCfg())
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestAuthProvider_Login_Refresh_Validate(t *testing.T) {
	if !checkJWTServiceAvailable() {
		t.Skip("Skipping test: JWT service not available at " + jwtServiceURL)
	}

	provider, err := auth.NewAuthProvider(newAuthCfg())
	require.NoError(t, err)
	ctx := context.Background()

	t.Run("login request shape", func(t *testing.T) {
		tokens, err := provider.Login(ctx, auth.AuthServiceLoginRequest{
			Id:            "test-user-id",
			Email:         "user@example.com",
			EmailVerified: true,
			Roles:         []string{"user"},
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
	})

	t.Run("refresh request shape", func(t *testing.T) {
		_, err := provider.RefreshToken(ctx, auth.AuthServiceRefreshRequest{RefreshToken: "invalid.refresh.token"})
		assert.Error(t, err)
	})

	t.Run("validate malformed token", func(t *testing.T) {
		_, err := provider.ValidateToken(ctx, "malformed.token")
		assert.Error(t, err)
	})
}

func TestAuthProvider_ValidateToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		provider, closeServer := newTestAuthProvider(t, map[string]http.HandlerFunc{
			"/auth/validate-token": func(w http.ResponseWriter, r *http.Request) {
				var payload map[string]string
				body, _ := io.ReadAll(r.Body)
				_ = json.Unmarshal(body, &payload)
				assert.Equal(t, "valid", payload["token"])

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"success":true,"code":"OK","message":"ok","data":{"sub":"user-1","email":"user@example.com","roles":"admin","token_type":"access","iss":"iss","exp":1,"iat":1,"nbf":1}}`))
			},
		})
		defer closeServer()

		claims, err := provider.ValidateToken(context.Background(), "Bearer valid")

		assert.NoError(t, err)
		assert.Equal(t, "user-1", claims.UserId)
		assert.Equal(t, "admin", claims.Roles)
	})

	t.Run("token expired code returns invalid", func(t *testing.T) {
		provider, closeServer := newTestAuthProvider(t, map[string]http.HandlerFunc{
			"/auth/validate-token": func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"success":false,"code":"TOKEN_EXPIRED","message":"expired","data":{}}`))
			},
		})
		defer closeServer()

		_, err := provider.ValidateToken(context.Background(), "expired")

		assert.ErrorIs(t, err, appErrors.TokenInvalid)
	})

	t.Run("other error code returns expired", func(t *testing.T) {
		provider, closeServer := newTestAuthProvider(t, map[string]http.HandlerFunc{
			"/auth/validate-token": func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"success":false,"code":"OTHER","message":"bad","data":{}}`))
			},
		})
		defer closeServer()

		_, err := provider.ValidateToken(context.Background(), "bad")

		assert.ErrorIs(t, err, appErrors.TokenExpired)
	})

	t.Run("decode error", func(t *testing.T) {
		provider, closeServer := newTestAuthProvider(t, map[string]http.HandlerFunc{
			"/auth/validate-token": func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`not-json`))
			},
		})
		defer closeServer()

		_, err := provider.ValidateToken(context.Background(), "bad-json")

		assert.Error(t, err)
	})
}

func TestAuthProvider_RefreshToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		provider, closeServer := newTestAuthProvider(t, map[string]http.HandlerFunc{
			"/auth/refresh": func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"success":true,"code":"OK","message":"ok","data":{"access_token":"a","refresh_token":"r","token_type":"bearer","expires_in":3600}}`))
			},
		})
		defer closeServer()

		tokens, err := provider.RefreshToken(context.Background(), auth.AuthServiceRefreshRequest{RefreshToken: "r"})

		assert.NoError(t, err)
		assert.Equal(t, "a", tokens.AccessToken)
		assert.Equal(t, "r", tokens.RefreshToken)
	})

	t.Run("non-200 status", func(t *testing.T) {
		provider, closeServer := newTestAuthProvider(t, map[string]http.HandlerFunc{
			"/auth/refresh": func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
		})
		defer closeServer()

		_, err := provider.RefreshToken(context.Background(), auth.AuthServiceRefreshRequest{RefreshToken: "r"})

		assert.ErrorIs(t, err, appErrors.TokenInvalid)
	})

	t.Run("success false", func(t *testing.T) {
		provider, closeServer := newTestAuthProvider(t, map[string]http.HandlerFunc{
			"/auth/refresh": func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"success":false,"code":"ERR","message":"bad","data":{}}`))
			},
		})
		defer closeServer()

		_, err := provider.RefreshToken(context.Background(), auth.AuthServiceRefreshRequest{RefreshToken: "r"})

		assert.ErrorIs(t, err, appErrors.TokenInvalid)
	})

	t.Run("decode error", func(t *testing.T) {
		provider, closeServer := newTestAuthProvider(t, map[string]http.HandlerFunc{
			"/auth/refresh": func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`not-json`))
			},
		})
		defer closeServer()

		_, err := provider.RefreshToken(context.Background(), auth.AuthServiceRefreshRequest{RefreshToken: "r"})

		assert.Error(t, err)
	})
}

func TestAuthProvider_Login(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		provider, closeServer := newTestAuthProvider(t, map[string]http.HandlerFunc{
			"/auth/login": func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"success":true,"code":"OK","message":"ok","data":{"access_token":"a","refresh_token":"r","token_type":"bearer","expires_in":3600}}`))
			},
		})
		defer closeServer()

		tokens, err := provider.Login(context.Background(), auth.AuthServiceLoginRequest{Id: "id", Email: "e"})

		assert.NoError(t, err)
		assert.Equal(t, "a", tokens.AccessToken)
		assert.Equal(t, "r", tokens.RefreshToken)
	})

	t.Run("success false", func(t *testing.T) {
		provider, closeServer := newTestAuthProvider(t, map[string]http.HandlerFunc{
			"/auth/login": func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"success":false,"code":"ERR","message":"bad","data":{}}`))
			},
		})
		defer closeServer()

		_, err := provider.Login(context.Background(), auth.AuthServiceLoginRequest{Id: "id", Email: "e"})

		assert.Error(t, err)
	})

	t.Run("decode error", func(t *testing.T) {
		provider, closeServer := newTestAuthProvider(t, map[string]http.HandlerFunc{
			"/auth/login": func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`not-json`))
			},
		})
		defer closeServer()

		_, err := provider.Login(context.Background(), auth.AuthServiceLoginRequest{Id: "id", Email: "e"})

		assert.Error(t, err)
	})
}
