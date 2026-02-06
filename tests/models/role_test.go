package models_test

import (
	"testing"
	"time"

	"serenibase/internal/models/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRole_TableName(t *testing.T) {
	role := tenant.Role{}
	schema := "test_schema"

	tableName := role.TableName(schema)

	assert.Equal(t, `"test_schema".roles`, tableName)
}

func TestRole_Fields(t *testing.T) {
	roleID := uuid.New()
	now := time.Now().UTC()
	desc := "Administrator role with full access"

	role := tenant.Role{
		ID:          roleID,
		Name:        "Admin",
		Description: &desc,
		IsDefault:   false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, roleID, role.ID)
	assert.Equal(t, "Admin", role.Name)
	assert.Equal(t, "Administrator role with full access", *role.Description)
	assert.False(t, role.IsDefault)
}

func TestRole_DefaultRole(t *testing.T) {
	role := tenant.Role{
		ID:        uuid.New(),
		Name:      "Member",
		IsDefault: true,
	}

	assert.Equal(t, "Member", role.Name)
	assert.True(t, role.IsDefault)
}

func TestRole_WithoutDescription(t *testing.T) {
	role := tenant.Role{
		ID:          uuid.New(),
		Name:        "Custom Role",
		Description: nil,
		IsDefault:   false,
	}

	assert.Nil(t, role.Description)
}

func TestRole_CommonRoles(t *testing.T) {
	testCases := []struct {
		name      string
		roleName  string
		isDefault bool
	}{
		{"admin", "Admin", false},
		{"member", "Member", true},
		{"viewer", "Viewer", true},
		{"editor", "Editor", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			role := tenant.Role{
				ID:        uuid.New(),
				Name:      tc.roleName,
				IsDefault: tc.isDefault,
			}

			assert.Equal(t, tc.roleName, role.Name)
			assert.Equal(t, tc.isDefault, role.IsDefault)
		})
	}
}

func TestRole_TableSchema(t *testing.T) {
	role := tenant.Role{}
	schema := "test_schema"

	tableSchema := role.TableSchema(schema)

	assert.NotNil(t, tableSchema)
	assert.Equal(t, `"test_schema".roles`, tableSchema.Name)
	assert.NotEmpty(t, tableSchema.Columns)
	assert.NotEmpty(t, tableSchema.Indexes)
}
