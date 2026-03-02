package providers_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"serenibase/internal/config"
	"serenibase/internal/providers/auth"

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
