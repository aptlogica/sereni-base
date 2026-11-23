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
	"time"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
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
	tableResponses, err := s.selectedTableResponses(ctx, schemaName, baseID, opts)
	if err != nil {
		return nil, err
	}

	reportProgress(ctx, 2)
	file := newExportFile(base)

	tables, idMap, err := s.loadExportTables(ctx, schemaName, tableResponses)
	if err != nil {
		return nil, err
	}
	reportProgress(ctx, 8)
	warnings := &file.Skipped
	skipped, asText := s.skippedColumns(tables, opts.LinksAsText, warnings)
	// Fields that a view, a trigger, a key or a constraint cannot use as they were: left out, or turned into text
	unusable := unionOf(skipped, asText)

	skippedKeys := skippedKeysOf(unusable, idMap)

	withFiles := s.exportsFiles(opts, warnings)
	file.Base.Cover = s.exportCover(ctx, schemaName, base.Image, withFiles, warnings)

	automations, err := s.optionalAutomations(ctx, schemaName, tables, unusable, opts, warnings)
	if err != nil {
		return nil, err
	}

	scope := exportScope{
		schemaName: schemaName, idMap: idMap, skipped: skipped, asText: asText,
		unusable: unusable, skippedKeys: skippedKeys, automations: automations, warnings: warnings,
	}
	for ti, t := range tables {
		reportFraction(ctx, 8, 15, ti, len(tables))
		table, err := s.exportFullTable(ctx, scope, t)
		if err != nil {
			return nil, err
		}
		file.Schema.Tables = append(file.Schema.Tables, table)
	}

	if opts.IncludeSystemColumns && !opts.IncludeData {
		*warnings = append(*warnings, "source columns not exported: they are values of the rows, so they need the data")
	}
	if opts.IncludeData {
		file.Data, err = s.exportData(ctx, schemaName, tables, exportDataOptions{
			skipped: skipped, asText: asText, filters: resolveFilters(tableResponses, opts.Filters),
			withFiles: withFiles, systemColumns: opts.IncludeSystemColumns, warnings: warnings,
		})
		if err != nil {
			return nil, err
		}
	}
	reportProgress(ctx, 65)
	// What was left out is only logged: the file and the response carry nothing about the export run
	for _, w := range file.Skipped {
		logger.Get().Warn().Str("base_id", baseID).Msg("base export: " + w)
	}
	return file, nil
}

// skippedKeysOf turns the ids of the fields that cannot be used as they were into their keys in the file
func skippedKeysOf(unusable map[string]struct{}, idMap map[string]string) map[string]struct{} {
	skippedKeys := map[string]struct{}{}
	for id := range unusable {
		if key, ok := idMap[id]; ok {
			skippedKeys[key] = struct{}{}
		}
	}
	return skippedKeys
}

// exportsFiles tells whether files travel in the export: they travel only in a zip (attachments, cover image), and
// only when storage can be read
func (s *baseTransferService) exportsFiles(opts dto.ExportBaseOptions, warnings *[]string) bool {
	withFiles := opts.IncludeAttachments && s.storage != nil
	if opts.IncludeAttachments && !withFiles {
		*warnings = append(*warnings, "files not exported: no storage provider is configured")
	}
	return withFiles
}

// optionalAutomations exports the functions and triggers of the tables when the options ask for them
func (s *baseTransferService) optionalAutomations(ctx context.Context, schemaName string, tables []*exportTable, unusable map[string]struct{}, opts dto.ExportBaseOptions, warnings *[]string) (map[string]*exportedAutomations, error) {
	if !opts.IncludeAutomations {
		return nil, nil
	}
	return s.exportAutomations(ctx, schemaName, tables, unusable, warnings)
}

// selectedTableResponses reads the tables of a base, keeping only the ones the options name
func (s *baseTransferService) selectedTableResponses(ctx context.Context, schemaName, baseID string, opts dto.ExportBaseOptions) ([]dto.TableResponse, error) {
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
	return tableResponses, nil
}

