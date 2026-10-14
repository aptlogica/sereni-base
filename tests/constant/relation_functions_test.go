package constant_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/stretchr/testify/assert"
)

// TestLookupDataFunctionDefined checks the read function for lookups is registered for startup creation
// and matches link cells to target rows by their primary-key values.
func TestLookupDataFunctionDefined(t *testing.T) {
	var fn *constant.Function
	for i := range constant.DefinedFunctions {
		if constant.DefinedFunctions[i].FunctionName == "get_table_data_with_lookups" {
			fn = &constant.DefinedFunctions[i]
		}
	}
	if !assert.NotNil(t, fn, "get_table_data_with_lookups must be in DefinedFunctions") {
		return
	}

	assert.Equal(t, "schema_name TEXT, source_table_name TEXT, relation_data JSON[]", fn.FunctionParams)
	for _, part := range []string{"RETURNS JSON", "'targets'", "'alias'", "'is_array'", "jsonb_array_elements(s.%I) WITH ORDINALITY", "jsonb_populate_record", "public.link_key_join", "ORDER BY s.created_time"} {
		assert.True(t, strings.Contains(fn.FunctionSQL, part), "function SQL should contain %q", part)
	}
}

var linkFunctionNames = []string{
	"link_row_key", "link_lock_row", "link_apply_change", "link_rows_holding",
	"link_cycle_parent", "link_cycle_children", "link_rows", "get_table_data_with_lookups",
	"link_key_sync", "link_install_key_sync", "link_ids_to_keys",
}

// linkKeyHelperNames are the SQL-language key helpers; Postgres checks their bodies when they are created
var linkKeyHelperNames = []string{"link_key_columns", "link_key_expr", "link_key_join"}

func definedFunction(name string) (constant.Function, bool) {
	for _, fn := range constant.DefinedFunctions {
		if fn.FunctionName == name {
			return fn, true
		}
	}
	return constant.Function{}, false
}

func paramCount(params string) int {
	if strings.TrimSpace(params) == "" {
		return 0
	}
	return len(strings.Split(params, ","))
}

// callArgCount counts top-level arguments of the call whose "(" is at open.
func callArgCount(sql string, open int) int {
	depth, count, empty := 0, 1, true
	for i := open; i < len(sql); i++ {
		switch sql[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				if empty {
					return 0
				}
				return count
			}
		case ',':
			if depth == 1 {
				count++
			}
		case ' ', '\t', '\n', '\r':
		default:
			empty = false
		}
	}
	return -1
}

// TestLinkFunctionsDefined checks every link function is registered for startup creation.
func TestLinkFunctionsDefined(t *testing.T) {
	for _, name := range linkFunctionNames {
		fn, ok := definedFunction(name)
		if assert.True(t, ok, "%s must be in DefinedFunctions", name) {
			assert.Contains(t, fn.FunctionSQL, "LANGUAGE plpgsql", name)
		}
	}
}

// TestLinkFunctionCallsMatchSignatures checks that every public.<fn>(...) call inside the link
// functions refers to a defined function and passes the number of arguments it declares.
func TestLinkFunctionCallsMatchSignatures(t *testing.T) {
	callPattern := regexp.MustCompile(`public\.([a-z_]+)\(`)
	for _, name := range linkFunctionNames {
		fn, _ := definedFunction(name)
		for _, match := range callPattern.FindAllStringSubmatchIndex(fn.FunctionSQL, -1) {
			called := fn.FunctionSQL[match[2]:match[3]]
			target, ok := definedFunction(called)
			if !assert.True(t, ok, "%s calls undefined function %s", name, called) {
				continue
			}
			got := callArgCount(fn.FunctionSQL, match[1]-1)
			assert.Equal(t, paramCount(target.FunctionParams), got, "%s calls %s with the wrong number of arguments", name, called)
		}
	}
}

// TestLinkRowsFunctionContract checks the parts of link_rows the Go side relies on.
func TestLinkRowsFunctionContract(t *testing.T) {
	fn, _ := definedFunction("link_rows")
	assert.Contains(t, fn.FunctionSQL, "RETURNS TEXT")
	for _, status := range []string{"'ok'", "'source_not_found'", "'target_not_found'", "'cycle'"} {
		assert.Contains(t, fn.FunctionSQL, "RETURN "+status)
	}
	// Rows are locked in a fixed order before anything is written.
	assert.Contains(t, fn.FunctionSQL, `COLLATE "C"`)
	assert.Less(t, strings.Index(fn.FunctionSQL, "public.link_lock_row"), strings.Index(fn.FunctionSQL, "public.link_apply_change"))
	// Loop checks run before existing partners are released.
	assert.Less(t, strings.Index(fn.FunctionSQL, "public.link_cycle_children"), strings.Index(fn.FunctionSQL, "public.link_rows_holding"))
	assert.Contains(t, fn.FunctionSQL, "public.link_cycle_parent")
}

