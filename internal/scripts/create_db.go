package scripts

import (
	"context"
	"fmt"
	"godbgrest/pkg"
	"godbgrest/pkg/models"
	"serenibase/internal/constant"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	"serenibase/internal/utils/helpers"
	"strings"
	"time"

	dbModels "godbgrest/pkg/models"

	"github.com/google/uuid"
)

// contains checks if a string contains a substring (case-insensitive)
func contains(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}

func createRole(repo *pkg.DatabaseService, req dto.RoleInsertion) (tenant.Role, error) {
	ctx := context.Background()
	tableName := tenant.Role{}.TableName(constant.MasterDatabase)

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
		return tenant.Role{}, err
	}

	var insertedRole tenant.Role
	if err := helpers.MapToStruct(insertedRoleData, &insertedRole); err != nil {
		return tenant.Role{}, err
	}
	return insertedRole, nil
}

func isRoleExists(repo *pkg.DatabaseService, name string) (bool, error) {
	ctx := context.Background()
	limit := 1
	tableName := tenant.Role{}.TableName(constant.MasterDatabase)
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
		// Only log if it's not an "already exists" error
		errMsg := err.Error()
		if !contains(errMsg, "already exists") {
			fmt.Printf("Error creating table %s: %v\n", schema.Name, err)
		}
	} else {
		fmt.Printf("Table %s created successfully.\n", schema.Name)
	}
}

func createFunctions(repo *pkg.DatabaseService, schemaName string) error {
	ctx := context.Background()
	for _, function := range constant.DefinedFunctions {
		functionName := fmt.Sprintf("\"%s\".%s(%s)", schemaName, function.FunctionName, function.FunctionParams)
		err := repo.TableService.CreateFunction(ctx, functionName, function.FunctionSQL)
		if err != nil {
			// Only log if it's not an "already exists" error
			if !contains(err.Error(), "already exists") {
				fmt.Printf("error creating function '%s': %v\n", function.FunctionName, err)
			}
		}
	}
	return nil
}

func CreateMasterSchema(repo *pkg.DatabaseService) {
	// create master schema
	createMasterSchema(repo)

	// Create all tables
	createTableUsingSchema(repo, tenant.User{}.TableSchema(constant.MasterDatabase))
	createTableUsingSchema(repo, tenant.UsageMetric{}.TableSchema(constant.MasterDatabase))
	createTableUsingSchema(repo, tenant.GlobalAuditLog{}.TableSchema(constant.MasterDatabase))
	createTableUsingSchema(repo, tenant.UserResetToken{}.TableSchema(constant.MasterDatabase))

	// Create RBAC tables
	createTableUsingSchema(repo, tenant.Resource{}.TableSchema(constant.MasterDatabase))
	createTableUsingSchema(repo, tenant.Action{}.TableSchema(constant.MasterDatabase))
	createTableUsingSchema(repo, tenant.AccessRole{}.TableSchema(constant.MasterDatabase))
	createTableUsingSchema(repo, tenant.Permission{}.TableSchema(constant.MasterDatabase))
	createTableUsingSchema(repo, tenant.RolePermission{}.TableSchema(constant.MasterDatabase))
	createTableUsingSchema(repo, tenant.AccessMember{}.TableSchema(constant.MasterDatabase))

	// serenibase tables
	createTableUsingSchema(repo, tenant.Assets{}.TableSchema(constant.MasterDatabase))
	createTableUsingSchema(repo, tenant.Workspace{}.TableSchema(constant.MasterDatabase))
	createTableUsingSchema(repo, tenant.Base{}.TableSchema(constant.MasterDatabase))
	createTableUsingSchema(repo, tenant.WorkspaceMember{}.TableSchema(constant.MasterDatabase))
	createTableUsingSchema(repo, tenant.Model{}.TableSchema(constant.MasterDatabase))
	createTableUsingSchema(repo, tenant.Column{}.TableSchema(constant.MasterDatabase))
	createTableUsingSchema(repo, tenant.View{}.TableSchema(constant.MasterDatabase))
	createTableUsingSchema(repo, tenant.Relation{}.TableSchema(constant.MasterDatabase))

	// create required roles
	if err := createDefaultRoles(repo); err != nil {
		fmt.Printf("Error while creating default roles: %v\n", err)
	}

	createFunctions(repo, constant.MasterDatabase)

}
