package models_test

import (
	"testing"

	"github.com/aptlogica/sereni-base/internal/models/tenant"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutomation_TableName(t *testing.T) {
	assert.Equal(t, `"tenant_1".automations`, tenant.Automation{}.TableName("tenant_1"))
}

func TestAutomation_Types(t *testing.T) {
	assert.Equal(t, "trigger", tenant.AutomationTypeTrigger)
	assert.Equal(t, "function", tenant.AutomationTypeFunction)
	assert.Equal(t, "webhook", tenant.AutomationTypeWebhook)
}

func TestAutomation_TableSchema(t *testing.T) {
	schema := tenant.Automation{}.TableSchema("tenant_1")

	assert.Equal(t, `"tenant_1".automations`, schema.Name)

	columns := map[string]bool{}
	for _, col := range schema.Columns {
		columns[col.Name] = col.NotNull
	}
	for _, name := range []string{"id", "model_id", "title", "type", "context", "event", "created_by", "last_modified_by", "created_time", "last_modified_time"} {
		_, ok := columns[name]
		assert.True(t, ok, "missing column %s", name)
	}
	for _, name := range []string{"id", "model_id", "type", "context", "created_time", "last_modified_time"} {
		assert.True(t, columns[name], "%s should be NOT NULL", name)
	}
	assert.False(t, columns["event"], "event is empty for functions and webhooks")

	require.Len(t, schema.Indexes, 1)
	assert.Equal(t, []string{"model_id"}, schema.Indexes[0].Columns)

	// Deleting a table deletes its automations
	require.Len(t, schema.ForeignKeys, 1)
	fk := schema.ForeignKeys[0]
	assert.Equal(t, []string{"model_id"}, fk.Columns)
	assert.Equal(t, `"tenant_1".models`, fk.ReferencedTable)
	assert.Equal(t, []string{"id"}, fk.ReferencedColumns)
	assert.Equal(t, "CASCADE", fk.OnDelete)
}
