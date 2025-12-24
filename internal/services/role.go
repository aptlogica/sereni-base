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
)

type roleService struct {
	repo *pkg.DatabaseService
}

func NewRoleService(repo *pkg.DatabaseService) interfaces.RoleService {
	return &roleService{repo: repo}
}

func (s *roleService) CreateRole(ctx context.Context, schemaName string, req dto.RoleInsertion) (tenant.Role, error) {
	tableName := tenant.Role{}.TableName(schemaName)
	insertedRoleData, err := s.repo.TableService.CreateRecord(ctx, tableName, req.Map())
	if err != nil {
		return tenant.Role{}, err
	}

	var insertedRole tenant.Role
	if err := helpers.MapToStruct(insertedRoleData, &insertedRole); err != nil {
		return tenant.Role{}, err
	}
	return insertedRole, nil
}

func (s *roleService) GetRoleByName(ctx context.Context, schemaName string, name string) (tenant.Role, error) {
	limit := 1
	tableName := tenant.Role{}.TableName(schemaName)
	query := dbModels.QueryParams{
		Select: []string{"id"},
		Filters: []dbModels.QueryFilter{
			{
				Column:   "name",
				Operator: "eq",
				Value:    name,
			},
		},
		Limit: &limit,
	}

	rolesData, err := s.repo.TableService.GetTableData(ctx, tableName, query)
	if err != nil {
		return tenant.Role{}, app_errors.DatabaseError
	}

	if len(rolesData) == 0 {
		return tenant.Role{}, app_errors.RoleNotFound
	}

	roleData := rolesData[0]

	var role tenant.Role
	if err := helpers.MapToStruct(roleData, &role); err != nil {
		return tenant.Role{}, app_errors.ErrMapToStruct
	}
	return role, nil
}
