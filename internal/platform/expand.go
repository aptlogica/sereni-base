// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"context"
	"fmt"

	"github.com/lib/pq"
)

// expand=Customer,Customer.Company returns the linked records' fields inside the link objects, instead of
// only their IDs. Each expanded link field costs one query per page, however many records the page has.
const (
	maxExpandDepth     = 2   // Customer.Company, not Customer.Company.Owner
	maxExpandFields    = 5   // link fields expanded in one request, all levels together
	maxExpandPerRecord = 100 // linked records expanded per link cell; the rest keep only their id
	expandKey          = "fields"
)

// expandSet is the link fields to expand, each with what to expand inside its linked records
type expandSet map[*field]*expandNode

type expandNode struct {
	target   *tableContext // the linked table
	children expandSet
}

// parseExpand reads expand: link field names or IDs, comma-separated, with a dot for the next level
func (a *API) parseExpand(ctx context.Context, tc *tableContext, values ...string) (expandSet, error) {
	var paths []string
	for _, v := range values {
		paths = append(paths, splitList(v)...)
	}
	if len(paths) == 0 {
		return nil, nil
	}
	set := expandSet{}
	count := 0
	for _, path := range paths {
		if err := a.addExpand(ctx, tc.ws, set, tc.fields, path, path, 1, &count); err != nil {
			return nil, err
		}
	}
	return set, nil
}

// addExpand adds one path to set. fields are the fields of the table the path starts in; full is the path as
// the client sent it, for errors.
func (a *API) addExpand(ctx context.Context, ws workspaceRef, set expandSet, fields []*field, path, full string, depth int, count *int) error {
	if depth > maxExpandDepth {
		return badRequest(fmt.Sprintf("expand: %q goes deeper than %d levels. Read the linked records' own links with another request.",
			full, maxExpandDepth))
	}
	f, rest, err := splitExpandPath(fields, path)
	if err != nil {
		return fieldLookupError(err, "expand", fields, path)
	}
	if f.Type != typeLinks {
		return badRequest(fmt.Sprintf("expand: %q is a %s field; only link fields can be expanded.", f.Name, f.APIType()))
	}
	node := set[f]
	if node == nil {
		if *count++; *count > maxExpandFields {
			return badRequest(fmt.Sprintf("expand: at most %d link fields in one request.", maxExpandFields))
		}
		target, err := a.linkedTable(ctx, ws, f)
		if err != nil {
			return err
		}
		if target == nil {
			return badRequest(fmt.Sprintf("expand: the linked table of %q no longer exists.", f.Name))
		}
		node = &expandNode{target: target, children: expandSet{}}
		set[f] = node
	}
	if rest == "" {
		return nil
	}
	return a.addExpand(ctx, ws, node.children, node.target.fields, rest, full, depth+1, count)
}

// splitExpandPath finds the field a path starts with: the whole path when it names a field (names may hold
// dots), else the longest part before a dot that does. rest is what follows that field's dot.
func splitExpandPath(fields []*field, path string) (*field, string, error) {
	f, err := resolveField(fields, path)
	if err == nil {
		return f, "", nil
	}
	for i := len(path) - 1; i > 0; i-- {
		if path[i] != '.' {
			continue
		}
		if f, prefixErr := resolveField(fields, path[:i]); prefixErr == nil {
			return f, path[i+1:], nil
		}
	}
	return nil, "", err
}

// linkedTable returns the table a link field points to, with its fields; nil when it no longer exists
func (a *API) linkedTable(ctx context.Context, ws workspaceRef, link *field) (*tableContext, error) {
	if link.LinkTargetModelID == "" {
		return nil, nil
	}
	table, err := a.tableByID(ctx, ws, link.LinkTargetModelID)
	if err != nil || table == nil {
		return nil, err
	}
	fields, err := a.fields(ctx, ws, table)
	if err != nil {
		return nil, err
	}
	return &tableContext{ws: ws, table: table, fields: fields}, nil
}

// expandedRecords reads the linked records of one expanded link field with one query and returns their fields
// by record ID: the first maxExpandPerRecord linked records of each cell. The node's own expand applies inside.
func (a *API) expandedRecords(ctx context.Context, node *expandNode, linkIDs [][]int64, keysByID bool) (map[int64]orderedFields, error) {
	var ids []int64
	seen := map[int64]bool{}
	for _, list := range linkIDs {
		for i, id := range list {
			if i >= maxExpandPerRecord {
				break
			}
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	out := map[int64]orderedFields{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := a.queryRows(ctx, fmt.Sprintf(`SELECT row_to_json(x)::text FROM %s AS x WHERE x.id = ANY($1)`,
		a.recordsTable(node.target.table)), pq.Array(ids))
	if err != nil {
		return nil, err
	}
	records, err := a.buildRecords(ctx, node.target, rows, node.target.fields, keysByID, node.children)
	if err != nil {
		return nil, err
	}
	for i, row := range rows {
		out[rowID(row)] = records[i].Fields
	}
	return out, nil
}

// withExpanded returns the fields to return plus any expanded link field that select left out, in the
// table's field order, so an expanded field is always in the answer
func withExpanded(selected, all []*field, expand expandSet) []*field {
	if len(expand) == 0 {
		return selected
	}
	has := map[*field]bool{}
	for _, f := range selected {
		has[f] = true
	}
	out := selected
	for _, f := range all {
		if expand[f] != nil && !has[f] {
			out = append(out[:len(out):len(out)], f)
		}
	}
	return out
}
