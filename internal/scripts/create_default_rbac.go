package scripts

import (
	"context"
	"go-postgres-rest/pkg"
	appConstant "serenibase/internal/constant"
	"serenibase/internal/services"
)

func CreateDefaultRBAC(dbService *pkg.DatabaseService) error {
	ctx := context.Background()

	// Initialize required service
	resourceService := services.NewResourceService(dbService)
	actionService := services.NewActionService(dbService)
	permissionService := services.NewPermissionService(dbService)
	rolePermissionService := services.NewRolePermissionService(dbService)
	accessMemberService := services.NewAccessMemberService(dbService)
	accessRoleService := services.NewAccessRoleService(dbService)
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
