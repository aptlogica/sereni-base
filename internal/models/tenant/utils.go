package tenant

import (
	"go-postgres-rest/pkg/models"
)

func StrPtr(s string) *string {
	return &s
}

// createIntegerColumn creates an integer column with default 0
func createIntegerColumn(name string) models.ColumnDefinition {
	return models.ColumnDefinition{Name: name, DataType: "integer", DefaultValue: StrPtr("0")}
}

// createBooleanColumn creates a boolean column with default false
func createBooleanColumn(name string) models.ColumnDefinition {
	return models.ColumnDefinition{Name: name, DataType: "boolean", DefaultValue: StrPtr("false")}
}

// createTimestampColumn creates a timestamp column definition with optional null default
func createTimestampColumn(name string, notNull bool, useNull bool) models.ColumnDefinition {
	null := "NULL"
	var defaultVal *string
	if useNull {
		defaultVal = &null
	} else if notNull {
		defaultVal = StrPtr("CURRENT_TIMESTAMP")
	}
	return models.ColumnDefinition{Name: name, DataType: "timestamp", NotNull: notNull, DefaultValue: defaultVal}
}

// createUUIDIDColumn creates a UUID primary key column definition
func createUUIDIDColumn() models.ColumnDefinition {
	return models.ColumnDefinition{Name: "id", DataType: "uuid", NotNull: true, Unique: true}
}

// createVarcharIDColumn creates a varchar primary key column definition
func createVarcharIDColumn() models.ColumnDefinition {
	return models.ColumnDefinition{Name: "id", DataType: "varchar", NotNull: true, Unique: true}
}
