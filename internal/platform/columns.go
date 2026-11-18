// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

// Constraint types the columns endpoint reports, by pg_constraint.contype. NOT NULL (contype n, Postgres 18)
// is reported per column as db_nullable instead.
var constraintTypeNames = map[string]string{
	"p": "primary_key",
	"u": "unique",
	"c": "check",
	"f": "foreign_key",
	"x": "exclusion",
}

// columnRefOut names a column of a constraint or of the primary key
type columnRefOut struct {
	ID         string `json:"id,omitempty"` // fld_ ID; empty for a column the app has no metadata for
	Name       string `json:"name"`
	ColumnName string `json:"column_name"`
}

// constraintOut is a constraint of the table as Postgres has it, with the title the app gave it
type constraintOut struct {
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Title      string         `json:"title,omitempty"` // set for constraints made in the app
	Columns    []columnRefOut `json:"columns"`
	Definition string         `json:"definition"`
}

// columnOut is one column of the table: the app's metadata, the physical column and its constraints
type columnOut struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ColumnName  string `json:"column_name"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type"` // the API type, as in GET /tables/{table}
	UIDT        string `json:"uidt"` // the app's type
	DT          string `json:"dt,omitempty"`
	// The physical column; null for columns without one (lookups, formulas, buttons)
	DBType     *string `json:"db_type"`
	DBNullable *bool   `json:"db_nullable"`
	DBDefault  *string `json:"db_default"`
	DBIdentity bool    `json:"db_identity"`

	PrimaryKey         bool     `json:"primary_key"`
	PrimaryKeyPosition *int     `json:"primary_key_position,omitempty"` // 1-based, in the key's order
	Unique             bool     `json:"unique"`                         // alone, it is the primary key or a unique key
	Constraints        []string `json:"constraints"`                    // names of the constraints it is part of

	System   bool `json:"system"`
	Virtual  bool `json:"virtual"`
	ReadOnly bool `json:"read_only"`
	// Whether the column is a field of a record's fields object (system columns, formulas and buttons are not)
	RecordField bool                   `json:"record_field"`
	Order       float64                `json:"order"`
	Meta        map[string]interface{} `json:"meta"`
}

type tableColumnsOut struct {
	TableID     string          `json:"table_id"`
	TableName   string          `json:"table_name"`
	BaseID      string          `json:"base_id"`
	PrimaryKey  []columnRefOut  `json:"primary_key"`
	Constraints []constraintOut `json:"constraints"`
	Columns     []columnOut     `json:"columns"`
}

// columnMeta is a row of the columns metadata table
type columnMeta struct {
	ID, Title, ColumnName, UIDT, DT, Description string
	System, Virtual                              bool
	Order                                        float64
	Meta                                         map[string]interface{}
}

// physicalColumn is a column of the records table, from pg_attribute
type physicalColumn struct {
	Type     string
	NotNull  bool
	Default  string
	Identity bool
}

// pgConstraint is a constraint of the records table, from pg_constraint
type pgConstraint struct {
	Name, Type, Definition string
	Columns                []string
}

// GET /tables/{table}/columns: every column of the table with its metadata, its database type,
// nullability and default, whether it is in the primary key, and the constraints it is part of. Read fresh on
// every request: constraint changes don't change the base's schema version.
func (a *API) listColumns(c *gin.Context) {
	base := currentBase(c)
	table, err := a.resolveTable(c, base, c.Param("table"))
	if err != nil {
		respondError(c, err)
		return
	}
	out, err := a.tableColumns(c, base, table)
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, out)
}

