// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package interfaces

import (
	"context"

	"github.com/aptlogica/sereni-base/internal/dto"
)

type WorkspaceAPIService interface {
	// Get returns the workspace's API status; a workspace that never turned the API on is returned as disabled
	Get(ctx context.Context, schemaName, workspaceID string) (dto.WorkspaceAPIResponse, error)
	// Enable gives the workspace its permanent public ID if it has none and turns the API on.
	// Enabling an enabled workspace changes nothing.
	Enable(ctx context.Context, schemaName, workspaceID string) (dto.WorkspaceAPIResponse, error)
}
