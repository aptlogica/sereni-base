package scripts

import (
	"context"
	"fmt"
	"serenibase/internal/config"
	"serenibase/internal/constant"
	"serenibase/internal/dto"
	"serenibase/internal/providers/auth"
	emailProvider "serenibase/internal/providers/email"
	otpProvider "serenibase/internal/providers/otp"
	"serenibase/internal/services"
	core "serenibase/internal/services/core"
	"serenibase/internal/services/interfaces"
	rbac "serenibase/internal/services/rbac"
	"time"

	"go-postgres-rest/pkg"

	"github.com/google/uuid"
)

// RegisterOwner registers a predefined owner user from configuration
// This function should be called during application initialization after CreateMasterSchema
func RegisterOwner(
	dbService *pkg.DatabaseService,
	authProvider auth.AuthProvider,
	cfg *config.Config,
) error {
	if skip := MaybeSkipOwnerRegistration(cfg); skip {
		return nil
	}

	if err := ValidateOwnerConfig(cfg); err != nil {
		return err
	}

	ctx := context.Background()

	core := initCoreServices(dbService)
	mgmt := initManagementServices(dbService, core, authProvider)
	providers := initAuthProviders(cfg)
	authManagementService := services.NewAuthManagementService(
		cfg.TemporaryAddedUserPassword,
		dbService,
		services.AuthManagementServiceDeps{
			UserManagementService:      mgmt.userManagementService,
			WorkspaceManagementService: mgmt.workspaceManagementService,
			UserResetTokenService:      core.userResetTokenService,
			RBACManagementService:      mgmt.rbacManagementService,
		},
		services.AuthManagementProviderDeps{
			OTPProviderService:   providers.otpProviderService,
			EmailTemplateService: providers.emailTemplateService,
			EmailProviderService: providers.emailProviderService,
			AuthProviderService:  authProvider,
		},
	)

	registerReq := PrepareOwnerRegisterRequest(cfg)

	loginResponse, err := authManagementService.RegisterOwner(ctx, registerReq)
	if err != nil {
		return fmt.Errorf("failed to register owner: %w", err)
	}

	printOwnerSuccess(loginResponse, cfg)

	err = CreateDefaultOrganization(dbService, cfg, loginResponse.User.Email)
	if err != nil {
		// Organization creation failure is non-critical, continue
	}

	return nil
}

func MaybeSkipOwnerRegistration(cfg *config.Config) bool {
	if cfg.OwnerRegistration.Email == "" {
		return true
	}
	return false
}

func ValidateOwnerConfig(cfg *config.Config) error {
	if cfg.OwnerRegistration.Password == "" {
		return fmt.Errorf("owner password is required in config.yaml")
	}
	if cfg.OwnerRegistration.FirstName == "" {
		return fmt.Errorf("owner first name is required in config.yaml")
	}
	if cfg.OwnerRegistration.LastName == "" {
		return fmt.Errorf("owner last name is required in config.yaml")
	}
	return nil
}

type coreServices struct {
	userService            interfaces.UserService
	workspaceService       interfaces.WorkspaceService
	workspaceMemberService interfaces.WorkspaceMemberService
	baseService            interfaces.BaseService
	modelService           interfaces.ModelService
	columnService          interfaces.ColumnService
	viewService            interfaces.ViewService
	relationshipService    interfaces.RelationshipService
	userResetTokenService  interfaces.UserResetTokenService
	resourceService        interfaces.ResourceService
	actionService          interfaces.ActionService
	permissionService      interfaces.PermissionService
	rolePermissionService  interfaces.RolePermissionService
	accessMemberService    interfaces.AccessMemberService
	accessRoleService      interfaces.AccessRoleService
	assetService           interfaces.AssetService
}

func initCoreServices(dbService *pkg.DatabaseService) coreServices {
	return coreServices{
		userService:            services.NewUserService(dbService),
		workspaceService:       services.NewWorkspaceService(dbService),
		workspaceMemberService: services.NewWorkspaceMemberService(dbService),
		baseService:            services.NewBaseService(dbService),
		modelService:           services.NewModelService(dbService),
		columnService:          services.NewColumnService(dbService),
		viewService:            core.NewViewService(dbService),
		relationshipService:    core.NewRelationshipService(dbService),
		userResetTokenService:  services.NewUserResetTokenService(dbService),
		resourceService:        core.NewResourceService(dbService),
		actionService:          core.NewActionService(dbService),
		permissionService:      rbac.NewPermissionService(dbService),
		rolePermissionService:  services.NewRolePermissionService(dbService),
		accessMemberService:    services.NewAccessMemberService(dbService),
		accessRoleService:      rbac.NewAccessRoleService(dbService),
		assetService:           services.NewAssetsService(dbService),
	}
}

