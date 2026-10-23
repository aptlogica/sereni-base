package models

import (
	"testing"

	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/stretchr/testify/assert"
)

func col(name, uidt string, system bool) tenant.Column {
	return tenant.Column{ColumnName: name, UIDT: uidt, System: system}
}

func order(c tenant.Column, index float64) tenant.Column {
	c.OrderIndex = &index
	return c
}

func TestIsLegacyTitle(t *testing.T) {
	assert.True(t, col("title", "text", true).IsLegacyTitle())
	// an ordinary Title, named like any column the user adds, is not the legacy one
	assert.False(t, col("title_1761123456_ab12", "text", false).IsLegacyTitle())
	assert.False(t, col("title", "text", false).IsLegacyTitle(), "a normal column named title is an ordinary column")
	assert.False(t, col("created_time", "datetime", true).IsLegacyTitle())
}

func TestIsDisplayColumnType(t *testing.T) {
	for _, uidt := range []string{"text", "longText", "number", "email", "select", "date", "formula", "createdBy"} {
		assert.True(t, tenant.IsDisplayColumnType(uidt), uidt)
	}
	for _, uidt := range []string{"links", "lookup", "rollup", "attachment", "json", "Links", "JSON"} {
		assert.False(t, tenant.IsDisplayColumnType(uidt), uidt)
	}
}

func TestIsDisplayColumn(t *testing.T) {
	deleted := col("gone", "text", false)
	deleted.Deleted = true

	assert.True(t, col("name", "text", false).IsDisplayColumn())
	assert.True(t, col("age", "number", false).IsDisplayColumn())
	assert.True(t, col("title", "text", true).IsDisplayColumn(), "the system Title of an older table")
	assert.True(t, col("id", "number", true).IsDisplayColumn(), "the system Id column can label records")
	assert.False(t, col("created_time", "datetime", true).IsDisplayColumn())
	assert.False(t, col("rel", "links", false).IsDisplayColumn())
	assert.False(t, col("files", "attachment", false).IsDisplayColumn())
	assert.False(t, deleted.IsDisplayColumn())
}

func TestFirstTextColumn(t *testing.T) {
	t.Run("picks the text column with the lowest order index", func(t *testing.T) {
		columns := []tenant.Column{
			order(col("id", "number", true), 0),
			order(col("notes", "longText", false), 5),
			order(col("name", "text", false), 2),
			order(col("age", "number", false), 1),
		}
		first, ok := tenant.FirstTextColumn(columns)
		assert.True(t, ok)
		assert.Equal(t, "name", first.ColumnName)
	})

	t.Run("a column without an order index comes last", func(t *testing.T) {
		columns := []tenant.Column{col("a", "text", false), order(col("b", "text", false), 9)}
		first, ok := tenant.FirstTextColumn(columns)
		assert.True(t, ok)
		assert.Equal(t, "b", first.ColumnName)
	})

	t.Run("keeps the first of equal order indexes", func(t *testing.T) {
		first, ok := tenant.FirstTextColumn([]tenant.Column{col("a", "text", false), col("b", "text", false)})
		assert.True(t, ok)
		assert.Equal(t, "a", first.ColumnName)
	})

	t.Run("an older table's system Title counts", func(t *testing.T) {
		first, ok := tenant.FirstTextColumn([]tenant.Column{
			order(col("id", "number", true), 0),
			order(col("title", "text", true), 1),
			order(col("name", "text", false), 6),
		})
		assert.True(t, ok)
		assert.Equal(t, "title", first.ColumnName)
	})

	t.Run("skips deleted, system and non-text columns", func(t *testing.T) {
		deleted := col("old", "text", false)
		deleted.Deleted = true
		_, ok := tenant.FirstTextColumn([]tenant.Column{
			deleted,
			col("created_by", "createdBy", true),
			col("age", "number", false),
			col("rel", "links", false),
		})
		assert.False(t, ok)
	})

	t.Run("an empty table has none", func(t *testing.T) {
		_, ok := tenant.FirstTextColumn(nil)
		assert.False(t, ok)
	})
}

func TestIDColumn(t *testing.T) {
	assert.True(t, col("id", "number", true).IsIDColumn())
	assert.False(t, col("id", "number", false).IsIDColumn(), "only the system column")
	assert.False(t, col("created_time", "datetime", true).IsIDColumn())

	id, ok := tenant.IDColumn([]tenant.Column{col("title", "text", true), col("id", "number", true)})
	assert.True(t, ok)
	assert.Equal(t, "id", id.ColumnName)

	deleted := col("id", "number", true)
	deleted.Deleted = true
	_, ok = tenant.IDColumn([]tenant.Column{deleted})
	assert.False(t, ok)
}
