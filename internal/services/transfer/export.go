// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
)

// exportTable is a table while it is being exported.
type exportTable struct {
	key     string
	model   dto.ModelResponse
	columns []dto.ColumnResponse
	// keys of the exported (non-system) fields by column id
	fieldKeys map[string]string
}

// ExportBase reads a base into a file value: tables, fields, views and, with opts.IncludeData, the rows.
func (s *baseTransferService) ExportBase(ctx context.Context, schemaName, baseID string, opts dto.ExportBaseOptions) (*dto.BaseExportFile, error) {
	base, err := s.baseService.GetBaseByID(ctx, schemaName, baseID)
	if err != nil {
		return nil, err
	}
	tableResponses, err := s.tableService.GetModelByBaseID(ctx, schemaName, baseID)
	if err != nil {
		return nil, err
	}

	if len(opts.Tables) > 0 {
		tableResponses = filterTableResponses(tableResponses, opts.Tables)
		if len(tableResponses) == 0 {
			return nil, app_errors.TableNotFound
		}
	}

	reportProgress(ctx, 2)
	file := &dto.BaseExportFile{
		Base: dto.BaseExportBase{
			Title:    base.Title,
			Settings: normalizeMap(base.Settings),
			Meta:     normalizeMap(base.Meta),
		},
		Schema: dto.BaseExportSchema{Tables: []dto.BaseExportTable{}},
	}
	if base.Description != nil {
		file.Base.Description = *base.Description
	}

	tables, idMap, err := s.loadExportTables(ctx, schemaName, tableResponses)
	if err != nil {
		return nil, err
	}
	reportProgress(ctx, 8)
	warnings := &file.Skipped
	skipped, asText := s.skippedColumns(tables, warnings)
	// Fields that a view, a trigger, a key or a constraint cannot use as they were: left out, or turned into text
	unusable := unionOf(skipped, asText)

	skippedKeys := map[string]struct{}{}
	for id := range unusable {
		if key, ok := idMap[id]; ok {
			skippedKeys[key] = struct{}{}
		}
	}

	// Files (attachments, cover image) travel only in a zip, and only when storage can be read
	withFiles := opts.IncludeAttachments && s.storage != nil
	if opts.IncludeAttachments && !withFiles {
		*warnings = append(*warnings, "files not exported: no storage provider is configured")
	}
	file.Base.Cover = s.exportCover(ctx, schemaName, base.Image, withFiles, warnings)

	var automations map[string]*exportedAutomations
	if opts.IncludeAutomations {
		automations, err = s.exportAutomations(ctx, schemaName, tables, unusable, warnings)
		if err != nil {
			return nil, err
		}
	}

	for ti, t := range tables {
		reportFraction(ctx, 8, 15, ti, len(tables))
		table := s.exportTableSchema(t, idMap, skipped, asText)
		if a := automations[t.key]; a != nil {
			table.Functions, table.Triggers = a.functions, a.triggers
		}
		table.PrimaryKey = s.exportPrimaryKey(ctx, schemaName, t, unusable, warnings)
		table.Constraints = s.exportConstraints(ctx, schemaName, t, unusable, warnings)
		views, err := s.exportViews(ctx, schemaName, t, idMap, skippedKeys)
		if err != nil {
			return nil, err
		}
		table.Views = views
		file.Schema.Tables = append(file.Schema.Tables, table)
	}

	if opts.IncludeData {
		data, err := s.exportData(ctx, schemaName, tables, skipped, asText, withFiles, warnings)
		if err != nil {
			return nil, err
		}
		file.Data = data
	}
	reportProgress(ctx, 65)
	// What was left out is only logged: the file and the response carry nothing about the export run
	for _, w := range file.Skipped {
		logger.Get().Warn().Str("base_id", baseID).Msg("base export: " + w)
	}
	return file, nil
}

