// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package transfer_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"regexp"
	"strings"
	"time"

	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	storageif "github.com/aptlogica/sereni-base/internal/providers/storage/interfaces"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/google/uuid"
)

// fakeDB is a small in-memory stand-in for the base, table, column, view and row services.
type fakeDB struct {
	interfaces.TableManagementService
	interfaces.BaseManagementService

	bases   map[string]tenant.Base
	models  map[string]*dto.ModelResponse
	columns map[string][]dto.ColumnResponse // by model id
	views   map[string][]dto.ViewResponse   // by model id
	rows    map[string][]map[string]interface{}
	nextRow map[string]int

	// failOnUIDT makes AddColumn fail, to test rollback
	failOnUIDT string
	// failRows makes row insertion fail, after attachments were uploaded
	failRows bool
	// refuseRow makes the database refuse a row, like a value that does not fit its column; the call fails as a whole
	refuseRow func(values map[string]interface{}) error

	// files is the object storage by path; assets are the stored asset records by id
	files  map[string][]byte
	assets map[string]tenant.Assets

	// primary key column ids and constraints by model id
	pk          map[string][]string
	constraints map[string][]storedConstraint
	// automations by model id, and the names of the trigger functions that exist in the database (global, like Postgres)
	automations map[string][]tenant.Automation
	functions   map[string]bool
	// failTrigger makes creating a trigger fail, after the functions exist
	failTrigger bool
	// seq makes generated column names differ, like the numeric suffix the real service adds
	seq int
}

type storedConstraint struct {
	id        string
	kind      string
	title     string
	columnIDs []string
	category  string
	condition string
}

func newFakeDB() *fakeDB {
	return &fakeDB{
		bases:       map[string]tenant.Base{},
		models:      map[string]*dto.ModelResponse{},
		columns:     map[string][]dto.ColumnResponse{},
		views:       map[string][]dto.ViewResponse{},
		rows:        map[string][]map[string]interface{}{},
		nextRow:     map[string]int{},
		files:       map[string][]byte{},
		assets:      map[string]tenant.Assets{},
		pk:          map[string][]string{},
		constraints: map[string][]storedConstraint{},
		automations: map[string][]tenant.Automation{},
		functions:   map[string]bool{},
	}
}

func ptr[T any](v T) *T { return &v }

func columnName(title string) string {
	return strings.ReplaceAll(strings.ToLower(title), " ", "_")
}

// --- base service ---

func (f *fakeDB) CreateBaseWithoutTable(_ context.Context, req dto.CreateBaseRequest, _ string, _ string) (tenant.Base, error) {
	b := tenant.Base{ID: uuid.New(), WorkspaceID: req.WorkspaceID, Title: req.Title, Description: req.Description, Meta: req.Meta, Type: "default"}
	f.bases[b.ID.String()] = b
	return b, nil
}

func (f *fakeDB) GetBaseByID(_ context.Context, _ string, id string) (tenant.Base, error) {
	b, ok := f.bases[id]
	if !ok {
		return tenant.Base{}, errors.New("base not found")
	}
	return b, nil
}

func (f *fakeDB) GetBasesByWorkspace(_ context.Context, _ string, workspaceID string) ([]tenant.Base, error) {
	var out []tenant.Base
	for _, b := range f.bases {
		if b.WorkspaceID == workspaceID {
			out = append(out, b)
		}
	}
	return out, nil
}

func (f *fakeDB) DeleteBase(_ context.Context, _ string, id string) error {
	delete(f.bases, id)
	return nil
}

// --- table service ---

