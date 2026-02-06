package workspace_test

import (
	"context"
	"errors"
	"testing"

	appConstant "serenibase/internal/constant"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	services "serenibase/internal/services/workspace"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewWorkspaceManagementService(t *testing.T) {
	db, _ := setupMockDB()

	svc := services.NewWorkspaceManagementService(db, &StubWorkspaceService{}, &StubWorkspaceMemberService{}, &StubBaseManagementService{}, &StubTableManagementService{}, &StubRBACManagementService{})

	assert.NotNil(t, svc)
}

func TestWorkspaceManagementCreate(t *testing.T) {
	t.Run("workspace insertion error", func(t *testing.T) {
		db, _ := setupMockDB()
		workspaceSvc := &StubWorkspaceService{
			WorkspaceInsertionFn: func(ctx context.Context, req dto.CreateWorkspaceRequest, schemaName string) (tenant.Workspace, error) {
				return tenant.Workspace{}, errors.New("fail")
			},
		}

		svc := services.NewWorkspaceManagementService(db, workspaceSvc, &StubWorkspaceMemberService{}, &StubBaseManagementService{}, &StubTableManagementService{}, &StubRBACManagementService{})

		_, err := svc.Create(context.Background(), dto.CreateWorkspaceRequest{Title: "T"}, "schema", "user")

		assert.Error(t, err)
	})

	t.Run("base create error", func(t *testing.T) {
		db, _ := setupMockDB()
		workspaceID := uuid.New()
		workspaceSvc := &StubWorkspaceService{
			WorkspaceInsertionFn: func(ctx context.Context, req dto.CreateWorkspaceRequest, schemaName string) (tenant.Workspace, error) {
				return tenant.Workspace{ID: workspaceID, Title: "T"}, nil
			},
		}
		baseSvc := &StubBaseManagementService{
			CreateBaseFn: func(ctx context.Context, req dto.CreateBaseRequest, schemaName string, userId string) (tenant.Base, error) {
				return tenant.Base{}, errors.New("fail")
			},
		}

		svc := services.NewWorkspaceManagementService(db, workspaceSvc, &StubWorkspaceMemberService{}, baseSvc, &StubTableManagementService{}, &StubRBACManagementService{})

		_, err := svc.Create(context.Background(), dto.CreateWorkspaceRequest{Title: "T"}, "schema", "user")

		assert.Error(t, err)
	})

	t.Run("success sets created_by", func(t *testing.T) {
		db, _ := setupMockDB()
		workspaceID := uuid.New()
		var capturedReq dto.CreateWorkspaceRequest
		workspaceSvc := &StubWorkspaceService{
			WorkspaceInsertionFn: func(ctx context.Context, req dto.CreateWorkspaceRequest, schemaName string) (tenant.Workspace, error) {
				capturedReq = req
				return tenant.Workspace{ID: workspaceID, Title: "T"}, nil
			},
		}
		var capturedBaseReq dto.CreateBaseRequest
		baseSvc := &StubBaseManagementService{
			CreateBaseFn: func(ctx context.Context, req dto.CreateBaseRequest, schemaName string, userId string) (tenant.Base, error) {
				capturedBaseReq = req
				return tenant.Base{}, nil
			},
		}

		svc := services.NewWorkspaceManagementService(db, workspaceSvc, &StubWorkspaceMemberService{}, baseSvc, &StubTableManagementService{}, &StubRBACManagementService{})

		resp, err := svc.Create(context.Background(), dto.CreateWorkspaceRequest{Title: "T"}, "schema", "user")

		assert.NoError(t, err)
		assert.Equal(t, "user", capturedReq.CreatedBy)
		assert.Equal(t, "user", capturedBaseReq.CreatedBy)
		assert.Equal(t, workspaceID, resp.ID)
	})

	t.Run("preserves provided created_by", func(t *testing.T) {
		db, _ := setupMockDB()
		workspaceID := uuid.New()
		var capturedReq dto.CreateWorkspaceRequest
		workspaceSvc := &StubWorkspaceService{
			WorkspaceInsertionFn: func(ctx context.Context, req dto.CreateWorkspaceRequest, schemaName string) (tenant.Workspace, error) {
				capturedReq = req
				return tenant.Workspace{ID: workspaceID, Title: "T"}, nil
			},
		}
		baseSvc := &StubBaseManagementService{
			CreateBaseFn: func(ctx context.Context, req dto.CreateBaseRequest, schemaName string, userId string) (tenant.Base, error) {
				return tenant.Base{}, nil
			},
		}

		svc := services.NewWorkspaceManagementService(db, workspaceSvc, &StubWorkspaceMemberService{}, baseSvc, &StubTableManagementService{}, &StubRBACManagementService{})

		_, err := svc.Create(context.Background(), dto.CreateWorkspaceRequest{Title: "T", CreatedBy: "creator"}, "schema", "user")

		assert.NoError(t, err)
		assert.Equal(t, "creator", capturedReq.CreatedBy)
	})
}

func TestWorkspaceManagementGetByID(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		db, _ := setupMockDB()
		workspaceSvc := &StubWorkspaceService{
			GetWorkspaceByIDFn: func(ctx context.Context, schemaName string, id string) (tenant.Workspace, error) {
				return tenant.Workspace{}, errors.New("fail")
			},
		}

		svc := services.NewWorkspaceManagementService(db, workspaceSvc, &StubWorkspaceMemberService{}, &StubBaseManagementService{}, &StubTableManagementService{}, &StubRBACManagementService{})

		_, err := svc.GetByID(context.Background(), "schema", "id")

		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		db, _ := setupMockDB()
		workspaceID := uuid.New()
		workspaceSvc := &StubWorkspaceService{
			GetWorkspaceByIDFn: func(ctx context.Context, schemaName string, id string) (tenant.Workspace, error) {
				return tenant.Workspace{ID: workspaceID}, nil
			},
		}

		svc := services.NewWorkspaceManagementService(db, workspaceSvc, &StubWorkspaceMemberService{}, &StubBaseManagementService{}, &StubTableManagementService{}, &StubRBACManagementService{})

		ws, err := svc.GetByID(context.Background(), "schema", "id")

		assert.NoError(t, err)
		assert.Equal(t, workspaceID, ws.ID)
	})
}

