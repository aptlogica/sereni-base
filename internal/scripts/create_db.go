package scripts

import (
	"context"
	"fmt"
	"godbgrest/pkg"
	"godbgrest/pkg/models"
	"serenibase/internal/constant"
	"serenibase/internal/dto"
	"serenibase/internal/models/master"
	"serenibase/internal/utils/helpers"
	"time"

	dbModels "godbgrest/pkg/models"

	"github.com/google/uuid"
)

func createRole(repo *pkg.DatabaseService, req dto.RoleInsertion) (master.Role, error) {
	ctx := context.Background()
	tableName := master.Role{}.TableName(constant.MasterDatabase)

	roleInsertion := dto.RoleInsertion{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		IsDefault:   req.IsDefault,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	insertedRoleData, err := repo.TableService.CreateRecord(ctx, tableName, roleInsertion.Map())
	if err != nil {
		return master.Role{}, err
	}

	var insertedRole master.Role
	if err := helpers.MapToStruct(insertedRoleData, &insertedRole); err != nil {
		return master.Role{}, err
	}
	return insertedRole, nil
}

func isRoleExists(repo *pkg.DatabaseService, name string) (bool, error) {
	ctx := context.Background()
	limit := 1
	tableName := master.Role{}.TableName(constant.MasterDatabase)
	query := dbModels.QueryParams{
		Select: []string{"id"},
		Filters: []dbModels.QueryFilter{
			{
				Column:   "name",
				Operator: "eq",
				Value:    name,
			},
		},
		Limit: &limit,
	}

	rolesData, err := repo.TableService.GetTableData(ctx, tableName, query)
	if err != nil {
		return false, err
	}

	if rolesData == nil || len(rolesData) == 0 {
		return true, nil // does not exist, so return true
	}

	return false, nil // already exists, so return false
}

func createSubscriptionPlan(repo *pkg.DatabaseService, req dto.PlanInsertion) (master.SubscriptionPlan, error) {
	ctx := context.Background()
	tableName := master.SubscriptionPlan{}.TableName(constant.MasterDatabase)

	planInsertion := dto.PlanInsertion{
		ID:                   uuid.New(),
		Name:                 req.Name,
		Slug:                 req.Slug,
		Description:          req.Description,
		Currency:             req.Currency,
		MaxWorkspaces:        req.MaxWorkspaces,
		MaxBasesPerWorkspace: req.MaxBasesPerWorkspace,
		MaxTablesPerBase:     req.MaxTablesPerBase,
		MaxRowsPerTable:      req.MaxRowsPerTable,
		MaxCollaborators:     req.MaxCollaborators,
		MaxAPICallsPerHour:   req.MaxAPICallsPerHour,
		StorageLimitGB:       req.StorageLimitGB,
		Features:             req.Features,
		IsActive:             req.IsActive,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	insertedPlanData, err := repo.TableService.CreateRecord(ctx, tableName, planInsertion.Map())
	if err != nil {
		return master.SubscriptionPlan{}, err
	}

	var insertedPlan master.SubscriptionPlan
	if err := helpers.MapToStruct(insertedPlanData, &insertedPlan); err != nil {
		return master.SubscriptionPlan{}, err
	}
	return insertedPlan, nil
}

func isPlanExists(repo *pkg.DatabaseService, slug string) (bool, error) {
	ctx := context.Background()
	limit := 1
	tableName := master.SubscriptionPlan{}.TableName(constant.MasterDatabase)
	query := dbModels.QueryParams{
		Select: []string{"id"},
		Filters: []dbModels.QueryFilter{
			{
				Column:   "slug",
				Operator: "eq",
				Value:    slug,
			},
		},
		Limit: &limit,
	}

	plansData, err := repo.TableService.GetTableData(ctx, tableName, query)
	if err != nil {
		return false, err
	}

	if plansData == nil || len(plansData) == 0 {
		return true, nil // does not exist, so return true
	}

	return false, nil // already exists, so return false
}

func createDefaultRoles(repo *pkg.DatabaseService) error {
	for _, role := range constant.DefaultRoles {
		if role.Name == constant.RoleNames.Admin {
			exists, err := isRoleExists(repo, role.Name)
			if err != nil {
				return fmt.Errorf("error checking if role '%s' exists: %w", role.Name, err)
			}
			if exists {
				_, err := createRole(repo, role)
				if err != nil {
					return fmt.Errorf("error creating role '%s': %w", role.Name, err)
				}
				fmt.Printf("Role '%s' created successfully.\n", role.Name)
			} else {
				fmt.Printf("Role '%s' already exists. Skipping creation.\n", role.Name)
			}
		}
	}
	return nil
}

func createMasterSchema(repo *pkg.DatabaseService) {
	ctx := context.Background()
	masterSchema := constant.MasterDatabase
	err := repo.TableService.CreateSchema(ctx, masterSchema)
	if err != nil {
		fmt.Printf("Error creating master schema: %v\n", err)
	} else {
		fmt.Printf("Master schema '%s' created successfully.\n", masterSchema)
	}
}

func createTableUsingSchema(repo *pkg.DatabaseService, schema models.CreateTableRequest) {
	if err := repo.TableService.CreateTable(schema); err != nil {
		fmt.Printf("Error creating table %s: %v\n", schema.Name, err)
	} else {
		fmt.Printf("Table %s created successfully.\n", schema.Name)
	}
}

func createDefaultPlans(repo *pkg.DatabaseService) error {
	for _, plan := range constant.DefaultPlans {
		exists, err := isPlanExists(repo, plan.Slug)
		if err != nil {
			return fmt.Errorf("error checking if plan '%s' exists: %w", plan.Slug, err)
		}
		if exists {
			_, err := createSubscriptionPlan(repo, plan)
			if err != nil {
				return fmt.Errorf("error creating plan '%s': %w", plan.Slug, err)
			}
			fmt.Printf("Plan '%s' created successfully.\n", plan.Slug)
		} else {
			fmt.Printf("Plan '%s' already exists. Skipping creation.\n", plan.Slug)
		}
	}
	return nil
}

func createFunctions(repo *pkg.DatabaseService, schemaName string) error {
	ctx := context.Background()
	for _, function := range constant.DefinedFunctions {
		functionName := fmt.Sprintf("\"%s\".%s(%s)", schemaName, function.FunctionName, function.FunctionParams)
		err := repo.TableService.CreateFunction(ctx, functionName, function.FunctionSQL)
		if err != nil {
			fmt.Printf("error creating function '%s': %w\n", function.FunctionName, err)
		}
	}
	return nil
}

func CreateMasterSchema(repo *pkg.DatabaseService) {
	// create master schema
	createMasterSchema(repo)

	// Create all tables
	createTableUsingSchema(repo, master.Role{}.TableSchema(constant.MasterDatabase))
	createTableUsingSchema(repo, master.User{}.TableSchema(constant.MasterDatabase))
	createTableUsingSchema(repo, master.Tenant{}.TableSchema(constant.MasterDatabase))
	createTableUsingSchema(repo, master.UsageMetric{}.TableSchema(constant.MasterDatabase))
	createTableUsingSchema(repo, master.SubscriptionPlan{}.TableSchema(constant.MasterDatabase))
	createTableUsingSchema(repo, master.TenantSubscription{}.TableSchema(constant.MasterDatabase))
	createTableUsingSchema(repo, master.TenantMembership{}.TableSchema(constant.MasterDatabase))
	createTableUsingSchema(repo, master.TenantDomain{}.TableSchema(constant.MasterDatabase))
	createTableUsingSchema(repo, master.SchemaMigration{}.TableSchema(constant.MasterDatabase))
	createTableUsingSchema(repo, master.GlobalAuditLog{}.TableSchema(constant.MasterDatabase))
	createTableUsingSchema(repo, master.UserResetToken{}.TableSchema(constant.MasterDatabase))

	// create required roles
	if err := createDefaultRoles(repo); err != nil {
		fmt.Printf("Error while creating default roles: %v\n", err)
	}

	createFunctions(repo, constant.MasterDatabase)

	// create default subscription plans
	if err := createDefaultPlans(repo); err != nil {
		fmt.Printf("Error while creating default plans: %v\n", err)
	}
}
