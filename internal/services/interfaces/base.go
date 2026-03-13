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

type BaseService interface {
	// CRUD
	CreateBase(ctx context.Context, schemaName string) (tenant.Base, error)
	BaseInsertion(ctx context.Context, req dto.BaseInsertion, schemaName string) (tenant.Base, error)
	GetBaseByID(ctx context.Context, schemaName string, id string) (tenant.Base, error)
	GetAllBases(ctx context.Context, schemaName string) ([]tenant.Base, error)
	UpdateBase(ctx context.Context, schemaName string, id string, req dto.BaseUpdate) (tenant.Base, error)
	DeleteBase(ctx context.Context, schemaName string, id string) error
	GetBasesByWorkspace(ctx context.Context, schemaName, workspaceID string) ([]tenant.Base, error)
	GetBulkbases(ctx context.Context, schemaName string, ids []string) ([]tenant.Base, error)
}
