package rbac_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aptlogica/go-postgres-rest/pkg"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	services "github.com/aptlogica/sereni-base/internal/services/rbac"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock service interfaces
type MockAccessRoleService struct {
	mock.Mock
}

func (m *MockAccessRoleService) CreateAccessRole(ctx context.Context, schemaName string, req dto.AccessRoleDTO) (tenant.AccessRole, error) {
	args := m.Called(ctx, schemaName, req)
	return args.Get(0).(tenant.AccessRole), args.Error(1)
}

func (m *MockAccessRoleService) GetAccessRoleByID(ctx context.Context, schemaName string, roleID uuid.UUID) (tenant.AccessRole, error) {
	args := m.Called(ctx, schemaName, roleID)
	return args.Get(0).(tenant.AccessRole), args.Error(1)
}

func (m *MockAccessRoleService) GetAccessRoleByName(ctx context.Context, schemaName string, name string) (tenant.AccessRole, error) {
	args := m.Called(ctx, schemaName, name)
	return args.Get(0).(tenant.AccessRole), args.Error(1)
}

func (m *MockAccessRoleService) GetAccessRolesByScope(ctx context.Context, schemaName string, scopeLevel string) ([]tenant.AccessRole, error) {
	args := m.Called(ctx, schemaName, scopeLevel)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]tenant.AccessRole), args.Error(1)
}

func (m *MockAccessRoleService) ListAccessRoles(ctx context.Context, schemaName string, limit, offset int) ([]tenant.AccessRole, int64, error) {
	args := m.Called(ctx, schemaName, limit, offset)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]tenant.AccessRole), args.Get(1).(int64), args.Error(2)
}

func (m *MockAccessRoleService) UpdateAccessRole(ctx context.Context, schemaName string, roleID uuid.UUID, req dto.AccessRoleDTO) (tenant.AccessRole, error) {
	args := m.Called(ctx, schemaName, roleID, req)
	return args.Get(0).(tenant.AccessRole), args.Error(1)
}

func (m *MockAccessRoleService) DeleteAccessRole(ctx context.Context, schemaName string, roleID uuid.UUID) error {
	args := m.Called(ctx, schemaName, roleID)
	return args.Error(0)
}

type MockResourceService struct {
	mock.Mock
}

func (m *MockResourceService) CreateResource(ctx context.Context, schemaName string, req dto.ResourceDTO) (tenant.Resource, error) {
	args := m.Called(ctx, schemaName, req)
	return args.Get(0).(tenant.Resource), args.Error(1)
}

func (m *MockResourceService) GetResourceByID(ctx context.Context, schemaName string, resourceID uuid.UUID) (tenant.Resource, error) {
	args := m.Called(ctx, schemaName, resourceID)
	return args.Get(0).(tenant.Resource), args.Error(1)
}

func (m *MockResourceService) GetResourceByCode(ctx context.Context, schemaName string, code string) (tenant.Resource, error) {
	args := m.Called(ctx, schemaName, code)
	return args.Get(0).(tenant.Resource), args.Error(1)
}

func (m *MockResourceService) GetOrCreateResource(ctx context.Context, schemaName string, code string, description *string) (tenant.Resource, error) {
	args := m.Called(ctx, schemaName, code, description)
	return args.Get(0).(tenant.Resource), args.Error(1)
}

func (m *MockResourceService) ListResources(ctx context.Context, schemaName string, limit, offset int) ([]tenant.Resource, int64, error) {
	args := m.Called(ctx, schemaName, limit, offset)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]tenant.Resource), args.Get(1).(int64), args.Error(2)
}

func (m *MockResourceService) UpdateResource(ctx context.Context, schemaName string, resourceID uuid.UUID, req dto.ResourceDTO) (tenant.Resource, error) {
	args := m.Called(ctx, schemaName, resourceID, req)
	return args.Get(0).(tenant.Resource), args.Error(1)
}

func (m *MockResourceService) DeleteResource(ctx context.Context, schemaName string, resourceID uuid.UUID) error {
	args := m.Called(ctx, schemaName, resourceID)
	return args.Error(0)
}

type MockActionService struct {
	mock.Mock
}

func (m *MockActionService) CreateAction(ctx context.Context, schemaName string, req dto.ActionDTO) (tenant.Action, error) {
	args := m.Called(ctx, schemaName, req)
	return args.Get(0).(tenant.Action), args.Error(1)
}

func (m *MockActionService) GetActionByID(ctx context.Context, schemaName string, actionID uuid.UUID) (tenant.Action, error) {
	args := m.Called(ctx, schemaName, actionID)
	return args.Get(0).(tenant.Action), args.Error(1)
}

func (m *MockActionService) GetActionByCode(ctx context.Context, schemaName string, code string) (tenant.Action, error) {
	args := m.Called(ctx, schemaName, code)
	return args.Get(0).(tenant.Action), args.Error(1)
}

func (m *MockActionService) GetOrCreateAction(ctx context.Context, schemaName string, code string, description *string) (tenant.Action, error) {
	args := m.Called(ctx, schemaName, code, description)
	return args.Get(0).(tenant.Action), args.Error(1)
}

func (m *MockActionService) ListActions(ctx context.Context, schemaName string, limit, offset int) ([]tenant.Action, int64, error) {
	args := m.Called(ctx, schemaName, limit, offset)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]tenant.Action), args.Get(1).(int64), args.Error(2)
}

func (m *MockActionService) UpdateAction(ctx context.Context, schemaName string, actionID uuid.UUID, req dto.ActionDTO) (tenant.Action, error) {
	args := m.Called(ctx, schemaName, actionID, req)
	return args.Get(0).(tenant.Action), args.Error(1)
}

