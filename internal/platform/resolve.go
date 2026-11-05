// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// workspaceTTL is how long a public ID lookup (found or not) is cached. Disabling the API or changing the
// schema reaches every instance within this time, without the app's services having to clear caches.
const workspaceTTL = 10 * time.Second

// workspaceRef is the workspace a request addresses
type workspaceRef struct {
	ID            string
	PublicID      string
	SchemaVersion int64
}

type workspaceEntry struct {
	ref     workspaceRef
	found   bool
	expires time.Time
}

// workspaceCache maps public IDs to workspaces, caching misses too so random IDs can't load the database
type workspaceCache struct {
	mu      sync.Mutex
	entries map[string]workspaceEntry
}

func (w *workspaceCache) get(publicID string, now time.Time) (workspaceEntry, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	e, ok := w.entries[publicID]
	if !ok || now.After(e.expires) {
		return workspaceEntry{}, false
	}
	return e, true
}

func (w *workspaceCache) put(publicID string, e workspaceEntry, now time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	// Drop expired entries now and then, so lookups of many unknown IDs don't grow the map forever
	if len(w.entries) >= 10000 {
		for k, old := range w.entries {
			if now.After(old.expires) {
				delete(w.entries, k)
			}
		}
	}
	w.entries[publicID] = e
}

// lookupWorkspace returns the enabled workspace with this public ID
func (a *API) lookupWorkspace(ctx context.Context, publicID string) (workspaceRef, bool, error) {
	now := a.now()
	if e, ok := a.workspaces.get(publicID, now); ok {
		return e.ref, e.found, nil
	}
	ref := workspaceRef{PublicID: publicID}
	err := a.db.QueryRow(fmt.Sprintf(`SELECT workspace_id, schema_version FROM %s.workspace_api WHERE public_id = $1 AND enabled = true`,
		a.schemaIdent()), publicID).Scan(&ref.ID, &ref.SchemaVersion)
	found := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return workspaceRef{}, false, err
	}
	a.workspaces.put(publicID, workspaceEntry{ref: ref, found: found, expires: now.Add(workspaceTTL)}, now)
	return ref, found, nil
}

// ---- Bases, tables and fields, cached per (workspace, schema_version) ----

type baseInfo struct {
	ID    string
	Title string
}

type tableInfo struct {
	ID     string
	BaseID string
	Title  string
	Alias  string // physical table name
}

// schemaSnapshot holds what has been read of one workspace's schema at one schema_version. It fills up as
// bases and tables are used. A rename bumps schema_version, so a new snapshot replaces this one.
type schemaSnapshot struct {
	mu         sync.Mutex
	version    int64
	bases      []*baseInfo
	tables     map[string][]*tableInfo // by base ID
	tablesByID map[string]*tableInfo
	fields     map[string][]*field   // by table ID
	specs      map[string]cachedSpec // OpenAPI specs, by base ID ("" for the whole workspace)
}

type schemaCache struct {
	mu          sync.Mutex
	byWorkspace map[string]*schemaSnapshot
}

func (s *schemaCache) snapshot(ws workspaceRef) *schemaSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap, ok := s.byWorkspace[ws.ID]
	if !ok || snap.version != ws.SchemaVersion {
		snap = &schemaSnapshot{
			version:    ws.SchemaVersion,
			tables:     map[string][]*tableInfo{},
			tablesByID: map[string]*tableInfo{},
			fields:     map[string][]*field{},
			specs:      map[string]cachedSpec{},
		}
		s.byWorkspace[ws.ID] = snap
	}
	return snap
}