func (a *API) tableColumns(ctx context.Context, base baseRef, table *tableInfo) (*tableColumnsOut, error) {
	metas, err := a.columnMetas(ctx, table)
	if err != nil {
		return nil, err
	}
	physical, err := a.physicalColumns(ctx, table)
	if err != nil {
		return nil, err
	}
	constraints, err := a.pgConstraints(ctx, table)
	if err != nil {
		return nil, err
	}
	titles, err := a.constraintTitles(ctx, table)
	if err != nil {
		return nil, err
	}

	byColumn := make(map[string]*columnMeta, len(metas))
	for _, m := range metas {
		if m.ColumnName != "" {
			byColumn[m.ColumnName] = m
		}
	}
	ref := func(col string) columnRefOut {
		if m, ok := byColumn[col]; ok {
			return columnRefOut{ID: EncodeUUID(PrefixField, m.ID), Name: m.Title, ColumnName: col}
		}
		return columnRefOut{Name: col, ColumnName: col}
	}

	out := &tableColumnsOut{
		TableID:     EncodeUUID(PrefixTable, table.ID),
		TableName:   table.Title,
		BaseID:      EncodeUUID(PrefixBase, base.ID),
		PrimaryKey:  []columnRefOut{},
		Constraints: []constraintOut{},
		Columns:     []columnOut{},
	}
	keyPosition := map[string]int{}   // column -> 1-based position in the primary key
	uniqueAlone := map[string]bool{}  // column -> alone in a primary or unique key
	memberOf := map[string][]string{} // column -> names of its constraints
	for _, con := range constraints {
		co := constraintOut{Name: con.Name, Type: constraintTypeNames[con.Type], Title: titles[con.Name],
			Columns: make([]columnRefOut, len(con.Columns)), Definition: con.Definition}
		for i, col := range con.Columns {
			co.Columns[i] = ref(col)
			memberOf[col] = append(memberOf[col], con.Name)
			if con.Type == "p" {
				keyPosition[col] = i + 1
				out.PrimaryKey = append(out.PrimaryKey, co.Columns[i])
			}
		}
		if (con.Type == "p" || con.Type == "u") && len(con.Columns) == 1 {
			uniqueAlone[con.Columns[0]] = true
		}
		out.Constraints = append(out.Constraints, co)
	}

	for _, m := range metas {
		out.Columns = append(out.Columns, columnInfo(m, physical, keyPosition, uniqueAlone, memberOf))
	}
	return out, nil
}

// columnInfo builds one column of the response
func columnInfo(m *columnMeta, physical map[string]physicalColumn, keyPosition map[string]int,
	uniqueAlone map[string]bool, memberOf map[string][]string) columnOut {
	co := columnOut{
		ID: EncodeUUID(PrefixField, m.ID), Name: m.Title, ColumnName: m.ColumnName, Description: m.Description,
		UIDT: m.UIDT, DT: m.DT, System: m.System, Virtual: m.Virtual, Order: m.Order, Meta: m.Meta,
		Constraints: []string{},
	}
	if m.ColumnName == "" {
		co.ReadOnly = true
		co.Type = (&field{Type: strings.ToLower(m.UIDT)}).APIType()
		return co
	}
	co.Unique = uniqueAlone[m.ColumnName]
	if names := memberOf[m.ColumnName]; names != nil {
		co.Constraints = names
	}
	if f, ok := newField(m.ID, m.Title, m.ColumnName, m.UIDT, m.DT, m.System, m.Order, m.Meta); ok {
		co.Type, co.ReadOnly, co.RecordField = f.APIType(), f.ReadOnly, true
	} else {
		co.Type, co.ReadOnly = (&field{Type: strings.ToLower(m.UIDT)}).APIType(), true
	}
	if pos, ok := keyPosition[m.ColumnName]; ok {
		co.PrimaryKey, co.PrimaryKeyPosition = true, &pos
	}
	if p, ok := physical[m.ColumnName]; ok {
		dbType, nullable := p.Type, !p.NotNull
		co.DBType, co.DBNullable, co.DBIdentity = &dbType, &nullable, p.Identity
		if p.Default != "" {
			def := p.Default
			co.DBDefault = &def
		}
	}
	return co
}

