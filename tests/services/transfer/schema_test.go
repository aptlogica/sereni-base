// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package transfer_test

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
)

const schemaPath = "../../../docs/import-export-schema/base-export.schema.json"

type schemaDef struct {
	Required   []string                   `json:"required"`
	Properties map[string]json.RawMessage `json:"properties"`
	Enum       []string                   `json:"enum"`
}

func loadSchema(t *testing.T) (top schemaDef, defs map[string]schemaDef) {
	t.Helper()
	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Skip("schema not present")
	}
	var doc struct {
		schemaDef
		Defs map[string]schemaDef `json:"$defs"`
	}
	must(t, json.Unmarshal(raw, &doc))
	return doc.schemaDef, doc.Defs
}

// jsonFields lists the JSON names a struct writes (fields tagged "-" are not part of the file).
func jsonFields(v interface{}) []string {
	typ := reflect.TypeOf(v)
	var names []string
	for i := 0; i < typ.NumField(); i++ {
		tag := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
		if tag != "" && tag != "-" {
			names = append(names, tag)
		}
	}
	sort.Strings(names)
	return names
}

func keysOf(m map[string]json.RawMessage) []string {
	var names []string
	for k := range m {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// The Go structs and the JSON Schema describe the same file. If one changes without the other, this fails.
func TestStructsMatchTheSchema(t *testing.T) {
	top, defs := loadSchema(t)

	// the deprecated format block is in the schema and in the struct, which reads it and ignores it
	topProps := keysOf(top.Properties)
	if got, want := jsonFields(dto.BaseExportFile{}), topProps; !reflect.DeepEqual(got, want) {
		t.Errorf("top level: code writes %v, schema has %v", got, want)
	}

	cases := map[string]interface{}{
		"base":       dto.BaseExportBase{},
		"schema":     dto.BaseExportSchema{},
		"table":      dto.BaseExportTable{},
		"field":      dto.BaseExportField{},
		"view":       dto.BaseExportView{},
		"tableView":  dto.BaseExportView{},
		"data":       dto.BaseExportData{},
		"attachment": dto.BaseExportAttachment{},
		"row":        dto.BaseExportRow{},
		"cover":      dto.BaseExportCover{},
		"constraint": dto.BaseExportConstraint{},
		"function":   dto.BaseExportFunction{},
		"trigger":    dto.BaseExportTrigger{},
	}
	for name, v := range cases {
		want := keysOf(defs[name].Properties)
		if got := jsonFields(v); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: code writes %v, schema has %v", name, got, want)
		}
		for _, req := range defs[name].Required {
			if _, ok := defs[name].Properties[req]; !ok {
				t.Errorf("%s: schema requires %q but does not define it", name, req)
			}
		}
	}
}

// The field types the schema allows are the ones the app can create.
func TestSchemaFieldTypesMatchTheApp(t *testing.T) {
	_, defs := loadSchema(t)
	// links has one mapping per role and relation type (links_source_has-many, ...); the schema lists plain "links"
	names := map[string]bool{"links": true, "lookup": true}
	for name := range constant.UITypeMappings {
		if !strings.HasPrefix(name, "links_") {
			names[name] = true
		}
	}
	var want []string
	for name := range names {
		want = append(want, name)
	}
	sort.Strings(want)
	got := append([]string(nil), defs["uidt"].Enum...)
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("schema uidt %v\napp field types %v", got, want)
	}
}

// jsonFieldsInOrder lists the JSON names a struct writes, in the order the struct declares them.
func jsonFieldsInOrder(v interface{}) []string {
	typ := reflect.TypeOf(v)
	var names []string
	for i := 0; i < typ.NumField(); i++ {
		tag := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
		if tag != "" && tag != "-" {
			names = append(names, tag)
		}
	}
	return names
}

// keysInOrder lists the property names of a schema object in the order the schema file has them.
func keysInOrder(t *testing.T, raw json.RawMessage) []string {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(raw))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		t.Fatalf("not an object: %s", raw)
	}
	var keys []string
	for dec.More() {
		tok, err := dec.Token()
		must(t, err)
		keys = append(keys, tok.(string))
		var skip json.RawMessage
		must(t, dec.Decode(&skip))
	}
	return keys
}