func TestWorkspaceManagementGetAll(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		db, _ := setupMockDB()
		workspaceSvc := &StubWorkspaceService{
			GetAllWorkspacesFn: func(ctx context.Context, schemaName string) ([]tenant.Workspace, error) {
				return nil, errors.New("fail")
			},
		}

		svc := services.NewWorkspaceManagementService(db, workspaceSvc, &StubWorkspaceMemberService{}, &StubBaseManagementService{}, &StubTableManagementService{}, &StubRBACManagementService{})

		_, err := svc.GetAll(context.Background(), "schema")

		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		db, _ := setupMockDB()
		workspaceID := uuid.New()
		workspaceSvc := &StubWorkspaceService{
			GetAllWorkspacesFn: func(ctx context.Context, schemaName string) ([]tenant.Workspace, error) {
				return []tenant.Workspace{{ID: workspaceID}}, nil
			},
		}

		svc := services.NewWorkspaceManagementService(db, workspaceSvc, &StubWorkspaceMemberService{}, &StubBaseManagementService{}, &StubTableManagementService{}, &StubRBACManagementService{})

		rows, err := svc.GetAll(context.Background(), "schema")

		assert.NoError(t, err)
		assert.Len(t, rows, 1)
	})
}

func TestWorkspaceManagementUpdate(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		db, _ := setupMockDB()
		workspaceSvc := &StubWorkspaceService{
			UpdateWorkspaceFn: func(ctx context.Context, schemaName string, id string, req dto.WorkspaceUpdate) (tenant.Workspace, error) {
				return tenant.Workspace{}, errors.New("fail")
			},
		}

		svc := services.NewWorkspaceManagementService(db, workspaceSvc, &StubWorkspaceMemberService{}, &StubBaseManagementService{}, &StubTableManagementService{}, &StubRBACManagementService{})

		_, err := svc.Update(context.Background(), "schema", "id", dto.WorkspaceUpdate{}, "user")

		assert.Error(t, err)
	})

	t.Run("sets updated_by", func(t *testing.T) {
		db, _ := setupMockDB()
		var captured dto.WorkspaceUpdate
		workspaceSvc := &StubWorkspaceService{
			UpdateWorkspaceFn: func(ctx context.Context, schemaName string, id string, req dto.WorkspaceUpdate) (tenant.Workspace, error) {
				captured = req
				return tenant.Workspace{ID: uuid.New()}, nil
			},
		}

		svc := services.NewWorkspaceManagementService(db, workspaceSvc, &StubWorkspaceMemberService{}, &StubBaseManagementService{}, &StubTableManagementService{}, &StubRBACManagementService{})

		_, err := svc.Update(context.Background(), "schema", "id", dto.WorkspaceUpdate{}, "user")

		assert.NoError(t, err)
		assert.Equal(t, "user", captured.UpdatedBy)
	})
}

