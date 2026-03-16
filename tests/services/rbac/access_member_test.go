package rbac_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aptlogica/go-postgres-rest/pkg"
	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	services "github.com/aptlogica/sereni-base/internal/services/rbac"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTableService is a mock implementation of TableService
type MockTableService struct {
	mock.Mock
}

func (m *MockTableService) GetTableData(tableName string, params dbModels.QueryParams) ([]map[string]interface{}, error) {
	args := m.Called(tableName, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *MockTableService) CreateRecord(tableName string, data map[string]interface{}) (map[string]interface{}, error) {
	args := m.Called(tableName, data)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockTableService) UpdateRecord(tableName string, id interface{}, data map[string]interface{}) (map[string]interface{}, error) {
	args := m.Called(tableName, id, data)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockTableService) DeleteRecord(tableName string, id interface{}) error {
	args := m.Called(tableName, id)
	return args.Error(0)
}

func (m *MockTableService) GetTables(schema string) ([]dbModels.Table, error) {
	args := m.Called(schema)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dbModels.Table), args.Error(1)
}

func (m *MockTableService) CreateTable(req dbModels.CreateTableRequest) error {
	args := m.Called(req)
	return args.Error(0)
}

func (m *MockTableService) AddColumn(tableName string, req dbModels.AddColumnRequest) error {
	args := m.Called(tableName, req)
	return args.Error(0)
}

func (m *MockTableService) AlterTable(tableName string, req dbModels.AlterTableRequest) error {
	args := m.Called(tableName, req)
	return args.Error(0)
}

func (m *MockTableService) BuildComplexQuery(tableName string, filters map[string]interface{}) (dbModels.QueryParams, error) {
	args := m.Called(tableName, filters)
	return args.Get(0).(dbModels.QueryParams), args.Error(1)
}

func (m *MockTableService) CreateSchema(ctx context.Context, schemaName string) error {
	args := m.Called(ctx, schemaName)
	return args.Error(0)
}

func (m *MockTableService) DropTable(ctx context.Context, tableName string) error {
	args := m.Called(ctx, tableName)
	return args.Error(0)
}

func (m *MockTableService) CreateView(ctx context.Context, viewName string, viewSQL string) error {
	args := m.Called(ctx, viewName, viewSQL)
	return args.Error(0)
}

func (m *MockTableService) CreateFunction(ctx context.Context, functionName string, functionSQL string) error {
	args := m.Called(ctx, functionName, functionSQL)
	return args.Error(0)
}

func (m *MockTableService) GetByFunction(ctx context.Context, functionName string, params map[string]interface{}) ([]map[string]interface{}, error) {
	args := m.Called(ctx, functionName, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func TestNewAccessMemberService(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}

	service := services.NewAccessMemberService(repo)

	assert.NotNil(t, service)
}

func TestAccessMember_AssignRoleToUser(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	memberID := uuid.New()
	userID := "user-123"
	roleID := "role-456"
	scopeID := "scope-789"

	req := dto.AccessMemberDTO{
		ID:         memberID,
		UserID:     userID,
		RoleID:     roleID,
		ScopeType:  "workspace",
		ScopeID:    &scopeID,
		AssignedBy: nil,
	}

	mockTable.On("CreateRecord", "\"test_schema\".access_members", mock.Anything).
		Return(map[string]interface{}{
			"id":          memberID.String(),
			"user_id":     userID,
			"role_id":     roleID,
			"scope_type":  "workspace",
			"scope_id":    "scope-789",
			"assigned_by": nil,
		}, nil)

	result, err := service.AssignRoleToUser(ctx, schemaName, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	mockTable.AssertExpectations(t)
}

func TestAccessMember_AssignRoleToUser_WithNilID(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	userID := "user-123"
	roleID := "role-456"
	scopeID := "scope-789"
	assignedBy := "admin"

	req := dto.AccessMemberDTO{
		ID:         uuid.Nil,
		UserID:     userID,
		RoleID:     roleID,
		ScopeType:  "workspace",
		ScopeID:    &scopeID,
		AssignedBy: &assignedBy,
	}

	mockTable.On("CreateRecord", "\"test_schema\".access_members", mock.Anything).
		Return(map[string]interface{}{
			"id":          uuid.New().String(),
			"user_id":     userID,
			"role_id":     roleID,
			"scope_type":  "workspace",
			"scope_id":    "scope-789",
			"assigned_by": "admin",
		}, nil)

	result, err := service.AssignRoleToUser(ctx, schemaName, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	mockTable.AssertExpectations(t)
}

func TestAccessMember_AssignRoleToUser_DatabaseError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	assignedBy := "admin"
	req := dto.AccessMemberDTO{
		ID:         uuid.New(),
		UserID:     "user-123",
		RoleID:     "role-456",
		ScopeType:  "workspace",
		AssignedBy: &assignedBy,
	}

	mockTable.On("CreateRecord", "\"test_schema\".access_members", mock.Anything).
		Return(nil, errors.New("database error"))

	result, err := service.AssignRoleToUser(ctx, schemaName, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockTable.AssertExpectations(t)
}

func TestAccessMember_RemoveRoleFromUser(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	userID := "user-123"
	scopeID := "scope-789"
	scopeType := "workspace"
	memberID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".access_members", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":         memberID.String(),
				"user_id":    userID,
				"scope_type": scopeType,
				"scope_id":   scopeID,
			},
		}, nil)

	mockTable.On("DeleteRecord", "\"test_schema\".access_members", mock.Anything).
		Return(nil)

	err := service.RemoveRoleFromUser(ctx, schemaName, userID, scopeID, scopeType)

	assert.NoError(t, err)
	mockTable.AssertExpectations(t)
}