type managementServices struct {
	assetManagementService     interfaces.AssetManagementService
	tableManagementService     interfaces.TableManagementService
	baseManagementService      interfaces.BaseManagementService
	rbacManagementService      interfaces.RBACManagementService
	workspaceManagementService interfaces.WorkspaceManagementService
	userManagementService      interfaces.UserManagementService
}

func initManagementServices(
	dbService *pkg.DatabaseService,
	core coreServices,
	authProvider auth.AuthProvider,
) managementServices {
	assetManagementService := services.NewAssetManagementService(
		dbService,
		core.assetService,
		nil,
		nil,
	)

	tableManagementService := services.NewTableManagementService(
		"postgres",
		dbService,
		core.modelService,
		core.columnService,
		core.viewService,
		core.relationshipService,
		assetManagementService,
	)

	baseManagementService := services.NewBaseManagementService(
		dbService,
		core.baseService,
		tableManagementService,
		core.modelService,
		assetManagementService,
	)

	rbacManagementService := services.NewRBACManagementService(
		dbService,
		services.RBACManagementServiceDeps{
			RoleService:           core.accessRoleService,
			ResourceService:       core.resourceService,
			ActionService:         core.actionService,
			PermissionService:     core.permissionService,
			RolePermissionService: core.rolePermissionService,
			AccessMemberService:   core.accessMemberService,
			BaseService:           core.baseService,
		},
	)

	workspaceManagementService := services.NewWorkspaceManagementService(
		dbService,
		core.workspaceService,
		core.workspaceMemberService,
		baseManagementService,
		tableManagementService,
		rbacManagementService,
	)

	userManagementService := services.NewUserManagementService(
		dbService,
		core.userService,
		assetManagementService,
		core.userResetTokenService,
		workspaceManagementService,
		rbacManagementService,
		authProvider,
	)

	return managementServices{
		assetManagementService:     assetManagementService,
		tableManagementService:     tableManagementService,
		baseManagementService:      baseManagementService,
		rbacManagementService:      rbacManagementService,
		workspaceManagementService: workspaceManagementService,
		userManagementService:      userManagementService,
	}
}

type authProviders struct {
	emailTemplateService emailProvider.EmailTemplateService
	emailProviderService emailProvider.EmailService
	otpProviderService   otpProvider.OtpService
}

func initAuthProviders(cfg *config.Config) authProviders {
	emailTemplateService := emailProvider.NewEmailTemplateService()
	emailProviderService := emailProvider.NewService(cfg.Email, 100, emailTemplateService)
	emailProviderService.Start(5)

	otpProviderService := otpProvider.NewService(5 * time.Minute)

	return authProviders{
		emailTemplateService: emailTemplateService,
		emailProviderService: emailProviderService,
		otpProviderService:   otpProviderService,
	}
}

func PrepareOwnerRegisterRequest(cfg *config.Config) dto.RegisterRequest {
	return dto.RegisterRequest{
		ID:            uuid.New(),
		Email:         cfg.OwnerRegistration.Email,
		FirstName:     cfg.OwnerRegistration.FirstName,
		LastName:      cfg.OwnerRegistration.LastName,
		Password:      cfg.OwnerRegistration.Password,
		AuthProvider:  "local",
		Status:        "active",
		EmailVerified: true,
		Country:       "",
		Timezone:      "UTC",
		Roles:         constant.RBACRoleNames.Owner,
	}
}

func printOwnerSuccess(loginResponse dto.LoginResponse, cfg *config.Config) {
	// Owner registration logging can be added to a proper logger if needed
}

// CreateDefaultOrganization creates a default organization with the owner's email
func CreateDefaultOrganization(
	dbService *pkg.DatabaseService,
	cfg *config.Config,
	ownerEmail string,
) error {
	ctx := context.Background()

	// Initialize organization service
	organizationService := services.NewOrganizationService(dbService)

	// Create organization request with default values
	orgRequest := dto.CreateOrganizationRequest{
		Name:  cfg.OwnerRegistration.FirstName + "'s Organization",
		Email: ownerEmail,
	}

	// Create organization in master schema
	organization, err := organizationService.CreateOrganization(ctx, constant.MasterDatabase, orgRequest)
	if err != nil {
		return nil // Don't fail owner registration if organization creation fails
	}

	fmt.Println("\n✓ Organization created successfully!")
	fmt.Printf("  Organization ID: %s\n", organization.ID.String())
	fmt.Printf("  Name: %s\n", organization.Name)
	fmt.Printf("  Email: %s\n", organization.Email)

	return nil
}
