// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package fieldrules

import (
	"fmt"
	"strings"
)

// CheckCell checks a value the app's row endpoints received for a field against the field's settings and
// returns the value to store, or an issue: an option the field doesn't have, an email / phone / URL that fails
// the field's check, or a rating above its highest. Decimal and percent values are kept to the field's
// decimals and 12-hour times become HH:MM, as the app's inputs do. Other types are returned unchanged; their
// column type checks them.
func CheckCell(r Rules, v interface{}) (interface{}, string) {
	if v == nil {
		return nil, ""
	}
	switch r.Type {
	case TypeSelect:
		s, ok := v.(string)
		if !ok {
			return nil, "must be the name of one of the field's options"
		}
		// The app clears a select with ""
		if s != "" && !r.HasOption(s) {
			return nil, fmt.Sprintf("%q is not one of the field's options", s)
		}
		return s, ""
	case TypeMultiSelect:
		var list []interface{}
		switch x := v.(type) {
		case []interface{}:
			list = x
		case []string:
			for _, s := range x {
				list = append(list, s)
			}
		case string:
			if x == "" {
				return []interface{}{}, ""
			}
			list = []interface{}{x}
		default:
			return nil, "must be a list of option names"
		}
		names := make([]interface{}, 0, len(list))
		for _, item := range list {
			s, ok := item.(string)
			if !ok {
				return nil, "must be a list of option names"
			}
			if !r.HasOption(s) {
				return nil, fmt.Sprintf("%q is not one of the field's options", s)
			}
			names = append(names, s)
		}
		return names, ""
	case TypeEmail, TypePhone, TypeURL:
		s, ok := v.(string)
		if !ok {
			return nil, "must be text"
		}
		if issue := FormatIssue(r, s); issue != "" {
			return nil, issue
		}
		return s, ""
	case TypeRating:
		n, ok := number(v)
		if !ok || (n != 0 && n != float64(int64(n))) {
			return nil, "must be a whole number"
		}
		if n < 0 || n > float64(r.RatingMax) {
			return nil, fmt.Sprintf("must be between 0 and %d", r.RatingMax)
		}
		return v, ""
	case TypeDecimal, TypePercent:
		if r.Places < 0 {
			return v, ""
		}
		if s, ok := v.(string); ok && strings.TrimSpace(s) == "" {
			return v, ""
		}
		if n, ok := number(v); ok {
			return Round(n, r.Places), ""
		}
		return v, ""
	case TypeTime:
		s, ok := v.(string)
		if !ok || strings.TrimSpace(s) == "" {
			return v, ""
		}
		t, ok := ParseTime(s)
		if !ok {
			return nil, "must be a time like 14:15 or 2:15 PM"
		}
		return t, ""
	}
	return v, ""
}
