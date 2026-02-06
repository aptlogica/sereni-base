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

func TestNewAccessRoleService(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}

	service := services.NewAccessRoleService(repo)

	assert.NotNil(t, service)
}

func TestCreateAccessRole(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessRoleService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()
	description := "Admin role"

	req := dto.AccessRoleDTO{
		ID:          roleID,
		Name:        "Admin",
		ScopeLevel:  "workspace",
		Priority:    100,
		Description: &description,
		IsDefault:   true,
	}

	mockTable.On("CreateRecord", "\"test_schema\".access_roles", mock.Anything).
		Return(map[string]interface{}{
			"id":          roleID.String(),
			"name":        "Admin",
			"scope_level": "workspace",
			"priority":    100,
			"description": "Admin role",
			"is_default":  true,
		}, nil)

	result, err := service.CreateAccessRole(ctx, schemaName, req)

	assert.NoError(t, err)
	assert.Equal(t, roleID, result.ID)
	assert.Equal(t, "Admin", result.Name)
	mockTable.AssertExpectations(t)
}

func TestCreateAccessRole_WithNilID(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessRoleService(repo)

	ctx := context.Background()
	schemaName := "test_schema"

	req := dto.AccessRoleDTO{
		ID:         uuid.Nil,
		Name:       "Admin",
		ScopeLevel: "workspace",
		Priority:   100,
	}

	mockTable.On("CreateRecord", "\"test_schema\".access_roles", mock.Anything).
		Return(map[string]interface{}{
			"id":          uuid.New().String(),
			"name":        "Admin",
			"scope_level": "workspace",
			"priority":    100,
		}, nil)

	result, err := service.CreateAccessRole(ctx, schemaName, req)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, result.ID)
	mockTable.AssertExpectations(t)
}

func TestCreateAccessRole_DatabaseError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessRoleService(repo)

	ctx := context.Background()
	schemaName := "test_schema"

	req := dto.AccessRoleDTO{
		ID:         uuid.New(),
		Name:       "Admin",
		ScopeLevel: "workspace",
	}

	mockTable.On("CreateRecord", "\"test_schema\".access_roles", mock.Anything).
		Return(nil, errors.New("database error"))

	result, err := service.CreateAccessRole(ctx, schemaName, req)

	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, result.ID)
	mockTable.AssertExpectations(t)
}

func TestCreateAccessRole_MapStructError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessRoleService(repo)

	ctx := context.Background()
	schemaName := "test_schema"

	req := dto.AccessRoleDTO{
		ID:         uuid.New(),
		Name:       "Admin",
		ScopeLevel: "workspace",
	}

	mockTable.On("CreateRecord", "\"test_schema\".access_roles", mock.Anything).
		Return(map[string]interface{}{
			"id": "invalid-uuid",
		}, nil)

	result, err := service.CreateAccessRole(ctx, schemaName, req)

	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, result.ID)
	mockTable.AssertExpectations(t)
}

func TestGetAccessRoleByID(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessRoleService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".access_roles", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":          roleID.String(),
				"name":        "Admin",
				"scope_level": "workspace",
				"priority":    100,
			},
		}, nil)

	result, err := service.GetAccessRoleByID(ctx, schemaName, roleID)

	assert.NoError(t, err)
	assert.Equal(t, roleID, result.ID)
	assert.Equal(t, "Admin", result.Name)
	mockTable.AssertExpectations(t)
}

func TestGetAccessRoleByID_NotFound(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessRoleService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".access_roles", mock.Anything).
		Return([]map[string]interface{}{}, nil)

	result, err := service.GetAccessRoleByID(ctx, schemaName, roleID)

	assert.Error(t, err)
	assert.Equal(t, app_errors.RoleNotFound, err)
	assert.Equal(t, uuid.Nil, result.ID)
	mockTable.AssertExpectations(t)
}

func TestGetAccessRoleByID_DatabaseError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessRoleService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".access_roles", mock.Anything).
		Return(nil, errors.New("database error"))

	result, err := service.GetAccessRoleByID(ctx, schemaName, roleID)

	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, result.ID)
	mockTable.AssertExpectations(t)
}

func TestGetAccessRoleByName(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessRoleService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()
	roleName := "Admin"

	mockTable.On("GetTableData", "\"test_schema\".access_roles", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":          roleID.String(),
				"name":        roleName,
				"scope_level": "workspace",
				"priority":    100,
			},
		}, nil)

	result, err := service.GetAccessRoleByName(ctx, schemaName, roleName)

	assert.NoError(t, err)
	assert.Equal(t, roleID, result.ID)
	assert.Equal(t, roleName, result.Name)
	mockTable.AssertExpectations(t)
}

func TestGetAccessRoleByName_NotFound(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessRoleService(repo)

	ctx := context.Background()
	schemaName := "test_schema"

	mockTable.On("GetTableData", "\"test_schema\".access_roles", mock.Anything).
		Return([]map[string]interface{}{}, nil)

	result, err := service.GetAccessRoleByName(ctx, schemaName, "NonExistent")

	assert.Error(t, err)
	assert.Equal(t, app_errors.RoleNotFound, err)
	assert.Equal(t, uuid.Nil, result.ID)
	mockTable.AssertExpectations(t)
}

func TestGetAccessRolesByScope(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessRoleService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	scopeLevel := "workspace"

	mockTable.On("GetTableData", "\"test_schema\".access_roles", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":          uuid.New().String(),
				"name":        "Admin",
				"scope_level": scopeLevel,
				"priority":    100,
			},
			{
				"id":          uuid.New().String(),
				"name":        "Member",
				"scope_level": scopeLevel,
				"priority":    50,
			},
		}, nil)

	result, err := service.GetAccessRolesByScope(ctx, schemaName, scopeLevel)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	mockTable.AssertExpectations(t)
}