// loadExportTables reads tables and columns, gives every table, column and relation a key, and returns the
// id -> "@key" map used to rewrite ids inside meta.
func (s *baseTransferService) loadExportTables(ctx context.Context, schemaName string, responses []dto.TableResponse) ([]*exportTable, map[string]string, error) {
	models := make([]dto.ModelResponse, 0, len(responses))
	for _, r := range responses {
		models = append(models, r.Model)
	}
	sort.SliceStable(models, func(i, j int) bool {
		if a, b := orderOf(models[i].OrderIndex), orderOf(models[j].OrderIndex); a != b {
			return a < b
		}
		return models[i].Title < models[j].Title
	})

	alloc := newKeyAllocator()
	idMap := map[string]string{}
	tables := make([]*exportTable, 0, len(models))

	for _, m := range models {
		cols, err := s.tableService.GetColumnsByModelID(ctx, schemaName, m.ID.String())
		if err != nil {
			return nil, nil, err
		}
		live := make([]dto.ColumnResponse, 0, len(cols))
		for _, c := range cols {
			if !boolOf(c.Deleted) {
				live = append(live, c)
			}
		}
		sort.SliceStable(live, func(i, j int) bool { return orderOf(live[i].OrderIndex) < orderOf(live[j].OrderIndex) })

		t := &exportTable{key: alloc.next("t_", m.Title), model: m, columns: live, fieldKeys: map[string]string{}}
		idMap[m.ID.String()] = dto.BaseExportRefPrefix + t.key
		tables = append(tables, t)
	}

	// Field and relation keys are given after all tables, so they come out in table order
	for _, t := range tables {
		for _, c := range t.columns {
			key := alloc.next("f_", c.Title)
			t.fieldKeys[c.ID.String()] = key
			idMap[c.ID.String()] = dto.BaseExportRefPrefix + key
		}
	}

	// Relations are named after their source field (r_customers_orders); one with no source field gets a plain name
	for _, role := range []string{entityRoleSource, ""} {
		for _, t := range tables {
			for _, c := range t.columns {
				rid := idString(c.Meta["relation_id"])
				if c.UIDT != uidtLinks || rid == "" || (role != "" && idString(c.Meta["entity_role"]) != role) {
					continue
				}
				if _, ok := idMap[rid]; !ok {
					name := "relation"
					if role != "" {
						name = strings.TrimPrefix(t.key, "t_") + " " + c.Title
					}
					idMap[rid] = dto.BaseExportRefPrefix + clipKey(alloc.next("r_", name))
				}
			}
		}
	}
	return tables, idMap, nil
}

// skippedColumns finds fields that import cannot create (unsupported type, or built on such a field) and says so.
// A link to a table that is not exported cannot stay a link: it is returned in the second set, with the lookups and
// rollups built on it, and is exported as a text field.
func (s *baseTransferService) skippedColumns(tables []*exportTable, warnings *[]string) (map[string]struct{}, map[string]struct{}) {
	skipped := map[string]struct{}{}
	asText := map[string]struct{}{}
	titles := map[string]string{}
	tableIDs := map[string]struct{}{}
	for _, t := range tables {
		tableIDs[t.model.ID.String()] = struct{}{}
	}

	for _, t := range tables {
		for _, c := range t.columns {
			titles[c.ID.String()] = t.model.Title + "." + c.Title
			if !boolOf(c.System) && !isSupportedUIDT(c.UIDT) {
				skipped[c.ID.String()] = struct{}{}
				*warnings = append(*warnings, fmt.Sprintf("field %s (%s) not exported: type %q is not supported yet", titles[c.ID.String()], c.ID, c.UIDT))
			}
			if c.UIDT == uidtLinks {
				withID := idString(nestedString(c.Meta, "relation", "with"))
				if withID != "" {
					if _, ok := tableIDs[withID]; !ok {
						asText[c.ID.String()] = struct{}{}
						*warnings = append(*warnings, fmt.Sprintf("field %s exported as text: linked table is not exported", titles[c.ID.String()]))
					}
				}
			}
		}
	}
	// A field built on a skipped field (e.g. a lookup of a rollup) cannot be rebuilt either
	for changed := true; changed; {
		changed = false
		for _, t := range tables {
			// the relations of the links of this table that become text: a lookup names its link by field id or by relation
			textRelations := map[string]struct{}{}
			for _, c := range t.columns {
				if _, ok := asText[c.ID.String()]; ok && c.UIDT == uidtLinks {
					if rid := idString(c.Meta["relation_id"]); rid != "" {
						textRelations[rid] = struct{}{}
					}
				}
			}
			for _, c := range t.columns {
				id := c.ID.String()
				if _, done := skipped[id]; done || boolOf(c.System) {
					continue
				}
				if _, done := asText[id]; done {
					continue
				}
				meta := normalizeMap(c.Meta)
				if containsString(meta, skipped) {
					skipped[id] = struct{}{}
					*warnings = append(*warnings, fmt.Sprintf("field %s not exported: it depends on a field that is not exported", titles[id]))
					changed = true
				} else if _, throughText := textRelations[idString(c.Meta["relation_id"])]; isLookupLike(c.UIDT) && throughText && !containsString(meta, asText) {
					asText[id] = struct{}{}
					*warnings = append(*warnings, fmt.Sprintf("field %s exported as text: it reads through a link that is exported as text", titles[id]))
					changed = true
				} else if containsString(meta, asText) {
					// a lookup or rollup of a link that becomes text becomes text; anything else built on it cannot
					if isLookupLike(c.UIDT) {
						asText[id] = struct{}{}
						*warnings = append(*warnings, fmt.Sprintf("field %s exported as text: it reads through a link that is exported as text", titles[id]))
					} else {
						skipped[id] = struct{}{}
						*warnings = append(*warnings, fmt.Sprintf("field %s not exported: it depends on a field that is exported as text", titles[id]))
					}
					changed = true
				}
			}
		}
	}
	return skipped, asText
}

