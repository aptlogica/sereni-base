// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"github.com/aptlogica/go-postgres-rest/pkg"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	common "github.com/aptlogica/sereni-base/internal/services/common"
	core "github.com/aptlogica/sereni-base/internal/services/core"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"

	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"

	"github.com/google/uuid"
)

type rolePermissionService struct {
	repo *pkg.DatabaseService
}

func NewRolePermissionService(repo *pkg.DatabaseService) interfaces.RolePermissionService {
	return &rolePermissionService{repo: repo}
}

func (s *rolePermissionService) AssignPermissionToRole(ctx context.Context, schemaName string, req dto.RolePermissionDTO) (tenant.RolePermission, error) {
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}

	tableName := tenant.RolePermission{}.TableName(schemaName)
	insertedData, err := s.repo.TableService.CreateRecord(tableName, req.Map())
	if err != nil {
		return tenant.RolePermission{}, err
	}

	var rolePermission tenant.RolePermission
	if err := helpers.MapToStruct(insertedData, &rolePermission); err != nil {
		return tenant.RolePermission{}, err
	}
	return rolePermission, nil
}

func (s *rolePermissionService) RemovePermissionFromRole(ctx context.Context, schemaName string, roleID, permissionID uuid.UUID) error {
	tableName := tenant.RolePermission{}.TableName(schemaName)
	// Build compound filter for role_id AND permission_id
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "role_id",
				Operator: "eq",
				Value:    roleID.String(),
			},
			{
				Column:   "permission_id",
				Operator: "eq",
				Value:    permissionID.String(),
			},
		},
	}

	// Get the role permission to delete
	data, err := s.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return app_errors.LogDatabaseError(err, "failed to fetch role permission for removal")
	}

	if len(data) == 0 {
		return app_errors.ErrRecordNotFound
	}

	filter := dbModels.QueryFilter{
		Column:   "role_id",
		Operator: "eq",
		Value:    roleID.String(),
	}
	return s.repo.TableService.DeleteRecord(tableName, filter)
}

func (s *rolePermissionService) GetRolePermissions(ctx context.Context, schemaName string, roleID uuid.UUID) ([]tenant.RolePermission, error) {
	tableName := tenant.RolePermission{}.TableName(schemaName)
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "role_id",
				Operator: "eq",
				Value:    roleID.String(),
			},
		},
	}

	data, err := s.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to get role permissions")
	}

	rolePermissions, err := common.MapToStructList[tenant.RolePermission](data)
	if err != nil {
		return nil, err
	}
	return rolePermissions, nil
}

func (s *rolePermissionService) GetPermissionsByRole(ctx context.Context, schemaName string, roleID uuid.UUID) ([]dto.PermissionWithDetails, error) {
	// Get all role permissions for this role
	rolePermissions, err := s.GetRolePermissions(ctx, schemaName, roleID)
	if err != nil {
		return nil, err
	}

	var permissions []dto.PermissionWithDetails

	for _, rp := range rolePermissions {
		// Get permission details
		perm, err := NewPermissionService(s.repo).GetPermissionByID(ctx, schemaName, rp.PermissionID)
		if err != nil {
			continue
		}

		// Get resource and action codes
		resource, _ := core.NewResourceService(s.repo).GetResourceByID(ctx, schemaName, perm.ResourceID)
		action, _ := core.NewActionService(s.repo).GetActionByID(ctx, schemaName, perm.ActionID)

		permission := dto.PermissionWithDetails{
			ID:           perm.ID,
			ResourceCode: resource.Code,
			ActionCode:   action.Code,
			CreatedAt:    perm.CreatedAt,
		}
		permissions = append(permissions, permission)
	}

	return permissions, nil
}

func (s *rolePermissionService) GetRolesByPermission(ctx context.Context, schemaName string, permissionID uuid.UUID) ([]tenant.AccessRole, error) {
	tableName := tenant.RolePermission{}.TableName(schemaName)
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "permission_id",
				Operator: "eq",
				Value:    permissionID.String(),
			},
		},
	}

	data, err := s.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to get roles by permission")
	}

	var roles []tenant.AccessRole
	accessRoleSvc := NewAccessRoleService(s.repo)

	for _, item := range data {
		var rp tenant.RolePermission
		if err := helpers.MapToStruct(item, &rp); err != nil {
			continue
		}

		role, err := accessRoleSvc.GetAccessRoleByID(ctx, schemaName, rp.RoleID)
		if err == nil {
			roles = append(roles, role)
		}
	}

	return roles, nil
}

func (s *rolePermissionService) CheckRoleHasPermission(ctx context.Context, schemaName string, roleID, permissionID uuid.UUID) (bool, error) {
	tableName := tenant.RolePermission{}.TableName(schemaName)
	limit := 1
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "role_id",
				Operator: "eq",
				Value:    roleID.String(),
			},
			{
				Column:   "permission_id",
				Operator: "eq",
				Value:    permissionID.String(),
			},
		},
		Limit: &limit,
	}

	data, err := s.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return false, app_errors.LogDatabaseError(err, "failed to check role permission")
	}

	return len(data) > 0, nil
}
