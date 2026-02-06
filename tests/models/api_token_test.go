package models_test

import (
	"testing"
	"time"

	"serenibase/internal/models/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAPIToken_TableName(t *testing.T) {
	token := tenant.APIToken{}
	schema := "test_schema"

	tableName := token.TableName(schema)

	assert.Equal(t, `"test_schema".api_tokens`, tableName)
}

func TestAPIToken_Fields(t *testing.T) {
	tokenID := uuid.New()
	userID := uuid.New().String()
	workspaceID := uuid.New().String()
	now := time.Now().UTC()
	expiresAt := now.Add(30 * 24 * time.Hour)

	token := tenant.APIToken{
		ID:          tokenID,
		UserID:      userID,
		WorkspaceID: &workspaceID,
		Name:        "Production API Token",
		TokenHash:   "hashed_token_value",
		Prefix:      "tk_prod",
		Status:      "active",
		ExpiresAt:   &expiresAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, tokenID, token.ID)
	assert.Equal(t, userID, token.UserID)
	assert.Equal(t, workspaceID, *token.WorkspaceID)
	assert.Equal(t, "Production API Token", token.Name)
	assert.Equal(t, "hashed_token_value", token.TokenHash)
	assert.Equal(t, "tk_prod", token.Prefix)
	assert.Equal(t, "active", token.Status)
	assert.NotNil(t, token.ExpiresAt)
}

func TestAPIToken_WithPermissions(t *testing.T) {
	permissions := "read,write"
	scopes := "workspace:123,base:456"

	token := tenant.APIToken{
		ID:          uuid.New(),
		UserID:      uuid.New().String(),
		Name:        "Limited Token",
		TokenHash:   "hash",
		Prefix:      "tk_lim",
		Permissions: &permissions,
		Scopes:      &scopes,
		Status:      "active",
	}

	assert.Equal(t, "read,write", *token.Permissions)
	assert.Equal(t, "workspace:123,base:456", *token.Scopes)
}

func TestAPIToken_WithRateLimit(t *testing.T) {
	rateLimit := 1000

	token := tenant.APIToken{
		ID:               uuid.New(),
		UserID:           uuid.New().String(),
		Name:             "Rate Limited Token",
		TokenHash:        "hash",
		Prefix:           "tk_rl",
		RateLimitPerHour: &rateLimit,
		Status:           "active",
	}

	assert.Equal(t, 1000, *token.RateLimitPerHour)
}

func TestAPIToken_UsageTracking(t *testing.T) {
	lastUsed := time.Now().UTC().Add(-1 * time.Hour)

	token := tenant.APIToken{
		ID:         uuid.New(),
		UserID:     uuid.New().String(),
		Name:       "Used Token",
		TokenHash:  "hash",
		Prefix:     "tk_used",
		Status:     "active",
		LastUsedAt: &lastUsed,
		UsageCount: 42,
	}

	assert.NotNil(t, token.LastUsedAt)
	assert.Equal(t, int64(42), token.UsageCount)
}

func TestAPIToken_RevokedStatus(t *testing.T) {
	token := tenant.APIToken{
		ID:        uuid.New(),
		UserID:    uuid.New().String(),
		Name:      "Revoked Token",
		TokenHash: "hash",
		Prefix:    "tk_rev",
		Status:    "revoked",
	}

	assert.Equal(t, "revoked", token.Status)
}

func TestAPIToken_BaseScoped(t *testing.T) {
	workspaceID := uuid.New().String()
	baseID := uuid.New().String()

	token := tenant.APIToken{
		ID:          uuid.New(),
		UserID:      uuid.New().String(),
		WorkspaceID: &workspaceID,
		BaseID:      &baseID,
		Name:        "Base-Scoped Token",
		TokenHash:   "hash",
		Prefix:      "tk_base",
		Status:      "active",
	}

	assert.NotNil(t, token.WorkspaceID)
	assert.NotNil(t, token.BaseID)
	assert.Equal(t, baseID, *token.BaseID)
}

func TestAPIToken_TableSchema(t *testing.T) {
	token := tenant.APIToken{}
	schema := "test_schema"

	tableSchema := token.TableSchema(schema)

	assert.NotNil(t, tableSchema)
	assert.Equal(t, `"test_schema".api_tokens`, tableSchema.Name)
	assert.NotEmpty(t, tableSchema.Columns)
}