func (f *fakeDB) CreateTableWithDefaultsImport(_ context.Context, req dto.CreateTableRequest, _ string) (dto.TableResponse, error) {
	baseID := uuid.MustParse(req.BaseID)
	f.seq++
	m := &dto.ModelResponse{ID: uuid.New(), BaseID: baseID, Title: req.Title, Description: req.Description, OrderIndex: ptr(req.OrderIndex),
		Alias: fmt.Sprintf("%s_%d", columnName(req.Title), 2000+f.seq)}
	id := m.ID.String()
	f.models[id] = m
	system := []struct {
		title string
		uidt  string
		order float64
	}{{"Id", "number", 0}, {"Created Time", "datetime", 2}, {"Last Modified Time", "datetime", 3}, {"Created By", "text", 4}}
	for _, s := range system {
		f.columns[id] = append(f.columns[id], dto.ColumnResponse{
			DT: map[string]string{"number": "BIGSERIAL", "datetime": "TIMESTAMP", "text": "TEXT"}[s.uidt],
			ID: uuid.New(), ModelID: m.ID, BaseID: baseID, Title: s.title, ColumnName: columnName(s.title),
			UIDT: s.uidt, System: ptr(true), Deleted: ptr(false), OrderIndex: ptr(s.order), Meta: map[string]interface{}{},
		})
	}
	view := dto.ViewResponse{ID: uuid.New(), ModelID: m.ID, BaseID: baseID, Title: "Default Grid View", Type: "grid", OrderIndex: ptr(0.0), Meta: &map[string]interface{}{}}
	f.views[id] = append(f.views[id], view)
	return dto.TableResponse{Model: *m, Columns: append([]dto.ColumnResponse(nil), f.columns[id]...), Views: []dto.ViewResponse{view}}, nil
}

func (f *fakeDB) GetModelByBaseID(_ context.Context, _ string, baseID string) ([]dto.TableResponse, error) {
	var out []dto.TableResponse
	for _, m := range f.models {
		if m.BaseID.String() == baseID {
			out = append(out, dto.TableResponse{Model: *m})
		}
	}
	return out, nil
}

func (f *fakeDB) GetColumnsByModelID(_ context.Context, _ string, modelID string) ([]dto.ColumnResponse, error) {
	return append([]dto.ColumnResponse(nil), f.columns[modelID]...), nil
}

func (f *fakeDB) GetViewsByModelID(_ context.Context, _ string, modelID string) ([]dto.ViewResponse, error) {
	return append([]dto.ViewResponse(nil), f.views[modelID]...), nil
}

func (f *fakeDB) GetAllRecords(_ context.Context, _ string, modelID string) (dto.RecordsResponse, error) {
	var out []map[string]interface{}
	for _, r := range f.rows[modelID] {
		cp := map[string]interface{}{}
		for k, v := range r {
			cp[k] = v
		}
		out = append(out, cp)
	}
	return dto.RecordsResponse{Records: out}, nil
}

func (f *fakeDB) DeleteTable(_ context.Context, _ string, modelID string) error {
	delete(f.models, modelID)
	delete(f.columns, modelID)
	delete(f.views, modelID)
	delete(f.rows, modelID)
	return nil
}

// dbType is what the app derives for a field type: the database type of its mapping. A link is stored as JSONB and a
// lookup keeps its own name.
func dbType(uidt string) string {
	switch uidt {
	case "links":
		return "JSONB"
	case "lookup":
		return "lookup"
	}
	if m, ok := constant.UITypeMappings[uidt]; ok {
		return m.Postgres
	}
	return uidt
}

func (f *fakeDB) addColumn(req dto.AddColumnRequest, title string, meta map[string]interface{}) dto.ColumnResponse {
	f.seq++
	// like the real service, the type comes from the field type and never from the request
	req.DT = dbType(req.UIDT)
	col := dto.ColumnResponse{DT: req.DT,
		ID: uuid.New(), ModelID: req.ModelID, BaseID: req.BaseID, Title: title, ColumnName: fmt.Sprintf("%s_%d", columnName(title), 1000+f.seq),
		UIDT: req.UIDT, Description: req.Description, Meta: meta, System: ptr(false), Deleted: ptr(false), OrderIndex: req.OrderIndex,
	}
	f.columns[req.ModelID.String()] = append(f.columns[req.ModelID.String()], col)
	return col
}

