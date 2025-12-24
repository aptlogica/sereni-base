package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"serenibase/internal/config"
	appConstant "serenibase/internal/constant"
	"serenibase/internal/dto"
	"serenibase/internal/models/master"
	"serenibase/internal/providers/auth"
	"serenibase/internal/services"
	"serenibase/internal/utils/helpers"
	"time"

	"godbgrest/pkg"
	dbConfig "godbgrest/pkg/config"

	"github.com/google/uuid"
)

func main() {
	fmt.Println("=== Owner Registration Script ===")

	// Load application config
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}
	config.AppConfig = cfg

	// Validate owner registration config
	if cfg.OwnerRegistration.Email == "" {
		log.Fatal("Owner email is required in config.yaml")
	}
	if cfg.OwnerRegistration.Password == "" {
		log.Fatal("Owner password is required in config.yaml")
	}
	if cfg.OwnerRegistration.FirstName == "" {
		log.Fatal("Owner first name is required in config.yaml")
	}
	if cfg.OwnerRegistration.LastName == "" {
		log.Fatal("Owner last name is required in config.yaml")
	}

	fmt.Printf("Registering owner: %s %s (%s)\n", cfg.OwnerRegistration.FirstName, cfg.OwnerRegistration.LastName, cfg.OwnerRegistration.Email)

	// Load database config
	dbCfg, err := dbConfig.Load()
	if err != nil {
		log.Fatal("Failed to load database config:", err)
	}

	// Initialize database service
	dbService := pkg.NewDatabaseService()
	db, err := dbService.Connect(dbCfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize services
	if err := dbService.InitServices(db); err != nil {
		log.Fatalf("Failed to init services: %v", err)
	}

	// Initialize auth provider
	authProvider, err := auth.NewAuthProvider(&cfg.Auth)
	if err != nil {
		log.Fatalf("Failed to connect to auth provider: %v", err)
	}

	// Initialize required services
	userService := services.NewUserService(dbService)
	roleService := services.NewRoleService(dbService)
	subscriptionPlanService := services.NewSubscriptionPlanService(dbService)
	tenantService := services.NewTenantService(dbService)
	tenantMembershipService := services.NewTenantMembershipService(dbService)
	tenantSubscriptionService := services.NewTenantSubscriptionService(dbService)
	workspaceService := services.NewWorkspaceService(dbService)
	workspaceMemberService := services.NewWorkspaceMemberService(dbService)
	baseService := services.NewBaseService(dbService)
	modelService := services.NewModelService(dbService)
	columnService := services.NewColumnService(dbService)
	viewService := services.NewViewService(dbService)
	relationshipService := services.NewRelationshipService(dbService)
	userResetTokenService := services.NewUserResetTokenService(dbService)
	userRoleService := services.NewUserRoleService(dbService)
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
		dbCfg.Database.Driver,
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

	workspaceManagementService := services.NewWorkspaceManagementService(
		dbService,
		workspaceService,
		workspaceMemberService,
		baseManagementService,
		tableManagementService,
	)

	tenantManagementService := services.NewTenantManagementService(
		dbService,
		tenantService,
		tenantSubscriptionService,
		tenantMembershipService,
	)

	userManagementService := services.NewUserManagementService(
		dbService,
		userService,
		tenantManagementService,
		subscriptionPlanService,
		assetManagementService,
		userResetTokenService,
		userRoleService,
		workspaceManagementService,
		authProvider,
	)

	rbacManagementService := services.NewRBACManagementService(
		dbService,
		accessRoleService,
		resourceService,
		actionService,
		permissionService,
		rolePermissionService,
		accessMemberService,
	)

	ctx := context.Background()

	// Step 1: Check if user already exists in master schema
	fmt.Println("\nStep 1: Checking if user already exists...")
	existingUser, err := userManagementService.GetUserByEmail(ctx, appConstant.MasterDatabase, cfg.OwnerRegistration.Email)
	if err == nil && existingUser.ID != uuid.Nil {
		log.Fatalf("User with email %s already exists (ID: %s)", cfg.OwnerRegistration.Email, existingUser.ID)
	}

	// Step 2: Create user in master schema
	fmt.Println("Step 2: Creating user in master schema...")

	// Get country from local machine (use environment variable or default)
	country := os.Getenv("COUNTRY")
	if country == "" {
		country = "US" // Default to US if not set
	}

	// Hash password
	hashedPassword, err := helpers.HashPassword(cfg.OwnerRegistration.Password)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	userID := uuid.New()
	registerReq := dto.RegisterRequest{
		ID:            userID,
		Email:         cfg.OwnerRegistration.Email,
		FirstName:     cfg.OwnerRegistration.FirstName,
		LastName:      cfg.OwnerRegistration.LastName,
		Password:      hashedPassword,
		AuthProvider:  "local",
		Status:        "pending",
		EmailVerified: false,
		Country:       country,
		Timezone:      time.Now().Location().String(),
	}

	insertedUser, err := userManagementService.CreateUser(ctx, appConstant.MasterDatabase, registerReq)
	if err != nil {
		log.Fatalf("Failed to create user in master schema: %v", err)
	}
	fmt.Printf("✓ User created in master schema with ID: %s\n", insertedUser.ID)

	// Step 3: Generate token and add user to auth provider (skip since we're verifying immediately)
	fmt.Println("\nStep 3: Adding user to auth provider...")
	role := appConstant.RoleNames.Admin
	tenantID := uuid.New()

	_, err = authProvider.AddUser(ctx, master.User{
		ID:           insertedUser.ID,
		Email:        insertedUser.Email,
		FirstName:    insertedUser.FirstName,
		LastName:     insertedUser.LastName,
		Password:     cfg.OwnerRegistration.Password, // Use plain password for auth provider
		AuthProvider: "local",
	}, tenantID.String(), role)
	if err != nil {
		log.Fatalf("Failed to add user to auth provider: %v", err)
	}
	fmt.Println("✓ User added to auth provider")

	// Step 4: Get subscription plan (Free plan)
	fmt.Println("\nStep 4: Getting subscription plan...")
	plan, err := subscriptionPlanService.GetSubscriptionPlanByName(ctx, appConstant.PlanNames.Free)
	if err != nil {
		log.Fatalf("Failed to get subscription plan: %v", err)
	}
	fmt.Printf("✓ Found subscription plan: %s\n", plan.Name)

	// Step 5: Get admin role from master schema
	fmt.Println("\nStep 5: Getting admin role...")
	adminRole, err := roleService.GetRoleByName(ctx, appConstant.MasterDatabase, appConstant.RoleNames.Admin)
	if err != nil {
		log.Fatalf("Failed to get admin role: %v", err)
	}
	fmt.Printf("✓ Found admin role: %s\n", adminRole.Name)

	// Step 6: Initialize tenant
	fmt.Println("\nStep 6: Initializing tenant...")
	tenantReq := dto.TenantRequest{
		UserID:   insertedUser.ID,
		TenantID: tenantID,
	}

	tenantData, err := tenantManagementService.InitializeTenant(ctx, tenantReq, plan.ID, adminRole.ID)
	if err != nil {
		log.Fatalf("Failed to initialize tenant: %v", err)
	}
	fmt.Printf("✓ Tenant created with ID: %s, Schema: %s\n", tenantData.ID, tenantData.Schema)

	// Step 7: Initialize RBAC system for tenant schema
	fmt.Println("\nStep 7: Initializing RBAC system...")
	err = rbacManagementService.InitializeRBACSystem(ctx, tenantData.Schema)
	if err != nil {
		log.Fatalf("Failed to initialize RBAC system: %v", err)
	}
	fmt.Println("✓ RBAC system initialized")

	// Step 8: Create user in tenant schema
	fmt.Println("\nStep 8: Creating user in tenant schema...")
	_, err = userManagementService.CreateUser(ctx, tenantData.Schema, dto.RegisterRequest{
		ID:            insertedUser.ID,
		Email:         insertedUser.Email,
		FirstName:     insertedUser.FirstName,
		LastName:      insertedUser.LastName,
		Password:      hashedPassword,
		AuthProvider:  "local",
		Status:        "active",
		EmailVerified: true,
		Country:       country,
		Timezone:      time.Now().Location().String(),
	})
	if err != nil {
		log.Fatalf("Failed to create user in tenant schema: %v", err)
	}
	fmt.Println("✓ User created in tenant schema")

	// Step 9: Update user in master schema to active and verified
	fmt.Println("\nStep 9: Updating user status in master schema...")
	updateData := map[string]interface{}{
		"status":         "active",
		"email_verified": true,
		"last_login_at":  time.Now(),
	}

	updatedUser, err := userManagementService.UpdateUser(ctx, appConstant.MasterDatabase, insertedUser.ID.String(), updateData)
	if err != nil {
		log.Fatalf("Failed to update user in master schema: %v", err)
	}
	fmt.Printf("✓ User updated: Status=%s, EmailVerified=%v\n", updatedUser.Status, updatedUser.EmailVerified)

	// Step 10: Assign admin role to user in tenant schema
	fmt.Println("\nStep 10: Assigning admin role to user...")
	tenantAdminRole, err := roleService.GetRoleByName(ctx, tenantData.Schema, appConstant.RoleNames.Admin)
	if err != nil {
		log.Fatalf("Failed to get admin role from tenant schema: %v", err)
	}

	err = userManagementService.AddUserRole(ctx, tenantData.Schema, insertedUser.ID, tenantAdminRole.ID)
	if err != nil {
		log.Fatalf("Failed to assign admin role to user: %v", err)
	}
	fmt.Println("✓ Admin role assigned to user")

	// Step 11: Create default workspace
	fmt.Println("\nStep 11: Creating default workspace...")
	workspaceReq := dto.CreateWorkspaceRequest{
		Title:       "Default Workspace",
		Description: helpers.StringPtr(""),
	}

	workspace, err := workspaceManagementService.Create(ctx, workspaceReq, tenantData.Schema, insertedUser.ID.String())
	if err != nil {
		log.Fatalf("Failed to create default workspace: %v", err)
	}
	fmt.Printf("✓ Default workspace created with ID: %s\n", workspace.ID)

	// Step 12: Set email as verified in auth provider
	fmt.Println("\nStep 12: Setting email as verified in auth provider...")
	// Get auth provider user ID
	ok, authProviderUserID, _, err := authProvider.CheckUserExistsByEmailAndReturnUser(ctx, cfg.OwnerRegistration.Email)
	if err != nil || !ok {
		log.Fatalf("Failed to get auth provider user: %v", err)
	}

	err = authProvider.SetEmailVerified(ctx, authProviderUserID)
	if err != nil {
		log.Fatalf("Failed to set email as verified: %v", err)
	}
	fmt.Println("✓ Email verified in auth provider")

	// Success!
	fmt.Println("\n==================================================")
	fmt.Println("✓ Owner registration completed successfully!")
	fmt.Println("==================================================")
	fmt.Printf("\nOwner Details:\n")
	fmt.Printf("  Name:      %s %s\n", insertedUser.FirstName, insertedUser.LastName)
	fmt.Printf("  Email:     %s\n", insertedUser.Email)
	fmt.Printf("  User ID:   %s\n", insertedUser.ID)
	fmt.Printf("  Tenant ID: %s\n", tenantData.ID)
	fmt.Printf("  Schema:    %s\n", tenantData.Schema)
	fmt.Printf("  Role:      Admin\n")
	fmt.Printf("\nYou can now login with:\n")
	fmt.Printf("  Email:    %s\n", cfg.OwnerRegistration.Email)
	fmt.Printf("  Password: <configured password>\n")
	fmt.Println()
}
