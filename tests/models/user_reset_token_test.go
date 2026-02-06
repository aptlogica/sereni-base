package models_test

import (
	"testing"

	"serenibase/internal/models/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUserResetToken_TableName(t *testing.T) {
	token := tenant.UserResetToken{}
	schema := "test_schema"

	tableName := token.TableName(schema)

	assert.Equal(t, `"test_schema".user_reset_tokens`, tableName)
}

func TestUserResetToken_Fields(t *testing.T) {
	tokenID := uuid.New()
	userID := uuid.New()
	tokenValue := "random_secure_token_12345"
	issuedAt := "2026-02-02T10:30:00Z"

	token := tenant.UserResetToken{
		ID:       tokenID,
		Token:    tokenValue,
		UserID:   userID,
		IssuedAt: issuedAt,
	}

	assert.Equal(t, tokenID, token.ID)
	assert.Equal(t, tokenValue, token.Token)
	assert.Equal(t, userID, token.UserID)
	assert.Equal(t, issuedAt, token.IssuedAt)
}

func TestUserResetToken_MultipleTokensPerUser(t *testing.T) {
	userID := uuid.New()

	token1 := tenant.UserResetToken{
		ID:       uuid.New(),
		Token:    "token_1",
		UserID:   userID,
		IssuedAt: "2026-02-02T10:00:00Z",
	}

	token2 := tenant.UserResetToken{
		ID:       uuid.New(),
		Token:    "token_2",
		UserID:   userID,
		IssuedAt: "2026-02-02T11:00:00Z",
	}

	assert.Equal(t, userID, token1.UserID)
	assert.Equal(t, userID, token2.UserID)
	assert.NotEqual(t, token1.Token, token2.Token)
	assert.NotEqual(t, token1.ID, token2.ID)
}

func TestUserResetToken_UniqueTokens(t *testing.T) {
	tokens := []tenant.UserResetToken{
		{
			ID:       uuid.New(),
			Token:    "token_abc123",
			UserID:   uuid.New(),
			IssuedAt: "2026-02-02T10:00:00Z",
		},
		{
			ID:       uuid.New(),
			Token:    "token_def456",
			UserID:   uuid.New(),
			IssuedAt: "2026-02-02T10:05:00Z",
		},
		{
			ID:       uuid.New(),
			Token:    "token_ghi789",
			UserID:   uuid.New(),
			IssuedAt: "2026-02-02T10:10:00Z",
		},
	}

	// All tokens should be unique
	tokenMap := make(map[string]bool)
	for _, token := range tokens {
		assert.False(t, tokenMap[token.Token], "Token should be unique")
		tokenMap[token.Token] = true
	}
}

func TestUserResetToken_TableSchema(t *testing.T) {
	token := tenant.UserResetToken{}
	schema := "test_schema"

	tableSchema := token.TableSchema(schema)

	assert.NotNil(t, tableSchema)
	assert.Equal(t, `"test_schema".user_reset_tokens`, tableSchema.Name)
	assert.NotEmpty(t, tableSchema.Columns)
	assert.NotEmpty(t, tableSchema.Indexes)
}