// newExportFile starts the file of a base: its title, settings, meta and description
func newExportFile(base tenant.Base) *dto.BaseExportFile {
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
	return file
}

// exportScope is what the export of every table needs besides the table itself
type exportScope struct {
	schemaName  string
	idMap       map[string]string
	skipped     map[string]struct{}
	asText      map[string]struct{}
	unusable    map[string]struct{}
	skippedKeys map[string]struct{}
	automations map[string]*exportedAutomations
	warnings    *[]string
}

// exportFullTable exports the schema of one table with its automations, keys, constraints and views
func (s *baseTransferService) exportFullTable(ctx context.Context, scope exportScope, t *exportTable) (dto.BaseExportTable, error) {
	table := s.exportTableSchema(t, scope.idMap, scope.skipped, scope.asText)
	if a := scope.automations[t.key]; a != nil {
		table.Functions, table.Triggers = a.functions, a.triggers
	}
	table.PrimaryKey = s.exportPrimaryKey(ctx, scope.schemaName, t, scope.unusable, scope.warnings)
	table.Constraints = s.exportConstraints(ctx, scope.schemaName, t, scope.unusable, scope.warnings)
	views, err := s.exportViews(ctx, scope.schemaName, t, scope.idMap, scope.skippedKeys)
	if err != nil {
		return table, err
	}
	table.Views = views
	return table, nil
}

// loadExportTables reads tables and columns, gives every table, column and relation a key, and returns the
// id -> "@key" map used to rewrite ids inside meta.
func (s *baseTransferService) loadExportTables(ctx context.Context, schemaName string, responses []dto.TableResponse) ([]*exportTable, map[string]string, error) {
	models := sortedModels(responses)

	alloc := newKeyAllocator()
	idMap := map[string]string{}
	tables := make([]*exportTable, 0, len(models))

	for _, m := range models {
		cols, err := s.tableService.GetColumnsByModelID(ctx, schemaName, m.ID.String())
		if err != nil {
			return nil, nil, err
		}
		t := &exportTable{key: alloc.next("t_", m.Title), model: m, columns: liveColumns(cols), fieldKeys: map[string]string{}}
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
			nameRelations(t, role, idMap, alloc)
		}
	}
	return tables, idMap, nil
}

// sortedModels takes the models of the tables, by order_index and then title
func sortedModels(responses []dto.TableResponse) []dto.ModelResponse {
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
	return models
}

// liveColumns keeps the columns that are not deleted, by order_index
func liveColumns(cols []dto.ColumnResponse) []dto.ColumnResponse {
	live := make([]dto.ColumnResponse, 0, len(cols))
	for _, c := range cols {
		if !boolOf(c.Deleted) {
			live = append(live, c)
		}
	}
	sort.SliceStable(live, func(i, j int) bool { return orderOf(live[i].OrderIndex) < orderOf(live[j].OrderIndex) })
	return live
}

// nameRelations gives the relations of the links of a table a key, for the links of one entity role
func nameRelations(t *exportTable, role string, idMap map[string]string, alloc *keyAllocator) {
	for _, c := range t.columns {
		rid := idString(c.Meta["relation_id"])
		if c.UIDT != uidtLinks || rid == "" || (role != "" && idString(c.Meta["entity_role"]) != role) {
			continue
		}
		if _, ok := idMap[rid]; ok {
			continue
		}
		name := "relation"
		if role != "" {
			name = strings.TrimPrefix(t.key, "t_") + " " + c.Title
		}
		idMap[rid] = dto.BaseExportRefPrefix + clipKey(alloc.next("r_", name))
	}
}

