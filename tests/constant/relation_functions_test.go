package constant_test

import (
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
