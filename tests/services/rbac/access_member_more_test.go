package rbac_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aptlogica/go-postgres-rest/pkg"
	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	services "github.com/aptlogica/sereni-base/internal/services/rbac"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAccessMember_GetUserPermissionsAndCheck(t *testing.T) {
	schema := "schema"
	userID := "user"
	roleID := uuid.New()
	permID := uuid.New()
	resourceID := uuid.New()
	actionID := uuid.New()

	stubTable := &StubTableService{}
	stubTable.GetTableDataFn = func(tableName string, _ dbModels.QueryParams) ([]map[string]interface{}, error) {
		return nil, nil
	}

	// wire table responses by name
	stubTable.GetTableDataFn = func(tableName string, _ dbModels.QueryParams) ([]map[string]interface{}, error) {
		switch tableName {
		case tenant.AccessMember{}.TableName(schema):
			return []map[string]interface{}{{
				"id":         uuid.New().String(),
				"user_id":    userID,
				"scope_type": "workspace",
				"scope_id":   uuid.New().String(),
				"role_id":    roleID.String(),
			}, {
				"id":         uuid.New().String(),
				"user_id":    userID,
				"scope_type": "workspace",
				"scope_id":   uuid.New().String(),
				"role_id":    "bad-uuid",
			}}, nil
		case tenant.RolePermission{}.TableName(schema):
			return []map[string]interface{}{{
				"id":            uuid.New().String(),
				"role_id":       roleID.String(),
				"permission_id": permID.String(),
			}}, nil
		case tenant.Permission{}.TableName(schema):
			return []map[string]interface{}{{
				"id":          permID.String(),
				"resource_id": resourceID.String(),
				"action_id":   actionID.String(),
			}}, nil
		case tenant.Resource{}.TableName(schema):
			return []map[string]interface{}{{"id": resourceID.String(), "code": "workspace"}}, nil
		case tenant.Action{}.TableName(schema):
			return []map[string]interface{}{{"id": actionID.String(), "code": "read"}}, nil
		default:
			return []map[string]interface{}{}, nil
		}
	}

	repo := &pkg.DatabaseService{TableService: stubTable}
	svc := services.NewAccessMemberService(repo)

	perms, err := svc.GetUserPermissions(context.Background(), schema, userID, "workspace", nil)
	assert.NoError(t, err)
	assert.Len(t, perms, 1)

	ok, err := svc.CheckUserPermission(context.Background(), schema, userID, "workspace", nil, "workspace", "read")
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestAccessMember_GetUserHighestRole(t *testing.T) {
	schema := "schema"
	userID := "user"
	roleLow := uuid.New()
	roleHigh := uuid.New()

	stubTable := &StubTableService{}
	stubTable.GetTableDataFn = func(tableName string, _ dbModels.QueryParams) ([]map[string]interface{}, error) {
		switch tableName {
		case tenant.AccessMember{}.TableName(schema):
			return []map[string]interface{}{{
				"id":         uuid.New().String(),
				"user_id":    userID,
				"scope_type": "workspace",
				"scope_id":   uuid.New().String(),
				"role_id":    roleLow.String(),
			}, {
				"id":         uuid.New().String(),
				"user_id":    userID,
				"scope_type": "workspace",
				"scope_id":   uuid.New().String(),
				"role_id":    roleHigh.String(),
			}}, nil
		case tenant.AccessRole{}.TableName(schema):
			// return high priority role
			return []map[string]interface{}{{
				"id":          roleHigh.String(),
				"name":        "owner",
				"scope_level": "workspace",
				"priority":    100,
			}}, nil
		default:
			return []map[string]interface{}{}, nil
		}
	}

	repo := &pkg.DatabaseService{TableService: stubTable}
	svc := services.NewAccessMemberService(repo)

	role, err := svc.GetUserHighestRole(context.Background(), schema, userID, "workspace", nil)
	assert.NoError(t, err)
	assert.Equal(t, roleHigh, role.ID)
}

func TestAccessMember_GetUserHighestRole_Errors(t *testing.T) {
	repo := &pkg.DatabaseService{TableService: &StubTableService{}}
	svc := services.NewAccessMemberService(repo)

	_, err := svc.GetUserHighestRole(context.Background(), "schema", "", "workspace", nil)
	assert.ErrorIs(t, err, app_errors.ErrRecordNotFound)

	_, err = svc.GetUserHighestRole(context.Background(), "schema", "user", "", nil)
	assert.ErrorIs(t, err, app_errors.InvalidScopeType)
}

func TestAccessMember_BulkAssignAndRemove(t *testing.T) {
	schema := "schema"

	stubTable := &StubTableService{}
	stubTable.CreateRecordFn = func(tableName string, data map[string]interface{}) (map[string]interface{}, error) {
		return nil, errors.New("fail")
	}
	repo := &pkg.DatabaseService{TableService: stubTable}
	svc := services.NewAccessMemberService(repo)

	err := svc.BulkAssignRoleToUsers(context.Background(), schema, dto.BulkAssignRoleRequest{})
	assert.ErrorIs(t, err, app_errors.EmptyUserList)

	err = svc.BulkAssignRoleToUsers(context.Background(), schema, dto.BulkAssignRoleRequest{UserIDs: []string{"u1"}, ScopeType: "workspace", RoleID: uuid.New().String()})
	assert.ErrorIs(t, err, app_errors.BulkAssignmentFailed)

	err = svc.BulkRemoveRoleFromUsers(context.Background(), schema, []string{}, "workspace", nil, "role")
	assert.ErrorIs(t, err, app_errors.EmptyUserList)

	err = svc.BulkRemoveRoleFromUsers(context.Background(), schema, []string{"u1"}, "", nil, "role")
	assert.ErrorIs(t, err, app_errors.InvalidScopeType)
}

func TestAccessMember_UpdateRoleForUser(t *testing.T) {
	schema := "schema"
	userID := "user"
	roleID := uuid.New().String()

	stubTable := &StubTableService{}
	stubTable.GetTableDataFn = func(tableName string, _ dbModels.QueryParams) ([]map[string]interface{}, error) {
		switch tableName {
		case tenant.AccessMember{}.TableName(schema):
			return []map[string]interface{}{{"id": uuid.New().String()}}, nil
		default:
			return []map[string]interface{}{}, nil
		}
	}
	stubTable.UpdateRecordFn = func(tableName string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"id": id}, nil
	}

	repo := &pkg.DatabaseService{TableService: stubTable}
	svc := services.NewAccessMemberService(repo)

	err := svc.UpdateRoleForUser(context.Background(), schema, "", "workspace", nil, roleID)
	assert.ErrorIs(t, err, app_errors.ErrRecordNotFound)

	err = svc.UpdateRoleForUser(context.Background(), schema, userID, "", nil, roleID)
	assert.ErrorIs(t, err, app_errors.InvalidScopeType)

	err = svc.UpdateRoleForUser(context.Background(), schema, userID, "workspace", nil, roleID)
	assert.NoError(t, err)
}

func TestAccessMember_GetUserAccessMembers_MapError(t *testing.T) {
	schema := "schema"
	stubTable := &StubTableService{}
	stubTable.GetTableDataFn = func(tableName string, _ dbModels.QueryParams) ([]map[string]interface{}, error) {
		return []map[string]interface{}{{"id": make(chan int)}}, nil
	}

	repo := &pkg.DatabaseService{TableService: stubTable}
	svc := services.NewAccessMemberService(repo)

	rows, err := svc.GetUserAccessMembers(context.Background(), schema, "user")
	assert.NoError(t, err)
	assert.Len(t, rows, 0)
}
