package services

import (
	"context"
	"go-postgres-rest/pkg"
	dbModels "go-postgres-rest/pkg/models"
	"time"

	app_errors "serenibase/internal/app-errors"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	"serenibase/internal/utils/helpers"

	"serenibase/internal/services/interfaces"
)

type assetsService struct {
	repo *pkg.DatabaseService
}

func NewAssetsService(repo *pkg.DatabaseService) interfaces.AssetService {
	return &assetsService{repo: repo}
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

	rows, err := s.repo.TableService.GetTableData(tableName, params)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to fetch bulk assets")
	}
	if len(rows) == 0 {
		return []tenant.Assets{}, nil
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
	limit := 1
	query := dbModels.QueryParams{
		Select: []string{"*"},
		Filters: []dbModels.QueryFilter{
			{
				Column:   "id",
				Operator: "eq",
				Value:    id,
			},
		},
		Limit: &limit,
	}

	assetsData, err := s.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return tenant.Assets{}, app_errors.LogDatabaseError(err, "failed to get asset by id")
	}

	if len(assetsData) == 0 {
		return tenant.Assets{}, app_errors.InvalidPayload
	}

	assetData := assetsData[0]

	var asset tenant.Assets
	if err := helpers.MapToStruct(assetData, &asset); err != nil {
		return tenant.Assets{}, app_errors.ErrMapToStruct
	}
	return asset, nil
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
	limit := 1
	query := dbModels.QueryParams{
		Select: []string{"*"},
		Filters: []dbModels.QueryFilter{
			{
				Column:   "url",
				Operator: "eq",
				Value:    url,
			},
		},
		Limit: &limit,
	}

	assetsData, err := s.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return tenant.Assets{}, app_errors.LogDatabaseError(err, "failed to get asset by url")
	}

	if len(assetsData) == 0 {
		return tenant.Assets{}, app_errors.InvalidPayload
	}

	assetData := assetsData[0]

	var asset tenant.Assets
	if err := helpers.MapToStruct(assetData, &asset); err != nil {
		return tenant.Assets{}, app_errors.ErrMapToStruct
	}
	return asset, nil
}
