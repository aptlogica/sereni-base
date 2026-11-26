// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"context"
	"fmt"
	"strings"

	"github.com/aptlogica/sereni-base/internal/constant"
)

// Types base management refuses until the data API can read and write them
var typesNotSupportedYet = map[string]string{
	"formula":    "Formula values are calculated in the app, and the data API doesn't read or write them yet.",
	"attachment": "Attachments aren't supported by the data API yet.",
	"user":       "User fields aren't supported by the data API yet.",
}

// Types that are computed or set by the system; nobody can create them by hand
var typesSetBySystem = map[string]bool{
	"createdtime": true, "lastmodifiedtime": true, "createdby": true, "lastmodifiedby": true, "button": true,
}

// columnConstraints are the optional per-column constraints of a create or change request
type columnConstraints struct {
	present      bool
	Default      interface{}
	hasDefault   bool
	IDGeneration *idGenerationInput
	Unique       *bool
	Check        map[string]interface{} // category and where
}

type idGenerationInput struct {
	Mode     string
	Version  string
	Backfill bool
}

// columnInput is a column as a create or change request describes it
type columnInput struct {
	Title       *string
	Description *string
	UIDT        string // as the app spells it, e.g. longText
	Meta        map[string]interface{}
	hasMeta     bool
	Constraints columnConstraints
	// change requests only
	EmptyFirst bool
}

// uidtByLower maps a lowercased type name to the app's spelling of it
func uidtByLower() map[string]string {
	out := make(map[string]string, len(constant.UITypeMappings)+2)
	for k := range constant.UITypeMappings {
		out[strings.ToLower(k)] = k
	}
	out["links"], out["lookup"] = "links", "lookup"
	return out
}

// appUIDT returns the app's spelling of a type name given in any case
func appUIDT(name string) (string, bool) {
	u, ok := uidtByLower()[strings.ToLower(strings.TrimSpace(name))]
	return u, ok
}

// parseColumnInput reads a column request body. allowed lists its keys; create requires title and uidt.
// All problems are collected so the caller can answer them together.
func parseColumnInput(body map[string]interface{}, where string, create bool) (*columnInput, []ErrorDetail, error) {
	if err := onlyKeys(body, where, "title", "name", "description", "uidt", "meta", "constraints", "empty_first"); err != nil {
		return nil, nil, err
	}
	in := &columnInput{}
	var details []ErrorDetail
	bad := func(field, issue string) {
		details = append(details, ErrorDetail{Field: where + "." + field, Issue: issue})
	}

	for _, key := range []string{"title", "name"} {
		if v, ok := body[key]; ok {
			s, isStr := v.(string)
			if !isStr || strings.TrimSpace(s) == "" {
				bad(key, "must be a non-empty string")
				continue
			}
			s = strings.TrimSpace(s)
			in.Title = &s
		}
	}
	if v, ok := body["description"]; ok {
		s, isStr := v.(string)
		if !isStr {
			bad("description", "must be a string")
		} else {
			in.Description = &s
		}
	}
	if v, ok := body["uidt"]; ok {
		s, isStr := v.(string)
		if !isStr {
			bad("uidt", "must be a string")
		} else if u, known := appUIDT(s); !known {
			bad("uidt", fmt.Sprintf("unknown type %q", s))
		} else {
			in.UIDT = u
		}
	}
	if v, ok := body["empty_first"].(bool); ok {
		in.EmptyFirst = v
	}
	if v, ok := body["meta"]; ok {
		m, isObj := v.(map[string]interface{})
		if !isObj {
			bad("meta", "must be a JSON object")
		} else {
			in.Meta, in.hasMeta = m, true
		}
	}
	if create {
		if in.Title == nil {
			bad("title", "is required")
		}
		if in.UIDT == "" {
			if _, given := body["uidt"]; !given {
				bad("uidt", "is required")
			}
		}
	}
	if in.UIDT != "" {
		lower := strings.ToLower(in.UIDT)
		if why, no := typesNotSupportedYet[lower]; no {
			return nil, nil, NewError(CodeTypeNotSupported, fmt.Sprintf("The type %q can't be used through the API yet. %s", in.UIDT, why))
		}
		if typesSetBySystem[lower] {
			return nil, nil, NewError(CodeTypeNotSupported, fmt.Sprintf("The type %q is set by the system and can't be created or chosen.", in.UIDT))
		}
	}
	if v, ok := body["constraints"]; ok {
		obj, isObj := v.(map[string]interface{})
		if !isObj {
			bad("constraints", "must be a JSON object")
		} else {
			cons, cd := parseColumnConstraints(obj, where+".constraints")
			in.Constraints = cons
			details = append(details, cd...)
		}
	}
	// default and id_generation are column settings, so they are given in meta
	if in.hasMeta {
		rest, cd := takeMetaSettings(in.Meta, &in.Constraints, where+".meta")
		in.Meta = rest
		details = append(details, cd...)
	}
	return in, details, nil
}

