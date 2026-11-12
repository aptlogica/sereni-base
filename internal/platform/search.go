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

// search=text finds records where any text field contains the text, ignoring case. It is the text contains
// filter on every searchable field, joined with any; there is no index yet (Performance layer: pg_trgm).
const maxSearchLength = 200

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
	var nodes []models.ComplexFilter
	for _, f := range fields {
		if !searchTypes[f.Type] {
			continue
		}
		t, err := buildCondition(fields, filterCondition{label: "search", ref: f.PublicID(), op: "contains", value: text}, nil)
		if err != nil {
			continue // a column name the filter builder doesn't take; the field can't be searched
		}
		nodes = append(nodes, t.node)
	}
	if len(nodes) == 0 {
		return filterTerm{}, false, badRequest("search: this table has no text fields to search.")
	}
	return filterTerm{key: "search:" + text, node: anyOf(nodes...)}, true, nil
}
