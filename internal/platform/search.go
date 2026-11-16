// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/aptlogica/go-postgres-rest/pkg/models"
)

// search=text finds records where every word of the text is in some text field, ignoring case: "acme corp"
// finds "Acme Corporation" and "Corp Acme", and a record with "acme" in one field and "corp" in another. Each
// word is the text contains filter on every searchable field, joined with any; the words are joined with all.
// Words are split on spaces only, so "C++" and "e-mail" stay whole. There is no index yet (Performance layer:
// pg_trgm).
const (
	maxSearchLength = 200
	maxSearchWords  = 10 // each word is one condition per text field
)

// searchTypes are the field types search looks in: text people type or pick
var searchTypes = map[string]bool{
	typeText: true, typeLongText: true, typeEmail: true, typeURL: true, typePhone: true, typeSelect: true,
}

// buildSearch builds the search filter. ok is false when there is nothing to search for.
func buildSearch(fields []*field, raw string) (term filterTerm, ok bool, err error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return filterTerm{}, false, nil
	}
	if utf8.RuneCountInString(text) > maxSearchLength {
		return filterTerm{}, false, badRequest(fmt.Sprintf("search: at most %d characters.", maxSearchLength))
	}
	words := searchWords(text)
	if len(words) > maxSearchWords {
		return filterTerm{}, false, badRequest(fmt.Sprintf("search: at most %d words.", maxSearchWords))
	}
	perWord := make([]models.ComplexFilter, 0, len(words))
	for _, word := range words {
		var nodes []models.ComplexFilter
		for _, f := range fields {
			if !searchTypes[f.Type] {
				continue
			}
			t, err := buildCondition(fields, filterCondition{label: "search", ref: f.PublicID(), op: "contains", value: word}, nil)
			if err != nil {
				continue // a column name the filter builder doesn't take; the field can't be searched
			}
			nodes = append(nodes, t.node)
		}
		if len(nodes) == 0 {
			return filterTerm{}, false, badRequest("search: this table has no text fields to search.")
		}
		perWord = append(perWord, anyOf(nodes...))
	}
	// The key stays the text as sent, so a search's cursors keep working across this change
	key := "search:" + text
	if len(perWord) == 1 {
		return filterTerm{key: key, node: perWord[0]}, true, nil
	}
	return filterTerm{key: key, node: models.ComplexFilter{Logic: "AND", Groups: perWord}}, true, nil
}

// searchWords splits the search text on spaces, each word once (ignoring case), in the order given
func searchWords(text string) []string {
	var words []string
	seen := map[string]bool{}
	for _, w := range strings.Fields(text) {
		if k := strings.ToLower(w); !seen[k] {
			seen[k] = true
			words = append(words, w)
		}
	}
	return words
}
