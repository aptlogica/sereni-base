package models_test

import (
	"testing"
	"time"

	"github.com/aptlogica/sereni-base/internal/models/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAction_TableName(t *testing.T) {
	action := tenant.Action{}
	schema := "test_schema"

	tableName := action.TableName(schema)

	assert.Equal(t, `"test_schema".actions`, tableName)
}

func TestAction_Fields(t *testing.T) {
	actionID := uuid.New()
	now := time.Now().UTC()
	desc := "Read access to resources"

	action := tenant.Action{
		ID:          actionID,
		Code:        "read",
		Description: &desc,
		CreatedAt:   now,
	}

	assert.Equal(t, actionID, action.ID)
	assert.Equal(t, "read", action.Code)
	assert.Equal(t, "Read access to resources", *action.Description)
	assert.Equal(t, now, action.CreatedAt)
}

func TestAction_CommonActions(t *testing.T) {
	testCases := []struct {
		name string
		code string
		desc string
	}{
		{"read", "read", "Read access"},
		{"create", "create", "Create new items"},
		{"update", "update", "Update existing items"},
		{"delete", "delete", "Delete items"},
		{"share", "share", "Share with others"},
		{"invite", "invite", "Invite members"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			desc := tc.desc
			action := tenant.Action{
				ID:          uuid.New(),
				Code:        tc.code,
				Description: &desc,
				CreatedAt:   time.Now().UTC(),
			}

			assert.Equal(t, tc.code, action.Code)
			assert.Equal(t, tc.desc, *action.Description)
		})
	}
}

func TestAction_WithoutDescription(t *testing.T) {
	action := tenant.Action{
		ID:          uuid.New(),
		Code:        "custom_action",
		Description: nil,
		CreatedAt:   time.Now().UTC(),
	}

	assert.Nil(t, action.Description)
}

func TestAction_TableSchema(t *testing.T) {
	action := tenant.Action{}
	schema := "test_schema"

	tableSchema := action.TableSchema(schema)

	assert.NotNil(t, tableSchema)
	assert.Equal(t, `"test_schema".actions`, tableSchema.Name)
	assert.NotEmpty(t, tableSchema.Columns)
	assert.NotEmpty(t, tableSchema.Indexes)
}
