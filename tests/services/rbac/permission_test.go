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

func TestNewPermissionService(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}

	service := services.NewPermissionService(repo)

	assert.NotNil(t, service)
}

func TestPerm_CreatePermission(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewPermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	permissionID := uuid.New()
	resourceID := uuid.New()
	actionID := uuid.New()

	req := dto.PermissionDTO{
		ID:         permissionID,
		ResourceID: resourceID,
		ActionID:   actionID,
	}

	mockTable.On("CreateRecord", "\"test_schema\".permissions", mock.Anything).
		Return(map[string]interface{}{
			"id":          permissionID.String(),
			"resource_id": resourceID.String(),
			"action_id":   actionID.String(),
		}, nil)

	result, err := service.CreatePermission(ctx, schemaName, req)

	assert.NoError(t, err)
	assert.Equal(t, permissionID, result.ID)
	assert.Equal(t, resourceID, result.ResourceID)
	assert.Equal(t, actionID, result.ActionID)
	mockTable.AssertExpectations(t)
}

func TestPerm_CreatePermission_WithNilID(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewPermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	resourceID := uuid.New()
	actionID := uuid.New()

	req := dto.PermissionDTO{
		ID:         uuid.Nil,
		ResourceID: resourceID,
		ActionID:   actionID,
	}

	mockTable.On("CreateRecord", "\"test_schema\".permissions", mock.Anything).
		Return(map[string]interface{}{
			"id":          uuid.New().String(),
			"resource_id": resourceID.String(),
			"action_id":   actionID.String(),
		}, nil)

	result, err := service.CreatePermission(ctx, schemaName, req)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, result.ID)
	mockTable.AssertExpectations(t)
}

func TestPerm_CreatePermission_DatabaseError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewPermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"

	req := dto.PermissionDTO{
		ID:         uuid.New(),
		ResourceID: uuid.New(),
		ActionID:   uuid.New(),
	}

	mockTable.On("CreateRecord", "\"test_schema\".permissions", mock.Anything).
		Return(nil, errors.New("database error"))

	result, err := service.CreatePermission(ctx, schemaName, req)

	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, result.ID)
	mockTable.AssertExpectations(t)
}

func TestPerm_CreatePermission_MapStructError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewPermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"

	req := dto.PermissionDTO{
		ID:         uuid.New(),
		ResourceID: uuid.New(),
		ActionID:   uuid.New(),
	}

	mockTable.On("CreateRecord", "\"test_schema\".permissions", mock.Anything).
		Return(map[string]interface{}{
			"id": "invalid-uuid",
		}, nil)

	result, err := service.CreatePermission(ctx, schemaName, req)

	assert.Error(t, err)
	assert.Equal(t, app_errors.ErrMapToStruct, err)
	assert.Equal(t, uuid.Nil, result.ID)
	mockTable.AssertExpectations(t)
}

func TestGetPermissionByID(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewPermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	permissionID := uuid.New()
	resourceID := uuid.New()
	actionID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".permissions", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":          permissionID.String(),
				"resource_id": resourceID.String(),
				"action_id":   actionID.String(),
			},
		}, nil)

	result, err := service.GetPermissionByID(ctx, schemaName, permissionID)

	assert.NoError(t, err)
	assert.Equal(t, permissionID, result.ID)
	assert.Equal(t, resourceID, result.ResourceID)
	mockTable.AssertExpectations(t)
}

func TestGetPermissionByID_NotFound(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewPermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	permissionID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".permissions", mock.Anything).
		Return([]map[string]interface{}{}, nil)

	result, err := service.GetPermissionByID(ctx, schemaName, permissionID)

	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, result.ID)
	mockTable.AssertExpectations(t)
}

func TestGetPermissionByID_DatabaseError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewPermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	permissionID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".permissions", mock.Anything).
		Return(nil, errors.New("database error"))

	result, err := service.GetPermissionByID(ctx, schemaName, permissionID)

	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, result.ID)
	mockTable.AssertExpectations(t)
}

func TestGetPermissionByResourceAndAction(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewPermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	permissionID := uuid.New()
	resourceID := uuid.New()
	actionID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".permissions", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":          permissionID.String(),
				"resource_id": resourceID.String(),
				"action_id":   actionID.String(),
			},
		}, nil)

	result, err := service.GetPermissionByResourceAndAction(ctx, schemaName, resourceID, actionID)

	assert.NoError(t, err)
	assert.Equal(t, permissionID, result.ID)
	assert.Equal(t, resourceID, result.ResourceID)
	assert.Equal(t, actionID, result.ActionID)
	mockTable.AssertExpectations(t)
}

func TestGetPermissionByResourceAndAction_NotFound(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewPermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	resourceID := uuid.New()
	actionID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".permissions", mock.Anything).
		Return([]map[string]interface{}{}, nil)

	result, err := service.GetPermissionByResourceAndAction(ctx, schemaName, resourceID, actionID)

	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, result.ID)
	mockTable.AssertExpectations(t)
}

