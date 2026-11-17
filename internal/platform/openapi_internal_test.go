package platform

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The generic spec parses and has the paths the spec builder uses as templates
func TestOpenAPIBaseParses(t *testing.T) {
	doc, err := openAPIBase()
	require.NoError(t, err)
	paths := child(doc, "paths")
	for _, suffix := range recordsPathSuffixes {
		assert.NotNil(t, child(paths, genericTablePath+suffix), suffix)
	}
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
			_, err := buildFilter([]*field{f}, "F", op, value, testLinks)
			assert.Equal(t, err == nil, slices.Contains(ops, op), "type %s, operator %s: %v", uidt, op, err)
		}
	}
}
