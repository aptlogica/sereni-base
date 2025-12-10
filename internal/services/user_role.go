package services

import (
	"context"
	"godbgrest/pkg"
	dbModels "godbgrest/pkg/models"
	app_errors "serenibase/internal/app-errors"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/helpers"
)

type userRoleService struct {
	repo *pkg.DatabaseService
}

func NewUserRoleService(repo *pkg.DatabaseService) interfaces.UserRoleService {
	return &userRoleService{repo: repo}
}

func (s *userRoleService) CreateUserRole(ctx context.Context, schemaName string, req dto.UserRoleInsertion) (tenant.UserRole, error) {
	existingUserRole, err := s.GetByUser(ctx, schemaName, req.UserID.String())
	if err != nil && err != app_errors.ErrRecordNotFound {
		return tenant.UserRole{}, err
	}
	if err == nil {
		return existingUserRole, app_errors.UserAlreadyExists
	}

	tableName := tenant.UserRole{}.TableName(schemaName)
	insertedUserRoleData, err := s.repo.TableService.CreateRecord(ctx, tableName, req.Map())
	if err != nil {
		return tenant.UserRole{}, err
	}

	var insertedUserRole tenant.UserRole
	if err := helpers.MapToStruct(insertedUserRoleData, &insertedUserRole); err != nil {
		return tenant.UserRole{}, err
	}
	return insertedUserRole, nil
}

func (s *userRoleService) GetByUser(ctx context.Context, schemaName string, userId string) (tenant.UserRole, error) {
	limit := 1
	tableName := tenant.UserRole{}.TableName(schemaName)
	query := dbModels.QueryParams{
		Select: []string{"id"},
		Filters: []dbModels.QueryFilter{
			{
				Column:   "user_id",
				Operator: "eq",
				Value:    userId,
			},
		},
		Limit: &limit,
	}

	userRolesData, err := s.repo.TableService.GetTableData(ctx, tableName, query)
	if err != nil {
		return tenant.UserRole{}, err
	}

	if len(userRolesData) == 0 {
		return tenant.UserRole{}, app_errors.ErrRecordNotFound
	}

	userRoleData := userRolesData[0]

	var userRole tenant.UserRole
	if err := helpers.MapToStruct(userRoleData, &userRole); err != nil {
		return tenant.UserRole{}, err
	}
	return userRole, nil
}

func (s *userRoleService) RemoveByUserID(ctx context.Context, schemaName string, userID interface{}) error {
	tableName := tenant.UserRole{}.TableName(schemaName)

	userRoleData, err := s.GetByUser(ctx, schemaName, userID.(string))
	if err != nil {
		return err
	}

	return s.repo.TableService.DeleteRecord(ctx, tableName, userRoleData.ID)
}
