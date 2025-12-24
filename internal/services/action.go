package services

import (
	"context"
	"godbgrest/pkg"
	app_errors "serenibase/internal/app-errors"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/helpers"

	dbModels "godbgrest/pkg/models"

	"github.com/google/uuid"
)

type actionService struct {
	repo *pkg.DatabaseService
}

func NewActionService(repo *pkg.DatabaseService) interfaces.ActionService {
	return &actionService{repo: repo}
}

func (s *actionService) CreateAction(ctx context.Context, schemaName string, req dto.ActionDTO) (tenant.Action, error) {
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}

	tableName := tenant.Action{}.TableName(schemaName)
	insertedData, err := s.repo.TableService.CreateRecord(ctx, tableName, req.Map())
	if err != nil {
		return tenant.Action{}, err
	}

	var action tenant.Action
	if err := helpers.MapToStruct(insertedData, &action); err != nil {
		return tenant.Action{}, err
	}
	return action, nil
}

func (s *actionService) GetActionByID(ctx context.Context, schemaName string, actionID uuid.UUID) (tenant.Action, error) {
	limit := 1
	tableName := tenant.Action{}.TableName(schemaName)
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "id",
				Operator: "eq",
				Value:    actionID.String(),
			},
		},
		Limit: &limit,
	}

	data, err := s.repo.TableService.GetTableData(ctx, tableName, query)
	if err != nil {
		return tenant.Action{}, app_errors.DatabaseError
	}

	if len(data) == 0 {
		return tenant.Action{}, app_errors.ErrRecordNotFound
	}

	var action tenant.Action
	if err := helpers.MapToStruct(data[0], &action); err != nil {
		return tenant.Action{}, app_errors.ErrMapToStruct
	}
	return action, nil
}

func (s *actionService) GetActionByCode(ctx context.Context, schemaName string, code string) (tenant.Action, error) {
	limit := 1
	tableName := tenant.Action{}.TableName(schemaName)
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "code",
				Operator: "eq",
				Value:    code,
			},
		},
		Limit: &limit,
	}

	data, err := s.repo.TableService.GetTableData(ctx, tableName, query)
	if err != nil {
		return tenant.Action{}, app_errors.DatabaseError
	}

	if len(data) == 0 {
		return tenant.Action{}, app_errors.ErrRecordNotFound
	}

	var action tenant.Action
	if err := helpers.MapToStruct(data[0], &action); err != nil {
		return tenant.Action{}, app_errors.ErrMapToStruct
	}
	return action, nil
}

func (s *actionService) ListActions(ctx context.Context, schemaName string, limit, offset int) ([]tenant.Action, int64, error) {
	tableName := tenant.Action{}.TableName(schemaName)
	query := dbModels.QueryParams{
		Limit:   &limit,
		Offset:  &offset,
		OrderBy: []string{"code"},
	}

	data, err := s.repo.TableService.GetTableData(ctx, tableName, query)
	if err != nil {
		return nil, 0, app_errors.DatabaseError
	}

	countQuery := dbModels.QueryParams{
		Aggregates: []dbModels.AggregateFunction{
			{
				Function: "COUNT",
				Column:   "id",
				Alias:    "total",
			},
		},
	}
	countData, err := s.repo.TableService.GetTableData(ctx, tableName, countQuery)
	if err != nil {
		return nil, 0, app_errors.DatabaseError
	}

	count := int64(0)
	if len(countData) > 0 {
		if total, ok := countData[0]["total"]; ok {
			count = int64(total.(float64))
		}
	}

	var actions []tenant.Action
	for _, item := range data {
		var action tenant.Action
		if err := helpers.MapToStruct(item, &action); err != nil {
			return nil, 0, app_errors.ErrMapToStruct
		}
		actions = append(actions, action)
	}
	return actions, count, nil
}

func (s *actionService) UpdateAction(ctx context.Context, schemaName string, actionID uuid.UUID, req dto.ActionDTO) (tenant.Action, error) {
	tableName := tenant.Action{}.TableName(schemaName)
	updateData := req.Map()
	// Remove ID from update data to prevent modifying the primary key
	delete(updateData, "id")

	updatedData, err := s.repo.TableService.UpdateRecord(ctx, tableName, actionID, updateData)
	if err != nil {
		return tenant.Action{}, err
	}

	var action tenant.Action
	if err := helpers.MapToStruct(updatedData, &action); err != nil {
		return tenant.Action{}, app_errors.ErrMapToStruct
	}
	return action, nil
}

func (s *actionService) DeleteAction(ctx context.Context, schemaName string, actionID uuid.UUID) error {
	tableName := tenant.Action{}.TableName(schemaName)
	filter := dbModels.QueryFilter{
		Column:   "id",
		Operator: "eq",
		Value:    actionID.String(),
	}
	return s.repo.TableService.DeleteRecord(ctx, tableName, filter)
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
