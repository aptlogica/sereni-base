package models_test

import (
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"testing"
)

// TestBaseTableName tests Base model TableName method
func TestBaseTableName(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		expected string
	}{
		{"with prefix", "tenant1", `"tenant1".bases`},
		{"empty prefix", "", `"".bases`},
		{"special prefix", "test_schema", `"test_schema".bases`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := tenant.Base{}
			result := base.TableName(tt.prefix)
			if result != tt.expected {
				t.Errorf("TableName(%q) = %q, want %q", tt.prefix, result, tt.expected)
			}
		})
	}
}

// TestBaseTableSchema tests Base model TableSchema method
func TestBaseTableSchema(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
	}{
		{"with tenant prefix", "tenant1"},
		{"with default prefix", "public"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := tenant.Base{}
			schema := base.TableSchema(tt.prefix)

			if schema.Name == "" {
				t.Error("TableSchema() returned empty name")
			}

			if len(schema.Columns) == 0 {
				t.Error("TableSchema() returned no columns")
			}

			// Verify essential columns exist
			essentialColumns := []string{"id", "workspace_id", "title", "type", "status", "visibility"}
			columnMap := make(map[string]bool)
			for _, col := range schema.Columns {
				columnMap[col.Name] = true
			}

			for _, essential := range essentialColumns {
				if !columnMap[essential] {
					t.Errorf("TableSchema() missing essential column: %s", essential)
				}
			}

			// Verify indexes exist
			if len(schema.Indexes) == 0 {
				t.Error("TableSchema() returned no indexes")
			}

			// Verify foreign keys exist
			if len(schema.ForeignKeys) == 0 {
				t.Error("TableSchema() returned no foreign keys")
			}
		})
	}
}

// TestStrPtrNilHandling tests StrPtr doesn't return nil
func TestStrPtrNilHandling(t *testing.T) {
	ptr := tenant.StrPtr("test")
	if ptr == nil {
		t.Fatal("StrPtr() should never return nil")
	}

	// Modify the value through pointer
	*ptr = "modified"
	if *ptr != "modified" {
		t.Error("Failed to modify value through pointer")
	}
}

// TestCreateIntegerColumn tests CreateIntegerColumn function
func TestCreateIntegerColumn(t *testing.T) {
	col := tenant.CreateIntegerColumn("test_column")

	if col.Name != "test_column" {
		t.Errorf("Expected name 'test_column', got '%s'", col.Name)
	}
	if col.DataType != "integer" {
		t.Errorf("Expected data type 'integer', got '%s'", col.DataType)
	}
	if col.DefaultValue == nil || *col.DefaultValue != "0" {
		t.Errorf("Expected default value '0', got %v", col.DefaultValue)
	}
}

// TestCreateBooleanColumn tests CreateBooleanColumn function
func TestCreateBooleanColumn(t *testing.T) {
	col := tenant.CreateBooleanColumn("is_active")

	if col.Name != "is_active" {
		t.Errorf("Expected name 'is_active', got '%s'", col.Name)
	}
	if col.DataType != "boolean" {
		t.Errorf("Expected data type 'boolean', got '%s'", col.DataType)
	}
	if col.DefaultValue == nil || *col.DefaultValue != "false" {
		t.Errorf("Expected default value 'false', got %v", col.DefaultValue)
	}
}

