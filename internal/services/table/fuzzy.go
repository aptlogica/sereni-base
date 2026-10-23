// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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

// fuzzyMatchPair records two row indices whose selected columns fuzzy-match.
type fuzzyMatchPair struct {
	i, j int
}

// fuzzyUnionFind groups row indices into connected components of fuzzy similarity.
type fuzzyUnionFind struct {
	parent []int
}

func newFuzzyUnionFind(n int) *fuzzyUnionFind {
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}
	return &fuzzyUnionFind{parent: parent}
}

func (u *fuzzyUnionFind) find(i int) int {
	root := i
	for root != u.parent[root] {
		root = u.parent[root]
	}
	// Path compression
	curr := i
	for curr != root {
		next := u.parent[curr]
		u.parent[curr] = root
		curr = next
	}
	return root
}

func (u *fuzzyUnionFind) union(i, j int) {
	rootI := u.find(i)
	rootJ := u.find(j)
	if rootI != rootJ {
		u.parent[rootI] = rootJ
	}
}

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
	uf := newFuzzyUnionFind(len(allRows))
	graphemesCache := buildFuzzyGraphemesCache(allRows, selectedColumns)
	matches := collectFuzzyMatches(allRows, graphemesCache, selectedColumns, threshold)

	// Apply unions sequentially on the collected matches
	for _, pair := range matches {
		uf.union(pair.i, pair.j)
	}

	// 2. Collect entries into their respective groups.
	groupMap := groupFuzzyRows(allRows, selectedColumns, uf)

	// 3. For each group, determine which one to keep and which ones are duplicates.
	return collectFuzzyDuplicateIDs(allRows, groupMap, keepRule)
}

// buildFuzzyGraphemesCache pre-calculates graphemes for selected columns of all rows.
func buildFuzzyGraphemesCache(allRows []map[string]interface{}, selectedColumns []string) []map[string][]string {
	graphemesCache := make([]map[string][]string, len(allRows))
	for i := range allRows {
		colMap := make(map[string][]string)
		for _, col := range selectedColumns {
			valStr := strings.ToLower(strings.TrimSpace(getStringVal(allRows[i][col])))
			colMap[col] = graphemes(valStr)
		}
		graphemesCache[i] = colMap
	}
	return graphemesCache
}

// isFuzzyRowEmpty reports whether every selected column of row is nil or blank.
func isFuzzyRowEmpty(row map[string]interface{}, selectedColumns []string) bool {
	for _, col := range selectedColumns {
		val := row[col]
		if val != nil && strings.TrimSpace(fmt.Sprintf("%v", val)) != "" {
			return false
		}
	}
	return true
}