func TestWorkspaceManagementDelete(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		db, _ := setupMockDB()
		workspaceSvc := &StubWorkspaceService{
			DeleteWorkspaceFn: func(ctx context.Context, schemaName string, id string) error {
				return errors.New("fail")
			},
		}

		svc := services.NewWorkspaceManagementService(db, workspaceSvc, &StubWorkspaceMemberService{}, &StubBaseManagementService{}, &StubTableManagementService{}, &StubRBACManagementService{})

		err := svc.Delete(context.Background(), "schema", "id")

		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		db, _ := setupMockDB()
		workspaceSvc := &StubWorkspaceService{
			DeleteWorkspaceFn: func(ctx context.Context, schemaName string, id string) error {
				return nil
			},
		}

		svc := services.NewWorkspaceManagementService(db, workspaceSvc, &StubWorkspaceMemberService{}, &StubBaseManagementService{}, &StubTableManagementService{}, &StubRBACManagementService{})

		err := svc.Delete(context.Background(), "schema", "id")

		assert.NoError(t, err)
	})
}

func TestGetTablesByWorkspaceId(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		db, _ := setupMockDB()
		tableSvc := &StubTableManagementService{
			GetModelByWorkspaceIDFn: func(ctx context.Context, schemaName string, workspaceID string) ([]dto.TableResponse, error) {
				return nil, errors.New("fail")
			},
		}

		svc := services.NewWorkspaceManagementService(db, &StubWorkspaceService{}, &StubWorkspaceMemberService{}, &StubBaseManagementService{}, tableSvc, &StubRBACManagementService{})

		_, err := svc.GetTablesByWorkspaceId(context.Background(), "schema", "ws")

		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		db, _ := setupMockDB()
		tableSvc := &StubTableManagementService{
			GetModelByWorkspaceIDFn: func(ctx context.Context, schemaName string, workspaceID string) ([]dto.TableResponse, error) {
				return []dto.TableResponse{{Model: dto.ModelResponse{Title: "T"}}}, nil
			},
		}

		svc := services.NewWorkspaceManagementService(db, &StubWorkspaceService{}, &StubWorkspaceMemberService{}, &StubBaseManagementService{}, tableSvc, &StubRBACManagementService{})

		rows, err := svc.GetTablesByWorkspaceId(context.Background(), "schema", "ws")

		assert.NoError(t, err)
		assert.Len(t, rows, 1)
	})
}

func TestGetBasesByWorkspaceId(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		db, _ := setupMockDB()
		baseSvc := &StubBaseManagementService{
			GetAllBasesWithAccessFn: func(ctx context.Context, schemaName string, workspaceMemberData *tenant.WorkspaceMember) ([]tenant.Base, error) {
				return nil, errors.New("fail")
			},
		}

		svc := services.NewWorkspaceManagementService(db, &StubWorkspaceService{}, &StubWorkspaceMemberService{}, baseSvc, &StubTableManagementService{}, &StubRBACManagementService{})

		_, err := svc.GetBasesByWorkspaceId(context.Background(), "schema", &tenant.WorkspaceMember{})

		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		db, _ := setupMockDB()
		baseSvc := &StubBaseManagementService{
			GetAllBasesWithAccessFn: func(ctx context.Context, schemaName string, workspaceMemberData *tenant.WorkspaceMember) ([]tenant.Base, error) {
				return []tenant.Base{{ID: uuid.New()}}, nil
			},
		}

		svc := services.NewWorkspaceManagementService(db, &StubWorkspaceService{}, &StubWorkspaceMemberService{}, baseSvc, &StubTableManagementService{}, &StubRBACManagementService{})

		rows, err := svc.GetBasesByWorkspaceId(context.Background(), "schema", &tenant.WorkspaceMember{})

		assert.NoError(t, err)
		assert.Len(t, rows, 1)
	})
}

