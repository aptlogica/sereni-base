// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// columnWriteOut is a column after a schema write
type columnWriteOut struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Type        string                 `json:"type"` // the API type, as in GET /tables/{table}
	UIDT        string                 `json:"uidt"`
	DT          string                 `json:"dt,omitempty"`
	Meta        map[string]interface{} `json:"meta,omitempty"`
}

func columnWriteOutOf(c dto.ColumnResponse) columnWriteOut {
	return columnWriteOut{
		ID: EncodeUUID(PrefixField, c.ID.String()), Name: c.Title, Description: c.Description,
		Type: (&field{Type: strings.ToLower(c.UIDT)}).APIType(), UIDT: c.UIDT, DT: c.DT, Meta: c.Meta,
	}
}

func systemColumnError(name string) error {
	return NewError(CodeSystemColumn, fmt.Sprintf("%q is a system column and can't be changed or deleted.", name))
}

func isSystemMeta(m *columnMeta) bool {
	return m.System && systemColumns[m.ColumnName]
}

// keyBlockedBy says why a column can't be in a unique key, or "" when it can
func keyBlockedBy(m *columnMeta) string {
	return tenant.KeyColumnBlockReason(tenant.Column{ColumnName: m.ColumnName, UIDT: m.UIDT, System: m.System, Meta: m.Meta})
}

// applyConstraintMeta puts the default and id generation of a request into the column's meta, which is where
// the app keeps them
func applyConstraintMeta(uidt string, meta map[string]interface{}, cons columnConstraints) ([]ErrorDetail, map[string]interface{}) {
	var details []ErrorDetail
	out := map[string]interface{}{}
	for k, v := range meta {
		out[k] = v
	}
	if cons.IDGeneration != nil {
		if uidt != constant.UIDTUUID {
			details = append(details, ErrorDetail{Field: "meta.id_generation", Issue: "only uuid columns can generate IDs"})
		} else {
			cfg := map[string]interface{}{"mode": cons.IDGeneration.Mode}
			if cons.IDGeneration.Version != "" {
				cfg["version"] = cons.IDGeneration.Version
			}
			out[constant.MetaIDGenerationKey] = cfg
		}
	}
	if cons.hasDefault {
		if cons.Default == nil {
			delete(out, metaDefaultKey)
		} else {
			out[metaDefaultKey] = cons.Default
		}
	}
	return details, out
}

// metaDefaultKey is the column meta key holding the default value
const metaDefaultKey = "defaultValue"

// addColumn creates one column with its optional constraints. If a constraint fails the column is removed
// again. The error is an API error that names what is wrong.
func (a *API) addColumn(ctx context.Context, base baseRef, table *tableInfo, in *columnInput, actor string) (columnWriteOut, error) {
	metas, err := a.columnMetas(ctx, table)
	if err != nil {
		return columnWriteOut{}, err
	}
	for _, m := range metas {
		if m.Title == *in.Title {
			return columnWriteOut{}, NewError(CodeConflict, fmt.Sprintf("This table already has a column named %q.", *in.Title))
		}
	}
	if linkOrLookup(in.UIDT) && in.Constraints.present {
		return columnWriteOut{}, NewError(CodeValidationFailed, "Links and lookups take no constraints.",
			ErrorDetail{Field: "constraints", Issue: "not allowed for " + in.UIDT + " columns"})
	}
	meta, err := a.resolveTypeMeta(ctx, base, table, in.UIDT, in.Meta, metas)
	if err != nil {
		return columnWriteOut{}, err
	}
	if details, m := applyConstraintMeta(in.UIDT, meta, in.Constraints); len(details) > 0 {
		return columnWriteOut{}, validationFailed(details)
	} else {
		meta = m
	}
	if in.Constraints.Unique != nil && *in.Constraints.Unique {
		if why := keyBlockedBy(&columnMeta{UIDT: in.UIDT, Meta: meta}); why != "" {
			return columnWriteOut{}, NewError(CodeValidationFailed, "This column can't be unique.",
				ErrorDetail{Field: "constraints.unique", Issue: why})
		}
	}
	if in.Constraints.hasDefault && in.Constraints.Default != nil && in.Constraints.Unique != nil && *in.Constraints.Unique {
		return columnWriteOut{}, NewError(CodeValidationFailed, tenant.UniqueDefaultConflictMessage,
			ErrorDetail{Field: "meta.default", Issue: "can't be combined with unique"})
	}
	if in.Constraints.hasDefault && in.Constraints.Default != nil && in.Constraints.Check != nil {
		return columnWriteOut{}, NewError(CodeValidationFailed, tenant.CheckDefaultConflictMessage,
			ErrorDetail{Field: "meta.default", Issue: "can't be combined with check"})
	}

	modelID, _ := uuid.Parse(table.ID)
	baseID, _ := uuid.Parse(base.ID)
	desc := ""
	if in.Description != nil {
		desc = *in.Description
	}
	// Columns made through the API are ordinary columns, never system ones
	notSystem := false
	// The new column goes last, as in the app; without a position it could not be reordered or deleted cleanly
	next := 1.0
	for _, m := range metas {
		if m.Order >= next {
			next = m.Order + 1
		}
	}
	created, err := a.schemaSvc.Tables.AddColumn(ctx, a.schema, dto.AddColumnRequest{
		ModelID: modelID, BaseID: baseID, Title: *in.Title, Description: desc, Meta: meta, UIDT: in.UIDT, CreatedBy: actor,
		System: &notSystem, OrderIndex: &next,
	})
	if err != nil {
		return columnWriteOut{}, guardedError(err)
	}
	if err := a.applyNewColumnConstraints(ctx, table, created, in.Constraints, actor); err != nil {
		if delErr := a.schemaSvc.Tables.DeleteColumn(ctx, a.schema, created.ID.String()); delErr != nil {
			return columnWriteOut{}, NewError(CodeInternalError, "The column was created but a constraint failed, and the column could not be removed again; delete it by hand.")
		}
		return columnWriteOut{}, err
	}
	return columnWriteOutOf(created), nil
}

