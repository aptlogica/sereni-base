package rbac_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-postgres-rest/pkg"
	dbModels "go-postgres-rest/pkg/models"
	app_errors "serenibase/internal/app-errors"
	"serenibase/internal/constant"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	services "serenibase/internal/services/rbac"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockBaseService struct{ mock.Mock }

func (m *MockBaseService) CreateBase(ctx context.Context, schemaName string) (tenant.Base, error) {
	args := m.Called(ctx, schemaName)
	return args.Get(0).(tenant.Base), args.Error(1)
}
func (m *MockBaseService) BaseInsertion(ctx context.Context, req dto.BaseInsertion, schemaName string) (tenant.Base, error) {
	args := m.Called(ctx, req, schemaName)
	return args.Get(0).(tenant.Base), args.Error(1)
}
func (m *MockBaseService) GetBaseByID(ctx context.Context, schemaName string, id string) (tenant.Base, error) {
	args := m.Called(ctx, schemaName, id)
	return args.Get(0).(tenant.Base), args.Error(1)
}
func (m *MockBaseService) GetAllBases(ctx context.Context, schemaName string) ([]tenant.Base, error) {
	args := m.Called(ctx, schemaName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]tenant.Base), args.Error(1)
}
func (m *MockBaseService) UpdateBase(ctx context.Context, schemaName string, id string, req dto.BaseUpdate) (tenant.Base, error) {
	args := m.Called(ctx, schemaName, id, req)
	return args.Get(0).(tenant.Base), args.Error(1)
}
func (m *MockBaseService) DeleteBase(ctx context.Context, schemaName string, id string) error {
	args := m.Called(ctx, schemaName, id)
	return args.Error(0)
}
func (m *MockBaseService) GetBasesByWorkspace(ctx context.Context, schemaName, workspaceID string) ([]tenant.Base, error) {
	args := m.Called(ctx, schemaName, workspaceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]tenant.Base), args.Error(1)
}
func (m *MockBaseService) GetBulkbases(ctx context.Context, schemaName string, ids []string) ([]tenant.Base, error) {
	args := m.Called(ctx, schemaName, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]tenant.Base), args.Error(1)
}

func TestInitializeRBACSystem_SuccessAndPartialErrors(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}

	mockRole := new(MockAccessRoleService)
	mockResource := new(MockResourceService)
	mockAction := new(MockActionService)
	mockPermission := new(MockPermissionService)
	mockRolePerm := new(MockRolePermissionService)
	mockAccessMember := new(MockAccessMemberService)

	mockResource.On("GetOrCreateResource", mock.Anything, "schema", constant.ResourceCodes.Settings, mock.Anything).Return(tenant.Resource{}, errors.New("fail"))
	mockResource.On("GetOrCreateResource", mock.Anything, "schema", mock.Anything, mock.Anything).Return(tenant.Resource{ID: uuid.New()}, nil)

	mockAction.On("GetOrCreateAction", mock.Anything, "schema", constant.ActionCodes.Export, mock.Anything).Return(tenant.Action{}, errors.New("fail"))
	mockAction.On("GetOrCreateAction", mock.Anything, "schema", mock.Anything, mock.Anything).Return(tenant.Action{ID: uuid.New()}, nil)

	mockRole.On("CreateAccessRole", mock.Anything, "schema", mock.Anything).Return(tenant.AccessRole{ID: uuid.New()}, nil)
	mockPermission.On("GetOrCreatePermission", mock.Anything, "schema", mock.Anything, mock.Anything).Return(tenant.Permission{ID: uuid.New()}, nil)
	mockRolePerm.On("AssignPermissionToRole", mock.Anything, "schema", mock.Anything).Return(tenant.RolePermission{ID: uuid.New()}, nil)

	deps := services.RBACManagementServiceDeps{
		RoleService:           mockRole,
		ResourceService:       mockResource,
		ActionService:         mockAction,
		PermissionService:     mockPermission,
		RolePermissionService: mockRolePerm,
		AccessMemberService:   mockAccessMember,
	}

	svc := services.NewRBACManagementService(repo, deps)

	err := svc.InitializeRBACSystem(context.Background(), "schema")
	assert.NoError(t, err)
}

func TestRBACSystemStatus_Counts(t *testing.T) {
	stubTable := &StubTableService{}
	stubTable.GetTableDataFn = func(tableName string, _ dbModels.QueryParams) ([]map[string]interface{}, error) {
		return []map[string]interface{}{{"total": int64(2)}}, nil
	}

	repo := &pkg.DatabaseService{TableService: stubTable}
	deps := services.RBACManagementServiceDeps{}
	svc := services.NewRBACManagementService(repo, deps)

	status, err := svc.GetRBACSystemStatus(context.Background(), "schema")
	assert.NoError(t, err)
	assert.True(t, status.Initialized)

	stubTable.GetTableDataFn = func(tableName string, _ dbModels.QueryParams) ([]map[string]interface{}, error) {
		return []map[string]interface{}{}, nil
	}
	status, err = svc.GetRBACSystemStatus(context.Background(), "schema")
	assert.NoError(t, err)
	assert.Equal(t, "not_initialized", status.Status)
}