func (f *fakeDB) AddColumn(_ context.Context, _ string, req dto.AddColumnRequest) (dto.ColumnResponse, error) {
	if f.failOnUIDT != "" && req.UIDT == f.failOnUIDT {
		return dto.ColumnResponse{}, errors.New("boom")
	}
	meta := req.Meta
	if meta == nil {
		meta = map[string]interface{}{}
	}
	switch req.UIDT {
	case "links":
		return f.addLink(req, meta)
	case "lookup":
		if _, err := uuid.Parse(fmt.Sprint(meta["lookup_column_id"])); err != nil {
			return dto.ColumnResponse{}, errors.New("lookup_column_id must be a column id")
		}
		if _, err := uuid.Parse(fmt.Sprint(meta["relation_id"])); err != nil {
			return dto.ColumnResponse{}, errors.New("relation_id must be an id")
		}
		// the rules of the real service (validateLookupTarget): a plain field on the other side of the link
		shown, ok := f.columnByID(fmt.Sprint(meta["lookup_column_id"]))
		if !ok {
			return dto.ColumnResponse{}, errors.New("the field to show does not exist")
		}
		switch shown.UIDT {
		case "links", "lookup", "rollup":
			return dto.ColumnResponse{}, errors.New("Lookup field not allowed (TBL_1079)")
		}
		onTheOtherSide := false
		for _, c := range f.columns[req.ModelID.String()] {
			if c.UIDT == "links" && fmt.Sprint(c.Meta["relation_id"]) == fmt.Sprint(meta["relation_id"]) {
				if rel, _ := c.Meta["relation"].(map[string]interface{}); fmt.Sprint(rel["with"]) == shown.ModelID.String() {
					onTheOtherSide = true
				}
			}
		}
		if !onTheOtherSide {
			return dto.ColumnResponse{}, errors.New("the field to show is not on the table the link points at (TBL_1077)")
		}
	}
	return f.addColumn(req, req.Title, meta), nil
}

func (f *fakeDB) addLink(req dto.AddColumnRequest, meta map[string]interface{}) (dto.ColumnResponse, error) {
	relation, _ := meta["relation"].(map[string]interface{})
	with, _ := relation["with"].(string)
	if _, ok := f.models[with]; !ok {
		return dto.ColumnResponse{}, errors.New("relation.with must be a table id")
	}
	inverse, _ := relation["inverse_title"].(string)
	inverseDisplay, _ := relation["inverse_display_column_id"].(string)
	delete(relation, "inverse_title")
	delete(relation, "inverse_display_column_id")

	relID := uuid.New()
	self := with == req.ModelID.String()
	meta["entity_role"] = "source"
	meta["relation_id"] = relID // like the real service, a uuid.UUID value
	meta["is_self_link"] = self
	if _, ok := meta["display_column_id"]; !ok {
		meta["display_column_id"] = f.columns[with][0].ID.String()
	}
	source := f.addColumn(req, req.Title, meta)
	if !self {
		if inverse == "" {
			inverse = f.models[req.ModelID.String()].Title
		}
		if inverseDisplay == "" {
			inverseDisplay = f.columns[req.ModelID.String()][0].ID.String()
		}
		targetMeta := map[string]interface{}{
			"relation":          map[string]interface{}{"with": req.ModelID.String(), "type": relation["type"]},
			"entity_role":       "target",
			"relation_id":       relID,
			"display_column_id": inverseDisplay,
		}
		f.addColumn(dto.AddColumnRequest{ModelID: uuid.MustParse(with), BaseID: req.BaseID, UIDT: "links", OrderIndex: ptr(99.0)}, inverse, targetMeta)
	}
	return source, nil
}

func (f *fakeDB) CreateView(_ context.Context, _ string, req dto.CreateViewRequest) (dto.ViewResponse, error) {
	v := dto.ViewResponse{ID: uuid.New(), ModelID: req.ModelID, BaseID: req.BaseID, Title: req.Title, Type: req.Type, Meta: req.Meta, OrderIndex: req.OrderIndex}
	f.views[req.ModelID.String()] = append(f.views[req.ModelID.String()], v)
	return v, nil
}

func (f *fakeDB) UpdateView(_ context.Context, _ string, id string, req dto.ViewUpdate) (dto.ViewResponse, error) {
	for model, views := range f.views {
		for i, v := range views {
			if v.ID.String() == id {
				f.views[model][i].Meta = req.Meta
				f.views[model][i].OrderIndex = req.OrderIndex
				return f.views[model][i], nil
			}
		}
	}
	return dto.ViewResponse{}, errors.New("view not found")
}

func (f *fakeDB) columnByID(id string) (dto.ColumnResponse, bool) {
	for _, cols := range f.columns {
		for _, c := range cols {
			if c.ID.String() == id {
				return c, true
			}
		}
	}
	return dto.ColumnResponse{}, false
}

