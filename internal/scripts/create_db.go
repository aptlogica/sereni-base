package scripts

import (
	"context"
	"fmt"
	"godbgrest/pkg"
	"godbgrest/pkg/models"
	appConstant "serenibase/internal/constant"
	"serenibase/internal/models/tenant"
	"strings"
)

// contains checks if a string contains a substring (case-insensitive)
func contains(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}

func createMasterSchema(repo *pkg.DatabaseService) {
	ctx := context.Background()
	masterSchema := appConstant.MasterDatabase
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
	for _, function := range appConstant.DefinedFunctions {
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

func CreateMasterSchema(dbService *pkg.DatabaseService) {
	// create master schema
	createMasterSchema(dbService)

	// Create all tables
	createTableUsingSchema(dbService, tenant.Organization{}.TableSchema(appConstant.MasterDatabase))
	createTableUsingSchema(dbService, tenant.User{}.TableSchema(appConstant.MasterDatabase))
	createTableUsingSchema(dbService, tenant.UsageMetric{}.TableSchema(appConstant.MasterDatabase))
	createTableUsingSchema(dbService, tenant.GlobalAuditLog{}.TableSchema(appConstant.MasterDatabase))
	createTableUsingSchema(dbService, tenant.UserResetToken{}.TableSchema(appConstant.MasterDatabase))

	// Create RBAC tables
	createTableUsingSchema(dbService, tenant.Resource{}.TableSchema(appConstant.MasterDatabase))
	createTableUsingSchema(dbService, tenant.Action{}.TableSchema(appConstant.MasterDatabase))
	createTableUsingSchema(dbService, tenant.AccessRole{}.TableSchema(appConstant.MasterDatabase))
	createTableUsingSchema(dbService, tenant.Permission{}.TableSchema(appConstant.MasterDatabase))
	createTableUsingSchema(dbService, tenant.RolePermission{}.TableSchema(appConstant.MasterDatabase))
	createTableUsingSchema(dbService, tenant.AccessMember{}.TableSchema(appConstant.MasterDatabase))

	// serenibase tables
	createTableUsingSchema(dbService, tenant.Assets{}.TableSchema(appConstant.MasterDatabase))
	createTableUsingSchema(dbService, tenant.Workspace{}.TableSchema(appConstant.MasterDatabase))
	createTableUsingSchema(dbService, tenant.Base{}.TableSchema(appConstant.MasterDatabase))
	createTableUsingSchema(dbService, tenant.WorkspaceMember{}.TableSchema(appConstant.MasterDatabase))
	createTableUsingSchema(dbService, tenant.Model{}.TableSchema(appConstant.MasterDatabase))
	createTableUsingSchema(dbService, tenant.Column{}.TableSchema(appConstant.MasterDatabase))
	createTableUsingSchema(dbService, tenant.View{}.TableSchema(appConstant.MasterDatabase))
	createTableUsingSchema(dbService, tenant.Relation{}.TableSchema(appConstant.MasterDatabase))

	createFunctions(dbService, appConstant.MasterDatabase)

}
