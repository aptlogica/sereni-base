package services

import (
	"context"
	"go-postgres-rest/pkg"
	app_errors "serenibase/internal/app-errors"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	common "serenibase/internal/services/common"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/helpers"

	dbModels "go-postgres-rest/pkg/models"

	"github.com/google/uuid"
)

type actionService struct {
	repo *pkg.DatabaseService
}

func NewActionService(repo *pkg.DatabaseService) interfaces.ActionService {
	return &actionService{repo: repo}
}

// Helper functions to reduce duplication
func (s *actionService) getTableName(schemaName string) string {
	return tenant.Action{}.TableName(schemaName)
}

func (s *actionService) mapToAction(data map[string]interface{}) (tenant.Action, error) {
	var action tenant.Action
	if err := helpers.MapToStruct(data, &action); err != nil {
		return tenant.Action{}, app_errors.ErrMapToStruct
	}
	return action, nil
}

func (s *actionService) getSingleRecord(ctx context.Context, schemaName string, query dbModels.QueryParams, errorMsg string) (tenant.Action, error) {
	tableName := s.getTableName(schemaName)
	return common.GetSingleRecordWithRepo[tenant.Action](s.repo, tableName, query, errorMsg)
}

func (s *actionService) CreateAction(ctx context.Context, schemaName string, req dto.ActionDTO) (tenant.Action, error) {
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}

	tableName := s.getTableName(schemaName)
	insertedData, err := s.repo.TableService.CreateRecord(tableName, req.Map())
	if err != nil {
		return tenant.Action{}, err
	}

	return s.mapToAction(insertedData)
}

func (s *actionService) GetActionByID(ctx context.Context, schemaName string, actionID uuid.UUID) (tenant.Action, error) {
	query := common.CreateSingleFilterQuery("id", "eq", actionID.String(), 1)
	return s.getSingleRecord(ctx, schemaName, query, "failed to get action by id")
}

func (s *actionService) GetActionByCode(ctx context.Context, schemaName string, code string) (tenant.Action, error) {
	query := common.CreateSingleFilterQuery("code", "eq", code, 1)
	return s.getSingleRecord(ctx, schemaName, query, "failed to get action by code")
}

func (s *actionService) ListActions(ctx context.Context, schemaName string, limit, offset int) ([]tenant.Action, int64, error) {
	tableName := s.getTableName(schemaName)
	query := dbModels.QueryParams{
		Limit:   &limit,
		Offset:  &offset,
		OrderBy: []string{"code"},
	}

	data, err := s.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return nil, 0, app_errors.LogDatabaseError(err, "failed to list actions")
	}

	count, err := common.CountRecordsWithRepo(s.repo, tableName, "failed to count actions")
	if err != nil {
		return nil, 0, err
	}

	actions, err := common.MapToStructList[tenant.Action](data)
	if err != nil {
		return nil, 0, err
	}
	return actions, count, nil
}

func (s *actionService) UpdateAction(ctx context.Context, schemaName string, actionID uuid.UUID, req dto.ActionDTO) (tenant.Action, error) {
	tableName := s.getTableName(schemaName)
	updateData := req.Map()
	// Remove ID from update data to prevent modifying the primary key
	delete(updateData, "id")

	updatedData, err := s.repo.TableService.UpdateRecord(tableName, actionID, updateData)
	if err != nil {
		return tenant.Action{}, err
	}

	return s.mapToAction(updatedData)
}

func (s *actionService) DeleteAction(ctx context.Context, schemaName string, actionID uuid.UUID) error {
	tableName := s.getTableName(schemaName)
	filter := dbModels.QueryFilter{
		Column:   "id",
		Operator: "eq",
		Value:    actionID.String(),
	}
	return s.repo.TableService.DeleteRecord(tableName, filter)
}

func (s *actionService) GetOrCreateAction(ctx context.Context, schemaName string, code string, description *string) (tenant.Action, error) {
	// Try to get existing action
	action, err := s.GetActionByCode(ctx, schemaName, code)
	if err == nil {
		return action, nil
	}

	// Create new action if not found
	req := dto.ActionDTO{
		ID:          uuid.New(),
		Code:        code,
		Description: description,
	}
	return s.CreateAction(ctx, schemaName, req)
}
