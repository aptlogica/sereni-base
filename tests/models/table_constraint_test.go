package models_test

import (
	"testing"

	"github.com/aptlogica/sereni-base/internal/models/tenant"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTableConstraint_TableName(t *testing.T) {
	assert.Equal(t, `"tenant_1".table_constraints`, tenant.TableConstraint{}.TableName("tenant_1"))
}

func TestTableConstraint_TableSchema(t *testing.T) {
	schema := tenant.TableConstraint{}.TableSchema("tenant_1")
	assert.Equal(t, `"tenant_1".table_constraints`, schema.Name)

	types := map[string]string{}
	notNull := map[string]bool{}
	unique := map[string]bool{}
	for _, col := range schema.Columns {
		types[col.Name] = col.DataType
		notNull[col.Name] = col.NotNull
		unique[col.Name] = col.Unique
	}
	assert.Equal(t, "jsonb", types["column_ids"])
	assert.NotContains(t, types, "options")
	for _, name := range []string{"id", "model_id", "type", "title", "constraint_name", "column_ids"} {
		assert.True(t, notNull[name], "%s should be NOT NULL", name)
	}
	assert.True(t, unique["constraint_name"], "Postgres names are unique")

	// Deleting a table deletes its keys
	require.Len(t, schema.ForeignKeys, 1)
	assert.Equal(t, `"tenant_1".models`, schema.ForeignKeys[0].ReferencedTable)
	assert.Equal(t, "CASCADE", schema.ForeignKeys[0].OnDelete)
}

func TestKeyColumnBlockReason(t *testing.T) {
	allowed := []tenant.Column{
		{ColumnName: "email", UIDT: "email"},
		{ColumnName: "emp_code", UIDT: "text"},
		{ColumnName: "salary", UIDT: "decimal"},
		{ColumnName: "joined", UIDT: "date"},
		{ColumnName: "status", UIDT: "select"},
		{ColumnName: "title", UIDT: "text", System: true},
	}
	for _, c := range allowed {
		assert.Empty(t, tenant.KeyColumnBlockReason(c), c.ColumnName)
	}

	blocked := map[string]tenant.Column{
		"it is already unique":                       {ColumnName: "id", UIDT: "number", System: true},
		"link fields can't be part of a key":         {ColumnName: "dept", UIDT: "links"},
		"lookup fields can't be part of a key":       {ColumnName: "dept_name", UIDT: "lookup"},
		"formula fields can't be part of a key":      {ColumnName: "total", UIDT: "Formula"},
		"attachments can't be part of a key":         {ColumnName: "photo", UIDT: "attachment"},
		"multi select fields can't be part of a key": {ColumnName: "tags", UIDT: "multiSelect"},
		"system fields can't be part of a key":       {ColumnName: "created_time", UIDT: "createdTime", System: true},
	}
	for reason, c := range blocked {
		assert.Equal(t, reason, tenant.KeyColumnBlockReason(c), c.ColumnName)
	}

	assert.Equal(t, "system fields can't be part of a key",
		tenant.KeyColumnBlockReason(tenant.Column{ColumnName: "last_modified_time", UIDT: "datetime", System: true}),
		"system columns other than title are blocked whatever their type")
	assert.Empty(t, tenant.KeyColumnBlockReason(tenant.Column{ColumnName: "employee_id", UIDT: "number", Virtual: true}),
		"columns made with the table are flagged virtual but are ordinary columns")
}
