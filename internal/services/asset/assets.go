// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"github.com/aptlogica/go-postgres-rest/pkg"
	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	"time"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"

	"github.com/aptlogica/sereni-base/internal/services/interfaces"
)

type assetsService struct {
	repo *pkg.DatabaseService
}

func NewAssetsService(repo *pkg.DatabaseService) interfaces.AssetService {
	return &assetsService{repo: repo}
}

func (s *assetsService) getSingleAsset(tableName string, filters []dbModels.QueryFilter, errorMsg string) (tenant.Assets, error) {
	limit := 1
	query := dbModels.QueryParams{
		Select:  []string{"*"},
		Filters: filters,
		Limit:   &limit,
	}

	assetsData, err := s.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return tenant.Assets{}, app_errors.LogDatabaseError(err, errorMsg)
	}

	if len(assetsData) == 0 {
		return tenant.Assets{}, app_errors.InvalidPayload
	}

	var asset tenant.Assets
	if err := helpers.MapToStruct(assetsData[0], &asset); err != nil {
		return tenant.Assets{}, app_errors.ErrMapToStruct
	}
	return asset, nil
}

func (s *assetsService) getAssets(tableName string, params dbModels.QueryParams, errorMsg string) ([]tenant.Assets, error) {
	rows, err := s.repo.TableService.GetTableData(tableName, params)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, errorMsg)
	}

	assets := make([]tenant.Assets, 0, len(rows))
	for _, row := range rows {
		var asset tenant.Assets
		if err := helpers.MapToStruct(row, &asset); err != nil {
			return nil, app_errors.ErrMapToStruct
		}
		assets = append(assets, asset)
	}
	return assets, nil
}

func (s *assetsService) AssetInsertion(ctx context.Context, assetData dto.AssetInsertion, schemaName string) (tenant.Assets, error) {
	tableName := tenant.Assets{}.TableName(schemaName)

	insertedData, err := s.repo.TableService.CreateRecord(tableName, assetData.Map())
	if err != nil {
		return tenant.Assets{}, app_errors.LogDatabaseError(err, "failed to insert asset")
	}

	var insertedAsset tenant.Assets
	if err := helpers.MapToStruct(insertedData, &insertedAsset); err != nil {
		return tenant.Assets{}, app_errors.ErrMapToStruct
	}

	return insertedAsset, nil
}

func (s *assetsService) GetBulkAssets(ctx context.Context, schemaName string, ids []string) ([]tenant.Assets, error) {
	if len(ids) == 0 {
		return []tenant.Assets{}, nil
	}

	tableName := tenant.Assets{}.TableName(schemaName)

	filters := []dbModels.QueryFilter{
		{
			Column:   "id",
			Operator: "in",
			Value:    ids,
		},
	}

	params := dbModels.QueryParams{
		Select:  []string{"*"},
		Filters: filters,
	}

	return s.getAssets(tableName, params, "failed to fetch bulk assets")
}

func (s *assetsService) AssetBulkInsertion(ctx context.Context, assetData []dto.AssetInsertion, schemaName string) ([]tenant.Assets, error) {
	tableName := tenant.Assets{}.TableName(schemaName)

	assetMaps := make([]map[string]interface{}, len(assetData))
	for i, asset := range assetData {
		assetMaps[i] = asset.Map()
	}

	insertedData, err := s.repo.BulkService.BulkInsert(tableName, assetMaps)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to bulk insert assets")
	}

	assets := make([]tenant.Assets, 0, len(insertedData))
	for _, data := range insertedData {
		var asset tenant.Assets
		if err := helpers.MapToStruct(data, &asset); err != nil {
			return nil, app_errors.ErrMapToStruct
		}
		assets = append(assets, asset)
	}

	return assets, nil
}

func (s *assetsService) AssetUpdate(ctx context.Context, assetId string, assetData dto.AssetUpdate, schemaName string) (tenant.Assets, error) {
	tableName := tenant.Assets{}.TableName(schemaName)
	assetDataMap := assetData.Map()

	if assetData.UpdatedAt.IsZero() {
		assetData.UpdatedAt = time.Now()
	}

	if len(assetDataMap) == 0 {
		return tenant.Assets{}, app_errors.InvalidPayload
	}

	updatedData, err := s.repo.TableService.UpdateRecord(tableName, assetId, assetDataMap)
	if err != nil {
		return tenant.Assets{}, app_errors.LogDatabaseError(err, "failed to update asset")
	}

	var asset tenant.Assets
	if err := helpers.MapToStruct(updatedData, &asset); err != nil {
		return tenant.Assets{}, app_errors.ErrMapToStruct
	}

	return asset, nil
}

func (s *assetsService) GetAssetByID(ctx context.Context, id string, schemaName string) (tenant.Assets, error) {
	tableName := tenant.Assets{}.TableName(schemaName)
	filters := []dbModels.QueryFilter{
		{
			Column:   "id",
			Operator: "eq",
			Value:    id,
		},
	}
	return s.getSingleAsset(tableName, filters, "failed to get asset by id")
}

func (s *assetsService) DeleteAsset(ctx context.Context, assetId string, schemaName string) error {
	tableName := tenant.Assets{}.TableName(schemaName)

	err := s.repo.TableService.DeleteRecord(tableName, assetId)
	if err != nil {
		return app_errors.LogDatabaseError(err, "failed to delete asset")
	}

	return nil
}

func (s *assetsService) GetAssetByURL(ctx context.Context, url string, schemaName string) (tenant.Assets, error) {
	tableName := tenant.Assets{}.TableName(schemaName)
	filters := []dbModels.QueryFilter{
		{
			Column:   "url",
			Operator: "eq",
			Value:    url,
		},
	}
	return s.getSingleAsset(tableName, filters, "failed to get asset by url")
}
