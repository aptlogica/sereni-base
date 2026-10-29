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
