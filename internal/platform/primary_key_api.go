// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"fmt"
	"net/http"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/gin-gonic/gin"
)

type primaryKeyColumnOut struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	InKey    bool   `json:"in_key"`
	Eligible bool   `json:"eligible"`
	Reason   string `json:"reason,omitempty"`
}

type primaryKeyOut struct {
	Columns []columnRefOut        `json:"columns"` // key columns in key order
	Fields  []primaryKeyColumnOut `json:"candidates"`
}

func primaryKeyOutOf(r dto.PrimaryKeyResponse) primaryKeyOut {
	out := primaryKeyOut{Columns: []columnRefOut{}, Fields: []primaryKeyColumnOut{}}
	for i, id := range r.ColumnIDs {
		name := ""
		if i < len(r.ColumnNames) {
			name = r.ColumnNames[i]
		}
		out.Columns = append(out.Columns, columnRefOut{ID: EncodeUUID(PrefixField, id), Name: name, ColumnName: name})
	}
	for _, f := range r.Columns {
		out.Fields = append(out.Fields, primaryKeyColumnOut{
			ID: EncodeUUID(PrefixField, f.ID), Name: f.Title, InKey: f.InKey, Eligible: f.Eligible, Reason: f.Reason,
		})
	}
	return out
}

// GET /tables/{table}/primary-key
func (a *API) getPrimaryKey(c *gin.Context) {
	tc, err := a.resolveTablePath(c)
	if err != nil {
		respondError(c, err)
		return
	}
	pk, err := a.schemaSvc.PrimaryKeys.Get(c, a.schema, tc.table.ID)
	if err != nil {
		respondError(c, guardedError(err))
		return
	}
	respond(c, http.StatusOK, primaryKeyOutOf(pk))
}

// PUT /tables/{table}/primary-key: {"columns": ["SKU", "Region"]}
func (a *API) setPrimaryKey(c *gin.Context) {
	body, err := readBody(c)
	if err != nil {
		respondError(c, err)
		return
	}
	if err := onlyKeys(body, "body", "columns"); err != nil {
		respondError(c, err)
		return
	}
	list, ok := body["columns"].([]interface{})
	if !ok || len(list) == 0 {
		respondError(c, badRequest("columns must be a non-empty array of column names or IDs."))
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
	var ids []string
	var details []ErrorDetail
	for i, v := range list {
		ref, _ := v.(string)
		m, err := pickMeta(metas, ref)
		if err != nil {
			details = append(details, ErrorDetail{Field: fmt.Sprintf("columns[%d]", i), Issue: err.Error()})
			continue
		}
		ids = append(ids, m.ID)
	}
	if len(details) > 0 {
		respondError(c, NewError(CodeValidationFailed, "Some columns don't exist.", details...))
		return
	}
	pk, err := a.schemaSvc.PrimaryKeys.Update(c, a.schema, tc.table.ID, dto.UpdatePrimaryKeyRequest{ColumnIDs: ids})
	if err != nil {
		respondError(c, guardedError(err))
		return
	}
	respond(c, http.StatusOK, primaryKeyOutOf(pk))
}