func TestGetAllBasesByWorkspaceId(t *testing.T) {
	t.Run("workspace-level role", func(t *testing.T) {
		db, _ := setupMockDB()
		baseSvc := &StubBaseManagementService{
			GetBasesByWorkspaceFn: func(ctx context.Context, schemaName string, workspaceID string) ([]tenant.Base, error) {
				return []tenant.Base{{ID: uuid.New()}}, nil
			},
		}

		svc := services.NewWorkspaceManagementService(db, &StubWorkspaceService{}, &StubWorkspaceMemberService{}, baseSvc, &StubTableManagementService{}, &StubRBACManagementService{})

		rows, err := svc.GetAllBasesByWorkspaceId(context.Background(), "schema", "ws", appConstant.RBACRoleNames.Owner, "user")

		assert.NoError(t, err)
		assert.Len(t, rows, 1)
		assert.Equal(t, appConstant.RBACRoleNames.Owner, rows[0].AccessLevel)
	})

	t.Run("workspace role base fetch error", func(t *testing.T) {
		db, _ := setupMockDB()
		baseSvc := &StubBaseManagementService{
			GetBasesByWorkspaceFn: func(ctx context.Context, schemaName string, workspaceID string) ([]tenant.Base, error) {
				return nil, errors.New("fail")
			},
		}

		svc := services.NewWorkspaceManagementService(db, &StubWorkspaceService{}, &StubWorkspaceMemberService{}, baseSvc, &StubTableManagementService{}, &StubRBACManagementService{})

		_, err := svc.GetAllBasesByWorkspaceId(context.Background(), "schema", "ws", appConstant.RBACRoleNames.Owner, "user")

		assert.Error(t, err)
	})

	t.Run("access members error", func(t *testing.T) {
		db, _ := setupMockDB()
		rbacSvc := &StubRBACManagementService{
			GetUserAccessMembersFn: func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
				return nil, errors.New("fail")
			},
		}

		svc := services.NewWorkspaceManagementService(db, &StubWorkspaceService{}, &StubWorkspaceMemberService{}, &StubBaseManagementService{}, &StubTableManagementService{}, rbacSvc)

		_, err := svc.GetAllBasesByWorkspaceId(context.Background(), "schema", "ws", "base-member", "user")

		assert.Error(t, err)
	})

	t.Run("workspace access via members", func(t *testing.T) {
		db, _ := setupMockDB()
		workspaceID := "ws"
		scopeID := workspaceID
		roleID := uuid.New()
		rbacSvc := &StubRBACManagementService{
			GetUserAccessMembersFn: func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
				return []dto.AccessMemberDTO{{ScopeType: "workspace", ScopeID: &scopeID, RoleID: roleID.String()}}, nil
			},
			GetRoleByIDFn: func(ctx context.Context, schemaName string, roleID uuid.UUID) (tenant.AccessRole, error) {
				return tenant.AccessRole{Name: appConstant.RBACRoleNames.CoOwner}, nil
			},
		}
		baseSvc := &StubBaseManagementService{
			GetBasesByWorkspaceFn: func(ctx context.Context, schemaName string, workspaceID string) ([]tenant.Base, error) {
				return []tenant.Base{{ID: uuid.New()}}, nil
			},
		}

		svc := services.NewWorkspaceManagementService(db, &StubWorkspaceService{}, &StubWorkspaceMemberService{}, baseSvc, &StubTableManagementService{}, rbacSvc)

		rows, err := svc.GetAllBasesByWorkspaceId(context.Background(), "schema", workspaceID, "base-member", "user")

		assert.NoError(t, err)
		assert.Len(t, rows, 1)
		assert.Equal(t, appConstant.RBACRoleNames.CoOwner, rows[0].AccessLevel)
	})

	t.Run("no base access", func(t *testing.T) {
		db, _ := setupMockDB()
		rbacSvc := &StubRBACManagementService{
			GetUserAccessMembersFn: func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
				return []dto.AccessMemberDTO{{ScopeType: "workspace", ScopeID: nil, RoleID: ""}}, nil
			},
		}

		svc := services.NewWorkspaceManagementService(db, &StubWorkspaceService{}, &StubWorkspaceMemberService{}, &StubBaseManagementService{}, &StubTableManagementService{}, rbacSvc)

		rows, err := svc.GetAllBasesByWorkspaceId(context.Background(), "schema", "ws", "base-member", "user")

		assert.NoError(t, err)
		assert.Empty(t, rows)
	})

	t.Run("base access with role id fallback", func(t *testing.T) {
		db, _ := setupMockDB()
		workspaceID := "ws"
		baseID := uuid.New().String()
		scopeID := baseID
		rbacSvc := &StubRBACManagementService{
			GetUserAccessMembersFn: func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
				return []dto.AccessMemberDTO{{ScopeType: "base", WorkspaceID: &workspaceID, ScopeID: &scopeID, RoleID: "not-a-uuid"}}, nil
			},
		}
		baseSvc := &StubBaseManagementService{
			GetBaseByIDFn: func(ctx context.Context, schemaName string, id string) (tenant.Base, error) {
				if id == baseID {
					return tenant.Base{ID: uuid.MustParse(baseID)}, nil
				}
				return tenant.Base{}, errors.New("not found")
			},
		}

		svc := services.NewWorkspaceManagementService(db, &StubWorkspaceService{}, &StubWorkspaceMemberService{}, baseSvc, &StubTableManagementService{}, rbacSvc)

		rows, err := svc.GetAllBasesByWorkspaceId(context.Background(), "schema", workspaceID, "base-member", "user")

		assert.NoError(t, err)
		assert.Len(t, rows, 1)
		assert.Equal(t, "not-a-uuid", rows[0].AccessLevel)
	})

	t.Run("base access with resolved role name", func(t *testing.T) {
		db, _ := setupMockDB()
		workspaceID := "ws"
		baseID := uuid.New().String()
		scopeID := baseID
		roleID := uuid.New()
		rbacSvc := &StubRBACManagementService{
			GetUserAccessMembersFn: func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
				return []dto.AccessMemberDTO{{ScopeType: "base", WorkspaceID: &workspaceID, ScopeID: &scopeID, RoleID: roleID.String()}}, nil
			},
			GetRoleByIDFn: func(ctx context.Context, schemaName string, roleID uuid.UUID) (tenant.AccessRole, error) {
				return tenant.AccessRole{Name: "role-name"}, nil
			},
		}
		baseSvc := &StubBaseManagementService{
			GetBaseByIDFn: func(ctx context.Context, schemaName string, id string) (tenant.Base, error) {
				return tenant.Base{ID: uuid.MustParse(baseID)}, nil
			},
		}

		svc := services.NewWorkspaceManagementService(db, &StubWorkspaceService{}, &StubWorkspaceMemberService{}, baseSvc, &StubTableManagementService{}, rbacSvc)

		rows, err := svc.GetAllBasesByWorkspaceId(context.Background(), "schema", workspaceID, "base-member", "user")

		assert.NoError(t, err)
		assert.Len(t, rows, 1)
		assert.Equal(t, "role-name", rows[0].AccessLevel)
	})

	t.Run("base access with role lookup error", func(t *testing.T) {
		db, _ := setupMockDB()
		workspaceID := "ws"
		baseID := uuid.New().String()
		scopeID := baseID
		roleID := uuid.New()
		rbacSvc := &StubRBACManagementService{
			GetUserAccessMembersFn: func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
				return []dto.AccessMemberDTO{{ScopeType: "base", WorkspaceID: &workspaceID, ScopeID: &scopeID, RoleID: roleID.String()}}, nil
			},
			GetRoleByIDFn: func(ctx context.Context, schemaName string, roleID uuid.UUID) (tenant.AccessRole, error) {
				return tenant.AccessRole{}, errors.New("fail")
			},
		}
		baseSvc := &StubBaseManagementService{
			GetBaseByIDFn: func(ctx context.Context, schemaName string, id string) (tenant.Base, error) {
				return tenant.Base{ID: uuid.MustParse(baseID)}, nil
			},
		}

		svc := services.NewWorkspaceManagementService(db, &StubWorkspaceService{}, &StubWorkspaceMemberService{}, baseSvc, &StubTableManagementService{}, rbacSvc)

		rows, err := svc.GetAllBasesByWorkspaceId(context.Background(), "schema", workspaceID, "base-member", "user")

		assert.NoError(t, err)
		assert.Len(t, rows, 1)
		assert.Equal(t, roleID.String(), rows[0].AccessLevel)
	})

	t.Run("base access with empty role id", func(t *testing.T) {
		db, _ := setupMockDB()
		workspaceID := "ws"
		baseID := uuid.New().String()
		scopeID := baseID
		rbacSvc := &StubRBACManagementService{
			GetUserAccessMembersFn: func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
				return []dto.AccessMemberDTO{{ScopeType: "base", WorkspaceID: &workspaceID, ScopeID: &scopeID, RoleID: ""}}, nil
			},
		}
		baseSvc := &StubBaseManagementService{
			GetBaseByIDFn: func(ctx context.Context, schemaName string, id string) (tenant.Base, error) {
				return tenant.Base{ID: uuid.MustParse(baseID)}, nil
			},
		}

		svc := services.NewWorkspaceManagementService(db, &StubWorkspaceService{}, &StubWorkspaceMemberService{}, baseSvc, &StubTableManagementService{}, rbacSvc)

		rows, err := svc.GetAllBasesByWorkspaceId(context.Background(), "schema", workspaceID, "base-member", "user")

		assert.NoError(t, err)
		assert.Len(t, rows, 1)
		assert.Equal(t, "", rows[0].AccessLevel)
	})

	t.Run("base access get base error", func(t *testing.T) {
		db, _ := setupMockDB()
		workspaceID := "ws"
		baseID := uuid.New().String()
		scopeID := baseID
		rbacSvc := &StubRBACManagementService{
			GetUserAccessMembersFn: func(ctx context.Context, schemaName string, userID string) ([]dto.AccessMemberDTO, error) {
				return []dto.AccessMemberDTO{{ScopeType: "base", WorkspaceID: &workspaceID, ScopeID: &scopeID, RoleID: "role"}}, nil
			},
			GetRoleByIDFn: func(ctx context.Context, schemaName string, roleID uuid.UUID) (tenant.AccessRole, error) {
				return tenant.AccessRole{Name: "role"}, nil
			},
		}
		baseSvc := &StubBaseManagementService{
			GetBaseByIDFn: func(ctx context.Context, schemaName string, id string) (tenant.Base, error) {
				return tenant.Base{}, errors.New("not found")
			},
		}

		svc := services.NewWorkspaceManagementService(db, &StubWorkspaceService{}, &StubWorkspaceMemberService{}, baseSvc, &StubTableManagementService{}, rbacSvc)

		rows, err := svc.GetAllBasesByWorkspaceId(context.Background(), "schema", workspaceID, "base-member", "user")

		assert.NoError(t, err)
		assert.Empty(t, rows)
	})
}