func TestRemoveRoleFromUser_EmptyUserID(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	err := service.RemoveRoleFromUser(ctx, "test_schema", "", "scope-789", "workspace")

	assert.Error(t, err)
	assert.Equal(t, app_errors.ErrRecordNotFound, err)
}

func TestRemoveRoleFromUser_EmptyScopeType(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	err := service.RemoveRoleFromUser(ctx, "test_schema", "user-123", "scope-789", "")

	assert.Error(t, err)
	assert.Equal(t, app_errors.InvalidScopeType, err)
}

func TestRemoveRoleFromUser_NotFound(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	schemaName := "test_schema"
	userID := "user-123"
	scopeID := "scope-789"
	scopeType := "workspace"

	mockTable.On("GetTableData", "\"test_schema\".access_members", mock.Anything).
		Return([]map[string]interface{}{}, nil)

	err := service.RemoveRoleFromUser(ctx, schemaName, userID, scopeID, scopeType)

	assert.Error(t, err)
	assert.Equal(t, app_errors.AccessMemberNotFound, err)
	mockTable.AssertExpectations(t)
}

func TestRemoveRoleFromUser_GetDataError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	mockTable.On("GetTableData", "\"test_schema\".access_members", mock.Anything).
		Return(nil, errors.New("database error"))

	err := service.RemoveRoleFromUser(ctx, "test_schema", "user-123", "scope-789", "workspace")

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}

func TestRemoveRoleFromUser_DeleteError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	memberID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".access_members", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":         memberID.String(),
				"user_id":    "user-123",
				"scope_type": "workspace",
				"scope_id":   "scope-789",
			},
		}, nil)

	mockTable.On("DeleteRecord", "\"test_schema\".access_members", mock.Anything).
		Return(errors.New("delete error"))

	err := service.RemoveRoleFromUser(ctx, "test_schema", "user-123", "scope-789", "workspace")

	assert.Error(t, err)
	assert.Equal(t, app_errors.AccessMemberDeleteFailed, err)
	mockTable.AssertExpectations(t)
}

func TestRemoveAccessMemberByID(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	memberID := "member-123"

	mockTable.On("DeleteRecord", "\"test_schema\".access_members", memberID).
		Return(nil)

	err := service.RemoveAccessMemberByID(ctx, "test_schema", memberID)

	assert.NoError(t, err)
	mockTable.AssertExpectations(t)
}

func TestRemoveAccessMemberByID_EmptyID(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	err := service.RemoveAccessMemberByID(ctx, "test_schema", "")

	assert.Error(t, err)
	assert.Equal(t, app_errors.ErrRecordNotFound, err)
}

func TestRemoveAccessMemberByID_DeleteError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	memberID := "member-123"

	mockTable.On("DeleteRecord", "\"test_schema\".access_members", memberID).
		Return(errors.New("delete error"))

	err := service.RemoveAccessMemberByID(ctx, "test_schema", memberID)

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}

func TestGetUserAccessMembers(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	userID := "user-123"
	memberID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".access_members", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":         memberID.String(),
				"user_id":    userID,
				"role_id":    "role-456",
				"scope_type": "workspace",
				"scope_id":   "scope-789",
			},
		}, nil)

	result, err := service.GetUserAccessMembers(ctx, "test_schema", userID)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	mockTable.AssertExpectations(t)
}

func TestGetUserAccessMembers_DatabaseError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	mockTable.On("GetTableData", "\"test_schema\".access_members", mock.Anything).
		Return(nil, errors.New("database error"))

	result, err := service.GetUserAccessMembers(ctx, "test_schema", "user-123")

	assert.Error(t, err)
	assert.Nil(t, result)
	mockTable.AssertExpectations(t)
}

