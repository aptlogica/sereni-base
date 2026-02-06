package rbac_test

import (
	"context"
	"errors"
	"testing"

	"go-postgres-rest/pkg"
	app_errors "serenibase/internal/app-errors"
	"serenibase/internal/dto"
	services "serenibase/internal/services/rbac"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewRolePermissionService(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}

	service := services.NewRolePermissionService(repo)

	assert.NotNil(t, service)
}

func TestRolePerm_AssignPermissionToRole(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewRolePermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	rolePermID := uuid.New()
	roleID := uuid.New()
	permissionID := uuid.New()

	req := dto.RolePermissionDTO{
		ID:           rolePermID,
		RoleID:       roleID,
		PermissionID: permissionID,
	}

	mockTable.On("CreateRecord", "\"test_schema\".role_permissions", mock.Anything).
		Return(map[string]interface{}{
			"id":            rolePermID.String(),
			"role_id":       roleID.String(),
			"permission_id": permissionID.String(),
		}, nil)

	result, err := service.AssignPermissionToRole(ctx, schemaName, req)

	assert.NoError(t, err)
	assert.Equal(t, rolePermID, result.ID)
	assert.Equal(t, roleID, result.RoleID)
	assert.Equal(t, permissionID, result.PermissionID)
	mockTable.AssertExpectations(t)
}

func TestRolePerm_AssignPermissionToRole_WithNilID(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewRolePermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()
	permissionID := uuid.New()

	req := dto.RolePermissionDTO{
		ID:           uuid.Nil,
		RoleID:       roleID,
		PermissionID: permissionID,
	}

	mockTable.On("CreateRecord", "\"test_schema\".role_permissions", mock.Anything).
		Return(map[string]interface{}{
			"id":            uuid.New().String(),
			"role_id":       roleID.String(),
			"permission_id": permissionID.String(),
		}, nil)

	result, err := service.AssignPermissionToRole(ctx, schemaName, req)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, result.ID)
	mockTable.AssertExpectations(t)
}

func TestRolePerm_AssignPermissionToRole_DatabaseError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewRolePermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"

	req := dto.RolePermissionDTO{
		ID:           uuid.New(),
		RoleID:       uuid.New(),
		PermissionID: uuid.New(),
	}

	mockTable.On("CreateRecord", "\"test_schema\".role_permissions", mock.Anything).
		Return(nil, errors.New("database error"))

	result, err := service.AssignPermissionToRole(ctx, schemaName, req)

	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, result.ID)
	mockTable.AssertExpectations(t)
}

func TestRolePerm_AssignPermissionToRole_MapStructError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewRolePermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"

	req := dto.RolePermissionDTO{
		ID:           uuid.New(),
		RoleID:       uuid.New(),
		PermissionID: uuid.New(),
	}

	mockTable.On("CreateRecord", "\"test_schema\".role_permissions", mock.Anything).
		Return(map[string]interface{}{
			"id": "invalid-uuid",
		}, nil)

	result, err := service.AssignPermissionToRole(ctx, schemaName, req)

	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, result.ID)
	mockTable.AssertExpectations(t)
}

func TestRemovePermissionFromRole(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewRolePermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()
	permissionID := uuid.New()
	rolePermID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".role_permissions", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":            rolePermID.String(),
				"role_id":       roleID.String(),
				"permission_id": permissionID.String(),
			},
		}, nil)

	mockTable.On("DeleteRecord", "\"test_schema\".role_permissions", mock.Anything).
		Return(nil)

	err := service.RemovePermissionFromRole(ctx, schemaName, roleID, permissionID)

	assert.NoError(t, err)
	mockTable.AssertExpectations(t)
}

func TestRemovePermissionFromRole_NotFound(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewRolePermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()
	permissionID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".role_permissions", mock.Anything).
		Return([]map[string]interface{}{}, nil)

	err := service.RemovePermissionFromRole(ctx, schemaName, roleID, permissionID)

	assert.Error(t, err)
	assert.Equal(t, app_errors.ErrRecordNotFound, err)
	mockTable.AssertExpectations(t)
}

func TestRemovePermissionFromRole_GetDataError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewRolePermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()
	permissionID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".role_permissions", mock.Anything).
		Return(nil, errors.New("database error"))

	err := service.RemovePermissionFromRole(ctx, schemaName, roleID, permissionID)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}

func TestRemovePermissionFromRole_DeleteError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewRolePermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()
	permissionID := uuid.New()
	rolePermID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".role_permissions", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":            rolePermID.String(),
				"role_id":       roleID.String(),
				"permission_id": permissionID.String(),
			},
		}, nil)

	mockTable.On("DeleteRecord", "\"test_schema\".role_permissions", mock.Anything).
		Return(errors.New("delete error"))

	err := service.RemovePermissionFromRole(ctx, schemaName, roleID, permissionID)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}

func TestGetRolePermissions(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewRolePermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".role_permissions", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":            uuid.New().String(),
				"role_id":       roleID.String(),
				"permission_id": uuid.New().String(),
			},
			{
				"id":            uuid.New().String(),
				"role_id":       roleID.String(),
				"permission_id": uuid.New().String(),
			},
		}, nil)

	result, err := service.GetRolePermissions(ctx, schemaName, roleID)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	mockTable.AssertExpectations(t)
}

func TestGetRolePermissions_DatabaseError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewRolePermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".role_permissions", mock.Anything).
		Return(nil, errors.New("database error"))

	result, err := service.GetRolePermissions(ctx, schemaName, roleID)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockTable.AssertExpectations(t)
}

