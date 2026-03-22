package models_test

import (
	"testing"
	"time"

	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUser_TableName_UserFile(t *testing.T) {
	user := tenant.User{}

	tests := []struct {
		name     string
		prefix   string
		expected string
	}{
		{"with public prefix", "public", "\"public\".users"},
		{"with tenant prefix", "tenant_123", "\"tenant_123\".users"},
		{"with custom prefix", "custom_schema", "\"custom_schema\".users"},
		{"with empty prefix", "", "\"\".users"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := user.TableName(tt.prefix)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUser_TableSchema_UserFile(t *testing.T) {
	user := tenant.User{}

	t.Run("generates valid schema", func(t *testing.T) {
		schema := user.TableSchema("public")

		assert.NotEmpty(t, schema.Name)
		assert.NotEmpty(t, schema.Columns)

		// Verify required columns exist
		columnNames := make(map[string]bool)
		for _, col := range schema.Columns {
			columnNames[col.Name] = true
		}

		requiredColumns := []string{"id", "email", "password", "first_name", "last_name", "status"}
		for _, colName := range requiredColumns {
			assert.True(t, columnNames[colName], "Column %s should exist", colName)
		}
	})
}

func TestUser_Fields_UserFile(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	user := tenant.User{
		ID:            id,
		Email:         "test@example.com",
		Password:      "hashedpassword",
		FirstName:     "John",
		LastName:      "Doe",
		DisplayName:   "John Doe",
		Avatar:        "https://example.com/avatar.png",
		Status:        "active",
		Timezone:      "UTC",
		Locale:        "en",
		MFAEnabled:    true,
		EmailVerified: true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	assert.Equal(t, id, user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "John", user.FirstName)
	assert.Equal(t, "Doe", user.LastName)
	assert.Equal(t, "John Doe", user.DisplayName)
	assert.Equal(t, "active", user.Status)
	assert.True(t, user.MFAEnabled)
	assert.True(t, user.EmailVerified)
}

func TestWorkspace_TableName_UserFile(t *testing.T) {
	workspace := tenant.Workspace{}

	tests := []struct {
		name     string
		prefix   string
		expected string
	}{
		{"with public prefix", "public", "\"public\".workspaces"},
		{"with tenant prefix", "tenant_abc", "\"tenant_abc\".workspaces"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := workspace.TableName(tt.prefix)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWorkspace_TableSchema_UserFile(t *testing.T) {
	workspace := tenant.Workspace{}

	t.Run("generates valid schema", func(t *testing.T) {
		schema := workspace.TableSchema("public")

		assert.NotEmpty(t, schema.Name)
		assert.NotEmpty(t, schema.Columns)

		// Check for indexes
		assert.NotEmpty(t, schema.Indexes)
	})
}

func TestWorkspace_Fields_UserFile(t *testing.T) {
	now := time.Now()
	id := uuid.New()
	description := "My workspace description"

	workspace := tenant.Workspace{
		ID:          id,
		Title:       "My Workspace",
		Description: &description,
		Slug:        "my-workspace",
		IsDefault:   true,
		Status:      "active",
		CreatedBy:   "user-123",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assert.Equal(t, id, workspace.ID)
	assert.Equal(t, "My Workspace", workspace.Title)
	assert.Equal(t, description, *workspace.Description)
	assert.Equal(t, "my-workspace", workspace.Slug)
	assert.True(t, workspace.IsDefault)
	assert.Equal(t, "active", workspace.Status)
}

func TestBase_TableName_Extended_UserFile(t *testing.T) {
	base := tenant.Base{}

	tests := []struct {
		name     string
		prefix   string
		expected string
	}{
		{"with public prefix", "public", "\"public\".bases"},
		{"with tenant prefix", "org_xyz", "\"org_xyz\".bases"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := base.TableName(tt.prefix)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBase_TableSchema_Extended_UserFile(t *testing.T) {
	base := tenant.Base{}

	t.Run("generates valid schema with foreign keys", func(t *testing.T) {
		schema := base.TableSchema("public")

		assert.NotEmpty(t, schema.Name)
		assert.NotEmpty(t, schema.Columns)
		assert.NotEmpty(t, schema.Indexes)
		assert.NotEmpty(t, schema.ForeignKeys)

		// Verify workspace_id foreign key
		hasWorkspaceFk := false
		for _, fk := range schema.ForeignKeys {
			if fk.Name == "fk_bases_workspace_id" {
				hasWorkspaceFk = true
				assert.Equal(t, "CASCADE", fk.OnDelete)
			}
		}
		assert.True(t, hasWorkspaceFk, "Should have workspace_id foreign key")
	})
}

func TestBase_Fields_Extended_UserFile(t *testing.T) {
	now := time.Now()
	id := uuid.New()
	description := "Test base"

	base := tenant.Base{
		ID:               id,
		WorkspaceID:      "ws-123",
		Title:            "My Base",
		Description:      &description,
		Type:             "internal",
		Status:           "active",
		Visibility:       "private",
		TableCount:       5,
		RowCount:         1000,
		StorageUsedBytes: 1048576,
		CreatedBy:        "user-123",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	assert.Equal(t, id, base.ID)
	assert.Equal(t, "ws-123", base.WorkspaceID)
	assert.Equal(t, "My Base", base.Title)
	assert.Equal(t, "internal", base.Type)
	assert.Equal(t, 5, base.TableCount)
	assert.Equal(t, int64(1000), base.RowCount)
	assert.Equal(t, int64(1048576), base.StorageUsedBytes)
}

func TestWorkspaceMember_TableName_UserFile(t *testing.T) {
	member := tenant.WorkspaceMember{}

	result := member.TableName("public")
	assert.Equal(t, "\"public\".workspace_members", result)
}

func TestAssets_TableName_UserFile(t *testing.T) {
	asset := tenant.Assets{}

	result := asset.TableName("tenant_123")
	assert.Equal(t, "\"tenant_123\".assets", result)
}

func TestColumn_TableName_UserFile(t *testing.T) {
	column := tenant.Column{}

	result := column.TableName("schema_abc")
	assert.Equal(t, "\"schema_abc\".columns", result)
}

func TestView_TableName_UserFile(t *testing.T) {
	view := tenant.View{}

	result := view.TableName("public")
	assert.Equal(t, "\"public\".views", result)
}

func TestModel_TableName_UserFile(t *testing.T) {
	model := tenant.Model{}

	result := model.TableName("app")
	assert.Equal(t, "\"app\".models", result)
}

func TestOrganization_TableName_UserFile(t *testing.T) {
	org := tenant.Organization{}

	result := org.TableName("public")
	assert.Equal(t, "\"public\".organizations", result)
}

func TestRole_TableName_UserFile(t *testing.T) {
	role := tenant.Role{}

	result := role.TableName("tenant_1")
	assert.Equal(t, "\"tenant_1\".roles", result)
}

func TestPermission_TableName_UserFile(t *testing.T) {
	permission := tenant.Permission{}

	result := permission.TableName("public")
	assert.Equal(t, "\"public\".permissions", result)
}

func TestResource_TableName_UserFile(t *testing.T) {
	resource := tenant.Resource{}

	result := resource.TableName("public")
	assert.Equal(t, "\"public\".resources", result)
}

func TestAction_TableName_UserFile(t *testing.T) {
	action := tenant.Action{}

	result := action.TableName("public")
	assert.Equal(t, "\"public\".actions", result)
}

func TestUserResetToken_TableName_UserFile(t *testing.T) {
	token := tenant.UserResetToken{}

	result := token.TableName("public")
	assert.Equal(t, "\"public\".user_reset_tokens", result)
}

func TestAPIToken_TableName_UserFile(t *testing.T) {
	apiToken := tenant.APIToken{}

	result := apiToken.TableName("tenant_x")
	assert.Equal(t, "\"tenant_x\".api_tokens", result)
}

func TestAccessMember_TableName_UserFile(t *testing.T) {
	member := tenant.AccessMember{}

	result := member.TableName("public")
	assert.Equal(t, "\"public\".access_members", result)
}

func TestAccessRole_TableName_UserFile(t *testing.T) {
	role := tenant.AccessRole{}

	result := role.TableName("public")
	assert.Equal(t, "\"public\".access_roles", result)
}

func TestRolePermission_TableName_UserFile(t *testing.T) {
	rp := tenant.RolePermission{}

	result := rp.TableName("public")
	assert.Equal(t, "\"public\".role_permissions", result)
}

func TestRelation_TableName_UserFile(t *testing.T) {
	relation := tenant.Relation{}

	result := relation.TableName("public")
	assert.Equal(t, "\"public\".relations", result)
}

func TestFeatureFlag_TableName_UserFile(t *testing.T) {
	flag := tenant.FeatureFlag{}

	result := flag.TableName("public")
	assert.Equal(t, "\"public\".shared.feature_flags", result)
}

func TestGlobalAuditLog_TableName_UserFile(t *testing.T) {
	log := tenant.GlobalAuditLog{}

	result := log.TableName("public")
	assert.Equal(t, "\"public\".global_audit_logs", result)
}

func TestHook_TableName_UserFile(t *testing.T) {
	hook := tenant.Hook{}

	result := hook.TableName("public")
	assert.Equal(t, "\"public\".hooks", result)
}

func TestUsageMetric_TableName_UserFile(t *testing.T) {
	metric := tenant.UsageMetric{}

	result := metric.TableName("public")
	assert.Equal(t, "\"public\".usage_metrics", result)
}

func TestViewColumn_TableName_UserFile(t *testing.T) {
	vc := tenant.ViewColumn{}

	result := vc.TableName("public")
	assert.Equal(t, "\"public\".view_columns", result)
}

