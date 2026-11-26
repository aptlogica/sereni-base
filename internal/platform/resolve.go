// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// baseTTL is how long an unknown public ID is cached, so random IDs can't load the database. A found
// base is read on every request instead (one lookup by the unique public_id), so a schema change
// (schema_version) or turning the API off applies to the next request on every instance. A newly enabled
// base can still answer 404 for up to this time after a request found it missing.
const baseTTL = 10 * time.Second

// baseRef is the base a request addresses
type baseRef struct {
	ID            string
	WorkspaceID   string
	PublicID      string
	SchemaVersion int64
	Settings      baseSettings
	// Owner is the user who created the base; schema writes fall back to them when the user who turned
	// base management on is gone
	Owner string
}

type baseEntry struct {
	ref     baseRef
	found   bool
	expires time.Time
}

// baseCache maps public IDs to bases, caching misses too so random IDs can't load the database
type baseCache struct {
	mu      sync.Mutex
	entries map[string]baseEntry
}

func (w *baseCache) get(publicID string, now time.Time) (baseEntry, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	e, ok := w.entries[publicID]
	if !ok || now.After(e.expires) {
		return baseEntry{}, false
	}
	return e, true
}

func (w *baseCache) put(publicID string, e baseEntry, now time.Time) {
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

// lookupBase returns the enabled base with this public ID, with its current schema_version. Only
// misses are cached (see baseTTL).
func (a *API) lookupBase(ctx context.Context, publicID string) (baseRef, bool, error) {
	now := a.now()
	if e, ok := a.bases.get(publicID, now); ok && !e.found {
		return e.ref, false, nil
	}
	ref, found, err := a.readBase(ctx, publicID)
	if err != nil {
		return baseRef{}, false, err
	}
	if !found {
		a.bases.put(publicID, baseEntry{ref: ref, expires: now.Add(baseTTL)}, now)
	}
	return ref, found, nil
}

// readBase reads the enabled base with this public ID from the database
func (a *API) readBase(ctx context.Context, publicID string) (baseRef, bool, error) {
	ref := baseRef{PublicID: publicID}
	rows, err := a.db.QueryContext(ctx, fmt.Sprintf(`SELECT b.id::text, b.workspace_id::text, a.schema_version, a.settings::text, COALESCE(b.created_by::text, '') FROM %[1]s.base_api a JOIN %[1]s.bases b ON b.id = a.base_id
		 WHERE a.public_id = $1 AND a.enabled = true AND COALESCE(b.status, 'active') = 'active'`,
		a.schemaIdent()), publicID)
	if err != nil {
		return ref, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return ref, false, rows.Err()
	}
	var rawSettings []byte
	if err := rows.Scan(&ref.ID, &ref.WorkspaceID, &ref.SchemaVersion, &rawSettings, &ref.Owner); err != nil {
		return ref, false, err
	}
	ref.Settings = parseBaseSettings(rawSettings)
	return ref, true, rows.Err()
}

// ---- Tables and fields, cached per (base, schema_version) ----

type tableInfo struct {
	ID     string
	BaseID string
	Title  string
	Alias  string // physical table name
}

// schemaSnapshot holds what has been read of one base's schema at one schema_version. It fills up as
// tables are used. A rename bumps schema_version, so a new snapshot replaces this one.
type schemaSnapshot struct {
	mu         sync.Mutex
	version    int64
	tables     map[string][]*tableInfo // by base ID
	tablesByID map[string]*tableInfo
	fields     map[string][]*field // by table ID
	spec       *cachedSpec         // the OpenAPI spec
}

type schemaCache struct {
	mu     sync.Mutex
	byBase map[string]*schemaSnapshot
}

func (s *schemaCache) snapshot(base baseRef) *schemaSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap, ok := s.byBase[base.ID]
	if !ok || snap.version != base.SchemaVersion {
		snap = &schemaSnapshot{
			version:    base.SchemaVersion,
			tables:     map[string][]*tableInfo{},
			tablesByID: map[string]*tableInfo{},
			fields:     map[string][]*field{},
		}
		s.byBase[base.ID] = snap
	}
	return snap
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

func (a *API) tables(ctx context.Context, base baseRef) ([]*tableInfo, error) {
	snap := a.schemas.snapshot(base)
	snap.mu.Lock()
	defer snap.mu.Unlock()
	if tables, ok := snap.tables[base.ID]; ok {
		return tables, nil
	}
	rows, err := a.db.QueryContext(ctx, fmt.Sprintf(
		`SELECT %s FROM %s.models WHERE base_id = $1 AND COALESCE(deleted, false) = false
		 ORDER BY order_index NULLS LAST, created_time, id`, tableColumns, a.schemaIdent()), base.ID)
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

// tableByID returns a table of the base, e.g. the table a link field points to; nil when there is none. Tables of other
// bases are not resolved: links stay within a base
func (a *API) tableByID(ctx context.Context, base baseRef, id string) (*tableInfo, error) {
	snap := a.schemas.snapshot(base)
	snap.mu.Lock()
	defer snap.mu.Unlock()
	if t, ok := snap.tablesByID[id]; ok {
		return t, nil
	}
	rows, err := a.db.QueryContext(ctx, fmt.Sprintf(
		`SELECT %s FROM %s.models WHERE id = $1 AND base_id = $2 AND COALESCE(deleted, false) = false`,
		tableColumns, a.schemaIdent()), id, base.ID)
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
func (a *API) fields(ctx context.Context, base baseRef, table *tableInfo) ([]*field, error) {
	snap := a.schemas.snapshot(base)
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
	var formulas *hiddenFormulas
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
		} else if strings.EqualFold(uidt, typeFormula) {
			if formulas == nil {
				formulas = &hiddenFormulas{byID: map[string]string{}, byName: map[string]bool{}}
			}
			formulas.byID[strings.ToLower(id)] = title
			formulas.byName[title] = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, f := range fields {
		f.formulas = formulas
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

func (a *API) resolveTable(ctx context.Context, base baseRef, value string) (*tableInfo, error) {
	tables, err := a.tables(ctx, base)
	if err != nil {
		return nil, err
	}
	return pick(tables, value, PrefixTable, "table", func(t *tableInfo) string { return t.ID }, func(t *tableInfo) string { return t.Title })
}

// resolveField finds a field of the table by name or fld_ ID
func resolveField(fields []*field, value string) (*field, error) {
	f, err := pick(fields, value, PrefixField, "field", func(f *field) string { return f.ID }, func(f *field) string { return f.Name })
	if apiErr, ok := err.(*Error); ok && apiErr.Code == CodeNotFound {
		if name, isFormula := hiddenFormula(fields, value); isFormula {
			return nil, formulaFieldError(name)
		}
	}
	return f, err
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