func TestGetRolePermissions_MapStructError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewRolePermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".role_permissions", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id": "invalid-uuid",
			},
		}, nil)

	result, err := service.GetRolePermissions(ctx, schemaName, roleID)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockTable.AssertExpectations(t)
}

func TestGetPermissionsByRole(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewRolePermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()
	permissionID := uuid.New()
	resourceID := uuid.New()
	actionID := uuid.New()

	// Mock GetRolePermissions
	mockTable.On("GetTableData", "\"test_schema\".role_permissions", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":            uuid.New().String(),
				"role_id":       roleID.String(),
				"permission_id": permissionID.String(),
			},
		}, nil).Once()

	// Mock GetPermissionByID
	mockTable.On("GetTableData", "\"test_schema\".permissions", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":          permissionID.String(),
				"resource_id": resourceID.String(),
				"action_id":   actionID.String(),
			},
		}, nil).Once()

	// Mock GetResourceByID
	mockTable.On("GetTableData", "\"test_schema\".resources", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":   resourceID.String(),
				"code": "workspace",
			},
		}, nil).Once()

	// Mock GetActionByID
	mockTable.On("GetTableData", "\"test_schema\".actions", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":   actionID.String(),
				"code": "read",
			},
		}, nil).Once()

	result, err := service.GetPermissionsByRole(ctx, schemaName, roleID)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "workspace", result[0].ResourceCode)
	assert.Equal(t, "read", result[0].ActionCode)
	mockTable.AssertExpectations(t)
}

func TestGetPermissionsByRole_GetRolePermissionsError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewRolePermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".role_permissions", mock.Anything).
		Return(nil, errors.New("database error"))

	result, err := service.GetPermissionsByRole(ctx, schemaName, roleID)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockTable.AssertExpectations(t)
}

func TestGetPermissionsByRole_GetPermissionError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewRolePermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()
	permissionID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".role_permissions", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":            uuid.New().String(),
				"role_id":       roleID.String(),
				"permission_id": permissionID.String(),
			},
		}, nil).Once()

	mockTable.On("GetTableData", "\"test_schema\".permissions", mock.Anything).
		Return(nil, errors.New("permission not found"))

	result, err := service.GetPermissionsByRole(ctx, schemaName, roleID)

	assert.NoError(t, err)
	assert.Len(t, result, 0)
	mockTable.AssertExpectations(t)
}

func TestGetRolesByPermission(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewRolePermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	permissionID := uuid.New()
	roleID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".role_permissions", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":            uuid.New().String(),
				"role_id":       roleID.String(),
				"permission_id": permissionID.String(),
			},
		}, nil).Once()

	mockTable.On("GetTableData", "\"test_schema\".access_roles", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":          roleID.String(),
				"name":        "Admin",
				"scope_level": "workspace",
				"priority":    100,
			},
		}, nil).Once()

	result, err := service.GetRolesByPermission(ctx, schemaName, permissionID)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, roleID, result[0].ID)
	mockTable.AssertExpectations(t)
}

func TestGetRolesByPermission_GetDataError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewRolePermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	permissionID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".role_permissions", mock.Anything).
		Return(nil, errors.New("database error"))

	result, err := service.GetRolesByPermission(ctx, schemaName, permissionID)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockTable.AssertExpectations(t)
}

func TestGetRolesByPermission_MapStructError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewRolePermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	permissionID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".role_permissions", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id": "invalid-uuid",
			},
		}, nil)

	result, err := service.GetRolesByPermission(ctx, schemaName, permissionID)

	assert.NoError(t, err)
	assert.Len(t, result, 0)
	mockTable.AssertExpectations(t)
}

func TestGetRolesByPermission_GetRoleError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewRolePermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	permissionID := uuid.New()
	roleID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".role_permissions", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":            uuid.New().String(),
				"role_id":       roleID.String(),
				"permission_id": permissionID.String(),
			},
		}, nil).Once()

	mockTable.On("GetTableData", "\"test_schema\".access_roles", mock.Anything).
		Return(nil, errors.New("role not found"))

	result, err := service.GetRolesByPermission(ctx, schemaName, permissionID)

	assert.NoError(t, err)
	assert.Len(t, result, 0)
	mockTable.AssertExpectations(t)
}

func TestCheckRoleHasPermission(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewRolePermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()
	permissionID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".role_permissions", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":            uuid.New().String(),
				"role_id":       roleID.String(),
				"permission_id": permissionID.String(),
			},
		}, nil)

	result, err := service.CheckRoleHasPermission(ctx, schemaName, roleID, permissionID)

	assert.NoError(t, err)
	assert.True(t, result)
	mockTable.AssertExpectations(t)
}

func TestCheckRoleHasPermission_NotFound(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewRolePermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()
	permissionID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".role_permissions", mock.Anything).
		Return([]map[string]interface{}{}, nil)

	result, err := service.CheckRoleHasPermission(ctx, schemaName, roleID, permissionID)

	assert.NoError(t, err)
	assert.False(t, result)
	mockTable.AssertExpectations(t)
}

func TestCheckRoleHasPermission_DatabaseError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewRolePermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()
	permissionID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".role_permissions", mock.Anything).
		Return(nil, errors.New("database error"))

	result, err := service.CheckRoleHasPermission(ctx, schemaName, roleID, permissionID)

	assert.Error(t, err)
	assert.False(t, result)
	mockTable.AssertExpectations(t)
}
