// serenibase/internal/services/view_service.go
package services

import (
	"context"
	"godbgrest/pkg"
	"time"

	dbModels "godbgrest/pkg/models"
	app_errors "serenibase/internal/app-errors"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	"serenibase/internal/providers/logger"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/helpers"
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

	inserted, err := s.repo.TableService.CreateRecord(ctx, tableName, viewData.Map())
	if err != nil {
		return tenant.View{}, app_errors.DatabaseError
	}

	var out tenant.View
	if err := helpers.MapToStruct(inserted, &out); err != nil {
		return tenant.View{}, app_errors.ErrMapToStruct
	}
	return out, nil
}

func (s *viewService) ensureAuditColumns(ctx context.Context, schemaName string) {
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
			// fmt.Printf("DEBUG: Failed to add column %s to %s: %v\n", col, tableName, err)
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
		return tenant.View{}, app_errors.DatabaseError
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
	rows, err := s.repo.TableService.GetTableData(ctx, tableName, params)
	if err != nil {
		return nil, app_errors.DatabaseError
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

	updatedRows, err := s.repo.TableService.UpdateRecord(ctx, tableName, id, update)
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
	if err := s.repo.TableService.DeleteRecord(ctx, tableName, id); err != nil {
		return app_errors.DatabaseError
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