// CreateRowsWithValues stores values by column name; link cells are not accepted here, like in the real service.
func (f *fakeDB) CreateRowsWithValues(_ context.Context, _ string, modelID string, rowsInput []map[string]interface{}, _ string, _ string) ([]dto.RecordResponse, error) {
	if f.failRows {
		return nil, errors.New("rows failed")
	}
	if f.refuseRow != nil {
		for _, in := range rowsInput {
			if err := f.refuseRow(in); err != nil {
				return nil, err
			}
		}
	}
	var out []dto.RecordResponse
	for _, in := range rowsInput {
		f.nextRow[modelID]++
		row := map[string]interface{}{"id": f.nextRow[modelID]}
		for colID, val := range in {
			// like the real service: the import can give the id of a row
			if name, ok := strings.CutPrefix(colID, "@system:"); ok {
				row[name] = val
				if n, isInt := val.(int); isInt && n > f.nextRow[modelID] {
					f.nextRow[modelID] = n
				}
				continue
			}
			col, ok := f.columnByID(colID)
			if !ok || col.ModelID.String() != modelID {
				return nil, fmt.Errorf("unknown column %s", colID)
			}
			if col.UIDT == "links" {
				return nil, errors.New("link cells are written by UpdateRawDataForLinks")
			}
			if isAutoUUIDColumn(col) {
				return nil, errors.New("an auto uuid field cannot be written") // like the real service (UUIDReadOnly)
			}
			row[col.ColumnName] = val
		}
		// the system fills in an auto uuid field for every new row
		for _, col := range f.columns[modelID] {
			if isAutoUUIDColumn(col) {
				row[col.ColumnName] = uuid.NewString()
			}
		}
		f.rows[modelID] = append(f.rows[modelID], row)
		out = append(out, dto.RecordResponse{Record: row})
	}
	return out, nil
}

// UpdateColumn applies the meta of an update, and, when a field becomes auto-generated and a backfill is asked for,
// gives the empty cells a value, like the real service.
func (f *fakeDB) UpdateColumn(_ context.Context, _ string, id string, req dto.ColumnUpdate) (dto.ColumnResponse, error) {
	for modelID, cols := range f.columns {
		for i, c := range cols {
			if c.ID.String() != id {
				continue
			}
			wasAuto := isAutoUUIDColumn(c)
			if req.Meta != nil {
				c.Meta = *req.Meta
				f.columns[modelID][i] = c
			}
			if !wasAuto && isAutoUUIDColumn(c) && req.Backfill != nil && *req.Backfill {
				for _, row := range f.rows[modelID] {
					if row[c.ColumnName] == nil {
						row[c.ColumnName] = uuid.NewString()
					}
				}
			}
			return c, nil
		}
	}
	return dto.ColumnResponse{}, errors.New("column not found")
}

// colByTitle finds a live column of a table by its title.
func (f *fakeDB) colByTitle(modelID, title string) dto.ColumnResponse {
	for _, c := range f.columns[modelID] {
		if c.Title == title {
			return c
		}
	}
	panic("no column " + title)
}

// cell reads a value of the first row, by column title.
func (f *fakeDB) cell(modelID string, row int, title string) interface{} {
	return f.rows[modelID][row][f.colByTitle(modelID, title).ColumnName]
}

// addRow adds a row; values are keyed by column title.
func (f *fakeDB) addRow(modelID, _ string, vals map[string]interface{}) int {
	f.nextRow[modelID]++
	row := map[string]interface{}{"id": f.nextRow[modelID]}
	for title, v := range vals {
		row[f.colByTitle(modelID, title).ColumnName] = v
	}
	f.rows[modelID] = append(f.rows[modelID], row)
	return f.nextRow[modelID]
}

func (f *fakeDB) rowByID(modelID string, id int) map[string]interface{} {
	for _, r := range f.rows[modelID] {
		if r["id"] == id {
			return r
		}
	}
	return nil
}

func appendUnique(cell interface{}, id interface{}) []interface{} {
	list, _ := cell.([]interface{})
	for _, v := range list {
		if v == id {
			return list
		}
	}
	return append(list, id)
}

// keyValue is what a link cell holds for a row: its primary-key value (the id unless the table has a custom key).
func (f *fakeDB) keyValue(modelID string, row map[string]interface{}) interface{} {
	if ids := f.pk[modelID]; len(ids) == 1 {
		if col, ok := f.columnByID(ids[0]); ok {
			return row[col.ColumnName]
		}
	}
	return row["id"]
}

