package tests

import (
	"github.com/aptlogica/sereni-base/internal/constant"
	"testing"
)

// TestMasterDatabase tests the master database constant
func TestMasterDatabase(t *testing.T) {
	if constant.MasterDatabase != "public" {
		t.Errorf("MasterDatabase = %q, want %q", constant.MasterDatabase, "public")
	}
}

// TestEmailFooterNotice tests the email footer notice constant
func TestEmailFooterNotice(t *testing.T) {
	if constant.EmailFooterNotice == "" {
		t.Error("EmailFooterNotice should not be empty")
	}

	if len(constant.EmailFooterNotice) < 100 {
		t.Error("EmailFooterNotice seems too short")
	}
}

// TestDatabaseTypeConstants tests database type constants
func TestDatabaseTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"DBTypeVarchar255", constant.DBTypeVarchar255, "VARCHAR(255)"},
		{"DBTypeNVarchar255", constant.DBTypeNVarchar255, "NVARCHAR(255)"},
		{"DBTypeNVarcharMax", constant.DBTypeNVarcharMax, "NVARCHAR(MAX)"},
		{"DBTypeVarchar255Lower", constant.DBTypeVarchar255Lower, "varchar(255)"},
		{"DBTypeVarchar100", constant.DBTypeVarchar100, "varchar(100)"},
		{"DBTypeVarchar50", constant.DBTypeVarchar50, "varchar(50)"},
		{"DBTypeVarchar150", constant.DBTypeVarchar150, "varchar(150)"},
		{"DBTypeOracleVarchar255", constant.DBTypeOracleVarchar255, "VARCHAR2(255)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// TestModelsTableFormat tests the models table format constant
func TestModelsTableFormat(t *testing.T) {
	expected := "\"%s\".models"
	if constant.ModelsTableFormat != expected {
		t.Errorf("ModelsTableFormat = %q, want %q", constant.ModelsTableFormat, expected)
	}
}

// TestScopeLevels tests RBAC scope levels
func TestScopeLevels(t *testing.T) {
	if constant.ScopeLevels.System != "system" {
		t.Errorf("ScopeLevels.System = %q, want %q", constant.ScopeLevels.System, "system")
	}

	if constant.ScopeLevels.Workspace != "workspace" {
		t.Errorf("ScopeLevels.Workspace = %q, want %q", constant.ScopeLevels.Workspace, "workspace")
	}

	if constant.ScopeLevels.Base != "base" {
		t.Errorf("ScopeLevels.Base = %q, want %q", constant.ScopeLevels.Base, "base")
	}
}

// TestRBACRoleNames tests RBAC role names
func TestRBACRoleNames(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected string
	}{
		{"Owner", constant.RBACRoleNames.Owner, "owner"},
		{"CoOwner", constant.RBACRoleNames.CoOwner, "co-owner"},
		{"WorkspaceMaintainer", constant.RBACRoleNames.WorkspaceMaintainer, "maintainer"},
		{"WorkspaceMaintainerRO", constant.RBACRoleNames.WorkspaceMaintainerRO, "workspace-read"},
		{"BaseMember", constant.RBACRoleNames.BaseMember, "base-member"},
		{"BaseMemberReadOnly", constant.RBACRoleNames.BaseMemberReadOnly, "base-read"},
		{"NoAccess", constant.RBACRoleNames.NoAccess, "user"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actual != tt.expected {
				t.Errorf("RBACRoleNames.%s = %q, want %q", tt.name, tt.actual, tt.expected)
			}
		})
	}
}

// TestResourceCodes tests RBAC resource codes
func TestResourceCodes(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected string
	}{
		{"Workspace", constant.ResourceCodes.Workspace, "workspace"},
		{"Base", constant.ResourceCodes.Base, "base"},
		{"Table", constant.ResourceCodes.Table, "table"},
		{"Records", constant.ResourceCodes.Records, "records"},
		{"Members", constant.ResourceCodes.Members, "members"},
		{"Views", constant.ResourceCodes.Views, "views"},
		{"Settings", constant.ResourceCodes.Settings, "settings"},
		{"ApiTokens", constant.ResourceCodes.ApiTokens, "api_tokens"},
		{"Webhooks", constant.ResourceCodes.Webhooks, "webhooks"},
		{"Automations", constant.ResourceCodes.Automations, "automations"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actual != tt.expected {
				t.Errorf("ResourceCodes.%s = %q, want %q", tt.name, tt.actual, tt.expected)
			}
		})
	}
}

