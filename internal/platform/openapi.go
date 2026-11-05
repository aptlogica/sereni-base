// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// openAPIBaseYAML is docs/api-platform/openapi.yaml (a test keeps the two equal). Its components and schema
// paths are reused as they are; its generic records paths are the templates of every table's paths.
//
//go:embed openapi_base.yaml
var openAPIBaseYAML []byte

var (
	openAPIBaseOnce sync.Once
	openAPIBaseJSON []byte
	openAPIBaseErr  error
)

// openAPIBase returns a fresh copy of the generic spec
func openAPIBase() (schemaObject, error) {
	openAPIBaseOnce.Do(func() {
		var doc schemaObject
		if openAPIBaseErr = yaml.Unmarshal(openAPIBaseYAML, &doc); openAPIBaseErr == nil {
			openAPIBaseJSON, openAPIBaseErr = json.Marshal(doc)
		}
	})
	if openAPIBaseErr != nil {
		return nil, openAPIBaseErr
	}
	var doc schemaObject
	err := json.Unmarshal(openAPIBaseJSON, &doc)
	return doc, err
}

const (
	genericTablePath = "/bases/{base}/tables/{table}"
	schemasPrefix    = "#/components/schemas/"
)

// The generic records paths, by what follows the table in the path
var recordsPathSuffixes = []string{"/records", "/records/{record}", "/records/delete"}

// cachedSpec is a built spec and its ETag
type cachedSpec struct {
	body []byte
	etag string
}

