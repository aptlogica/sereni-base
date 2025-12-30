package interfaces

import (
	"context"
	"mime/multipart"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
)

type BaseManagementService interface {
	// CreateBase(ctx context.Context, schemaName string) (tenant.Base, error)
	CreateBase(ctx context.Context, req dto.CreateBaseRequest, schemaName string, userId string) (tenant.Base, error)
	CreateBaseWithImage(ctx context.Context, req dto.CreateBaseRequest, schemaName string, userId string, fileHeader *multipart.FileHeader) (tenant.Base, error)
	GetBaseByID(ctx context.Context, schemaName string, id string) (tenant.Base, error)
	GetAllBasesWithAccess(ctx context.Context, schemaName string, workspaceMemberData *tenant.WorkspaceMember) ([]tenant.Base, error)
	UpdateBase(ctx context.Context, schemaName string, id string, req dto.BaseUpdate, userId string) (tenant.Base, error)
	DeleteBase(ctx context.Context, schemaName string, id string) error
	GetTablesByBaseId(ctx context.Context, schemaName string, baseID string) ([]dto.TableResponse, error)
	GetBasesByWorkspace(ctx context.Context, schemaName string, workspaceID string) ([]tenant.Base, error)
	AddBaseImage(ctx context.Context, schema string, baseID string, fileHeader *multipart.FileHeader, userId string) (tenant.Base, error)
	RemoveBaseImage(ctx context.Context, schema string, baseID string, userId string) (tenant.Base, error)
}