// TestActionCodes tests RBAC action codes
func TestActionCodes(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected string
	}{
		{"Read", constant.ActionCodes.Read, "read"},
		{"Create", constant.ActionCodes.Create, "create"},
		{"Update", constant.ActionCodes.Update, "update"},
		{"Delete", constant.ActionCodes.Delete, "delete"},
		{"Share", constant.ActionCodes.Share, "share"},
		{"Invite", constant.ActionCodes.Invite, "invite"},
		{"Export", constant.ActionCodes.Export, "export"},
		{"Import", constant.ActionCodes.Import, "import"},
		{"Execute", constant.ActionCodes.Execute, "execute"},
		{"Manage", constant.ActionCodes.Manage, "manage"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actual != tt.expected {
				t.Errorf("ActionCodes.%s = %q, want %q", tt.name, tt.actual, tt.expected)
			}
		})
	}
}

// TestDefaultAccessRoles tests default RBAC roles
func TestDefaultAccessRoles(t *testing.T) {
	if len(constant.DefaultAccessRoles) == 0 {
		t.Fatal("DefaultAccessRoles should not be empty")
	}

	// Test that we have expected number of default roles
	expectedRoleCount := 7
	if len(constant.DefaultAccessRoles) != expectedRoleCount {
		t.Errorf("DefaultAccessRoles count = %d, want %d", len(constant.DefaultAccessRoles), expectedRoleCount)
	}

	// Test each role has required fields
	for i, role := range constant.DefaultAccessRoles {
		t.Run(role.Name, func(t *testing.T) {
			if role.Name == "" {
				t.Errorf("Role at index %d has empty Name", i)
			}

			if role.ScopeLevel == "" {
				t.Errorf("Role %s has empty ScopeLevel", role.Name)
			}

			if role.Priority == 0 {
				t.Errorf("Role %s has zero Priority", role.Name)
			}

			// Verify scope level is valid
			validScopes := map[string]bool{
				constant.ScopeLevels.System:    true,
				constant.ScopeLevels.Workspace: true,
				constant.ScopeLevels.Base:      true,
			}

			if !validScopes[role.ScopeLevel] {
				t.Errorf("Role %s has invalid ScopeLevel: %s", role.Name, role.ScopeLevel)
			}
		})
	}

	// Test priority uniqueness
	priorityMap := make(map[int]string)
	for _, role := range constant.DefaultAccessRoles {
		if existingRole, exists := priorityMap[role.Priority]; exists {
			t.Errorf("Duplicate priority %d found for roles %s and %s", role.Priority, role.Name, existingRole)
		}
		priorityMap[role.Priority] = role.Name
	}
}

// TestSystemColumns tests system columns configuration
func TestSystemColumns(t *testing.T) {
	if len(constant.SystemColumns) == 0 {
		t.Fatal("SystemColumns should not be empty")
	}

	expectedColumns := 6 // Id, Title, Created Time, Last Modified Time, Created By, Last Modified By
	if len(constant.SystemColumns) != expectedColumns {
		t.Errorf("SystemColumns count = %d, want %d", len(constant.SystemColumns), expectedColumns)
	}

	// Test each column has required fields
	for i, col := range constant.SystemColumns {
		t.Run(col.Title, func(t *testing.T) {
			if col.Title == "" {
				t.Errorf("Column at index %d has empty Title", i)
			}

			if col.UIDT == "" {
				t.Errorf("Column %s has empty UIDT", col.Title)
			}

			if col.DT == "" {
				t.Errorf("Column %s has empty DT", col.Title)
			}

			if col.OrderIndex == nil {
				t.Errorf("Column %s has nil OrderIndex", col.Title)
			}

			if col.Virtual == nil {
				t.Errorf("Column %s has nil Virtual", col.Title)
			}

			if col.System == nil {
				t.Errorf("Column %s has nil System", col.Title)
			}
		})
	}

	// Test order index uniqueness and sequential ordering
	orderIndexMap := make(map[float64]string)
	for _, col := range constant.SystemColumns {
		if col.OrderIndex != nil {
			if existingCol, exists := orderIndexMap[*col.OrderIndex]; exists {
				t.Errorf("Duplicate OrderIndex %f found for columns %s and %s", *col.OrderIndex, col.Title, existingCol)
			}
			orderIndexMap[*col.OrderIndex] = col.Title
		}
	}
}

