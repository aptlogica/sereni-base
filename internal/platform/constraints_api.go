// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/gin-gonic/gin"
)

// PrefixConstraint is the prefix of table constraint IDs
const PrefixConstraint = "con_"

// constraintCheckOut is the check of a check constraint, in the shape the request takes
type constraintCheckOut struct {
	Category string                 `json:"category"`
	Where    map[string]interface{} `json:"where,omitempty"`
	// The SQL Postgres enforces. Read only, and the only form of a check made in the app (it has no where)
	Condition string `json:"condition,omitempty"`
}

// tableConstraintOut is a unique or check constraint of a table. title, type, columns and check are the
// fields POST and PUT take, so a response can be edited and sent back; the rest is read only.
type tableConstraintOut struct {
	ID             string              `json:"id"`
	Title          string              `json:"title"`
	Type           string              `json:"type"`    // unique or check
	Columns        []string            `json:"columns"` // column names
	Check          *constraintCheckOut `json:"check,omitempty"`
	ConstraintName string              `json:"constraint_name"`
	Status         string              `json:"status"` // ok, or missing when Postgres no longer has it
	Definition     string              `json:"definition,omitempty"`
}

// constraintReadOnlyKeys are in every response but can't be changed; a PUT that sends a response back
// may include them
var constraintReadOnlyKeys = []string{"id", "constraint_name", "status", "definition"}

func constraintOutOf(c dto.TableConstraintResponse, kind string) tableConstraintOut {
	out := tableConstraintOut{
		ID: EncodeUUID(PrefixConstraint, c.ID), Title: c.Title, Type: kind, ConstraintName: c.ConstraintName,
		Status: c.Status, Definition: c.Definition, Columns: make([]string, len(c.Columns)),
	}
	for i, col := range c.Columns {
		out.Columns[i] = col.Title
	}
	if kind == "check" {
		out.Check = &constraintCheckOut{Category: c.Category, Where: c.Where, Condition: c.Condition}
	}
	return out
}

// constraintBody is the body of a constraint request, after validation
type constraintBody struct {
	Title     string
	Type      string
	ColumnIDs []string
	Check     dto.CheckInput
}

// parseConstraintBody reads title, type, columns and check. create: title and type are required; the type
// can't change afterwards.
func (a *API) parseConstraintBody(c *gin.Context, tc *tableContext, body map[string]interface{}, mode string) (*constraintBody, error) {
	allowed := []string{"title", "name", "type", "columns", "check"}
	if err := onlyKeys(body, "body", allowed...); err != nil {
		return nil, err
	}
	out := &constraintBody{}
	var details []ErrorDetail
	bad := func(field, issue string) { details = append(details, ErrorDetail{Field: field, Issue: issue}) }

	for _, k := range []string{"title", "name"} {
		if s, ok := body[k].(string); ok && strings.TrimSpace(s) != "" {
			out.Title = strings.TrimSpace(s)
		}
	}
	if out.Title == "" && mode != "check" {
		bad("title", "is required")
	}
	if t, ok := body["type"].(string); ok {
		out.Type = t
	}
	if mode == "update" {
		if _, given := body["type"]; given {
			bad("type", "can't be changed; delete the constraint and add a new one")
		}
	} else if out.Type != "unique" && out.Type != "check" && !(mode == "check" && out.Type == "") {
		bad("type", "must be unique or check")
	}
	metas, err := a.columnMetas(c, tc.table)
	if err != nil {
		return nil, err
	}
	if list, ok := body["columns"].([]interface{}); ok {
		for i, v := range list {
			ref, _ := v.(string)
			m, err := pickMeta(metas, ref)
			if err != nil {
				bad(fmt.Sprintf("columns[%d]", i), err.Error())
				continue
			}
			out.ColumnIDs = append(out.ColumnIDs, m.ID)
		}
	} else if _, given := body["columns"]; given {
		bad("columns", "must be an array of column names or IDs")
	}
	if out.Type == "unique" && len(out.ColumnIDs) == 0 && len(details) == 0 {
		bad("columns", "a unique constraint needs at least one column")
	}
	if rawCheck, given := body["check"]; given {
		obj, ok := rawCheck.(map[string]interface{})
		if !ok {
			bad("check", "must be an object with category and where")
		} else if err := onlyKeys(obj, "check", "category", "where"); err != nil {
			return nil, err
		} else if where, ok := obj["where"].(map[string]interface{}); !ok {
			bad("check.where", "is required: a condition like {\"all\": [{\"column\": \"Age\", \"op\": \"gte\", \"value\": 18}]}")
		} else {
			sql, used, err := compileCondition(where, metas)
			if err != nil {
				return nil, err
			}
			out.Check.Condition = sql
			out.Check.Where = where
			out.Check.Category, _ = obj["category"].(string)
			if out.Check.Category == "" {
				out.Check.Category = "custom"
			}
			// The columns the condition uses are the constraint's columns when none are named
			if len(out.ColumnIDs) == 0 {
				for _, u := range used {
					out.ColumnIDs = append(out.ColumnIDs, u.ID)
				}
			}
		}
	}
	if (out.Type == "check" || (mode == "update")) && out.Check.Condition == "" && mode != "check" && out.Type == "check" {
		bad("check", "a check constraint needs a check condition")
	}
	if len(details) > 0 {
		return nil, NewError(CodeValidationFailed, "Some fields are not valid.", details...)
	}
	return out, nil
}

