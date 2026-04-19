// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package interfaces

import (
	"context"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
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
