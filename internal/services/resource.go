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

type resourceService struct {
	repo *pkg.DatabaseService
}

func NewResourceService(repo *pkg.DatabaseService) interfaces.ResourceService {
	return &resourceService{repo: repo}
}

func (s *resourceService) CreateResource(ctx context.Context, schemaName string, req dto.ResourceDTO) (tenant.Resource, error) {
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}

	tableName := tenant.Resource{}.TableName(schemaName)
	insertedData, err := s.repo.TableService.CreateRecord(ctx, tableName, req.Map())
	if err != nil {
		return tenant.Resource{}, err
	}

	var resource tenant.Resource
	if err := helpers.MapToStruct(insertedData, &resource); err != nil {
		return tenant.Resource{}, err
	}
	return resource, nil
}

func (s *resourceService) GetResourceByID(ctx context.Context, schemaName string, resourceID uuid.UUID) (tenant.Resource, error) {
	limit := 1
	tableName := tenant.Resource{}.TableName(schemaName)
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "id",
				Operator: "eq",
				Value:    resourceID.String(),
			},
		},
		Limit: &limit,
	}

	data, err := s.repo.TableService.GetTableData(ctx, tableName, query)
	if err != nil {
		return tenant.Resource{}, app_errors.DatabaseError
	}

	if len(data) == 0 {
		return tenant.Resource{}, app_errors.ErrRecordNotFound
	}

	var resource tenant.Resource
	if err := helpers.MapToStruct(data[0], &resource); err != nil {
		return tenant.Resource{}, app_errors.ErrMapToStruct
	}
	return resource, nil
}

func (s *resourceService) GetResourceByCode(ctx context.Context, schemaName string, code string) (tenant.Resource, error) {
	limit := 1
	tableName := tenant.Resource{}.TableName(schemaName)
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
		return tenant.Resource{}, app_errors.DatabaseError
	}

	if len(data) == 0 {
		return tenant.Resource{}, app_errors.ErrRecordNotFound
	}

	var resource tenant.Resource
	if err := helpers.MapToStruct(data[0], &resource); err != nil {
		return tenant.Resource{}, app_errors.ErrMapToStruct
	}
	return resource, nil
}

func (s *resourceService) ListResources(ctx context.Context, schemaName string, limit, offset int) ([]tenant.Resource, int64, error) {
	tableName := tenant.Resource{}.TableName(schemaName)
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

	count := int64(len(countData))
	if len(countData) > 0 {
		if total, ok := countData[0]["total"]; ok {
			if totalVal, ok := total.(float64); ok {
				count = int64(totalVal)
			}
		}
	}

	var resources []tenant.Resource
	for _, item := range data {
		var resource tenant.Resource
		if err := helpers.MapToStruct(item, &resource); err != nil {
			return nil, 0, app_errors.ErrMapToStruct
		}
		resources = append(resources, resource)
	}
	return resources, count, nil
}

func (s *resourceService) UpdateResource(ctx context.Context, schemaName string, resourceID uuid.UUID, req dto.ResourceDTO) (tenant.Resource, error) {
	tableName := tenant.Resource{}.TableName(schemaName)
	updateData := req.Map()
	// Remove ID from update data to prevent modifying the primary key
	delete(updateData, "id")

	updatedData, err := s.repo.TableService.UpdateRecord(ctx, tableName, resourceID, updateData)
	if err != nil {
		return tenant.Resource{}, err
	}

	var resource tenant.Resource
	if err := helpers.MapToStruct(updatedData, &resource); err != nil {
		return tenant.Resource{}, app_errors.ErrMapToStruct
	}
	return resource, nil
}

func (s *resourceService) DeleteResource(ctx context.Context, schemaName string, resourceID uuid.UUID) error {
	tableName := tenant.Resource{}.TableName(schemaName)
	filter := dbModels.QueryFilter{
		Column:   "id",
		Operator: "eq",
		Value:    resourceID.String(),
	}
	return s.repo.TableService.DeleteRecord(ctx, tableName, filter)
}

func (s *resourceService) GetOrCreateResource(ctx context.Context, schemaName string, code string, description *string) (tenant.Resource, error) {
	// Try to get existing resource
	resource, err := s.GetResourceByCode(ctx, schemaName, code)
	if err == nil {
		return resource, nil
	}

	// Create new resource if not found
	req := dto.ResourceDTO{
		ID:          uuid.New(),
		Code:        code,
		Description: description,
	}
	return s.CreateResource(ctx, schemaName, req)
}
