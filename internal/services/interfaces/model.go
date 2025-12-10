package interfaces

import (
	"context"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
)

type ModelService interface {
	Create(ctx context.Context, tableData dto.ModelInsertion, schemaName string) (tenant.Model, error)
	GetModelByID(ctx context.Context, schemaName string, id string) (tenant.Model, error)
	GetAllModels(ctx context.Context, schemaName string) ([]tenant.Model, error)
	Update(ctx context.Context, schemaName string, id string, req dto.UpdateModelRequest) (tenant.Model, error)
	DeleteModels(ctx context.Context, schemaName string, id string) error
	GetModelByBaseID(ctx context.Context, schemaName string, base_id string) ([]tenant.Model, error)
	GetModelByWorkspaceID(ctx context.Context, schemaName string, workspace_id string) ([]tenant.Model, error)
	DeleteModel(ctx context.Context, schemaName string, id string) error
}
