// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package interfaces

import (
	"context"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
)

type ColumnService interface {
	// CRUD
	Create(ctx context.Context, req dto.ColumnInsertion, schemaName string) (tenant.Column, error)
	GetColumnByID(ctx context.Context, schemaName string, id string) (tenant.Column, error)
	GetColumnByModelID(ctx context.Context, schemaName, modelID string) ([]tenant.Column, error)
	GetAllColumns(ctx context.Context, schemaName string) ([]tenant.Column, error)
	UpdateColumn(ctx context.Context, schemaName string, id string, req dto.ColumnUpdate) (tenant.Column, error)
	DeleteColumn(ctx context.Context, schemaName string, id string) error
	BulkInsert(reqs []dto.ColumnInsertion, schemaName string) ([]tenant.Column, error)
	GetMaxOrderIndexOfColumn(ctx context.Context, schemaName string, modelId string) (float64, error)
}
