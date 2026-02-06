package rbac_test

import (
	"context"
	"errors"
	"testing"

	"go-postgres-rest/pkg"
	dbModels "go-postgres-rest/pkg/models"
	app_errors "serenibase/internal/app-errors"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	services "serenibase/internal/services/rbac"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAccessMember_AssignRoleToUser_MapError(t *testing.T) {
	stubTable := &StubTableService{}
	stubTable.CreateRecordFn = func(tableName string, data map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"id": make(chan int)}, nil
	}

	repo := &pkg.DatabaseService{TableService: stubTable}
	svc := services.NewAccessMemberService(repo)

	_, err := svc.AssignRoleToUser(context.Background(), "schema", dto.AccessMemberDTO{
		UserID:    "user",
		ScopeType: "workspace",
	})
	assert.Error(t, err)
}

func TestAccessMember_RemoveRoleFromUser_MapError(t *testing.T) {
	stubTable := &StubTableService{}
	stubTable.GetTableDataFn = func(tableName string, _ dbModels.QueryParams) ([]map[string]interface{}, error) {
		return []map[string]interface{}{{"id": make(chan int)}}, nil
	}

	repo := &pkg.DatabaseService{TableService: stubTable}
	svc := services.NewAccessMemberService(repo)

	err := svc.RemoveRoleFromUser(context.Background(), "schema", "user", "scope", "workspace")
	assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
}

func TestAccessMember_GetUserAccessByScope_GetDataError(t *testing.T) {
	stubTable := &StubTableService{}
	stubTable.GetTableDataFn = func(tableName string, _ dbModels.QueryParams) ([]map[string]interface{}, error) {
		return nil, errors.New("db error")
	}

	repo := &pkg.DatabaseService{TableService: stubTable}
	svc := services.NewAccessMemberService(repo)

	_, err := svc.GetUserAccessByScope(context.Background(), "schema", "user", "workspace", nil)
	assert.Error(t, err)
}

