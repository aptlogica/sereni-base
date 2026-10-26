package table_test

import (
	"testing"
	"time"

	services "github.com/aptlogica/sereni-base/internal/services/table"
	"github.com/stretchr/testify/assert"
)

func TestSimilarityScore(t *testing.T) {
	assert.InDelta(t, 1.0, services.SimilarityScore("", ""), 1e-9)
	assert.InDelta(t, 1.0, services.SimilarityScore("hello", "hello"), 1e-9)
	assert.InDelta(t, 0.0, services.SimilarityScore("abc", ""), 1e-9)
	assert.InDelta(t, 0.0, services.SimilarityScore("", "abc"), 1e-9)
	assert.InDelta(t, 0.0, services.SimilarityScore("abc", "xyz"), 1e-9)

	// an anagram scores higher than a different word of the same length
	assert.Greater(t, services.SimilarityScore("listen", "silent"), services.SimilarityScore("listen", "monkey"))
	// close spellings stay above the medium threshold
	assert.Greater(t, services.SimilarityScore("john smith", "jon smith"), 0.8)
}

func TestSimilarityScoreWeighted(t *testing.T) {
	// weight 0 is purely positional, weight 1 purely order-independent
	assert.Less(t, services.SimilarityScoreWeighted("abc", "cba", 0), 0.5)
	assert.InDelta(t, 1.0, services.SimilarityScoreWeighted("abc", "cba", 1), 1e-9)

	// out-of-range weights are clamped
	assert.InDelta(t, services.SimilarityScoreWeighted("abc", "cba", 0), services.SimilarityScoreWeighted("abc", "cba", -3), 1e-9)
	assert.InDelta(t, services.SimilarityScoreWeighted("abc", "cba", 1), services.SimilarityScoreWeighted("abc", "cba", 7), 1e-9)

	// grapheme clusters count as one character
	assert.InDelta(t, 1.0, services.SimilarityScore("👍🏽a", "👍🏽a"), 1e-9)
	assert.Greater(t, services.SimilarityScore("éa", "éa"), 0.0)
}

func TestFindFuzzyDuplicates_EmptyInput(t *testing.T) {
	assert.Nil(t, services.FindFuzzyDuplicates(nil, []string{"name"}, "keep_first", 0.8))
	assert.Nil(t, services.FindFuzzyDuplicates([]map[string]interface{}{}, []string{"name"}, "keep_first", 0.8))
}

func TestFindFuzzyDuplicates_NoMatches(t *testing.T) {
	rows := []map[string]interface{}{
		{"id": "a", "name": "apple"},
		{"id": "b", "name": "zucchini"},
		{"id": "c", "name": "kiwi"},
	}
	assert.Empty(t, services.FindFuzzyDuplicates(rows, []string{"name"}, "keep_first", 0.8))
}

func TestFindFuzzyDuplicates_KeepRules(t *testing.T) {
	rows := []map[string]interface{}{
		{"id": 1, "name": "John Smith", "last_modified_time": "2024-01-01"},
		{"id": 2, "name": "Jon Smith", "last_modified_time": "2024-03-01"},
		{"id": 3, "name": "john smith", "last_modified_time": "2024-02-01"},
		{"id": 4, "name": "Completely Different", "last_modified_time": "2024-01-01"},
	}
	cols := []string{"name"}

	t.Run("keep_first removes later ids", func(t *testing.T) {
		got := services.FindFuzzyDuplicates(rows, cols, "keep_first", 0.8)
		assert.ElementsMatch(t, []interface{}{2, 3}, got)
	})

	t.Run("keep_last removes earlier ids", func(t *testing.T) {
		got := services.FindFuzzyDuplicates(rows, cols, "keep_last", 0.8)
		assert.ElementsMatch(t, []interface{}{1, 2}, got)
	})

	t.Run("keep_latest_updated keeps newest modification", func(t *testing.T) {
		got := services.FindFuzzyDuplicates(rows, cols, "keep_latest_updated", 0.8)
		assert.ElementsMatch(t, []interface{}{1, 3}, got)
	})

	t.Run("keep_latest_updated ties fall back to id order", func(t *testing.T) {
		tied := []map[string]interface{}{
			{"id": 5, "name": "same", "last_modified_time": "2024-01-01"},
			{"id": 4, "name": "same", "last_modified_time": "2024-01-01"},
		}
		got := services.FindFuzzyDuplicates(tied, cols, "keep_latest_updated", 0.8)
		assert.Equal(t, []interface{}{5}, got)
	})
}

