// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// When a table is exported without a table it links to, the link field cannot stay a link. It is exported as a text
// field instead, and so is every lookup and rollup built on it. The cells hold the text of what they showed: for a
// link, the display value of each linked row.

const (
	// textFieldType is the field type a converted link, lookup or rollup becomes
	textFieldType = "longText"
	// uidtRollup is the field type of a rollup
	uidtRollup = "rollup"
	// linkTextSeparator is written between the values of a cell that held several rows or values. It is empty on
	// purpose: the values are joined as they are.
	linkTextSeparator = ""
)

// isLookupLike reports whether a field shows values read through a link.
func isLookupLike(uidt string) bool {
	return uidt == uidtLookup || uidt == uidtRollup
}

// unionOf returns the ids that are in either set.
func unionOf(a, b map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(a)+len(b))
	for id := range a {
		out[id] = struct{}{}
	}
	for id := range b {
		out[id] = struct{}{}
	}
	return out
}

// loadLinkTexts reads, for every link that becomes text, the display text of the rows of the table it points at.
// The result is by link field id, then by the key a link cell holds (the primary key value of the linked row).
func (s *baseTransferService) loadLinkTexts(ctx context.Context, schemaName string, tables []*exportTable, asText map[string]struct{}, warnings *[]string) (map[string]map[string]string, error) {
	out := map[string]map[string]string{}
	read := map[string]map[string]string{} // target table + display column -> texts
	for _, t := range tables {
		for _, c := range t.columns {
			if _, ok := asText[c.ID.String()]; !ok || c.UIDT != uidtLinks {
				continue
			}
			target := idString(nestedString(c.Meta, "relation", "with"))
			display := idString(c.Meta["display_column_id"])
			cacheKey := target + "|" + display
			texts, ok := read[cacheKey]
			if !ok {
				var err error
				if texts, err = s.readDisplayTexts(ctx, schemaName, target, display, warnings); err != nil {
					return nil, err
				}
				read[cacheKey] = texts
			}
			out[c.ID.String()] = texts
		}
	}
	return out, nil
}

// readDisplayTexts returns the display text of every row of a table, by primary key value and by id. The display
// column is the one the link shows; without one the primary key is used.
func (s *baseTransferService) readDisplayTexts(ctx context.Context, schemaName, modelID, displayColumnID string, warnings *[]string) (map[string]string, error) {
	cols, err := s.tableService.GetColumnsByModelID(ctx, schemaName, modelID)
	if err != nil {
		return nil, err
	}
	res, err := s.tableService.GetAllRecords(ctx, schemaName, modelID)
	if err != nil {
		return nil, err
	}
	pk := s.pkColumn(ctx, schemaName, modelID, warnings)
	display := pk
	for _, c := range cols {
		if displayColumnID != "" && c.ID.String() == displayColumnID {
			display = c.ColumnName
		}
	}
	texts := make(map[string]string, len(res.Records))
	for _, rec := range res.Records {
		text := valueText(rec[display])
		if key := cellKey(rec[pk]); key != "" {
			texts[key] = text
		}
		if key := cellKey(rec["id"]); key != "" {
			if _, taken := texts[key]; !taken {
				texts[key] = text
			}
		}
	}
	return texts, nil
}

// linkCellText turns a link cell into text: the display text of each linked row, joined with linkTextSeparator.
func linkCellText(cell interface{}, texts map[string]string) string {
	var parts []string
	for _, item := range toList(cell) {
		if obj, ok := item.(map[string]interface{}); ok {
			item = obj["id"]
		}
		if text := texts[cellKey(item)]; text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, linkTextSeparator)
}

// valueText writes a cell value as text. A list (a lookup of several rows) is the values joined with
// linkTextSeparator.
func valueText(v interface{}) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		return strconv.FormatBool(x)
	case float32, float64, json.Number:
		return cellKey(x)
	case []interface{}:
		parts := make([]string, 0, len(x))
		for _, item := range x {
			if text := valueText(item); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, linkTextSeparator)
	case map[string]interface{}:
		raw, err := json.Marshal(x)
		if err != nil {
			return fmt.Sprint(x)
		}
		return string(raw)
	default:
		return fmt.Sprint(x)
	}
}
