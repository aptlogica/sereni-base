// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
	"github.com/google/uuid"
)

// rowRef is where an imported row ended up.
type rowRef struct {
	table string // table key
	id    int
}

// importRun holds the state of one import: what was created so far and the map from file keys to new ids.
type importRun struct {
	s           *baseTransferService
	ctx         context.Context
	schema      string
	workspaceID string
	userID      string
	plan        *importPlan
	includeData bool
	// systemColumns restores the source columns (id, created / modified time and user) from the file
	systemColumns bool
	// titleGiven is true when the request names the new base; a name that is already used is then refused instead of
	// being changed to "name (2)"
	titleGiven bool

	baseID uuid.UUID
	// "@key" -> new id, for tables, fields and relations
	ids map[string]string
	// table key -> created table
	tables map[string]dto.TableResponse
	// field key -> created column
	columns map[string]dto.ColumnResponse
	// table key -> the default grid view the new table came with, until a file view has used it
	defaultView map[string]*dto.ViewResponse

	// attachments is the zip's files (nil for a plain JSON file); index is the file's list of them by ref
	attachments *dto.Attachments
	index       map[string]dto.BaseExportAttachment
	// assetIDs are the files uploaded so far, so a failed import can delete them
	assetIDs []string
	// coverAdded is true once the cover image was set, so a failed import can remove it
	coverAdded bool
	// includeAutomations creates the trigger functions and triggers; automationIDs are those created so far, so a failed
	// import can drop them again, and functionNames are the names the functions of the file got in the new base
	includeAutomations bool
	automationIDs      []string
	// autoUUIDs are the auto uuid fields created as manual ones while the rows are inserted (see createPlainFields)
	autoUUIDs     []pendingAutoUUID
	functionNames map[string]string

	// skipFailed leaves out the rows the database refuses instead of failing the import; skipped are those left out,
	// and source says where each row came from, for the error log
	skipFailed bool
	skipped    []skippedRow
	source     *dto.ImportSource

	createdModels []string
	rows          map[string]rowRef
	result        dto.ImportBaseResult
	// progressFrom and progressTo are the percentages the running step spans
	progressFrom, progressTo int
}

// ImportBase validates the file and creates it as a new base in the workspace. On any failure everything
// created so far is removed again.
func (s *baseTransferService) ImportBase(ctx context.Context, schemaName, workspaceID, userID string, file *dto.BaseExportFile, opts dto.ImportBaseOptions) (dto.ImportBaseResult, error) {
	plan, problems := buildPlan(file)
	if len(problems) > 0 {
		return dto.ImportBaseResult{}, &dto.ImportValidationError{Problems: problems}
	}
	// without IncludeAttachments the files of the zip are never read
	if !opts.IncludeAttachments {
		opts.Attachments = nil
	}

	if problems := checkAttachmentFiles(file, opts); len(problems) > 0 {
		return dto.ImportBaseResult{}, &dto.ImportValidationError{Problems: problems}
	}

	run := s.newImportRun(ctx, schemaName, workspaceID, userID, plan, file, opts)
	run.result = dto.ImportBaseResult{
		Title:  firstNonEmpty(opts.Title, file.Base.Title),
		Tables: len(file.Schema.Tables),
		Views:  len(plan.views),
	}
	run.countSchema(file)
	run.describeData(file, opts)
	if functions, triggers := countAutomations(file); functions+triggers > 0 && !opts.IncludeAutomations {
		run.warn("%d trigger function(s) and %d trigger(s) in the file were not created: they run SQL, so they are only created when the request has include_automations=true (a form field or a URL parameter) and the user has the automations permission", functions, triggers)
	}

	if err := run.execute(); err != nil {
		run.rollback()
		return dto.ImportBaseResult{}, err
	}
	run.finishReport()
	return run.result, nil
}

// newImportRun sets up the state of an import of a file that passed validation
func (s *baseTransferService) newImportRun(ctx context.Context, schemaName, workspaceID, userID string, plan *importPlan, file *dto.BaseExportFile, opts dto.ImportBaseOptions) *importRun {
	run := &importRun{
		s: s, ctx: ctx, schema: schemaName, workspaceID: workspaceID, userID: userID,
		plan:          plan,
		includeData:   opts.IncludeData && file.Data != nil,
		systemColumns: opts.IncludeSystemColumns && opts.IncludeData && file.Data != nil,
		titleGiven:    strings.TrimSpace(opts.Title) != "",
		skipFailed:    opts.SkipFailedRows,
		source:        opts.Source,
		ids:           map[string]string{},
		tables:        map[string]dto.TableResponse{},
		columns:       map[string]dto.ColumnResponse{},
		defaultView:   map[string]*dto.ViewResponse{},
		rows:          map[string]rowRef{},
		attachments:   opts.Attachments,
		index:         map[string]dto.BaseExportAttachment{},

		includeAutomations: opts.IncludeAutomations,
		functionNames:      map[string]string{},
	}
	if file.Data != nil {
		for _, a := range file.Data.Attachments {
			run.index[a.Ref] = a
		}
	}
	return run
}