func (m *MockActionService) DeleteAction(ctx context.Context, schemaName string, actionID uuid.UUID) error {
	args := m.Called(ctx, schemaName, actionID)
	return args.Error(0)
}

type MockPermissionService struct {
	mock.Mock
}

func (m *MockPermissionService) CreatePermission(ctx context.Context, schemaName string, req dto.PermissionDTO) (tenant.Permission, error) {
	args := m.Called(ctx, schemaName, req)
	return args.Get(0).(tenant.Permission), args.Error(1)
}

func (m *MockPermissionService) GetPermissionByID(ctx context.Context, schemaName string, permissionID uuid.UUID) (tenant.Permission, error) {
	args := m.Called(ctx, schemaName, permissionID)
	return args.Get(0).(tenant.Permission), args.Error(1)
}

func (m *MockPermissionService) GetPermissionByResourceAndAction(ctx context.Context, schemaName string, resourceID, actionID uuid.UUID) (tenant.Permission, error) {
	args := m.Called(ctx, schemaName, resourceID, actionID)
	return args.Get(0).(tenant.Permission), args.Error(1)
}

func (m *MockPermissionService) GetOrCreatePermission(ctx context.Context, schemaName string, resourceID, actionID uuid.UUID) (tenant.Permission, error) {
	args := m.Called(ctx, schemaName, resourceID, actionID)
	return args.Get(0).(tenant.Permission), args.Error(1)
}

func (m *MockPermissionService) ListPermissions(ctx context.Context, schemaName string, limit, offset int) ([]tenant.Permission, int64, error) {
	args := m.Called(ctx, schemaName, limit, offset)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]tenant.Permission), args.Get(1).(int64), args.Error(2)
}

func (m *MockPermissionService) DeletePermission(ctx context.Context, schemaName string, permissionID uuid.UUID) error {
	args := m.Called(ctx, schemaName, permissionID)
	return args.Error(0)
}

func (m *MockPermissionService) GetPermissionsByResource(ctx context.Context, schemaName string, resourceID uuid.UUID) ([]tenant.Permission, error) {
	args := m.Called(ctx, schemaName, resourceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]tenant.Permission), args.Error(1)
}

type MockRolePermissionService struct {
	mock.Mock
}

func (m *MockRolePermissionService) AssignPermissionToRole(ctx context.Context, schemaName string, req dto.RolePermissionDTO) (tenant.RolePermission, error) {
	args := m.Called(ctx, schemaName, req)
	return args.Get(0).(tenant.RolePermission), args.Error(1)
}

func (m *MockRolePermissionService) RemovePermissionFromRole(ctx context.Context, schemaName string, roleID, permissionID uuid.UUID) error {
	args := m.Called(ctx, schemaName, roleID, permissionID)
	return args.Error(0)
}

func (m *MockRolePermissionService) GetRolePermissions(ctx context.Context, schemaName string, roleID uuid.UUID) ([]tenant.RolePermission, error) {
	args := m.Called(ctx, schemaName, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]tenant.RolePermission), args.Error(1)
}

func (m *MockRolePermissionService) GetPermissionsByRole(ctx context.Context, schemaName string, roleID uuid.UUID) ([]dto.PermissionWithDetails, error) {
	args := m.Called(ctx, schemaName, roleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dto.PermissionWithDetails), args.Error(1)
}

func (m *MockRolePermissionService) GetRolesByPermission(ctx context.Context, schemaName string, permissionID uuid.UUID) ([]tenant.AccessRole, error) {
	args := m.Called(ctx, schemaName, permissionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]tenant.AccessRole), args.Error(1)
}

func (m *MockRolePermissionService) CheckRoleHasPermission(ctx context.Context, schemaName string, roleID, permissionID uuid.UUID) (bool, error) {
	args := m.Called(ctx, schemaName, roleID, permissionID)
	return args.Bool(0), args.Error(1)
}

type MockAccessMemberService struct {
	mock.Mock
}

func (m *MockAccessMemberService) AssignRoleToUser(ctx context.Context, schemaName string, req dto.AccessMemberDTO) (interface{}, error) {
	args := m.Called(ctx, schemaName, req)
	return args.Get(0), args.Error(1)
}

func (m *MockAccessMemberService) RemoveRoleFromUser(ctx context.Context, schemaName string, userID, scopeID string, scopeType string) error {
	args := m.Called(ctx, schemaName, userID, scopeID, scopeType)
	return args.Error(0)
}

func (m *MockAccessMemberService) RemoveAccessMemberByID(ctx context.Context, schemaName string, memberID string) error {
	args := m.Called(ctx, schemaName, memberID)
	return args.Error(0)
}

func (m *MockAccessMemberService) GetUserAccessMembers(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
	args := m.Called(ctx, schemaName, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dto.AccessMemberDTO), args.Error(1)
}

func (m *MockAccessMemberService) GetUserAccessByScope(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string) ([]dto.AccessMemberDTO, error) {
	args := m.Called(ctx, schemaName, userID, scopeType, scopeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dto.AccessMemberDTO), args.Error(1)
}

func (m *MockAccessMemberService) GetScopeMembers(ctx context.Context, schemaName string, scopeType string, scopeID *string) ([]dto.AccessMemberDTO, error) {
	args := m.Called(ctx, schemaName, scopeType, scopeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dto.AccessMemberDTO), args.Error(1)
}

func (m *MockAccessMemberService) GetUserPermissions(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string) ([]dto.PermissionWithDetails, error) {
	args := m.Called(ctx, schemaName, userID, scopeType, scopeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dto.PermissionWithDetails), args.Error(1)
}

func (m *MockAccessMemberService) CheckUserPermission(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string, resourceCode, actionCode string) (bool, error) {
	args := m.Called(ctx, schemaName, userID, scopeType, scopeID, resourceCode, actionCode)
	return args.Bool(0), args.Error(1)
}

