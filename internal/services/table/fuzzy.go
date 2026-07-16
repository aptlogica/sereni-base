// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0

package services

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/rivo/uniseg"
)

// graphemes splits s into user-perceived characters (grapheme clusters).
func graphemes(s string) []string {
	if s == "" {
		return nil
	}
	out := make([]string, 0, len(s))
	g := uniseg.NewGraphemes(s)
	for g.Next() {
		out = append(out, g.Str())
	}
	return out
}

// levenshtein returns the edit distance between two grapheme slices.
func levenshtein(a, b []string) int {
	m, n := len(a), len(b)
	if m == 0 {
		return n
	}
	if n == 0 {
		return m
	}

	prev := make([]int, n+1)
	curr := make([]int, n+1)
	for j := 0; j <= n; j++ {
		prev[j] = j
	}

	for i := 1; i <= m; i++ {
		curr[0] = i
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				curr[j] = prev[j-1]
			} else {
				curr[j] = 1 + min3(prev[j], curr[j-1], prev[j-1])
			}
		}
		prev, curr = curr, prev
	}
	return prev[n]
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}

// positionalSimilarity is the order-sensitive Levenshtein component, mapped to
// 0..1 over grapheme length.
func positionalSimilarity(a, b []string) float64 {
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	if maxLen == 0 {
		return 1.0
	}
	return 1.0 - float64(levenshtein(a, b))/float64(maxLen)
}

// multisetSimilarity is the order-INDEPENDENT component: a Sørensen–Dice
// coefficient over grapheme counts.
func multisetSimilarity(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0
	}
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	countA := make(map[string]int, len(a))
	for _, g := range a {
		countA[g]++
	}
	countB := make(map[string]int, len(b))
	for _, g := range b {
		countB[g]++
	}

	intersection := 0
	for g, ca := range countA {
		if cb, ok := countB[g]; ok {
			intersection += min2(ca, cb)
		}
	}

	return 2.0 * float64(intersection) / float64(len(a)+len(b))
}

func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// DefaultReorderWeight controls how much the order-independent (anagram-aware)
// component contributes to the blended score.
const DefaultReorderWeight = 0.25

// SimilarityScore returns a 0..1 similarity using the default blend.
func SimilarityScore(a, b string) float64 {
	return SimilarityScoreWeighted(a, b, DefaultReorderWeight)
}

// SimilarityScoreWeighted blends the positional and multiset components.
func SimilarityScoreWeighted(a, b string, reorderWeight float64) float64 {
	if reorderWeight < 0 {
		reorderWeight = 0
	} else if reorderWeight > 1 {
		reorderWeight = 1
	}

	ga, gb := graphemes(a), graphemes(b)

	pos := positionalSimilarity(ga, gb)
	set := multisetSimilarity(ga, gb)

	return (1-reorderWeight)*pos + reorderWeight*set
}

// // Match pairs an item with its similarity score against the query.
// type Match[T any] struct {
// 	Item  T
// 	Score float64
// }

// // KeyFunc extracts the searchable text from an item.
// type KeyFunc[T any] func(T) string

// // Search returns items whose blended similarity to query meets or exceeds
// // threshold, sorted by score descending. Matching is case-insensitive.
// func Search[T any](query string, items []T, key KeyFunc[T], threshold float64) []Match[T] {
// 	return SearchWeighted(query, items, key, threshold, DefaultReorderWeight)
// }

// // SearchWeighted is Search with an explicit reorder weight.
// func SearchWeighted[T any](query string, items []T, key KeyFunc[T], threshold, reorderWeight float64) []Match[T] {
// 	if query == "" {
// 		return nil
// 	}
// 	query = strings.ToLower(query)

// 	results := make([]Match[T], 0, len(items))
// 	for _, item := range items {
// 		text := strings.ToLower(key(item))
// 		score := SimilarityScoreWeighted(query, text, reorderWeight)
// 		if score >= threshold {
// 			results = append(results, Match[T]{Item: item, Score: score})
// 		}
// 	}

// 	sort.SliceStable(results, func(i, j int) bool {
// 		return results[i].Score > results[j].Score
// 	})
// 	return results
// }

// // SearchStrings is a convenience wrapper for a plain []string.
// func SearchStrings(query string, items []string, threshold float64) []Match[string] {
// 	return Search(query, items, func(s string) string { return s }, threshold)
// }