// skippedColumns finds fields that import cannot create (unsupported type, or built on such a field) and says so.
// A link to a table that is not exported cannot stay a link: it is returned in the second set, with the lookups and
// rollups built on it, and is exported as a text field.
func (s *baseTransferService) skippedColumns(tables []*exportTable, linksAsText bool, warnings *[]string) (map[string]struct{}, map[string]struct{}) {
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
			markLinkAsText(c, titles[c.ID.String()], linksAsText, tableIDs, asText, warnings)
		}
	}
	// A field built on a skipped field (e.g. a lookup of a rollup) cannot be rebuilt either
	for changed := true; changed; {
		changed = false
		for _, t := range tables {
			if propagateSkips(t, titles, skipped, asText, warnings) {
				changed = true
			}
		}
	}
	return skipped, asText
}

// markLinkAsText puts a link in asText when its format cannot hold links or its linked table is not exported
func markLinkAsText(c dto.ColumnResponse, title string, linksAsText bool, tableIDs, asText map[string]struct{}, warnings *[]string) {
	if c.UIDT != uidtLinks {
		return
	}
	withID := idString(nestedString(c.Meta, "relation", "with"))
	if linksAsText {
		// a format that cannot hold links (csv): every link is text
		asText[c.ID.String()] = struct{}{}
		*warnings = append(*warnings, fmt.Sprintf("field %s exported as text: this format cannot hold links", title))
	} else if withID != "" {
		if _, ok := tableIDs[withID]; !ok {
			asText[c.ID.String()] = struct{}{}
			*warnings = append(*warnings, fmt.Sprintf("field %s exported as text: linked table is not exported", title))
		}
	}
}

// textLinkRelations are the relations of the links of a table that become text: a lookup names its link by field id
// or by relation
func textLinkRelations(t *exportTable, asText map[string]struct{}) map[string]struct{} {
	textRelations := map[string]struct{}{}
	for _, c := range t.columns {
		if _, ok := asText[c.ID.String()]; ok && c.UIDT == uidtLinks {
			if rid := idString(c.Meta["relation_id"]); rid != "" {
				textRelations[rid] = struct{}{}
			}
		}
	}
	return textRelations
}

// propagateSkips skips or turns into text the fields of a table built on fields that are skipped or text; it reports
// whether it changed anything
func propagateSkips(t *exportTable, titles map[string]string, skipped, asText map[string]struct{}, warnings *[]string) bool {
	changed := false
	textRelations := textLinkRelations(t, asText)
	for _, c := range t.columns {
		id := c.ID.String()
		if _, done := skipped[id]; done || boolOf(c.System) {
			continue
		}
		if _, done := asText[id]; done {
			continue
		}
		if propagateSkip(c, titles[id], textRelations, skipped, asText, warnings) {
			changed = true
		}
	}
	return changed
}

// propagateSkip decides one field that may depend on a skipped field or on a link that is exported as text
func propagateSkip(c dto.ColumnResponse, title string, textRelations, skipped, asText map[string]struct{}, warnings *[]string) bool {
	id := c.ID.String()
	meta := normalizeMap(c.Meta)
	if containsString(meta, skipped) {
		skipped[id] = struct{}{}
		*warnings = append(*warnings, fmt.Sprintf("field %s not exported: it depends on a field that is not exported", title))
		return true
	}
	if _, throughText := textRelations[idString(c.Meta["relation_id"])]; isLookupLike(c.UIDT) && throughText && !containsString(meta, asText) {
		asText[id] = struct{}{}
		*warnings = append(*warnings, fmt.Sprintf("field %s exported as text: it reads through a link that is exported as text", title))
		return true
	}
	if containsString(meta, asText) {
		// a lookup or rollup of a link that becomes text becomes text; anything else built on it cannot
		if isLookupLike(c.UIDT) {
			asText[id] = struct{}{}
			*warnings = append(*warnings, fmt.Sprintf("field %s exported as text: it reads through a link that is exported as text", title))
		} else {
			skipped[id] = struct{}{}
			*warnings = append(*warnings, fmt.Sprintf("field %s not exported: it depends on a field that is exported as text", title))
		}
		return true
	}
	return false
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
		out.Fields = append(out.Fields, exportField(t, c, idMap, asText))
	}
	return out
}

