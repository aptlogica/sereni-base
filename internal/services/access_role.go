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

type accessRoleService struct {
	repo *pkg.DatabaseService
}

func NewAccessRoleService(repo *pkg.DatabaseService) interfaces.AccessRoleService {
	return &accessRoleService{repo: repo}
}

func (s *accessRoleService) CreateAccessRole(ctx context.Context, schemaName string, req dto.AccessRoleDTO) (tenant.AccessRole, error) {
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}

	tableName := tenant.AccessRole{}.TableName(schemaName)
	insertedRoleData, err := s.repo.TableService.CreateRecord(tableName, req.Map())
	if err != nil {
		return tenant.AccessRole{}, err
	}

	var insertedRole tenant.AccessRole
	if err := helpers.MapToStruct(insertedRoleData, &insertedRole); err != nil {
		return tenant.AccessRole{}, err
	}
	return insertedRole, nil
}

func (s *accessRoleService) GetAccessRoleByID(ctx context.Context, schemaName string, roleID uuid.UUID) (tenant.AccessRole, error) {
	limit := 1
	tableName := tenant.AccessRole{}.TableName(schemaName)
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "id",
				Operator: "eq",
				Value:    roleID.String(),
			},
		},
		Limit: &limit,
	}

	rolesData, err := s.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return tenant.AccessRole{}, app_errors.LogDatabaseError(err, "failed to get access role by id")
	}

	if len(rolesData) == 0 {
		return tenant.AccessRole{}, app_errors.RoleNotFound
	}

	var role tenant.AccessRole
	if err := helpers.MapToStruct(rolesData[0], &role); err != nil {
		return tenant.AccessRole{}, app_errors.ErrMapToStruct
	}
	return role, nil
}

func (s *accessRoleService) GetAccessRoleByName(ctx context.Context, schemaName string, name string) (tenant.AccessRole, error) {
	limit := 1
	tableName := tenant.AccessRole{}.TableName(schemaName)
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "name",
				Operator: "eq",
				Value:    name,
			},
		},
		Limit: &limit,
	}

	rolesData, err := s.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return tenant.AccessRole{}, app_errors.LogDatabaseError(err, "failed to get access role by name")
	}

	if len(rolesData) == 0 {
		return tenant.AccessRole{}, app_errors.RoleNotFound
	}

	var role tenant.AccessRole
	if err := helpers.MapToStruct(rolesData[0], &role); err != nil {
		return tenant.AccessRole{}, app_errors.ErrMapToStruct
	}
	return role, nil
}

func (s *accessRoleService) GetAccessRolesByScope(ctx context.Context, schemaName string, scopeLevel string) ([]tenant.AccessRole, error) {
	tableName := tenant.AccessRole{}.TableName(schemaName)
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "scope_level",
				Operator: "eq",
				Value:    scopeLevel,
			},
		},
		OrderBy: []string{"priority"},
	}

	rolesData, err := s.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to get access roles by scope")
	}

	roles, err := mapToStructList[tenant.AccessRole](rolesData)
	if err != nil {
		return nil, err
	}
	return roles, nil
}

func (s *accessRoleService) ListAccessRoles(ctx context.Context, schemaName string, limit, offset int) ([]tenant.AccessRole, int64, error) {
	tableName := tenant.AccessRole{}.TableName(schemaName)
	query := dbModels.QueryParams{
		Limit:   &limit,
		Offset:  &offset,
		OrderBy: []string{"priority"},
	}

	rolesData, err := s.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return nil, 0, app_errors.LogDatabaseError(err, "failed to list access roles")
	}

	// Get count
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
		return nil, 0, app_errors.LogDatabaseError(err, "failed to count access roles")
	}

	var count int64
	if len(countData) > 0 {
		if total, ok := countData[0]["total"]; ok {
			count = int64(total.(float64))
		}
	}

	roles, err := mapToStructList[tenant.AccessRole](rolesData)
	if err != nil {
		return nil, 0, err
	}
	return roles, count, nil
}

func (s *accessRoleService) UpdateAccessRole(ctx context.Context, schemaName string, roleID uuid.UUID, req dto.AccessRoleDTO) (tenant.AccessRole, error) {
	tableName := tenant.AccessRole{}.TableName(schemaName)
	updateData := req.Map()

	updatedRoleData, err := s.repo.TableService.UpdateRecord(tableName, roleID, updateData)
	if err != nil {
		return tenant.AccessRole{}, err
	}

	var updatedRole tenant.AccessRole
	if err := helpers.MapToStruct(updatedRoleData, &updatedRole); err != nil {
		return tenant.AccessRole{}, app_errors.ErrMapToStruct
	}
	return updatedRole, nil
}

func (s *accessRoleService) DeleteAccessRole(ctx context.Context, schemaName string, roleID uuid.UUID) error {
	tableName := tenant.AccessRole{}.TableName(schemaName)
	filter := dbModels.QueryFilter{
		Column:   "id",
		Operator: "eq",
		Value:    roleID.String(),
	}

	return s.repo.TableService.DeleteRecord(tableName, filter)
}
