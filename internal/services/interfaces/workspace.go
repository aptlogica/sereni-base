// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package interfaces

import (
	"context"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
)

type WorkspaceService interface {
	// CRUD
	CreateWorkspace(ctx context.Context, schemaName string) (tenant.Workspace, error)
	WorkspaceInsertion(ctx context.Context, req dto.CreateWorkspaceRequest, schemaName string) (tenant.Workspace, error)
	GetWorkspaceByID(ctx context.Context, schemaName string, id string) (tenant.Workspace, error)
	GetAllWorkspaces(ctx context.Context, schemaName string) ([]tenant.Workspace, error)
	UpdateWorkspace(ctx context.Context, schemaName string, id string, req dto.WorkspaceUpdate) (tenant.Workspace, error)
	DeleteWorkspace(ctx context.Context, schemaName string, id string) error
	GetBulkWorkspaces(ctx context.Context, schemaName string, ids []string) ([]tenant.Workspace, error)
}
