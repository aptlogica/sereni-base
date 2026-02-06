package models_test

import (
	"testing"
	"time"

	"serenibase/internal/models/tenant"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUserModel_TableName(t *testing.T) {
	user := tenant.User{}
	schema := "test_schema"

	tableName := user.TableName(schema)

	assert.Contains(t, tableName, schema)
	assert.Contains(t, tableName, "users")
}

func TestUserModel_Fields(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC()

	user := tenant.User{
		ID:            userID,
		Email:         "test@example.com",
		FirstName:     "John",
		LastName:      "Doe",
		DisplayName:   "John Doe",
		EmailVerified: true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	assert.Equal(t, userID, user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "John", user.FirstName)
	assert.Equal(t, "Doe", user.LastName)
	assert.True(t, user.EmailVerified)
}

func TestBaseModel_TableName(t *testing.T) {
	base := tenant.Base{}
	schema := "test_schema"

	tableName := base.TableName(schema)

	assert.Contains(t, tableName, schema)
	assert.Contains(t, tableName, "bases")
}

func TestWorkspaceModel_TableName(t *testing.T) {
	workspace := tenant.Workspace{}
	schema := "test_schema"

	tableName := workspace.TableName(schema)

	assert.Contains(t, tableName, schema)
	assert.Contains(t, tableName, "workspaces")
}

func TestOrganizationModel_TableName(t *testing.T) {
	org := tenant.Organization{}
	schema := "test_schema"

	tableName := org.TableName(schema)

	assert.Contains(t, tableName, schema)
	assert.Contains(t, tableName, "organizations")
}

func TestAssetsModel_TableName(t *testing.T) {
	asset := tenant.Assets{}
	schema := "test_schema"

	tableName := asset.TableName(schema)

	assert.Contains(t, tableName, schema)
	assert.Contains(t, tableName, "assets")
}

func TestAssetsModel_Map(t *testing.T) {
	assetID := uuid.New()
	now := time.Now().UTC()

	asset := tenant.Assets{
		ID:           assetID,
		Title:        "test-image.png",
		Url:          "https://storage.example.com/test-image.png",
		ThumbnailUrl: "https://storage.example.com/thumb.png",
		MimeType:     "image/png",
		Size:         1024000,
		Width:        1920,
		Height:       1080,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	mapped := asset.Map()

	assert.Equal(t, assetID, mapped["id"])
	assert.Equal(t, "test-image.png", mapped["title"])
	assert.Equal(t, "https://storage.example.com/test-image.png", mapped["url"])
	assert.Equal(t, "image/png", mapped["mime_type"])
	assert.Equal(t, int64(1024000), mapped["size"])
}

func TestWorkspaceMemberModel_TableName(t *testing.T) {
	member := tenant.WorkspaceMember{}
	schema := "test_schema"

	tableName := member.TableName(schema)

	assert.Contains(t, tableName, schema)
	assert.Contains(t, tableName, "workspace_members")
}

func TestAccessRoleModel_TableName(t *testing.T) {
	role := tenant.AccessRole{}
	schema := "test_schema"

	tableName := role.TableName(schema)

	assert.Contains(t, tableName, schema)
	assert.Contains(t, tableName, "access_roles")
}

func TestAccessMemberModel_TableName(t *testing.T) {
	member := tenant.AccessMember{}
	schema := "test_schema"

	tableName := member.TableName(schema)

	assert.Contains(t, tableName, schema)
	assert.Contains(t, tableName, "access_members")
}

func TestPermissionModel_TableName(t *testing.T) {
	permission := tenant.Permission{}
	schema := "test_schema"

	tableName := permission.TableName(schema)

	assert.Contains(t, tableName, schema)
	assert.Contains(t, tableName, "permissions")
}

func TestResourceModel_TableName(t *testing.T) {
	resource := tenant.Resource{}
	schema := "test_schema"

	tableName := resource.TableName(schema)

	assert.Contains(t, tableName, schema)
	assert.Contains(t, tableName, "resources")
}

func TestActionModel_TableName(t *testing.T) {
	action := tenant.Action{}
	schema := "test_schema"

	tableName := action.TableName(schema)

	assert.Contains(t, tableName, schema)
	assert.Contains(t, tableName, "actions")
}

func TestRolePermissionModel_TableName(t *testing.T) {
	rolePerm := tenant.RolePermission{}
	schema := "test_schema"

	tableName := rolePerm.TableName(schema)

	assert.Contains(t, tableName, schema)
	assert.Contains(t, tableName, "role_permissions")
}

func TestUserResetTokenModel_TableName(t *testing.T) {
	token := tenant.UserResetToken{}
	schema := "test_schema"

	tableName := token.TableName(schema)

	assert.Contains(t, tableName, schema)
	assert.Contains(t, tableName, "user_reset_tokens")
}

func TestAPITokenModel_TableName(t *testing.T) {
	apiToken := tenant.APIToken{}
	schema := "test_schema"

	tableName := apiToken.TableName(schema)

	assert.Contains(t, tableName, schema)
	assert.Contains(t, tableName, "api_tokens")
}