// countSchema counts the constraints and the fields (not the system ones) of the file for the report
func (r *importRun) countSchema(file *dto.BaseExportFile) {
	for _, t := range file.Schema.Tables {
		r.result.Constraints += len(t.Constraints)
		for _, f := range t.Fields {
			if !f.System {
				r.result.Fields++
			}
		}
	}
}

// describeData counts the rows and files of the file for the report, and warns about what is not imported
func (r *importRun) describeData(file *dto.BaseExportFile, opts dto.ImportBaseOptions) {
	if !r.includeData {
		if file.Data != nil {
			r.result.Warnings = append(r.result.Warnings, "rows in the file were not imported")
		}
		return
	}
	for _, t := range file.Data.Tables {
		r.result.Rows += len(t.Rows)
	}
	r.result.Files = len(r.index)
	if len(r.index) > 0 && !opts.IncludeAttachments {
		r.result.Files = 0
		r.warn("attachments were not imported as requested; attachment cells are left empty")
	} else if len(r.index) > 0 && r.attachments == nil {
		r.result.Files = 0
		r.warn("attachment files were not imported: upload the .sbase.zip to import them; attachment cells are left empty")
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func (r *importRun) execute() error {
	// weight is the share of the whole import a step is expected to take, for the progress percentage
	type step struct {
		run    func() error
		weight int
	}
	steps := []step{
		{r.createBase, 2},
		{r.importCover, 2},
		{r.createTables, 8},
		{r.createPlainFields, 10},
		// The key is set before any link exists: the app does not allow a key change once links store its values
		{r.setPrimaryKeys, 2},
		{r.createLinkFields, 6},
		{r.createLookupFields, 4},
		{r.createViews, 4},
	}
	if r.includeData {
		// The auto uuid fields were created as manual fields so the rows could keep their values; now they become auto
		steps = append(steps, step{r.createRows, 50}, step{r.makeUUIDsAuto, 1}, step{r.linkRows, 6})
		if r.systemColumns {
			// last, because linking rows changes their modified time and user
			steps = append(steps, step{r.restoreSystemColumns, 4})
		}
	}
	// Constraints come last, so the rows being inserted are never held up by a rule
	steps = append(steps, step{r.createConstraints, 3}, step{r.createAutomations, 3})
	total := 0
	for _, st := range steps {
		total += st.weight
	}
	done := 0
	for _, st := range steps {
		r.progressFrom, r.progressTo = done*99/total, (done+st.weight)*99/total
		if err := st.run(); err != nil {
			return err
		}
		done += st.weight
		reportProgress(r.ctx, done*99/total)
	}
	return nil
}

// rollback removes what the failed import created. It runs on a context that ignores cancellation, so a
// dropped connection still cleans up.
func (r *importRun) rollback() {
	if r.baseID == uuid.Nil && len(r.assetIDs) == 0 && len(r.automationIDs) == 0 {
		return
	}
	ctx := context.WithoutCancel(r.ctx)
	r.rollbackAutomations(ctx)
	r.rollbackAssets(ctx)
	r.rollbackTables(ctx)
	if r.baseID == uuid.Nil {
		return
	}
	r.rollbackBase(ctx)
}

// rollbackAutomations drops triggers and functions first, newest first: a function cannot be dropped while a trigger
// still uses it
func (r *importRun) rollbackAutomations(ctx context.Context) {
	lg := logger.Get()
	for i := len(r.automationIDs) - 1; i >= 0; i-- {
		if err := r.s.automations.Delete(ctx, r.schema, r.automationIDs[i]); err != nil {
			lg.Error().Err(err).Str("automation_id", r.automationIDs[i]).Msg("base import rollback: failed to delete trigger or function")
		}
	}
}

func (r *importRun) rollbackAssets(ctx context.Context) {
	lg := logger.Get()
	for _, id := range r.assetIDs {
		if err := r.s.assets.DeleteAsset(ctx, id, r.schema); err != nil {
			lg.Error().Err(err).Str("asset_id", id).Msg("base import rollback: failed to delete attachment")
		}
	}
}

func (r *importRun) rollbackTables(ctx context.Context) {
	lg := logger.Get()
	for i := len(r.createdModels) - 1; i >= 0; i-- {
		if err := r.s.tableService.DeleteTable(ctx, r.schema, r.createdModels[i]); err != nil {
			lg.Error().Err(err).Str("model_id", r.createdModels[i]).Msg("base import rollback: failed to delete table")
		}
	}
}

func (r *importRun) rollbackBase(ctx context.Context) {
	lg := logger.Get()
	if r.coverAdded {
		if _, err := r.s.baseService.RemoveBaseImage(ctx, r.schema, r.baseID.String(), r.userID); err != nil {
			lg.Error().Err(err).Str("base_id", r.baseID.String()).Msg("base import rollback: failed to remove cover image")
		}
	}
	if err := r.s.baseService.DeleteBase(ctx, r.schema, r.baseID.String()); err != nil {
		lg.Error().Err(err).Str("base_id", r.baseID.String()).Msg("base import rollback: failed to delete base")
	}
}

func (r *importRun) warn(format string, args ...interface{}) {
	r.result.Warnings = append(r.result.Warnings, fmt.Sprintf(format, args...))
}

// uniqueBaseTitle returns title, or "title (2)", "title (3)"... when the workspace already has a base with it.
func (r *importRun) uniqueBaseTitle(title string) (string, error) {
	bases, err := r.s.baseService.GetBasesByWorkspace(r.ctx, r.schema, r.workspaceID)
	if err != nil {
		return "", err
	}
	taken := map[string]bool{}
	for _, b := range bases {
		taken[strings.ToLower(b.Title)] = true
	}
	candidate := title
	for n := 2; taken[strings.ToLower(candidate)]; n++ {
		candidate = fmt.Sprintf("%s (%d)", title, n)
	}
	return candidate, nil
}

func (r *importRun) createBase() error {
	title, err := r.uniqueBaseTitle(r.result.Title)
	if err != nil {
		return err
	}
	// a name the user typed must be free; the name stored in the file is made free with "(2)"
	if r.titleGiven && !strings.EqualFold(title, r.result.Title) {
		return app_errors.BaseNameExists
	}
	description := r.plan.file.Base.Description
	base, err := r.s.baseService.CreateBaseWithoutTable(r.ctx, dto.CreateBaseRequest{
		Title:       title,
		Description: &description,
		WorkspaceID: r.workspaceID,
		Meta:        r.plan.file.Base.Meta,
		CreatedBy:   r.userID,
	}, r.schema, r.userID)
	if err != nil {
		return err
	}
	r.baseID = base.ID
	r.result.BaseID = base.ID.String()
	r.result.Title = title
	return nil
}

// importCover sets the cover image from the zip. A cover that cannot be set is only a warning: it is decoration,
// and the app limits its size (800x400) and type.
func (r *importRun) importCover() error {
	cover := r.plan.file.Base.Cover
	if cover == nil {
		return nil
	}
	if r.attachments == nil {
		r.warn("cover image was not imported: upload the .sbase.zip to import it")
		return nil
	}
	data, err := readFile(r.attachments, cover.Filename, cover.Path, cover.SHA256)
	if err != nil {
		return fmt.Errorf("read cover image: %w", err)
	}
	header, cleanup, err := fileHeaderFor(cover.Filename, cover.ContentType, data)
	if err != nil {
		return err
	}
	defer cleanup()
	if _, err := r.s.baseService.AddBaseImage(r.ctx, r.schema, r.baseID.String(), header, r.userID); err != nil {
		r.warn("cover image was not imported: %v", err)
		return nil
	}
	r.coverAdded = true
	r.result.Cover = true
	return nil
}

func (r *importRun) createTables() error {
	for _, t := range r.plan.file.Schema.Tables {
		resp, err := r.s.tableService.CreateTableWithDefaultsImport(r.ctx, dto.CreateTableRequest{
			BaseID:      r.baseID.String(),
			WorkspaceID: r.workspaceID,
			Title:       t.Title,
			Description: t.Description,
			OrderIndex:  t.OrderIndex,
			CreatedBy:   r.userID,
		}, r.schema)
		if err != nil {
			return fmt.Errorf("create table %q: %w", t.Title, err)
		}
		r.registerTable(t, resp)
		r.mapSystemFields(t, resp)
	}
	return nil
}

// registerTable remembers a created table: its model, its id and the default view it came with
func (r *importRun) registerTable(t dto.BaseExportTable, resp dto.TableResponse) {
	modelID := resp.Model.ID.String()
	r.createdModels = append(r.createdModels, modelID)
	r.tables[t.Key] = resp
	r.ids[dto.BaseExportRefPrefix+t.Key] = modelID
	if len(resp.Views) > 0 {
		view := resp.Views[0]
		r.defaultView[t.Key] = &view
	}
}

// mapSystemFields points the file's system field keys at the system fields the new table already has
func (r *importRun) mapSystemFields(t dto.BaseExportTable, resp dto.TableResponse) {
	for _, f := range t.Fields {
		if !f.System {
			continue
		}
		c, found := columnByName(resp.Columns, f.ColumnName)
		if !found {
			r.warn("system field %s of table %q was not found in the new table", f.Title, t.Title)
			continue
		}
		r.columns[f.Key] = c
		r.ids[dto.BaseExportRefPrefix+f.Key] = c.ID.String()
	}
}

// columnByName finds the column with a column name
func columnByName(columns []dto.ColumnResponse, name string) (dto.ColumnResponse, bool) {
	for _, c := range columns {
		if c.ColumnName == name {
			return c, true
		}
	}
	return dto.ColumnResponse{}, false
}

// metaFor rewrites the @keys of a meta object to the new ids.
func (r *importRun) metaFor(meta map[string]interface{}) map[string]interface{} {
	return rewriteMap(meta, r.ids)
}

// hasUnresolved reports whether a meta still holds an @key that has no new id yet.
func hasUnresolved(meta map[string]interface{}) bool {
	refs := map[string]struct{}{}
	collectRefs(meta, refs)
	return len(refs) > 0
}

func (r *importRun) addColumn(table dto.BaseExportTable, f dto.BaseExportField, meta map[string]interface{}) (dto.ColumnResponse, error) {
	order := f.OrderIndex
	col, err := r.s.tableService.AddColumn(r.ctx, r.schema, dto.AddColumnRequest{
		ModelID:     r.tables[table.Key].Model.ID,
		BaseID:      r.baseID,
		Title:       f.Title,
		Description: f.Description,
		Meta:        meta,
		UIDT:        f.UIDT,
		OrderIndex:  &order,
		CreatedBy:   r.userID,
	})
	if err != nil {
		return dto.ColumnResponse{}, fmt.Errorf("create field %q in table %q: %w", f.Title, table.Title, err)
	}
	r.columns[f.Key] = col
	r.ids[dto.BaseExportRefPrefix+f.Key] = col.ID.String()
	return col, nil
}

func (r *importRun) createPlainFields() error {
	for _, t := range r.plan.file.Schema.Tables {
		for _, f := range sortedFields(t.Fields) {
			if f.System || f.UIDT == uidtLinks || f.UIDT == uidtLookup {
				continue
			}
			meta := r.metaFor(f.Meta)
			// The app refuses a written value for an auto uuid field, so with data the field is created as a manual
			// one, the rows keep the uuids of the file, and makeUUIDsAuto then makes it auto again
			if r.includeData && f.UIDT == constant.UIDTUUID && isAutoUUID(meta) {
				r.autoUUIDs = append(r.autoUUIDs, pendingAutoUUID{key: f.Key, meta: meta})
				meta = manualUUIDMeta(meta)
			}
			if _, err := r.addColumn(t, f, meta); err != nil {
				return err
			}
		}
	}
	return nil
}

// pendingAutoUUID is an auto uuid field that was created as a manual one, with the meta it has in the file
type pendingAutoUUID struct {
	key  string
	meta map[string]interface{}
}

// manualUUIDMeta is the meta of a uuid field with its generation set to manual
func manualUUIDMeta(meta map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(meta))
	for k, v := range meta {
		out[k] = v
	}
	out[constant.MetaIDGenerationKey] = map[string]interface{}{"mode": constant.IDModeManual}
	return out
}

// makeUUIDsAuto makes the auto uuid fields auto-generated again, once the rows are in. The rows keep the uuids of the
// file; a row that had none (a file from an older export) gets a new one.
func (r *importRun) makeUUIDsAuto() error {
	backfill := true
	for _, p := range r.autoUUIDs {
		meta := p.meta
		col := r.columns[p.key]
		if _, err := r.s.tableService.UpdateColumn(r.ctx, r.schema, col.ID.String(), dto.ColumnUpdate{Meta: &meta, Backfill: &backfill, UpdatedBy: r.userID}); err != nil {
			return fmt.Errorf("make field %q auto-generated: %w", col.Title, err)
		}
	}
	return nil
}

// linkMeta builds the meta for creating the source side of a link. The service creates the partner field and the
// relation itself, so the relation ids and the role are left out and the partner's title goes in as inverse_title.
func (r *importRun) linkMeta(f dto.BaseExportField, partner *dto.BaseExportField) map[string]interface{} {
	meta := r.metaFor(f.Meta)
	for _, k := range []string{"relation_id", "entity_role", "is_self_link"} {
		delete(meta, k)
	}
	if display, ok := meta["display_column_id"].(string); ok && strings.HasPrefix(display, dto.BaseExportRefPrefix) {
		delete(meta, "display_column_id")
	}
	relation, ok := meta["relation"].(map[string]interface{})
	if !ok {
		return meta
	}
	if partner != nil {
		relation["inverse_title"] = partner.Title
		if display, ok := r.metaFor(partner.Meta)["display_column_id"].(string); ok && !strings.HasPrefix(display, dto.BaseExportRefPrefix) {
			relation["inverse_display_column_id"] = display
		}
	}
	return meta
}

func (r *importRun) createLinkFields() error {
	for _, t := range r.plan.file.Schema.Tables {
		for _, f := range sortedFields(t.Fields) {
			if f.System || f.UIDT != uidtLinks || idString(f.Meta["entity_role"]) != entityRoleSource {
				continue
			}
			rel := strings.TrimPrefix(idString(f.Meta["relation_id"]), dto.BaseExportRefPrefix)
			var partner *dto.BaseExportField
			if key, ok := r.plan.relTarget[rel]; ok {
				p := r.plan.fields[key]
				partner = &p
			}
			col, err := r.addColumn(t, f, r.linkMeta(f, partner))
			if err != nil {
				return err
			}
			relationID := idString(col.Meta["relation_id"])
			r.ids[dto.BaseExportRefPrefix+rel] = relationID
			if partner != nil {
				if err := r.bindPartner(col, *partner, relationID); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// bindPartner finds the field the service created on the other table and gives it the key of the file's partner field.
func (r *importRun) bindPartner(source dto.ColumnResponse, partner dto.BaseExportField, relationID string) error {
	with := nestedString(source.Meta, "relation", "with")
	cols, err := r.s.tableService.GetColumnsByModelID(r.ctx, r.schema, with)
	if err != nil {
		return err
	}
	for _, c := range cols {
		if c.UIDT == uidtLinks && idString(c.Meta["relation_id"]) == relationID && idString(c.Meta["entity_role"]) == entityRoleTarget {
			r.columns[partner.Key] = c
			r.ids[dto.BaseExportRefPrefix+partner.Key] = c.ID.String()
			return nil
		}
	}
	// A self-link is one column used by both sides
	r.columns[partner.Key] = source
	r.ids[dto.BaseExportRefPrefix+partner.Key] = source.ID.String()
	return nil
}

func (r *importRun) createLookupFields() error {
	type pendingLookup struct {
		table dto.BaseExportTable
		field dto.BaseExportField
	}
	var pending []pendingLookup
	for _, t := range r.plan.file.Schema.Tables {
		for _, f := range sortedFields(t.Fields) {
			if !f.System && f.UIDT == uidtLookup {
				pending = append(pending, pendingLookup{t, f})
			}
		}
	}
	// Each pass creates the lookups whose references already exist (the validation has already refused a lookup that reads a lookup)
	for len(pending) > 0 {
		var rest []pendingLookup
		for _, p := range pending {
			meta := r.metaFor(p.field.Meta)
			if hasUnresolved(meta) {
				rest = append(rest, p)
				continue
			}
			if _, err := r.addColumn(p.table, p.field, meta); err != nil {
				return err
			}
		}
		if len(rest) == len(pending) {
			return fmt.Errorf("cannot order lookup fields: %q depends on a field that is never created", rest[0].field.Title)
		}
		pending = rest
	}
	return nil
}

func (r *importRun) createViews() error {
	now := time.Now().UTC()
	for _, pv := range r.plan.views {
		v := pv.view
		v.Table = pv.table
		meta := r.metaFor(v.Meta)
		if hasUnresolved(meta) {
			r.warn("view %q refers to a field that was not created; its settings may be incomplete", v.Title)
		}
		order := v.OrderIndex
		model := r.tables[v.Table].Model

		// The table's own default view is reused, so the new table does not get a second one
		if def := r.defaultView[v.Table]; def != nil && v.Type == "grid" && v.Title == def.Title {
			r.defaultView[v.Table] = nil
			if _, err := r.s.tableService.UpdateView(r.ctx, r.schema, def.ID.String(), dto.ViewUpdate{
				Meta: &meta, OrderIndex: &order, UpdatedBy: r.userID, UpdatedAt: now,
			}); err != nil {
				return fmt.Errorf("update view %q: %w", v.Title, err)
			}
			continue
		}
		if _, err := r.s.tableService.CreateView(r.ctx, r.schema, dto.CreateViewRequest{
			ModelID:     model.ID,
			BaseID:      r.baseID,
			Title:       v.Title,
			Description: v.Description,
			Type:        v.Type,
			Meta:        &meta,
			OrderIndex:  &order,
			CreatedBy:   r.userID,
		}); err != nil {
			return fmt.Errorf("create view %q: %w", v.Title, err)
		}
	}
	return nil
}

// valuesByColumn turns a row's values (by field key) into values by new column id. Link cells are left out: they
// are written afterwards, once every row exists. Attachment cells are uploaded here and hold the new assets.
func (r *importRun) valuesByColumn(row dto.BaseExportRow) (map[string]interface{}, error) {
	out := make(map[string]interface{}, len(row.Values))
	for key, val := range row.Values {
		f := r.plan.fields[key]
		if f.System {
			// only the id is given when the row is created; the times and users are restored once the rows are linked
			if r.systemColumns && f.ColumnName == "id" {
				out[constant.SystemValueKeyPrefix+"id"] = int(numericID(val))
			}
			continue
		}
		v, ok, err := r.columnValue(f, val)
		if err != nil {
			return nil, err
		}
		if ok {
			out[r.columns[key].ID.String()] = v
		}
	}
	return out, nil
}

// columnValue gives the value to write for a field of a row; ok is false when the field has nothing to write
func (r *importRun) columnValue(f dto.BaseExportField, val interface{}) (interface{}, bool, error) {
	if f.UIDT == uidtLinks || val == nil {
		return nil, false, nil
	}
	if f.UIDT == uidtAttachment {
		if r.attachments == nil {
			return nil, false, nil
		}
		assets, err := r.uploadCell(val)
		if err != nil {
			return nil, false, err
		}
		return assets, true, nil
	}
	return val, true, nil
}

// uploadCell uploads the files an attachment cell refers to and returns the cell value made of the new assets.
func (r *importRun) uploadCell(cell interface{}) ([]map[string]interface{}, error) {
	var assets []map[string]interface{}
	for _, item := range toList(cell) {
		ref, _ := item.(string)
		asset, err := r.uploadAttachment(r.index[ref])
		if err != nil {
			return nil, err
		}
		assets = append(assets, asset.Map())
	}
	return assets, nil
}

func (r *importRun) createRows() error {
	creator, ok := r.s.tableService.(rowCreator)
	if !ok {
		return errors.New("the table service cannot insert rows with values")
	}
	totalRows, doneRows := 0, 0
	for _, t := range r.plan.file.Schema.Tables {
		totalRows += len(r.plan.file.Data.Tables[t.Key].Rows)
	}
	for _, t := range r.plan.file.Schema.Tables {
		rows := r.plan.file.Data.Tables[t.Key].Rows
		modelID := r.tables[t.Key].Model.ID.String()
		for start := 0; start < len(rows); start += rowBatchSize {
			end := min(start+rowBatchSize, len(rows))
			// the rows of this batch that have values to insert, with the files uploaded for each
			batch := make([]map[string]interface{}, 0, end-start)
			batchRows := make([]dto.BaseExportRow, 0, end-start)
			batchAssets := make([][]string, 0, end-start)
			for _, row := range rows[start:end] {
				uploadedBefore := len(r.assetIDs)
				values, err := r.valuesByColumn(row)
				if err != nil {
					if r.skipFailed {
						r.skipRow(t, row, err, append([]string(nil), r.assetIDs[uploadedBefore:]...))
						continue
					}
					return fmt.Errorf("prepare rows of table %q: %w", t.Title, err)
				}
				batch = append(batch, values)
				batchRows = append(batchRows, row)
				batchAssets = append(batchAssets, append([]string(nil), r.assetIDs[uploadedBefore:]...))
			}
			if err := r.insertBatch(creator, t, modelID, batch, batchRows, batchAssets); err != nil {
				return err
			}
			doneRows += end - start
			reportFraction(r.ctx, r.progressFrom, r.progressTo, doneRows, totalRows)
		}
	}
	return nil
}

// insertBatch inserts the rows of a batch in one call. When the call fails and the import skips failed rows, the rows
// are inserted one by one: the ones the database refuses are skipped and logged, the others are imported. Otherwise
// a failure fails the import.
func (r *importRun) insertBatch(creator rowCreator, t dto.BaseExportTable, modelID string, batch []map[string]interface{}, batchRows []dto.BaseExportRow, batchAssets [][]string) error {
	if len(batch) == 0 {
		return nil
	}
	created, err := creator.CreateRowsWithValues(r.ctx, r.schema, modelID, batch, r.userID, r.userID)
	if err == nil && len(created) != len(batch) {
		return fmt.Errorf("insert rows of table %q: expected %d rows back, got %d", t.Title, len(batch), len(created))
	}
	if err == nil {
		for i, rec := range created {
			r.rows[batchRows[i].RID] = rowRef{table: t.Key, id: int(numericID(rec.Record["id"]))}
		}
		return nil
	}
	if !r.skipFailed {
		return fmt.Errorf("insert rows of table %q: %w", t.Title, err)
	}
	if r.ctx.Err() != nil {
		return r.ctx.Err()
	}
	return r.insertRowsOneByOne(creator, t, modelID, batch, batchRows, batchAssets)
}

// insertRowsOneByOne inserts the rows of a failed batch separately: the ones the database refuses are skipped and
// logged, the others are imported
func (r *importRun) insertRowsOneByOne(creator rowCreator, t dto.BaseExportTable, modelID string, batch []map[string]interface{}, batchRows []dto.BaseExportRow, batchAssets [][]string) error {
	for i, values := range batch {
		one, err := creator.CreateRowsWithValues(r.ctx, r.schema, modelID, []map[string]interface{}{values}, r.userID, r.userID)
		if err != nil {
			if r.ctx.Err() != nil {
				return r.ctx.Err()
			}
			r.skipRow(t, batchRows[i], err, batchAssets[i])
			continue
		}
		if len(one) != 1 {
			return fmt.Errorf("insert rows of table %q: expected 1 row back, got %d", t.Title, len(one))
		}
		r.rows[batchRows[i].RID] = rowRef{table: t.Key, id: int(numericID(one[0].Record["id"]))}
	}
	return nil
}

// linkRows writes the link cells. Each relation is written once, from its source field; the service fills both sides.
func (r *importRun) linkRows() error {
	for _, t := range r.plan.file.Schema.Tables {
		modelID := r.tables[t.Key].Model.ID.String()
		fields := sortedFields(t.Fields)
		for _, row := range r.plan.file.Data.Tables[t.Key].Rows {
			if _, imported := r.rows[row.RID]; !imported {
				continue // a row that was skipped has nothing to link
			}
			if err := r.linkRowCells(modelID, fields, row); err != nil {
				return err
			}
		}
	}
	return nil
}

// linkRowCells writes the link cells of one imported row, from the fields that are the source of their relation
func (r *importRun) linkRowCells(modelID string, fields []dto.BaseExportField, row dto.BaseExportRow) error {
	from := r.rows[row.RID]
	for _, f := range fields {
		cell, ok := row.Values[f.Key].([]interface{})
		if f.System || f.UIDT != uidtLinks || idString(f.Meta["entity_role"]) != entityRoleSource || !ok {
			continue
		}
		for _, item := range cell {
			to := r.rows[item.(string)]
			_, err := r.s.tableService.UpdateRawDataForLinks(r.ctx, r.schema, dto.UpdateRowDataLinksRequest{
				ModelID:     modelID,
				ColumnId:    r.columns[f.Key].ID.String(),
				SourceRowId: from.id,
				TargetRowId: to.id,
				Action:      linkAction,
				UpdatedBy:   r.userID,
			})
			if err != nil {
				return fmt.Errorf("link %s to %s in field %q: %w", row.RID, item, f.Title, err)
			}
		}
	}
	return nil
}

// checkAttachmentFiles checks that every file the index lists, and the cover image, is in the uploaded zip.
func checkAttachmentFiles(file *dto.BaseExportFile, opts dto.ImportBaseOptions) []string {
	if file == nil || opts.Attachments == nil {
		return nil
	}
	var problems problemList
	if c := file.Base.Cover; c != nil && attachmentPathPattern.MatchString(c.Path) && !opts.Attachments.Has(c.Path) {
		problems.add("base.cover: %s is not in the zip", c.Path)
	}
	if file.Data != nil && opts.IncludeData {
		for i, a := range file.Data.Attachments {
			if attachmentPathPattern.MatchString(a.Path) && !opts.Attachments.Has(a.Path) {
				problems.add("data.attachments[%d]: %s is not in the zip", i, a.Path)
			}
		}
	}
	return problems
}

// setPrimaryKeys gives tables their custom key. It runs before links are created.
func (r *importRun) setPrimaryKeys() error {
	for _, t := range r.plan.file.Schema.Tables {
		if len(t.PrimaryKey) == 0 {
			continue
		}
		if r.s.primaryKeyService == nil {
			r.warn("primary key of table %q was not set: the primary key service is not configured", t.Title)
			continue
		}
		ids := make([]string, 0, len(t.PrimaryKey))
		for _, key := range t.PrimaryKey {
			ids = append(ids, r.columns[key].ID.String())
		}
		if _, err := r.s.primaryKeyService.Update(r.ctx, r.schema, r.tables[t.Key].Model.ID.String(), dto.UpdatePrimaryKeyRequest{ColumnIDs: ids}); err != nil {
			return fmt.Errorf("set primary key of table %q: %w", t.Title, err)
		}
	}
	return nil
}

// columnNames maps field keys to the column names of the new fields, to rebuild check conditions.
func (r *importRun) columnNames(t dto.BaseExportTable) map[string]string {
	names := make(map[string]string, len(t.Fields))
	for _, f := range t.Fields {
		if col, ok := r.columns[f.Key]; ok {
			names[f.Key] = col.ColumnName
		}
	}
	return names
}

// createConstraints creates the unique and check constraints after all rows exist.
func (r *importRun) createConstraints() error {
	for _, t := range r.plan.file.Schema.Tables {
		if err := r.createTableConstraints(t); err != nil {
			return err
		}
	}
	return nil
}

// createTableConstraints creates the constraints of one table
func (r *importRun) createTableConstraints(t dto.BaseExportTable) error {
	if len(t.Constraints) == 0 {
		return nil
	}
	if r.s.constraints == nil {
		r.warn("constraints of table %q were not created: the constraint service is not configured", t.Title)
		return nil
	}
	names := r.columnNames(t)
	modelID := r.tables[t.Key].Model.ID.String()
	for _, c := range t.Constraints {
		req := dto.CreateTableConstraintRequest{ModelID: modelID, Type: c.Type, Title: c.Title, CreatedBy: r.userID}
		for _, key := range c.FieldKeys {
			req.ColumnIDs = append(req.ColumnIDs, r.columns[key].ID.String())
		}
		if c.Type == "check" {
			req.CheckInput = dto.CheckInput{Category: c.Category, Condition: detokenizeCondition(c.Condition, names)}
		}
		if _, err := r.s.constraints.Create(r.ctx, r.schema, req); err != nil {
			return fmt.Errorf("create %s constraint %q on table %q: %w", c.Type, c.Title, t.Title, err)
		}
	}
	return nil
}

// systemRestorer is implemented by the table service: it writes the source columns of a row as they are, without
// touching the modified time or user.
type systemRestorer interface {
	RestoreRowSystemValues(ctx context.Context, schemaName, modelID string, rowID int, values map[string]interface{}) error
}

// restoreSystemColumns writes the created / modified time and user of the file onto the new rows. The id was given
// when the row was created.
func (r *importRun) restoreSystemColumns() error {
	restorer, ok := r.s.tableService.(systemRestorer)
	if !ok {
		r.warn("source columns were not restored: the table service cannot write them")
		return nil
	}
	total, done := 0, 0
	for _, t := range r.plan.file.Schema.Tables {
		total += len(r.plan.file.Data.Tables[t.Key].Rows)
	}
	for _, t := range r.plan.file.Schema.Tables {
		modelID := r.tables[t.Key].Model.ID.String()
		for _, row := range r.plan.file.Data.Tables[t.Key].Rows {
			done++
			values := r.sourceValues(row)
			if len(values) > 0 {
				if err := restorer.RestoreRowSystemValues(r.ctx, r.schema, modelID, r.rows[row.RID].id, values); err != nil {
					return fmt.Errorf("restore source columns of %s: %w", row.RID, err)
				}
			}
			reportFraction(r.ctx, r.progressFrom, r.progressTo, done, total)
		}
	}
	return nil
}

// sourceValues takes the source columns (not the id) of a row of the file, by column name
func (r *importRun) sourceValues(row dto.BaseExportRow) map[string]interface{} {
	values := map[string]interface{}{}
	for key, val := range row.Values {
		f := r.plan.fields[key]
		if !f.System || f.ColumnName == "id" || !isSourceColumn(f.ColumnName) || val == nil {
			continue
		}
		if v, ok := sourceColumnValue(f.ColumnName, val); ok {
			values[f.ColumnName] = v
		}
	}
	return values
}

// sourceColumnValue converts a source value of the file: a time is parsed (it was validated when the file was read,
// so this parses), the rest is text; ok is false for a time that does not parse
func sourceColumnValue(columnName string, val interface{}) (interface{}, bool) {
	if columnName == "created_time" || columnName == "last_modified_time" {
		return parseSourceTime(fmt.Sprint(val))
	}
	return fmt.Sprint(val), true
}
