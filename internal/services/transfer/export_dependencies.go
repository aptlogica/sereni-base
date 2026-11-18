// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"strings"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
)

// ExportDependencies lists the tables of a base and which table needs which. It applies the rules the export applies
// when a table is left out: a link field to a table that is not exported, a lookup or rollup read through such a
// link, and a trigger or function whose SQL uses the other table. includeAutomations says whether the triggers and
// functions are looked at (the user may not read them).
func (s *baseTransferService) ExportDependencies(ctx context.Context, schemaName, baseID string, includeAutomations bool) (*dto.ExportDependencies, error) {
	if _, err := s.baseService.GetBaseByID(ctx, schemaName, baseID); err != nil {
		return nil, err
	}
	responses, err := s.tableService.GetModelByBaseID(ctx, schemaName, baseID)
	if err != nil {
		return nil, err
	}
	tables, _, err := s.loadExportTables(ctx, schemaName, responses)
	if err != nil {
		return nil, err
	}

	out := &dto.ExportDependencies{Tables: []dto.ExportTableInfo{}, Dependencies: []dto.ExportDependency{}}
	byID := map[string]*exportTable{}
	for _, t := range tables {
		byID[t.model.ID.String()] = t
		info := dto.ExportTableInfo{ID: t.model.ID.String(), Title: t.model.Title, Fields: []dto.ExportFieldInfo{}}
		for _, c := range t.columns {
			if isFilterableField(c) {
				info.Fields = append(info.Fields, dto.ExportFieldInfo{ID: c.ID.String(), Title: c.Title, UIDT: c.UIDT, Options: filterOptions(c)})
			}
		}
		out.Tables = append(out.Tables, info)
	}

	seen := map[string]struct{}{}
	add := func(from, to *exportTable, kind, item string) {
		if from == nil || to == nil || from == to {
			return
		}
		key := strings.Join([]string{from.model.ID.String(), to.model.ID.String(), kind, item}, "|")
		if _, dup := seen[key]; dup {
			return
		}
		seen[key] = struct{}{}
		out.Dependencies = append(out.Dependencies, dto.ExportDependency{
			TableID:        from.model.ID.String(),
			TableTitle:     from.model.Title,
			DependsOnID:    to.model.ID.String(),
			DependsOnTitle: to.model.Title,
			Kind:           kind,
			Item:           item,
		})
	}

	// the table each link field points at, to follow the lookups and rollups built on it
	linkTarget := map[string]*exportTable{}
	for _, t := range tables {
		for _, c := range t.columns {
			if c.UIDT == uidtLinks {
				if target := byID[idString(nestedString(c.Meta, "relation", "with"))]; target != nil {
					linkTarget[c.ID.String()] = target
					add(t, target, dto.ExportDependencyLink, c.Title)
				}
			}
		}
	}
	for _, t := range tables {
		for _, c := range t.columns {
			if c.UIDT != uidtLookup && c.UIDT != "rollup" {
				continue
			}
			if target := linkTarget[idString(c.Meta["link_column_id"])]; target != nil {
				add(t, target, dto.ExportDependencyLookup, c.Title)
			}
		}
	}

	if includeAutomations && s.automations != nil {
		for _, t := range tables {
			list, err := s.automations.List(ctx, schemaName, t.model.ID.String())
			if err != nil {
				return nil, err
			}
			for _, a := range list {
				if a.Type != tenant.AutomationTypeTrigger && a.Type != tenant.AutomationTypeFunction {
					continue
				}
				for _, other := range tables {
					if other != t && containsWord(a.Context, other.model.Alias) {
						add(t, other, dto.ExportDependencyTrigger, a.Title)
					}
				}
			}
		}
	}
	return out, nil
}
