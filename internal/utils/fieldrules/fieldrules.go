// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

// Package fieldrules holds the value rules of a field as its settings (column meta) define them, the same
// rules the app's field inputs apply: select options, the email / phone / URL checks a field can turn on or
// off, decimal places and time input. The public API and the app's row endpoints both use it, so a value
// accepted in one place is accepted in the other.
package fieldrules

import (
	"math"
	"strconv"
	"strings"
)

// Field types (column uidt, lowercased) with rules here
const (
	TypeEmail       = "email"
	TypeURL         = "url"
	TypePhone       = "phonenumber"
	TypeSelect      = "select"
	TypeMultiSelect = "multiselect"
	TypeRating      = "rating"
	TypeDecimal     = "decimal"
	TypePercent     = "percent"
	TypeTime        = "time"
)

// Settings keys of the email, phone and URL checks, and whether each is on when a field has no setting
// (as the app's inputs treat a missing setting)
const (
	EmailCheckKey = "emailValid"
	PhoneCheckKey = "phoneValid"
	URLCheckKey   = "urlValid"
)

// DefaultRatingMax is the highest rating when a field has no ratingMax
const DefaultRatingMax = 5

// PercentPlaces is how many decimals the app keeps for percent values
const PercentPlaces = 1

// Rules are the value rules of one field
type Rules struct {
	Type       string
	Options    []string // select and multi-select option names
	RatingMax  int
	EmailCheck bool
	PhoneCheck bool
	URLCheck   bool
	// Places is how many decimals a value keeps (decimal and percent fields); -1 keeps them all
	Places int
}

// FromMeta reads the rules of a field of type uidt from its settings
func FromMeta(uidt string, meta map[string]interface{}) Rules {
	r := Rules{Type: strings.ToLower(uidt), RatingMax: DefaultRatingMax, Places: -1}
	switch r.Type {
	case TypeSelect, TypeMultiSelect:
		r.Options = Options(meta["options"])
	case TypeRating:
		if n, ok := number(meta["ratingMax"]); ok && n > 0 {
			r.RatingMax = int(n)
		}
	case TypeEmail:
		r.EmailCheck = flag(meta, EmailCheckKey, true)
	case TypePhone:
		r.PhoneCheck = flag(meta, PhoneCheckKey, true)
	case TypeURL:
		r.URLCheck = flag(meta, URLCheckKey, false)
	case TypeDecimal:
		r.Places = DecimalPlaces(meta["precision"])
	case TypePercent:
		r.Places = PercentPlaces
	}
	return r
}

// HasOption reports whether name is one of the field's options
func (r Rules) HasOption(name string) bool {
	for _, o := range r.Options {
		if o == name {
			return true
		}
	}
	return false
}

// Options reads option names from meta.options: [{option, color}] or plain strings
func Options(raw interface{}) []string {
	list, _ := raw.([]interface{})
	options := make([]string, 0, len(list))
	for _, item := range list {
		switch v := item.(type) {
		case string:
			options = append(options, v)
		case map[string]interface{}:
			if name, ok := v["option"].(string); ok {
				options = append(options, name)
			}
		}
	}
	return options
}

// DecimalPlaces reads a decimal field's precision: a number of places (2), or the app's pattern ("1.00" is
// two places). -1 when the field has none.
func DecimalPlaces(raw interface{}) int {
	if s, ok := raw.(string); ok {
		if i := strings.IndexByte(s, '.'); i >= 0 {
			return len(s) - i - 1
		}
	}
	if n, ok := number(raw); ok && n >= 0 && n <= 20 && n == math.Trunc(n) {
		return int(n)
	}
	return -1
}

// Round rounds n to places decimals, halves away from zero; places < 0 keeps n
func Round(n float64, places int) float64 {
	if places < 0 {
		return n
	}
	p := math.Pow(10, float64(places))
	return math.Round(n*p) / p
}

func flag(meta map[string]interface{}, key string, def bool) bool {
	switch v := meta[key].(type) {
	case bool:
		return v
	case string:
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func number(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case interface{ Float64() (float64, error) }: // json.Number
		f, err := n.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(n, 64)
		return f, err == nil
	}
	return 0, false
}

// FormatIssue is why s isn't a valid email, phone number or URL for the field, or "" when it is valid, empty,
// or the field's check is off
func FormatIssue(r Rules, s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	switch {
	case r.Type == TypeEmail && r.EmailCheck && !ValidEmail(s):
		return "must be a valid email address"
	case r.Type == TypePhone && r.PhoneCheck && !ValidPhone(s):
		return "must be a valid phone number"
	case r.Type == TypeURL && r.URLCheck && !ValidURL(s):
		return "must be a valid URL"
	}
	return ""
}
