package services

import (
	"context"
	"fmt"

	"github.com/lib/pq"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
)

const metaOptionsKey = "options"

// optionNames lists the option labels of a select/multiSelect meta. Options are
// either {"option": "x", ...} objects or plain strings.
func optionNames(meta map[string]interface{}) []string {
	list, _ := meta[metaOptionsKey].([]interface{})
	names := make([]string, 0, len(list))
	for _, o := range list {
		switch v := o.(type) {
		case string:
			names = append(names, v)
		case map[string]interface{}:
			if name, ok := v["option"].(string); ok {
				names = append(names, name)
			}
		}
	}
	return names
}

// removedOptions returns the options present in oldMeta but missing from newMeta.
func removedOptions(oldMeta, newMeta map[string]interface{}) []string {
	kept := map[string]bool{}
	for _, n := range optionNames(newMeta) {
		kept[n] = true
	}
	var removed []string
	for _, n := range optionNames(oldMeta) {
		if !kept[n] {
			removed = append(removed, n)
		}
	}
	return removed
}

// dropRemovedFromDefault removes deleted options from meta.defaultValue, so a
// deleted option can't stay behind as the column default.
func dropRemovedFromDefault(meta map[string]interface{}, removed []string) {
	gone := map[string]bool{}
	for _, r := range removed {
		gone[r] = true
	}
	switch v := meta[metaDefaultValueKey].(type) {
	case string:
		if gone[v] {
			delete(meta, metaDefaultValueKey)
		}
	case []interface{}:
		left := make([]interface{}, 0, len(v))
		for _, it := range v {
			if s, ok := it.(string); ok && gone[s] {
				continue
			}
			left = append(left, it)
		}
		if len(left) == 0 {
			delete(meta, metaDefaultValueKey)
		} else {
			meta[metaDefaultValueKey] = left
		}
	}
}

// clearRemovedOptionValues empties the cells of a select/multiSelect column that
// still hold a deleted option.
func (s tableManagementService) clearRemovedOptionValues(ctx context.Context, schemaName, alias, columnName, uidt string, removed []string) error {
	table := fmt.Sprintf(SchemaTableFormat, schemaName, alias)
	col := pq.QuoteIdentifier(columnName)

	var query string
	if uidt == "multiSelect" {
		query = fmt.Sprintf(`UPDATE %[1]s SET %[2]s = NULLIF(ARRAY(SELECT x FROM unnest(%[2]s) AS x WHERE x <> ALL($1::text[])), '{}')
			WHERE %[2]s && $1::text[]`, table, col)
	} else {
		query = fmt.Sprintf(`UPDATE %[1]s SET %[2]s = NULL WHERE %[2]s = ANY($1::text[])`, table, col)
	}
	if _, err := s.repo.DB.ExecContext(ctx, query, pq.Array(removed)); err != nil {
		return app_errors.LogDatabaseError(err, "failed to clear removed option values")
	}
	return nil
}