// TestDBMappingStructure tests DBMapping structure
func TestDBMappingStructure(t *testing.T) {
	mapping := constant.DBMapping{
		Component: "text",
		Label:     "Text",
		Postgres:  "TEXT",
		MongoDB:   "String",
		MySQL:     "VARCHAR(255)",
		SQLite:    "TEXT",
		MSSQL:     "NVARCHAR(255)",
		Oracle:    "VARCHAR2(255)",
	}

	if mapping.Component == "" {
		t.Error("Component should not be empty")
	}

	if mapping.Label == "" {
		t.Error("Label should not be empty")
	}

	if mapping.Postgres == "" {
		t.Error("Postgres type should not be empty")
	}

	if mapping.MongoDB == "" {
		t.Error("MongoDB type should not be empty")
	}

	if mapping.MySQL == "" {
		t.Error("MySQL type should not be empty")
	}

	if mapping.SQLite == "" {
		t.Error("SQLite type should not be empty")
	}

	if mapping.MSSQL == "" {
		t.Error("MSSQL type should not be empty")
	}

	if mapping.Oracle == "" {
		t.Error("Oracle type should not be empty")
	}
}

// TestRoleNamesUniqueness tests that all role names are unique
func TestRoleNamesUniqueness(t *testing.T) {
	roleNames := []string{
		constant.RBACRoleNames.Owner,
		constant.RBACRoleNames.CoOwner,
		constant.RBACRoleNames.WorkspaceMaintainer,
		constant.RBACRoleNames.WorkspaceMaintainerRO,
		constant.RBACRoleNames.BaseMember,
		constant.RBACRoleNames.BaseMemberReadOnly,
		constant.RBACRoleNames.NoAccess,
	}

	roleMap := make(map[string]bool)
	for _, name := range roleNames {
		if roleMap[name] {
			t.Errorf("Duplicate role name found: %s", name)
		}
		roleMap[name] = true
	}
}

// TestResourceCodesUniqueness tests that all resource codes are unique
func TestResourceCodesUniqueness(t *testing.T) {
	resourceCodes := []string{
		constant.ResourceCodes.Workspace,
		constant.ResourceCodes.Base,
		constant.ResourceCodes.Table,
		constant.ResourceCodes.Records,
		constant.ResourceCodes.Members,
		constant.ResourceCodes.Views,
		constant.ResourceCodes.Settings,
		constant.ResourceCodes.ApiTokens,
		constant.ResourceCodes.Webhooks,
		constant.ResourceCodes.Automations,
	}

	resourceMap := make(map[string]bool)
	for _, code := range resourceCodes {
		if resourceMap[code] {
			t.Errorf("Duplicate resource code found: %s", code)
		}
		resourceMap[code] = true
	}
}

// TestActionCodesUniqueness tests that all action codes are unique
func TestActionCodesUniqueness(t *testing.T) {
	actionCodes := []string{
		constant.ActionCodes.Read,
		constant.ActionCodes.Create,
		constant.ActionCodes.Update,
		constant.ActionCodes.Delete,
		constant.ActionCodes.Share,
		constant.ActionCodes.Invite,
		constant.ActionCodes.Export,
		constant.ActionCodes.Import,
		constant.ActionCodes.Execute,
		constant.ActionCodes.Manage,
	}

	actionMap := make(map[string]bool)
	for _, code := range actionCodes {
		if actionMap[code] {
			t.Errorf("Duplicate action code found: %s", code)
		}
		actionMap[code] = true
	}
}

// TestDefaultAccessRolesDefaultField tests the IsDefault field
func TestDefaultAccessRolesDefaultField(t *testing.T) {
	defaultCount := 0
	for _, role := range constant.DefaultAccessRoles {
		if role.IsDefault {
			defaultCount++
		}
	}

	// Only the "NoAccess" role should be default
	if defaultCount != 1 {
		t.Errorf("Expected exactly 1 default role, got %d", defaultCount)
	}

	// Verify NoAccess is the default role
	for _, role := range constant.DefaultAccessRoles {
		if role.Name == constant.RBACRoleNames.NoAccess && !role.IsDefault {
			t.Error("NoAccess role should be marked as default")
		}
	}
}

// TestUITypeMappings tests the UITypeMappings map
func TestUITypeMappings(t *testing.T) {
	if len(constant.UITypeMappings) == 0 {
		t.Error("UITypeMappings should not be empty")
	}

	// Test a few key mappings
	textMapping, exists := constant.UITypeMappings["text"]
	if !exists {
		t.Error("UITypeMappings should contain 'text'")
	} else {
		if textMapping.Component != "TextInput" {
			t.Errorf("text mapping component = %q, want %q", textMapping.Component, "TextInput")
		}
		if textMapping.Postgres != "TEXT" {
			t.Errorf("text mapping postgres = %q, want %q", textMapping.Postgres, "TEXT")
		}
	}

	numberMapping, exists := constant.UITypeMappings["number"]
	if !exists {
		t.Error("UITypeMappings should contain 'number'")
	} else {
		if numberMapping.Component != "NumberInput" {
			t.Errorf("number mapping component = %q, want %q", numberMapping.Component, "NumberInput")
		}
	}
}