// UpdateRawDataForLinks fills the link cell on both sides, as link_rows does.
func (f *fakeDB) UpdateRawDataForLinks(_ context.Context, _ string, req dto.UpdateRowDataLinksRequest) (dto.RecordResponse, error) {
	col, ok := f.columnByID(req.ColumnId)
	if !ok || col.ModelID.String() != req.ModelID {
		return dto.RecordResponse{}, errors.New("link column not found")
	}
	relation, _ := col.Meta["relation"].(map[string]interface{})
	with := fmt.Sprint(relation["with"])
	src := f.rowByID(req.ModelID, req.SourceRowId)
	dst := f.rowByID(with, req.TargetRowId)
	if src == nil || dst == nil {
		return dto.RecordResponse{}, errors.New("row not found")
	}
	src[col.ColumnName] = appendUnique(src[col.ColumnName], f.keyValue(with, dst))
	for _, other := range f.columns[with] {
		if other.UIDT == "links" && fmt.Sprint(other.Meta["relation_id"]) == fmt.Sprint(col.Meta["relation_id"]) && other.ID != col.ID {
			dst[other.ColumnName] = appendUnique(dst[other.ColumnName], f.keyValue(req.ModelID, src))
		}
	}
	return dto.RecordResponse{Record: src}, nil
}

// --- storage and asset services ---

func (f *fakeDB) Upload(context.Context, string, io.Reader, int64, string) (storageif.UploadResponse, error) {
	return storageif.UploadResponse{}, errors.New("not used")
}