func TestGetUserAccessByScope(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	userID := "user-123"
	scopeType := "workspace"
	scopeID := "scope-789"

	mockTable.On("GetTableData", "\"test_schema\".access_members", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":         uuid.New().String(),
				"user_id":    userID,
				"role_id":    "role-456",
				"scope_type": scopeType,
				"scope_id":   scopeID,
			},
		}, nil)

	result, err := service.GetUserAccessByScope(ctx, "test_schema", userID, scopeType, &scopeID)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	mockTable.AssertExpectations(t)
}

func TestGetUserAccessByScope_EmptyUserID(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	scopeType := "workspace"

	result, err := service.GetUserAccessByScope(ctx, "test_schema", "", scopeType, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, app_errors.ErrRecordNotFound, err)
}

func TestGetUserAccessByScope_EmptyScopeType(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()

	result, err := service.GetUserAccessByScope(ctx, "test_schema", "user-123", "", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, app_errors.InvalidScopeType, err)
}

func TestGetUserAccessByScope_WithoutScopeID(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	userID := "user-123"
	scopeType := "workspace"

	mockTable.On("GetTableData", "\"test_schema\".access_members", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":         uuid.New().String(),
				"user_id":    userID,
				"role_id":    "role-456",
				"scope_type": scopeType,
			},
		}, nil)

	result, err := service.GetUserAccessByScope(ctx, "test_schema", userID, scopeType, nil)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	mockTable.AssertExpectations(t)
}

func TestGetScopeMembers(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	scopeType := "workspace"
	scopeID := "scope-789"

	mockTable.On("GetTableData", "\"test_schema\".access_members", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":         uuid.New().String(),
				"user_id":    "user-123",
				"role_id":    "role-456",
				"scope_type": scopeType,
				"scope_id":   scopeID,
			},
		}, nil)

	result, err := service.GetScopeMembers(ctx, "test_schema", scopeType, &scopeID)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	mockTable.AssertExpectations(t)
}

func TestGetScopeMembers_DatabaseError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	scopeType := "workspace"

	mockTable.On("GetTableData", "\"test_schema\".access_members", mock.Anything).
		Return(nil, errors.New("database error"))

	result, err := service.GetScopeMembers(ctx, "test_schema", scopeType, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockTable.AssertExpectations(t)
}

func TestAccessMember_CheckUserPermission(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	userID := "user-123"
	scopeType := "workspace"
	resourceCode := "workspace"
	actionCode := "read"

	// Mock GetUserAccessByScope
	mockTable.On("GetTableData", "\"test_schema\".access_members", mock.Anything).
		Return([]map[string]interface{}{}, nil).Once()

	hasPermission, err := service.CheckUserPermission(ctx, "test_schema", userID, scopeType, nil, resourceCode, actionCode)

	assert.NoError(t, err)
	assert.False(t, hasPermission)
	mockTable.AssertExpectations(t)
}

func TestAccessMember_BulkAssignRoleToUsers(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	scopeID := "scope-789"
	assignedBy := "admin"
	req := dto.BulkAssignRoleRequest{
		UserIDs:    []string{"user-1", "user-2"},
		RoleID:     "role-123",
		ScopeType:  "workspace",
		ScopeID:    &scopeID,
		AssignedBy: &assignedBy,
	}

	mockTable.On("CreateRecord", "\"test_schema\".access_members", mock.Anything).
		Return(map[string]interface{}{
			"id":          uuid.New().String(),
			"user_id":     mock.Anything,
			"role_id":     "role-123",
			"scope_type":  "workspace",
			"scope_id":    "scope-789",
			"assigned_by": "admin",
		}, nil).Times(2)

	err := service.BulkAssignRoleToUsers(ctx, "test_schema", req)

	assert.NoError(t, err)
	mockTable.AssertExpectations(t)
}

func TestBulkAssignRoleToUsers_EmptyList(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	req := dto.BulkAssignRoleRequest{
		UserIDs:   []string{},
		RoleID:    "role-123",
		ScopeType: "workspace",
	}

	err := service.BulkAssignRoleToUsers(ctx, "test_schema", req)

	assert.Error(t, err)
	assert.Equal(t, app_errors.EmptyUserList, err)
}

func TestBulkAssignRoleToUsers_AllFailed(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	req := dto.BulkAssignRoleRequest{
		UserIDs:   []string{"user-1", "user-2"},
		RoleID:    "role-123",
		ScopeType: "workspace",
	}

	mockTable.On("CreateRecord", "\"test_schema\".access_members", mock.Anything).
		Return(nil, errors.New("database error")).Times(2)

	err := service.BulkAssignRoleToUsers(ctx, "test_schema", req)

	assert.Error(t, err)
	assert.Equal(t, app_errors.BulkAssignmentFailed, err)
	mockTable.AssertExpectations(t)
}

