package models_test

import (
	"testing"
	"time"

	"github.com/aptlogica/sereni-base/internal/models/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestResource_TableName(t *testing.T) {
	resource := tenant.Resource{}
	schema := "test_schema"

	tableName := resource.TableName(schema)

	assert.Equal(t, `"test_schema".resources`, tableName)
}

func TestResource_Fields(t *testing.T) {
	resourceID := uuid.New()
	now := time.Now().UTC()
	desc := "Workspace management"

	resource := tenant.Resource{
		ID:          resourceID,
		Code:        "workspace",
		Description: &desc,
		CreatedAt:   now,
	}

	assert.Equal(t, resourceID, resource.ID)
	assert.Equal(t, "workspace", resource.Code)
	assert.Equal(t, "Workspace management", *resource.Description)
	assert.Equal(t, now, resource.CreatedAt)
}

func TestResource_CommonResources(t *testing.T) {
	testCases := []struct {
		name string
		code string
		desc string
	}{
		{"workspace", "workspace", "Workspace resources"},
		{"base", "base", "Base resources"},
		{"records", "records", "Record data"},
		{"members", "members", "Member management"},
		{"settings", "settings", "Configuration settings"},
		{"api_tokens", "api_tokens", "API token management"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			desc := tc.desc
			resource := tenant.Resource{
				ID:          uuid.New(),
				Code:        tc.code,
				Description: &desc,
				CreatedAt:   time.Now().UTC(),
			}

			assert.Equal(t, tc.code, resource.Code)
			assert.Equal(t, tc.desc, *resource.Description)
		})
	}
}

func TestResource_WithoutDescription(t *testing.T) {
	resource := tenant.Resource{
		ID:          uuid.New(),
		Code:        "custom_resource",
		Description: nil,
		CreatedAt:   time.Now().UTC(),
	}

	assert.Nil(t, resource.Description)
}

func TestResource_TableSchema(t *testing.T) {
	resource := tenant.Resource{}
	schema := "test_schema"

	tableSchema := resource.TableSchema(schema)

	assert.NotNil(t, tableSchema)
	assert.Equal(t, `"test_schema".resources`, tableSchema.Name)
	assert.NotEmpty(t, tableSchema.Columns)
	assert.NotEmpty(t, tableSchema.Indexes)
}
