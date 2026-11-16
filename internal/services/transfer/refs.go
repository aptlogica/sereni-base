// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/aptlogica/sereni-base/internal/dto"
)

var (
	refPattern     = regexp.MustCompile(`^@[a-z][a-z0-9_]{0,62}$`)
	keyPattern     = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)
	nonKeyChars    = regexp.MustCompile(`[^a-z0-9]+`)
	maxKeyBodySize = 50
)

// keyAllocator hands out readable file keys (t_orders, f_name, ...) that are unique in one file.
type keyAllocator struct {
	used map[string]int
}

func newKeyAllocator() *keyAllocator {
	return &keyAllocator{used: map[string]int{}}
}

// next returns prefix + slug(title); a repeat gets a numeric suffix.
func (k *keyAllocator) next(prefix, title string) string {
	body := strings.Trim(nonKeyChars.ReplaceAllString(strings.ToLower(title), "_"), "_")
	if body == "" {
		body = "item"
	}
	if len(body) > maxKeyBodySize {
		body = strings.TrimRight(body[:maxKeyBodySize], "_")
	}
	base := prefix + body
	// A suffixed key can equal the key of another title (titles "Links", "Links" and "Links 2"
	// would all want f_links_2), so count up until the key is really unused.
	key := base
	for n := 2; k.used[key] > 0; n++ {
		key = base + "_" + strconv.Itoa(n)
	}
	k.used[key]++
	return key
}

// normalizeMap copies a map through JSON so it holds only maps, slices, strings and numbers
// (database values such as uuid.UUID become plain strings).
func normalizeMap(in map[string]interface{}) map[string]interface{} {
	if in == nil {
		return nil
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return in
	}
	out := map[string]interface{}{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return in
	}
	return out
}

// rewriteRefs returns a copy of v in which every string (and map key) found in mapping is replaced by its value.
// Export uses it to turn ids into "@key", import to turn "@key" into the new ids.
func rewriteRefs(v interface{}, mapping map[string]string) interface{} {
	switch x := v.(type) {
	case string:
		if to, ok := mapping[x]; ok {
			return to
		}
		return x
	case map[string]interface{}:
		out := make(map[string]interface{}, len(x))
		for k, val := range x {
			if to, ok := mapping[k]; ok {
				k = to
			}
			out[k] = rewriteRefs(val, mapping)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(x))
		for i, val := range x {
			out[i] = rewriteRefs(val, mapping)
		}
		return out
	default:
		return v
	}
}

func rewriteMap(in map[string]interface{}, mapping map[string]string) map[string]interface{} {
	if in == nil {
		return map[string]interface{}{}
	}
	out, _ := rewriteRefs(normalizeMap(in), mapping).(map[string]interface{})
	return out
}

// collectRefs adds every "@key" string (values and map keys) found in v to out.
func collectRefs(v interface{}, out map[string]struct{}) {
	switch x := v.(type) {
	case string:
		if strings.HasPrefix(x, dto.BaseExportRefPrefix) && refPattern.MatchString(x) {
			out[x] = struct{}{}
		}
	case map[string]interface{}:
		for k, val := range x {
			collectRefs(k, out)
			collectRefs(val, out)
		}
	case []interface{}:
		for _, val := range x {
			collectRefs(val, out)
		}
	}
}

// containsString reports whether v holds a string (or map key) that is in set.
func containsString(v interface{}, set map[string]struct{}) bool {
	switch x := v.(type) {
	case string:
		_, ok := set[x]
		return ok
	case map[string]interface{}:
		for k, val := range x {
			if _, ok := set[k]; ok || containsString(val, set) {
				return true
			}
		}
	case []interface{}:
		for _, val := range x {
			if containsString(val, set) {
				return true
			}
		}
	}
	return false
}

// idString reads an id that may be a string or a uuid.UUID.
func idString(v interface{}) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case fmt.Stringer:
		return x.String()
	default:
		return ""
	}
}

// nestedString reads meta[outer][inner] as an id string.
func nestedString(meta map[string]interface{}, outer, inner string) string {
	obj, ok := meta[outer].(map[string]interface{})
	if !ok {
		return ""
	}
	return idString(obj[inner])
}

// cellKey turns a primary-key value (number or text, as read from the database or a file) into a map key.
func cellKey(v interface{}) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(x), 'f', -1, 32)
	case json.Number:
		return x.String()
	default:
		return fmt.Sprint(x)
	}
}

// toList reads a link cell: a JSON array, a JSON text of an array, or one value.
func toList(v interface{}) []interface{} {
	switch x := v.(type) {
	case nil:
		return nil
	case []interface{}:
		return x
	case []map[string]interface{}:
		out := make([]interface{}, len(x))
		for i, m := range x {
			out[i] = m
		}
		return out
	case []string:
		out := make([]interface{}, len(x))
		for i, s := range x {
			out[i] = s
		}
		return out
	case []byte:
		return toList(string(x))
	case string:
		trimmed := strings.TrimSpace(x)
		if strings.HasPrefix(trimmed, "[") {
			var out []interface{}
			if err := json.Unmarshal([]byte(trimmed), &out); err == nil {
				return out
			}
		}
		if trimmed == "" {
			return nil
		}
		return []interface{}{x}
	default:
		return []interface{}{x}
	}
}

var conditionToken = regexp.MustCompile(`\{(f_[a-z0-9_]{1,62})\}`)

// tokenizeCondition writes the column names of a check condition as {field_key}. The longest names are matched
// first, and only whole words, so "age" does not match inside "average".
func tokenizeCondition(condition string, names map[string]string) string {
	if condition == "" || len(names) == 0 {
		return condition
	}
	list := make([]string, 0, len(names))
	for name := range names {
		if name != "" {
			list = append(list, name)
		}
	}
	sort.Slice(list, func(i, j int) bool { return len(list[i]) > len(list[j]) })
	quoted := make([]string, len(list))
	for i, name := range list {
		quoted[i] = regexp.QuoteMeta(name)
	}
	re := regexp.MustCompile(`\b(?:` + strings.Join(quoted, "|") + `)\b`)
	return re.ReplaceAllStringFunc(condition, func(name string) string { return "{" + names[name] + "}" })
}

// detokenizeCondition turns {field_key} back into the column name of the new field.
func detokenizeCondition(condition string, names map[string]string) string {
	return conditionToken.ReplaceAllStringFunc(condition, func(token string) string {
		if name, ok := names[token[1:len(token)-1]]; ok {
			return name
		}
		return token
	})
}

// conditionKeys lists the {field_key} tokens of a condition.
func conditionKeys(condition string) []string {
	var keys []string
	for _, m := range conditionToken.FindAllStringSubmatch(condition, -1) {
		keys = append(keys, m[1])
	}
	return keys
}
