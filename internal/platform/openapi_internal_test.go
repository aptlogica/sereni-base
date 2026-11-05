package platform

import (
	"bytes"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The embedded base spec is a copy of the docs file; the two must not drift apart
func TestOpenAPIBaseMatchesDocs(t *testing.T) {
	docs, err := os.ReadFile("../../docs/api-platform/openapi.yaml")
	require.NoError(t, err)
	normalize := func(b []byte) []byte { return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n")) }
	assert.Equal(t, string(normalize(docs)), string(normalize(openAPIBaseYAML)),
		"copy docs/api-platform/openapi.yaml to internal/platform/openapi_base.yaml")
}

// The operators the spec lists for each field type are exactly the ones buildFilter accepts
func TestFilterOperatorsMatchBuildFilter(t *testing.T) {
	allOps := []string{"eq", "neq", "gt", "gte", "lt", "lte", "in", "contains", "is_null"}
	sample := func(f *field) string {
		switch {
		case numberTypes[f.Type]:
			return "5"
		case f.Type == typeDate:
			return "2026-11-03"
		case timeTypes[f.Type]:
			return "2026-11-03T10:15:00Z"
		case f.Type == typeCheckbox:
			return "true"
		case f.Type == typeLinks:
			return "rec_1"
		}
		return "x"
	}
	linkTarget := func(*field) string { return `"public"."target"` }

	types := []string{"somethingnew"}
	for uidt := range apiTypeNames {
		types = append(types, uidt)
	}
	for _, uidt := range types {
		f := &field{ID: "aaaaaaaa-0000-0000-0000-000000000001", Name: "F", ColumnName: "f", Type: uidt}
		ops := filterOperators(f)
		for _, op := range allOps {
			value := sample(f)
			if op == "is_null" {
				value = "true"
			}
			_, err := buildFilter([]*field{f}, "F", op, value, linkTarget)
			assert.Equal(t, err == nil, slices.Contains(ops, op), "type %s, operator %s: %v", uidt, op, err)
		}
	}
}