// columnMetas reads every column of the table from the metadata, system columns and formulas included
func (a *API) columnMetas(ctx context.Context, table *tableInfo) ([]*columnMeta, error) {
	rows, err := a.db.QueryContext(ctx, fmt.Sprintf(
		`SELECT id::text, title, COALESCE(column_name, ''), uidt, COALESCE(dt, ''), COALESCE(description, ''),
		        COALESCE(meta::text, '{}'), COALESCE(system, false), COALESCE(virtual, false), COALESCE(order_index, 0)
		 FROM %s.columns WHERE model_id = $1 AND COALESCE(deleted, false) = false
		 ORDER BY order_index NULLS LAST, created_time, id`, a.schemaIdent()), table.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*columnMeta{}
	for rows.Next() {
		m := &columnMeta{}
		var rawMeta string
		if err := rows.Scan(&m.ID, &m.Title, &m.ColumnName, &m.UIDT, &m.DT, &m.Description, &rawMeta,
			&m.System, &m.Virtual, &m.Order); err != nil {
			return nil, err
		}
		m.Meta = map[string]interface{}{}
		_ = json.Unmarshal([]byte(rawMeta), &m.Meta)
		out = append(out, m)
	}
	return out, rows.Err()
}

// physicalColumns reads the records table's columns from the catalog, by name
func (a *API) physicalColumns(ctx context.Context, table *tableInfo) (map[string]physicalColumn, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT a.attname::text, format_type(a.atttypid, a.atttypmod), a.attnotnull,
		       COALESCE(pg_get_expr(d.adbin, d.adrelid), ''), a.attidentity::text <> ''
		FROM pg_attribute a
		LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
		WHERE a.attrelid = $1::regclass AND a.attnum > 0 AND NOT a.attisdropped
		ORDER BY a.attnum`, a.recordsTable(table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]physicalColumn{}
	for rows.Next() {
		var name string
		var p physicalColumn
		if err := rows.Scan(&name, &p.Type, &p.NotNull, &p.Default, &p.Identity); err != nil {
			return nil, err
		}
		out[name] = p
	}
	return out, rows.Err()
}

// pgConstraints reads the records table's primary key, unique, check, foreign key and exclusion constraints,
// each with its columns in key order
func (a *API) pgConstraints(ctx context.Context, table *tableInfo) ([]pgConstraint, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT c.conname::text, c.contype::text, pg_get_constraintdef(c.oid),
		       COALESCE(array_agg(a.attname::text ORDER BY k.pos) FILTER (WHERE a.attname IS NOT NULL), '{}')
		FROM pg_constraint c
		LEFT JOIN LATERAL unnest(c.conkey) WITH ORDINALITY AS k(attnum, pos) ON true
		LEFT JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = k.attnum
		WHERE c.conrelid = $1::regclass AND c.contype IN ('p', 'u', 'c', 'f', 'x')
		GROUP BY c.oid, c.conname, c.contype
		ORDER BY CASE c.contype WHEN 'p' THEN 0 WHEN 'u' THEN 1 WHEN 'c' THEN 2 WHEN 'f' THEN 3 ELSE 4 END, c.conname`,
		a.recordsTable(table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []pgConstraint{}
	for rows.Next() {
		var con pgConstraint
		var cols pq.StringArray
		if err := rows.Scan(&con.Name, &con.Type, &con.Definition, &cols); err != nil {
			return nil, err
		}
		con.Columns = []string(cols)
		out = append(out, con)
	}
	return out, rows.Err()
}

// constraintTitles returns the titles of the constraints made in the app, by Postgres name
func (a *API) constraintTitles(ctx context.Context, table *tableInfo) (map[string]string, error) {
	rows, err := a.db.QueryContext(ctx, fmt.Sprintf(
		`SELECT constraint_name, title FROM %s.table_constraints WHERE model_id = $1`, a.schemaIdent()), table.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var name, title string
		if err := rows.Scan(&name, &title); err != nil {
			return nil, err
		}
		out[name] = title
	}
	return out, rows.Err()
}
