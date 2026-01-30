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

func (s *resourceService) CreateResource(ctx context.Context, schemaName string, req dto.ResourceDTO) (tenant.Resource, error) {
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}

	tableName := s.getTableName(schemaName)
	insertedData, err := s.repo.TableService.CreateRecord(tableName, req.Map())
	if err != nil {
		return tenant.Resource{}, err
	}

	var resource tenant.Resource
	if err := helpers.MapToStruct(insertedData, &resource); err != nil {
		return tenant.Resource{}, app_errors.ErrMapToStruct
	}
	return resource, nil
}

func (s *resourceService) GetResourceByID(ctx context.Context, schemaName string, resourceID uuid.UUID) (tenant.Resource, error) {
	query := createSingleFilterQuery("id", "eq", resourceID.String(), 1)
	tableName := s.getTableName(schemaName)
	return getSingleRecord[tenant.Resource](s.repo, tableName, query, "failed to get resource by id")
}

func (s *resourceService) GetResourceByCode(ctx context.Context, schemaName string, code string) (tenant.Resource, error) {
	query := createSingleFilterQuery("code", "eq", code, 1)
	tableName := s.getTableName(schemaName)
	return getSingleRecord[tenant.Resource](s.repo, tableName, query, "failed to get resource by code")
}

func (s *resourceService) ListResources(ctx context.Context, schemaName string, limit, offset int) ([]tenant.Resource, int64, error) {
	tableName := s.getTableName(schemaName)
	query := dbModels.QueryParams{
		Limit:   &limit,
		Offset:  &offset,
		OrderBy: []string{"code"},
	}

	resources, err := listRecords[tenant.Resource](s.repo, tableName, query, "failed to list resources")
	if err != nil {
		return nil, 0, err
	}

	count, err := countRecords(s.repo, tableName, "failed to count resources")
	if err != nil {
		return nil, 0, err
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

	var resource tenant.Resource
	if err := helpers.MapToStruct(updatedData, &resource); err != nil {
		return tenant.Resource{}, app_errors.ErrMapToStruct
	}
	return resource, nil
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
