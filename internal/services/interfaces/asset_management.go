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

type AssetManagementService interface {
	Upload(ctx context.Context, req dto.UploadAssetRequest, schema string) ([]tenant.Assets, error)
	UploadImage(ctx context.Context, req dto.UploadAssetRequest, schema string) ([]tenant.Assets, error)
	GetBulkAssets(ctx context.Context, schemaName string, ids []string) ([]tenant.Assets, error)
	UpdateAsset(ctx context.Context, assetId string, assetData dto.AssetUpdate, schemaName string) (tenant.Assets, error)
	DeleteAsset(ctx context.Context, assetId string, schemaName string) error

	GetAssetByURL(ctx context.Context, schemaName string, url string) (tenant.Assets, error)
}