// applyNewColumnConstraints adds the unique and check constraints of a request for a column that exists
func (a *API) applyNewColumnConstraints(ctx context.Context, table *tableInfo, col dto.ColumnResponse, cons columnConstraints, actor string) error {
	if cons.Unique != nil && *cons.Unique {
		if _, err := a.schemaSvc.Constraints.Create(ctx, a.schema, dto.CreateTableConstraintRequest{
			ModelID: table.ID, Type: "unique", Title: col.Title + " is unique", ColumnIDs: []string{col.ID.String()}, CreatedBy: actor,
		}); err != nil {
			return prefixDetails(constraintError(err), "constraints.unique")
		}
	}
	if cons.Check != nil {
		metas, err := a.columnMetas(ctx, table)
		if err != nil {
			return err
		}
		if err := onlyKeys(cons.Check, "constraints.check", "category", "where"); err != nil {
			return err
		}
		where, ok := cons.Check["where"].(map[string]interface{})
		if !ok {
			return NewError(CodeValidationFailed, "The check needs a condition.", ErrorDetail{Field: "constraints.check.where", Issue: "is required"})
		}
		sql, used, err := compileCondition(where, metas)
		if err != nil {
			return prefixDetails(err, "constraints.check")
		}
		for _, u := range used {
			if u.ID != col.ID.String() {
				return NewError(CodeValidationFailed, "A column's check can only use that column. Use POST /tables/{table}/constraints for conditions over several columns.",
					ErrorDetail{Field: "constraints.check.where", Issue: fmt.Sprintf("uses %q", u.Title)})
			}
		}
		category, _ := cons.Check["category"].(string)
		if category == "" {
			category = "custom"
		}
		if _, err := a.schemaSvc.Constraints.Create(ctx, a.schema, dto.CreateTableConstraintRequest{
			ModelID: table.ID, Type: "check", Title: col.Title + " check", ColumnIDs: []string{col.ID.String()},
			CheckInput: dto.CheckInput{Category: category, Condition: sql, Where: where}, CreatedBy: actor,
		}); err != nil {
			return prefixDetails(constraintError(err), "constraints.check")
		}
	}
	return nil
}

// POST /tables/{table}/columns
func (a *API) createColumn(c *gin.Context) {
	body, err := readBody(c)
	if err != nil {
		respondError(c, err)
		return
	}
	in, details, err := parseColumnInput(body, "body", true)
	if err != nil {
		respondError(c, err)
		return
	}
	if len(details) > 0 {
		respondError(c, validationFailed(details))
		return
	}
	tc, err := a.resolveTablePath(c)
	if err != nil {
		respondError(c, err)
		return
	}
	out, err := a.addColumn(c, tc.base, tc.table, in, a.actor(c, tc.base))
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusCreated, out)
}

// allowedTargets lists the types a column of this type can become
func allowedTargets(from string) []string {
	out := append([]string(nil), constant.AllowedConversions[from]...)
	sort.Strings(out)
	return out
}

