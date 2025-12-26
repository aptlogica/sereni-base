package interfaces

import (
	"context"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
)

type WorkspaceManagementService interface {
	Create(ctx context.Context, req dto.CreateWorkspaceRequest, schemaName string, userId string) (dto.WorkspaceResponse, error)
	GetByID(ctx context.Context, schemaName string, id string) (tenant.Workspace, error)
	GetAll(ctx context.Context, schemaName string) ([]tenant.Workspace, error)
	Update(ctx context.Context, schemaName string, id string, req dto.WorkspaceUpdate, userId string) (tenant.Workspace, error)
	Delete(ctx context.Context, schemaName string, id string) error
	GetTablesByWorkspaceId(ctx context.Context, schemaName string, workspaceID string) ([]dto.TableResponse, error)
	GetBasesByWorkspaceId(ctx context.Context, schemaName string, workspaceMemberData *tenant.WorkspaceMember) ([]tenant.Base, error)
	GetAllBasesByWorkspaceId(ctx context.Context, schemaName string, workspaceID string) ([]tenant.Base, error)
	GetWorkspaceMemberByUser(ctx context.Context, schemaName string, userID string) ([]tenant.WorkspaceMember, error)
	GetWorkspaceMembers(ctx context.Context, schemaName string, workspaceID string) ([]tenant.WorkspaceMember, error)
	GetBulkWorkspaces(ctx context.Context, schemaName string, workspaceIDs []string) ([]tenant.Workspace, error)
	GetWorkspaceBaseMembers(ctx context.Context, schemaName string, baseID string) ([]tenant.WorkspaceMember, error)
	DeleteUserMappings(ctx context.Context, schemaName string, userID string) error
	UpdateWorkspaceMemberBases(ctx context.Context, schemaName string, workspaceID string, userID string, accessLevel string, basesIds string) error
}