func TestWorkspaceManagementMembers(t *testing.T) {
	t.Run("remove user error", func(t *testing.T) {
		db, _ := setupMockDB()
		memberSvc := &StubWorkspaceMemberService{
			GetWorkspaceMemberByUserAndWorkspaceFn: func(ctx context.Context, schemaName string, userId string, workspaceId string) (*tenant.WorkspaceMember, error) {
				return nil, errors.New("fail")
			},
		}

		svc := services.NewWorkspaceManagementService(db, &StubWorkspaceService{}, memberSvc, &StubBaseManagementService{}, &StubTableManagementService{}, &StubRBACManagementService{})

		err := svc.RemoveUserFromWorkspace(context.Background(), "schema", "ws", "user")

		assert.Error(t, err)
	})

	t.Run("remove user success", func(t *testing.T) {
		db, _ := setupMockDB()
		id := uuid.New()
		memberSvc := &StubWorkspaceMemberService{
			GetWorkspaceMemberByUserAndWorkspaceFn: func(ctx context.Context, schemaName string, userId string, workspaceId string) (*tenant.WorkspaceMember, error) {
				return &tenant.WorkspaceMember{ID: id}, nil
			},
			DeleteWorkspaceMemberFn: func(ctx context.Context, schemaName string, id string) error {
				return nil
			},
		}

		svc := services.NewWorkspaceManagementService(db, &StubWorkspaceService{}, memberSvc, &StubBaseManagementService{}, &StubTableManagementService{}, &StubRBACManagementService{})

		err := svc.RemoveUserFromWorkspace(context.Background(), "schema", "ws", "user")

		assert.NoError(t, err)
	})

	t.Run("get workspace member by user", func(t *testing.T) {
		db, _ := setupMockDB()
		memberSvc := &StubWorkspaceMemberService{
			GetWorkspaceMemberByUserFn: func(ctx context.Context, schemaName string, userId string) ([]tenant.WorkspaceMember, error) {
				return []tenant.WorkspaceMember{{ID: uuid.New()}}, nil
			},
		}

		svc := services.NewWorkspaceManagementService(db, &StubWorkspaceService{}, memberSvc, &StubBaseManagementService{}, &StubTableManagementService{}, &StubRBACManagementService{})

		rows, err := svc.GetWorkspaceMemberByUser(context.Background(), "schema", "user")

		assert.NoError(t, err)
		assert.Len(t, rows, 1)
	})

	t.Run("get workspace members", func(t *testing.T) {
		db, _ := setupMockDB()
		memberSvc := &StubWorkspaceMemberService{
			GetWorkspaceMembersByWorkspaceFn: func(ctx context.Context, schemaName string, workspaceId string) ([]tenant.WorkspaceMember, error) {
				return []tenant.WorkspaceMember{{ID: uuid.New()}}, nil
			},
		}

		svc := services.NewWorkspaceManagementService(db, &StubWorkspaceService{}, memberSvc, &StubBaseManagementService{}, &StubTableManagementService{}, &StubRBACManagementService{})

		rows, err := svc.GetWorkspaceMembers(context.Background(), "schema", "ws")

		assert.NoError(t, err)
		assert.Len(t, rows, 1)
	})

	t.Run("get bulk workspaces", func(t *testing.T) {
		db, _ := setupMockDB()
		workspaceSvc := &StubWorkspaceService{
			GetBulkWorkspacesFn: func(ctx context.Context, schemaName string, ids []string) ([]tenant.Workspace, error) {
				return []tenant.Workspace{{ID: uuid.New()}}, nil
			},
		}

		svc := services.NewWorkspaceManagementService(db, workspaceSvc, &StubWorkspaceMemberService{}, &StubBaseManagementService{}, &StubTableManagementService{}, &StubRBACManagementService{})

		rows, err := svc.GetBulkWorkspaces(context.Background(), "schema", []string{"id"})

		assert.NoError(t, err)
		assert.Len(t, rows, 1)
	})
}