// dropEchoedFields removes what a response has and a request doesn't take, so a response can be edited and
// sent back: the read-only fields, an unchanged type, the SQL condition, and a check that only repeats
// the category of one made in the app (it has no where to send back)
func dropEchoedFields(body map[string]interface{}, kind, category string) {
	for _, k := range constraintReadOnlyKeys {
		delete(body, k)
	}
	if t, _ := body["type"].(string); t == kind {
		delete(body, "type")
	}
	chk, ok := body["check"].(map[string]interface{})
	if !ok {
		return
	}
	delete(chk, "condition")
	if _, hasWhere := chk["where"]; hasWhere {
		return
	}
	if cat, _ := chk["category"].(string); len(chk) == 0 || (len(chk) == 1 && cat == category) {
		delete(body, "check")
	}
}

// findConstraint returns the table's constraint with this ID or title
func (a *API) findConstraint(c *gin.Context, tc *tableContext, ref string) (dto.TableConstraintResponse, string, error) {
	var found []dto.TableConstraintResponse
	var kinds []string
	for _, kind := range []string{"unique", "check"} {
		list, err := a.schemaSvc.Constraints.List(c, a.schema, tc.table.ID, kind)
		if err != nil {
			return dto.TableConstraintResponse{}, "", err
		}
		for _, item := range list {
			if isUUIDID(PrefixConstraint, ref) {
				if id, ok := DecodeUUID(PrefixConstraint, ref); ok && strings.EqualFold(id, item.ID) {
					found, kinds = append(found, item), append(kinds, kind)
				}
			} else if item.Title == ref {
				found, kinds = append(found, item), append(kinds, kind)
			}
		}
	}
	switch len(found) {
	case 0:
		return dto.TableConstraintResponse{}, "", notFound(fmt.Sprintf("No constraint %q on this table.", ref))
	case 1:
		return found[0], kinds[0], nil
	}
	details := make([]ErrorDetail, len(found))
	for i, f := range found {
		details[i] = ErrorDetail{ID: EncodeUUID(PrefixConstraint, f.ID)}
	}
	return dto.TableConstraintResponse{}, "", NewError(CodeAmbiguousName,
		fmt.Sprintf("%d constraints are named %q. Use one of their IDs instead.", len(found), ref), details...)
}

// constraintError says what is wrong when the app refuses a constraint
func constraintError(err error) error {
	if err == nil {
		return nil
	}
	var apiErr *Error
	if errors.As(err, &apiErr) {
		return err
	}
	if errors.Is(err, app_errors.DatabaseError) {
		return err
	}
	return guardedError(err)
}

// GET /tables/{table}/constraints?type=unique|check
func (a *API) listConstraints(c *gin.Context) {
	tc, err := a.resolveTablePath(c)
	if err != nil {
		respondError(c, err)
		return
	}
	kinds := []string{"unique", "check"}
	if t := c.Query("type"); t != "" {
		if t != "unique" && t != "check" {
			respondError(c, badRequest("type must be unique or check."))
			return
		}
		kinds = []string{t}
	}
	out := []tableConstraintOut{}
	for _, kind := range kinds {
		list, err := a.schemaSvc.Constraints.List(c, a.schema, tc.table.ID, kind)
		if err != nil {
			respondError(c, constraintError(err))
			return
		}
		for _, item := range list {
			out = append(out, constraintOutOf(item, kind))
		}
	}
	respond(c, http.StatusOK, out)
}

