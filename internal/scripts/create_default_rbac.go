package scripts

import (
	"context"
	"godbgrest/pkg"
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
		accessRoleService,
		resourceService,
		actionService,
		permissionService,
		rolePermissionService,
		accessMemberService,
		baseService,
	)

	err := rbacManagementService.InitializeRBACSystem(ctx, appConstant.MasterDatabase)
	if err != nil {
		return err
	}
	return nil
}