func (m *MockAccessMemberService) GetUserHighestRole(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string) (*dto.AccessRoleDTO, error) {
	args := m.Called(ctx, schemaName, userID, scopeType, scopeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.AccessRoleDTO), args.Error(1)
}

func (m *MockAccessMemberService) BulkAssignRoleToUsers(ctx context.Context, schemaName string, req dto.BulkAssignRoleRequest) error {
	args := m.Called(ctx, schemaName, req)
	return args.Error(0)
}

func (m *MockAccessMemberService) BulkRemoveRoleFromUsers(ctx context.Context, schemaName string, userIDs []string, scopeType string, scopeID *string, roleID string) error {
	args := m.Called(ctx, schemaName, userIDs, scopeType, scopeID, roleID)
	return args.Error(0)
}

func (m *MockAccessMemberService) UpdateRoleForUser(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string, newRoleID string) error {
	args := m.Called(ctx, schemaName, userID, scopeType, scopeID, newRoleID)
	return args.Error(0)
}

func TestNewRBACManagementService(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}

	mockRole := new(MockAccessRoleService)
	mockResource := new(MockResourceService)
	mockAction := new(MockActionService)
	mockPermission := new(MockPermissionService)
	mockRolePerm := new(MockRolePermissionService)
	mockAccessMember := new(MockAccessMemberService)

	deps := services.RBACManagementServiceDeps{
		RoleService:           mockRole,
		ResourceService:       mockResource,
		ActionService:         mockAction,
		PermissionService:     mockPermission,
		RolePermissionService: mockRolePerm,
		AccessMemberService:   mockAccessMember,
	}

	service := services.NewRBACManagementService(repo, deps)

	assert.NotNil(t, service)
}

