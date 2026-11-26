// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/gin-gonic/gin"
)

// ---- Tables (base management, APIs 3 to 5) ----

// tableWriteOut is a table after a schema write
type tableWriteOut struct {
	schemaTableOut
	Columns []columnWriteOut `json:"columns,omitempty"`
}

// dependentOut is one thing that uses what is being deleted
type dependentOut struct {
	Kind  string `json:"kind"` // link_column, lookup_column, constraint
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Table string `json:"table,omitempty"`
}

func dependentDetails(deps []dependentOut) []ErrorDetail {
	out := make([]ErrorDetail, len(deps))
	for i, d := range deps {
		where := d.Name
		if d.Table != "" {
			where = d.Table + "." + d.Name
		}
		out[i] = ErrorDetail{ID: d.ID, Field: where, Issue: d.Kind}
	}
	return out
}

// tableTitleTaken reports whether the base already has a table with this title (read fresh, not from the cache)
func (a *API) tableTitleTaken(ctx context.Context, base baseRef, title, exceptID string) (bool, error) {
	var one int
	err := a.db.QueryRow(fmt.Sprintf(
		`SELECT 1 FROM %s.models WHERE base_id = $1 AND title = $2 AND id::text <> $3 AND COALESCE(deleted, false) = false LIMIT 1`,
		a.schemaIdent()), base.ID, title, exceptID).Scan(&one)
	if err != nil {
		return false, nil // no row (or nothing readable): not taken
	}
	return true, nil
}

// POST /tables: {"name": "Orders", "description": "...", "columns": [ {title, uidt, ...} ]}
// The table never gets a default Title column; it has only the columns sent. If a step fails, the half-built table is removed.
func (a *API) createTable(c *gin.Context) {
	body, err := readBody(c)
	if err != nil {
		respondError(c, err)
		return
	}
	if err := onlyKeys(body, "body", "name", "title", "description", "columns"); err != nil {
		respondError(c, err)
		return
	}
	base := currentBase(c)
	name := ""
	for _, k := range []string{"name", "title"} {
		if s, ok := body[k].(string); ok {
			name = strings.TrimSpace(s)
		}
	}
	var details []ErrorDetail
	if name == "" {
		details = append(details, ErrorDetail{Field: "name", Issue: "is required"})
	}
	description, _ := body["description"].(string)

	var inputs []*columnInput
	if raw, given := body["columns"]; given {
		list, ok := raw.([]interface{})
		if !ok {
			details = append(details, ErrorDetail{Field: "columns", Issue: "must be an array"})
		}
		seen := map[string]bool{}
		for i, item := range list {
			where := fmt.Sprintf("columns[%d]", i)
			obj, ok := item.(map[string]interface{})
			if !ok {
				details = append(details, ErrorDetail{Field: where, Issue: "must be an object"})
				continue
			}
			in, d, err := parseColumnInput(obj, where, true)
			if err != nil {
				respondError(c, err)
				return
			}
			details = append(details, d...)
			if in.Title != nil {
				if seen[*in.Title] {
					details = append(details, ErrorDetail{Field: where + ".title", Issue: "two columns have this title"})
				}
				seen[*in.Title] = true
			}
			inputs = append(inputs, in)
		}
	}
	if len(details) > 0 {
		respondError(c, validationFailed(details))
		return
	}
	if taken, _ := a.tableTitleTaken(c, base, name, ""); taken {
		respondError(c, NewError(CodeConflict, fmt.Sprintf("This base already has a table named %q.", name)))
		return
	}

	actor := a.actor(c, base)
	req := dto.CreateTableRequest{BaseID: base.ID, WorkspaceID: base.WorkspaceID, Title: name, Description: description, CreatedBy: actor}
	// Never add the default Title column: the table has only the columns the caller sends
	created, err := a.schemaSvc.Tables.CreateTableWithDefaultsImport(c, req, a.schema)
	if err != nil {
		respondError(c, guardedError(err))
		return
	}
	table := &tableInfo{ID: created.Model.ID.String(), BaseID: base.ID, Title: created.Model.Title, Alias: created.Model.Alias}

	out := tableWriteOut{schemaTableOut: schemaTableOf(base, created.Model)}
	for _, col := range created.Columns {
		if col.System != nil && *col.System {
			continue
		}
		out.Columns = append(out.Columns, columnWriteOutOf(col))
	}
	for i, in := range inputs {
		col, err := a.addColumn(c, base, table, in, actor)
		if err != nil {
			// Remove the half-built table, so a failed request leaves nothing behind
			if delErr := a.schemaSvc.Tables.DeleteTable(c, a.schema, table.ID); delErr != nil {
				respondError(c, NewError(CodeInternalError, "The table could not be completed and could not be removed again; delete it by hand."))
				return
			}
			respondError(c, prefixDetails(err, fmt.Sprintf("columns[%d]", i)))
			return
		}
		out.Columns = append(out.Columns, col)
	}
	respond(c, http.StatusCreated, out)
}

