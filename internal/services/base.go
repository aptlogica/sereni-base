package services

import (
	"context"
	"go-postgres-rest/pkg"
	"time"

	dbModels "go-postgres-rest/pkg/models"
	// "serenibase/internal/dto"
	app_errors "serenibase/internal/app-errors"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/helpers"

	"github.com/google/uuid"
)

type baseService struct {
	repo *pkg.DatabaseService
}

func NewBaseService(repo *pkg.DatabaseService) interfaces.BaseService {
	return &baseService{repo: repo}
}

func (s *baseService) BaseInsertion(ctx context.Context, req dto.BaseInsertion, schemaName string) (tenant.Base, error) {
	// Set defaults for empty JSON fields
	config := req.Config
	if config == nil || len(config) == 0 {
		config = map[string]interface{}{}
	}

	settings := req.Settings
	if settings == nil || len(settings) == 0 {
		settings = map[string]interface{}{}
	}

	meta := req.Meta
	if meta == nil || len(meta) == 0 {
		meta = map[string]interface{}{}
	}

	// Set defaults for other fields
	baseType := req.Type
	if baseType == "" {
		baseType = "internal"
	}

	status := req.Status
	if status == "" {
		status = "active"
	}

	visibility := req.Visibility
	if visibility == "" {
		visibility = "private"
	}

	// Construct new base record
	baseData := dto.BaseInsertion{
		ID:               uuid.New(),
		WorkspaceID:      req.WorkspaceID,
		Title:            req.Title,
		Description:      req.Description,
		Type:             baseType,
		Config:           config,
		Settings:         settings,
		Meta:             meta,
		Status:           status,
		Visibility:       visibility,
		TableCount:       req.TableCount,
		RowCount:         req.RowCount,
		StorageUsedBytes: req.StorageUsedBytes,
		CreatedBy:        req.CreatedBy,
		UpdatedBy:        req.CreatedBy,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}

	tableName := tenant.Base{}.TableName(schemaName)

	// Insert into DB
	insertedData, err := s.repo.TableService.CreateRecord(tableName, baseData.Map())
	if err != nil {
		return tenant.Base{}, app_errors.LogDatabaseError(err, "failed to create base record")
	}

	// Convert map → struct directly
	var insertedBase tenant.Base
	if err := helpers.MapToStruct(insertedData, &insertedBase); err != nil {
		return tenant.Base{}, app_errors.ErrMapToStruct
	}

	return insertedBase, nil
}

func (s *baseService) CreateBase(ctx context.Context, schemaName string) (tenant.Base, error) {

	base := tenant.Base{}

	if err := s.repo.TableService.CreateTable(base.TableSchema(schemaName)); err != nil {
		return tenant.Base{}, err
	}

	return base, nil
}

func (s *baseService) GetBaseByID(ctx context.Context, schemaName, id string) (tenant.Base, error) {
	if id == "" {
		return tenant.Base{}, app_errors.InvalidPayload
	}

	limit := 1
	bases, err := s.fetchBases(ctx, schemaName, dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{Column: "id", Operator: "eq", Value: id},
		},
		Limit: &limit,
	})
	if err != nil {
		return tenant.Base{}, err
	}
	if len(bases) == 0 {
		return tenant.Base{}, app_errors.BaseNotFound
	}
	return bases[0], nil
}

func (s *baseService) GetAllBases(ctx context.Context, schemaName string) ([]tenant.Base, error) {
	return s.fetchBases(ctx, schemaName, dbModels.QueryParams{})
}

// --- shared private helper ---
func (s *baseService) fetchBases(ctx context.Context, schemaName string, params dbModels.QueryParams) ([]tenant.Base, error) {

	tableName := tenant.Base{}.TableName(schemaName)
	rows, err := s.repo.TableService.GetTableData(tableName, params)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to fetch bases")
	}

	bases := make([]tenant.Base, 0, len(rows))
	for _, row := range rows {
		var b tenant.Base
		if err := helpers.MapToStruct(row, &b); err != nil {
			return nil, app_errors.ErrMapToStruct
		}
		bases = append(bases, b)
	}

	return bases, nil
}

func (s *baseService) UpdateBase(ctx context.Context, schemaName string, id string, req dto.BaseUpdate) (tenant.Base, error) {
	base := tenant.Base{}
	tableName := base.TableName(schemaName)
	// Check if base exists
	existingBase, err := s.GetBaseByID(ctx, schemaName, id)
	if err != nil {
		return tenant.Base{}, app_errors.LogDatabaseError(err, "failed to get base by id for update")
	}
	// Prepare update data
	updateData := req.Map()
	if len(updateData) == 0 {
		return existingBase, nil // Nothing to update
	}
	updateData["last_modified_time"] = time.Now().UTC()
	if req.UpdatedBy != "" {
		updateData["last_modified_by"] = req.UpdatedBy
	}
	// Perform update
	updatedRows, err := s.repo.TableService.UpdateRecord(tableName, id, updateData)
	if err != nil {
		return tenant.Base{}, app_errors.LogDatabaseError(err, "failed to update base")
	}
	if updatedRows == nil || len(updatedRows) == 0 {
		return tenant.Base{}, app_errors.InvalidPayload
	}
	// Return updated base
	return s.GetBaseByID(ctx, schemaName, id)
}

func (s *baseService) DeleteBase(ctx context.Context, schemaName string, id string) error {
	base := tenant.Base{}
	tableName := base.TableName(schemaName)
	// Check if base exists
	_, err := s.GetBaseByID(ctx, schemaName, id)
	if err != nil {
		return err
	}
	// Perform deletion
	if err := s.repo.TableService.DeleteRecord(tableName, id); err != nil {
		return app_errors.LogDatabaseError(err, "failed to delete base")
	}
	return nil
}

func (s *baseService) GetBasesByWorkspace(ctx context.Context, schemaName, workspaceID string) ([]tenant.Base, error) {
	bases, err := s.fetchBases(ctx, schemaName, dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{Column: "workspace_id", Operator: "eq", Value: workspaceID},
		},
	})
	if err != nil {
		return []tenant.Base{}, app_errors.LogDatabaseError(err, "failed to get bases by workspace")
	}
	return bases, nil
}

func (s *baseService) GetBulkbases(ctx context.Context, schemaName string, ids []string) ([]tenant.Base, error) {
	if len(ids) == 0 {
		return []tenant.Base{}, nil
	}

	tableName := tenant.Base{}.TableName(schemaName)

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
		return nil, app_errors.LogDatabaseError(err, "failed to get bulk bases")
	}
	if len(rows) == 0 {
		return []tenant.Base{}, nil
	}

	bases := make([]tenant.Base, 0, len(rows))
	for _, row := range rows {
		var base tenant.Base
		if err := helpers.MapToStruct(row, &base); err != nil {
			return nil, app_errors.ErrMapToStruct
		}
		bases = append(bases, base)
	}

	return bases, nil
}