// takeMetaSettings removes the default and id_generation settings from a request's meta and records them in
// cons. The rest of meta is returned as it was given.
func takeMetaSettings(meta map[string]interface{}, cons *columnConstraints, where string) (map[string]interface{}, []ErrorDetail) {
	var details []ErrorDetail
	rest := make(map[string]interface{}, len(meta))
	for k, v := range meta {
		if k != "default" && k != "id_generation" {
			rest[k] = v
		}
	}
	if v, ok := meta["default"]; ok {
		cons.Default, cons.hasDefault, cons.present = v, true, true
	}
	if v, ok := meta["id_generation"]; ok {
		cons.present = true
		g, isObj := v.(map[string]interface{})
		if !isObj {
			details = append(details, ErrorDetail{Field: where + ".id_generation", Issue: "must be a JSON object like {\"mode\": \"auto\"}"})
		} else {
			ig := &idGenerationInput{Mode: constant.IDModeAuto}
			if m, ok := g["mode"].(string); ok && m != "" {
				ig.Mode = m
			}
			ig.Version, _ = g["version"].(string)
			ig.Backfill, _ = g["backfill"].(bool)
			cons.IDGeneration = ig
		}
	}
	return rest, details
}

func parseColumnConstraints(obj map[string]interface{}, where string) (columnConstraints, []ErrorDetail) {
	cons := columnConstraints{present: true}
	var details []ErrorDetail
	bad := func(field, issue string) {
		details = append(details, ErrorDetail{Field: where + "." + field, Issue: issue})
	}
	for k := range obj {
		switch k {
		case "unique", "check":
		case "default", "id_generation":
			bad(k, "is a column setting: give it in meta, not in constraints")
		default:
			bad(k, "unknown constraint; allowed: unique, check")
		}
	}
	if v, ok := obj["unique"]; ok {
		b, isBool := v.(bool)
		if !isBool {
			bad("unique", "must be true or false")
		} else {
			cons.Unique = &b
		}
	}
	if v, ok := obj["check"]; ok {
		c, isObj := v.(map[string]interface{})
		if !isObj {
			bad("check", "must be a JSON object with a where condition")
		} else {
			cons.Check = c
		}
	}
	return cons, details
}

// linkOrLookup reports whether the type takes no per-column constraints
func linkOrLookup(uidt string) bool {
	l := strings.ToLower(uidt)
	return l == "links" || l == "lookup"
}

// resolveTypeMeta translates the names and IDs the API accepts in meta into the IDs the app stores:
// a link's target table (meta.relation.with) and a lookup's link and looked-up fields
func (a *API) resolveTypeMeta(ctx context.Context, base baseRef, table *tableInfo, uidt string, meta map[string]interface{}, metas []*columnMeta) (map[string]interface{}, error) {
	out := map[string]interface{}{}
	for k, v := range meta {
		out[k] = v
	}
	switch strings.ToLower(uidt) {
	case "links":
		rel, ok := out["relation"].(map[string]interface{})
		if !ok {
			return nil, NewError(CodeValidationFailed, "A links column needs meta.relation with the linked table and the relation type.",
				ErrorDetail{Field: "meta.relation", Issue: "required: {\"with\": <table name or ID>, \"type\": \"has-many\" | \"many-to-many\" | \"one-to-one\"}"})
		}
		with, _ := rel["with"].(string)
		target, err := a.resolveTable(ctx, base, with)
		if err != nil {
			return nil, NewError(CodeValidationFailed, "The linked table doesn't exist in this base.", ErrorDetail{Field: "meta.relation.with", Issue: err.Error()})
		}
		copied := map[string]interface{}{}
		for k, v := range rel {
			copied[k] = v
		}
		copied["with"] = target.ID
		out["relation"] = copied
	case "lookup":
		linkRef, _ := out["link_column_id"].(string)
		link, err := pickMeta(metas, linkRef)
		if err != nil || strings.ToLower(link.UIDT) != "links" {
			return nil, NewError(CodeValidationFailed, "A lookup column needs meta.link_column_id: a links column of this table.",
				ErrorDetail{Field: "meta.link_column_id", Issue: "must name a links column of this table"})
		}
		out["link_column_id"] = link.ID
		if rid, ok := link.Meta["relation_id"].(string); ok {
			out["relation_id"] = rid
		}
		targetID, _ := link.Meta["relation"].(map[string]interface{})["with"].(string)
		targetTable, err := a.tableByID(ctx, base, targetID)
		if err != nil || targetTable == nil {
			return nil, NewError(CodeValidationFailed, "The link points to a table that no longer exists.")
		}
		targetMetas, err := a.columnMetas(ctx, targetTable)
		if err != nil {
			return nil, err
		}
		lookupRef, _ := out["lookup_column_id"].(string)
		looked, err := pickMeta(targetMetas, lookupRef)
		if err != nil {
			return nil, NewError(CodeValidationFailed, "meta.lookup_column_id must name a column of the linked table.",
				ErrorDetail{Field: "meta.lookup_column_id", Issue: err.Error()})
		}
		out["lookup_column_id"] = looked.ID
	}
	return out, nil
}

// pickMeta finds a column among metas by name or fld_ ID
func pickMeta(metas []*columnMeta, ref string) (*columnMeta, error) {
	return pick(metas, ref, PrefixField, "column", func(m *columnMeta) string { return m.ID }, func(m *columnMeta) string { return m.Title })
}
