// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aptlogica/go-postgres-rest/pkg"

	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"

	"github.com/google/uuid"
)

type columnService struct {
	repo *pkg.DatabaseService
}

func NewColumnService(repo *pkg.DatabaseService) interfaces.ColumnService {
	return &columnService{repo: repo}
}

// ColumnInsertion inserts a new column record into the DB
func (s *columnService) Create(ctx context.Context, req dto.ColumnInsertion, schemaName string) (tenant.Column, error) {

	// validationRules := req.ValidationRules
	// if validationRules == nil || len(validationRules) == 0 {
	// 	validationRules = json.RawMessage("{}")
	// }

	// Construct new column record
	colData := dto.ColumnInsertion{
		ID:          uuid.New(),
		ModelID:     req.ModelID,
		BaseID:      req.BaseID,
		ColumnName:  req.ColumnName,
		Title:       req.Title,
		UIDT:        req.UIDT,
		DT:          req.DT,
		Description: req.Description,
		Meta:        req.Meta,
		// PK:               req.PK,
		// PV:               req.PV,
		// RQD:              req.RQD,
		// UN:               req.UN,
		// AI:               req.AI,
		// UniqueConstraint: req.UniqueConstraint,
		// MaxLength:        req.MaxLength,
		// PrecisionValue:   req.PrecisionValue,
		// ScaleValue:       req.ScaleValue,
		// DefaultValue:     req.DefaultValue,
		// ValidationRules:  validationRules,
		Virtual:    req.Virtual,
		System:     req.System,
		Deleted:    req.Deleted,
		OrderIndex: req.OrderIndex,
		CreatedBy:  req.CreatedBy,
		UpdatedBy:  req.CreatedBy,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	tableName := tenant.Column{}.TableName(schemaName)
	s.ensureAuditColumns(ctx, schemaName)

	// Insert into DB
	insertedData, err := s.repo.TableService.CreateRecord(tableName, colData.Map())
	if err != nil {
		return tenant.Column{}, app_errors.LogDatabaseError(err, "failed to create column")
	}

	// Convert map → struct
	var insertedCol tenant.Column
	if err := helpers.MapToStruct(insertedData, &insertedCol); err != nil {
		return tenant.Column{}, app_errors.ErrMapToStruct
	}

	return insertedCol, nil
}

func (s *columnService) ensureAuditColumns(ctx context.Context, schemaName string) {
	tableName := fmt.Sprintf("\"%s\".\"columns\"", schemaName)
	columns := []string{"created_by", "last_modified_by"}
	for _, col := range columns {
		req := dbModels.AddColumnRequest{
			Column: dbModels.ColumnDefinition{
				Name:     col,
				DataType: "varchar",
			},
		}
		if err := s.repo.TableService.AddColumn(tableName, req); err != nil {
			// Silently ignore "already exists" errors as columns are defined in TableSchema
			errMsg := err.Error()
			if !strings.Contains(errMsg, "already exists") {
				fmt.Printf("DEBUG: Failed to add column %s to %s: %v\n", col, tableName, err)
			}
		}
	}
}

// CreateColumn ensures the columns table exists in schema
func (s *columnService) CreateColumn(ctx context.Context, schemaName string) (tenant.Column, error) {
	col := tenant.Column{}

	if err := s.repo.TableService.CreateTable(col.TableSchema(schemaName)); err != nil {
		return tenant.Column{}, err
	}

	s.ensureAuditColumns(ctx, schemaName)

	return col, nil
}

func (s *columnService) GetColumnByID(ctx context.Context, schemaName, id string) (tenant.Column, error) {
	limit := 1
	cols, err := s.fetchColumns(ctx, schemaName, dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{Column: "id", Operator: "eq", Value: id},
		},
		Limit: &limit,
	})
	if err != nil {
		return tenant.Column{}, err
	}
	if len(cols) == 0 {
		return tenant.Column{}, app_errors.ColumnNotFound
	}
	return cols[0], nil
}

func (s *columnService) GetAllColumns(ctx context.Context, schemaName string) ([]tenant.Column, error) {
	return s.fetchColumns(ctx, schemaName, dbModels.QueryParams{})
}

// --- shared private helper ---
func (s *columnService) fetchColumns(ctx context.Context, schemaName string, params dbModels.QueryParams) ([]tenant.Column, error) {
	tableName := tenant.Column{}.TableName(schemaName)
	rows, err := s.repo.TableService.GetTableData(tableName, params)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to fetch columns")
	}

	cols := make([]tenant.Column, 0, len(rows))
	for _, row := range rows {
		var c tenant.Column
		if err := helpers.MapToStruct(row, &c); err != nil {
			return nil, app_errors.ErrMapToStruct
		}
		cols = append(cols, c)
	}

	return cols, nil
}

func (s *columnService) UpdateColumn(ctx context.Context, schemaName string, id string, req dto.ColumnUpdate) (tenant.Column, error) {
	col := tenant.Column{}
	tableName := col.TableName(schemaName)

	// Check if column exists
	existingCol, err := s.GetColumnByID(ctx, schemaName, id)
	if err != nil {
		return tenant.Column{}, err
	}

	// Prepare update data
	updateData := req.Map()
	if len(updateData) == 0 {
		return existingCol, nil // Nothing to update
	}
	updateData["last_modified_time"] = time.Now()

	// Perform update
	updatedRows, err := s.repo.TableService.UpdateRecord(tableName, id, updateData)
	if err != nil {
		return tenant.Column{}, app_errors.ColumnUpdateFailed
	}
	if updatedRows == nil || len(updatedRows) == 0 {
		return tenant.Column{}, app_errors.InvalidPayload
	}

	// Return updated column
	return s.GetColumnByID(ctx, schemaName, id)
}

