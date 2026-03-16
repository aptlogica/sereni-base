package rbac_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aptlogica/go-postgres-rest/pkg"
	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	services "github.com/aptlogica/sereni-base/internal/services/rbac"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRBACManagement_Counts_Branches(t *testing.T) {
	schema := "schema"
	roleID := uuid.New()

	stubTable := &StubTableService{}
	stubTable.GetTableDataFn = func(tableName string, _ dbModels.QueryParams) ([]map[string]interface{}, error) {
		switch tableName {
		case (tenant.AccessRole{}).TableName(schema):
			return nil, errors.New("count error")
		case (tenant.Resource{}).TableName(schema):
			return []map[string]interface{}{}, nil
		case (tenant.Action{}).TableName(schema):
			return []map[string]interface{}{{"total": "bad"}}, nil
		case (tenant.Permission{}).TableName(schema):
			return []map[string]interface{}{{"total": int64(5)}}, nil
		case (tenant.RolePermission{}).TableName(schema):
			return []map[string]interface{}{{"total": int64(0)}}, nil
		default:
			return []map[string]interface{}{}, nil
		}
	}

	svc := services.NewRBACManagementService(&pkg.DatabaseService{TableService: stubTable}, services.RBACManagementServiceDeps{})

	_, err := svc.CountRoles(context.Background(), schema)
	assert.Error(t, err)

	count, err := svc.CountResources(context.Background(), schema)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)

	count, err = svc.CountActions(context.Background(), schema)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)

	count, err = svc.CountPermissions(context.Background(), schema)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), count)

	count, err = svc.CountRolePermissions(context.Background(), schema, roleID)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestRBACManagement_CountRolePermissions_Error(t *testing.T) {
	schema := "schema"
	roleID := uuid.New()

	stubTable := &StubTableService{}
	stubTable.GetTableDataFn = func(tableName string, _ dbModels.QueryParams) ([]map[string]interface{}, error) {
		return nil, errors.New("count error")
	}

	svc := services.NewRBACManagementService(&pkg.DatabaseService{TableService: stubTable}, services.RBACManagementServiceDeps{})
	_, err := svc.CountRolePermissions(context.Background(), schema, roleID)
	assert.Error(t, err)
}

func TestRBACManagement_GetUnusedRoles(t *testing.T) {
	schema := "schema"
	roleUnused := uuid.New()
	roleUsed := uuid.New()

	stubTable := &StubTableService{}
	stubTable.GetTableDataFn = func(tableName string, params dbModels.QueryParams) ([]map[string]interface{}, error) {
		if tableName != (tenant.RolePermission{}).TableName(schema) {
			return []map[string]interface{}{}, nil
		}
		if len(params.Filters) > 0 && params.Filters[0].Value == roleUnused.String() {
			return []map[string]interface{}{{"total": int64(0)}}, nil
		}
		return []map[string]interface{}{{"total": int64(2)}}, nil
	}

	mockRole := new(MockAccessRoleService)
	mockRole.On("ListAccessRoles", mock.Anything, schema, 1000, 0).
		Return([]tenant.AccessRole{
			{ID: roleUnused, Name: "unused"},
			{ID: roleUsed, Name: "used"},
		}, int64(2), nil)

	svc := services.NewRBACManagementService(&pkg.DatabaseService{TableService: stubTable}, services.RBACManagementServiceDeps{
		RoleService: mockRole,
	})

	roles, err := svc.GetUnusedRoles(context.Background(), schema)
	assert.NoError(t, err)
	if assert.Len(t, roles, 1) {
		assert.Equal(t, roleUnused, roles[0].ID)
	}
}

func TestRBACManagement_ValidateRoleConfiguration_Success(t *testing.T) {
	schema := "schema"
	roleID := uuid.New()

	mockRole := new(MockAccessRoleService)
	mockRole.On("GetAccessRoleByID", mock.Anything, schema, roleID).
		Return(tenant.AccessRole{ID: roleID, Name: "owner", ScopeLevel: "workspace"}, nil)

	call := 0
	stubTable := &StubTableService{}
	stubTable.GetTableDataFn = func(tableName string, _ dbModels.QueryParams) ([]map[string]interface{}, error) {
		call++
		if call == 1 {
			return []map[string]interface{}{{"total": int64(0)}}, nil
		}
		return []map[string]interface{}{{"total": int64(2)}}, nil
	}

	svc := services.NewRBACManagementService(&pkg.DatabaseService{TableService: stubTable}, services.RBACManagementServiceDeps{
		RoleService: mockRole,
	})

	res, err := svc.ValidateRoleConfiguration(context.Background(), schema, roleID)
	assert.NoError(t, err)
	assert.False(t, res.HasPermissions)
	assert.Len(t, res.Warnings, 1)

	res, err = svc.ValidateRoleConfiguration(context.Background(), schema, roleID)
	assert.NoError(t, err)
	assert.True(t, res.HasPermissions)
}