func TestGetWorkspaceBaseMembers(t *testing.T) {
	t.Run("get base error", func(t *testing.T) {
		db, _ := setupMockDB()
		baseSvc := &StubBaseManagementService{
			GetBaseByIDFn: func(ctx context.Context, schemaName string, id string) (tenant.Base, error) {
				return tenant.Base{}, errors.New("fail")
			},
		}

		svc := services.NewWorkspaceManagementService(db, &StubWorkspaceService{}, &StubWorkspaceMemberService{}, baseSvc, &StubTableManagementService{}, &StubRBACManagementService{})

		_, err := svc.GetWorkspaceBaseMembers(context.Background(), "schema", "base")

		assert.Error(t, err)
	})

	t.Run("function error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		baseSvc := &StubBaseManagementService{
			GetBaseByIDFn: func(ctx context.Context, schemaName string, id string) (tenant.Base, error) {
				return tenant.Base{WorkspaceID: "ws"}, nil
			},
		}

		expectedArgs := map[string]interface{}{"p_schema_name": "schema", "p_workspace_id": "ws", "p_base_id": "base"}
		mockTable.On("GetByFunction", mock.Anything, "public.get_workspace_base_users", expectedArgs).
			Return(nil, errors.New("db error"))

		svc := services.NewWorkspaceManagementService(db, &StubWorkspaceService{}, &StubWorkspaceMemberService{}, baseSvc, &StubTableManagementService{}, &StubRBACManagementService{})

		_, err := svc.GetWorkspaceBaseMembers(context.Background(), "schema", "base")

		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		baseSvc := &StubBaseManagementService{
			GetBaseByIDFn: func(ctx context.Context, schemaName string, id string) (tenant.Base, error) {
				return tenant.Base{WorkspaceID: "ws"}, nil
			},
		}

		expectedArgs := map[string]interface{}{"p_schema_name": "schema", "p_workspace_id": "ws", "p_base_id": "base"}
		mockTable.On("GetByFunction", mock.Anything, "public.get_workspace_base_users", expectedArgs).
			Return([]map[string]interface{}{
				{"get_workspace_base_users": map[string]interface{}{"id": uuid.New().String(), "user_id": "u"}},
				{"get_workspace_base_users": "not-a-map"},
				{"get_workspace_base_users": map[string]interface{}{"id": make(chan int)}},
			}, nil)

		svc := services.NewWorkspaceManagementService(db, &StubWorkspaceService{}, &StubWorkspaceMemberService{}, baseSvc, &StubTableManagementService{}, &StubRBACManagementService{})

		rows, err := svc.GetWorkspaceBaseMembers(context.Background(), "schema", "base")

		assert.NoError(t, err)
		assert.Len(t, rows, 1)
	})
}

