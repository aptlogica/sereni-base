package rbac_test

import (
	"context"
	"errors"
	"testing"

	"go-postgres-rest/pkg"
	app_errors "serenibase/internal/app-errors"
	"serenibase/internal/constant"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	services "serenibase/internal/services/rbac"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func matchScopeID(expected string) interface{} {
	return mock.MatchedBy(func(id *string) bool {
		return id != nil && *id == expected
	})
}

func matchWorkspaceAccessReq(userID, workspaceID, roleID string) interface{} {
	return mock.MatchedBy(func(req dto.AccessMemberDTO) bool {
		if req.UserID != userID || req.ScopeType != constant.ScopeLevels.Workspace || req.RoleID != roleID {
			return false
		}
		return req.ScopeID != nil && *req.ScopeID == workspaceID
	})
}

func matchBaseAccessReq(userID, baseID, workspaceID, roleID string) interface{} {
	return mock.MatchedBy(func(req dto.AccessMemberDTO) bool {
		if req.UserID != userID || req.ScopeType != constant.ScopeLevels.Base || req.RoleID != roleID {
			return false
		}
		if req.ScopeID == nil || *req.ScopeID != baseID {
			return false
		}
		return req.WorkspaceID != nil && *req.WorkspaceID == workspaceID
	})
}

func TestRBACManagement_UpdateRoleForUser(t *testing.T) {
	type updateOps interface {
		UpdateRoleForUser(ctx context.Context, schemaName string, userID, scopeType string, scopeID *string, newRoleID string) error
	}

	svc := services.NewRBACManagementService(&pkg.DatabaseService{}, services.RBACManagementServiceDeps{})
	ops, ok := svc.(updateOps)
	if assert.True(t, ok) {
		err := ops.UpdateRoleForUser(context.Background(), "schema", "user", "workspace", nil, "role")
		assert.ErrorIs(t, err, app_errors.ErrServiceNotInitialized)
	}

	mockAccessMember := new(MockAccessMemberService)
	mockAccessMember.On("UpdateRoleForUser", mock.Anything, "schema", "user", "workspace", matchScopeID("ws"), "role").Return(nil)

	svc = services.NewRBACManagementService(&pkg.DatabaseService{}, services.RBACManagementServiceDeps{
		AccessMemberService: mockAccessMember,
	})
	ops, ok = svc.(updateOps)
	if assert.True(t, ok) {
		err := ops.UpdateRoleForUser(context.Background(), "schema", "user", "workspace", strPtr("ws"), "role")
		assert.NoError(t, err)
	}
}

func TestProcessUserMemberships_Workspace_UpdateExisting(t *testing.T) {
	repo := &pkg.DatabaseService{TableService: &StubTableService{}}
	mockRole := new(MockAccessRoleService)
	mockAccess := new(MockAccessMemberService)

	wsID := "ws"
	newRoleID := uuid.New()
	oldRoleID := uuid.New()
	baseRecordID := uuid.New()
	wsRecordID := uuid.New()

	mockRole.On("GetAccessRoleByName", mock.Anything, "schema", constant.RBACRoleNames.WorkspaceMaintainer).
		Return(tenant.AccessRole{ID: newRoleID, Name: constant.RBACRoleNames.WorkspaceMaintainer}, nil)

	mockAccess.On("GetUserAccessMembers", mock.Anything, "schema", "user").
		Return([]dto.AccessMemberDTO{
			{ID: baseRecordID, ScopeType: constant.ScopeLevels.Base, WorkspaceID: strPtr(wsID), RoleID: uuid.New().String()},
			{ID: wsRecordID, ScopeType: constant.ScopeLevels.Workspace, ScopeID: strPtr(wsID), RoleID: oldRoleID.String()},
		}, nil)

	mockAccess.On("RemoveAccessMemberByID", mock.Anything, "schema", baseRecordID.String()).Return(nil)
	mockAccess.On("UpdateRoleForUser", mock.Anything, "schema", "user", constant.ScopeLevels.Workspace, matchScopeID(wsID), newRoleID.String()).Return(nil)

	svc := services.NewRBACManagementService(repo, services.RBACManagementServiceDeps{
		RoleService:         mockRole,
		AccessMemberService: mockAccess,
	})

	summaryAny, err := svc.ProcessUserMemberships(context.Background(), "schema", "user", "admin", []dto.MembershipRequest{
		{Role: constant.RBACRoleNames.WorkspaceMaintainer, WorkspaceID: wsID},
	})

	assert.NoError(t, err)
	summary := requireSummary(t, summaryAny)
	assert.Equal(t, 1, summary.ProcessedCount)
}

func TestProcessUserMemberships_Workspace_AlreadyHasRole(t *testing.T) {
	repo := &pkg.DatabaseService{TableService: &StubTableService{}}
	mockRole := new(MockAccessRoleService)
	mockAccess := new(MockAccessMemberService)

	wsID := "ws"
	roleID := uuid.New()

	mockRole.On("GetAccessRoleByName", mock.Anything, "schema", constant.RBACRoleNames.WorkspaceMaintainer).
		Return(tenant.AccessRole{ID: roleID, Name: constant.RBACRoleNames.WorkspaceMaintainer}, nil)

	mockAccess.On("GetUserAccessMembers", mock.Anything, "schema", "user").
		Return([]dto.AccessMemberDTO{
			{ID: uuid.New(), ScopeType: constant.ScopeLevels.Workspace, ScopeID: strPtr(wsID), RoleID: roleID.String()},
		}, nil)

	svc := services.NewRBACManagementService(repo, services.RBACManagementServiceDeps{
		RoleService:         mockRole,
		AccessMemberService: mockAccess,
	})

	summaryAny, err := svc.ProcessUserMemberships(context.Background(), "schema", "user", "admin", []dto.MembershipRequest{
		{Role: constant.RBACRoleNames.WorkspaceMaintainer, WorkspaceID: wsID},
	})

	assert.NoError(t, err)
	summary := requireSummary(t, summaryAny)
	assert.Equal(t, 1, summary.SkippedCount)
}

func TestProcessUserMemberships_Workspace_CreateAccess(t *testing.T) {
	repo := &pkg.DatabaseService{TableService: &StubTableService{}}
	mockRole := new(MockAccessRoleService)
	mockAccess := new(MockAccessMemberService)

	wsID := "ws"
	roleID := uuid.New()

	mockRole.On("GetAccessRoleByName", mock.Anything, "schema", constant.RBACRoleNames.WorkspaceMaintainer).
		Return(tenant.AccessRole{ID: roleID, Name: constant.RBACRoleNames.WorkspaceMaintainer}, nil)

	mockAccess.On("GetUserAccessMembers", mock.Anything, "schema", "user").
		Return([]dto.AccessMemberDTO{}, nil)

	mockAccess.On("AssignRoleToUser", mock.Anything, "schema", matchWorkspaceAccessReq("user", wsID, roleID.String())).
		Return(&tenant.AccessMember{ID: uuid.New()}, nil)

	svc := services.NewRBACManagementService(repo, services.RBACManagementServiceDeps{
		RoleService:         mockRole,
		AccessMemberService: mockAccess,
	})

	summaryAny, err := svc.ProcessUserMemberships(context.Background(), "schema", "user", "admin", []dto.MembershipRequest{
		{Role: constant.RBACRoleNames.WorkspaceMaintainer, WorkspaceID: wsID},
	})

	assert.NoError(t, err)
	summary := requireSummary(t, summaryAny)
	assert.Equal(t, 1, summary.ProcessedCount)
	assert.Equal(t, "workspace-level", summary.ProcessedMembers[0].Type)
}

func TestProcessUserMemberships_Workspace_BaseToWorkspaceConversion(t *testing.T) {
	repo := &pkg.DatabaseService{TableService: &StubTableService{}}
	mockRole := new(MockAccessRoleService)
	mockAccess := new(MockAccessMemberService)

	wsID := "ws"
	roleID := uuid.New()
	baseRecordID := uuid.New()

	mockRole.On("GetAccessRoleByName", mock.Anything, "schema", constant.RBACRoleNames.WorkspaceMaintainer).
		Return(tenant.AccessRole{ID: roleID, Name: constant.RBACRoleNames.WorkspaceMaintainer}, nil)

	mockAccess.On("GetUserAccessMembers", mock.Anything, "schema", "user").
		Return([]dto.AccessMemberDTO{
			{ID: baseRecordID, ScopeType: constant.ScopeLevels.Base, WorkspaceID: strPtr(wsID), RoleID: uuid.New().String()},
		}, nil)

	mockAccess.On("RemoveAccessMemberByID", mock.Anything, "schema", baseRecordID.String()).Return(nil)
	mockAccess.On("AssignRoleToUser", mock.Anything, "schema", matchWorkspaceAccessReq("user", wsID, roleID.String())).
		Return(&tenant.AccessMember{ID: uuid.New()}, nil)

	svc := services.NewRBACManagementService(repo, services.RBACManagementServiceDeps{
		RoleService:         mockRole,
		AccessMemberService: mockAccess,
	})

	summaryAny, err := svc.ProcessUserMemberships(context.Background(), "schema", "user", "admin", []dto.MembershipRequest{
		{Role: constant.RBACRoleNames.WorkspaceMaintainer, WorkspaceID: wsID},
	})

	assert.NoError(t, err)
	summary := requireSummary(t, summaryAny)
	assert.Equal(t, 1, summary.ProcessedCount)
	assert.Equal(t, "base-to-workspace-conversion", summary.ProcessedMembers[0].Type)
}

func TestProcessUserMemberships_Workspace_CreateAccessError(t *testing.T) {
	repo := &pkg.DatabaseService{TableService: &StubTableService{}}
	mockRole := new(MockAccessRoleService)
	mockAccess := new(MockAccessMemberService)

	wsID := "ws"
	roleID := uuid.New()

	mockRole.On("GetAccessRoleByName", mock.Anything, "schema", constant.RBACRoleNames.WorkspaceMaintainer).
		Return(tenant.AccessRole{ID: roleID, Name: constant.RBACRoleNames.WorkspaceMaintainer}, nil)

	mockAccess.On("GetUserAccessMembers", mock.Anything, "schema", "user").
		Return([]dto.AccessMemberDTO{}, nil)

	mockAccess.On("AssignRoleToUser", mock.Anything, "schema", matchWorkspaceAccessReq("user", wsID, roleID.String())).
		Return(nil, errors.New("assign failed"))

	svc := services.NewRBACManagementService(repo, services.RBACManagementServiceDeps{
		RoleService:         mockRole,
		AccessMemberService: mockAccess,
	})

	summaryAny, err := svc.ProcessUserMemberships(context.Background(), "schema", "user", "admin", []dto.MembershipRequest{
		{Role: constant.RBACRoleNames.WorkspaceMaintainer, WorkspaceID: wsID},
	})

	assert.NoError(t, err)
	summary := requireSummary(t, summaryAny)
	assert.Equal(t, 1, summary.FailedCount)
}

func TestProcessUserMemberships_Base_AssignAndUpdate(t *testing.T) {
	repo := &pkg.DatabaseService{TableService: &StubTableService{}}
	mockRole := new(MockAccessRoleService)
	mockAccess := new(MockAccessMemberService)
	mockBase := new(MockBaseService)

	wsID := "ws"
	baseID := "base"
	roleID := uuid.New()

	mockRole.On("GetAccessRoleByName", mock.Anything, "schema", "base-member").
		Return(tenant.AccessRole{ID: roleID, Name: "base-member"}, nil)

	mockAccess.On("GetUserAccessByScope", mock.Anything, "schema", "user", constant.ScopeLevels.Workspace, (*string)(nil)).
		Return([]dto.AccessMemberDTO{}, nil)

	mockBase.On("GetBaseByID", mock.Anything, "schema", baseID).
		Return(tenant.Base{ID: uuid.New(), WorkspaceID: wsID}, nil)

	mockAccess.On("GetUserAccessByScope", mock.Anything, "schema", "user", constant.ScopeLevels.Base, matchScopeID(baseID)).
		Return([]dto.AccessMemberDTO{}, nil)

	mockAccess.On("AssignRoleToUser", mock.Anything, "schema", matchBaseAccessReq("user", baseID, wsID, roleID.String())).
		Return(&tenant.AccessMember{ID: uuid.New()}, nil)

	svc := services.NewRBACManagementService(repo, services.RBACManagementServiceDeps{
		RoleService:         mockRole,
		AccessMemberService: mockAccess,
		BaseService:         mockBase,
	})

	summaryAny, err := svc.ProcessUserMemberships(context.Background(), "schema", "user", "admin", []dto.MembershipRequest{
		{Role: "", Bases: []dto.BaseMembership{{BaseID: baseID, Role: "base-member"}}},
	})

	assert.NoError(t, err)
	summary := requireSummary(t, summaryAny)
	assert.Equal(t, 1, summary.ProcessedCount)
	assert.Equal(t, "base-level", summary.ProcessedMembers[0].Type)
}

func TestProcessUserMemberships_Base_UpdateExistingAndSkip(t *testing.T) {
	repo := &pkg.DatabaseService{TableService: &StubTableService{}}
	mockRole := new(MockAccessRoleService)
	mockAccess := new(MockAccessMemberService)
	mockBase := new(MockBaseService)

	wsID := "ws"
	baseID := "base"
	roleID := uuid.New()
	existingID := uuid.New()

	mockRole.On("GetAccessRoleByName", mock.Anything, "schema", "base-member").
		Return(tenant.AccessRole{ID: roleID, Name: "base-member"}, nil)

	mockAccess.On("GetUserAccessByScope", mock.Anything, "schema", "user", constant.ScopeLevels.Workspace, (*string)(nil)).
		Return([]dto.AccessMemberDTO{}, nil)

	mockBase.On("GetBaseByID", mock.Anything, "schema", baseID).
		Return(tenant.Base{ID: uuid.New(), WorkspaceID: wsID}, nil)

	mockAccess.On("GetUserAccessByScope", mock.Anything, "schema", "user", constant.ScopeLevels.Base, matchScopeID(baseID)).
		Return([]dto.AccessMemberDTO{
			{ID: existingID, ScopeType: constant.ScopeLevels.Base, ScopeID: strPtr(baseID), RoleID: uuid.New().String()},
		}, nil)

	mockAccess.On("UpdateRoleForUser", mock.Anything, "schema", "user", constant.ScopeLevels.Base, matchScopeID(baseID), roleID.String()).
		Return(nil)

	svc := services.NewRBACManagementService(repo, services.RBACManagementServiceDeps{
		RoleService:         mockRole,
		AccessMemberService: mockAccess,
		BaseService:         mockBase,
	})

	summaryAny, err := svc.ProcessUserMemberships(context.Background(), "schema", "user", "admin", []dto.MembershipRequest{
		{Role: "", Bases: []dto.BaseMembership{{BaseID: baseID, Role: "base-member"}}},
	})

	assert.NoError(t, err)
	summary := requireSummary(t, summaryAny)
	assert.Equal(t, 1, summary.ProcessedCount)
	assert.Equal(t, "base-level-updated", summary.ProcessedMembers[0].Type)

	// Same role should skip
	mockAccess.ExpectedCalls = nil
	mockAccess.On("GetUserAccessByScope", mock.Anything, "schema", "user", constant.ScopeLevels.Workspace, (*string)(nil)).
		Return([]dto.AccessMemberDTO{}, nil)
	mockAccess.On("GetUserAccessByScope", mock.Anything, "schema", "user", constant.ScopeLevels.Base, matchScopeID(baseID)).
		Return([]dto.AccessMemberDTO{
			{ID: existingID, ScopeType: constant.ScopeLevels.Base, ScopeID: strPtr(baseID), RoleID: roleID.String()},
		}, nil)

	summaryAny, err = svc.ProcessUserMemberships(context.Background(), "schema", "user", "admin", []dto.MembershipRequest{
		{Role: "", Bases: []dto.BaseMembership{{BaseID: baseID, Role: "base-member"}}},
	})

	assert.NoError(t, err)
	summary = requireSummary(t, summaryAny)
	assert.Equal(t, 1, summary.SkippedCount)
}

func TestProcessUserMemberships_Base_ConvertWorkspaceRemovalError(t *testing.T) {
	repo := &pkg.DatabaseService{TableService: &StubTableService{}}
	mockRole := new(MockAccessRoleService)
	mockAccess := new(MockAccessMemberService)
	mockBase := new(MockBaseService)

	wsID := "ws"
	baseID := "base"
	memberID := uuid.New()

	mockRole.On("GetAccessRoleByName", mock.Anything, "schema", "base-member").
		Return(tenant.AccessRole{ID: uuid.New(), Name: "base-member"}, nil)

	mockAccess.On("GetUserAccessByScope", mock.Anything, "schema", "user", constant.ScopeLevels.Workspace, (*string)(nil)).
		Return([]dto.AccessMemberDTO{
			{ID: memberID, ScopeType: constant.ScopeLevels.Workspace, ScopeID: strPtr(wsID)},
		}, nil)

	mockBase.On("GetBaseByID", mock.Anything, "schema", baseID).
		Return(tenant.Base{ID: uuid.New(), WorkspaceID: wsID}, nil)

	mockAccess.On("RemoveAccessMemberByID", mock.Anything, "schema", memberID.String()).
		Return(errors.New("remove failed"))

	svc := services.NewRBACManagementService(repo, services.RBACManagementServiceDeps{
		RoleService:         mockRole,
		AccessMemberService: mockAccess,
		BaseService:         mockBase,
	})

	summaryAny, err := svc.ProcessUserMemberships(context.Background(), "schema", "user", "admin", []dto.MembershipRequest{
		{Role: "", Bases: []dto.BaseMembership{{BaseID: baseID, Role: "base-member"}}},
	})

	assert.NoError(t, err)
	summary := requireSummary(t, summaryAny)
	assert.Equal(t, 1, summary.FailedCount)
}

func TestProcessUserMemberships_Base_SkipAndErrorCases(t *testing.T) {
	repo := &pkg.DatabaseService{TableService: &StubTableService{}}
	mockRole := new(MockAccessRoleService)
	mockAccess := new(MockAccessMemberService)
	mockBase := new(MockBaseService)

	mockAccess.On("GetUserAccessByScope", mock.Anything, "schema", "user", constant.ScopeLevels.Workspace, (*string)(nil)).
		Return([]dto.AccessMemberDTO{}, nil)

	svc := services.NewRBACManagementService(repo, services.RBACManagementServiceDeps{
		RoleService:         mockRole,
		AccessMemberService: mockAccess,
		BaseService:         mockBase,
	})

	// Empty bases should skip
	summaryAny, err := svc.ProcessUserMemberships(context.Background(), "schema", "user", "admin", []dto.MembershipRequest{
		{Role: "", Bases: nil},
	})
	assert.NoError(t, err)
	summary := requireSummary(t, summaryAny)
	assert.Equal(t, 1, summary.SkippedCount)

	// Invalid base role and empty base id should skip
	summaryAny, err = svc.ProcessUserMemberships(context.Background(), "schema", "user", "admin", []dto.MembershipRequest{
		{Role: "", Bases: []dto.BaseMembership{{BaseID: "", Role: "base-member"}, {BaseID: "b1", Role: "invalid"}}},
	})
	assert.NoError(t, err)
	summary = requireSummary(t, summaryAny)
	assert.Equal(t, 2, summary.SkippedCount)
}

func TestProcessUserMemberships_Base_BaseAndRoleLookupErrors(t *testing.T) {
	repo := &pkg.DatabaseService{TableService: &StubTableService{}}
	mockRole := new(MockAccessRoleService)
	mockAccess := new(MockAccessMemberService)
	mockBase := new(MockBaseService)

	mockAccess.On("GetUserAccessByScope", mock.Anything, "schema", "user", constant.ScopeLevels.Workspace, (*string)(nil)).
		Return([]dto.AccessMemberDTO{}, nil)
	mockAccess.On("GetUserAccessByScope", mock.Anything, "schema", "user", constant.ScopeLevels.Base, matchScopeID("base-2")).
		Return([]dto.AccessMemberDTO{}, nil)

	mockBase.On("GetBaseByID", mock.Anything, "schema", "base-1").
		Return(tenant.Base{}, errors.New("base error"))
	mockBase.On("GetBaseByID", mock.Anything, "schema", "base-2").
		Return(tenant.Base{ID: uuid.New(), WorkspaceID: "ws"}, nil)

	mockRole.On("GetAccessRoleByName", mock.Anything, "schema", "base-read").
		Return(tenant.AccessRole{}, errors.New("role error"))

	svc := services.NewRBACManagementService(repo, services.RBACManagementServiceDeps{
		RoleService:         mockRole,
		AccessMemberService: mockAccess,
		BaseService:         mockBase,
	})

	summaryAny, err := svc.ProcessUserMemberships(context.Background(), "schema", "user", "admin", []dto.MembershipRequest{
		{Role: "", Bases: []dto.BaseMembership{
			{BaseID: "base-1", Role: "base-member"},
			{BaseID: "base-2", Role: "base-read"},
		}},
	})

	assert.NoError(t, err)
	summary := requireSummary(t, summaryAny)
	assert.Equal(t, 2, summary.FailedCount)
}

func TestProcessUserMemberships_Base_AssignAndUpdateErrors(t *testing.T) {
	repo := &pkg.DatabaseService{TableService: &StubTableService{}}
	mockRole := new(MockAccessRoleService)
	mockAccess := new(MockAccessMemberService)
	mockBase := new(MockBaseService)

	wsID := "ws"
	baseID := "base"
	roleID := uuid.New()
	existingID := uuid.New()

	mockAccess.On("GetUserAccessByScope", mock.Anything, "schema", "user", constant.ScopeLevels.Workspace, (*string)(nil)).
		Return([]dto.AccessMemberDTO{}, nil)

	mockBase.On("GetBaseByID", mock.Anything, "schema", baseID).
		Return(tenant.Base{ID: uuid.New(), WorkspaceID: wsID}, nil)

	mockRole.On("GetAccessRoleByName", mock.Anything, "schema", "base-member").
		Return(tenant.AccessRole{ID: roleID, Name: "base-member"}, nil)

	mockAccess.On("GetUserAccessByScope", mock.Anything, "schema", "user", constant.ScopeLevels.Base, matchScopeID(baseID)).
		Return([]dto.AccessMemberDTO{
			{ID: existingID, ScopeType: constant.ScopeLevels.Base, ScopeID: strPtr(baseID), RoleID: uuid.New().String()},
		}, nil)

	mockAccess.On("UpdateRoleForUser", mock.Anything, "schema", "user", constant.ScopeLevels.Base, matchScopeID(baseID), roleID.String()).
		Return(errors.New("update failed"))

	svc := services.NewRBACManagementService(repo, services.RBACManagementServiceDeps{
		RoleService:         mockRole,
		AccessMemberService: mockAccess,
		BaseService:         mockBase,
	})

	summaryAny, err := svc.ProcessUserMemberships(context.Background(), "schema", "user", "admin", []dto.MembershipRequest{
		{Role: "", Bases: []dto.BaseMembership{{BaseID: baseID, Role: "base-member"}}},
	})

	assert.NoError(t, err)
	summary := requireSummary(t, summaryAny)
	assert.Equal(t, 1, summary.FailedCount)

	// Assign error when no existing base records
	mockAccess.ExpectedCalls = nil
	mockAccess.On("GetUserAccessByScope", mock.Anything, "schema", "user", constant.ScopeLevels.Workspace, (*string)(nil)).
		Return([]dto.AccessMemberDTO{}, nil)
	mockAccess.On("GetUserAccessByScope", mock.Anything, "schema", "user", constant.ScopeLevels.Base, matchScopeID(baseID)).
		Return([]dto.AccessMemberDTO{}, nil)
	mockAccess.On("AssignRoleToUser", mock.Anything, "schema", matchBaseAccessReq("user", baseID, wsID, roleID.String())).
		Return(nil, errors.New("assign failed"))

	summaryAny, err = svc.ProcessUserMemberships(context.Background(), "schema", "user", "admin", []dto.MembershipRequest{
		{Role: "", Bases: []dto.BaseMembership{{BaseID: baseID, Role: "base-member"}}},
	})

	assert.NoError(t, err)
	summary = requireSummary(t, summaryAny)
	assert.Equal(t, 1, summary.FailedCount)
}

func TestProcessUserMemberships_Workspace_SkipAndRoleError(t *testing.T) {
	repo := &pkg.DatabaseService{TableService: &StubTableService{}}
	mockRole := new(MockAccessRoleService)
	mockAccess := new(MockAccessMemberService)

	mockAccess.On("GetUserAccessMembers", mock.Anything, "schema", "user").
		Return([]dto.AccessMemberDTO{}, nil)

	mockRole.On("GetAccessRoleByName", mock.Anything, "schema", constant.RBACRoleNames.WorkspaceMaintainer).
		Return(tenant.AccessRole{}, errors.New("role error"))

	svc := services.NewRBACManagementService(repo, services.RBACManagementServiceDeps{
		RoleService:         mockRole,
		AccessMemberService: mockAccess,
	})

	summaryAny, err := svc.ProcessUserMemberships(context.Background(), "schema", "user", "admin", []dto.MembershipRequest{
		{Role: constant.RBACRoleNames.WorkspaceMaintainer, WorkspaceID: ""},
		{Role: constant.RBACRoleNames.WorkspaceMaintainer, WorkspaceID: "ws"},
	})

	assert.NoError(t, err)
	summary := requireSummary(t, summaryAny)
	assert.Equal(t, 1, summary.SkippedCount)
	assert.Equal(t, 1, summary.FailedCount)
}

func TestProcessUserMemberships_Workspace_DeleteBaseRecordError(t *testing.T) {
	repo := &pkg.DatabaseService{TableService: &StubTableService{}}
	mockRole := new(MockAccessRoleService)
	mockAccess := new(MockAccessMemberService)

	wsID := "ws"
	roleID := uuid.New()
	baseRecordID := uuid.New()

	mockRole.On("GetAccessRoleByName", mock.Anything, "schema", constant.RBACRoleNames.WorkspaceMaintainer).
		Return(tenant.AccessRole{ID: roleID, Name: constant.RBACRoleNames.WorkspaceMaintainer}, nil)

	mockAccess.On("GetUserAccessMembers", mock.Anything, "schema", "user").
		Return([]dto.AccessMemberDTO{
			{ID: baseRecordID, ScopeType: constant.ScopeLevels.Base, WorkspaceID: strPtr(wsID), RoleID: uuid.New().String()},
		}, nil)

	mockAccess.On("RemoveAccessMemberByID", mock.Anything, "schema", baseRecordID.String()).
		Return(errors.New("remove failed"))
	mockAccess.On("AssignRoleToUser", mock.Anything, "schema", matchWorkspaceAccessReq("user", wsID, roleID.String())).
		Return(&tenant.AccessMember{ID: uuid.New()}, nil)

	svc := services.NewRBACManagementService(repo, services.RBACManagementServiceDeps{
		RoleService:         mockRole,
		AccessMemberService: mockAccess,
	})

	summaryAny, err := svc.ProcessUserMemberships(context.Background(), "schema", "user", "admin", []dto.MembershipRequest{
		{Role: constant.RBACRoleNames.WorkspaceMaintainer, WorkspaceID: wsID},
	})

	assert.NoError(t, err)
	summary := requireSummary(t, summaryAny)
	assert.Equal(t, 1, summary.ProcessedCount)
}

func strPtr(val string) *string {
	return &val
}

func requireSummary(t *testing.T, summaryAny interface{}) *services.MembershipProcessingSummary {
	t.Helper()
	summary, ok := summaryAny.(*services.MembershipProcessingSummary)
	if !assert.True(t, ok) {
		return &services.MembershipProcessingSummary{}
	}
	return summary
}