func (a *API) bases(ctx context.Context, ws workspaceRef) ([]*baseInfo, error) {
	snap := a.schemas.snapshot(ws)
	snap.mu.Lock()
	defer snap.mu.Unlock()
	if snap.bases != nil {
		return snap.bases, nil
	}
	rows, err := a.db.QueryContext(ctx, fmt.Sprintf(
		`SELECT id::text, title FROM %s.bases WHERE workspace_id = $1 AND COALESCE(status, 'active') = 'active' ORDER BY created_time, id`,
		a.schemaIdent()), ws.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	bases := []*baseInfo{}
	for rows.Next() {
		b := &baseInfo{}
		if err := rows.Scan(&b.ID, &b.Title); err != nil {
			return nil, err
		}
		bases = append(bases, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	snap.bases = bases
	return bases, nil
}

const tableColumns = `id::text, base_id::text, title, alias`

func scanTables(rows *sql.Rows) ([]*tableInfo, error) {
	defer rows.Close()
	tables := []*tableInfo{}
	for rows.Next() {
		t := &tableInfo{}
		if err := rows.Scan(&t.ID, &t.BaseID, &t.Title, &t.Alias); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}
	return tables, rows.Err()
}

func (a *API) tables(ctx context.Context, ws workspaceRef, base *baseInfo) ([]*tableInfo, error) {
	snap := a.schemas.snapshot(ws)
	snap.mu.Lock()
	defer snap.mu.Unlock()
	if tables, ok := snap.tables[base.ID]; ok {
		return tables, nil
	}
	rows, err := a.db.QueryContext(ctx, fmt.Sprintf(
		`SELECT %s FROM %s.models WHERE base_id = $1 AND workspace_id = $2 AND COALESCE(deleted, false) = false
		 ORDER BY order_index NULLS LAST, created_time, id`, tableColumns, a.schemaIdent()), base.ID, ws.ID)
	if err != nil {
		return nil, err
	}
	tables, err := scanTables(rows)
	if err != nil {
		return nil, err
	}
	snap.tables[base.ID] = tables
	for _, t := range tables {
		snap.tablesByID[t.ID] = t
	}
	return tables, nil
}

// tableByID returns any table of the workspace, e.g. the table a link field points to; nil when there is none
func (a *API) tableByID(ctx context.Context, ws workspaceRef, id string) (*tableInfo, error) {
	snap := a.schemas.snapshot(ws)
	snap.mu.Lock()
	defer snap.mu.Unlock()
	if t, ok := snap.tablesByID[id]; ok {
		return t, nil
	}
	rows, err := a.db.QueryContext(ctx, fmt.Sprintf(
		`SELECT %s FROM %s.models WHERE id = $1 AND workspace_id = $2 AND COALESCE(deleted, false) = false`,
		tableColumns, a.schemaIdent()), id, ws.ID)
	if err != nil {
		return nil, err
	}
	tables, err := scanTables(rows)
	if err != nil || len(tables) == 0 {
		return nil, err
	}
	snap.tablesByID[id] = tables[0]
	return tables[0], nil
}

// fields returns the user fields of a table, in the table's field order
func (a *API) fields(ctx context.Context, ws workspaceRef, table *tableInfo) ([]*field, error) {
	snap := a.schemas.snapshot(ws)
	snap.mu.Lock()
	defer snap.mu.Unlock()
	if fields, ok := snap.fields[table.ID]; ok {
		return fields, nil
	}
	rows, err := a.db.QueryContext(ctx, fmt.Sprintf(
		`SELECT id::text, title, column_name, uidt, COALESCE(dt, ''), COALESCE(meta::text, '{}'), COALESCE(system, false),
		        COALESCE(order_index, 0)
		 FROM %s.columns WHERE model_id = $1 AND COALESCE(deleted, false) = false
		 ORDER BY order_index NULLS LAST, created_time, id`, a.schemaIdent()), table.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	fields := []*field{}
	for rows.Next() {
		var id, title, columnName, uidt, dt, rawMeta string
		var system bool
		var order float64
		if err := rows.Scan(&id, &title, &columnName, &uidt, &dt, &rawMeta, &system, &order); err != nil {
			return nil, err
		}
		meta := map[string]interface{}{}
		_ = json.Unmarshal([]byte(rawMeta), &meta)
		if f, ok := newField(id, title, columnName, uidt, dt, system, order, meta); ok {
			fields = append(fields, f)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	snap.fields[table.ID] = fields
	return fields, nil
}

// ---- Names or IDs ----

// pick finds the item a path segment or field reference means. A value with the ID prefix is an ID; anything
// else is an exact, case-sensitive name. A name used by several items is ambiguous; their IDs always work.
func pick[T any](items []T, value, prefix, kind string, id func(T) string, name func(T) string) (T, error) {
	var zero T
	if isUUIDID(prefix, value) {
		want, ok := DecodeUUID(prefix, value)
		if ok {
			for _, item := range items {
				if strings.EqualFold(id(item), want) {
					return item, nil
				}
			}
		}
		return zero, notFound(fmt.Sprintf("No %s with ID %q.", kind, value))
	}
	var matches []T
	for _, item := range items {
		if name(item) == value {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return zero, notFound(fmt.Sprintf("No %s named %q.", kind, value))
	case 1:
		return matches[0], nil
	}
	details := make([]ErrorDetail, len(matches))
	for i, m := range matches {
		details[i] = ErrorDetail{ID: EncodeUUID(prefix, id(m))}
	}
	return zero, NewError(CodeAmbiguousName,
		fmt.Sprintf("%d %ss are named %q. Use one of their IDs instead.", len(matches), kind, value), details...)
}

func (a *API) resolveBase(ctx context.Context, ws workspaceRef, value string) (*baseInfo, error) {
	bases, err := a.bases(ctx, ws)
	if err != nil {
		return nil, err
	}
	return pick(bases, value, PrefixBase, "base", func(b *baseInfo) string { return b.ID }, func(b *baseInfo) string { return b.Title })
}

func (a *API) resolveTable(ctx context.Context, ws workspaceRef, base *baseInfo, value string) (*tableInfo, error) {
	tables, err := a.tables(ctx, ws, base)
	if err != nil {
		return nil, err
	}
	return pick(tables, value, PrefixTable, "table", func(t *tableInfo) string { return t.ID }, func(t *tableInfo) string { return t.Title })
}

// resolveField finds a field of the table by name or fld_ ID
func resolveField(fields []*field, value string) (*field, error) {
	return pick(fields, value, PrefixField, "field", func(f *field) string { return f.ID }, func(f *field) string { return f.Name })
}

// fieldByID returns the field with this column ID, or nil
func fieldByID(fields []*field, id string) *field {
	for _, f := range fields {
		if strings.EqualFold(f.ID, id) {
			return f
		}
	}
	return nil
}