func TestRBACManagement_AuditUserAccess_Scopes(t *testing.T) {
	schema := "schema"
	roleSystem := uuid.New()
	roleWorkspace := uuid.New()
	roleBase := uuid.New()
	wsID := "ws"
	baseID := "base"

	mockAccess := new(MockAccessMemberService)
	mockRole := new(MockAccessRoleService)

	mockAccess.On("GetUserAccessMembers", mock.Anything, schema, "user").
		Return([]dto.AccessMemberDTO{
			{ID: uuid.New(), ScopeType: "system", RoleID: roleSystem.String()},
			{ID: uuid.New(), ScopeType: "workspace", ScopeID: strPtr(wsID), RoleID: roleWorkspace.String()},
			{ID: uuid.New(), ScopeType: "base", ScopeID: strPtr(baseID), RoleID: roleBase.String()},
		}, nil)

	mockRole.On("GetAccessRoleByID", mock.Anything, schema, roleSystem).
		Return(tenant.AccessRole{ID: roleSystem, Name: "owner", ScopeLevel: "system", Priority: 100}, nil)
	mockRole.On("GetAccessRoleByID", mock.Anything, schema, roleWorkspace).
		Return(tenant.AccessRole{ID: roleWorkspace, Name: "maintainer", ScopeLevel: "workspace", Priority: 80}, nil)
	mockRole.On("GetAccessRoleByID", mock.Anything, schema, roleBase).
		Return(tenant.AccessRole{ID: roleBase, Name: "base-member", ScopeLevel: "base", Priority: 60}, nil)

	svc := services.NewRBACManagementService(&pkg.DatabaseService{}, services.RBACManagementServiceDeps{
		AccessMemberService: mockAccess,
		RoleService:         mockRole,
	})

	audit, err := svc.AuditUserAccess(context.Background(), schema, "user")
	assert.NoError(t, err)
	assert.Equal(t, 3, audit.TotalRoles)
	assert.Len(t, audit.SystemRoles, 1)
	assert.Len(t, audit.WorkspaceRoles, 1)
	assert.Len(t, audit.BaseRoles, 1)
}

func TestRBACManagement_GetAllUserAccessInWorkspace_Error(t *testing.T) {
	type workspaceAccessOps interface {
		GetAllUserAccessInWorkspace(ctx context.Context, schemaName string, userID, workspaceID string) ([]dto.AccessMemberDTO, error)
	}

	mockAccess := new(MockAccessMemberService)
	mockAccess.On("GetUserAccessMembers", mock.Anything, "schema", "user").
		Return(nil, errors.New("failed"))

	svc := services.NewRBACManagementService(&pkg.DatabaseService{}, services.RBACManagementServiceDeps{
		AccessMemberService: mockAccess,
	})

	accessOps, ok := svc.(workspaceAccessOps)
	if assert.True(t, ok) {
		_, err := accessOps.GetAllUserAccessInWorkspace(context.Background(), "schema", "user", "ws")
		assert.Error(t, err)
	}
}

