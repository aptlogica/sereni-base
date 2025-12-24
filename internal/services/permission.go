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

type permissionService struct {
	repo *pkg.DatabaseService
}

func NewPermissionService(repo *pkg.DatabaseService) interfaces.PermissionService {
	return &permissionService{repo: repo}
}

func (s *permissionService) CreatePermission(ctx context.Context, schemaName string, req dto.PermissionDTO) (tenant.Permission, error) {
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}

	tableName := tenant.Permission{}.TableName(schemaName)
	insertedData, err := s.repo.TableService.CreateRecord(ctx, tableName, req.Map())
	if err != nil {
		return tenant.Permission{}, err
	}

	var permission tenant.Permission
	if err := helpers.MapToStruct(insertedData, &permission); err != nil {
		return tenant.Permission{}, err
	}
	return permission, nil
}

func (s *permissionService) GetPermissionByID(ctx context.Context, schemaName string, permissionID uuid.UUID) (tenant.Permission, error) {
	limit := 1
	tableName := tenant.Permission{}.TableName(schemaName)
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "id",
				Operator: "eq",
				Value:    permissionID.String(),
			},
		},
		Limit: &limit,
	}

	data, err := s.repo.TableService.GetTableData(ctx, tableName, query)
	if err != nil {
		return tenant.Permission{}, app_errors.DatabaseError
	}

	if len(data) == 0 {
		return tenant.Permission{}, app_errors.ErrRecordNotFound
	}

	var permission tenant.Permission
	if err := helpers.MapToStruct(data[0], &permission); err != nil {
		return tenant.Permission{}, app_errors.ErrMapToStruct
	}
	return permission, nil
}

func (s *permissionService) GetPermissionByResourceAndAction(ctx context.Context, schemaName string, resourceID, actionID uuid.UUID) (tenant.Permission, error) {
	limit := 1
	tableName := tenant.Permission{}.TableName(schemaName)
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "resource_id",
				Operator: "eq",
				Value:    resourceID.String(),
			},
			{
				Column:   "action_id",
				Operator: "eq",
				Value:    actionID.String(),
			},
		},
		Limit: &limit,
	}

	data, err := s.repo.TableService.GetTableData(ctx, tableName, query)
	if err != nil {
		return tenant.Permission{}, app_errors.DatabaseError
	}

	if len(data) == 0 {
		return tenant.Permission{}, app_errors.ErrRecordNotFound
	}

	var permission tenant.Permission
	if err := helpers.MapToStruct(data[0], &permission); err != nil {
		return tenant.Permission{}, app_errors.ErrMapToStruct
	}
	return permission, nil
}

func (s *permissionService) ListPermissions(ctx context.Context, schemaName string, limit, offset int) ([]tenant.Permission, int64, error) {
	tableName := tenant.Permission{}.TableName(schemaName)
	query := dbModels.QueryParams{
		Limit:  &limit,
		Offset: &offset,
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

	var permissions []tenant.Permission
	for _, item := range data {
		var permission tenant.Permission
		if err := helpers.MapToStruct(item, &permission); err != nil {
			return nil, 0, app_errors.ErrMapToStruct
		}
		permissions = append(permissions, permission)
	}
	return permissions, count, nil
}

func (s *permissionService) DeletePermission(ctx context.Context, schemaName string, permissionID uuid.UUID) error {
	tableName := tenant.Permission{}.TableName(schemaName)
	filter := dbModels.QueryFilter{
		Column:   "id",
		Operator: "eq",
		Value:    permissionID.String(),
	}
	return s.repo.TableService.DeleteRecord(ctx, tableName, filter)
}

func (s *permissionService) GetOrCreatePermission(ctx context.Context, schemaName string, resourceID, actionID uuid.UUID) (tenant.Permission, error) {
	// Try to get existing permission
	permission, err := s.GetPermissionByResourceAndAction(ctx, schemaName, resourceID, actionID)
	if err == nil {
		return permission, nil
	}

	// Create new permission if not found
	req := dto.PermissionDTO{
		ID:         uuid.New(),
		ResourceID: resourceID,
		ActionID:   actionID,
	}
	return s.CreatePermission(ctx, schemaName, req)
}

func (s *permissionService) GetPermissionsByResource(ctx context.Context, schemaName string, resourceID uuid.UUID) ([]tenant.Permission, error) {
	tableName := tenant.Permission{}.TableName(schemaName)
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "resource_id",
				Operator: "eq",
				Value:    resourceID.String(),
			},
		},
	}

	data, err := s.repo.TableService.GetTableData(ctx, tableName, query)
	if err != nil {
		return nil, app_errors.DatabaseError
	}

	var permissions []tenant.Permission
	for _, item := range data {
		var permission tenant.Permission
		if err := helpers.MapToStruct(item, &permission); err != nil {
			return nil, app_errors.ErrMapToStruct
		}
		permissions = append(permissions, permission)
	}
	return permissions, nil
}