var numericTargets = map[string]bool{"number": true, "decimal": true, "currency": true, "percent": true, "rating": true, "year": true}

// conversionPreview counts what a type change would do to the column's values: filled values, and for numeric
// targets the values that are not numbers (they become empty)
func (a *API) conversionPreview(ctx context.Context, table *tableInfo, m *columnMeta, target string) map[string]any {
	preview := map[string]any{"from": m.UIDT, "to": target}
	col := pq.QuoteIdentifier(m.ColumnName)
	tbl := a.recordsTable(table)
	var filled int64
	if err := a.db.QueryRow(fmt.Sprintf(`SELECT count(*) FROM %s WHERE %s IS NOT NULL AND %s::text <> ''`, tbl, col, col)).Scan(&filled); err == nil {
		preview["filled_values"] = filled
	}
	if numericTargets[target] {
		var lost int64
		if err := a.db.QueryRow(fmt.Sprintf(
			`SELECT count(*) FROM %s WHERE %s IS NOT NULL AND %s::text <> '' AND %s::text !~ '^\s*-?[0-9]+(\.[0-9]+)?\s*$'`,
			tbl, col, col, col)).Scan(&lost); err == nil {
			preview["values_lost"] = lost
			preview["exact"] = true
			return preview
		}
	}
	// For other targets the server can't tell without converting: filled_values is the most that can be lost
	preview["exact"] = false
	return preview
}

// PATCH /tables/{table}/columns/{column}
func (a *API) updateColumn(c *gin.Context) {
	body, err := readBody(c)
	if err != nil {
		respondError(c, err)
		return
	}
	in, details, err := parseColumnInput(body, "body", false)
	if err != nil {
		respondError(c, err)
		return
	}
	if len(details) > 0 {
		respondError(c, validationFailed(details))
		return
	}
	tc, err := a.resolveTablePath(c)
	if err != nil {
		respondError(c, err)
		return
	}
	metas, err := a.columnMetas(c, tc.table)
	if err != nil {
		respondError(c, err)
		return
	}
	m, err := pickMeta(metas, c.Param("column"))
	if err != nil {
		respondError(c, err)
		return
	}
	if isSystemMeta(m) {
		respondError(c, systemColumnError(m.Title))
		return
	}
	actor := a.actor(c, tc.base)
	dry := c.Query("dry_run") == "true"

	req := dto.ColumnUpdate{UpdatedBy: actor}
	if in.Title != nil && *in.Title != m.Title {
		for _, other := range metas {
			if other.Title == *in.Title {
				respondError(c, NewError(CodeConflict, fmt.Sprintf("This table already has a column named %q.", *in.Title)))
				return
			}
		}
		req.Title = in.Title
	}
	if in.Description != nil {
		req.Description = in.Description
	}

	// Type change: as in the app, the target must be allowed for the current type
	typeChange := in.UIDT != "" && in.UIDT != m.UIDT
	if typeChange {
		if linkOrLookup(m.UIDT) || linkOrLookup(in.UIDT) {
			respondError(c, NewError(CodeConversionNotAllowed, "Links and lookups can't change type; delete the column and add a new one."))
			return
		}
		allowed := allowedTargets(m.UIDT)
		ok := false
		for _, t := range allowed {
			ok = ok || t == in.UIDT
		}
		if !ok {
			details := make([]ErrorDetail, len(allowed))
			for i, t := range allowed {
				details[i] = ErrorDetail{Field: "uidt", Issue: "allowed: " + t}
			}
			respondError(c, NewError(CodeConversionNotAllowed,
				fmt.Sprintf("A %s column can't be changed to %s.", m.UIDT, in.UIDT), details...))
			return
		}
		if dry {
			respond(c, http.StatusOK, a.conversionPreview(c, tc.table, m, strings.ToLower(in.UIDT)))
			return
		}
		mapping := constant.UITypeMappings[in.UIDT]
		dt := mapping.Postgres
		req.UIDT, req.DT = &in.UIDT, &dt
	} else if dry {
		respondError(c, badRequest("dry_run is for type changes: send uidt with a different type."))
		return
	}
	if in.EmptyFirst && !typeChange {
		respondError(c, badRequest("empty_first goes with a type change."))
		return
	}

	// Settings: merged into the column's meta; default and id generation are constraints in the API
	if in.hasMeta || in.Constraints.present {
		if linkOrLookup(m.UIDT) && in.Constraints.present {
			respondError(c, NewError(CodeValidationFailed, "Links and lookups take no constraints."))
			return
		}
		meta := map[string]interface{}{}
		for k, v := range m.Meta {
			meta[k] = v
		}
		for k, v := range in.Meta {
			meta[k] = v
		}
		uidt := m.UIDT
		if typeChange {
			uidt = in.UIDT
		}
		if d, merged := applyConstraintMeta(uidt, meta, in.Constraints); len(d) > 0 {
			respondError(c, validationFailed(d))
			return
		} else {
			meta = merged
		}
		req.Meta = &meta
		if in.Constraints.IDGeneration != nil && in.Constraints.IDGeneration.Backfill {
			yes := true
			req.Backfill = &yes
		}
	}
	if in.Constraints.Unique != nil && *in.Constraints.Unique {
		if why := keyBlockedBy(m); why != "" {
			respondError(c, NewError(CodeValidationFailed, "This column can't be unique.", ErrorDetail{Field: "constraints.unique", Issue: why}))
			return
		}
	}

	if in.EmptyFirst {
		if err := a.schemaSvc.Tables.ResetColumnValues(c, a.schema, tc.table.ID, m.ID); err != nil {
			respondError(c, guardedError(err))
			return
		}
	}
	updated, err := a.schemaSvc.Tables.UpdateColumn(c, a.schema, m.ID, req)
	if err != nil {
		respondError(c, guardedError(err))
		return
	}
	if err := a.changeColumnConstraints(c, tc.table, updated, in.Constraints, actor); err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, columnWriteOutOf(updated))
}

