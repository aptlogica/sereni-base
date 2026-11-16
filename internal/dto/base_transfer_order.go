// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

import (
	"bytes"
	"encoding/json"
	"sort"
)

// metaOrder is the order of the known keys of a field's meta, by field type; any other key follows, alphabetically.
func metaOrder(uidt string) []string {
	switch uidt {
	case "links":
		return []string{"relation", "relation_id", "entity_role", "display_column_id", "is_self_link"}
	case "lookup":
		return []string{"lookup_column_id", "relation_id", "link_column_id"}
	}
	return []string{"options", "defaultValue", "formula", "idGeneration"}
}

// fieldMetaNested orders the objects inside those keys.
var fieldMetaNested = map[string][]string{
	"relation":     {"type", "with"},
	"options":      {"option", "color"},
	"idGeneration": {"mode", "version"},
}

// OrderBy sets the order in which the values of the row are written: the keys given first, in that order, then any
// other key alphabetically. It does not change the row.
func (r *BaseExportRow) OrderBy(keys []string) {
	r.order = keys
}

// MarshalJSON writes the values in the order set by OrderBy.
func (r BaseExportRow) MarshalJSON() ([]byte, error) {
	values, err := marshalOrdered(r.Values, r.order, nil)
	if err != nil {
		return nil, err
	}
	return marshalPlain(struct {
		RID    string          `json:"_rid"`
		Values json.RawMessage `json:"values"`
	}{RID: r.RID, Values: values})
}

// MarshalJSON writes the meta with its main keys first.
func (f BaseExportField) MarshalJSON() ([]byte, error) {
	var meta json.RawMessage
	if len(f.Meta) > 0 {
		var err error
		if meta, err = marshalOrdered(f.Meta, metaOrder(f.UIDT), fieldMetaNested); err != nil {
			return nil, err
		}
	}
	return marshalPlain(struct {
		Key         string          `json:"key"`
		Title       string          `json:"title"`
		Description string          `json:"description,omitempty"`
		UIDT        string          `json:"uidt"`
		DT          string          `json:"dt,omitempty"`
		OrderIndex  float64         `json:"order_index,omitempty"`
		System      bool            `json:"system,omitempty"`
		ColumnName  string          `json:"column_name,omitempty"`
		Meta        json.RawMessage `json:"meta,omitempty"`
	}{f.Key, f.Title, f.Description, f.UIDT, f.DT, f.OrderIndex, f.System, f.ColumnName, meta})
}

// marshalPlain is json.Marshal without escaping <, > and &, so SQL such as "a >= 0" stays readable.
func marshalPlain(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// marshalOrdered writes a map with the keys in `first` leading (when present), then the others alphabetically.
// nested gives the order of the objects found under some keys, directly or inside a list.
func marshalOrdered(m map[string]interface{}, first []string, nested map[string][]string) (json.RawMessage, error) {
	seen := make(map[string]bool, len(m))
	keys := make([]string, 0, len(m))
	for _, k := range first {
		if _, ok := m[k]; ok && !seen[k] {
			keys = append(keys, k)
			seen[k] = true
		}
	}
	var rest []string
	for k := range m {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	keys = append(keys, rest...)

	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		name, err := marshalPlain(k)
		if err != nil {
			return nil, err
		}
		buf.Write(name)
		buf.WriteByte(':')
		value, err := orderedValue(m[k], nested[k])
		if err != nil {
			return nil, err
		}
		buf.Write(value)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// orderedValue writes v, putting the keys of an object (or of each object of a list) in the given order.
func orderedValue(v interface{}, order []string) (json.RawMessage, error) {
	if len(order) > 0 {
		switch x := v.(type) {
		case map[string]interface{}:
			return marshalOrdered(x, order, nil)
		case []interface{}:
			var buf bytes.Buffer
			buf.WriteByte('[')
			for i, item := range x {
				if i > 0 {
					buf.WriteByte(',')
				}
				part, err := orderedValue(item, order)
				if err != nil {
					return nil, err
				}
				buf.Write(part)
			}
			buf.WriteByte(']')
			return buf.Bytes(), nil
		}
	}
	return marshalPlain(v)
}