func (s *baseTransferService) exportTableSchema(t *exportTable, idMap map[string]string, skipped, asText map[string]struct{}) dto.BaseExportTable {
	out := dto.BaseExportTable{
		Key:        t.key,
		Title:      t.model.Title,
		OrderIndex: orderOf(t.model.OrderIndex),
		Fields:     []dto.BaseExportField{},
	}
	out.Description = t.model.Description
	// System fields (Id, created and modified) come first, then the table's own fields, each group by order_index
	columns := make([]dto.ColumnResponse, 0, len(t.columns))
	for _, system := range []bool{true, false} {
		for _, c := range t.columns {
			if boolOf(c.System) == system {
				columns = append(columns, c)
			}
		}
	}
	for _, c := range columns {
		if _, skip := skipped[c.ID.String()]; skip {
			continue
		}
		field := dto.BaseExportField{
			Key:         t.fieldKeys[c.ID.String()],
			Title:       c.Title,
			Description: c.Description,
			UIDT:        c.UIDT,
			DT:          c.DT,
			OrderIndex:  orderOf(c.OrderIndex),
		}
		if boolOf(c.System) {
			// System fields are not created by import; the column name lets it find them in the new table
			field.System = true
			field.ColumnName = c.ColumnName
		} else if _, text := asText[c.ID.String()]; text {
			// a link, lookup or rollup exported as text: a plain text field, with none of the relation
			field.UIDT, field.DT, field.Meta = textFieldType, "TEXT", nil
		} else {
			field.Meta = rewriteMap(c.Meta, idMap)
			if len(field.Meta) == 0 {
				field.Meta = nil
			}
		}
		out.Fields = append(out.Fields, field)
	}
	return out
}

func (s *baseTransferService) exportViews(ctx context.Context, schemaName string, t *exportTable, idMap map[string]string, skippedKeys map[string]struct{}) ([]dto.BaseExportView, error) {
	views, err := s.tableService.GetViewsByModelID(ctx, schemaName, t.model.ID.String())
	if err != nil {
		return nil, err
	}
	sort.SliceStable(views, func(i, j int) bool { return orderOf(views[i].OrderIndex) < orderOf(views[j].OrderIndex) })

	alloc := newKeyAllocator()
	out := make([]dto.BaseExportView, 0, len(views))
	for _, v := range views {
		view := dto.BaseExportView{
			// Keys only need to be unique in the file; the table key makes them so
			Key:        clipKey(alloc.next("v_"+t.key[2:]+"_", v.Title)),
			Title:      v.Title,
			Type:       v.Type,
			OrderIndex: orderOf(v.OrderIndex),
		}
		if v.Description != nil {
			view.Description = *v.Description
		}
		if v.Meta != nil {
			view.Meta = rewriteMap(*v.Meta, idMap)
			rewritten := rewriteMap(*v.Meta, idMap)
			if len(skippedKeys) > 0 {
				rewritten = cleanViewMeta(rewritten, skippedKeys)
			}
			view.Meta = rewritten
		}
		out = append(out, view)
	}
	return out, nil
}

