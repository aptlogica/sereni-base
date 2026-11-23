// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package csv

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aptlogica/sereni-base/internal/dto"
)

// cellText turns the value of one cell into text. It is pure: the value, the field and the joiner in, a string out.
//
//	text, numbers, dates   the value as it is (numbers with "." and no thousands separator, dates as ISO 8601)
//	boolean                true or false
//	multi select, lists    the values joined with the joiner
//	attachment             the paths in the zip of the files that were packed, joined with the joiner
//	JSON and the rest      compact JSON or the text of the value
//
// A null or missing value is an empty cell.
func cellText(f dto.BaseExportField, v interface{}, joiner string, paths map[string]string) string {
	if v == nil {
		return ""
	}
	if f.UIDT == "attachment" {
		var out []string
		for _, ref := range toStrings(v) {
			if p, ok := paths[ref]; ok {
				out = append(out, p)
			}
		}
		return strings.Join(out, joiner)
	}
	if f.UIDT == "json" {
		if raw, err := json.Marshal(v); err == nil {
			return string(raw)
		}
	}
	return valueText(v, joiner)
}

func valueText(v interface{}, joiner string) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		return strconv.FormatBool(x)
	case int:
		return strconv.Itoa(x)
	case int32:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case float32:
		return strconv.FormatFloat(float64(x), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case json.Number:
		return x.String()
	case time.Time:
		return x.UTC().Format(time.RFC3339)
	case []string:
		return strings.Join(x, joiner)
	case []interface{}:
		parts := make([]string, 0, len(x))
		for _, item := range x {
			if text := valueText(item, joiner); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, joiner)
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

// toStrings reads the refs of an attachment cell
func toStrings(v interface{}) []string {
	switch x := v.(type) {
	case []string:
		return x
	case []interface{}:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case string:
		return []string{x}
	}
	return nil
}