func TestCreateRole(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockRole := new(MockAccessRoleService)

	deps := services.RBACManagementServiceDeps{
		RoleService: mockRole,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()
	req := dto.AccessRoleDTO{
		ID:         roleID,
		Name:       "Admin",
		ScopeLevel: "workspace",
		Priority:   100,
	}

	mockRole.On("CreateAccessRole", ctx, schemaName, req).
		Return(tenant.AccessRole{
			ID:         roleID,
			Name:       "Admin",
			ScopeLevel: "workspace",
			Priority:   100,
		}, nil)

	result, err := service.CreateRole(ctx, schemaName, req)

	assert.NoError(t, err)
	assert.Equal(t, roleID, result.ID)
	mockRole.AssertExpectations(t)
}

func TestGetRoleByID(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockRole := new(MockAccessRoleService)

	deps := services.RBACManagementServiceDeps{
		RoleService: mockRole,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()

	mockRole.On("GetAccessRoleByID", ctx, schemaName, roleID).
		Return(tenant.AccessRole{
			ID:   roleID,
			Name: "Admin",
		}, nil)

	result, err := service.GetRoleByID(ctx, schemaName, roleID)

	assert.NoError(t, err)
	assert.Equal(t, roleID, result.ID)
	mockRole.AssertExpectations(t)
}

func TestGetRoleByName(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockRole := new(MockAccessRoleService)

	deps := services.RBACManagementServiceDeps{
		RoleService: mockRole,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	roleName := "Admin"

	mockRole.On("GetAccessRoleByName", ctx, schemaName, roleName).
		Return(tenant.AccessRole{
			ID:   uuid.New(),
			Name: roleName,
		}, nil)

	result, err := service.GetRoleByName(ctx, schemaName, roleName)

	assert.NoError(t, err)
	assert.Equal(t, roleName, result.Name)
	mockRole.AssertExpectations(t)
}

func TestListRoles(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockRole := new(MockAccessRoleService)

	deps := services.RBACManagementServiceDeps{
		RoleService: mockRole,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"

	mockRole.On("ListAccessRoles", ctx, schemaName, 10, 0).
		Return([]tenant.AccessRole{
			{ID: uuid.New(), Name: "Admin"},
		}, int64(1), nil)

	result, count, err := service.ListRoles(ctx, schemaName, 10, 0)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, int64(1), count)
	mockRole.AssertExpectations(t)
}

func TestUpdateRole(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockRole := new(MockAccessRoleService)

	deps := services.RBACManagementServiceDeps{
		RoleService: mockRole,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()
	req := dto.AccessRoleDTO{
		Name: "Updated Admin",
	}

	mockRole.On("UpdateAccessRole", ctx, schemaName, roleID, req).
		Return(tenant.AccessRole{
			ID:   roleID,
			Name: "Updated Admin",
		}, nil)

	result, err := service.UpdateRole(ctx, schemaName, roleID, req)

	assert.NoError(t, err)
	assert.Equal(t, "Updated Admin", result.Name)
	mockRole.AssertExpectations(t)
}

func TestDeleteRole(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockRole := new(MockAccessRoleService)

	deps := services.RBACManagementServiceDeps{
		RoleService: mockRole,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()

	mockRole.On("DeleteAccessRole", ctx, schemaName, roleID).
		Return(nil)

	err := service.DeleteRole(ctx, schemaName, roleID)

	assert.NoError(t, err)
	mockRole.AssertExpectations(t)
}

func TestCreateResource(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockResource := new(MockResourceService)

	deps := services.RBACManagementServiceDeps{
		ResourceService: mockResource,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	resourceID := uuid.New()
	req := dto.ResourceDTO{
		ID:   resourceID,
		Code: "workspace",
	}

	mockResource.On("CreateResource", ctx, schemaName, req).
		Return(tenant.Resource{
			ID:   resourceID,
			Code: "workspace",
		}, nil)

	result, err := service.CreateResource(ctx, schemaName, req)

	assert.NoError(t, err)
	assert.Equal(t, resourceID, result.ID)
	mockResource.AssertExpectations(t)
}

func TestCreatePermission(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockPermission := new(MockPermissionService)

	deps := services.RBACManagementServiceDeps{
		PermissionService: mockPermission,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	permissionID := uuid.New()
	req := dto.PermissionDTO{
		ID:         permissionID,
		ResourceID: uuid.New(),
		ActionID:   uuid.New(),
	}

	mockPermission.On("CreatePermission", ctx, schemaName, req).
		Return(tenant.Permission{
			ID: permissionID,
		}, nil)

	result, err := service.CreatePermission(ctx, schemaName, req)

	assert.NoError(t, err)
	assert.Equal(t, permissionID, result.ID)
	mockPermission.AssertExpectations(t)
}

func TestAssignPermissionToRole(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockRolePerm := new(MockRolePermissionService)

	deps := services.RBACManagementServiceDeps{
		RolePermissionService: mockRolePerm,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	rolePermID := uuid.New()
	req := dto.RolePermissionDTO{
		ID:           rolePermID,
		RoleID:       uuid.New(),
		PermissionID: uuid.New(),
	}

	mockRolePerm.On("AssignPermissionToRole", ctx, schemaName, req).
		Return(tenant.RolePermission{
			ID: rolePermID,
		}, nil)

	result, err := service.AssignPermissionToRole(ctx, schemaName, req)

	assert.NoError(t, err)
	assert.Equal(t, rolePermID, result.ID)
	mockRolePerm.AssertExpectations(t)
}

func TestRBACMgmt_AssignRoleToUser(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockAccessMember := new(MockAccessMemberService)

	deps := services.RBACManagementServiceDeps{
		AccessMemberService: mockAccessMember,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	req := dto.AccessMemberDTO{
		ID:        uuid.New(),
		UserID:    "user-123",
		RoleID:    "role-456",
		ScopeType: "workspace",
	}

	mockAccessMember.On("AssignRoleToUser", ctx, schemaName, req).
		Return(tenant.AccessMember{}, nil)

	result, err := service.AssignRoleToUser(ctx, schemaName, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	mockAccessMember.AssertExpectations(t)
}

func TestRBACMgmt_AssignRoleToUser_NilService(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}

	deps := services.RBACManagementServiceDeps{
		AccessMemberService: nil,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	req := dto.AccessMemberDTO{
		ID:     uuid.New(),
		UserID: "user-123",
	}

	result, err := service.AssignRoleToUser(ctx, schemaName, req)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestRBACMgmt_RemoveRoleFromUser(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockAccessMember := new(MockAccessMemberService)

	deps := services.RBACManagementServiceDeps{
		AccessMemberService: mockAccessMember,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	userID := "user-123"
	scopeID := "scope-789"
	scopeType := "workspace"

	mockAccessMember.On("RemoveRoleFromUser", ctx, schemaName, userID, scopeID, scopeType).
		Return(nil)

	err := service.RemoveRoleFromUser(ctx, schemaName, userID, scopeID, scopeType)

	assert.NoError(t, err)
	mockAccessMember.AssertExpectations(t)
}

func TestGetUserPermissions(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockAccessMember := new(MockAccessMemberService)

	deps := services.RBACManagementServiceDeps{
		AccessMemberService: mockAccessMember,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	userID := "user-123"
	scopeType := "workspace"

	mockAccessMember.On("GetUserPermissions", ctx, schemaName, userID, scopeType, (*string)(nil)).
		Return([]dto.PermissionWithDetails{
			{
				ID:           uuid.New(),
				ResourceCode: "workspace",
				ActionCode:   "read",
			},
		}, nil)

	result, err := service.GetUserPermissions(ctx, schemaName, userID, scopeType, nil)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	mockAccessMember.AssertExpectations(t)
}

func TestRBACMgmt_CheckUserPermission(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockAccessMember := new(MockAccessMemberService)

	deps := services.RBACManagementServiceDeps{
		AccessMemberService: mockAccessMember,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	userID := "user-123"
	scopeType := "workspace"
	resourceCode := "workspace"
	actionCode := "read"

	mockAccessMember.On("CheckUserPermission", ctx, schemaName, userID, scopeType, (*string)(nil), resourceCode, actionCode).
		Return(true, nil)

	result, err := service.CheckUserPermission(ctx, schemaName, userID, scopeType, nil, resourceCode, actionCode)

	assert.NoError(t, err)
	assert.True(t, result)
	mockAccessMember.AssertExpectations(t)
}

func TestRBACMgmt_BulkAssignRoleToUsers(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockAccessMember := new(MockAccessMemberService)

	deps := services.RBACManagementServiceDeps{
		AccessMemberService: mockAccessMember,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	req := dto.BulkAssignRoleRequest{
		UserIDs:   []string{"user-1", "user-2"},
		RoleID:    "role-123",
		ScopeType: "workspace",
	}

	mockAccessMember.On("BulkAssignRoleToUsers", ctx, schemaName, req).
		Return(nil)

	err := service.BulkAssignRoleToUsers(ctx, schemaName, req)

	assert.NoError(t, err)
	mockAccessMember.AssertExpectations(t)
}

func TestBulkAssignPermissionsToRole(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockRolePerm := new(MockRolePermissionService)

	deps := services.RBACManagementServiceDeps{
		RolePermissionService: mockRolePerm,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()
	permissionIDs := []uuid.UUID{uuid.New(), uuid.New()}

	mockRolePerm.On("AssignPermissionToRole", ctx, schemaName, mock.Anything).
		Return(tenant.RolePermission{ID: uuid.New()}, nil).Times(2)

	err := service.BulkAssignPermissionsToRole(ctx, schemaName, roleID, permissionIDs)

	assert.NoError(t, err)
	mockRolePerm.AssertExpectations(t)
}

func TestBulkAssignPermissionsToRole_Error(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockRolePerm := new(MockRolePermissionService)

	deps := services.RBACManagementServiceDeps{
		RolePermissionService: mockRolePerm,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()
	permissionIDs := []uuid.UUID{uuid.New()}

	mockRolePerm.On("AssignPermissionToRole", ctx, schemaName, mock.Anything).
		Return(tenant.RolePermission{}, errors.New("assignment error"))

	err := service.BulkAssignPermissionsToRole(ctx, schemaName, roleID, permissionIDs)

	assert.Error(t, err)
	mockRolePerm.AssertExpectations(t)
}

func TestCountRoles(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}

	deps := services.RBACManagementServiceDeps{}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"

	mockTable.On("GetTableData", mock.Anything, mock.Anything).
		Return([]map[string]interface{}{
			{"total": int64(5)},
		}, nil)

	result, err := service.CountRoles(ctx, schemaName)

	assert.NoError(t, err)
	assert.Equal(t, int64(5), result)
	mockTable.AssertExpectations(t)
}

func TestCountRoles_EmptyData(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}

	deps := services.RBACManagementServiceDeps{}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"

	mockTable.On("GetTableData", mock.Anything, mock.Anything).
		Return([]map[string]interface{}{}, nil)

	result, err := service.CountRoles(ctx, schemaName)

	assert.NoError(t, err)
	assert.Equal(t, int64(0), result)
	mockTable.AssertExpectations(t)
}

func TestCountRoles_DatabaseError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}

	deps := services.RBACManagementServiceDeps{}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"

	mockTable.On("GetTableData", mock.Anything, mock.Anything).
		Return(nil, errors.New("database error"))

	result, err := service.CountRoles(ctx, schemaName)

	assert.Error(t, err)
	assert.Equal(t, int64(0), result)
	mockTable.AssertExpectations(t)
}

func TestCountPermissions(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}

	deps := services.RBACManagementServiceDeps{}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"

	mockTable.On("GetTableData", mock.Anything, mock.Anything).
		Return([]map[string]interface{}{
			{"total": int64(10)},
		}, nil)

	result, err := service.CountPermissions(ctx, schemaName)

	assert.NoError(t, err)
	assert.Equal(t, int64(10), result)
	mockTable.AssertExpectations(t)
}

func TestCountRolePermissions(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}

	deps := services.RBACManagementServiceDeps{}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()

	mockTable.On("GetTableData", mock.Anything, mock.Anything).
		Return([]map[string]interface{}{
			{"total": int64(3)},
		}, nil)

	result, err := service.CountRolePermissions(ctx, schemaName, roleID)

	assert.NoError(t, err)
	assert.Equal(t, int64(3), result)
	mockTable.AssertExpectations(t)
}

func TestGetRBACSystemStatus(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}

	deps := services.RBACManagementServiceDeps{}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"

	// Mock CountRoles
	mockTable.On("GetTableData", mock.Anything, mock.Anything).
		Return([]map[string]interface{}{{"total": int64(5)}}, nil).Once()

	// Mock CountResources
	mockTable.On("GetTableData", mock.Anything, mock.Anything).
		Return([]map[string]interface{}{{"total": int64(10)}}, nil).Once()

	// Mock CountActions
	mockTable.On("GetTableData", mock.Anything, mock.Anything).
		Return([]map[string]interface{}{{"total": int64(8)}}, nil).Once()

	// Mock CountPermissions
	mockTable.On("GetTableData", mock.Anything, mock.Anything).
		Return([]map[string]interface{}{{"total": int64(20)}}, nil).Once()

	result, err := service.GetRBACSystemStatus(ctx, schemaName)

	assert.NoError(t, err)
	assert.True(t, result.Initialized)
	assert.Equal(t, int64(5), result.TotalRoles)
	assert.Equal(t, int64(10), result.TotalResources)
	assert.Equal(t, int64(8), result.TotalActions)
	assert.Equal(t, int64(20), result.TotalPermissions)
	assert.Equal(t, "healthy", result.Status)
	mockTable.AssertExpectations(t)
}

func TestGetRBACSystemStatus_NotInitialized(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}

	deps := services.RBACManagementServiceDeps{}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"

	// Mock all count methods returning 0
	mockTable.On("GetTableData", mock.Anything, mock.Anything).
		Return([]map[string]interface{}{{"total": int64(0)}}, nil)

	result, err := service.GetRBACSystemStatus(ctx, schemaName)

	assert.NoError(t, err)
	assert.False(t, result.Initialized)
	assert.Equal(t, "not_initialized", result.Status)
	mockTable.AssertExpectations(t)
}

// ==================== Resource Management Tests ====================

func TestRBACMgmt_GetResourceByID(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockResource := new(MockResourceService)

	deps := services.RBACManagementServiceDeps{
		ResourceService: mockResource,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	resourceID := uuid.New()

	mockResource.On("GetResourceByID", ctx, schemaName, resourceID).
		Return(tenant.Resource{ID: resourceID, Code: "workspace"}, nil)

	result, err := service.GetResourceByID(ctx, schemaName, resourceID)

	assert.NoError(t, err)
	assert.Equal(t, resourceID, result.ID)
	mockResource.AssertExpectations(t)
}

func TestRBACMgmt_GetResourceByCode(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockResource := new(MockResourceService)

	deps := services.RBACManagementServiceDeps{
		ResourceService: mockResource,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	code := "workspace"

	mockResource.On("GetResourceByCode", ctx, schemaName, code).
		Return(tenant.Resource{ID: uuid.New(), Code: code}, nil)

	result, err := service.GetResourceByCode(ctx, schemaName, code)

	assert.NoError(t, err)
	assert.Equal(t, code, result.Code)
	mockResource.AssertExpectations(t)
}

func TestRBACMgmt_ListResources(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockResource := new(MockResourceService)

	deps := services.RBACManagementServiceDeps{
		ResourceService: mockResource,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"

	mockResource.On("ListResources", ctx, schemaName, 10, 0).
		Return([]tenant.Resource{{ID: uuid.New(), Code: "workspace"}}, int64(1), nil)

	result, count, err := service.ListResources(ctx, schemaName, 10, 0)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, int64(1), count)
	mockResource.AssertExpectations(t)
}

func TestRBACMgmt_UpdateResource(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockResource := new(MockResourceService)

	deps := services.RBACManagementServiceDeps{
		ResourceService: mockResource,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	resourceID := uuid.New()
	req := dto.ResourceDTO{Code: "updated_workspace"}

	mockResource.On("UpdateResource", ctx, schemaName, resourceID, req).
		Return(tenant.Resource{ID: resourceID, Code: "updated_workspace"}, nil)

	result, err := service.UpdateResource(ctx, schemaName, resourceID, req)

	assert.NoError(t, err)
	assert.Equal(t, "updated_workspace", result.Code)
	mockResource.AssertExpectations(t)
}

func TestRBACMgmt_DeleteResource(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockResource := new(MockResourceService)

	deps := services.RBACManagementServiceDeps{
		ResourceService: mockResource,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	resourceID := uuid.New()

	mockResource.On("DeleteResource", ctx, schemaName, resourceID).
		Return(nil)

	err := service.DeleteResource(ctx, schemaName, resourceID)

	assert.NoError(t, err)
	mockResource.AssertExpectations(t)
}

func TestRBACMgmt_GetOrCreateResource(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockResource := new(MockResourceService)

	deps := services.RBACManagementServiceDeps{
		ResourceService: mockResource,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	code := "workspace"
	desc := "Workspace resource"

	mockResource.On("GetOrCreateResource", ctx, schemaName, code, &desc).
		Return(tenant.Resource{ID: uuid.New(), Code: code}, nil)

	result, err := service.GetOrCreateResource(ctx, schemaName, code, &desc)

	assert.NoError(t, err)
	assert.Equal(t, code, result.Code)
	mockResource.AssertExpectations(t)
}

// ==================== Action Management Tests ====================

func TestCreateAction(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockAction := new(MockActionService)

	deps := services.RBACManagementServiceDeps{
		ActionService: mockAction,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	actionID := uuid.New()
	req := dto.ActionDTO{ID: actionID, Code: "read"}

	mockAction.On("CreateAction", ctx, schemaName, req).
		Return(tenant.Action{ID: actionID, Code: "read"}, nil)

	result, err := service.CreateAction(ctx, schemaName, req)

	assert.NoError(t, err)
	assert.Equal(t, actionID, result.ID)
	mockAction.AssertExpectations(t)
}

func TestRBACMgmt_GetActionByID(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockAction := new(MockActionService)

	deps := services.RBACManagementServiceDeps{
		ActionService: mockAction,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	actionID := uuid.New()

	mockAction.On("GetActionByID", ctx, schemaName, actionID).
		Return(tenant.Action{ID: actionID, Code: "read"}, nil)

	result, err := service.GetActionByID(ctx, schemaName, actionID)

	assert.NoError(t, err)
	assert.Equal(t, actionID, result.ID)
	mockAction.AssertExpectations(t)
}

func TestRBACMgmt_GetActionByCode(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockAction := new(MockActionService)

	deps := services.RBACManagementServiceDeps{
		ActionService: mockAction,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	code := "read"

	mockAction.On("GetActionByCode", ctx, schemaName, code).
		Return(tenant.Action{ID: uuid.New(), Code: code}, nil)

	result, err := service.GetActionByCode(ctx, schemaName, code)

	assert.NoError(t, err)
	assert.Equal(t, code, result.Code)
	mockAction.AssertExpectations(t)
}

func TestRBACMgmt_ListActions(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockAction := new(MockActionService)

	deps := services.RBACManagementServiceDeps{
		ActionService: mockAction,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"

	mockAction.On("ListActions", ctx, schemaName, 10, 0).
		Return([]tenant.Action{{ID: uuid.New(), Code: "read"}}, int64(1), nil)

	result, count, err := service.ListActions(ctx, schemaName, 10, 0)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, int64(1), count)
	mockAction.AssertExpectations(t)
}

func TestRBACMgmt_UpdateAction(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockAction := new(MockActionService)

	deps := services.RBACManagementServiceDeps{
		ActionService: mockAction,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	actionID := uuid.New()
	req := dto.ActionDTO{Code: "write"}

	mockAction.On("UpdateAction", ctx, schemaName, actionID, req).
		Return(tenant.Action{ID: actionID, Code: "write"}, nil)

	result, err := service.UpdateAction(ctx, schemaName, actionID, req)

	assert.NoError(t, err)
	assert.Equal(t, "write", result.Code)
	mockAction.AssertExpectations(t)
}

func TestRBACMgmt_DeleteAction(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockAction := new(MockActionService)

	deps := services.RBACManagementServiceDeps{
		ActionService: mockAction,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	actionID := uuid.New()

	mockAction.On("DeleteAction", ctx, schemaName, actionID).
		Return(nil)

	err := service.DeleteAction(ctx, schemaName, actionID)

	assert.NoError(t, err)
	mockAction.AssertExpectations(t)
}

func TestRBACMgmt_GetOrCreateAction(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockAction := new(MockActionService)

	deps := services.RBACManagementServiceDeps{
		ActionService: mockAction,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	code := "read"
	desc := "Read action"

	mockAction.On("GetOrCreateAction", ctx, schemaName, code, &desc).
		Return(tenant.Action{ID: uuid.New(), Code: code}, nil)

	result, err := service.GetOrCreateAction(ctx, schemaName, code, &desc)

	assert.NoError(t, err)
	assert.Equal(t, code, result.Code)
	mockAction.AssertExpectations(t)
}

// ==================== Permission Management Tests ====================

func TestRBACMgmt_GetPermissionByID(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockPermission := new(MockPermissionService)

	deps := services.RBACManagementServiceDeps{
		PermissionService: mockPermission,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	permissionID := uuid.New()

	mockPermission.On("GetPermissionByID", ctx, schemaName, permissionID).
		Return(tenant.Permission{ID: permissionID}, nil)

	result, err := service.GetPermissionByID(ctx, schemaName, permissionID)

	assert.NoError(t, err)
	assert.Equal(t, permissionID, result.ID)
	mockPermission.AssertExpectations(t)
}

func TestRBACMgmt_GetPermissionByResourceAndAction(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockPermission := new(MockPermissionService)

	deps := services.RBACManagementServiceDeps{
		PermissionService: mockPermission,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	resourceID := uuid.New()
	actionID := uuid.New()

	mockPermission.On("GetPermissionByResourceAndAction", ctx, schemaName, resourceID, actionID).
		Return(tenant.Permission{ID: uuid.New()}, nil)

	result, err := service.GetPermissionByResourceAndAction(ctx, schemaName, resourceID, actionID)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, result.ID)
	mockPermission.AssertExpectations(t)
}

func TestRBACMgmt_ListPermissions(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockPermission := new(MockPermissionService)

	deps := services.RBACManagementServiceDeps{
		PermissionService: mockPermission,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"

	mockPermission.On("ListPermissions", ctx, schemaName, 10, 0).
		Return([]tenant.Permission{{ID: uuid.New()}}, int64(1), nil)

	result, count, err := service.ListPermissions(ctx, schemaName, 10, 0)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, int64(1), count)
	mockPermission.AssertExpectations(t)
}

func TestRBACMgmt_DeletePermission(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockPermission := new(MockPermissionService)

	deps := services.RBACManagementServiceDeps{
		PermissionService: mockPermission,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	permissionID := uuid.New()

	mockPermission.On("DeletePermission", ctx, schemaName, permissionID).
		Return(nil)

	err := service.DeletePermission(ctx, schemaName, permissionID)

	assert.NoError(t, err)
	mockPermission.AssertExpectations(t)
}

func TestRBACMgmt_GetOrCreatePermission(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockPermission := new(MockPermissionService)

	deps := services.RBACManagementServiceDeps{
		PermissionService: mockPermission,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	resourceID := uuid.New()
	actionID := uuid.New()

	mockPermission.On("GetOrCreatePermission", ctx, schemaName, resourceID, actionID).
		Return(tenant.Permission{ID: uuid.New()}, nil)

	result, err := service.GetOrCreatePermission(ctx, schemaName, resourceID, actionID)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, result.ID)
	mockPermission.AssertExpectations(t)
}

func TestRBACMgmt_GetPermissionsByResource(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockPermission := new(MockPermissionService)

	deps := services.RBACManagementServiceDeps{
		PermissionService: mockPermission,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	resourceID := uuid.New()

	mockPermission.On("GetPermissionsByResource", ctx, schemaName, resourceID).
		Return([]tenant.Permission{{ID: uuid.New()}}, nil)

	result, err := service.GetPermissionsByResource(ctx, schemaName, resourceID)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	mockPermission.AssertExpectations(t)
}

// ==================== Role Permission Management Tests ====================

func TestRBACMgmt_RemovePermissionFromRole(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockRolePerm := new(MockRolePermissionService)

	deps := services.RBACManagementServiceDeps{
		RolePermissionService: mockRolePerm,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()
	permissionID := uuid.New()

	mockRolePerm.On("RemovePermissionFromRole", ctx, schemaName, roleID, permissionID).
		Return(nil)

	err := service.RemovePermissionFromRole(ctx, schemaName, roleID, permissionID)

	assert.NoError(t, err)
	mockRolePerm.AssertExpectations(t)
}

func TestRBACMgmt_GetRolePermissions(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockRolePerm := new(MockRolePermissionService)

	deps := services.RBACManagementServiceDeps{
		RolePermissionService: mockRolePerm,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()

	mockRolePerm.On("GetRolePermissions", ctx, schemaName, roleID).
		Return([]tenant.RolePermission{{ID: uuid.New()}}, nil)

	result, err := service.GetRolePermissions(ctx, schemaName, roleID)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	mockRolePerm.AssertExpectations(t)
}

func TestRBACMgmt_GetPermissionsByRole(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockRolePerm := new(MockRolePermissionService)

	deps := services.RBACManagementServiceDeps{
		RolePermissionService: mockRolePerm,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()

	mockRolePerm.On("GetPermissionsByRole", ctx, schemaName, roleID).
		Return([]dto.PermissionWithDetails{{ID: uuid.New()}}, nil)

	result, err := service.GetPermissionsByRole(ctx, schemaName, roleID)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	mockRolePerm.AssertExpectations(t)
}

func TestRBACMgmt_GetRolesByPermission(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockRolePerm := new(MockRolePermissionService)

	deps := services.RBACManagementServiceDeps{
		RolePermissionService: mockRolePerm,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	permissionID := uuid.New()

	mockRolePerm.On("GetRolesByPermission", ctx, schemaName, permissionID).
		Return([]tenant.AccessRole{{ID: uuid.New()}}, nil)

	result, err := service.GetRolesByPermission(ctx, schemaName, permissionID)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	mockRolePerm.AssertExpectations(t)
}

func TestRBACMgmt_CheckRoleHasPermission(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockRolePerm := new(MockRolePermissionService)

	deps := services.RBACManagementServiceDeps{
		RolePermissionService: mockRolePerm,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	roleID := uuid.New()
	permissionID := uuid.New()

	mockRolePerm.On("CheckRoleHasPermission", ctx, schemaName, roleID, permissionID).
		Return(true, nil)

	result, err := service.CheckRoleHasPermission(ctx, schemaName, roleID, permissionID)

	assert.NoError(t, err)
	assert.True(t, result)
	mockRolePerm.AssertExpectations(t)
}

// ==================== Access Member Management Tests ====================

func TestRBACMgmt_GetUserAccessMembers(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockAccessMember := new(MockAccessMemberService)

	deps := services.RBACManagementServiceDeps{
		AccessMemberService: mockAccessMember,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	userID := "user-123"

	mockAccessMember.On("GetUserAccessMembers", ctx, schemaName, userID).
		Return([]dto.AccessMemberDTO{{ID: uuid.New()}}, nil)

	result, err := service.GetUserAccessMembers(ctx, schemaName, userID)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	mockAccessMember.AssertExpectations(t)
}

func TestRBACMgmt_GetUserAccessByScope(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockAccessMember := new(MockAccessMemberService)

	deps := services.RBACManagementServiceDeps{
		AccessMemberService: mockAccessMember,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	userID := "user-123"
	scopeType := "workspace"

	mockAccessMember.On("GetUserAccessByScope", ctx, schemaName, userID, scopeType, (*string)(nil)).
		Return([]dto.AccessMemberDTO{{ID: uuid.New()}}, nil)

	result, err := service.GetUserAccessByScope(ctx, schemaName, userID, scopeType, nil)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	mockAccessMember.AssertExpectations(t)
}

func TestRBACMgmt_GetScopeMembers(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockAccessMember := new(MockAccessMemberService)

	deps := services.RBACManagementServiceDeps{
		AccessMemberService: mockAccessMember,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	scopeType := "workspace"

	mockAccessMember.On("GetScopeMembers", ctx, schemaName, scopeType, (*string)(nil)).
		Return([]dto.AccessMemberDTO{{ID: uuid.New()}}, nil)

	result, err := service.GetScopeMembers(ctx, schemaName, scopeType, nil)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	mockAccessMember.AssertExpectations(t)
}

func TestRBACMgmt_GetUserHighestRole(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockAccessMember := new(MockAccessMemberService)

	deps := services.RBACManagementServiceDeps{
		AccessMemberService: mockAccessMember,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	userID := "user-123"
	scopeType := "workspace"

	roleID := uuid.New()
	mockAccessMember.On("GetUserHighestRole", ctx, schemaName, userID, scopeType, (*string)(nil)).
		Return(&dto.AccessRoleDTO{ID: roleID}, nil)

	result, err := service.GetUserHighestRole(ctx, schemaName, userID, scopeType, nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, roleID, result.ID)
	mockAccessMember.AssertExpectations(t)
}

func TestRBACMgmt_BulkRemoveRoleFromUsers(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockAccessMember := new(MockAccessMemberService)

	deps := services.RBACManagementServiceDeps{
		AccessMemberService: mockAccessMember,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	userIDs := []string{"user-1", "user-2"}
	scopeType := "workspace"
	roleID := "role-123"

	mockAccessMember.On("BulkRemoveRoleFromUsers", ctx, schemaName, userIDs, scopeType, (*string)(nil), roleID).
		Return(nil)

	err := service.BulkRemoveRoleFromUsers(ctx, schemaName, userIDs, scopeType, nil, roleID)

	assert.NoError(t, err)
	mockAccessMember.AssertExpectations(t)
}

func TestGetRolesByScope(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}
	mockRole := new(MockAccessRoleService)

	deps := services.RBACManagementServiceDeps{
		RoleService: mockRole,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	scopeLevel := "workspace"

	mockRole.On("GetAccessRolesByScope", ctx, schemaName, scopeLevel).
		Return([]tenant.AccessRole{{ID: uuid.New(), Name: "Admin"}}, nil)

	result, err := service.GetRolesByScope(ctx, schemaName, scopeLevel)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	mockRole.AssertExpectations(t)
}

// ==================== Nil Service Tests ====================

func TestRemoveRoleFromUser_NilService(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}

	deps := services.RBACManagementServiceDeps{
		AccessMemberService: nil,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"

	err := service.RemoveRoleFromUser(ctx, schemaName, "user-123", "scope-456", "workspace")

	assert.Error(t, err)
}

func TestGetUserPermissions_NilService(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}

	deps := services.RBACManagementServiceDeps{
		AccessMemberService: nil,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"

	result, err := service.GetUserPermissions(ctx, schemaName, "user-123", "workspace", nil)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestCheckUserPermission_NilService(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}

	deps := services.RBACManagementServiceDeps{
		AccessMemberService: nil,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"

	result, err := service.CheckUserPermission(ctx, schemaName, "user-123", "workspace", nil, "workspace", "read")

	assert.Error(t, err)
	assert.False(t, result)
}

func TestBulkAssignRoleToUsers_NilService(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}

	deps := services.RBACManagementServiceDeps{
		AccessMemberService: nil,
	}

	service := services.NewRBACManagementService(repo, deps)

	ctx := context.Background()
	schemaName := "test_schema"
	req := dto.BulkAssignRoleRequest{
		UserIDs: []string{"user-1"},
		RoleID:  "role-123",
	}

	err := service.BulkAssignRoleToUsers(ctx, schemaName, req)

	assert.Error(t, err)
}