// TestCreateTimestampColumn tests CreateTimestampColumn function
func TestCreateTimestampColumn(t *testing.T) {
	tests := []struct {
		name            string
		columnName      string
		notNull         bool
		useNull         bool
		expectedDefault *string
	}{
		{"not null with current timestamp", "created_at", true, false, tenant.StrPtr("CURRENT_TIMESTAMP")},
		{"nullable with null default", "updated_at", false, true, tenant.StrPtr("NULL")},
		{"nullable without default", "deleted_at", false, false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col := tenant.CreateTimestampColumn(tt.columnName, tt.notNull, tt.useNull)

			if col.Name != tt.columnName {
				t.Errorf("Expected name '%s', got '%s'", tt.columnName, col.Name)
			}
			if col.DataType != "timestamp" {
				t.Errorf("Expected data type 'timestamp', got '%s'", col.DataType)
			}
			if col.NotNull != tt.notNull {
				t.Errorf("Expected NotNull %v, got %v", tt.notNull, col.NotNull)
			}

			if tt.expectedDefault == nil && col.DefaultValue != nil {
				t.Errorf("Expected nil default value, got %v", col.DefaultValue)
			} else if tt.expectedDefault != nil && (col.DefaultValue == nil || *col.DefaultValue != *tt.expectedDefault) {
				t.Errorf("Expected default value '%s', got %v", *tt.expectedDefault, col.DefaultValue)
			}
		})
	}
}

// TestCreateUUIDIDColumn tests CreateUUIDIDColumn function
func TestCreateUUIDIDColumn(t *testing.T) {
	col := tenant.CreateUUIDIDColumn()

	if col.Name != "id" {
		t.Errorf("Expected name 'id', got '%s'", col.Name)
	}
	if col.DataType != "uuid" {
		t.Errorf("Expected data type 'uuid', got '%s'", col.DataType)
	}
	if !col.NotNull {
		t.Error("Expected NotNull to be true")
	}
	if !col.Unique {
		t.Error("Expected Unique to be true")
	}
}

// TestCreateVarcharIDColumn tests createVarcharIDColumn function
func TestCreateVarcharIDColumn(t *testing.T) {
	col := tenant.CreateVarcharIDColumn()

	if col.Name != "id" {
		t.Errorf("Expected name 'id', got '%s'", col.Name)
	}
	if col.DataType != "varchar" {
		t.Errorf("Expected data type 'varchar', got '%s'", col.DataType)
	}
	if !col.NotNull {
		t.Error("Expected NotNull to be true")
	}
	if !col.Unique {
		t.Error("Expected Unique to be true")
	}
}

// TestUserTableName tests User model TableName method
func TestUserTableName(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		expected string
	}{
		{"with prefix", "tenant1", `"tenant1".users`},
		{"empty prefix", "", `"".users`},
		{"special prefix", "test_schema", `"test_schema".users`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := tenant.User{}
			result := user.TableName(tt.prefix)
			if result != tt.expected {
				t.Errorf("TableName(%q) = %q, want %q", tt.prefix, result, tt.expected)
			}
		})
	}
}

// TestUserTableSchema tests User model TableSchema method
func TestUserTableSchema(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
	}{
		{"with tenant prefix", "tenant1"},
		{"with default prefix", "public"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := tenant.User{}
			schema := user.TableSchema(tt.prefix)

			if schema.Name == "" {
				t.Error("TableSchema() returned empty name")
			}

			if len(schema.Columns) == 0 {
				t.Error("TableSchema() returned no columns")
			}

			// Verify essential columns exist
			essentialColumns := []string{"id", "email", "first_name", "last_name", "status", "created_time"}
			columnMap := make(map[string]bool)
			for _, col := range schema.Columns {
				columnMap[col.Name] = true
			}

			for _, essential := range essentialColumns {
				if !columnMap[essential] {
					t.Errorf("TableSchema() missing essential column: %s", essential)
				}
			}
		})
	}
}

// TestOrganizationTableName tests Organization model TableName method
func TestOrganizationTableName(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		expected string
	}{
		{"with prefix", "tenant1", `"tenant1".organizations`},
		{"empty prefix", "", `"".organizations`},
		{"special prefix", "test_schema", `"test_schema".organizations`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org := tenant.Organization{}
			result := org.TableName(tt.prefix)
			if result != tt.expected {
				t.Errorf("TableName(%q) = %q, want %q", tt.prefix, result, tt.expected)
			}
		})
	}
}