// GET /openapi.json: the workspace's spec, with typed paths and records for every table. ?base= limits it to
// one base. It is the spec itself, not wrapped in the envelope, so tools can read it directly.
func (a *API) openAPISpec(c *gin.Context) {
	ws := currentWorkspace(c)
	var only *baseInfo
	if raw := c.Query("base"); raw != "" {
		base, err := a.resolveBase(c, ws, raw)
		if err != nil {
			respondError(c, err)
			return
		}
		only = base
	}
	spec, err := a.spec(c, ws, only)
	if err != nil {
		respondError(c, err)
		return
	}
	c.Header("ETag", spec.etag)
	c.Header("Cache-Control", "no-cache")
	if c.GetHeader("If-None-Match") == spec.etag {
		c.Status(http.StatusNotModified)
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", spec.body)
}

// spec returns the built spec, cached with the rest of the schema until schema_version changes
func (a *API) spec(ctx context.Context, ws workspaceRef, only *baseInfo) (cachedSpec, error) {
	key := ""
	if only != nil {
		key = only.ID
	}
	snap := a.schemas.snapshot(ws)
	snap.mu.Lock()
	cached, ok := snap.specs[key]
	snap.mu.Unlock()
	if ok {
		return cached, nil
	}

	body, err := a.buildSpec(ctx, ws, only)
	if err != nil {
		return cachedSpec{}, err
	}
	sum := sha256.Sum256(body)
	cached = cachedSpec{body: body, etag: `"` + hex.EncodeToString(sum[:8]) + `"`}
	snap.mu.Lock()
	snap.specs[key] = cached
	snap.mu.Unlock()
	return cached, nil
}

// specBuilder adds the tables of a workspace to a copy of the generic spec
type specBuilder struct {
	a         *API
	ws        workspaceRef
	doc       schemaObject
	paths     schemaObject
	schemas   schemaObject
	templates map[string]schemaObject
	tags      []any
	tagNames  map[string]bool
}

func (a *API) buildSpec(ctx context.Context, ws workspaceRef, only *baseInfo) ([]byte, error) {
	doc, err := openAPIBase()
	if err != nil {
		return nil, err
	}
	info := child(doc, "info")
	info["title"] = "SereniBase Workspace API: " + ws.PublicID
	doc["servers"] = []any{schemaObject{"url": "/api/v1/" + ws.PublicID}}

	b := &specBuilder{
		a: a, ws: ws, doc: doc,
		paths:     child(doc, "paths"),
		schemas:   child(child(doc, "components"), "schemas"),
		templates: map[string]schemaObject{},
		tagNames:  map[string]bool{},
	}
	for _, suffix := range recordsPathSuffixes {
		b.templates[suffix] = child(b.paths, genericTablePath+suffix)
		delete(b.paths, genericTablePath+suffix)
	}
	// Keep the Schema tag; the generic Records tag gives way to one tag per table
	if tags, ok := doc["tags"].([]any); ok {
		for _, t := range tags {
			if tag, ok := t.(schemaObject); ok && tag["name"] == "Schema" {
				b.tags = append(b.tags, tag)
			}
		}
	}

	bases, err := a.bases(ctx, ws)
	if err != nil {
		return nil, err
	}
	baseNames := nameCounts(bases, func(x *baseInfo) string { return x.Title })
	if only != nil {
		bases = []*baseInfo{only}
	}
	for _, base := range bases {
		tables, err := a.tables(ctx, ws, base)
		if err != nil {
			return nil, err
		}
		tableNames := nameCounts(tables, func(x *tableInfo) string { return x.Title })
		basePath := "/bases/" + pathSegment(base.Title, base.ID, PrefixBase, baseNames)
		for _, table := range tables {
			tablePath := basePath + "/tables/" + pathSegment(table.Title, table.ID, PrefixTable, tableNames)
			if err := b.addTable(ctx, base, table, tablePath); err != nil {
				return nil, err
			}
		}
	}
	doc["tags"] = b.tags
	return json.Marshal(doc)
}

func nameCounts[T any](items []T, name func(T) string) map[string]int {
	counts := map[string]int{}
	for _, item := range items {
		counts[name(item)]++
	}
	return counts
}

// pathSegment is the name, URL-encoded, or the ID when the name is shared (it would be ambiguous), empty,
// or looks like an ID
func pathSegment(name, id, prefix string, counts map[string]int) string {
	if name == "" || counts[name] > 1 || strings.HasPrefix(name, prefix) {
		return EncodeUUID(prefix, id)
	}
	return url.PathEscape(name)
}

// addTable adds the table's record types and its records paths
func (b *specBuilder) addTable(ctx context.Context, base *baseInfo, table *tableInfo, tablePath string) error {
	fields, err := b.a.fields(ctx, b.ws, table)
	if err != nil {
		return err
	}
	tid := EncodeUUID(PrefixTable, table.ID)
	keys := fieldKeys(fields, false)

	read, write, bulkWrite := schemaObject{}, schemaObject{}, schemaObject{}
	var selectable, sortable, filterable []string
	for _, f := range fields {
		key := keys[f]
		selectable = append(selectable, key)
		if f.Type == typeLookup {
			continue // inside the link field's objects
		}
		var s schemaObject
		if f.Type == typeLinks {
			linkRef, err := b.addLinkRef(ctx, tid, fields, f)
			if err != nil {
				return err
			}
			s = schemaObject{"type": "array", "items": schemaRef(linkRef)}
		} else {
			s = fieldSchema(f)
		}
		about := fmt.Sprintf("%s field, %s.", f.APIType(), f.PublicID())
		if extra, ok := s["description"].(string); ok {
			about += " " + extra
		}
		s["description"] = about
		if f.ReadOnly {
			s["readOnly"] = true
		} else if f.Type == typeLinks {
			ls := linkWriteSchema(f)
			ls["description"] = about + " Record IDs, or objects with an id. Only when writing one record."
			write[key] = ls
		} else {
			write[key] = s
			bulkWrite[key] = s
		}
		read[key] = s

		if !unsortableTypes[f.Type] {
			sortable = append(sortable, "`"+key+"`")
		}
		if ops := filterOperators(f); len(ops) > 0 {
			filterable = append(filterable, fmt.Sprintf("- `%s`: %s", key, strings.Join(ops, ", ")))
		}
	}

	title := table.Title
	b.schemas["Fields_"+tid] = schemaObject{"type": "object", "title": title + " fields", "properties": read,
		"description": "Values of " + title + " by field name. Fields that share a name are keyed by their ID."}
	b.schemas["FieldsWrite_"+tid] = schemaObject{"type": "object", "title": title + " fields (write)", "properties": write}
	b.schemas["FieldsBulkWrite_"+tid] = schemaObject{"type": "object", "title": title + " fields (bulk write)", "properties": bulkWrite,
		"description": "Link fields can't be set in bulk writes."}
	b.schemas["FieldsInput_"+tid] = fieldsInput("FieldsWrite_" + tid)
	b.schemas["FieldsBulkInput_"+tid] = fieldsInput("FieldsBulkWrite_" + tid)
	b.schemas["Record_"+tid] = schemaObject{
		"type": "object", "title": title + " record",
		"required": []string{"id", "fields", "created_at", "updated_at"},
		"properties": schemaObject{
			"id":         schemaRef(recordIDRef),
			"fields":     schemaRef(schemasPrefix + "Fields_" + tid),
			"created_at": schemaObject{"type": "string", "format": "date-time"},
			"updated_at": schemaObject{"type": "string", "format": "date-time"},
		},
	}

	tag := base.Title + " / " + table.Title
	if b.tagNames[tag] {
		tag += " (" + tid + ")"
	}
	b.tagNames[tag] = true
	b.tags = append(b.tags, schemaObject{"name": tag, "description": "Records of table " + tid})

	refs := map[string]string{
		schemasPrefix + "Record":      schemasPrefix + "Record_" + tid,
		schemasPrefix + "FieldsInput": schemasPrefix + "FieldsInput_" + tid,
		schemasPrefix + "Fields":      schemasPrefix + "FieldsBulkWrite_" + tid, // only the bulk update uses it
	}
	for _, suffix := range recordsPathSuffixes {
		item := copyObject(b.templates[suffix])
		replaceRefs(item, refs)
		item["parameters"] = withoutTableParams(item["parameters"])
		for _, method := range []string{"get", "post", "patch", "delete"} {
			op := child(item, method)
			if op == nil {
				continue
			}
			op["operationId"] = fmt.Sprintf("%s_%s", op["operationId"], tid)
			op["tags"] = []any{tag}
			if body := child(child(child(op, "requestBody"), "content"), "application/json"); body != nil {
				delete(body, "examples") // the generic examples use made-up fields
			}
		}
		if suffix == "/records" {
			// Bulk creates can't set links
			if oneOf, ok := child(child(child(child(child(item, "post"), "requestBody"), "content"), "application/json"), "schema")["oneOf"].([]any); ok && len(oneOf) == 2 {
				child(child(oneOf[1].(schemaObject), "properties"), "records")["items"] = schemaRef(schemasPrefix + "FieldsBulkInput_" + tid)
			}
			describeListParams(child(item, "get"), selectable, sortable, filterable)
		}
		b.paths[tablePath+suffix] = item
	}
	return nil
}

// addLinkRef adds the schema of a link field's objects: the linked record's id and the lookups through the link
func (b *specBuilder) addLinkRef(ctx context.Context, tid string, fields []*field, link *field) (string, error) {
	targets, keys, err := b.a.linkObjectKeys(ctx, b.ws, fields, link)
	if err != nil {
		return "", err
	}
	props := schemaObject{"id": schemaRef(recordIDRef)}
	for _, tf := range targets {
		s := fieldSchema(tf)
		s["description"] = fmt.Sprintf("Lookup of %q (%s field, %s) in the linked record.", tf.Name, tf.APIType(), tf.PublicID())
		props[keys[tf]] = s
	}
	name := "LinkRef_" + tid + "_" + link.PublicID()
	b.schemas[name] = schemaObject{"type": "object", "title": link.Name + " link", "required": []string{"id"}, "properties": props}
	return schemasPrefix + name, nil
}

func fieldsInput(fieldsSchema string) schemaObject {
	return schemaObject{
		"type": "object", "additionalProperties": false, "required": []string{"fields"},
		"properties": schemaObject{"fields": schemaRef(schemasPrefix + fieldsSchema)},
	}
}

// describeListParams fills the list operation's select, sort and filter parameters with the table's fields
func describeListParams(op schemaObject, selectable, sortable, filterable []string) {
	params, _ := op["parameters"].([]any)
	for _, p := range params {
		param, ok := p.(schemaObject)
		if !ok {
			continue
		}
		switch param["name"] {
		case "select":
			items := schemaObject{"type": "string"}
			if len(selectable) > 0 {
				items["enum"] = selectable
			}
			param["schema"] = schemaObject{"type": "array", "items": items}
			param["style"] = "form"
			param["explode"] = false
			delete(param, "example")
		case "sort":
			param["description"] = fmt.Sprintf("%s\n\nSortable: `id`, `created_at`, `updated_at`", param["description"])
			if len(sortable) > 0 {
				param["description"] = fmt.Sprintf("%s, %s", param["description"], strings.Join(sortable, ", "))
			}
			param["description"] = fmt.Sprintf("%s.", param["description"])
		case "filter":
			text := "`filter[Field]=value` (equals) or `filter[Field][operator]=value`. `id`, `created_at` and `updated_at` " +
				"take eq, neq, gt, gte, lt, lte, in and is_null."
			if len(filterable) > 0 {
				text += " Operators by field:\n\n" + strings.Join(filterable, "\n")
			}
			param["description"] = text
			delete(param, "example")
		}
	}
}

// withoutTableParams drops the base and table path parameters: a table's paths name them
func withoutTableParams(raw any) []any {
	params, _ := raw.([]any)
	out := []any{}
	for _, p := range params {
		if param, ok := p.(schemaObject); ok {
			if r := param["$ref"]; r == "#/components/parameters/Base" || r == "#/components/parameters/Table" {
				continue
			}
		}
		out = append(out, p)
	}
	return out
}

// replaceRefs rewrites the $ref values found in the map, at any depth
func replaceRefs(v any, refs map[string]string) {
	switch x := v.(type) {
	case schemaObject:
		if r, ok := x["$ref"].(string); ok {
			if to, ok := refs[r]; ok {
				x["$ref"] = to
			}
		}
		for _, item := range x {
			replaceRefs(item, refs)
		}
	case []any:
		for _, item := range x {
			replaceRefs(item, refs)
		}
	}
}

// child returns the object under key, or nil
func child(m schemaObject, key string) schemaObject {
	c, _ := m[key].(schemaObject)
	return c
}

// copyObject deep-copies an object decoded from JSON
func copyObject(m schemaObject) schemaObject {
	out := make(schemaObject, len(m))
	for k, v := range m {
		out[k] = copyValue(v)
	}
	return out
}

func copyValue(v any) any {
	switch x := v.(type) {
	case schemaObject:
		return copyObject(x)
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = copyValue(item)
		}
		return out
	}
	return v
}