func TestRBACManagement_AccessMemberDelegates(t *testing.T) {
	svc := services.NewRBACManagementService(&pkg.DatabaseService{}, services.RBACManagementServiceDeps{})

	_, err := svc.AssignRoleToUser(context.Background(), "schema", dto.AccessMemberDTO{})
	assert.ErrorIs(t, err, app_errors.ErrServiceNotInitialized)

	err = svc.RemoveRoleFromUser(context.Background(), "schema", "u", "s", "workspace")
	assert.ErrorIs(t, err, app_errors.ErrServiceNotInitialized)

	type accessMemberOps interface {
		RemoveAccessMemberByID(ctx context.Context, schemaName string, memberID string) error
	}
	accessOps, ok := svc.(accessMemberOps)
	if assert.True(t, ok) {
		err = accessOps.RemoveAccessMemberByID(context.Background(), "schema", "id")
		assert.ErrorIs(t, err, app_errors.ErrServiceNotInitialized)
	}
}

func TestGetAllUserAccessInWorkspace_Filtering(t *testing.T) {
	mockAccessMember := new(MockAccessMemberService)
	deps := services.RBACManagementServiceDeps{AccessMemberService: mockAccessMember}
	svc := services.NewRBACManagementService(&pkg.DatabaseService{}, deps)

	wsID := "ws"
	baseWS := wsID
	mockAccessMember.On("GetUserAccessMembers", mock.Anything, "schema", "user").Return([]dto.AccessMemberDTO{
		{ScopeType: constant.ScopeLevels.Base, WorkspaceID: &baseWS},
		{ScopeType: constant.ScopeLevels.Workspace, ScopeID: &wsID},
		{ScopeType: constant.ScopeLevels.Workspace, ScopeID: nil},
	}, nil)

	type workspaceAccessOps interface {
		GetAllUserAccessInWorkspace(ctx context.Context, schemaName string, userID, workspaceID string) ([]dto.AccessMemberDTO, error)
	}
	accessOps, ok := svc.(workspaceAccessOps)
	if assert.True(t, ok) {
		rows, err := accessOps.GetAllUserAccessInWorkspace(context.Background(), "schema", "user", wsID)
		assert.NoError(t, err)
		assert.Len(t, rows, 2)
	}
}

func TestRBACAnalyticsAndReports(t *testing.T) {
	stubTable := &StubTableService{}
	stubTable.GetTableDataFn = func(tableName string, _ dbModels.QueryParams) ([]map[string]interface{}, error) {
		return []map[string]interface{}{{"total": int64(1)}}, nil
	}
	repo := &pkg.DatabaseService{TableService: stubTable}

	mockRole := new(MockAccessRoleService)
	mockResource := new(MockResourceService)
	mockPermission := new(MockPermissionService)
	mockRolePerm := new(MockRolePermissionService)

	roleID := uuid.New()
	permID := uuid.New()
	resID := uuid.New()

	mockRole.On("ListAccessRoles", mock.Anything, "schema", 100, 0).Return([]tenant.AccessRole{{ID: roleID, Name: "owner"}}, int64(1), nil)
	mockRole.On("GetAccessRoleByID", mock.Anything, "schema", roleID).Return(tenant.AccessRole{ID: roleID, Name: "owner", ScopeLevel: "workspace"}, nil)

	mockResource.On("ListResources", mock.Anything, "schema", 100, 0).Return([]tenant.Resource{{ID: resID, Code: "workspace"}}, int64(1), nil)

	mockPermission.On("GetPermissionsByResource", mock.Anything, "schema", resID).Return([]tenant.Permission{{ID: permID}}, nil)
	mockPermission.On("ListPermissions", mock.Anything, "schema", 1000, 0).Return([]tenant.Permission{{ID: permID}}, int64(1), nil)
	mockPermission.On("GetPermissionByID", mock.Anything, "schema", permID).Return(tenant.Permission{ID: permID, ResourceID: resID}, nil)
	mockRolePerm.On("GetRolesByPermission", mock.Anything, "schema", permID).Return([]tenant.AccessRole{}, nil)

	deps := services.RBACManagementServiceDeps{
		RoleService:           mockRole,
		ResourceService:       mockResource,
		PermissionService:     mockPermission,
		RolePermissionService: mockRolePerm,
	}

	svc := services.NewRBACManagementService(repo, deps)

	analytics, err := svc.GetRBACAnalytics(context.Background(), "schema")
	assert.NoError(t, err)
	assert.Len(t, analytics.TopRoles, 1)

	stats, err := svc.GetRoleUsageStats(context.Background(), "schema", roleID)
	assert.NoError(t, err)
	assert.Equal(t, roleID, stats.RoleID)

	permStats, err := svc.GetPermissionUsageStats(context.Background(), "schema", permID)
	assert.NoError(t, err)
	assert.Equal(t, permID, permStats.PermissionID)

	matrix, err := svc.GetResourceAccessMatrix(context.Background(), "schema")
	assert.NoError(t, err)
	assert.Len(t, matrix, 1)
}