// TestOrganizationTableSchema tests Organization model TableSchema method
func TestOrganizationTableSchema(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
	}{
		{"with tenant prefix", "tenant1"},
		{"with default prefix", "public"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org := tenant.Organization{}
			schema := org.TableSchema(tt.prefix)

			if schema.Name == "" {
				t.Error("TableSchema() returned empty name")
			}

			if len(schema.Columns) == 0 {
				t.Error("TableSchema() returned no columns")
			}

			// Verify essential columns exist
			essentialColumns := []string{"id", "name", "email", "status", "created_time"}
			columnMap := make(map[string]bool)
			for _, col := range schema.Columns {
				columnMap[col.Name] = true
			}

			for _, essential := range essentialColumns {
				if !columnMap[essential] {
					t.Errorf("TableSchema() missing essential column: %s", essential)
				}
			}

			// Verify indexes exist
			if len(schema.Indexes) == 0 {
				t.Error("TableSchema() returned no indexes")
			}
		})
	}
}

// TestAssetsTableName tests Assets model TableName method
func TestAssetsTableName(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		expected string
	}{
		{"with prefix", "tenant1", `"tenant1".assets`},
		{"empty prefix", "", `"".assets`},
		{"special prefix", "test_schema", `"test_schema".assets`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assets := tenant.Assets{}
			result := assets.TableName(tt.prefix)
			if result != tt.expected {
				t.Errorf("TableName(%q) = %q, want %q", tt.prefix, result, tt.expected)
			}
		})
	}
}

// TestAssetsTableSchema tests Assets model TableSchema method
func TestAssetsTableSchema(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
	}{
		{"with tenant prefix", "tenant1"},
		{"with default prefix", "public"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assets := tenant.Assets{}
			schema := assets.TableSchema(tt.prefix)

			if schema.Name == "" {
				t.Error("TableSchema() returned empty name")
			}

			if len(schema.Columns) == 0 {
				t.Error("TableSchema() returned no columns")
			}

			// Verify essential columns exist
			essentialColumns := []string{"id", "title", "url", "thumbnail_url", "base_path", "created_time"}
			columnMap := make(map[string]bool)
			for _, col := range schema.Columns {
				columnMap[col.Name] = true
			}

			for _, essential := range essentialColumns {
				if !columnMap[essential] {
					t.Errorf("TableSchema() missing essential column: %s", essential)
				}
			}
		})
	}
}

// TestColumnTableName tests Column model TableName method
func TestColumnTableName(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		expected string
	}{
		{"with prefix", "tenant1", `"tenant1".columns`},
		{"empty prefix", "", `"".columns`},
		{"special prefix", "test_schema", `"test_schema".columns`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			column := tenant.Column{}
			result := column.TableName(tt.prefix)
			if result != tt.expected {
				t.Errorf("TableName(%q) = %q, want %q", tt.prefix, result, tt.expected)
			}
		})
	}
}

// TestColumnTableSchema tests Column model TableSchema method
func TestColumnTableSchema(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
	}{
		{"with tenant prefix", "tenant1"},
		{"with default prefix", "public"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			column := tenant.Column{}
			schema := column.TableSchema(tt.prefix)

			if schema.Name == "" {
				t.Error("TableSchema() returned empty name")
			}

			if len(schema.Columns) == 0 {
				t.Error("TableSchema() returned no columns")
			}

			// Verify essential columns exist
			essentialColumns := []string{"id", "model_id", "base_id", "column_name", "title", "created_time"}
			columnMap := make(map[string]bool)
			for _, col := range schema.Columns {
				columnMap[col.Name] = true
			}

			for _, essential := range essentialColumns {
				if !columnMap[essential] {
					t.Errorf("TableSchema() missing essential column: %s", essential)
				}
			}

			// Verify indexes exist
			if len(schema.Indexes) == 0 {
				t.Error("TableSchema() returned no indexes")
			}

			// Verify foreign keys exist
			if len(schema.ForeignKeys) == 0 {
				t.Error("TableSchema() returned no foreign keys")
			}
		})
	}
}