// collectFuzzyMatches compares all row pairs in parallel and returns the matching pairs.
func collectFuzzyMatches(
	allRows []map[string]interface{},
	graphemesCache []map[string][]string,
	selectedColumns []string,
	threshold float64,
) []fuzzyMatchPair {
	var matches []fuzzyMatchPair
	var mu sync.Mutex

	numWorkers := runtime.NumCPU()
	if numWorkers < 1 {
		numWorkers = 1
	}

	var wg sync.WaitGroup
	var nextI int64

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			localMatches := fuzzyMatchWorker(allRows, graphemesCache, selectedColumns, threshold, &nextI)
			if len(localMatches) > 0 {
				mu.Lock()
				matches = append(matches, localMatches...)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	return matches
}

// fuzzyMatchWorker claims row indices from nextI until exhausted and returns its matches.
func fuzzyMatchWorker(
	allRows []map[string]interface{},
	graphemesCache []map[string][]string,
	selectedColumns []string,
	threshold float64,
	nextI *int64,
) []fuzzyMatchPair {
	var localMatches []fuzzyMatchPair
	n := len(allRows)

	for {
		i := int(atomic.AddInt64(nextI, 1) - 1)
		if i >= n {
			break
		}
		localMatches = appendFuzzyMatchesForRow(localMatches, allRows, graphemesCache, selectedColumns, threshold, i)
	}
	return localMatches
}

// appendFuzzyMatchesForRow appends every pair (i, j) with j > i whose rows fuzzy-match.
func appendFuzzyMatchesForRow(
	localMatches []fuzzyMatchPair,
	allRows []map[string]interface{},
	graphemesCache []map[string][]string,
	selectedColumns []string,
	threshold float64,
	i int,
) []fuzzyMatchPair {
	if isFuzzyRowEmpty(allRows[i], selectedColumns) {
		return localMatches
	}

	for j := i + 1; j < len(allRows); j++ {
		if isFuzzyRowEmpty(allRows[j], selectedColumns) {
			continue
		}
		if fuzzyRowsMatch(graphemesCache[i], graphemesCache[j], selectedColumns, threshold) {
			localMatches = append(localMatches, fuzzyMatchPair{i: i, j: j})
		}
	}
	return localMatches
}

// fuzzyRowsMatch reports whether every selected column of two rows meets the threshold.
func fuzzyRowsMatch(rowI, rowJ map[string][]string, selectedColumns []string, threshold float64) bool {
	for _, col := range selectedColumns {
		if !fuzzyColumnMatches(rowI[col], rowJ[col], threshold) {
			return false
		}
	}
	return true
}

// fuzzyColumnMatches compares one column's graphemes; two empty values count as a match.
func fuzzyColumnMatches(gI, gJ []string, threshold float64) bool {
	if len(gI) == 0 && len(gJ) == 0 {
		return true
	}
	if len(gI) == 0 || len(gJ) == 0 {
		return false
	}
	if exceedsFuzzyLengthBound(gI, gJ, threshold) {
		return false
	}

	pos := positionalSimilarity(gI, gJ)
	set := multisetSimilarity(gI, gJ)
	score := (1.0-DefaultReorderWeight)*pos + DefaultReorderWeight*set

	return score >= threshold
}

// exceedsFuzzyLengthBound is the length pruning check: it reports whether the length
// difference alone already makes reaching the threshold impossible.
func exceedsFuzzyLengthBound(gI, gJ []string, threshold float64) bool {
	minPos := (threshold - 0.25) / 0.75
	if minPos < 0 {
		minPos = 0
	}
	maxLen := len(gI)
	if len(gJ) > maxLen {
		maxLen = len(gJ)
	}
	maxEditDist := (1.0 - minPos) * float64(maxLen)
	lenDiff := len(gI) - len(gJ)
	if lenDiff < 0 {
		lenDiff = -lenDiff
	}
	return float64(lenDiff) > maxEditDist
}

// groupFuzzyRows maps each component root to its row indices; empty rows stay alone.
func groupFuzzyRows(allRows []map[string]interface{}, selectedColumns []string, uf *fuzzyUnionFind) map[int][]int {
	groupMap := make(map[int][]int)
	for i := range allRows {
		root := i
		if !isFuzzyRowEmpty(allRows[i], selectedColumns) {
			root = uf.find(i)
		}
		groupMap[root] = append(groupMap[root], i)
	}
	return groupMap
}

// collectFuzzyDuplicateIDs sorts each group by keepRule and returns the IDs of every
// member except the keeper.
func collectFuzzyDuplicateIDs(allRows []map[string]interface{}, groupMap map[int][]int, keepRule string) []interface{} {
	var duplicateRowIDs []interface{}

	for _, indices := range groupMap {
		if len(indices) <= 1 {
			continue
		}

		// Sort the group members according to the keepRule to decide the keeper
		sort.SliceStable(indices, func(x, y int) bool {
			return fuzzyKeepRuleLess(allRows, indices, keepRule, x, y)
		})

		// All other members in sorted group are duplicates
		for idx := 1; idx < len(indices); idx++ {
			duplicateRowIDs = append(duplicateRowIDs, allRows[indices[idx]]["id"])
		}
	}

	return duplicateRowIDs
}

// fuzzyKeepRuleLess orders group members so the record to keep comes first.
func fuzzyKeepRuleLess(allRows []map[string]interface{}, indices []int, keepRule string, x, y int) bool {
	rowA, rowB := allRows[indices[x]], allRows[indices[y]]

	if keepRule == "keep_latest_updated" {
		if less, decided := compareFuzzyLastModified(rowA, rowB); decided {
			return less
		}
	}

	if less, decided := compareFuzzyIDs(rowA["id"], rowB["id"], keepRule); decided {
		return less
	}

	// Fallback to original record indices
	if keepRule == "keep_last" {
		return indices[y] < indices[x] // Descending index
	}
	return indices[x] < indices[y] // Ascending index
}

// compareFuzzyLastModified orders newer records first; decided is false on a tie.
func compareFuzzyLastModified(rowA, rowB map[string]interface{}) (less bool, decided bool) {
	timeAStr := fmt.Sprintf("%v", rowA["last_modified_time"])
	timeBStr := fmt.Sprintf("%v", rowB["last_modified_time"])
	if timeAStr != timeBStr {
		return timeAStr > timeBStr, true // Descending: newer first
	}
	return false, false
}

// compareFuzzyIDs orders by numeric ID when both parse, then by string ID; decided is
// false when the IDs are equal.
func compareFuzzyIDs(idA, idB interface{}, keepRule string) (less bool, decided bool) {
	numA, errA := getFloat64(idA)
	numB, errB := getFloat64(idB)
	if errA == nil && errB == nil && numA != numB {
		if keepRule == "keep_last" {
			return numA > numB, true // Descending: larger first
		}
		return numA < numB, true // Ascending: smaller first
	}

	strA := fmt.Sprintf("%v", idA)
	strB := fmt.Sprintf("%v", idB)
	if strA != strB {
		if keepRule == "keep_last" {
			return strA > strB, true
		}
		return strA < strB, true
	}
	return false, false
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
	// Mirror the API's JSON encoding so scores match the client-side preview.
	switch v := val.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64) // avoid %v's "1.5e+06"
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case time.Time:
		return v.Format(time.RFC3339Nano)
	}
	return fmt.Sprintf("%v", val)
}