func TestBulkRemoveRoleFromUsers(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	userIDs := []string{"user-1", "user-2"}
	scopeType := "workspace"
	scopeID := "scope-789"
	roleID := "role-123"

	mockTable.On("GetTableData", "\"test_schema\".access_members", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":         uuid.New().String(),
				"user_id":    mock.Anything,
				"scope_type": scopeType,
				"scope_id":   scopeID,
			},
		}, nil).Times(2)

	mockTable.On("DeleteRecord", "\"test_schema\".access_members", mock.Anything).
		Return(nil).Times(2)

	err := service.BulkRemoveRoleFromUsers(ctx, "test_schema", userIDs, scopeType, &scopeID, roleID)

	assert.NoError(t, err)
	mockTable.AssertExpectations(t)
}

func TestBulkRemoveRoleFromUsers_EmptyList(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	err := service.BulkRemoveRoleFromUsers(ctx, "test_schema", []string{}, "workspace", nil, "role-123")

	assert.Error(t, err)
	assert.Equal(t, app_errors.EmptyUserList, err)
}

func TestBulkRemoveRoleFromUsers_EmptyScopeType(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	err := service.BulkRemoveRoleFromUsers(ctx, "test_schema", []string{"user-1"}, "", nil, "role-123")

	assert.Error(t, err)
	assert.Equal(t, app_errors.InvalidScopeType, err)
}

func TestUpdateRoleForUser(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	userID := "user-123"
	scopeType := "workspace"
	scopeID := "scope-789"
	newRoleID := "new-role-456"
	memberID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".access_members", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":         memberID.String(),
				"user_id":    userID,
				"scope_type": scopeType,
				"scope_id":   scopeID,
				"role_id":    "old-role-123",
			},
		}, nil)

	mockTable.On("UpdateRecord", "\"test_schema\".access_members", memberID.String(), mock.Anything).
		Return(map[string]interface{}{
			"id":         memberID.String(),
			"user_id":    userID,
			"scope_type": scopeType,
			"scope_id":   scopeID,
			"role_id":    newRoleID,
		}, nil)

	err := service.UpdateRoleForUser(ctx, "test_schema", userID, scopeType, &scopeID, newRoleID)

	assert.NoError(t, err)
	mockTable.AssertExpectations(t)
}

func TestUpdateRoleForUser_EmptyUserID(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	err := service.UpdateRoleForUser(ctx, "test_schema", "", "workspace", nil, "role-123")

	assert.Error(t, err)
	assert.Equal(t, app_errors.ErrRecordNotFound, err)
}

func TestUpdateRoleForUser_EmptyScopeType(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	err := service.UpdateRoleForUser(ctx, "test_schema", "user-123", "", nil, "role-123")

	assert.Error(t, err)
	assert.Equal(t, app_errors.InvalidScopeType, err)
}

func TestUpdateRoleForUser_NotFound(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	mockTable.On("GetTableData", "\"test_schema\".access_members", mock.Anything).
		Return([]map[string]interface{}{}, nil)

	err := service.UpdateRoleForUser(ctx, "test_schema", "user-123", "workspace", nil, "role-123")

	assert.Error(t, err)
	assert.Equal(t, app_errors.AccessMemberNotFound, err)
	mockTable.AssertExpectations(t)
}

func TestUpdateRoleForUser_GetDataError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	mockTable.On("GetTableData", "\"test_schema\".access_members", mock.Anything).
		Return(nil, errors.New("database error"))

	err := service.UpdateRoleForUser(ctx, "test_schema", "user-123", "workspace", nil, "role-123")

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}

func TestUpdateRoleForUser_UpdateError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	service := services.NewAccessMemberService(repo)

	ctx := context.Background()
	memberID := uuid.New()

	mockTable.On("GetTableData", "\"test_schema\".access_members", mock.Anything).
		Return([]map[string]interface{}{
			{
				"id":         memberID.String(),
				"user_id":    "user-123",
				"scope_type": "workspace",
				"role_id":    "old-role",
			},
		}, nil)

	mockTable.On("UpdateRecord", "\"test_schema\".access_members", memberID.String(), mock.Anything).
		Return(nil, errors.New("update error"))

	err := service.UpdateRoleForUser(ctx, "test_schema", "user-123", "workspace", nil, "new-role")

	assert.Error(t, err)
	mockTable.AssertExpectations(t)
}