// changeColumnConstraints adds or drops the unique and check constraints of a column change
func (a *API) changeColumnConstraints(ctx context.Context, table *tableInfo, col dto.ColumnResponse, cons columnConstraints, actor string) error {
	if cons.Unique != nil && !*cons.Unique {
		list, err := a.schemaSvc.Constraints.List(ctx, a.schema, table.ID, "unique")
		if err != nil {
			return constraintError(err)
		}
		for _, item := range list {
			if len(item.Columns) == 1 && strings.EqualFold(item.Columns[0].ID, col.ID.String()) {
				if err := a.schemaSvc.Constraints.Delete(ctx, a.schema, item.ID); err != nil {
					return constraintError(err)
				}
			}
		}
		cons.Unique = nil
	}
	return a.applyNewColumnConstraints(ctx, table, col, cons, actor)
}

// columnDependents lists what uses a column: constraints and lookups
func (a *API) columnDependents(ctx context.Context, table *tableInfo, m *columnMeta) ([]dependentOut, error) {
	deps := []dependentOut{}
	for _, kind := range []string{"unique", "check"} {
		list, err := a.schemaSvc.Constraints.List(ctx, a.schema, table.ID, kind)
		if err != nil {
			return nil, err
		}
		for _, item := range list {
			for _, col := range item.Columns {
				if strings.EqualFold(col.ID, m.ID) {
					deps = append(deps, dependentOut{Kind: "constraint", ID: EncodeUUID(PrefixConstraint, item.ID), Name: item.Title, Table: table.Title})
				}
			}
		}
	}
	rows, err := a.db.QueryContext(ctx, fmt.Sprintf(
		`SELECT c.id::text, c.title, m.title FROM %[1]s.columns c JOIN %[1]s.models m ON m.id = c.model_id
		 WHERE lower(c.uidt) = 'lookup' AND COALESCE(c.deleted, false) = false AND COALESCE(m.deleted, false) = false
		   AND (c.meta->>'link_column_id' = $1 OR c.meta->>'lookup_column_id' = $1) ORDER BY m.title, c.title`, a.schemaIdent()), m.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		d := dependentOut{Kind: "lookup_column"}
		if err := rows.Scan(&id, &d.Name, &d.Table); err != nil {
			return nil, err
		}
		d.ID = EncodeUUID(PrefixField, id)
		deps = append(deps, d)
	}
	return deps, rows.Err()
}

// columnHoldsData reports whether any record has a value in the column
func (a *API) columnHoldsData(ctx context.Context, table *tableInfo, m *columnMeta) bool {
	if m.ColumnName == "" || strings.EqualFold(m.UIDT, "lookup") {
		return false
	}
	var one int
	err := a.db.QueryRow(fmt.Sprintf(`SELECT 1 FROM %s WHERE %s IS NOT NULL LIMIT 1`,
		a.recordsTable(table), pq.QuoteIdentifier(m.ColumnName))).Scan(&one)
	return err == nil
}

