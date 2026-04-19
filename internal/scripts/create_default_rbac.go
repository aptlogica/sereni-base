// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package scripts

import (
	"context"

	"github.com/aptlogica/go-postgres-rest/pkg"
	appConstant "github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/services"
	core "github.com/aptlogica/sereni-base/internal/services/core"
	rbac "github.com/aptlogica/sereni-base/internal/services/rbac"
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
