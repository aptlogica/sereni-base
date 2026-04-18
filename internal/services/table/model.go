// Copyright (c) 2026 Aptlogica Technologies Private Limited
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"fmt"

	"github.com/aptlogica/go-postgres-rest/pkg"
	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"
)

const (
	ErrRepositoryNotInitialized = "repository not initialized"
)

type modelService struct {
	repo *pkg.DatabaseService
}

func NewModelService(repo *pkg.DatabaseService) interfaces.ModelService {
	return &modelService{repo: repo}
}

func (s *modelService) CreateModel(ctx context.Context, schemaName string) (tenant.Model, error) {
	if s.repo == nil || s.repo.TableService == nil {
		return tenant.Model{}, fmt.Errorf(ErrRepositoryNotInitialized)
	}

	model := tenant.Model{}

	if err := s.repo.TableService.CreateTable(model.TableSchema(schemaName)); err != nil {
		return tenant.Model{}, err
	}

	s.ensureAuditColumns(ctx, schemaName)

	return model, nil
}

func (s *modelService) ensureAuditColumns(ctx context.Context, schemaName string) {
	tableName := fmt.Sprintf("\"%s\".\"models\"", schemaName)
	columns := []string{"created_by", "last_modified_by"}
	for _, col := range columns {
		req := dbModels.AddColumnRequest{
			Column: dbModels.ColumnDefinition{
				Name:     col,
				DataType: "varchar",
			},
		}
		if err := s.repo.TableService.AddColumn(tableName, req); err != nil {
			// fmt.Printf("DEBUG: Failed to add column %s to %s: %v\n", col, tableName, err)
		}
	}
}

func (s *modelService) Create(ctx context.Context, tableData dto.ModelInsertion, schemaName string) (tenant.Model, error) {
	tableName := tenant.Model{}.TableName(schemaName)
	s.ensureAuditColumns(ctx, schemaName)
	modelData := tableData.Map()
	insertedData, err := s.repo.TableService.CreateRecord(tableName, modelData)
	if err != nil {
		return tenant.Model{}, app_errors.LogDatabaseError(err, "failed to create model")
	}

	var insertedModel tenant.Model
	if err := helpers.MapToStruct(insertedData, &insertedModel); err != nil {
		return tenant.Model{}, app_errors.ErrMapToStruct
	}

	return insertedModel, nil
}

// GetModelByID fetches a single model by its ID
func (s *modelService) GetModelByID(ctx context.Context, schemaName, id string) (tenant.Model, error) {
	limit := 1
	models, err := s.fetchModels(ctx, schemaName, dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{Column: "id", Operator: "eq", Value: id},
		},
		Limit: &limit,
	})
	if err != nil {
		return tenant.Model{}, err
	}
	if len(models) == 0 {
		return tenant.Model{}, app_errors.TableNotFound
	}
	return models[0], nil
}

// GetAllModels fetches all models
func (s *modelService) GetAllModels(ctx context.Context, schemaName string) ([]tenant.Model, error) {
	return s.fetchModels(ctx, schemaName, dbModels.QueryParams{})
}

// --- shared private helper ---
func (s *modelService) fetchModels(ctx context.Context, schemaName string, params dbModels.QueryParams) ([]tenant.Model, error) {
	if s.repo == nil || s.repo.TableService == nil {
		return nil, fmt.Errorf(ErrRepositoryNotInitialized)
	}

	tableName := tenant.Model{}.TableName(schemaName)
	rows, err := s.repo.TableService.GetTableData(tableName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}

	models := make([]tenant.Model, 0, len(rows))
	for _, row := range rows {
		var m tenant.Model
		if err := helpers.MapToStruct(row, &m); err != nil {
			return nil, fmt.Errorf("failed to map model data: %w", err)
		}
		models = append(models, m)
	}

	return models, nil
}

// UpdateModels updates a model by its ID
func (s *modelService) Update(ctx context.Context, schemaName string, id string, req dto.UpdateModelRequest) (tenant.Model, error) {

	// Convert request DTO to map
	updateData := req.Map()
	if len(updateData) == 0 {
		return tenant.Model{}, app_errors.InvalidPayload
	}

	tableName := tenant.Model{}.TableName(schemaName)

	// Update in DB
	updatedRow, err := s.repo.TableService.UpdateRecord(tableName, id, updateData)
	if err != nil {
		return tenant.Model{}, app_errors.LogDatabaseError(err, "failed to update model")
	}

	// Convert map → struct
	var updatedModel tenant.Model
	if err := helpers.MapToStruct(updatedRow, &updatedModel); err != nil {
		return tenant.Model{}, app_errors.ErrMapToStruct
	}

	return updatedModel, nil
}

// DeleteModels deletes a model by its ID
func (s *modelService) DeleteModels(ctx context.Context, schemaName string, id string) error {
	if s.repo == nil || s.repo.TableService == nil {
		return fmt.Errorf(ErrRepositoryNotInitialized)
	}
	if id == "" {
		return fmt.Errorf("model ID cannot be empty")
	}

	tableName := tenant.Model{}.TableName(schemaName)

	// Perform delete
	if err := s.repo.TableService.DeleteRecord(tableName, id); err != nil {
		return fmt.Errorf("failed to delete model: %w", err)
	}

	return nil
}

func (s *modelService) GetModelByBaseID(ctx context.Context, schemaName string, baseID string) ([]tenant.Model, error) {
	models, err := s.fetchModels(ctx, schemaName, dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{Column: "base_id", Operator: "eq", Value: baseID},
		},
		OrderBy: []string{"order_index"},
	})
	if err != nil {
		return []tenant.Model{}, app_errors.LogDatabaseError(err, "failed to get models by base id")
	}
	return models, nil
}

func (s *modelService) GetModelByWorkspaceID(ctx context.Context, schemaName string, workspaceID string) ([]tenant.Model, error) {
	models, err := s.fetchModels(ctx, schemaName, dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{Column: "workspace_id", Operator: "eq", Value: workspaceID},
		},
	})
	if err != nil {
		return []tenant.Model{}, app_errors.LogDatabaseError(err, "failed to get models by workspace id")
	}
	return models, nil
}

func (s *modelService) DeleteModel(ctx context.Context, schemaName string, id string) error {

	tableName := tenant.Model{}.TableName(schemaName)

	if err := s.repo.TableService.DeleteRecord(tableName, id); err != nil {
		return fmt.Errorf("failed to delete model: %w", err)
	}

	return nil
}