// FindFuzzyDuplicates clusters allRows into groups of fuzzy duplicates based on threshold
// and selectedColumns, sorts each group using the keepRule, and returns the slice of
// duplicate row IDs to be updated or removed.
func FindFuzzyDuplicates(
	allRows []map[string]interface{},
	selectedColumns []string,
	keepRule string,
	threshold float64,
) []interface{} {
	if len(allRows) == 0 {
		return nil
	}

	// 1. Group records into connected components of fuzzy similarity.
	parent := make([]int, len(allRows))
	for i := range parent {
		parent[i] = i
	}

	var find func(int) int
	find = func(i int) int {
		root := i
		for root != parent[root] {
			root = parent[root]
		}
		// Path compression
		curr := i
		for curr != root {
			next := parent[curr]
			parent[curr] = root
			curr = next
		}
		return root
	}

	union := func(i, j int) {
		rootI := find(i)
		rootJ := find(j)
		if rootI != rootJ {
			parent[rootI] = rootJ
		}
	}

	// Perform fuzzy comparisons between all pairs of non-empty rows
	for i := 0; i < len(allRows); i++ {
		emptyI := true
		for _, col := range selectedColumns {
			val := allRows[i][col]
			if val != nil && strings.TrimSpace(fmt.Sprintf("%v", val)) != "" {
				emptyI = false
				break
			}
		}
		if emptyI {
			continue
		}

		for j := i + 1; j < len(allRows); j++ {
			emptyJ := true
			for _, col := range selectedColumns {
				val := allRows[j][col]
				if val != nil && strings.TrimSpace(fmt.Sprintf("%v", val)) != "" {
					emptyJ = false
					break
				}
			}
			if emptyJ {
				continue
			}

			match := true
			for _, col := range selectedColumns {
				valI := strings.ToLower(strings.TrimSpace(getStringVal(allRows[i][col])))
				valJ := strings.ToLower(strings.TrimSpace(getStringVal(allRows[j][col])))

				if valI == "" && valJ == "" {
					continue
				}
				if valI == "" || valJ == "" {
					match = false
					break
				}

				score := SimilarityScore(valI, valJ)
				if score < threshold {
					match = false
					break
				}
			}
			if match {
				union(i, j)
			}
		}
	}

	// 2. Collect entries into their respective groups.
	groupMap := make(map[int][]int)
	for i := range allRows {
		emptyRow := true
		for _, col := range selectedColumns {
			val := allRows[i][col]
			if val != nil && strings.TrimSpace(fmt.Sprintf("%v", val)) != "" {
				emptyRow = false
				break
			}
		}

		root := i
		if !emptyRow {
			root = find(i)
		}
		groupMap[root] = append(groupMap[root], i)
	}

	// 3. For each group, determine which one to keep and which ones are duplicates.
	var duplicateRowIDs []interface{}

	for _, indices := range groupMap {
		if len(indices) <= 1 {
			continue
		}

		// Sort the group members according to the keepRule to decide the keeper
		sort.SliceStable(indices, func(x, y int) bool {
			rowA, rowB := allRows[indices[x]], allRows[indices[y]]

			if keepRule == "keep_latest_updated" {
				timeAStr := fmt.Sprintf("%v", rowA["last_modified_time"])
				timeBStr := fmt.Sprintf("%v", rowB["last_modified_time"])
				if timeAStr != timeBStr {
					return timeAStr > timeBStr // Descending: newer first
				}
			}

			idA := rowA["id"]
			idB := rowB["id"]

			numA, errA := getFloat64(idA)
			numB, errB := getFloat64(idB)
			if errA == nil && errB == nil {
				if numA != numB {
					if keepRule == "keep_last" {
						return numA > numB // Descending: larger first
					}
					return numA < numB // Ascending: smaller first
				}
			}

			strA := fmt.Sprintf("%v", idA)
			strB := fmt.Sprintf("%v", idB)
			if strA != strB {
				if keepRule == "keep_last" {
					return strA > strB
				}
				return strA < strB
			}

			// Fallback to original record indices
			if keepRule == "keep_last" {
				return indices[y] < indices[x] // Descending index
			}
			return indices[x] < indices[y] // Ascending index
		})

		// All other members in sorted group are duplicates
		for idx := 1; idx < len(indices); idx++ {
			duplicateRowIDs = append(duplicateRowIDs, allRows[indices[idx]]["id"])
		}
	}

	return duplicateRowIDs
}
func getFloat64(val interface{}) (float64, error) {
	if val == nil {
		return 0, fmt.Errorf("nil value")
	}
	switch v := val.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float32:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("unsupported type: %T", val)
	}
}

func getStringVal(val interface{}) string {
	if val == nil {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", val)
}
