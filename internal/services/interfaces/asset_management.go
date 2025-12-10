package interfaces

import (
	"context"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
)

type AssetManagementService interface {
	Upload(ctx context.Context, req dto.UploadAssetRequest, schema string) ([]tenant.Assets, error)
	UploadImage(ctx context.Context, req dto.UploadAssetRequest, schema string) ([]tenant.Assets, error)
	GetBulkAssets(ctx context.Context, schemaName string, ids []string) ([]tenant.Assets, error)
	UpdateAsset(ctx context.Context, assetId string, assetData dto.AssetUpdate, schemaName string) (tenant.Assets, error)
	DeleteAsset(ctx context.Context, assetId string, schemaName string) error

	GetAssetByURL(ctx context.Context, schemaName string, url string) (tenant.Assets, error)
}