func TestFindFuzzyDuplicates_IDTypes(t *testing.T) {
	cols := []string{"name"}
	mk := func(a, b interface{}) []map[string]interface{} {
		return []map[string]interface{}{{"id": a, "name": "dup"}, {"id": b, "name": "dup"}}
	}

	assert.Equal(t, []interface{}{int64(2)}, services.FindFuzzyDuplicates(mk(int64(1), int64(2)), cols, "keep_first", 0.8))
	assert.Equal(t, []interface{}{float64(1)}, services.FindFuzzyDuplicates(mk(float64(1), float64(2)), cols, "keep_last", 0.8))
	assert.Equal(t, []interface{}{float32(2)}, services.FindFuzzyDuplicates(mk(float32(1), float32(2)), cols, "keep_first", 0.8))
	// numeric strings compare numerically ("10" > "9")
	assert.Equal(t, []interface{}{"10"}, services.FindFuzzyDuplicates(mk("9", "10"), cols, "keep_first", 0.8))
	// non-numeric ids compare as strings
	assert.Equal(t, []interface{}{"b"}, services.FindFuzzyDuplicates(mk("a", "b"), cols, "keep_first", 0.8))
	assert.Equal(t, []interface{}{"a"}, services.FindFuzzyDuplicates(mk("a", "b"), cols, "keep_last", 0.8))
	// unsupported / nil id types fall back to string comparison
	assert.Len(t, services.FindFuzzyDuplicates(mk(true, false), cols, "keep_first", 0.8), 1)
	assert.Len(t, services.FindFuzzyDuplicates(mk(nil, 3), cols, "keep_first", 0.8), 1)
	// identical ids fall back to original row order
	same := services.FindFuzzyDuplicates(mk(7, 7), cols, "keep_first", 0.8)
	assert.Equal(t, []interface{}{7}, same)
	sameLast := services.FindFuzzyDuplicates(mk(7, 7), cols, "keep_last", 0.8)
	assert.Equal(t, []interface{}{7}, sameLast)
}

func TestFindFuzzyDuplicates_EmptyAndNilValues(t *testing.T) {
	cols := []string{"name"}
	rows := []map[string]interface{}{
		{"id": 1, "name": nil},
		{"id": 2, "name": ""},
		{"id": 3, "name": "   "},
		{"id": 4},
		{"id": 5, "name": "real"},
		{"id": 6, "name": "real"},
	}
	// blank rows never match each other or anything else
	assert.Equal(t, []interface{}{6}, services.FindFuzzyDuplicates(rows, cols, "keep_first", 0.8))
}

func TestFindFuzzyDuplicates_MultipleColumns(t *testing.T) {
	cols := []string{"first", "last"}
	rows := []map[string]interface{}{
		{"id": 1, "first": "Alice", "last": "Johnson"},
		{"id": 2, "first": "Alice", "last": "Johnsen"},
		{"id": 3, "first": "Alice", "last": "Zimmerman"}, // same first, different last
		{"id": 4, "first": "Alice", "last": nil},         // one side empty never matches a filled value
		{"id": 5, "first": "Bob", "last": nil},
		{"id": 6, "first": "Bob", "last": ""}, // both empty counts as equal
	}
	got := services.FindFuzzyDuplicates(rows, cols, "keep_first", 0.8)
	assert.ElementsMatch(t, []interface{}{2, 6}, got)
}

func TestFindFuzzyDuplicates_TransitiveGrouping(t *testing.T) {
	// a~b and b~c but a and c alone are below the threshold; union-find still groups all three
	rows := []map[string]interface{}{
		{"id": 1, "name": "abcdefghij"},
		{"id": 2, "name": "abcdefghxx"},
		{"id": 3, "name": "abcdefxxxx"},
	}
	got := services.FindFuzzyDuplicates(rows, []string{"name"}, "keep_first", 0.8)
	assert.ElementsMatch(t, []interface{}{2, 3}, got)
}

func TestFindFuzzyDuplicates_LengthPruning(t *testing.T) {
	rows := []map[string]interface{}{
		{"id": 1, "name": "a"},
		{"id": 2, "name": "aaaaaaaaaaaaaaaaaaaa"},
	}
	assert.Empty(t, services.FindFuzzyDuplicates(rows, []string{"name"}, "keep_first", 0.9))
	// a threshold at or below the order-independent floor never prunes by length
	assert.Empty(t, services.FindFuzzyDuplicates(rows, []string{"name"}, "keep_first", 0.1))
	assert.NotNil(t, services.FindFuzzyDuplicates(rows, []string{"name"}, "keep_first", 0.0))
}

func TestFindFuzzyDuplicates_ValueTypes(t *testing.T) {
	ts := time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)
	rows := []map[string]interface{}{
		{"id": 1, "v": 1500000.0},
		{"id": 2, "v": "1500000"},
		{"id": 3, "v": float32(2.5)},
		{"id": 4, "v": "2.5"},
		{"id": 5, "v": ts},
		{"id": 6, "v": ts.Format(time.RFC3339Nano)},
		{"id": 7, "v": 42},
		{"id": 8, "v": "42"},
	}
	got := services.FindFuzzyDuplicates(rows, []string{"v"}, "keep_first", 0.9)
	assert.ElementsMatch(t, []interface{}{2, 4, 6, 8}, got)
}

func TestFindFuzzyDuplicates_ManyRowsParallel(t *testing.T) {
	rows := make([]map[string]interface{}, 0, 200)
	for i := 0; i < 100; i++ {
		rows = append(rows, map[string]interface{}{"id": i * 2, "name": "duplicated value"})
	}
	for i := 0; i < 100; i++ {
		rows = append(rows, map[string]interface{}{"id": i*2 + 1, "name": "duplicated value"})
	}
	got := services.FindFuzzyDuplicates(rows, []string{"name"}, "keep_first", 0.8)
	assert.Len(t, got, 199)
}