// exportField describes one column as a field of the file
func exportField(t *exportTable, c dto.ColumnResponse, idMap map[string]string, asText map[string]struct{}) dto.BaseExportField {
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
	return field
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

// cleanViewMeta leaves out of a view's meta what points at a field that is not exported
func cleanViewMeta(meta map[string]interface{}, skippedKeys map[string]struct{}) map[string]interface{} {
	if meta == nil || len(skippedKeys) == 0 {
		return meta
	}
	out := make(map[string]interface{}, len(meta))
	for k, v := range meta {
		if list, ok := cleanViewList(k, v, skippedKeys); ok {
			out[k] = list
			continue
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

// viewListKey is the item key that names a field, for the lists of a view's meta that hold fields
func viewListKey(k string) (string, bool) {
	switch k {
	case "filters", "sorts", "groups":
		return "column", true
	case "fieldConfig":
		return "id", true
	}
	return "", false
}

// cleanViewList drops the items of a filters, sorts, groups or fieldConfig list that name a skipped field; ok is
// false when the entry is not such a list
func cleanViewList(k string, v interface{}, skippedKeys map[string]struct{}) ([]interface{}, bool) {
	itemKey, isList := viewListKey(k)
	if !isList {
		return nil, false
	}
	list, ok := v.([]interface{})
	if !ok {
		return nil, false
	}
	var filtered []interface{}
	for _, item := range list {
		if m, isMap := item.(map[string]interface{}); isMap {
			if _, skip := skippedKeys[idString(m[itemKey])]; skip {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	return filtered, true
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

// exportDataOptions are what exportData needs besides the tables
type exportDataOptions struct {
	skipped       map[string]struct{}
	asText        map[string]struct{}
	filters       map[string]dto.ExportTableFilter
	withFiles     bool
	systemColumns bool
	warnings      *[]string
}

// exportRecords are the records of the tables read in pass 1 of exportData
type exportRecords struct {
	records map[string][]map[string]interface{}
	rids    map[string]map[string]string // model id -> primary-key value -> _rid
	rowRID  map[string][]string          // model id -> _rid per record, in record order
}

func (s *baseTransferService) exportData(ctx context.Context, schemaName string, tables []*exportTable, opts exportDataOptions) (*dto.BaseExportData, error) {
	data := &dto.BaseExportData{
		RowCounts: map[string]int{},
		Tables:    map[string]dto.BaseExportTableRow{},
	}

	// Pass 1: read every table and give each record a _rid, so link cells can be turned into _rid in pass 2.
	recs, err := s.readExportRecords(ctx, schemaName, tables, opts)
	if err != nil {
		return nil, err
	}

	// The display text of the rows that the links exported as text pointed at
	linkTexts, err := s.loadLinkTexts(ctx, schemaName, tables, opts.asText, opts.warnings)
	if err != nil {
		return nil, err
	}

	// Pass 2: values by field key.
	cells := &cellWriter{
		opts: opts, linkTexts: linkTexts, rids: recs.rids,
		attachments: newAttachmentCollector(), attachmentWarned: map[string]bool{},
	}
	for ti, t := range tables {
		reportFraction(ctx, 40, 65, ti, len(tables))
		modelID := t.model.ID.String()
		rows := cells.tableRows(t, recs.records[modelID], recs.rowRID[modelID])
		data.Tables[t.key] = dto.BaseExportTableRow{Rows: rows}
		data.RowCounts[t.key] = len(rows)
	}
	data.Attachments = cells.attachments.files
	return data, nil
}

// readExportRecords is pass 1 of exportData: it reads every table, sorted and filtered, and gives each record a _rid
func (s *baseTransferService) readExportRecords(ctx context.Context, schemaName string, tables []*exportTable, opts exportDataOptions) (*exportRecords, error) {
	out := &exportRecords{
		records: map[string][]map[string]interface{}{},
		rids:    map[string]map[string]string{},
		rowRID:  map[string][]string{},
	}
	for ti, t := range tables {
		reportFraction(ctx, 15, 40, ti, len(tables))
		modelID := t.model.ID.String()
		res, err := s.tableService.GetAllRecords(ctx, schemaName, modelID)
		if err != nil {
			return nil, err
		}
		recs := res.Records
		sort.SliceStable(recs, func(i, j int) bool { return numericID(recs[i]["id"]) < numericID(recs[j]["id"]) })
		// A row filter keeps only the rows that match. It runs before the rows get a _rid, so links to a row that was
		// filtered out are dropped, and files, row counts and the rest only see the rows that stay.
		if f, ok := opts.filters[modelID]; ok {
			recs = filterRecords(recs, t.columns, f)
		}
		out.records[modelID] = recs

		pk := s.pkColumn(ctx, schemaName, modelID, opts.warnings)
		out.rids[modelID] = map[string]string{}
		for _, rec := range recs {
			rid := fmt.Sprintf("%s_%s", t.key, cellKey(rec["id"]))
			out.rowRID[modelID] = append(out.rowRID[modelID], rid)
			out.rids[modelID][cellKey(rec[pk])] = rid
			// A key may also be written by id when the key column holds nothing useful
			if _, ok := out.rids[modelID][cellKey(rec["id"])]; !ok {
				out.rids[modelID][cellKey(rec["id"])] = rid
			}
		}
	}
	return out, nil
}

// cellWriter turns the records of a table into rows of values by field key (pass 2 of exportData)
type cellWriter struct {
	opts             exportDataOptions
	linkTexts        map[string]map[string]string
	rids             map[string]map[string]string
	attachments      *attachmentCollector
	attachmentWarned map[string]bool
}

// tableRows writes the rows of one table; the cells of a row are written in the order of the fields of the table
func (w *cellWriter) tableRows(t *exportTable, records []map[string]interface{}, rowRIDs []string) []dto.BaseExportRow {
	rows := make([]dto.BaseExportRow, 0, len(records))
	valueOrder := make([]string, 0, len(t.columns))
	for _, c := range t.columns {
		if !boolOf(c.System) || (w.opts.systemColumns && isSourceColumn(c.ColumnName)) {
			valueOrder = append(valueOrder, t.fieldKeys[c.ID.String()])
		}
	}
	for i, rec := range records {
		values := map[string]interface{}{}
		for _, c := range t.columns {
			w.writeCell(t, c, rec, values)
		}
		row := dto.BaseExportRow{RID: rowRIDs[i], Values: values}
		row.OrderBy(valueOrder)
		rows = append(rows, row)
	}
	return rows
}

// writeCell puts the value of one column of a record in values, unless the column has nothing to write
func (w *cellWriter) writeCell(t *exportTable, c dto.ColumnResponse, rec, values map[string]interface{}) {
	id := c.ID.String()
	if boolOf(c.System) {
		w.writeSourceCell(t, c, rec, values)
		return
	}
	if _, skip := w.opts.skipped[id]; skip {
		return
	}
	if _, text := w.opts.asText[id]; text {
		w.writeTextCell(t, c, rec, values)
		return
	}
	if c.UIDT == uidtLookup {
		return
	}
	val := rec[c.ColumnName]
	if val == nil {
		return
	}
	w.writeValue(t, c, val, values)
}

// writeSourceCell writes a system column: the source columns are written only when asked for; the other system
// fields never
func (w *cellWriter) writeSourceCell(t *exportTable, c dto.ColumnResponse, rec, values map[string]interface{}) {
	if w.opts.systemColumns && isSourceColumn(c.ColumnName) {
		if v := sourceValue(c.ColumnName, rec[c.ColumnName]); v != nil {
			values[t.fieldKeys[c.ID.String()]] = v
		}
	}
}

// writeTextCell writes a link, lookup or rollup exported as text: the text of what it showed
func (w *cellWriter) writeTextCell(t *exportTable, c dto.ColumnResponse, rec, values map[string]interface{}) {
	id := c.ID.String()
	var cell string
	if c.UIDT == uidtLinks {
		cell = linkCellText(rec[c.ColumnName], w.linkTexts[id])
	} else {
		cell = valueText(rec[c.ColumnName])
	}
	if cell != "" {
		values[t.fieldKeys[id]] = cell
	}
}

// writeValue writes a value that is not empty, by the type of its column
func (w *cellWriter) writeValue(t *exportTable, c dto.ColumnResponse, val interface{}, values map[string]interface{}) {
	id := c.ID.String()
	switch c.UIDT {
	case uidtAttachment:
		w.writeAttachments(t, c, val, values)
	case constant.UIDTUUID:
		// An auto uuid field is filled in by the system for every new row, and the app refuses a written value
		if isAutoUUID(c.Meta) {
			return
		}
		values[t.fieldKeys[id]] = val
	case uidtLinks:
		target := nestedString(c.Meta, "relation", "with")
		linked := linkedRIDs(val, w.rids[target])
		if len(linked) == 0 {
			return
		}
		values[t.fieldKeys[id]] = linked
	default:
		values[t.fieldKeys[id]] = val
	}
}

// writeAttachments puts the refs of the files of an attachment cell in values; without files it warns once per field
func (w *cellWriter) writeAttachments(t *exportTable, c dto.ColumnResponse, val interface{}, values map[string]interface{}) {
	id := c.ID.String()
	warnings := w.opts.warnings
	if !w.opts.withFiles {
		if !w.attachmentWarned[id] {
			w.attachmentWarned[id] = true
			*warnings = append(*warnings, fmt.Sprintf("attachment files of %s.%s are not exported; the cells are left empty", t.model.Title, c.Title))
		}
		return
	}
	var refs []string
	for _, item := range attachmentItems(val) {
		ref, reason := w.attachments.add(item)
		if reason != "" {
			*warnings = append(*warnings, fmt.Sprintf("attachment of %s.%s skipped: %s", t.model.Title, c.Title, reason))
			continue
		}
		refs = append(refs, ref)
	}
	if len(refs) > 0 {
		values[t.fieldKeys[id]] = refs
	}
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

// sourceColumns are the system columns that "include source columns" writes and import restores.
var sourceColumns = map[string]struct{}{
	"id": {}, "created_time": {}, "last_modified_time": {}, "created_by": {}, "last_modified_by": {},
}

func isSourceColumn(columnName string) bool {
	_, ok := sourceColumns[columnName]
	return ok
}

// sourceValue turns the value of a source column into what the file holds: the id as a number, a time as RFC 3339
// in UTC, a user as text. It returns nil for an empty value, which is then left out.
func sourceValue(columnName string, v interface{}) interface{} {
	switch x := v.(type) {
	case nil:
		return nil
	case time.Time:
		return x.UTC().Format(time.RFC3339Nano)
	}
	switch columnName {
	case "id":
		if n := numericID(v); n > 0 {
			return int64(n)
		}
		return nil
	case "created_time", "last_modified_time":
		// the database may give the time as text, with or without a zone: the file always holds RFC 3339 in UTC
		if ts, ok := parseSourceTime(valueText(v)); ok {
			return ts.Format(time.RFC3339Nano)
		}
		return nil
	}
	if text := valueText(v); text != "" {
		return text
	}
	return nil
}

// sourceTimeLayouts are the forms a time may have: RFC 3339, or the text of a TIMESTAMP column (no zone, UTC).
var sourceTimeLayouts = []string{
	time.RFC3339Nano,
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02",
}

// parseSourceTime reads the time of a source column and returns it in UTC.
func parseSourceTime(text string) (time.Time, bool) {
	text = strings.TrimSpace(text)
	for _, layout := range sourceTimeLayouts {
		if ts, err := time.Parse(layout, text); err == nil {
			return ts.UTC(), true
		}
	}
	return time.Time{}, false
}
