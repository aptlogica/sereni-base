package constant_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/stretchr/testify/assert"
)

// TestLookupDataFunctionDefined checks the read function for lookups is registered for startup creation
// and takes per-target output aliases and the is_array flag.
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
	for _, part := range []string{"RETURNS JSON", "'targets'", "'alias'", "'is_array'", "ANY(s.%I)", "ORDER BY s.created_time"} {
		assert.True(t, strings.Contains(fn.FunctionSQL, part), "function SQL should contain %q", part)
	}
}

var linkFunctionNames = []string{
	"link_lock_row", "link_apply_change", "link_rows_holding",
	"link_cycle_parent", "link_cycle_children", "link_rows", "remove_link_back_references",
}

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

// TestLinkHelperFunctionsSQL checks the row-level helpers lock rows and update arrays in place.
func TestLinkHelperFunctionsSQL(t *testing.T) {
	lock, _ := definedFunction("link_lock_row")
	assert.Contains(t, lock.FunctionSQL, "FOR UPDATE")

	holding, _ := definedFunction("link_rows_holding")
	assert.Contains(t, holding.FunctionSQL, "FOR UPDATE")
	assert.Contains(t, holding.FunctionSQL, "ANY(%I)")

	apply, _ := definedFunction("link_apply_change")
	for _, part := range []string{"array_append", "array_remove", "last_modified_time", "last_modified_by", "%I = NULL"} {
		assert.Contains(t, apply.FunctionSQL, part)
	}

	for _, name := range []string{"link_cycle_parent", "link_cycle_children"} {
		cycle, _ := definedFunction(name)
		assert.Contains(t, cycle.FunctionSQL, "WITH RECURSIVE")
		assert.Contains(t, cycle.FunctionSQL, "IF p_parent = p_child THEN")
	}

	cleanup, _ := definedFunction("remove_link_back_references")
	assert.Contains(t, cleanup.FunctionSQL, "public.link_rows_holding")
	assert.Contains(t, cleanup.FunctionSQL, "COALESCE(p_partners")
	assert.Contains(t, cleanup.FunctionSQL, "COALESCE(p_row_ids")
}
