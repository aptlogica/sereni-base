// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package scripts

import (
	"context"
	"go-postgres-rest/pkg"
	appConstant "serenibase/internal/constant"
	"serenibase/internal/services"
	core "serenibase/internal/services/core"
	rbac "serenibase/internal/services/rbac"
)

func CreateDefaultRBAC(dbService *pkg.DatabaseService) error {
	ctx := context.Background()

	// Initialize required service
	resourceService := core.NewResourceService(dbService)
	actionService := core.NewActionService(dbService)
	permissionService := rbac.NewPermissionService(dbService)
	rolePermissionService := services.NewRolePermissionService(dbService)
	accessMemberService := services.NewAccessMemberService(dbService)
	accessRoleService := rbac.NewAccessRoleService(dbService)
	baseService := services.NewBaseService(dbService)

	rbacManagementService := services.NewRBACManagementService(
		dbService,
		services.RBACManagementServiceDeps{
			RoleService:           accessRoleService,
			ResourceService:       resourceService,
			ActionService:         actionService,
			PermissionService:     permissionService,
			RolePermissionService: rolePermissionService,
			AccessMemberService:   accessMemberService,
			BaseService:           baseService,
		},
	)

	err := rbacManagementService.InitializeRBACSystem(ctx, appConstant.MasterDatabase)
	if err != nil {
		return err
	}
	return nil
}