func (s *columnService) DeleteColumn(ctx context.Context, schemaName string, id string) error {
	col := tenant.Column{}
	tableName := col.TableName(schemaName)

	// Check if column exists
	_, err := s.GetColumnByID(ctx, schemaName, id)
	if err != nil {
		return err
	}

	// Perform deletion
	if err := s.repo.TableService.DeleteRecord(tableName, id); err != nil {
		return app_errors.LogDatabaseError(err, "failed to delete column")
	}
	return nil
}

func (s *columnService) GetColumnByModelID(ctx context.Context, schemaName, modelID string) ([]tenant.Column, error) {
	cols, err := s.fetchColumns(ctx, schemaName, dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{Column: "model_id", Operator: "eq", Value: modelID},
		},
		OrderBy: []string{"order_index"},
	})
	if err != nil {
		return []tenant.Column{}, err
	}
	return cols, nil
}

func (s *columnService) BulkInsert(colDataList []dto.ColumnInsertion, schemaName string) ([]tenant.Column, error) {
	tableName := tenant.Column{}.TableName(schemaName)

	var colDataMaps []map[string]interface{}
	for _, col := range colDataList {
		colDataMaps = append(colDataMaps, col.Map())
	}

	insertedRows, err := s.repo.BulkService.BulkInsert(tableName, colDataMaps)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to bulk insert columns")
	}

	cols := make([]tenant.Column, 0, len(insertedRows))
	for _, row := range insertedRows {
		var c tenant.Column
		if err := helpers.MapToStruct(row, &c); err != nil {
			return nil, app_errors.ErrMapToStruct
		}
		cols = append(cols, c)
	}

	return cols, nil
}

func (s *columnService) GetMaxOrderIndex(ctx context.Context, schemaName, modelID string) ([]tenant.Column, error) {
	cols, err := s.fetchColumns(ctx, schemaName, dbModels.QueryParams{
		// Aggregates: []dbModels.AggregateFunction{}{

		// },
		Filters: []dbModels.QueryFilter{
			{Column: "model_id", Operator: "eq", Value: modelID},
		},
	})
	if err != nil {
		return []tenant.Column{}, err
	}
	return cols, nil
}

func (s *columnService) GetMaxOrderIndexOfColumn(ctx context.Context, schemaName string, modelId string) (float64, error) {
	tableName := tenant.Column{}.TableName(schemaName)
	params := dbModels.QueryParams{
		Aggregates: []dbModels.AggregateFunction{{Function: "MAX", Column: "order_index"}},
		Filters:    []dbModels.QueryFilter{{Column: "model_id", Operator: "eq", Value: modelId}},
	}
	data, err := s.repo.TableService.GetTableData(tableName, params)
	if err != nil || len(data) == 0 {
		return 0, err
	}
	if v, ok := data[0]["max"]; ok && v != nil {
		switch n := v.(type) {
		case int:
			return float64(n), nil
		case int64:
			return float64(n), nil
		case float64:
			return n, nil
		case float32:
			return float64(n), nil
		}
	}
	return 0, nil
}

func (s *columnService) BulkUpdate(ctx context.Context, schemaName string, tableName string, columnName string, updates []dto.UpdateColumnsRequest) error {
	if len(updates) == 0 {
		return nil
	}

	functionName := "bulk_update"
	schemaFunctionName := fmt.Sprintf("%s.%s", schemaName, functionName)

	// Convert updates to JSON for JSONB parameter
	jsonData, err := json.Marshal(updates)
	if err != nil {
		return app_errors.LogDatabaseError(err, "failed to marshal updates to JSON")
	}

	args := map[string]interface{}{
		"p_schema_name": schemaName,
		"p_table_name":  tableName,
		"p_column_name": columnName,
		"p_data":        jsonData,
	}

	// Execute the bulk_update function
	_, err = s.repo.TableService.GetByFunction(
		ctx,
		schemaFunctionName,
		args,
	)
	if err != nil {
		fmt.Printf("DEBUG: Bulk update failed for column %s in table %s: %v\n", columnName, tableName, err)
		return app_errors.LogDatabaseError(err, "failed to perform bulk update on columns")
	}

	return nil
}

func (s *columnService) BulkUpdateByColumns(ctx context.Context, schemaName string, tableName string, updates []dto.UpdateColumnValueRequest) error {
	if len(updates) == 0 {
		return nil
	}

	functionName := "bulk_update_by_columns"
	schemaFunctionName := fmt.Sprintf("%s.%s", schemaName, functionName)

	jsonData, err := json.Marshal(updates)
	if err != nil {
		return app_errors.LogDatabaseError(err, "failed to marshal multi-column updates to JSON")
	}

	args := map[string]interface{}{
		"p_schema_name": schemaName,
		"p_table_name":  tableName,
		"p_data":        jsonData,
	}

	_, err = s.repo.TableService.GetByFunction(
		ctx,
		schemaFunctionName,
		args,
	)
	if err != nil {
		return app_errors.LogDatabaseError(err, "failed to perform multi-column bulk update")
	}

	return nil
}

func (s *columnService) ResetColumn(ctx context.Context, schemaName string, tableName string, columnName string) error {
	functionName := "reset_column"
	schemaFunctionName := fmt.Sprintf("%s.%s", schemaName, functionName)

	// Execute the reset_column function
	args := map[string]interface{}{
		"p_schema_name": schemaName,
		"p_table_name":  tableName,
		"p_column_name": columnName,
	}

	_, err := s.repo.TableService.GetByFunction(
		ctx,
		schemaFunctionName,
		args,
	)
	if err != nil {
		return app_errors.LogDatabaseError(err, "failed to reset column values")
	}

	return nil
}