// POST /tables/{table}/constraints
func (a *API) createConstraint(c *gin.Context) {
	body, err := readBody(c)
	if err != nil {
		respondError(c, err)
		return
	}
	tc, err := a.resolveTablePath(c)
	if err != nil {
		respondError(c, err)
		return
	}
	in, err := a.parseConstraintBody(c, tc, body, "create")
	if err != nil {
		respondError(c, err)
		return
	}
	created, err := a.schemaSvc.Constraints.Create(c, a.schema, dto.CreateTableConstraintRequest{
		ModelID: tc.table.ID, Type: in.Type, Title: in.Title, ColumnIDs: in.ColumnIDs, CheckInput: in.Check,
		CreatedBy: a.actor(c, tc.base),
	})
	if err != nil {
		respondError(c, constraintError(err))
		return
	}
	respond(c, http.StatusCreated, constraintOutOf(created, in.Type))
}

// PUT /tables/{table}/constraints/{constraint}
func (a *API) updateConstraint(c *gin.Context) {
	body, err := readBody(c)
	if err != nil {
		respondError(c, err)
		return
	}
	tc, err := a.resolveTablePath(c)
	if err != nil {
		respondError(c, err)
		return
	}
	existing, kind, err := a.findConstraint(c, tc, c.Param("constraint"))
	if err != nil {
		respondError(c, err)
		return
	}
	dropEchoedFields(body, kind, existing.Category)
	in, err := a.parseConstraintBody(c, tc, body, "update")
	if err != nil {
		respondError(c, err)
		return
	}
	in.Type = kind
	// A check left out stays as it is, so a title or columns change doesn't have to repeat it
	if kind == "check" && in.Check.Condition == "" {
		in.Check = existing.CheckInput
	}
	if in.Title == "" {
		in.Title = existing.Title
	}
	updated, err := a.schemaSvc.Constraints.Update(c, a.schema, existing.ID, dto.UpdateTableConstraintRequest{
		Title: in.Title, ColumnIDs: in.ColumnIDs, CheckInput: in.Check, UpdatedBy: a.actor(c, tc.base),
	})
	if err != nil {
		respondError(c, constraintError(err))
		return
	}
	respond(c, http.StatusOK, constraintOutOf(updated, kind))
}

// DELETE /tables/{table}/constraints/{constraint}
func (a *API) deleteConstraint(c *gin.Context) {
	tc, err := a.resolveTablePath(c)
	if err != nil {
		respondError(c, err)
		return
	}
	existing, _, err := a.findConstraint(c, tc, c.Param("constraint"))
	if err != nil {
		respondError(c, err)
		return
	}
	if err := a.schemaSvc.Constraints.Delete(c, a.schema, existing.ID); err != nil {
		respondError(c, constraintError(err))
		return
	}
	respond(c, http.StatusOK, map[string]any{"id": EncodeUUID(PrefixConstraint, existing.ID), "deleted": true})
}

// POST /tables/{table}/constraints/check: a dry run; nothing is changed
func (a *API) checkConstraint(c *gin.Context) {
	body, err := readBody(c)
	if err != nil {
		respondError(c, err)
		return
	}
	tc, err := a.resolveTablePath(c)
	if err != nil {
		respondError(c, err)
		return
	}
	in, err := a.parseConstraintBody(c, tc, body, "check")
	if err != nil {
		respondError(c, err)
		return
	}
	if in.Type == "" {
		in.Type = "unique"
	}
	result, err := a.schemaSvc.Constraints.Check(c, a.schema, dto.CheckTableConstraintRequest{
		ModelID: tc.table.ID, Type: in.Type, ColumnIDs: in.ColumnIDs, CheckInput: in.Check,
	})
	if err != nil {
		respondError(c, constraintError(err))
		return
	}
	respond(c, http.StatusOK, result)
}
