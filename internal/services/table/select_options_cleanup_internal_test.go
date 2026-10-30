package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func opts(names ...string) []interface{} {
	out := make([]interface{}, 0, len(names))
	for _, n := range names {
		out = append(out, map[string]interface{}{"option": n, "color": "#fff"})
	}
	return out
}

func TestRemovedOptions(t *testing.T) {
	oldMeta := map[string]interface{}{"options": opts("1", "2", "3")}
	newMeta := map[string]interface{}{"options": opts("1", "3")}
	assert.Equal(t, []string{"2"}, removedOptions(oldMeta, newMeta))
	assert.Empty(t, removedOptions(oldMeta, oldMeta))
}

func TestDropRemovedFromDefault(t *testing.T) {
	single := map[string]interface{}{"defaultValue": "2"}
	dropRemovedFromDefault(single, []string{"2"})
	assert.NotContains(t, single, "defaultValue")

	keep := map[string]interface{}{"defaultValue": "1"}
	dropRemovedFromDefault(keep, []string{"2"})
	assert.Equal(t, "1", keep["defaultValue"])

	multi := map[string]interface{}{"defaultValue": []interface{}{"1", "2"}}
	dropRemovedFromDefault(multi, []string{"2"})
	assert.Equal(t, []interface{}{"1"}, multi["defaultValue"])

	all := map[string]interface{}{"defaultValue": []interface{}{"2"}}
	dropRemovedFromDefault(all, []string{"2"})
	assert.NotContains(t, all, "defaultValue")
}

func TestOptionNames(t *testing.T) {
	meta := map[string]interface{}{"options": []interface{}{"plain", map[string]interface{}{"option": "obj"}, map[string]interface{}{"color": "#fff"}, 7}}
	assert.Equal(t, []string{"plain", "obj"}, optionNames(meta))
	assert.Empty(t, optionNames(map[string]interface{}{}))
}

func TestDropRemovedFromDefault_KeepsNonStringItems(t *testing.T) {
	meta := map[string]interface{}{"defaultValue": []interface{}{"2", 5.0}}
	dropRemovedFromDefault(meta, []string{"2"})
	assert.Equal(t, []interface{}{5.0}, meta["defaultValue"])
}

func TestArrayDefaultSQL(t *testing.T) {
	t.Run("keeps bool and number items", func(t *testing.T) {
		lit, ok, err := arrayDefaultSQL("multiSelect", "TEXT[]", []interface{}{true, 1.5, "a"})
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, `'{"true","1.5","a"}'`, lit)
	})
	t.Run("an empty list sets no default", func(t *testing.T) {
		_, ok, err := arrayDefaultSQL("multiSelect", "TEXT[]", []interface{}{})
		assert.NoError(t, err)
		assert.False(t, ok)
	})
	t.Run("rejects an item that is not a scalar", func(t *testing.T) {
		_, _, err := arrayDefaultSQL("multiSelect", "TEXT[]", []interface{}{map[string]interface{}{"a": 1}})
		assert.Error(t, err)
	})
	t.Run("rejects other field types and non-array columns", func(t *testing.T) {
		_, _, err := arrayDefaultSQL("text", "TEXT[]", []interface{}{"a"})
		assert.Error(t, err)
		_, _, err = arrayDefaultSQL("multiSelect", "TEXT", []interface{}{"a"})
		assert.Error(t, err)
	})
}