func TestAccessMember_GetUserPermissions_EmptyAndCheckFalse(t *testing.T) {
	stubTable := &StubTableService{}
	stubTable.GetTableDataFn = func(tableName string, _ dbModels.QueryParams) ([]map[string]interface{}, error) {
		if tableName == (tenant.AccessMember{}).TableName("schema") {
			return []map[string]interface{}{}, nil
		}
		return nil, nil
	}

	repo := &pkg.DatabaseService{TableService: stubTable}
	svc := services.NewAccessMemberService(repo)

	perms, err := svc.GetUserPermissions(context.Background(), "schema", "user", "workspace", nil)
	assert.NoError(t, err)
	assert.Len(t, perms, 0)

	ok, err := svc.CheckUserPermission(context.Background(), "schema", "user", "workspace", nil, "workspace", "read")
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestAccessMember_GetUserPermissions_ErrorPropagation(t *testing.T) {
	stubTable := &StubTableService{}
	stubTable.GetTableDataFn = func(tableName string, _ dbModels.QueryParams) ([]map[string]interface{}, error) {
		return nil, errors.New("db error")
	}

	repo := &pkg.DatabaseService{TableService: stubTable}
	svc := services.NewAccessMemberService(repo)

	_, err := svc.GetUserPermissions(context.Background(), "schema", "user", "workspace", nil)
	assert.Error(t, err)

	ok, err := svc.CheckUserPermission(context.Background(), "schema", "user", "workspace", nil, "workspace", "read")
	assert.Error(t, err)
	assert.False(t, ok)
}

func TestAccessMember_GetUserHighestRole_NoAccessAndRoleNotFound(t *testing.T) {
	schema := "schema"
	roleID := uuid.New().String()

	stubTable := &StubTableService{}
	stubTable.GetTableDataFn = func(tableName string, _ dbModels.QueryParams) ([]map[string]interface{}, error) {
		switch tableName {
		case (tenant.AccessMember{}).TableName(schema):
			return []map[string]interface{}{}, nil
		case (tenant.AccessRole{}).TableName(schema):
			return []map[string]interface{}{}, nil
		default:
			return []map[string]interface{}{}, nil
		}
	}

	repo := &pkg.DatabaseService{TableService: stubTable}
	svc := services.NewAccessMemberService(repo)

	_, err := svc.GetUserHighestRole(context.Background(), schema, "user", "workspace", nil)
	assert.ErrorIs(t, err, app_errors.AccessMemberNotFound)

	// Provide access member but no role record
	stubTable.GetTableDataFn = func(tableName string, _ dbModels.QueryParams) ([]map[string]interface{}, error) {
		switch tableName {
		case (tenant.AccessMember{}).TableName(schema):
			return []map[string]interface{}{{
				"id":         uuid.New().String(),
				"user_id":    "user",
				"scope_type": "workspace",
				"scope_id":   uuid.New().String(),
				"role_id":    roleID,
			}}, nil
		case (tenant.AccessRole{}).TableName(schema):
			return []map[string]interface{}{}, nil
		default:
			return []map[string]interface{}{}, nil
		}
	}

	_, err = svc.GetUserHighestRole(context.Background(), schema, "user", "workspace", nil)
	assert.ErrorIs(t, err, app_errors.RoleNotFound)
}

func TestAccessMember_GetUserHighestRole_InvalidRoleID(t *testing.T) {
	schema := "schema"
	stubTable := &StubTableService{}
	stubTable.GetTableDataFn = func(tableName string, _ dbModels.QueryParams) ([]map[string]interface{}, error) {
		if tableName == (tenant.AccessMember{}).TableName(schema) {
			return []map[string]interface{}{{
				"id":         uuid.New().String(),
				"user_id":    "user",
				"scope_type": "workspace",
				"scope_id":   uuid.New().String(),
				"role_id":    "not-a-uuid",
			}}, nil
		}
		return []map[string]interface{}{}, nil
	}

	repo := &pkg.DatabaseService{TableService: stubTable}
	svc := services.NewAccessMemberService(repo)

	_, err := svc.GetUserHighestRole(context.Background(), schema, "user", "workspace", nil)
	assert.ErrorIs(t, err, app_errors.ErrRecordNotFound)
}

func TestAccessMember_BulkRemoveRoleFromUsers_AllFailed(t *testing.T) {
	stubTable := &StubTableService{}
	stubTable.GetTableDataFn = func(tableName string, _ dbModels.QueryParams) ([]map[string]interface{}, error) {
		return []map[string]interface{}{}, nil
	}

	repo := &pkg.DatabaseService{TableService: stubTable}
	svc := services.NewAccessMemberService(repo)

	userIDs := []string{"u1", "u2"}
	err := svc.BulkRemoveRoleFromUsers(context.Background(), "schema", userIDs, "workspace", nil, "role")
	assert.ErrorIs(t, err, app_errors.BulkRemovalFailed)
}

func TestAccessMember_BulkAssignAndRemove_SkipEmptyUser(t *testing.T) {
	stubTable := &StubTableService{}
	stubTable.CreateRecordFn = func(tableName string, data map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"id": uuid.New().String()}, nil
	}
	stubTable.GetTableDataFn = func(tableName string, _ dbModels.QueryParams) ([]map[string]interface{}, error) {
		return []map[string]interface{}{{
			"id":         uuid.New().String(),
			"user_id":    "u1",
			"scope_type": "workspace",
		}}, nil
	}
	stubTable.DeleteRecordFn = func(tableName string, id interface{}) error {
		return nil
	}

	repo := &pkg.DatabaseService{TableService: stubTable}
	svc := services.NewAccessMemberService(repo)

	err := svc.BulkAssignRoleToUsers(context.Background(), "schema", dto.BulkAssignRoleRequest{
		UserIDs:   []string{"", "u1"},
		ScopeType: "workspace",
		RoleID:    uuid.New().String(),
	})
	assert.NoError(t, err)

	err = svc.BulkRemoveRoleFromUsers(context.Background(), "schema", []string{"", "u1"}, "workspace", nil, "role")
	assert.NoError(t, err)
}