func TestRBACManagement_AccessMemberDelegates_Success(t *testing.T) {
	mockAccess := new(MockAccessMemberService)
	expected := []dto.AccessMemberDTO{{ID: uuid.New(), ScopeType: constant.ScopeLevels.Workspace}}

	mockAccess.On("GetUserAccessMembers", mock.Anything, "schema", "user").
		Return(expected, nil)
	mockAccess.On("GetUserAccessByScope", mock.Anything, "schema", "user", constant.ScopeLevels.Workspace, matchScopeID("ws")).
		Return(expected, nil)
	mockAccess.On("GetScopeMembers", mock.Anything, "schema", constant.ScopeLevels.Workspace, matchScopeID("ws")).
		Return(expected, nil)
	mockAccess.On("GetUserHighestRole", mock.Anything, "schema", "user", constant.ScopeLevels.Workspace, matchScopeID("ws")).
		Return(&dto.AccessRoleDTO{Name: "owner"}, nil)
	mockAccess.On("BulkRemoveRoleFromUsers", mock.Anything, "schema", []string{"u1"}, constant.ScopeLevels.Workspace, matchScopeID("ws"), "role").
		Return(nil)

	svc := services.NewRBACManagementService(&pkg.DatabaseService{}, services.RBACManagementServiceDeps{
		AccessMemberService: mockAccess,
	})

	rows, err := svc.GetUserAccessMembers(context.Background(), "schema", "user")
	assert.NoError(t, err)
	assert.Len(t, rows, 1)

	rows, err = svc.GetUserAccessByScope(context.Background(), "schema", "user", constant.ScopeLevels.Workspace, strPtr("ws"))
	assert.NoError(t, err)

	rows, err = svc.GetScopeMembers(context.Background(), "schema", constant.ScopeLevels.Workspace, strPtr("ws"))
	assert.NoError(t, err)
	assert.Len(t, rows, 1)

	role, err := svc.GetUserHighestRole(context.Background(), "schema", "user", constant.ScopeLevels.Workspace, strPtr("ws"))
	assert.NoError(t, err)
	assert.Equal(t, "owner", role.Name)

	err = svc.BulkRemoveRoleFromUsers(context.Background(), "schema", []string{"u1"}, constant.ScopeLevels.Workspace, strPtr("ws"), "role")
	assert.NoError(t, err)
}

func TestRBACManagement_AccessMemberDelegates_Errors(t *testing.T) {
	svc := services.NewRBACManagementService(&pkg.DatabaseService{}, services.RBACManagementServiceDeps{})

	_, err := svc.GetUserAccessMembers(context.Background(), "schema", "user")
	assert.Error(t, err)

	_, err = svc.GetUserAccessByScope(context.Background(), "schema", "user", constant.ScopeLevels.Workspace, nil)
	assert.Error(t, err)

	_, err = svc.GetScopeMembers(context.Background(), "schema", constant.ScopeLevels.Workspace, nil)
	assert.Error(t, err)

	_, err = svc.GetUserHighestRole(context.Background(), "schema", "user", constant.ScopeLevels.Workspace, nil)
	assert.Error(t, err)

	err = svc.BulkRemoveRoleFromUsers(context.Background(), "schema", []string{"u1"}, constant.ScopeLevels.Workspace, nil, "role")
	assert.Error(t, err)
}

func TestRBACManagement_RoleAndPermissionUsage_ErrorBranches(t *testing.T) {
	schema := "schema"
	roleID := uuid.New()
	permID := uuid.New()

	mockRole := new(MockAccessRoleService)
	mockPermission := new(MockPermissionService)

	mockRole.On("GetAccessRoleByID", mock.Anything, schema, roleID).
		Return(tenant.AccessRole{}, errors.New("role error"))
	mockPermission.On("GetPermissionByID", mock.Anything, schema, permID).
		Return(tenant.Permission{}, errors.New("perm error"))

	svc := services.NewRBACManagementService(&pkg.DatabaseService{}, services.RBACManagementServiceDeps{
		RoleService:       mockRole,
		PermissionService: mockPermission,
	})

	_, err := svc.GetRoleUsageStats(context.Background(), schema, roleID)
	assert.Error(t, err)

	_, err = svc.GetPermissionUsageStats(context.Background(), schema, permID)
	assert.Error(t, err)
}

func TestRBACManagement_AnalyticsAndMatrix_ErrorBranches(t *testing.T) {
	schema := "schema"

	mockRole := new(MockAccessRoleService)
	mockResource := new(MockResourceService)
	mockPermission := new(MockPermissionService)

	stubTable := &StubTableService{}
	stubTable.GetTableDataFn = func(tableName string, _ dbModels.QueryParams) ([]map[string]interface{}, error) {
		return nil, errors.New("count error")
	}

	mockRole.On("ListAccessRoles", mock.Anything, schema, 100, 0).
		Return(nil, int64(0), errors.New("role list error"))

	mockResource.On("ListResources", mock.Anything, schema, 100, 0).
		Return(nil, int64(0), errors.New("resource list error"))

	mockPermission.On("ListPermissions", mock.Anything, schema, 1000, 0).
		Return(nil, int64(0), errors.New("perm list error"))

	svc := services.NewRBACManagementService(&pkg.DatabaseService{TableService: stubTable}, services.RBACManagementServiceDeps{
		RoleService:       mockRole,
		ResourceService:   mockResource,
		PermissionService: mockPermission,
	})

	_, err := svc.GetRBACAnalytics(context.Background(), schema)
	assert.Error(t, err)

	_, err = svc.GetResourceAccessMatrix(context.Background(), schema)
	assert.Error(t, err)

	_, err = svc.GetOrphanedPermissions(context.Background(), schema)
	assert.Error(t, err)
}