func TestGetAccessRolesByScope_DatabaseError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessRoleService(repo)

	ctx := context.Background()
	schemaName := "test_schema"

	mockTable.On("GetTableData", "\"test_schema\".access_roles", mock.Anything).
		Return(nil, errors.New("database error"))

	result, err := service.GetAccessRolesByScope(ctx, schemaName, "workspace")

	assert.Error(t, err)
	assert.Nil(t, result)
	mockTable.AssertExpectations(t)
}

func TestGetAccessRolesByScope_MapStructError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessRoleService(repo)

	ctx := context.Background()
	schemaName := "test_schema"

	mockTable.On("GetTableData", "\"test_schema\".access_roles", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id": "invalid-uuid",
			},
		}, nil)

	result, err := service.GetAccessRolesByScope(ctx, schemaName, "workspace")

	assert.Error(t, err)
	assert.Nil(t, result)
	mockTable.AssertExpectations(t)
}

func TestListAccessRoles(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessRoleService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	limit := 10
	offset := 0

	mockTable.On("GetTableData", "\"test_schema\".access_roles", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":          uuid.New().String(),
				"name":        "Admin",
				"scope_level": "workspace",
				"priority":    100,
			},
		}, nil).Once()

	mockTable.On("GetTableData", "\"test_schema\".access_roles", mock.Anything).
		Return([]map[string]interface{}{
			{"total": float64(5)},
		}, nil).Once()

	result, count, err := service.ListAccessRoles(ctx, schemaName, limit, offset)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, int64(5), count)
	mockTable.AssertExpectations(t)
}

func TestListAccessRoles_DatabaseError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessRoleService(repo)

	ctx := context.Background()
	schemaName := "test_schema"

	mockTable.On("GetTableData", "\"test_schema\".access_roles", mock.Anything).
		Return(nil, errors.New("database error"))

	result, count, err := service.ListAccessRoles(ctx, schemaName, 10, 0)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, int64(0), count)
	mockTable.AssertExpectations(t)
}

func TestListAccessRoles_CountError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessRoleService(repo)

	ctx := context.Background()
	schemaName := "test_schema"

	mockTable.On("GetTableData", "\"test_schema\".access_roles", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":          uuid.New().String(),
				"name":        "Admin",
				"scope_level": "workspace",
				"priority":    100,
			},
		}, nil).Once()

	mockTable.On("GetTableData", "\"test_schema\".access_roles", mock.Anything).
		Return(nil, errors.New("count error")).Once()

	result, count, err := service.ListAccessRoles(ctx, schemaName, 10, 0)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, int64(0), count)
	mockTable.AssertExpectations(t)
}

func TestListAccessRoles_MapStructError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessRoleService(repo)

	ctx := context.Background()
	schemaName := "test_schema"

	mockTable.On("GetTableData", "\"test_schema\".access_roles", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id": "invalid-uuid",
			},
		}, nil).Once()

	mockTable.On("GetTableData", "\"test_schema\".access_roles", mock.Anything).
		Return([]map[string]interface{}{
			{"total": float64(1)},
		}, nil).Once()

	result, count, err := service.ListAccessRoles(ctx, schemaName, 10, 0)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, int64(0), count)
	mockTable.AssertExpectations(t)
}

func TestUpdateAccessRole(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessRoleService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()
	description := "Updated description"

	req := dto.AccessRoleDTO{
		Name:        "Updated Admin",
		ScopeLevel:  "workspace",
		Priority:    150,
		Description: &description,
	}

	mockTable.On("UpdateRecord", "\"test_schema\".access_roles", roleID, mock.Anything).
		Return(map[string]interface{}{
			"id":          roleID.String(),
			"name":        "Updated Admin",
			"scope_level": "workspace",
			"priority":    150,
			"description": "Updated description",
		}, nil)

	result, err := service.UpdateAccessRole(ctx, schemaName, roleID, req)

	assert.NoError(t, err)
	assert.Equal(t, "Updated Admin", result.Name)
	assert.Equal(t, 150, result.Priority)
	mockTable.AssertExpectations(t)
}

func TestUpdateAccessRole_DatabaseError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessRoleService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()

	req := dto.AccessRoleDTO{
		Name: "Updated Admin",
	}

	mockTable.On("UpdateRecord", "\"test_schema\".access_roles", roleID, mock.Anything).
		Return(nil, errors.New("database error"))

	result, err := service.UpdateAccessRole(ctx, schemaName, roleID, req)

	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, result.ID)
	mockTable.AssertExpectations(t)
}

func TestUpdateAccessRole_MapStructError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessRoleService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()

	req := dto.AccessRoleDTO{
		Name: "Updated Admin",
	}

	mockTable.On("UpdateRecord", "\"test_schema\".access_roles", roleID, mock.Anything).
		Return(map[string]interface{}{
			"id": "invalid-uuid",
		}, nil)

	result, err := service.UpdateAccessRole(ctx, schemaName, roleID, req)

	assert.Error(t, err)
	assert.Equal(t, app_errors.ErrMapToStruct, err)
	assert.Equal(t, uuid.Nil, result.ID)
	mockTable.AssertExpectations(t)
}

func TestDeleteAccessRole(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessRoleService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()

	mockTable.On("DeleteRecord", "\"test_schema\".access_roles", mock.Anything).
		Return(nil)

	err := service.DeleteAccessRole(ctx, schemaName, roleID)

	assert.NoError(t, err)
	mockTable.AssertExpectations(t)
}

func TestDeleteAccessRole_DatabaseError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessRoleService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()

	mockTable.On("DeleteRecord", "\"test_schema\".access_roles", mock.Anything).
		Return(errors.New("database error"))

	err := service.DeleteAccessRole(ctx, schemaName, roleID)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}
