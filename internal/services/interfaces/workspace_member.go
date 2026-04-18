// Copyright (c) 2026 Aptlogica Technologies Private Limited
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package interfaces

import (
	"context"

	"github.com/aptlogica/sereni-base/internal/models/tenant"
)

type WorkspaceMemberService interface {
	GetAllWorkspaceMembersByUser(ctx context.Context, schemaName string, userId string) ([]tenant.WorkspaceMember, error)
	DeleteWorkspaceMember(ctx context.Context, schemaName string, id string) error
	GetWorkspaceMemberByUserAndWorkspace(ctx context.Context, schemaName string, userId string, workspaceId string) (*tenant.WorkspaceMember, error)
	GetWorkspaceMemberByUser(ctx context.Context, schemaName string, userId string) ([]tenant.WorkspaceMember, error)
	GetWorkspaceMembersByWorkspace(ctx context.Context, schemaName string, workspaceId string) ([]tenant.WorkspaceMember, error)
	DeleteUserMappings(ctx context.Context, schemaName string, userId string) error
	UpdateWorkspaceMemberBases(ctx context.Context, schemaName string, workspaceId string, userId string, accessLevel string, basesIds string) error
}
