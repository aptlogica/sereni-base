// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type baseOut struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type tableOut struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type fieldOut struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Type     string    `json:"type"`
	ReadOnly bool      `json:"read_only"`
	Options  *[]string `json:"options,omitempty"` // select fields only; [] when the field has none
	// Links: the table the field links to, whether it holds several records, and the lookups each linked
	// record's object carries besides its id
	LinkedTableID string       `json:"linked_table_id,omitempty"`
	Multiple      *bool        `json:"multiple,omitempty"`
	Lookups       *[]lookupOut `json:"lookups,omitempty"`
}

// lookupOut is a key of a link field's objects: the looked-up field in the linked table
type lookupOut struct {
	Key  string `json:"key"`
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type tableDetailOut struct {
	ID     string     `json:"id"`
	Name   string     `json:"name"`
	BaseID string     `json:"base_id"`
	Fields []fieldOut `json:"fields"`
}

// GET /bases
func (a *API) listBases(c *gin.Context) {
	bases, err := a.bases(c, currentWorkspace(c))
	if err != nil {
		respondError(c, err)
		return
	}
	out := make([]baseOut, len(bases))
	for i, b := range bases {
		out[i] = baseOut{ID: EncodeUUID(PrefixBase, b.ID), Name: b.Title}
	}
	respond(c, http.StatusOK, out)
}

// GET /bases/{base}/tables
func (a *API) listTables(c *gin.Context) {
	ws := currentWorkspace(c)
	base, err := a.resolveBase(c, ws, c.Param("base"))
	if err != nil {
		respondError(c, err)
		return
	}
	tables, err := a.tables(c, ws, base)
	if err != nil {
		respondError(c, err)
		return
	}
	out := make([]tableOut, len(tables))
	for i, t := range tables {
		out[i] = tableOut{ID: EncodeUUID(PrefixTable, t.ID), Name: t.Title}
	}
	respond(c, http.StatusOK, out)
}

// GET /bases/{base}/tables/{table}: the table with its fields. Lookups are not listed as fields: they are
// listed under the link field whose objects carry them.
func (a *API) getTable(c *gin.Context) {
	tc, err := a.resolveTablePath(c)
	if err != nil {
		respondError(c, err)
		return
	}
	out := tableDetailOut{
		ID:     EncodeUUID(PrefixTable, tc.table.ID),
		Name:   tc.table.Title,
		BaseID: EncodeUUID(PrefixBase, tc.base.ID),
		Fields: []fieldOut{},
	}
	for _, f := range tc.fields {
		if f.Type == typeLookup {
			continue
		}
		fo := fieldOut{ID: f.PublicID(), Name: f.Name, Type: f.APIType(), ReadOnly: f.ReadOnly}
		if f.Type == typeSelect || f.Type == typeMultiSelect {
			options := append([]string{}, f.Options...)
			fo.Options = &options
		}
		if f.Type == typeLinks && f.LinkTargetModelID != "" {
			fo.LinkedTableID = EncodeUUID(PrefixTable, f.LinkTargetModelID)
			multiple := f.LinkMulti
			fo.Multiple = &multiple
			lookups, err := a.linkLookupFields(c, tc, f)
			if err != nil {
				respondError(c, err)
				return
			}
			fo.Lookups = &lookups
		}
		out.Fields = append(out.Fields, fo)
	}
	respond(c, http.StatusOK, out)
}

// linkLookupFields lists the lookups that go through a link field, as keyed in its objects
func (a *API) linkLookupFields(c *gin.Context, tc *tableContext, link *field) ([]lookupOut, error) {
	var lookups []*field
	for _, f := range tc.fields {
		if f.Type == typeLookup && strings.EqualFold(f.LookupLinkColumnID, link.ID) {
			lookups = append(lookups, f)
		}
	}
	_, targets, err := a.lookupTargets(c, tc.ws, link, lookups)
	if err != nil {
		return nil, err
	}
	keys := lookupKeys(targets, false)
	out := make([]lookupOut, len(targets))
	for i, tf := range targets {
		out[i] = lookupOut{Key: keys[tf], ID: tf.PublicID(), Name: tf.Name, Type: tf.APIType()}
	}
	return out, nil
}
