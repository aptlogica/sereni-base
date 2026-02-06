package tenant

import (
	"go-postgres-rest/pkg/models"
)

func StrPtr(s string) *string {
	return &s
}

// CreateIntegerColumn creates an integer column with default 0
func CreateIntegerColumn(name string) models.ColumnDefinition {
	return models.ColumnDefinition{Name: name, DataType: "integer", DefaultValue: StrPtr("0")}
}

// CreateBooleanColumn creates a boolean column with default false
func CreateBooleanColumn(name string) models.ColumnDefinition {
	return models.ColumnDefinition{Name: name, DataType: "boolean", DefaultValue: StrPtr("false")}
}

// CreateTimestampColumn creates a timestamp column definition with optional null default
func CreateTimestampColumn(name string, notNull bool, useNull bool) models.ColumnDefinition {
	null := "NULL"
	var defaultVal *string
	if useNull {
		defaultVal = &null
	} else if notNull {
		defaultVal = StrPtr("CURRENT_TIMESTAMP")
	}
	return models.ColumnDefinition{Name: name, DataType: "timestamp", NotNull: notNull, DefaultValue: defaultVal}
}

// CreateUUIDIDColumn creates a UUID primary key column definition
func CreateUUIDIDColumn() models.ColumnDefinition {
	return models.ColumnDefinition{Name: "id", DataType: "uuid", NotNull: true, Unique: true}
}

// CreateVarcharIDColumn creates a varchar primary key column definition
func CreateVarcharIDColumn() models.ColumnDefinition {
	return models.ColumnDefinition{Name: "id", DataType: "varchar", NotNull: true, Unique: true}
}
