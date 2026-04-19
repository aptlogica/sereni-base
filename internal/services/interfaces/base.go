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
