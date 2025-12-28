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
	"time"

	"godbgrest/pkg"

	"github.com/google/uuid"
)

// RegisterOwner registers a predefined owner user from configuration
// This function should be called during application initialization after CreateMasterSchema
func RegisterOwner(
	dbService *pkg.DatabaseService,
	authProvider auth.AuthProvider,
	cfg *config.Config,
) error {
	// Check if owner registration is configured
	if cfg.OwnerRegistration.Email == "" {
		fmt.Println("⚠ Owner registration skipped: no email configured")
		return nil
	}

	fmt.Println("\n=== Owner Registration ===")
	fmt.Printf("Checking owner registration for: %s %s (%s)\n",
		cfg.OwnerRegistration.FirstName,
		cfg.OwnerRegistration.LastName,
		cfg.OwnerRegistration.Email)

	ctx := context.Background()

	// Initialize required services
	userService := services.NewUserService(dbService)
	workspaceService := services.NewWorkspaceService(dbService)
	workspaceMemberService := services.NewWorkspaceMemberService(dbService)
	baseService := services.NewBaseService(dbService)
	modelService := services.NewModelService(dbService)
	columnService := services.NewColumnService(dbService)
	viewService := services.NewViewService(dbService)
	relationshipService := services.NewRelationshipService(dbService)
	userResetTokenService := services.NewUserResetTokenService(dbService)
	resourceService := services.NewResourceService(dbService)
	actionService := services.NewActionService(dbService)
	permissionService := services.NewPermissionService(dbService)
	rolePermissionService := services.NewRolePermissionService(dbService)
	accessMemberService := services.NewAccessMemberService(dbService)
	accessRoleService := services.NewAccessRoleService(dbService)
	assetService := services.NewAssetsService(dbService)

	// Create management services
	assetManagementService := services.NewAssetManagementService(
		dbService,
		assetService,
		nil, // storageProvider not needed for user registration
		nil, // antivirusProvider not needed for user registration
	)

	tableManagementService := services.NewTableManagementService(
		"postgres",
		dbService,
		modelService,
		columnService,
		viewService,
		relationshipService,
		assetManagementService,
	)

	baseManagementService := services.NewBaseManagementService(
		dbService,
		baseService,
		modelService,
	)

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

	workspaceManagementService := services.NewWorkspaceManagementService(
		dbService,
		workspaceService,
		workspaceMemberService,
		baseManagementService,
		tableManagementService,
		rbacManagementService,
	
	)

	

	userManagementService := services.NewUserManagementService(
		dbService,
		userService,
		assetManagementService,
		userResetTokenService,
		workspaceManagementService,
		rbacManagementService,
		authProvider,
	)

	// Initialize provider services for AuthManagementService
	emailTemplateService := emailProvider.NewEmailTemplateService()
	emailProviderService := emailProvider.NewService(cfg.Email, 100, emailTemplateService)
	emailProviderService.Start(5)

	otpProviderService := otpProvider.NewService(5 * time.Minute)

	// Create AuthManagementService
	authManagementService := services.NewAuthManagementService(
		cfg.TemporaryAddedUserPassword,
		dbService,
		userManagementService,
		workspaceManagementService,
		userResetTokenService,
		rbacManagementService,
		otpProviderService,
		emailTemplateService,
		emailProviderService,
		authProvider,
	)

	// Validate required configuration
	if cfg.OwnerRegistration.Password == "" {
		return fmt.Errorf("owner password is required in config.yaml")
	}
	if cfg.OwnerRegistration.FirstName == "" {
		return fmt.Errorf("owner first name is required in config.yaml")
	}
	if cfg.OwnerRegistration.LastName == "" {
		return fmt.Errorf("owner last name is required in config.yaml")
	}

	// Get country from local machine (use environment variable or default)
	country := os.Getenv("COUNTRY")
	if country == "" {
		country = "US" // Default to US if not set
	}

	// Prepare registration request
	userID := uuid.New()
	registerReq := dto.RegisterRequest{
		ID:            userID,
		Email:         cfg.OwnerRegistration.Email,
		FirstName:     cfg.OwnerRegistration.FirstName,
		LastName:      cfg.OwnerRegistration.LastName,
		Password:      cfg.OwnerRegistration.Password, // RegisterOwner will hash it
		AuthProvider:  "local",
		Status:        "active",
		EmailVerified: true,
		Country:       country,
		Timezone:      time.Now().Location().String(),
		Roles:         constant.RBACRoleNames.Owner,
	}

	// Use AuthManagementService.RegisterOwner to handle the registration
	fmt.Println("\nStep 1: Registering owner using AuthManagementService...")
	loginResponse, err := authManagementService.RegisterOwner(ctx, registerReq)
	if err != nil {
		return fmt.Errorf("failed to register owner: %w", err)
	}

	// Success!
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

	// Create default organization after owner registration
	fmt.Println("\n=== Creating Default Organization ===")
	err = CreateDefaultOrganization(dbService, cfg, loginResponse.User.Email)
	if err != nil {
		fmt.Printf("⚠ Warning: Failed to create default organization: %v\n", err)
		// Don't fail the entire flow if organization creation fails
	}

	fmt.Println()

	return nil
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