func TestValidateAndAudit(t *testing.T) {
	stubTable := &StubTableService{}
	stubTable.GetTableDataFn = func(tableName string, _ dbModels.QueryParams) ([]map[string]interface{}, error) {
		return []map[string]interface{}{{"total": int64(0)}}, nil
	}
	repo := &pkg.DatabaseService{TableService: stubTable}

	mockRole := new(MockAccessRoleService)
	mockAccessMember := new(MockAccessMemberService)

	roleID := uuid.New()
	mockRole.On("GetAccessRoleByID", mock.Anything, "schema", roleID).Return(tenant.AccessRole{}, errors.New("not found"))

	deps := services.RBACManagementServiceDeps{
		RoleService:         mockRole,
		AccessMemberService: mockAccessMember,
	}

	svc := services.NewRBACManagementService(repo, deps)

	res, err := svc.ValidateRoleConfiguration(context.Background(), "schema", roleID)
	assert.Error(t, err)
	assert.False(t, res.IsValid)

	mockRole.On("GetAccessRoleByID", mock.Anything, "schema", roleID).Return(tenant.AccessRole{ID: roleID, Name: "owner", ScopeLevel: "workspace"}, nil)

	mockAccessMember.On("GetUserAccessMembers", mock.Anything, "schema", "user").Return([]dto.AccessMemberDTO{
		{ScopeType: "system", RoleID: roleID.String(), CreatedAt: time.Now()},
	}, nil)

	audit, err := svc.AuditUserAccess(context.Background(), "schema", "user")
	assert.NoError(t, err)
	assert.Equal(t, "user", audit.UserID)
}

func TestProcessUserMemberships_Variants(t *testing.T) {
	stubTable := &StubTableService{}
	repo := &pkg.DatabaseService{TableService: stubTable}

	mockRole := new(MockAccessRoleService)
	mockAccessMember := new(MockAccessMemberService)
	mockBase := new(MockBaseService)

	roleID := uuid.New()
	mockRole.On("GetAccessRoleByName", mock.Anything, "schema", mock.Anything).Return(tenant.AccessRole{ID: roleID}, nil)

	wsID := "ws"
	memberID := uuid.New()
	mockAccessMember.On("GetUserAccessMembers", mock.Anything, "schema", "user").Return([]dto.AccessMemberDTO{
		{ID: memberID, ScopeType: constant.ScopeLevels.Workspace, ScopeID: &wsID, RoleID: roleID.String()},
	}, nil)
	mockAccessMember.On("UpdateRoleForUser", mock.Anything, "schema", "user", constant.ScopeLevels.Workspace, &wsID, mock.Anything).Return(nil)
	mockAccessMember.On("AssignRoleToUser", mock.Anything, "schema", mock.Anything).Return(&tenant.AccessMember{ID: memberID}, nil)
	mockAccessMember.On("RemoveAccessMemberByID", mock.Anything, "schema", memberID.String()).Return(nil)
	mockAccessMember.On("GetUserAccessByScope", mock.Anything, "schema", "user", constant.ScopeLevels.Workspace, (*string)(nil)).Return([]dto.AccessMemberDTO{{ID: memberID, ScopeID: &wsID}}, nil)
	mockAccessMember.On("GetUserAccessByScope", mock.Anything, "schema", "user", constant.ScopeLevels.Base, mock.Anything).Return([]dto.AccessMemberDTO{}, nil)

	baseID := "base"
	mockBase.On("GetBaseByID", mock.Anything, "schema", baseID).Return(tenant.Base{ID: uuid.New(), WorkspaceID: wsID}, nil)

	deps := services.RBACManagementServiceDeps{
		RoleService:         mockRole,
		AccessMemberService: mockAccessMember,
		BaseService:         mockBase,
	}
	svc := services.NewRBACManagementService(repo, deps)

	summary, err := svc.ProcessUserMemberships(context.Background(), "schema", "user", "admin", []dto.MembershipRequest{
		{Role: constant.RBACRoleNames.BaseMember, WorkspaceID: wsID}, // invalid base role with workspace
		{Role: constant.RBACRoleNames.WorkspaceMaintainer, WorkspaceID: ""},
		{Role: constant.RBACRoleNames.WorkspaceMaintainer, WorkspaceID: wsID},
		{Role: "", Bases: []dto.BaseMembership{{BaseID: baseID, Role: "base-member"}}},
	})

	assert.NoError(t, err)
	assert.NotNil(t, summary)
}
