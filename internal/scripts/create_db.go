// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package scripts

import (
	"context"
	"fmt"
	"strings"

	"github.com/aptlogica/go-postgres-rest/pkg"
	"github.com/aptlogica/go-postgres-rest/pkg/models"
	appConstant "github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
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

// createFunctions installs DefinedFunctions with CREATE OR REPLACE, so changed function bodies reach
// existing databases on the next start. Superseded signatures are dropped first.
func createFunctions(repo *pkg.DatabaseService, schemaName string) error {
	for _, signature := range appConstant.DroppedFunctions {
		if _, err := repo.DB.Exec(fmt.Sprintf("DROP FUNCTION IF EXISTS \"%s\".%s", schemaName, signature)); err != nil {
			fmt.Printf("error dropping function '%s': %v\n", signature, err)
		}
	}
	for _, function := range appConstant.DefinedFunctions {
		functionName := fmt.Sprintf("\"%s\".%s(%s)", schemaName, function.FunctionName, function.FunctionParams)
		create := fmt.Sprintf("CREATE OR REPLACE FUNCTION %s %s", functionName, function.FunctionSQL)
		_, err := repo.DB.Exec(create)
		if err != nil && contains(err.Error(), "cannot change") {
			// A changed return type or parameter name can't be replaced in place
			if _, err = repo.DB.Exec("DROP FUNCTION IF EXISTS " + functionName); err == nil {
				_, err = repo.DB.Exec(create)
			}
		}
		if err != nil {
			fmt.Printf("error creating function '%s': %v\n", function.FunctionName, err)
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
	createTableUsingSchema(dbService, tenant.Automation{}.TableSchema(appConstant.MasterDatabase))
	createTableUsingSchema(dbService, tenant.TableConstraint{}.TableSchema(appConstant.MasterDatabase))

	createFunctions(dbService, appConstant.MasterDatabase)

	if err := MigrateLinkKeys(dbService, appConstant.MasterDatabase); err != nil {
		fmt.Printf("⚠ Warning: link key migration failed: %v\n", err)
	}
}