// The properties of every object are in the same order in the schema file and in the file an export writes.
func TestSchemaPropertiesAreInTheOrderTheExportWrites(t *testing.T) {
	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Skip("schema not present")
	}
	var doc struct {
		Properties json.RawMessage `json:"properties"`
		Defs       map[string]struct {
			Required   []string        `json:"required"`
			Properties json.RawMessage `json:"properties"`
		} `json:"$defs"`
	}
	must(t, json.Unmarshal(raw, &doc))

	if got, want := keysInOrder(t, doc.Properties), jsonFieldsInOrder(dto.BaseExportFile{}); !reflect.DeepEqual(got, want) {
		t.Errorf("top level: schema %v, code writes %v", got, want)
	}
	cases := map[string]interface{}{
		"base": dto.BaseExportBase{}, "cover": dto.BaseExportCover{}, "schema": dto.BaseExportSchema{}, "table": dto.BaseExportTable{},
		"field": dto.BaseExportField{}, "constraint": dto.BaseExportConstraint{}, "view": dto.BaseExportView{}, "tableView": dto.BaseExportView{},
		"function": dto.BaseExportFunction{}, "trigger": dto.BaseExportTrigger{}, "data": dto.BaseExportData{},
		"row": dto.BaseExportRow{}, "attachment": dto.BaseExportAttachment{},
	}
	for name, v := range cases {
		props := keysInOrder(t, doc.Defs[name].Properties)
		if want := jsonFieldsInOrder(v); !reflect.DeepEqual(props, want) {
			t.Errorf("%s: schema %v, code writes %v", name, props, want)
		}
		// a required list follows the same order as the properties
		var wantRequired []string
		for _, p := range props {
			for _, r := range doc.Defs[name].Required {
				if p == r {
					wantRequired = append(wantRequired, p)
				}
			}
		}
		if !reflect.DeepEqual(doc.Defs[name].Required, wantRequired) {
			t.Errorf("%s: required %v is not in property order %v", name, doc.Defs[name].Required, wantRequired)
		}
	}
}

// The tree at the top of the Quick Guide lists each object's properties in the order the file writes them.
func TestQuickGuideTreeFollowsTheOrderOfTheFile(t *testing.T) {
	raw, err := os.ReadFile("../../../docs/import-export-schema/BASE_EXPORT_QUICK_GUIDE.md")
	if err != nil {
		t.Skip("quick guide not present")
	}
	text := string(raw)
	start := strings.Index(text, "file ★")
	end := strings.Index(text[start:], "\n```")
	if start < 0 || end < 0 {
		t.Fatal("the tree is missing from the quick guide")
	}
	tree := text[start : start+end]

	// slice returns the part of the tree between two markers
	slice := func(from, to string) string {
		i := strings.Index(tree, from)
		if i < 0 {
			t.Fatalf("%q is not in the tree", from)
		}
		rest := tree[i:]
		if to == "" {
			return rest
		}
		j := strings.Index(rest[len(from):], to)
		if j < 0 {
			t.Fatalf("%q is not in the tree after %q", to, from)
		}
		return rest[:len(from)+j]
	}
	// firstSeen lists the names in the order they first appear as "name ★" or "name ○"
	firstSeen := func(part string, names []string) []string {
		type hit struct {
			name string
			at   int
		}
		var hits []hit
		for _, n := range names {
			loc := regexp.MustCompile(`\b` + regexp.QuoteMeta(n) + ` [★○]`).FindStringIndex(part)
			if loc == nil {
				t.Errorf("%q is not in this part of the tree:\n%s", n, part)
				continue
			}
			hits = append(hits, hit{n, loc[0]})
		}
		sort.Slice(hits, func(i, j int) bool { return hits[i].at < hits[j].at })
		out := make([]string, len(hits))
		for i, h := range hits {
			out[i] = h.name
		}
		return out
	}

	cases := []struct {
		label string
		part  string
		names []string
	}{
		{"base", slice("base ★", "schema ★"), jsonFieldsInOrderWithout(dto.BaseExportBase{}, "type")},
		{"table", slice("tables ★ [ ]", "data ★"), jsonFieldsInOrder(dto.BaseExportTable{})},
		{"field", slice("fields ★ [ ]", "views ○"), jsonFieldsInOrder(dto.BaseExportField{})},
		{"view", slice("views ○ [ ]", "primary_key ○"), jsonFieldsInOrderWithout(dto.BaseExportView{}, "table")},
		{"data", slice("data ★", ""), jsonFieldsInOrder(dto.BaseExportData{})},
		{"attachment", slice("attachments ○ [ ]", ""), jsonFieldsInOrder(dto.BaseExportAttachment{})},
		{"cover", slice("cover ○", "schema ★"), jsonFieldsInOrder(dto.BaseExportCover{})},
	}
	for _, c := range cases {
		names := c.names
		if c.label == "base" {
			names = names[:len(names)-1] // cover has its own line
		}
		if c.label == "data" {
			names = []string{"row_counts", "tables", "attachments"}
		}
		if c.label == "table" {
			// the list of a table starts at its own properties, not at the ones of its fields
			names = []string{"key", "title", "description", "order_index", "fields", "views", "primary_key", "constraints", "functions", "triggers"}
		}
		if got := firstSeen(c.part, names); !reflect.DeepEqual(got, names) {
			t.Errorf("%s: the tree lists %v, the file writes %v", c.label, got, names)
		}
	}
}

func jsonFieldsInOrderWithout(v interface{}, skip string) []string {
	var out []string
	for _, n := range jsonFieldsInOrder(v) {
		if n != skip {
			out = append(out, n)
		}
	}
	return out
}