func TestGetPermissionByResourceAndAction_DatabaseError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewPermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	resourceID := uuid.New()
	actionID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".permissions", mock.Anything).
		Return(nil, errors.New("database error"))

	result, err := service.GetPermissionByResourceAndAction(ctx, schemaName, resourceID, actionID)

	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, result.ID)
	mockTable.AssertExpectations(t)
}

func TestListPermissions(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewPermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	limit := 10
	offset := 0

	mockTable.On("GetTableData", "\"test_schema\".permissions", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":          uuid.New().String(),
				"resource_id": uuid.New().String(),
				"action_id":   uuid.New().String(),
			},
		}, nil).Once()

	mockTable.On("GetTableData", "\"test_schema\".permissions", mock.Anything).
		Return([]map[string]interface{}{
			{"total": float64(5)},
		}, nil).Once()

	result, count, err := service.ListPermissions(ctx, schemaName, limit, offset)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, int64(5), count)
	mockTable.AssertExpectations(t)
}

func TestListPermissions_DatabaseError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewPermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"

	mockTable.On("GetTableData", "\"test_schema\".permissions", mock.Anything).
		Return(nil, errors.New("database error"))

	result, count, err := service.ListPermissions(ctx, schemaName, 10, 0)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, int64(0), count)
	mockTable.AssertExpectations(t)
}

func TestListPermissions_CountError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewPermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"

	mockTable.On("GetTableData", "\"test_schema\".permissions", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":          uuid.New().String(),
				"resource_id": uuid.New().String(),
				"action_id":   uuid.New().String(),
			},
		}, nil).Once()

	mockTable.On("GetTableData", "\"test_schema\".permissions", mock.Anything).
		Return(nil, errors.New("count error")).Once()

	result, count, err := service.ListPermissions(ctx, schemaName, 10, 0)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, int64(0), count)
	mockTable.AssertExpectations(t)
}

func TestDeletePermission(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewPermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	permissionID := uuid.New()

	mockTable.On("DeleteRecord", "\"test_schema\".permissions", mock.Anything).
		Return(nil)

	err := service.DeletePermission(ctx, schemaName, permissionID)

	assert.NoError(t, err)
	mockTable.AssertExpectations(t)
}

func TestDeletePermission_DatabaseError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewPermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	permissionID := uuid.New()

	mockTable.On("DeleteRecord", "\"test_schema\".permissions", mock.Anything).
		Return(errors.New("database error"))

	err := service.DeletePermission(ctx, schemaName, permissionID)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}

func TestGetOrCreatePermission_ExistingPermission(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewPermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	permissionID := uuid.New()
	resourceID := uuid.New()
	actionID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".permissions", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":          permissionID.String(),
				"resource_id": resourceID.String(),
				"action_id":   actionID.String(),
			},
		}, nil)

	result, err := service.GetOrCreatePermission(ctx, schemaName, resourceID, actionID)

	assert.NoError(t, err)
	assert.Equal(t, permissionID, result.ID)
	mockTable.AssertExpectations(t)
}

func TestGetOrCreatePermission_CreateNew(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewPermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	resourceID := uuid.New()
	actionID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".permissions", mock.Anything).
		Return([]map[string]interface{}{}, nil)

	mockTable.On("CreateRecord", "\"test_schema\".permissions", mock.Anything).
		Return(map[string]interface{}{
			"id":          uuid.New().String(),
			"resource_id": resourceID.String(),
			"action_id":   actionID.String(),
		}, nil)

	result, err := service.GetOrCreatePermission(ctx, schemaName, resourceID, actionID)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, result.ID)
	mockTable.AssertExpectations(t)
}

func TestGetOrCreatePermission_CreateError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewPermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	resourceID := uuid.New()
	actionID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".permissions", mock.Anything).
		Return([]map[string]interface{}{}, nil)

	mockTable.On("CreateRecord", "\"test_schema\".permissions", mock.Anything).
		Return(nil, errors.New("create error"))

	result, err := service.GetOrCreatePermission(ctx, schemaName, resourceID, actionID)

	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, result.ID)
	mockTable.AssertExpectations(t)
}

func TestGetPermissionsByResource(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewPermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	resourceID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".permissions", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":          uuid.New().String(),
				"resource_id": resourceID.String(),
				"action_id":   uuid.New().String(),
			},
			{
				"id":          uuid.New().String(),
				"resource_id": resourceID.String(),
				"action_id":   uuid.New().String(),
			},
		}, nil)

	result, err := service.GetPermissionsByResource(ctx, schemaName, resourceID)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	mockTable.AssertExpectations(t)
}

func TestGetPermissionsByResource_DatabaseError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewPermissionService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	resourceID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".permissions", mock.Anything).
		Return(nil, errors.New("database error"))

	result, err := service.GetPermissionsByResource(ctx, schemaName, resourceID)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockTable.AssertExpectations(t)
}