// DELETE /tables/{table}/columns/{column}: 409 in_use lists what uses the column; ?force=true is needed when
// the column holds data
func (a *API) deleteColumn(c *gin.Context) {
	tc, err := a.resolveTablePath(c)
	if err != nil {
		respondError(c, err)
		return
	}
	metas, err := a.columnMetas(c, tc.table)
	if err != nil {
		respondError(c, err)
		return
	}
	m, err := pickMeta(metas, c.Param("column"))
	if err != nil {
		respondError(c, err)
		return
	}
	if isSystemMeta(m) {
		respondError(c, systemColumnError(m.Title))
		return
	}
	deps, err := a.columnDependents(c, tc.table, m)
	if err != nil {
		respondError(c, err)
		return
	}
	if len(deps) > 0 {
		respondError(c, NewError(CodeInUse, fmt.Sprintf("%d things use this column. Remove them first.", len(deps)), dependentDetails(deps)...))
		return
	}
	if c.Query("force") != "true" && a.columnHoldsData(c, tc.table, m) {
		respondError(c, NewError(CodeConflict, "This column holds data. Send ?force=true to delete the column and its values."))
		return
	}
	if err := a.schemaSvc.Tables.DeleteColumn(c, a.schema, m.ID); err != nil {
		respondError(c, guardedError(err))
		return
	}
	respond(c, http.StatusOK, map[string]any{"id": EncodeUUID(PrefixField, m.ID), "deleted": true})
}

// POST /tables/{table}/columns/reorder: {"column": "Email", "before": "Name"} or {"column": "Email", "after": "Name"}
func (a *API) reorderColumn(c *gin.Context) {
	body, err := readBody(c)
	if err != nil {
		respondError(c, err)
		return
	}
	if err := onlyKeys(body, "body", "column", "before", "after"); err != nil {
		respondError(c, err)
		return
	}
	colRef, _ := body["column"].(string)
	beforeRef, hasBefore := body["before"].(string)
	afterRef, hasAfter := body["after"].(string)
	if colRef == "" || hasBefore == hasAfter {
		respondError(c, badRequest("Send column and exactly one of before or after."))
		return
	}
	tc, err := a.resolveTablePath(c)
	if err != nil {
		respondError(c, err)
		return
	}
	metas, err := a.columnMetas(c, tc.table)
	if err != nil {
		respondError(c, err)
		return
	}
	src, err := pickMeta(metas, colRef)
	if err != nil {
		respondError(c, err)
		return
	}
	if isSystemMeta(src) {
		respondError(c, systemColumnError(src.Title))
		return
	}
	targetRef := beforeRef
	if hasAfter {
		targetRef = afterRef
	}
	target, err := pickMeta(metas, targetRef)
	if err != nil {
		respondError(c, err)
		return
	}
	if target.ID == src.ID {
		respondError(c, badRequest("A column can't be moved next to itself."))
		return
	}
	// The new position is halfway between the target and its neighbour on that side
	neighbour, hasNeighbour := 0.0, false
	for _, o := range metas {
		if o.ID == src.ID || o.ID == target.ID {
			continue
		}
		if hasBefore && o.Order < target.Order && (!hasNeighbour || o.Order > neighbour) {
			neighbour, hasNeighbour = o.Order, true
		}
		if hasAfter && o.Order > target.Order && (!hasNeighbour || o.Order < neighbour) {
			neighbour, hasNeighbour = o.Order, true
		}
	}
	var position float64
	switch {
	case hasNeighbour:
		position = (neighbour + target.Order) / 2
	case hasBefore:
		position = target.Order - 1
	default:
		position = target.Order + 1
	}
	if _, err := a.schemaSvc.Tables.UpdateColumn(c, a.schema, src.ID, dto.ColumnUpdate{OrderIndex: &position, UpdatedBy: a.actor(c, tc.base)}); err != nil {
		respondError(c, guardedError(err))
		return
	}
	fresh, err := a.columnMetas(c, tc.table)
	if err != nil {
		respondError(c, err)
		return
	}
	out := make([]columnWriteOut, 0, len(fresh))
	for _, f := range fresh {
		if isSystemMeta(f) {
			continue
		}
		out = append(out, columnWriteOut{ID: EncodeUUID(PrefixField, f.ID), Name: f.Title, UIDT: f.UIDT,
			Type: (&field{Type: strings.ToLower(f.UIDT)}).APIType()})
	}
	respond(c, http.StatusOK, out)
}