func TestWorkspaceManagementDelegates(t *testing.T) {
	t.Run("delete user mappings error", func(t *testing.T) {
		db, _ := setupMockDB()
		memberSvc := &StubWorkspaceMemberService{
			DeleteUserMappingsFn: func(ctx context.Context, schemaName string, userId string) error {
				return errors.New("fail")
			},
		}

		svc := services.NewWorkspaceManagementService(db, &StubWorkspaceService{}, memberSvc, &StubBaseManagementService{}, &StubTableManagementService{}, &StubRBACManagementService{})

		err := svc.DeleteUserMappings(context.Background(), "schema", "user")

		assert.Error(t, err)
	})
	t.Run("delete user mappings", func(t *testing.T) {
		db, _ := setupMockDB()
		memberSvc := &StubWorkspaceMemberService{
			DeleteUserMappingsFn: func(ctx context.Context, schemaName string, userId string) error {
				return nil
			},
		}

		svc := services.NewWorkspaceManagementService(db, &StubWorkspaceService{}, memberSvc, &StubBaseManagementService{}, &StubTableManagementService{}, &StubRBACManagementService{})

		err := svc.DeleteUserMappings(context.Background(), "schema", "user")

		assert.NoError(t, err)
	})

	t.Run("update workspace member bases", func(t *testing.T) {
		db, _ := setupMockDB()
		memberSvc := &StubWorkspaceMemberService{
			UpdateWorkspaceMemberBasesFn: func(ctx context.Context, schemaName string, workspaceId string, userId string, accessLevel string, basesIds string) error {
				return nil
			},
		}

		svc := services.NewWorkspaceManagementService(db, &StubWorkspaceService{}, memberSvc, &StubBaseManagementService{}, &StubTableManagementService{}, &StubRBACManagementService{})

		err := svc.UpdateWorkspaceMemberBases(context.Background(), "schema", "ws", "user", "full_access", "*")

		assert.NoError(t, err)
	})
}