func TestRBACManagement_GetUnusedRoles_Error(t *testing.T) {
	mockRole := new(MockAccessRoleService)
	mockRole.On("ListAccessRoles", mock.Anything, "schema", 1000, 0).
		Return(nil, int64(0), errors.New("list error"))

	svc := services.NewRBACManagementService(&pkg.DatabaseService{}, services.RBACManagementServiceDeps{
		RoleService: mockRole,
	})

	_, err := svc.GetUnusedRoles(context.Background(), "schema")
	assert.Error(t, err)
}

func TestRBACManagement_GetRBACSystemStatus_ErrorCounts(t *testing.T) {
	schema := "schema"
	stubTable := &StubTableService{}
	stubTable.GetTableDataFn = func(tableName string, _ dbModels.QueryParams) ([]map[string]interface{}, error) {
		return nil, errors.New("db error")
	}

	svc := services.NewRBACManagementService(&pkg.DatabaseService{TableService: stubTable}, services.RBACManagementServiceDeps{})
	status, err := svc.GetRBACSystemStatus(context.Background(), schema)
	assert.NoError(t, err)
	assert.Equal(t, "not_initialized", status.Status)
}

func TestRBACManagement_InitializeRBACSystem_AssignPermissionError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}

	mockRole := new(MockAccessRoleService)
	mockResource := new(MockResourceService)
	mockAction := new(MockActionService)
	mockPermission := new(MockPermissionService)
	mockRolePerm := new(MockRolePermissionService)
	mockAccessMember := new(MockAccessMemberService)

	mockResource.On("GetOrCreateResource", mock.Anything, "schema", mock.Anything, mock.Anything).
		Return(tenant.Resource{ID: uuid.New()}, nil)
	mockAction.On("GetOrCreateAction", mock.Anything, "schema", mock.Anything, mock.Anything).
		Return(tenant.Action{ID: uuid.New()}, nil)
	mockRole.On("CreateAccessRole", mock.Anything, "schema", mock.Anything).
		Return(tenant.AccessRole{ID: uuid.New()}, nil)
	mockPermission.On("GetOrCreatePermission", mock.Anything, "schema", mock.Anything, mock.Anything).
		Return(tenant.Permission{ID: uuid.New()}, nil)
	mockRolePerm.On("AssignPermissionToRole", mock.Anything, "schema", mock.Anything).
		Return(tenant.RolePermission{}, errors.New("assign error"))

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

func TestRBACManagement_InitializeRBACSystem_RoleCreationError(t *testing.T) {
	mockTable := new(MockTableService)
	repo := &pkg.DatabaseService{TableService: mockTable}

	mockRole := new(MockAccessRoleService)
	mockResource := new(MockResourceService)
	mockAction := new(MockActionService)
	mockPermission := new(MockPermissionService)
	mockRolePerm := new(MockRolePermissionService)
	mockAccessMember := new(MockAccessMemberService)

	mockResource.On("GetOrCreateResource", mock.Anything, "schema", mock.Anything, mock.Anything).
		Return(tenant.Resource{ID: uuid.New()}, nil)
	mockAction.On("GetOrCreateAction", mock.Anything, "schema", mock.Anything, mock.Anything).
		Return(tenant.Action{ID: uuid.New()}, nil)
	mockRole.On("CreateAccessRole", mock.Anything, "schema", mock.Anything).
		Return(tenant.AccessRole{}, errors.New("role create error"))
	mockPermission.On("GetOrCreatePermission", mock.Anything, "schema", mock.Anything, mock.Anything).
		Return(tenant.Permission{}, errors.New("perm create error"))
	mockRolePerm.On("AssignPermissionToRole", mock.Anything, "schema", mock.Anything).
		Return(tenant.RolePermission{}, nil)

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
