package scripts

import (
	"context"
	"fmt"
	"os"
	"serenibase/internal/config"
	"serenibase/internal/constant"
	"serenibase/internal/dto"
	"serenibase/internal/providers/auth"
	emailProvider "serenibase/internal/providers/email"
	otpProvider "serenibase/internal/providers/otp"
	"serenibase/internal/services"
	"serenibase/internal/services/interfaces"
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
	if skip := maybeSkipOwnerRegistration(cfg); skip {
		return nil
	}

	fmt.Println("\n=== Owner Registration ===")
	fmt.Printf("Checking owner registration for: %s %s (%s)\n",
		cfg.OwnerRegistration.FirstName,
		cfg.OwnerRegistration.LastName,
		cfg.OwnerRegistration.Email)

	if err := validateOwnerConfig(cfg); err != nil {
		return err
	}

	ctx := context.Background()

	core := initCoreServices(dbService)
	mgmt := initManagementServices(dbService, core, authProvider)
	providers := initAuthProviders(cfg)
	authManagementService := services.NewAuthManagementService(
		cfg.TemporaryAddedUserPassword,
		dbService,
		mgmt.userManagementService,
		mgmt.workspaceManagementService,
		core.userResetTokenService,
		mgmt.rbacManagementService,
		providers.otpProviderService,
		providers.emailTemplateService,
		providers.emailProviderService,
		authProvider,
	)

	registerReq := prepareOwnerRegisterRequest(cfg)

	fmt.Println("\nStep 1: Registering owner using AuthManagementService...")
	loginResponse, err := authManagementService.RegisterOwner(ctx, registerReq)
	if err != nil {
		return fmt.Errorf("failed to register owner: %w", err)
	}

	printOwnerSuccess(loginResponse, cfg)

	fmt.Println("\n=== Creating Default Organization ===")
	err = CreateDefaultOrganization(dbService, cfg, loginResponse.User.Email)
	if err != nil {
		fmt.Printf("⚠ Warning: Failed to create default organization: %v\n", err)
	}

	fmt.Println()

	return nil
}

func maybeSkipOwnerRegistration(cfg *config.Config) bool {
	if cfg.OwnerRegistration.Email == "" {
		fmt.Println("⚠ Owner registration skipped: no email configured")
		return true
	}
	return false
}

func validateOwnerConfig(cfg *config.Config) error {
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
		viewService:            services.NewViewService(dbService),
		relationshipService:    services.NewRelationshipService(dbService),
		userResetTokenService:  services.NewUserResetTokenService(dbService),
		resourceService:        services.NewResourceService(dbService),
		actionService:          services.NewActionService(dbService),
		permissionService:      services.NewPermissionService(dbService),
		rolePermissionService:  services.NewRolePermissionService(dbService),
		accessMemberService:    services.NewAccessMemberService(dbService),
		accessRoleService:      services.NewAccessRoleService(dbService),
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
		core.accessRoleService,
		core.resourceService,
		core.actionService,
		core.permissionService,
		core.rolePermissionService,
		core.accessMemberService,
		core.baseService,
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

func prepareOwnerRegisterRequest(cfg *config.Config) dto.RegisterRequest {
	country := os.Getenv("COUNTRY")
	if country == "" {
		country = "US"
	}

	return dto.RegisterRequest{
		ID:            uuid.New(),
		Email:         cfg.OwnerRegistration.Email,
		FirstName:     cfg.OwnerRegistration.FirstName,
		LastName:      cfg.OwnerRegistration.LastName,
		Password:      cfg.OwnerRegistration.Password,
		AuthProvider:  "local",
		Status:        "active",
		EmailVerified: true,
		Country:       country,
		Timezone:      time.Now().Location().String(),
		Roles:         constant.RBACRoleNames.Owner,
	}
}

func printOwnerSuccess(loginResponse dto.LoginResponse, cfg *config.Config) {
	fmt.Println("\n==================================================")
	fmt.Println("✓ Owner registration completed successfully!")
	fmt.Println("==================================================")

	if loginResponse.User != nil {
		fmt.Printf("\nOwner Details:\n")
		fmt.Printf("  Name:      %s %s\n", loginResponse.User.FirstName, loginResponse.User.LastName)
		fmt.Printf("  Email:     %s\n", loginResponse.User.Email)
		fmt.Printf("  User ID:   %s\n", loginResponse.User.ID)
		fmt.Printf("  Role:      Admin\n")
	}

	fmt.Printf("\nYou can now login with:\n")
	fmt.Printf("  Email:    %s\n", cfg.OwnerRegistration.Email)
	fmt.Printf("  Password: <configured password>\n")
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
	fmt.Printf("Creating organization: %s\n", orgRequest.Name)
	fmt.Printf("Organization email: %s\n", orgRequest.Email)

	organization, err := organizationService.CreateOrganization(ctx, constant.MasterDatabase, orgRequest)
	if err != nil {
		fmt.Printf("⚠ Warning: Failed to create default organization: %v\n", err)
		return nil // Don't fail owner registration if organization creation fails
	}

	fmt.Println("\n✓ Organization created successfully!")
	fmt.Printf("  Organization ID: %s\n", organization.ID.String())
	fmt.Printf("  Name: %s\n", organization.Name)
	fmt.Printf("  Email: %s\n", organization.Email)

	return nil
}
