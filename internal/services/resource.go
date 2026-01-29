package services

import (
	"context"
	"go-postgres-rest/pkg"
	app_errors "serenibase/internal/app-errors"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/helpers"

	dbModels "go-postgres-rest/pkg/models"

	"github.com/google/uuid"
)

type resourceService struct {
	repo *pkg.DatabaseService
}

func NewResourceService(repo *pkg.DatabaseService) interfaces.ResourceService {
	return &resourceService{repo: repo}
}

// Helper functions to reduce duplication
func (s *resourceService) getTableName(schemaName string) string {
	return tenant.Resource{}.TableName(schemaName)
}

func (s *resourceService) mapToResource(data map[string]interface{}) (tenant.Resource, error) {
	var resource tenant.Resource
	if err := helpers.MapToStruct(data, &resource); err != nil {
		return tenant.Resource{}, app_errors.ErrMapToStruct
	}
	return resource, nil
}

func (s *resourceService) createSingleFilterQuery(column, operator, value string, limit int) dbModels.QueryParams {
	return dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   column,
				Operator: operator,
				Value:    value,
			},
		},
		Limit: &limit,
	}
}

func (s *resourceService) getSingleRecord(ctx context.Context, schemaName string, query dbModels.QueryParams, errorMsg string) (tenant.Resource, error) {
	tableName := s.getTableName(schemaName)
	data, err := s.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return tenant.Resource{}, app_errors.LogDatabaseError(err, errorMsg)
	}

	if len(data) == 0 {
		return tenant.Resource{}, app_errors.ErrRecordNotFound
	}

	return s.mapToResource(data[0])
}

func (s *resourceService) CreateResource(ctx context.Context, schemaName string, req dto.ResourceDTO) (tenant.Resource, error) {
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}

	tableName := s.getTableName(schemaName)
	insertedData, err := s.repo.TableService.CreateRecord(tableName, req.Map())
	if err != nil {
		return tenant.Resource{}, err
	}

	return s.mapToResource(insertedData)
}

func (s *resourceService) GetResourceByID(ctx context.Context, schemaName string, resourceID uuid.UUID) (tenant.Resource, error) {
	query := s.createSingleFilterQuery("id", "eq", resourceID.String(), 1)
	return s.getSingleRecord(ctx, schemaName, query, "failed to get resource by id")
}

func (s *resourceService) GetResourceByCode(ctx context.Context, schemaName string, code string) (tenant.Resource, error) {
	query := s.createSingleFilterQuery("code", "eq", code, 1)
	return s.getSingleRecord(ctx, schemaName, query, "failed to get resource by code")
}

func (s *resourceService) ListResources(ctx context.Context, schemaName string, limit, offset int) ([]tenant.Resource, int64, error) {
	tableName := s.getTableName(schemaName)
	query := dbModels.QueryParams{
		Limit:   &limit,
		Offset:  &offset,
		OrderBy: []string{"code"},
	}

	data, err := s.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return nil, 0, app_errors.LogDatabaseError(err, "failed to list resources")
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
	countData, err := s.repo.TableService.GetTableData(tableName, countQuery)
	if err != nil {
		return nil, 0, app_errors.LogDatabaseError(err, "failed to count resources")
	}

	count := int64(0)
	if len(countData) > 0 {
		if total, ok := countData[0]["total"]; ok {
			count = int64(total.(float64))
		}
	}

	var resources []tenant.Resource
	for _, item := range data {
		resource, err := s.mapToResource(item)
		if err != nil {
			return nil, 0, err
		}
		resources = append(resources, resource)
	}
	return resources, count, nil
}

func (s *resourceService) UpdateResource(ctx context.Context, schemaName string, resourceID uuid.UUID, req dto.ResourceDTO) (tenant.Resource, error) {
	tableName := s.getTableName(schemaName)
	updateData := req.Map()
	// Remove ID from update data to prevent modifying the primary key
	delete(updateData, "id")

	updatedData, err := s.repo.TableService.UpdateRecord(tableName, resourceID, updateData)
	if err != nil {
		return tenant.Resource{}, err
	}

	return s.mapToResource(updatedData)
}

func (s *resourceService) DeleteResource(ctx context.Context, schemaName string, resourceID uuid.UUID) error {
	tableName := s.getTableName(schemaName)
	filter := dbModels.QueryFilter{
		Column:   "id",
		Operator: "eq",
		Value:    resourceID.String(),
	}
	return s.repo.TableService.DeleteRecord(tableName, filter)
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