func (f *fakeDB) Download(_ context.Context, objectName string) (io.ReadCloser, error) {
	data, ok := f.files[objectName]
	if !ok {
		return nil, fmt.Errorf("%s not found", objectName)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (f *fakeDB) Delete(context.Context, string) error         { return nil }
func (f *fakeDB) Exists(context.Context, string) (bool, error) { return true, nil }

// fakeAssets stores uploads in the same map the storage fake reads from.
type fakeAssets struct {
	interfaces.AssetManagementService
	db *fakeDB
}

func (a fakeAssets) Upload(_ context.Context, req dto.UploadAssetRequest, schema string) ([]tenant.Assets, error) {
	var out []tenant.Assets
	for _, h := range req.Files {
		src, err := h.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(src)
		src.Close()
		if err != nil {
			return nil, err
		}
		if bytes.Contains(data, []byte("EICAR")) {
			return nil, fmt.Errorf("file '%s' is infected", h.Filename)
		}
		asset := tenant.Assets{ID: uuid.New(), Title: h.Filename, MimeType: h.Header.Get("Content-Type"), Size: int64(len(data))}
		asset.BasePath = schema + "/" + asset.ID.String() + "_" + h.Filename
		asset.Url = "https://files/" + asset.BasePath
		a.db.files[asset.BasePath] = data
		a.db.assets[asset.ID.String()] = asset
		out = append(out, asset)
	}
	return out, nil
}

func (a fakeAssets) DeleteAsset(_ context.Context, id string, _ string) error {
	if asset, ok := a.db.assets[id]; ok {
		delete(a.db.files, asset.BasePath)
		delete(a.db.assets, id)
	}
	return nil
}

func (a fakeAssets) GetAssetByURL(_ context.Context, _ string, url string) (tenant.Assets, error) {
	for _, asset := range a.db.assets {
		if asset.Url == url {
			return asset, nil
		}
	}
	return tenant.Assets{}, errors.New("asset not found")
}

// storeFile puts a file in storage and returns the attachment cell item that points at it.
func (f *fakeDB) storeFile(name, contentType string, data []byte) map[string]interface{} {
	asset := tenant.Assets{ID: uuid.New(), Title: name, MimeType: contentType, Size: int64(len(data))}
	asset.BasePath = "s/" + asset.ID.String() + "_" + name
	asset.Url = "https://files/" + asset.BasePath
	f.files[asset.BasePath] = data
	f.assets[asset.ID.String()] = asset
	return asset.Map()
}

// --- cover image ---

// setCover stores a cover image for a base and returns nothing; the base's image is the asset url.
func (f *fakeDB) setCover(baseID, name string, data []byte) {
	cell := f.storeFile(name, "image/png", data)
	b := f.bases[baseID]
	b.Image = fmt.Sprint(cell["url"])
	f.bases[baseID] = b
}

func (f *fakeDB) AddBaseImage(_ context.Context, _ string, baseID string, h *multipart.FileHeader, _ string) (tenant.Base, error) {
	if strings.HasSuffix(h.Filename, ".bad") {
		return tenant.Base{}, errors.New("unsupported image type")
	}
	src, err := h.Open()
	if err != nil {
		return tenant.Base{}, err
	}
	data, _ := io.ReadAll(src)
	src.Close()
	f.setCover(baseID, h.Filename, data)
	return f.bases[baseID], nil
}

func (f *fakeDB) RemoveBaseImage(_ context.Context, _ string, baseID string, _ string) (tenant.Base, error) {
	b := f.bases[baseID]
	for id, a := range f.assets {
		if a.Url == b.Image {
			delete(f.files, a.BasePath)
			delete(f.assets, id)
		}
	}
	b.Image = ""
	f.bases[baseID] = b
	return b, nil
}

// --- primary key service ---

type fakePK struct {
	interfaces.PrimaryKeyService
	db *fakeDB
}

func (p fakePK) Get(_ context.Context, _ string, modelID string) (dto.PrimaryKeyResponse, error) {
	ids := p.db.pk[modelID]
	if len(ids) == 0 {
		for _, c := range p.db.columns[modelID] {
			if c.ColumnName == "id" {
				ids = []string{c.ID.String()}
			}
		}
	}
	resp := dto.PrimaryKeyResponse{ModelID: modelID, ColumnIDs: ids}
	for _, id := range ids {
		c, _ := p.db.columnByID(id)
		resp.ColumnNames = append(resp.ColumnNames, c.ColumnName)
	}
	return resp, nil
}

// Update refuses a new key while the table has link fields, like the real service.
func (p fakePK) Update(_ context.Context, _ string, modelID string, req dto.UpdatePrimaryKeyRequest) (dto.PrimaryKeyResponse, error) {
	for _, c := range p.db.columns[modelID] {
		if c.UIDT == "links" {
			return dto.PrimaryKeyResponse{}, errors.New("the primary key can't be changed while this table has link fields")
		}
	}
	p.db.pk[modelID] = req.ColumnIDs
	return p.Get(context.Background(), "", modelID)
}

// --- constraint service ---

type fakeConstraints struct {
	interfaces.TableConstraintService
	db *fakeDB
}

func (c fakeConstraints) List(_ context.Context, _ string, modelID, kind string) ([]dto.TableConstraintResponse, error) {
	var out []dto.TableConstraintResponse
	for _, s := range c.db.constraints[modelID] {
		if s.kind != kind {
			continue
		}
		resp := dto.TableConstraintResponse{ID: s.id, Title: s.title, Status: dto.ConstraintStatusOK}
		resp.Category, resp.Condition = s.category, s.condition
		ids := s.columnIDs
		if kind == "check" {
			ids = nil
			for _, col := range c.db.columns[modelID] {
				if strings.Contains(s.condition, col.ColumnName) {
					ids = append(ids, col.ID.String())
				}
			}
		}
		for _, id := range ids {
			col, _ := c.db.columnByID(id)
			resp.Columns = append(resp.Columns, dto.ConstraintColumn{ID: id, Title: col.Title})
		}
		out = append(out, resp)
	}
	return out, nil
}

func (c fakeConstraints) Create(_ context.Context, _ string, req dto.CreateTableConstraintRequest) (dto.TableConstraintResponse, error) {
	if req.Type == "check" && strings.ContainsAny(req.Condition, "{}") {
		return dto.TableConstraintResponse{}, errors.New("the condition still holds a placeholder")
	}
	if req.Type == "check" && req.Category != "" && req.Category != "number_date" && req.Category != "text" && req.Category != "compare" && req.Category != "custom" {
		return dto.TableConstraintResponse{}, fmt.Errorf("Unknown category %q", req.Category) // like the real service
	}
	if req.Type == "unique" && len(req.ColumnIDs) == 0 {
		return dto.TableConstraintResponse{}, errors.New("a unique constraint needs columns")
	}
	c.db.constraints[req.ModelID] = append(c.db.constraints[req.ModelID], storedConstraint{
		id: uuid.NewString(), kind: req.Type, title: req.Title, columnIDs: req.ColumnIDs, category: req.Category, condition: req.Condition,
	})
	return dto.TableConstraintResponse{Title: req.Title}, nil
}

// --- automation service ---

type fakeAutomations struct {
	interfaces.AutomationService
	db *fakeDB
}

var fakeFunctionName = regexp.MustCompile(`(?is)^\s*CREATE\s+(?:OR\s+REPLACE\s+)?FUNCTION\s+(?:"?\w+"?\.)?"?(\w+)"?\s*\(`)
var fakeExecuteFunction = regexp.MustCompile(`(?is)EXECUTE\s+(?:FUNCTION|PROCEDURE)\s+(?:"?\w+"?\.)?"?(\w+)"?\s*\(`)

// List returns the newest first, like the real service.
func (a fakeAutomations) List(_ context.Context, _ string, modelID string) ([]tenant.Automation, error) {
	list := a.db.automations[modelID]
	out := make([]tenant.Automation, len(list))
	for i, x := range list {
		out[len(list)-1-i] = x
	}
	return out, nil
}

// Create behaves like Postgres where it matters: a function name must be new, a trigger needs its table and its
// function to exist, and no placeholder may be left in the SQL.
func (a fakeAutomations) Create(_ context.Context, _ string, req dto.CreateAutomationRequest) (tenant.Automation, error) {
	if strings.ContainsAny(req.Context, "{}") {
		return tenant.Automation{}, errors.New("the SQL still holds a placeholder")
	}
	for _, x := range a.db.automations[req.ModelID] {
		if x.Type == req.Type && strings.EqualFold(x.Title, req.Title) {
			return tenant.Automation{}, errors.New("title already used")
		}
	}
	switch req.Type {
	case tenant.AutomationTypeFunction:
		m := fakeFunctionName.FindStringSubmatch(req.Context)
		if m == nil {
			return tenant.Automation{}, errors.New("not a CREATE FUNCTION")
		}
		if a.db.functions[strings.ToLower(m[1])] {
			return tenant.Automation{}, fmt.Errorf("function %s already exists", m[1])
		}
		a.db.functions[strings.ToLower(m[1])] = true
	case tenant.AutomationTypeTrigger:
		if a.db.failTrigger {
			return tenant.Automation{}, errors.New("trigger failed")
		}
		if regexp.MustCompile(`(?i)\bOR\s+(BEFORE|AFTER|INSTEAD\s+OF)\b`).MatchString(req.Context) {
			return tenant.Automation{}, errors.New(`syntax error at or near "BEFORE"`) // what Postgres says
		}
		model := a.db.models[req.ModelID]
		if model == nil || !strings.Contains(req.Context, `"`+model.Alias+`"`) {
			return tenant.Automation{}, errors.New("the trigger is not on this table")
		}
		m := fakeExecuteFunction.FindStringSubmatch(req.Context)
		if m == nil || !a.db.functions[strings.ToLower(m[1])] {
			return tenant.Automation{}, errors.New("the function does not exist")
		}
		if strings.TrimSpace(req.Event) == "" {
			return tenant.Automation{}, errors.New("the trigger needs an event")
		}
	}
	created := tenant.Automation{ID: uuid.New(), ModelID: req.ModelID, Title: req.Title, Type: req.Type, Context: req.Context, Event: req.Event,
		CreatedAt: time.Now().Add(time.Duration(len(a.db.automations[req.ModelID])) * time.Millisecond)}
	a.db.automations[req.ModelID] = append(a.db.automations[req.ModelID], created)
	return created, nil
}

func (a fakeAutomations) Delete(_ context.Context, _ string, id string) error {
	for model, list := range a.db.automations {
		for i, x := range list {
			if x.ID.String() == id {
				if x.Type == tenant.AutomationTypeFunction {
					if m := fakeFunctionName.FindStringSubmatch(x.Context); m != nil {
						delete(a.db.functions, strings.ToLower(m[1]))
					}
				}
				a.db.automations[model] = append(list[:i:i], list[i+1:]...)
				return nil
			}
		}
	}
	return errors.New("automation not found")
}

func isAutoUUIDColumn(col dto.ColumnResponse) bool {
	if col.UIDT != "uuid" {
		return false
	}
	cfg, _ := col.Meta["idGeneration"].(map[string]interface{})
	return cfg["mode"] == "auto"
}

// RestoreRowSystemValues writes system columns of a row as they are, like the real service.
func (f *fakeDB) RestoreRowSystemValues(_ context.Context, _ string, modelID string, rowID int, values map[string]interface{}) error {
	for _, row := range f.rows[modelID] {
		if id, _ := row["id"].(int); id == rowID {
			for k, v := range values {
				row[k] = v
			}
			return nil
		}
	}
	return fmt.Errorf("no row %d", rowID)
}