// prefixDetails puts the position of the failing item in front of an error's detail fields
func prefixDetails(err error, where string) error {
	apiErr, ok := err.(*Error)
	if !ok {
		return err
	}
	out := &Error{Code: apiErr.Code, Message: apiErr.Message, Details: append([]ErrorDetail(nil), apiErr.Details...)}
	for i := range out.Details {
		if out.Details[i].Field == "" {
			out.Details[i].Field = where
		} else if !strings.HasPrefix(out.Details[i].Field, where) {
			out.Details[i].Field = where + "." + out.Details[i].Field
		}
	}
	if len(out.Details) == 0 {
		out.Details = []ErrorDetail{{Field: where, Issue: apiErr.Message}}
	}
	return out
}

// PATCH /tables/{table}: rename or change the description; IDs survive renames
func (a *API) updateTable(c *gin.Context) {
	body, err := readBody(c)
	if err != nil {
		respondError(c, err)
		return
	}
	if err := onlyKeys(body, "body", "name", "title", "description"); err != nil {
		respondError(c, err)
		return
	}
	tc, err := a.resolveTablePath(c)
	if err != nil {
		respondError(c, err)
		return
	}
	req := dto.UpdateTableRequest{UpdatedBy: a.actor(c, tc.base)}
	changed := false
	for _, k := range []string{"name", "title"} {
		v, given := body[k]
		if !given {
			continue
		}
		s, ok := v.(string)
		if !ok || strings.TrimSpace(s) == "" {
			respondError(c, validationFailed([]ErrorDetail{{Field: k, Issue: "must be a non-empty string"}}))
			return
		}
		s = strings.TrimSpace(s)
		if taken, _ := a.tableTitleTaken(c, tc.base, s, tc.table.ID); taken {
			respondError(c, NewError(CodeConflict, fmt.Sprintf("This base already has a table named %q.", s)))
			return
		}
		req.Title, changed = &s, true
	}
	if v, given := body["description"]; given {
		s, ok := v.(string)
		if !ok {
			respondError(c, validationFailed([]ErrorDetail{{Field: "description", Issue: "must be a string"}}))
			return
		}
		req.Description, changed = &s, true
	}
	if !changed {
		respondError(c, badRequest("Send name or description."))
		return
	}
	updated, err := a.schemaSvc.Tables.UpdateTable(c, tc.table.ID, req, a.schema)
	if err != nil {
		respondError(c, guardedError(err))
		return
	}
	respond(c, http.StatusOK, schemaTableOf(tc.base, updated.Model))
}

// tablesLinkingTo lists the link columns of other tables that point at this table
func (a *API) tablesLinkingTo(ctx context.Context, base baseRef, table *tableInfo) ([]dependentOut, error) {
	rows, err := a.db.QueryContext(ctx, fmt.Sprintf(
		`SELECT c.id::text, c.title, m.title FROM %[1]s.columns c JOIN %[1]s.models m ON m.id = c.model_id
		 WHERE m.base_id = $1 AND m.id::text <> $2 AND lower(c.uidt) = 'links'
		   AND COALESCE(c.deleted, false) = false AND COALESCE(m.deleted, false) = false
		   AND c.meta->'relation'->>'with' = $2 ORDER BY m.title, c.title`, a.schemaIdent()), base.ID, table.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	deps := []dependentOut{}
	for rows.Next() {
		var d dependentOut
		var id string
		if err := rows.Scan(&id, &d.Name, &d.Table); err != nil {
			return nil, err
		}
		d.Kind, d.ID = "link_column", EncodeUUID(PrefixField, id)
		deps = append(deps, d)
	}
	return deps, rows.Err()
}

// DELETE /tables/{table}: deletes the table and its records. Other tables that link to it make this 409
// in_use, unless ?force=true.
func (a *API) deleteTable(c *gin.Context) {
	tc, err := a.resolveTablePath(c)
	if err != nil {
		respondError(c, err)
		return
	}
	if c.Query("force") != "true" {
		deps, err := a.tablesLinkingTo(c, tc.base, tc.table)
		if err != nil {
			respondError(c, err)
			return
		}
		if len(deps) > 0 {
			respondError(c, NewError(CodeInUse,
				fmt.Sprintf("%d link columns in other tables point at this table. Delete them first, or send ?force=true to delete the table anyway.", len(deps)),
				dependentDetails(deps)...))
			return
		}
	}
	if err := a.schemaSvc.Tables.DeleteTable(c, a.schema, tc.table.ID); err != nil {
		respondError(c, guardedError(err))
		return
	}
	respond(c, http.StatusOK, map[string]any{"id": EncodeUUID(PrefixTable, tc.table.ID), "deleted": true})
}
