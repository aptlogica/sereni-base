// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
package constant_test

import (
	"testing"

	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/stretchr/testify/assert"
)

func TestDatabaseConstantsExist(t *testing.T) {
	// Verify database constants are properly defined and can be accessed
	assert.NotEmpty(t, constant.MasterDatabase)
	assert.NotEmpty(t, constant.ModelsTableFormat)
}

func TestDBTypeConstantsExist(t *testing.T) {
	// Verify database type constants are defined
	assert.NotEmpty(t, constant.DBTypeVarchar255)
	assert.NotEmpty(t, constant.DBTypeNVarchar255)
	assert.NotEmpty(t, constant.DBTypeNVarcharMax)
	assert.NotEmpty(t, constant.DBTypeVarchar255Lower)
	assert.NotEmpty(t, constant.DBTypeVarchar100)
	assert.NotEmpty(t, constant.DBTypeVarchar50)
	assert.NotEmpty(t, constant.DBTypeVarchar150)
	assert.NotEmpty(t, constant.DBTypeOracleVarchar255)
}

func TestScopeLevelsExist(t *testing.T) {
	// Verify RBAC scope levels are defined
	assert.NotEmpty(t, constant.ScopeLevels.System)
	assert.NotEmpty(t, constant.ScopeLevels.Workspace)
	assert.NotEmpty(t, constant.ScopeLevels.Base)
}

func TestRBACRoleNamesExist(t *testing.T) {
	// Verify RBAC role names are defined
	assert.NotEmpty(t, constant.RBACRoleNames.Owner)
	assert.NotEmpty(t, constant.RBACRoleNames.CoOwner)
	assert.NotEmpty(t, constant.RBACRoleNames.WorkspaceMaintainer)
	assert.NotEmpty(t, constant.RBACRoleNames.WorkspaceMaintainerRO)
	assert.NotEmpty(t, constant.RBACRoleNames.BaseMember)
	assert.NotEmpty(t, constant.RBACRoleNames.BaseMemberReadOnly)
	assert.NotEmpty(t, constant.RBACRoleNames.NoAccess)
}

func TestResourceAndActionCodesExist(t *testing.T) {
	// Verify resource and action codes are defined
	assert.NotEmpty(t, constant.ResourceCodes.Workspace)
	assert.NotEmpty(t, constant.ResourceCodes.Base)
	assert.NotEmpty(t, constant.ResourceCodes.Table)
	assert.NotEmpty(t, constant.ResourceCodes.Records)
	assert.NotEmpty(t, constant.ResourceCodes.Members)
	assert.NotEmpty(t, constant.ResourceCodes.Views)
	assert.NotEmpty(t, constant.ResourceCodes.Settings)
	assert.NotEmpty(t, constant.ResourceCodes.ApiTokens)
	assert.NotEmpty(t, constant.ResourceCodes.Webhooks)
	assert.NotEmpty(t, constant.ResourceCodes.Automations)

	assert.NotEmpty(t, constant.ActionCodes.Read)
	assert.NotEmpty(t, constant.ActionCodes.Create)
	assert.NotEmpty(t, constant.ActionCodes.Update)
	assert.NotEmpty(t, constant.ActionCodes.Delete)
	assert.NotEmpty(t, constant.ActionCodes.Share)
	assert.NotEmpty(t, constant.ActionCodes.Invite)
	assert.NotEmpty(t, constant.ActionCodes.Export)
	assert.NotEmpty(t, constant.ActionCodes.Import)
	assert.NotEmpty(t, constant.ActionCodes.Execute)
	assert.NotEmpty(t, constant.ActionCodes.Manage)
}

func TestDefaultAccessRolesExist(t *testing.T) {
	// Verify default access roles are defined
	assert.Greater(t, len(constant.DefaultAccessRoles), 0)
}

func TestSystemColumnsExist(t *testing.T) {
	// Verify system columns are defined
	assert.Greater(t, len(constant.SystemColumns), 0)
	for _, col := range constant.SystemColumns {
		assert.NotNil(t, col.System)
	}
}

func TestUITypeMappingsExist(t *testing.T) {
	// Verify UI type mappings are defined
	textMapping, ok := constant.UITypeMappings["text"]
	assert.True(t, ok)
	assert.NotEmpty(t, textMapping.Postgres)
}
