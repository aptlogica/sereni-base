// serenibase/internal/services/view_service.go
// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"github.com/aptlogica/go-postgres-rest/pkg"
	"strings"
	"time"

	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"
)

type viewService struct {
	repo *pkg.DatabaseService
}

func NewViewService(repo *pkg.DatabaseService) interfaces.ViewService {
	return &viewService{repo: repo}
}

// ViewInsertion inserts a new view record into the DB (same flow as ColumnInsertion)
func (s *viewService) Create(ctx context.Context, viewData dto.ViewInsertion, schemaName string) (tenant.View, error) {
	tableName := tenant.View{}.TableName(schemaName)
	s.ensureAuditColumns(ctx, schemaName)

	inserted, err := s.repo.TableService.CreateRecord(tableName, viewData.Map())
	if err != nil {
		return tenant.View{}, app_errors.LogDatabaseError(err, "failed to create view")
	}

	var out tenant.View
	if err := helpers.MapToStruct(inserted, &out); err != nil {
		return tenant.View{}, app_errors.ErrMapToStruct
	}
	return out, nil
}

func (s *viewService) ensureAuditColumns(ctx context.Context, schemaName string) {
	lg := logger.Get()
	tableName := tenant.View{}.TableName(schemaName)
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
				lg.Warn().Err(err).Str("column", col).Str("table", tableName).Msg("Failed to add audit column")
			}
		}
	}
}

func (s *viewService) GetViewByID(ctx context.Context, schemaName, id string) (tenant.View, error) {
	limit := 1
	views, err := s.fetchViews(ctx, schemaName, dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{Column: "id", Operator: "eq", Value: id},
		},
		Limit: &limit,
	})
	if err != nil {
		return tenant.View{}, app_errors.LogDatabaseError(err, "failed to get view by id")
	}
	if len(views) == 0 {
		return tenant.View{}, app_errors.ViewNotFound
	}
	return views[0], nil
}

func (s *viewService) GetAllViews(ctx context.Context, schemaName string) ([]tenant.View, error) {
	return s.fetchViews(ctx, schemaName, dbModels.QueryParams{})
}

// --- shared private helper (same as fetchColumns) ---
func (s *viewService) fetchViews(ctx context.Context, schemaName string, params dbModels.QueryParams) ([]tenant.View, error) {
	tableName := tenant.View{}.TableName(schemaName)
	rows, err := s.repo.TableService.GetTableData(tableName, params)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to fetch views")
	}

	views := make([]tenant.View, 0, len(rows))
	for _, row := range rows {
		var v tenant.View
		if err := helpers.MapToStruct(row, &v); err != nil {
			return nil, app_errors.ErrMapToStruct
		}
		views = append(views, v)
	}
	return views, nil
}

func (s *viewService) UpdateView(ctx context.Context, schemaName string, id string, req dto.ViewUpdate) (tenant.View, error) {
	view := tenant.View{}
	tableName := view.TableName(schemaName)
	s.ensureAuditColumns(ctx, schemaName)

	existing, err := s.GetViewByID(ctx, schemaName, id)
	if err != nil {
		return tenant.View{}, err
	}

	// Prepare update
	update := req.Map()
	if len(update) == 0 {
		return existing, nil
	}
	update["last_modified_time"] = time.Now()

	updatedRows, err := s.repo.TableService.UpdateRecord(tableName, id, update)
	if err != nil {
		return tenant.View{}, app_errors.ViewUploadFailed
	}
	if len(updatedRows) == 0 {
		return tenant.View{}, app_errors.InvalidPayload
	}

	// Return updated
	return s.GetViewByID(ctx, schemaName, id)
}

func (s *viewService) DeleteView(ctx context.Context, schemaName string, id string) error {
	lg := logger.Get()
	view := tenant.View{}
	tableName := view.TableName(schemaName)

	// Ensure exists
	_, err := s.GetViewByID(ctx, schemaName, id)
	if err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to get view for deletion")
		return err
	}

	// Delete
	if err := s.repo.TableService.DeleteRecord(tableName, id); err != nil {
		return app_errors.LogDatabaseError(err, "failed to delete view")
	}
	return nil
}

func (s *viewService) GetViewsByModelID(ctx context.Context, schemaName string, modelID string) ([]tenant.View, error) {
	views, err := s.fetchViews(ctx, schemaName, dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{Column: "model_id", Operator: "eq", Value: modelID},
		},
		OrderBy: []string{"order_index"},
	})
	if err != nil {
		return []tenant.View{}, err
	}
	return views, nil
}
