package interfaces

import (
	"context"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
)

type WorkspaceMemberService interface {
	CreateWorkspaceMember(ctx context.Context, workspaceMemberReq *dto.CreateMemberRequest, schemaName string) error
	GetAllWorkspaceMembersByUser(ctx context.Context, schemaName string, userId string) ([]tenant.WorkspaceMember, error)
	DeleteWorkspaceMember(ctx context.Context, schemaName string, id string) error
	GetWorkspaceMemberByUserAndWorkspace(ctx context.Context, schemaName string, userId string, workspaceId string) (*tenant.WorkspaceMember, error)
	GetWorkspaceMemberByUser(ctx context.Context, schemaName string, userId string) ([]tenant.WorkspaceMember, error)
	GetWorkspaceMembersByWorkspace(ctx context.Context, schemaName string, workspaceId string) ([]tenant.WorkspaceMember, error)
	DeleteUserMappings(ctx context.Context, schemaName string, userId string) error
	UpdateWorkspaceMemberBases(ctx context.Context, schemaName string, workspaceId string, userId string, accessLevel string, basesIds string) error
}