// TestLinkHelperFunctionsSQL checks the row-level helpers lock rows and update key arrays in place.
func TestLinkHelperFunctionsSQL(t *testing.T) {
	lock, _ := definedFunction("link_lock_row")
	assert.Contains(t, lock.FunctionSQL, "FOR UPDATE")

	holding, _ := definedFunction("link_rows_holding")
	assert.Equal(t, "p_table TEXT, p_column TEXT, p_key JSONB", holding.FunctionParams)
	assert.Contains(t, holding.FunctionSQL, "FOR UPDATE")
	assert.Contains(t, holding.FunctionSQL, "@> jsonb_build_array($1)")

	apply, _ := definedFunction("link_apply_change")
	assert.Contains(t, apply.FunctionParams, "p_key JSONB")
	for _, part := range []string{"@> jsonb_build_array($1)", "|| jsonb_build_array($1)", "x.e <> $1", "last_modified_time", "last_modified_by"} {
		assert.Contains(t, apply.FunctionSQL, part)
	}

	for _, name := range []string{"link_cycle_parent", "link_cycle_children"} {
		cycle, _ := definedFunction(name)
		assert.Contains(t, cycle.FunctionSQL, "WITH RECURSIVE")
		assert.Contains(t, cycle.FunctionSQL, "IF p_parent = p_child THEN")
	}

	// link_rows stores keys, not row ids
	rows, _ := definedFunction("link_rows")
	assert.Contains(t, rows.FunctionSQL, "source_key := public.link_row_key(p_source_table, p_source_row)")
	assert.Contains(t, rows.FunctionSQL, "target_key := public.link_row_key(p_target_table, p_target_row)")
}

// TestLinkKeyHelpersSQL checks the key helpers read the primary key constraint and fall back to id.
func TestLinkKeyHelpersSQL(t *testing.T) {
	for _, name := range linkKeyHelperNames {
		fn, ok := definedFunction(name)
		if assert.True(t, ok, "%s must be in DefinedFunctions", name) {
			assert.Contains(t, fn.FunctionSQL, "LANGUAGE sql STABLE", name)
		}
	}
	columns, _ := definedFunction("link_key_columns")
	assert.Contains(t, columns.FunctionSQL, "contype = 'p'")
	assert.Contains(t, columns.FunctionSQL, "ARRAY['id']")
}

// TestKeyHelpersDefinedBeforeUse checks SQL-language helpers are created after the helpers they call,
// since Postgres validates their bodies at creation time.
func TestKeyHelpersDefinedBeforeUse(t *testing.T) {
	position := map[string]int{}
	for i, fn := range constant.DefinedFunctions {
		position[fn.FunctionName] = i
	}
	callPattern := regexp.MustCompile(`public\.([a-z_]+)\(`)
	for _, name := range linkKeyHelperNames {
		fn, _ := definedFunction(name)
		for _, match := range callPattern.FindAllStringSubmatch(fn.FunctionSQL, -1) {
			assert.Less(t, position[match[1]], position[name], "%s must be defined after %s", name, match[1])
		}
	}
}

// TestLinkKeySyncTrigger checks the trigger rewrites keys on UPDATE and drops them on DELETE for every
// link field whose relation points at the changed table.
func TestLinkKeySyncTrigger(t *testing.T) {
	sync, _ := definedFunction("link_key_sync")
	assert.Contains(t, sync.FunctionSQL, "RETURNS TRIGGER")
	for _, part := range []string{"link_old_rows", "link_new_rows", "c.meta->''relation''->>''with''", "IS DISTINCT FROM", "ch ? ''new''"} {
		assert.Contains(t, sync.FunctionSQL, part)
	}
	install, _ := definedFunction("link_install_key_sync")
	for _, part := range []string{"AFTER UPDATE", "AFTER DELETE", "REFERENCING OLD TABLE AS link_old_rows", "FOR EACH STATEMENT"} {
		assert.Contains(t, install.FunctionSQL, part)
	}
}

// TestDroppedFunctions checks superseded signatures are dropped and no longer defined.
func TestDroppedFunctions(t *testing.T) {
	assert.Contains(t, constant.DroppedFunctions, "remove_link_back_references(JSON[], BIGINT[], TEXT)")
	// Placeholder key values are no longer generated; CASCADE also removes the triggers that used it
	assert.Contains(t, constant.DroppedFunctions, "key_fill_defaults() CASCADE")
	_, defined := definedFunction("key_fill_defaults")
	assert.False(t, defined)
	_, ok := definedFunction("remove_link_back_references")
	assert.False(t, ok)
}