func filterTableResponses(responses []dto.TableResponse, filters []string) []dto.TableResponse {
	allowed := make(map[string]struct{}, len(filters))
	for _, f := range filters {
		f = strings.ToLower(strings.TrimSpace(f))
		if f != "" {
			allowed[f] = struct{}{}
		}
	}
	if len(allowed) == 0 {
		return responses
	}
	var filtered []dto.TableResponse
	for _, r := range responses {
		id := strings.ToLower(r.Model.ID.String())
		title := strings.ToLower(r.Model.Title)
		alias := strings.ToLower(r.Model.Alias)
		matched := false
		for a := range allowed {
			if a == id || a == title || a == alias || strings.TrimPrefix(a, "t_") == title {
				matched = true
				break
			}
		}
		if matched {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func cleanViewMeta(meta map[string]interface{}, skippedKeys map[string]struct{}) map[string]interface{} {
	if meta == nil || len(skippedKeys) == 0 {
		return meta
	}
	out := make(map[string]interface{}, len(meta))
	for k, v := range meta {
		switch k {
		case "filters", "sorts", "groups":
			if list, ok := v.([]interface{}); ok {
				var filtered []interface{}
				for _, item := range list {
					if m, isMap := item.(map[string]interface{}); isMap {
						col := idString(m["column"])
						if _, skip := skippedKeys[col]; skip {
							continue
						}
					}
					filtered = append(filtered, item)
				}
				out[k] = filtered
				continue
			}
		case "fieldConfig":
			if list, ok := v.([]interface{}); ok {
				var filtered []interface{}
				for _, item := range list {
					if m, isMap := item.(map[string]interface{}); isMap {
						id := idString(m["id"])
						if _, skip := skippedKeys[id]; skip {
							continue
						}
					}
					filtered = append(filtered, item)
				}
				out[k] = filtered
				continue
			}
		}
		if str, isStr := v.(string); isStr {
			if _, skip := skippedKeys[str]; skip {
				continue
			}
		}
		out[k] = v
	}
	return out
}

// pkColumn returns the column whose values link cells of this table hold.
func (s *baseTransferService) pkColumn(ctx context.Context, schemaName, modelID string, warnings *[]string) string {
	if s.primaryKeyService == nil {
		return constant.PrimaryKeyIDColumn
	}
	pk, err := s.primaryKeyService.Get(ctx, schemaName, modelID)
	if err != nil || len(pk.ColumnNames) == 0 {
		return constant.PrimaryKeyIDColumn
	}
	if len(pk.ColumnNames) > 1 {
		*warnings = append(*warnings, fmt.Sprintf("table %s has a composite primary key; its link cells are read by id", modelID))
		return constant.PrimaryKeyIDColumn
	}
	return pk.ColumnNames[0]
}

func (s *baseTransferService) exportData(ctx context.Context, schemaName string, tables []*exportTable, skipped, asText map[string]struct{}, withFiles bool, warnings *[]string) (*dto.BaseExportData, error) {
	data := &dto.BaseExportData{
		RowCounts: map[string]int{},
		Tables:    map[string]dto.BaseExportTableRow{},
	}

	// Pass 1: read every table and give each record a _rid, so link cells can be turned into _rid in pass 2.
	records := map[string][]map[string]interface{}{}
	rids := map[string]map[string]string{} // model id -> primary-key value -> _rid
	rowRID := map[string][]string{}        // model id -> _rid per record, in record order
	for ti, t := range tables {
		reportFraction(ctx, 15, 40, ti, len(tables))
		modelID := t.model.ID.String()
		res, err := s.tableService.GetAllRecords(ctx, schemaName, modelID)
		if err != nil {
			return nil, err
		}
		recs := res.Records
		sort.SliceStable(recs, func(i, j int) bool { return numericID(recs[i]["id"]) < numericID(recs[j]["id"]) })
		records[modelID] = recs

		pk := s.pkColumn(ctx, schemaName, modelID, warnings)
		rids[modelID] = map[string]string{}
		for _, rec := range recs {
			rid := fmt.Sprintf("%s_%s", t.key, cellKey(rec["id"]))
			rowRID[modelID] = append(rowRID[modelID], rid)
			rids[modelID][cellKey(rec[pk])] = rid
			// A key may also be written by id when the key column holds nothing useful
			if _, ok := rids[modelID][cellKey(rec["id"])]; !ok {
				rids[modelID][cellKey(rec["id"])] = rid
			}
		}
	}

	// The display text of the rows that the links exported as text pointed at
	linkTexts, err := s.loadLinkTexts(ctx, schemaName, tables, asText, warnings)
	if err != nil {
		return nil, err
	}

	// Pass 2: values by field key.
	attachments := newAttachmentCollector()
	attachmentWarned := map[string]bool{}
	for ti, t := range tables {
		reportFraction(ctx, 40, 65, ti, len(tables))
		modelID := t.model.ID.String()
		rows := make([]dto.BaseExportRow, 0, len(records[modelID]))
		// the cells of a row are written in the order of the fields of the table
		valueOrder := make([]string, 0, len(t.columns))
		for _, c := range t.columns {
			if !boolOf(c.System) {
				valueOrder = append(valueOrder, t.fieldKeys[c.ID.String()])
			}
		}
		for i, rec := range records[modelID] {
			values := map[string]interface{}{}
			for _, c := range t.columns {
				id := c.ID.String()
				if _, skip := skipped[id]; skip || boolOf(c.System) {
					continue
				}
				if _, text := asText[id]; text {
					// a link, lookup or rollup exported as text: the text of what it showed
					var cell string
					if c.UIDT == uidtLinks {
						cell = linkCellText(rec[c.ColumnName], linkTexts[id])
					} else {
						cell = valueText(rec[c.ColumnName])
					}
					if cell != "" {
						values[t.fieldKeys[id]] = cell
					}
					continue
				}
				if c.UIDT == uidtLookup {
					continue
				}
				val := rec[c.ColumnName]
				if val == nil {
					continue
				}
				switch c.UIDT {
				case uidtAttachment:
					if !withFiles {
						if !attachmentWarned[id] {
							attachmentWarned[id] = true
							*warnings = append(*warnings, fmt.Sprintf("attachment files of %s.%s are not exported; the cells are left empty", t.model.Title, c.Title))
						}
						continue
					}
					var refs []string
					for _, item := range attachmentItems(val) {
						ref, reason := attachments.add(item)
						if reason != "" {
							*warnings = append(*warnings, fmt.Sprintf("attachment of %s.%s skipped: %s", t.model.Title, c.Title, reason))
							continue
						}
						refs = append(refs, ref)
					}
					if len(refs) > 0 {
						values[t.fieldKeys[id]] = refs
					}
					continue
				case constant.UIDTUUID:
					// An auto uuid field is filled in by the system for every new row, and the app refuses a written value
					if isAutoUUID(c.Meta) {
						continue
					}
					values[t.fieldKeys[id]] = val
				case uidtLinks:
					target := nestedString(c.Meta, "relation", "with")
					linked := linkedRIDs(val, rids[target])
					if len(linked) == 0 {
						continue
					}
					values[t.fieldKeys[id]] = linked
				default:
					values[t.fieldKeys[id]] = val
				}
			}
			row := dto.BaseExportRow{RID: rowRID[modelID][i], Values: values}
			row.OrderBy(valueOrder)
			rows = append(rows, row)
		}
		data.Tables[t.key] = dto.BaseExportTableRow{Rows: rows}
		data.RowCounts[t.key] = len(rows)
	}
	data.Attachments = attachments.files
	return data, nil
}

// linkedRIDs turns the key values of a link cell into _rid; keys with no matching record are dropped.
func linkedRIDs(cell interface{}, byKey map[string]string) []string {
	var out []string
	for _, item := range toList(cell) {
		if obj, ok := item.(map[string]interface{}); ok {
			item = obj["id"]
		}
		if rid, ok := byKey[cellKey(item)]; ok {
			out = append(out, rid)
		}
	}
	return out
}

func numericID(v interface{}) float64 {
	f, err := strconv.ParseFloat(cellKey(v), 64)
	if err != nil {
		return 0
	}
	return f
}

// clipKey keeps a key within the length the file format allows.
func clipKey(key string) string {
	if len(key) > 63 {
		return key[:63]
	}
	return key
}

// exportCover describes the cover image of the base. The file itself is packed into the zip later.
func (s *baseTransferService) exportCover(ctx context.Context, schemaName, image string, withFiles bool, warnings *[]string) *dto.BaseExportCover {
	if image == "" {
		return nil
	}
	if !withFiles {
		*warnings = append(*warnings, "cover image not exported: ask for attachments (a zip) to include it")
		return nil
	}
	if s.assets == nil {
		*warnings = append(*warnings, "cover image not exported: the asset service is not configured")
		return nil
	}
	asset, err := s.assets.GetAssetByURL(ctx, schemaName, image)
	if err != nil || asset.BasePath == "" {
		*warnings = append(*warnings, "cover image not exported: its file could not be found")
		return nil
	}
	if asset.Size > MaxAttachmentBytes {
		*warnings = append(*warnings, fmt.Sprintf("cover image not exported: it is larger than %d MB", MaxAttachmentBytes>>20))
		return nil
	}
	name := asset.Title
	if name == "" {
		name = "cover"
	}
	return &dto.BaseExportCover{
		Filename:    name,
		ContentType: asset.MimeType,
		Size:        asset.Size,
		Path:        zipAttachmentDir + "cover_" + safeFileName(name),
		StoragePath: asset.BasePath,
	}
}

// exportPrimaryKey returns the key fields of a table that has a custom primary key, as field keys in key order.
// A table with the default Id key returns nothing.
func (s *baseTransferService) exportPrimaryKey(ctx context.Context, schemaName string, t *exportTable, skipped map[string]struct{}, warnings *[]string) []string {
	if s.primaryKeyService == nil {
		return nil
	}
	pk, err := s.primaryKeyService.Get(ctx, schemaName, t.model.ID.String())
	if err != nil {
		*warnings = append(*warnings, fmt.Sprintf("primary key of table %s not exported: %v", t.model.Title, err))
		return nil
	}
	if len(pk.ColumnNames) == 0 || (len(pk.ColumnNames) == 1 && pk.ColumnNames[0] == constant.PrimaryKeyIDColumn) {
		return nil
	}
	keys := make([]string, 0, len(pk.ColumnIDs))
	for _, id := range pk.ColumnIDs {
		key, ok := t.fieldKeys[id]
		if _, isSkipped := skipped[id]; !ok || isSkipped {
			*warnings = append(*warnings, fmt.Sprintf("primary key of table %s not exported: it uses a field that is not exported", t.model.Title))
			return nil
		}
		keys = append(keys, key)
	}
	return keys
}

// exportConstraints reads the unique and check constraints of a table. Column names inside a check condition are
// replaced by {field_key}, because the generated column names differ in the new base.
func (s *baseTransferService) exportConstraints(ctx context.Context, schemaName string, t *exportTable, skipped map[string]struct{}, warnings *[]string) []dto.BaseExportConstraint {
	if s.constraints == nil {
		return nil
	}
	names := map[string]string{} // column name -> field key
	for _, c := range t.columns {
		names[c.ColumnName] = t.fieldKeys[c.ID.String()]
	}
	var out []dto.BaseExportConstraint
	for _, kind := range []string{"unique", "check"} {
		list, err := s.constraints.List(ctx, schemaName, t.model.ID.String(), kind)
		if err != nil {
			*warnings = append(*warnings, fmt.Sprintf("%s constraints of table %s not exported: %v", kind, t.model.Title, err))
			continue
		}
		for _, c := range list {
			con, reason := exportConstraint(kind, c, t, names, skipped)
			if reason != "" {
				*warnings = append(*warnings, fmt.Sprintf("constraint %q of table %s not exported: %s", c.Title, t.model.Title, reason))
				continue
			}
			out = append(out, con)
		}
	}
	return out
}

func exportConstraint(kind string, c dto.TableConstraintResponse, t *exportTable, names map[string]string, skipped map[string]struct{}) (dto.BaseExportConstraint, string) {
	if c.Status != dto.ConstraintStatusOK {
		return dto.BaseExportConstraint{}, "it is missing in the database"
	}
	con := dto.BaseExportConstraint{Type: kind, Title: c.Title}
	for _, col := range c.Columns {
		key, ok := t.fieldKeys[col.ID]
		if _, isSkipped := skipped[col.ID]; !ok || isSkipped {
			return dto.BaseExportConstraint{}, "it uses a field that is not exported"
		}
		con.FieldKeys = append(con.FieldKeys, key)
	}
	if kind == "check" {
		con.Category = c.Category
		con.Condition = tokenizeCondition(c.Condition, names)
	}
	return con, ""
}

// isAutoUUID reports whether a uuid field gets its value from the system (idGeneration mode "auto").
func isAutoUUID(meta map[string]interface{}) bool {
	cfg, ok := dto.IDGenerationFromMeta(meta)
	return ok && cfg.Mode == constant.IDModeAuto
}
