// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package transfer_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	transfer "github.com/aptlogica/sereni-base/internal/services/transfer"
	"github.com/google/uuid"
)

const workspace = "11111111-1111-1111-1111-111111111111"

type crm struct {
	db                      *fakeDB
	baseID                  string
	customers, orders       string
	name, orderLink, amount string
}

// seedCRM builds a base through the fake services: customers and orders linked has-many, a lookup, views and rows.
func seedCRM(t *testing.T) *crm {
	t.Helper()
	ctx := context.Background()
	db := newFakeDB()
	base, _ := db.CreateBaseWithoutTable(ctx, dto.CreateBaseRequest{Title: "CRM", Description: ptr("Customers and orders"), WorkspaceID: workspace}, "s", "u")
	baseID := base.ID

	cust, _ := db.CreateTableWithDefaultsImport(ctx, dto.CreateTableRequest{BaseID: baseID.String(), Title: "Customers", OrderIndex: 1}, "s")
	ord, _ := db.CreateTableWithDefaultsImport(ctx, dto.CreateTableRequest{BaseID: baseID.String(), Title: "Orders", OrderIndex: 2}, "s")

	name, err := db.AddColumn(ctx, "s", dto.AddColumnRequest{ModelID: cust.Model.ID, BaseID: baseID, Title: "Name", UIDT: "text", OrderIndex: ptr(1.0), Meta: map[string]interface{}{}})
	must(t, err)
	// a custom primary key, set before any link exists (the real service refuses it afterwards)
	code, err := db.AddColumn(ctx, "s", dto.AddColumnRequest{ModelID: cust.Model.ID, BaseID: baseID, Title: "Code", UIDT: "text", OrderIndex: ptr(1.5)})
	must(t, err)
	_, err = fakePK{db: db}.Update(ctx, "s", cust.Model.ID.String(), dto.UpdatePrimaryKeyRequest{ColumnIDs: []string{code.ID.String()}})
	must(t, err)

	amount, err := db.AddColumn(ctx, "s", dto.AddColumnRequest{ModelID: ord.Model.ID, BaseID: baseID, Title: "Amount", UIDT: "decimal", OrderIndex: ptr(1.0)})
	must(t, err)
	status, err := db.AddColumn(ctx, "s", dto.AddColumnRequest{ModelID: ord.Model.ID, BaseID: baseID, Title: "Status", UIDT: "select", OrderIndex: ptr(2.0),
		Meta: map[string]interface{}{"options": []interface{}{map[string]interface{}{"option": "New", "color": "#fff"}}}})
	must(t, err)
	_, err = db.AddColumn(ctx, "s", dto.AddColumnRequest{ModelID: ord.Model.ID, BaseID: baseID, Title: "Files", UIDT: "attachment", OrderIndex: ptr(3.0)})
	must(t, err)
	// formula fields keep their text, which names other fields by title
	_, err = db.AddColumn(ctx, "s", dto.AddColumnRequest{ModelID: ord.Model.ID, BaseID: baseID, Title: "Total with tax", UIDT: "formula", OrderIndex: ptr(3.5),
		Meta: map[string]interface{}{"formula": "MULTIPLY({Amount},1.18)"}})
	must(t, err)

	orderLink, err := db.AddColumn(ctx, "s", dto.AddColumnRequest{ModelID: cust.Model.ID, BaseID: baseID, Title: "Orders", UIDT: "links", OrderIndex: ptr(4.0),
		Meta: map[string]interface{}{
			"relation":          map[string]interface{}{"with": ord.Model.ID.String(), "type": "has-many", "inverse_title": "Customer"},
			"display_column_id": ord.Columns[0].ID.String(),
		}})
	must(t, err)
	_, err = db.AddColumn(ctx, "s", dto.AddColumnRequest{ModelID: ord.Model.ID, BaseID: baseID, Title: "Customer name", UIDT: "lookup", OrderIndex: ptr(5.0),
		Meta: map[string]interface{}{"lookup_column_id": name.ID.String(), "relation_id": orderLink.Meta["relation_id"].(uuid.UUID).String()}})
	must(t, err)

	// a view that points at a column by id, in a key and in a value
	_, err = db.CreateView(ctx, "s", dto.CreateViewRequest{ModelID: ord.Model.ID, BaseID: baseID, Title: "Board", Type: "kanban", OrderIndex: ptr(1.0),
		Meta: &map[string]interface{}{"view_target_field": status.ID.String(), "hidden": map[string]interface{}{name.ID.String(): true}}})
	must(t, err)

	// many-to-many with tags, a self link, and one field of several other types
	tags, _ := db.CreateTableWithDefaultsImport(ctx, dto.CreateTableRequest{BaseID: baseID.String(), Title: "Tags", OrderIndex: 3}, "s")
	_, err = db.AddColumn(ctx, "s", dto.AddColumnRequest{ModelID: tags.Model.ID, BaseID: baseID, Title: "Label", UIDT: "text", OrderIndex: ptr(1.0)})
	must(t, err)
	tagLink, err := db.AddColumn(ctx, "s", dto.AddColumnRequest{ModelID: ord.Model.ID, BaseID: baseID, Title: "Tags", UIDT: "links", OrderIndex: ptr(6.0),
		Meta: map[string]interface{}{"relation": map[string]interface{}{"with": tags.Model.ID.String(), "type": "many-to-many", "inverse_title": "Tagged orders"}}})
	must(t, err)
	referrals, err := db.AddColumn(ctx, "s", dto.AddColumnRequest{ModelID: cust.Model.ID, BaseID: baseID, Title: "Referrals", UIDT: "links", OrderIndex: ptr(5.0),
		Meta: map[string]interface{}{"relation": map[string]interface{}{"with": cust.Model.ID.String(), "type": "has-many"}}})
	must(t, err)
	for _, col := range []dto.AddColumnRequest{
		{Title: "Notes", UIDT: "longText"}, {Title: "Done", UIDT: "boolean"}, {Title: "Due", UIDT: "date"}, {Title: "Phone", UIDT: "phoneNumber"},
		{Title: "Ref", UIDT: "uuid", Meta: map[string]interface{}{"idGeneration": map[string]interface{}{"mode": "auto", "version": "v7"}}},
		{Title: "External id", UIDT: "uuid", Meta: map[string]interface{}{"idGeneration": map[string]interface{}{"mode": "manual"}}},
		{Title: "Labels", UIDT: "multiSelect", Meta: map[string]interface{}{"options": []interface{}{map[string]interface{}{"option": "A"}, map[string]interface{}{"option": "B"}}}},
	} {
		col.ModelID, col.BaseID, col.OrderIndex = tags.Model.ID, baseID, ptr(2.0)
		_, err = db.AddColumn(ctx, "s", col)
		must(t, err)
	}

	c1 := db.addRow(cust.Model.ID.String(), "", map[string]interface{}{"Name": "Acme", "Code": "C-001"})
	c2 := db.addRow(cust.Model.ID.String(), "", map[string]interface{}{"Name": "Globex", "Code": "C-002"})
	o1 := db.addRow(ord.Model.ID.String(), "", map[string]interface{}{"Amount": 120.5, "Status": "New", "Total with tax": "142.19",
		"Files": []interface{}{db.storeFile("invoice.pdf", "application/pdf", invoicePDF)}})
	o2 := db.addRow(ord.Model.ID.String(), "", map[string]interface{}{"Amount": 80.0})
	g1 := db.addRow(tags.Model.ID.String(), "", map[string]interface{}{"Label": "VIP", "Notes": "long text", "Done": true, "Due": "2026-12-01", "Phone": "+911234567890", "Labels": []interface{}{"A", "B"},
		"Ref": "018f0000-0000-7000-8000-000000000001", "External id": "6d1f2c3e-1a2b-4c3d-8e9f-0a1b2c3d4e5f"})
	_, err = db.UpdateRawDataForLinks(ctx, "s", dto.UpdateRowDataLinksRequest{ModelID: ord.Model.ID.String(), ColumnId: tagLink.ID.String(), SourceRowId: o1, TargetRowId: g1, Action: "link"})
	must(t, err)
	_, err = db.UpdateRawDataForLinks(ctx, "s", dto.UpdateRowDataLinksRequest{ModelID: cust.Model.ID.String(), ColumnId: referrals.ID.String(), SourceRowId: c1, TargetRowId: c2, Action: "link"})
	must(t, err)
	for _, o := range []int{o1, o2} {
		_, err = db.UpdateRawDataForLinks(ctx, "s", dto.UpdateRowDataLinksRequest{ModelID: cust.Model.ID.String(), ColumnId: orderLink.ID.String(), SourceRowId: c1, TargetRowId: o, Action: "link"})
		must(t, err)
	}
	// constraints, and a cover image
	cons := fakeConstraints{db: db}
	_, err = cons.Create(ctx, "s", dto.CreateTableConstraintRequest{ModelID: cust.Model.ID.String(), Type: "unique", Title: "Unique name", ColumnIDs: []string{name.ID.String()}})
	must(t, err)
	_, err = cons.Create(ctx, "s", dto.CreateTableConstraintRequest{ModelID: ord.Model.ID.String(), Type: "check", Title: "Amount not negative",
		CheckInput: dto.CheckInput{Category: "number_date", Condition: amount.ColumnName + " >= 0"}})
	must(t, err)
	db.setCover(baseID.String(), "cover.png", coverPNG)

	// a trigger function and a trigger on Orders, written with the real names of this base
	autos := fakeAutomations{db: db}
	ordersAlias := db.models[ord.Model.ID.String()].Alias
	amountCol := amount.ColumnName
	_, err = autos.Create(ctx, "s", dto.CreateAutomationRequest{ModelID: ord.Model.ID.String(), Title: "Round amount", Type: "function",
		Context: `CREATE FUNCTION "s".round_amount() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN NEW.` + amountCol + ` := ROUND(NEW.` + amountCol + `, 2); ` +
			`INSERT INTO "s"."` + ordersAlias + `"(id) SELECT 1 WHERE false; RETURN NEW; END; $$`})
	must(t, err)
	_, err = autos.Create(ctx, "s", dto.CreateAutomationRequest{ModelID: ord.Model.ID.String(), Title: "Round on write", Type: "trigger", Event: "BEFORE INSERT, BEFORE UPDATE",
		Context: `CREATE TRIGGER round_on_write BEFORE INSERT OR UPDATE ON "s"."` + ordersAlias + `" FOR EACH ROW EXECUTE FUNCTION "s".round_amount()`})
	must(t, err)

	return &crm{db: db, baseID: baseID.String(), customers: cust.Model.ID.String(), orders: ord.Model.ID.String(), name: name.ID.String(), orderLink: orderLink.ID.String()}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func newService(db *fakeDB) *transferSvc {
	return &transferSvc{svc: transfer.NewBaseTransferService(db, db, fakePK{db: db}, db, fakeAssets{db: db}, fakeConstraints{db: db}, fakeAutomations{db: db})}
}

type transferSvc struct {
	svc interfaces.BaseTransferService
}

// roundTrip marshals and parses the file, as the HTTP download and upload do.
func roundTrip(t *testing.T, f *dto.BaseExportFile) *dto.BaseExportFile {
	t.Helper()
	raw, err := json.Marshal(f)
	must(t, err)
	var out dto.BaseExportFile
	must(t, json.Unmarshal(raw, &out))
	return &out
}

func TestExportHasNoIDsAndPassesValidation(t *testing.T) {
	c := seedCRM(t)
	svc := newService(c.db)
	file, err := svc.svc.ExportBase(context.Background(), "s", c.baseID, dto.ExportBaseOptions{IncludeData: true})
	must(t, err)

	raw, _ := json.Marshal(file)
	for _, id := range []string{c.customers, c.orders, c.name, c.baseID} {
		if strings.Contains(string(raw), id) {
			t.Fatalf("export still holds database id %s:\n%s", id, raw)
		}
	}
	if problems := transfer.ValidateBaseFile(roundTrip(t, file)); len(problems) > 0 {
		t.Fatalf("own export is not valid: %v", problems)
	}
	if len(file.Schema.Tables) != 3 || file.Schema.Tables[0].Title != "Customers" {
		t.Fatalf("tables = %+v", file.Schema.Tables)
	}
	if got := file.Data.RowCounts; got["t_customers"] != 2 || got["t_orders"] != 2 || got["t_tags"] != 1 {
		t.Fatalf("row counts = %v", got)
	}
	// the link cell of the first customer holds the _rid of both orders
	first := file.Data.Tables["t_customers"].Rows[0]
	if links, _ := first.Values["f_orders"].([]string); len(links) != 2 || links[0] != "t_orders_1" {
		t.Fatalf("link cell = %#v", first.Values["f_orders"])
	}
	if len(file.Skipped) == 0 || !strings.Contains(strings.Join(file.Skipped, " "), "attachment") {
		t.Fatalf("expected an attachment warning, got %v", file.Skipped)
	}
}

func TestExportWithoutData(t *testing.T) {
	c := seedCRM(t)
	file, err := newService(c.db).svc.ExportBase(context.Background(), "s", c.baseID, dto.ExportBaseOptions{})
	must(t, err)
	if file.Data != nil {
		t.Fatal("structure-only export must have no data")
	}
	if problems := transfer.ValidateBaseFile(roundTrip(t, file)); len(problems) > 0 {
		t.Fatalf("problems: %v", problems)
	}
}

// normalized drops what legitimately differs between two exports of the same content.
func normalized(t *testing.T, f *dto.BaseExportFile) string {
	t.Helper()
	cp := roundTrip(t, f)
	cp.Base.Title = ""
	raw, err := json.MarshalIndent(cp, "", " ")
	must(t, err)
	return string(raw)
}

func TestRoundTripExportImportExport(t *testing.T) {
	c := seedCRM(t)
	svc := newService(c.db)
	ctx := context.Background()

	first, err := svc.svc.ExportBase(ctx, "s", c.baseID, dto.ExportBaseOptions{IncludeData: true})
	must(t, err)

	res, err := svc.svc.ImportBase(ctx, "s", workspace, "user-1", roundTrip(t, first), dto.ImportBaseOptions{IncludeData: true})
	must(t, err)
	if res.BaseID == "" || res.BaseID == c.baseID {
		t.Fatalf("import must create a new base, got %q", res.BaseID)
	}
	if res.Title != "CRM (2)" {
		t.Fatalf("title = %q, want a unique title", res.Title)
	}
	if res.Tables != 3 || res.Rows != 5 || res.Views != 4 {
		t.Fatalf("result = %+v", res)
	}

	second, err := svc.svc.ExportBase(ctx, "s", res.BaseID, dto.ExportBaseOptions{IncludeData: true})
	must(t, err)
	if a, b := normalized(t, first), normalized(t, second); a != b {
		t.Fatalf("export after import differs from the original: %s", firstDifference(a, b))
	}

	// the original base is untouched, and ids in the new base are its own
	if len(c.db.rows[c.customers]) != 2 {
		t.Fatal("original rows changed")
	}
	var newOrders string
	for id, m := range c.db.models {
		if m.BaseID.String() == res.BaseID && m.Title == "Orders" {
			newOrders = id
		}
	}
	for _, col := range c.db.columns[newOrders] {
		if col.UIDT == "lookup" {
			target, ok := c.db.columnByID(col.Meta["lookup_column_id"].(string))
			if !ok || target.BaseID.String() != res.BaseID || target.Title != "Name" {
				t.Fatalf("lookup points at %+v, want the Name field of the new base", target)
			}
		}
	}
}

func TestImportStructureOnly(t *testing.T) {
	c := seedCRM(t)
	svc := newService(c.db)
	ctx := context.Background()
	file, err := svc.svc.ExportBase(ctx, "s", c.baseID, dto.ExportBaseOptions{IncludeData: true})
	must(t, err)

	res, err := svc.svc.ImportBase(ctx, "s", workspace, "u", roundTrip(t, file), dto.ImportBaseOptions{IncludeData: false, Title: "Template"})
	must(t, err)
	if res.Title != "Template" || res.Rows != 0 {
		t.Fatalf("result = %+v", res)
	}
	for id, m := range c.db.models {
		if m.BaseID.String() == res.BaseID && len(c.db.rows[id]) != 0 {
			t.Fatalf("table %s has rows although data was switched off", m.Title)
		}
	}
}

func TestImportFailureRemovesEverything(t *testing.T) {
	c := seedCRM(t)
	svc := newService(c.db)
	file, _ := svc.svc.ExportBase(context.Background(), "s", c.baseID, dto.ExportBaseOptions{IncludeData: true})
	bases, models := len(c.db.bases), len(c.db.models)

	c.db.failOnUIDT = "lookup" // fails after tables, plain fields and links exist
	_, err := svc.svc.ImportBase(context.Background(), "s", workspace, "u", roundTrip(t, file), dto.ImportBaseOptions{IncludeData: true})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("err = %v", err)
	}
	if len(c.db.bases) != bases || len(c.db.models) != models {
		t.Fatalf("partial base left behind: %d bases (want %d), %d tables (want %d)", len(c.db.bases), bases, len(c.db.models), models)
	}
}

func TestValidationReportsProblems(t *testing.T) {
	c := seedCRM(t)
	svc := newService(c.db)
	file, _ := svc.svc.ExportBase(context.Background(), "s", c.baseID, dto.ExportBaseOptions{IncludeData: true})

	cases := map[string]func(f *dto.BaseExportFile){
		"duplicate key": func(f *dto.BaseExportFile) { f.Schema.Tables[1].Key = f.Schema.Tables[0].Key },
		"unknown field type": func(f *dto.BaseExportFile) {
			for i := range f.Schema.Tables[0].Fields {
				if !f.Schema.Tables[0].Fields[i].System {
					f.Schema.Tables[0].Fields[i].UIDT = "hologram"
					return
				}
			}
		},
		"missing ref": func(f *dto.BaseExportFile) {
			f.Schema.Tables[1].Views[0].Meta = map[string]interface{}{"x": "@f_ghost"}
		},
		"unknown rid": func(f *dto.BaseExportFile) {
			r := f.Data.Tables["t_customers"].Rows[0]
			r.Values["f_orders"] = []string{"nope"}
		},
		"row count": func(f *dto.BaseExportFile) { f.Data.RowCounts["t_orders"] = 9 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			cp := roundTrip(t, file)
			mutate(cp)
			if len(transfer.ValidateBaseFile(cp)) == 0 {
				t.Fatal("expected a problem")
			}
			_, err := svc.svc.ImportBase(context.Background(), "s", workspace, "u", cp, dto.ImportBaseOptions{IncludeData: true})
			var invalid *dto.ImportValidationError
			if !errors.As(err, &invalid) {
				t.Fatalf("err = %v, want a validation error", err)
			}
		})
	}
}

// The example file in the docs must import.
func TestFullExampleFile(t *testing.T) {
	raw, err := os.ReadFile("../../../docs/import-export-schema/examples/full.sbase.json")
	if err != nil {
		t.Skip("example not present")
	}
	var file dto.BaseExportFile
	must(t, json.Unmarshal(raw, &file))
	if problems := transfer.ValidateBaseFile(&file); len(problems) > 0 {
		t.Fatalf("example is not valid: %v", problems)
	}
}

// UPDATE_EXAMPLE=1 go test ./tests/services/transfer -run TestWriteExample regenerates the docs examples
// (full.sbase.json and full.sbase.zip) from a real export.
func TestWriteExample(t *testing.T) {
	if os.Getenv("UPDATE_EXAMPLE") == "" {
		t.Skip("set UPDATE_EXAMPLE=1 to rewrite the files in docs/import-export-schema/examples")
	}
	c := seedCRM(t)
	svc := newService(c.db)
	file, err := svc.svc.ExportBase(context.Background(), "s", c.baseID, dto.ExportBaseOptions{IncludeData: true, IncludeAttachments: true, IncludeAutomations: true})
	must(t, err)
	var buf bytes.Buffer
	must(t, writeArchive(svc.svc, file, &buf))
	dir := "../../../docs/import-export-schema/examples/"
	must(t, os.WriteFile(dir+"full.sbase.zip", buf.Bytes(), 0o644))
	must(t, os.WriteFile(dir+"full.sbase.json", readZipFile(t, openZip(t, buf.Bytes()), "base.sbase.json"), 0o644))
}

// firstDifference shows the first line where two texts differ, with a little context.
func firstDifference(a, b string) string {
	x, y := strings.Split(a, "\n"), strings.Split(b, "\n")
	for i := 0; i < len(x) && i < len(y); i++ {
		if x[i] != y[i] {
			lo := max(0, i-6)
			return fmt.Sprintf("line %d\n--- first\n%s\n--- second\n%s", i+1, strings.Join(x[lo:min(len(x), i+4)], "\n"), strings.Join(y[lo:min(len(y), i+4)], "\n"))
		}
	}
	return fmt.Sprintf("lengths differ: %d vs %d lines", len(x), len(y))
}

// exportZip exports with attachments and packs the zip, as the HTTP download does.
func exportZip(t *testing.T, c *crm, baseID string) (*dto.BaseExportFile, []byte) {
	t.Helper()
	svc := newService(c.db)
	file, err := svc.svc.ExportBase(context.Background(), "s", baseID, dto.ExportBaseOptions{IncludeData: true, IncludeAttachments: true})
	must(t, err)
	var buf bytes.Buffer
	must(t, writeArchive(svc.svc, file, &buf))
	return file, buf.Bytes()
}

func openZip(t *testing.T, raw []byte) *zip.Reader {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	must(t, err)
	return zr
}

func readZipFile(t *testing.T, zr *zip.Reader, name string) []byte {
	t.Helper()
	for _, f := range zr.File {
		if f.Name == name {
			rc, err := f.Open()
			must(t, err)
			defer rc.Close()
			data, err := io.ReadAll(rc)
			must(t, err)
			return data
		}
	}
	t.Fatalf("%s is not in the zip", name)
	return nil
}

func TestExportZipHoldsAttachments(t *testing.T) {
	c := seedCRM(t)
	file, raw := exportZip(t, c, c.baseID)
	zr := openZip(t, raw)

	if len(file.Data.Attachments) != 1 || file.Base.Cover == nil {
		t.Fatalf("attachments = %+v", file.Data)
	}
	a := file.Data.Attachments[0]
	if a.Ref != "a1" || a.Filename != "invoice.pdf" || a.Path != "attachments/a1_invoice.pdf" {
		t.Fatalf("index entry = %+v", a)
	}
	if got := readZipFile(t, zr, a.Path); !bytes.Equal(got, invoicePDF) {
		t.Fatalf("zip file = %q", got)
	}
	// the JSON inside the zip is the same file, with size and checksum filled from the real bytes
	var inZip dto.BaseExportFile
	must(t, json.Unmarshal(readZipFile(t, zr, "base.sbase.json"), &inZip))
	sum := sha256.Sum256(invoicePDF)
	if got := inZip.Data.Attachments[0]; got.SHA256 != "sha256:"+hex.EncodeToString(sum[:]) || got.Size != int64(len(invoicePDF)) {
		t.Fatalf("zip index = %+v", got)
	}
	if problems := transfer.ValidateBaseFile(&inZip); len(problems) > 0 {
		t.Fatalf("problems: %v", problems)
	}
	if cell, _ := inZip.Data.Tables["t_orders"].Rows[0].Values["f_files"].([]interface{}); len(cell) != 1 || cell[0] != "a1" {
		t.Fatalf("attachment cell = %#v", inZip.Data.Tables["t_orders"].Rows[0].Values["f_files"])
	}
	if strings.Contains(string(readZipFile(t, zr, "base.sbase.json")), "base_path") {
		t.Fatal("storage paths must not be written to the file")
	}
}

func importZip(t *testing.T, c *crm, raw []byte) (dto.ImportBaseResult, error) {
	t.Helper()
	zr := openZip(t, raw)
	var file dto.BaseExportFile
	must(t, json.Unmarshal(readZipFile(t, zr, "base.sbase.json"), &file))
	return newService(c.db).svc.ImportBase(context.Background(), "s", workspace, "u", &file, dto.ImportBaseOptions{IncludeData: true, IncludeAttachments: true, Attachments: transfer.NewZipSource(zr)})
}

func TestImportZipRestoresAttachments(t *testing.T) {
	c := seedCRM(t)
	first, raw := exportZip(t, c, c.baseID)
	assetsBefore := len(c.db.assets)

	res, err := importZip(t, c, raw)
	must(t, err)
	if res.Files != 1 || len(c.db.assets) != assetsBefore+2 { // the attachment and the cover image
		t.Fatalf("result = %+v, assets %d -> %d", res, assetsBefore, len(c.db.assets))
	}

	// the new base has its own copy of the file, and exporting it again gives the same file
	second, _ := exportZip(t, c, res.BaseID)
	if a, b := normalized(t, first), normalized(t, second); a != b {
		t.Fatalf("second export differs: %s", firstDifference(a, b))
	}
	var newPaths, oldPaths []string
	for _, a := range c.db.assets {
		newPaths = append(newPaths, a.BasePath)
	}
	for _, r := range c.db.rows[c.orders] {
		files, _ := r[c.db.colByTitle(c.orders, "Files").ColumnName].([]interface{})
		for _, item := range files {
			oldPaths = append(oldPaths, item.(map[string]interface{})["base_path"].(string))
		}
	}
	if len(newPaths) != 4 || len(oldPaths) != 1 {
		t.Fatalf("paths: new %v old %v", newPaths, oldPaths)
	}
}

func TestImportPlainJSONSkipsAttachmentsWithWarning(t *testing.T) {
	c := seedCRM(t)
	file, _ := exportZip(t, c, c.baseID)
	assetsBefore := len(c.db.assets)
	res, err := newService(c.db).svc.ImportBase(context.Background(), "s", workspace, "u", roundTrip(t, file), dto.ImportBaseOptions{IncludeData: true})
	must(t, err)
	if len(c.db.assets) != assetsBefore || res.Files != 0 {
		t.Fatalf("nothing should be uploaded without a zip: %+v", res)
	}
	if !strings.Contains(strings.Join(res.Warnings, " "), "upload the .sbase.zip") {
		t.Fatalf("warnings = %v", res.Warnings)
	}
}

// rewriteZip returns a copy of the zip with one file's bytes replaced, or removed when data is nil.
func rewriteZip(t *testing.T, raw []byte, name string, data []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	src := openZip(t, raw)
	for _, f := range src.File {
		if f.Name == name && data == nil {
			continue
		}
		w, err := zw.Create(f.Name)
		must(t, err)
		content := readZipFile(t, src, f.Name)
		if f.Name == name {
			content = data
		}
		_, err = w.Write(content)
		must(t, err)
	}
	must(t, zw.Close())
	return out.Bytes()
}

func TestImportRejectsDamagedAndMissingFiles(t *testing.T) {
	c := seedCRM(t)
	_, raw := exportZip(t, c, c.baseID)
	bases, assets := len(c.db.bases), len(c.db.assets)

	_, err := importZip(t, c, rewriteZip(t, raw, "attachments/a1_invoice.pdf", []byte("tampered")))
	if err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("damaged file: err = %v", err)
	}
	var invalid *dto.ImportValidationError
	_, err = importZip(t, c, rewriteZip(t, raw, "attachments/a1_invoice.pdf", nil))
	if !errors.As(err, &invalid) || !strings.Contains(err.Error(), "is not in the zip") {
		t.Fatalf("missing file: err = %v", err)
	}
	if len(c.db.bases) != bases || len(c.db.assets) != assets {
		t.Fatalf("a rejected import left something behind: %d bases, %d assets", len(c.db.bases), len(c.db.assets))
	}
}

func TestInfectedAttachmentFailsImportAndCleansUp(t *testing.T) {
	c := seedCRM(t)
	_, raw := exportZip(t, c, c.baseID)
	bases, models, assets := len(c.db.bases), len(c.db.models), len(c.db.assets)

	// the zip holds a file the antivirus rejects; its checksum is fixed up so only the scan can stop it
	zr := openZip(t, raw)
	var file dto.BaseExportFile
	must(t, json.Unmarshal(readZipFile(t, zr, "base.sbase.json"), &file))
	bad := []byte("EICAR test file")
	sum := sha256.Sum256(bad)
	file.Data.Attachments[0].SHA256 = "sha256:" + hex.EncodeToString(sum[:])
	patched, _ := json.Marshal(file)
	raw = rewriteZip(t, rewriteZip(t, raw, "attachments/a1_invoice.pdf", bad), "base.sbase.json", patched)

	_, err := importZip(t, c, raw)
	if err == nil || !strings.Contains(err.Error(), "infected") {
		t.Fatalf("err = %v", err)
	}
	if len(c.db.bases) != bases || len(c.db.models) != models || len(c.db.assets) != assets {
		t.Fatalf("left behind: %d bases, %d tables, %d assets", len(c.db.bases), len(c.db.models), len(c.db.assets))
	}
}

func TestRollbackDeletesUploadedAttachments(t *testing.T) {
	c := seedCRM(t)
	_, raw := exportZip(t, c, c.baseID)
	assets, files := len(c.db.assets), len(c.db.files)
	c.db.failRows = true // fails when rows are inserted, after the attachment was uploaded

	_, err := importZip(t, c, raw)
	if err == nil || !strings.Contains(err.Error(), "rows failed") {
		t.Fatalf("err = %v", err)
	}
	if len(c.db.assets) != assets || len(c.db.files) != files {
		t.Fatalf("uploaded files were not removed: %d assets (want %d), %d files (want %d)", len(c.db.assets), assets, len(c.db.files), files)
	}
}

func TestOversizeAttachmentIsSkippedWithWarning(t *testing.T) {
	c := seedCRM(t)
	// the row cell carries the size
	item := c.db.cell(c.orders, 0, "Files").([]interface{})[0].(map[string]interface{})
	item["size"] = int64(transfer.MaxAttachmentBytes + 1)

	file, raw := exportZip(t, c, c.baseID)
	if len(file.Data.Attachments) != 0 || !strings.Contains(strings.Join(file.Skipped, " "), "larger than 100 MB") {
		t.Fatalf("attachments = %v, warnings = %v", file.Data.Attachments, file.Skipped)
	}
	if _, err := importZip(t, c, raw); err != nil {
		t.Fatalf("a zip with no attachments must still import: %v", err)
	}
}

func findTable(f *dto.BaseExportFile, title string) dto.BaseExportTable {
	for _, tb := range f.Schema.Tables {
		if tb.Title == title {
			return tb
		}
	}
	panic("no table " + title)
}

func TestExportWritesPrimaryKeyConstraintsFormulaAndCover(t *testing.T) {
	c := seedCRM(t)
	file, raw := exportZip(t, c, c.baseID)

	customers, orders := findTable(file, "Customers"), findTable(file, "Orders")
	if got := customers.PrimaryKey; len(got) != 1 || got[0] != "f_code" {
		t.Fatalf("primary key = %v", got)
	}
	if len(orders.PrimaryKey) != 0 {
		t.Fatalf("a table with the default Id key must not write a primary key: %v", orders.PrimaryKey)
	}
	if got := customers.Constraints; len(got) != 1 || got[0].Type != "unique" || got[0].FieldKeys[0] != "f_name" {
		t.Fatalf("customer constraints = %+v", got)
	}
	// the check condition names the field by key, not by the generated column name
	check := orders.Constraints[0]
	if check.Type != "check" || check.Condition != "{f_amount} >= 0" || check.Category != "number_date" {
		t.Fatalf("check = %+v", check)
	}
	if strings.Contains(check.Condition, "amount_") {
		t.Fatalf("condition still holds a column name: %s", check.Condition)
	}
	// formula text is copied as it is: it names fields by title
	for _, f := range orders.Fields {
		if f.UIDT == "formula" && f.Meta["formula"] != "MULTIPLY({Amount},1.18)" {
			t.Fatalf("formula meta = %v", f.Meta)
		}
	}
	// link cells to a table with a custom key are read by that key, not by id
	row := file.Data.Tables["t_orders"].Rows[0]
	if cell, _ := row.Values["f_customer"].([]string); len(cell) != 1 || cell[0] != "t_customers_1" {
		t.Fatalf("link to a custom-key table = %#v", row.Values["f_customer"])
	}

	cover := file.Base.Cover
	if cover == nil || cover.Filename != "cover.png" || cover.Path != "attachments/cover_cover.png" {
		t.Fatalf("cover = %+v", cover)
	}
	zr := openZip(t, raw)
	if got := readZipFile(t, zr, cover.Path); !bytes.Equal(got, coverPNG) {
		t.Fatalf("cover file = %q", got)
	}
	if strings.Contains(string(readZipFile(t, zr, "base.sbase.json")), "storage") {
		t.Fatal("storage details must not be written to the file")
	}
}

func TestImportRestoresPrimaryKeyConstraintsFormulaAndCover(t *testing.T) {
	c := seedCRM(t)
	_, raw := exportZip(t, c, c.baseID)
	res, err := importZip(t, c, raw)
	must(t, err)
	if !res.Cover || res.Constraints != 2 {
		t.Fatalf("result = %+v", res)
	}

	var newCustomers, newOrders string
	for id, m := range c.db.models {
		if m.BaseID.String() != res.BaseID {
			continue
		}
		switch m.Title {
		case "Customers":
			newCustomers = id
		case "Orders":
			newOrders = id
		}
	}
	// primary key: the new Code field, set although the table later got links
	pk := c.db.pk[newCustomers]
	if len(pk) != 1 {
		t.Fatalf("primary key = %v", pk)
	}
	if col, _ := c.db.columnByID(pk[0]); col.Title != "Code" || col.ModelID.String() != newCustomers {
		t.Fatalf("primary key column = %+v", col)
	}
	// constraints use the new base's own column names
	amount := c.db.colByTitle(newOrders, "Amount").ColumnName
	oldAmount := c.db.colByTitle(c.orders, "Amount").ColumnName
	checks := c.db.constraints[newOrders]
	if len(checks) != 1 || checks[0].condition != amount+" >= 0" || amount == oldAmount {
		t.Fatalf("check = %+v (new column %s, old column %s)", checks, amount, oldAmount)
	}
	unique := c.db.constraints[newCustomers]
	if len(unique) != 1 || len(unique[0].columnIDs) != 1 {
		t.Fatalf("unique = %+v", unique)
	}
	if col, _ := c.db.columnByID(unique[0].columnIDs[0]); col.Title != "Name" || col.ModelID.String() != newCustomers {
		t.Fatalf("unique column = %+v", col)
	}
	// formula keeps its text and its value
	var formula dto.ColumnResponse
	for _, col := range c.db.columns[newOrders] {
		if col.UIDT == "formula" {
			formula = col
		}
	}
	if formula.Meta["formula"] != "MULTIPLY({Amount},1.18)" {
		t.Fatalf("formula meta = %v", formula.Meta)
	}
	if got := c.db.rows[newOrders][0][formula.ColumnName]; got != "142.19" {
		t.Fatalf("formula value = %v", got)
	}
	// cover image
	newBase := c.db.bases[res.BaseID]
	if newBase.Image == "" || newBase.Image == c.db.bases[c.baseID].Image {
		t.Fatalf("cover = %q (original %q)", newBase.Image, c.db.bases[c.baseID].Image)
	}
	var stored []byte
	for _, a := range c.db.assets {
		if a.Url == newBase.Image {
			stored = c.db.files[a.BasePath]
		}
	}
	if !bytes.Equal(stored, coverPNG) {
		t.Fatalf("cover file = %q", stored)
	}
}

func TestPlainJSONSkipsCoverWithWarning(t *testing.T) {
	c := seedCRM(t)
	file, _ := exportZip(t, c, c.baseID)
	res, err := newService(c.db).svc.ImportBase(context.Background(), "s", workspace, "u", roundTrip(t, file), dto.ImportBaseOptions{IncludeData: true})
	must(t, err)
	if res.Cover || c.db.bases[res.BaseID].Image != "" {
		t.Fatal("a plain JSON file has no cover file to restore")
	}
	if !strings.Contains(strings.Join(res.Warnings, " "), "cover image was not imported") {
		t.Fatalf("warnings = %v", res.Warnings)
	}
}

func TestExportWithoutFilesWarnsAboutCover(t *testing.T) {
	c := seedCRM(t)
	file, err := newService(c.db).svc.ExportBase(context.Background(), "s", c.baseID, dto.ExportBaseOptions{IncludeData: true})
	must(t, err)
	if file.Base.Cover != nil || !strings.Contains(strings.Join(file.Skipped, " "), "cover image not exported") {
		t.Fatalf("cover = %+v, warnings = %v", file.Base.Cover, file.Skipped)
	}
}

func TestStructureOnlyZipStillCarriesCover(t *testing.T) {
	c := seedCRM(t)
	svc := newService(c.db)
	file, err := svc.svc.ExportBase(context.Background(), "s", c.baseID, dto.ExportBaseOptions{IncludeAttachments: true})
	must(t, err)
	if file.Data != nil || file.Base.Cover == nil {
		t.Fatalf("a template export must hold the cover and no rows: data %v, cover %v", file.Data, file.Base.Cover)
	}
	var buf bytes.Buffer
	must(t, writeArchive(svc.svc, file, &buf))
	zr := openZip(t, buf.Bytes())
	var parsed dto.BaseExportFile
	must(t, json.Unmarshal(readZipFile(t, zr, "base.sbase.json"), &parsed))
	res, err := svc.svc.ImportBase(context.Background(), "s", workspace, "u", &parsed, dto.ImportBaseOptions{IncludeAttachments: true, Attachments: transfer.NewZipSource(zr)})
	must(t, err)
	if !res.Cover || res.Rows != 0 {
		t.Fatalf("result = %+v", res)
	}
}

func TestCoverThatCannotBeSetIsOnlyAWarning(t *testing.T) {
	c := seedCRM(t)
	_, raw := exportZip(t, c, c.baseID)
	zr := openZip(t, raw)
	var file dto.BaseExportFile
	must(t, json.Unmarshal(readZipFile(t, zr, "base.sbase.json"), &file))
	file.Base.Cover.Filename = "cover.bad" // the fake base service refuses this type, like the real one does for bad images
	res, err := newService(c.db).svc.ImportBase(context.Background(), "s", workspace, "u", &file, dto.ImportBaseOptions{IncludeData: true, IncludeAttachments: true, Attachments: transfer.NewZipSource(zr)})
	must(t, err)
	if res.Cover || !strings.Contains(strings.Join(res.Warnings, " "), "unsupported image type") {
		t.Fatalf("result = %+v", res)
	}
}

func TestRollbackRemovesCoverImage(t *testing.T) {
	c := seedCRM(t)
	_, raw := exportZip(t, c, c.baseID)
	assets, files := len(c.db.assets), len(c.db.files)
	c.db.failRows = true
	if _, err := importZip(t, c, raw); err == nil {
		t.Fatal("expected the import to fail")
	}
	if len(c.db.assets) != assets || len(c.db.files) != files {
		t.Fatalf("cover or attachment left behind: %d assets (want %d), %d files (want %d)", len(c.db.assets), assets, len(c.db.files), files)
	}
}

func TestImportRejectsMissingCoverFile(t *testing.T) {
	c := seedCRM(t)
	_, raw := exportZip(t, c, c.baseID)
	_, err := importZip(t, c, rewriteZip(t, raw, "attachments/cover_cover.png", nil))
	var invalid *dto.ImportValidationError
	if !errors.As(err, &invalid) || !strings.Contains(err.Error(), "base.cover") {
		t.Fatalf("err = %v", err)
	}
}

func TestValidationOfPrimaryKeyAndConstraints(t *testing.T) {
	c := seedCRM(t)
	file, err := newService(c.db).svc.ExportBase(context.Background(), "s", c.baseID, dto.ExportBaseOptions{IncludeData: true})
	must(t, err)

	cases := map[string]func(f *dto.BaseExportFile){
		"key of another table": func(f *dto.BaseExportFile) { f.Schema.Tables[0].PrimaryKey = []string{"f_amount"} },
		"unknown key field":    func(f *dto.BaseExportFile) { f.Schema.Tables[0].PrimaryKey = []string{"f_ghost"} },
		"key on a link field":  func(f *dto.BaseExportFile) { f.Schema.Tables[0].PrimaryKey = []string{"f_orders"} },
		"key listed twice":     func(f *dto.BaseExportFile) { f.Schema.Tables[0].PrimaryKey = []string{"f_code", "f_code"} },
		"unique without fields": func(f *dto.BaseExportFile) {
			f.Schema.Tables[0].Constraints[0].FieldKeys = nil
		},
		"unique on another table's field": func(f *dto.BaseExportFile) {
			f.Schema.Tables[0].Constraints[0].FieldKeys = []string{"f_amount"}
		},
		"check without condition": func(f *dto.BaseExportFile) { f.Schema.Tables[1].Constraints[0].Condition = "" },
		"check with unknown field": func(f *dto.BaseExportFile) {
			f.Schema.Tables[1].Constraints[0].Condition = "{f_name} > 0" // a field of the other table
		},
		"unknown constraint type": func(f *dto.BaseExportFile) { f.Schema.Tables[0].Constraints[0].Type = "primary" },
		"cover with a bad path":   func(f *dto.BaseExportFile) { f.Base.Cover = &dto.BaseExportCover{Filename: "c.png", Path: "../c.png"} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			cp := roundTrip(t, file)
			mutate(cp)
			if len(transfer.ValidateBaseFile(cp)) == 0 {
				t.Fatal("expected a problem")
			}
		})
	}
}

// toLegacyViews moves the views of every table into the top-level list of format 1.0.0.
func toLegacyViews(f *dto.BaseExportFile) {
	for i := range f.Schema.Tables {
		for _, v := range f.Schema.Tables[i].Views {
			v.Table = f.Schema.Tables[i].Key
			f.Schema.Views = append(f.Schema.Views, v)
		}
		f.Schema.Tables[i].Views = nil
	}
}

func TestExportWritesViewsInsideTables(t *testing.T) {
	c := seedCRM(t)
	file, err := newService(c.db).svc.ExportBase(context.Background(), "s", c.baseID, dto.ExportBaseOptions{IncludeData: true})
	must(t, err)
	if len(file.Schema.Views) != 0 {
		t.Fatalf("top-level views %d", len(file.Schema.Views))
	}
	if got := len(findTable(file, "Orders").Views); got != 2 {
		t.Fatalf("Orders has %d views, want the default grid and the board", got)
	}
	raw, _ := json.Marshal(file)
	var generic struct {
		Schema struct {
			Tables []struct {
				Views []map[string]interface{} `json:"views"`
			} `json:"tables"`
		} `json:"schema"`
	}
	must(t, json.Unmarshal(raw, &generic))
	for _, tb := range generic.Schema.Tables {
		for _, v := range tb.Views {
			if _, has := v["table"]; has {
				t.Fatalf("a view inside a table must not repeat its table: %v", v)
			}
		}
	}
}

func TestImportStillReadsTheTopLevelViewListOfFormat10(t *testing.T) {
	c := seedCRM(t)
	svc := newService(c.db)
	ctx := context.Background()
	first, err := svc.svc.ExportBase(ctx, "s", c.baseID, dto.ExportBaseOptions{IncludeData: true})
	must(t, err)

	legacy := roundTrip(t, first)
	toLegacyViews(legacy)
	if problems := transfer.ValidateBaseFile(legacy); len(problems) > 0 {
		t.Fatalf("a 1.0.0 file must still be valid: %v", problems)
	}
	res, err := svc.svc.ImportBase(ctx, "s", workspace, "u", legacy, dto.ImportBaseOptions{IncludeData: true})
	must(t, err)
	if res.Views != 4 {
		t.Fatalf("views = %d", res.Views)
	}
	second, err := svc.svc.ExportBase(ctx, "s", res.BaseID, dto.ExportBaseOptions{IncludeData: true})
	must(t, err)
	if a, b := normalized(t, first), normalized(t, second); a != b {
		t.Fatalf("importing the old layout changed the result: %s", firstDifference(a, b))
	}
}

func TestViewListsMustNotDisagree(t *testing.T) {
	c := seedCRM(t)
	file, err := newService(c.db).svc.ExportBase(context.Background(), "s", c.baseID, dto.ExportBaseOptions{IncludeData: true})
	must(t, err)

	both := roundTrip(t, file)
	dup := both.Schema.Tables[1].Views[1]
	dup.Table = "t_orders"
	both.Schema.Views = []dto.BaseExportView{dup} // the same view in the table and in the top-level list
	if problems := transfer.ValidateBaseFile(both); len(problems) == 0 || !strings.Contains(strings.Join(problems, " "), "already used") {
		t.Fatalf("problems = %v", problems)
	}

	wrong := roundTrip(t, file)
	wrong.Schema.Tables[1].Views[0].Table = "t_customers" // names another table than the one it is in
	if problems := transfer.ValidateBaseFile(wrong); len(problems) == 0 || !strings.Contains(strings.Join(problems, " "), "does not match") {
		t.Fatalf("problems = %v", problems)
	}

	// a view that names its own table is fine
	same := roundTrip(t, file)
	same.Schema.Tables[1].Views[0].Table = "t_orders"
	if problems := transfer.ValidateBaseFile(same); len(problems) > 0 {
		t.Fatalf("problems = %v", problems)
	}
}

func TestFileHasNoFormatBlock(t *testing.T) {
	c := seedCRM(t)
	file, _ := exportZip(t, c, c.baseID)
	raw, _ := json.Marshal(file)
	var generic map[string]json.RawMessage
	must(t, json.Unmarshal(raw, &generic))
	if len(generic) != 3 {
		t.Fatalf("the file must hold only base, schema and data: %v", generic)
	}
	for _, key := range []string{"base", "schema", "data"} {
		if _, ok := generic[key]; !ok {
			t.Fatalf("the file has no %s", key)
		}
	}
	if len(file.Skipped) != 0 {
		t.Fatalf("nothing should have been skipped: %v", file.Skipped)
	}
}

func TestJSONThatIsNotABaseIsRejectedWithAClearMessage(t *testing.T) {
	problems := transfer.ValidateBaseFile(&dto.BaseExportFile{})
	if len(problems) != 1 || !strings.Contains(problems[0], "not a base export file") {
		t.Fatalf("problems = %v", problems)
	}
}

func TestZipHoldsOnlyTheFileAndItsFiles(t *testing.T) {
	c := seedCRM(t)
	// an unsupported field gives the report something to say
	c.db.addColumn(dto.AddColumnRequest{ModelID: uuid.MustParse(c.orders), BaseID: c.db.models[c.orders].BaseID, UIDT: "rollup", OrderIndex: ptr(9.0)}, "Margin", map[string]interface{}{})
	file, raw := exportZip(t, c, c.baseID)
	zr := openZip(t, raw)

	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	sort.Strings(names)
	want := []string{"attachments/a1_invoice.pdf", "attachments/cover_cover.png", "base.sbase.json"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("zip entries = %v, want %v", names, want)
	}
	// what the export left out is only kept in memory (and logged): not in the zip and not in the file
	if len(file.Skipped) != 1 || !strings.Contains(file.Skipped[0], "rollup") {
		t.Fatalf("skipped = %v", file.Skipped)
	}
	if strings.Contains(string(readZipFile(t, zr, "base.sbase.json")), "warnings") {
		t.Fatal("warnings must not be inside the file")
	}
	if _, err := importZip(t, c, raw); err != nil {
		t.Fatalf("the zip must import: %v", err)
	}
}

func TestFilesWithAnOldFormatBlockStillImport(t *testing.T) {
	c := seedCRM(t)
	svc := newService(c.db)
	file, err := svc.svc.ExportBase(context.Background(), "s", c.baseID, dto.ExportBaseOptions{IncludeData: true})
	must(t, err)
	raw, _ := json.Marshal(file)

	var generic map[string]interface{}
	must(t, json.Unmarshal(raw, &generic))
	// files of the older layouts carry a format block; it is ignored
	generic["format"] = map[string]interface{}{
		"name": "serenibase.base", "version": "1.1.0", "exported_at": "2026-11-03T09:00:00Z",
		"include_data": true, "include_attachments": false, "warnings": []string{"from an old exporter"},
	}
	old, _ := json.Marshal(generic)

	var parsed dto.BaseExportFile
	must(t, json.Unmarshal(old, &parsed))
	if problems := transfer.ValidateBaseFile(&parsed); len(problems) > 0 {
		t.Fatalf("a 1.x file must still be valid: %v", problems)
	}
	res, err := svc.svc.ImportBase(context.Background(), "s", workspace, "u", &parsed, dto.ImportBaseOptions{IncludeData: true})
	must(t, err)
	if res.Rows != 5 {
		t.Fatalf("result = %+v", res)
	}
}

func TestValidationEnforcesTheLimitsOfTheSchema(t *testing.T) {
	c := seedCRM(t)
	file, _ := exportZip(t, c, c.baseID)
	long := strings.Repeat("x", 256)

	cases := map[string]func(f *dto.BaseExportFile){
		"base title too long":  func(f *dto.BaseExportFile) { f.Base.Title = long },
		"table title too long": func(f *dto.BaseExportFile) { f.Schema.Tables[0].Title = long },
		"field title too long": func(f *dto.BaseExportFile) { f.Schema.Tables[0].Fields[1].Title = long },
		"view title too long":  func(f *dto.BaseExportFile) { f.Schema.Tables[1].Views[0].Title = long },
		"rid too long": func(f *dto.BaseExportFile) {
			r := f.Data.Tables["t_customers"].Rows[0]
			r.RID = strings.Repeat("r", 129)
			f.Data.Tables["t_customers"].Rows[0] = r
		},
		"attachment filename too long": func(f *dto.BaseExportFile) { f.Data.Attachments[0].Filename = long },
		"attachment bad sha256":        func(f *dto.BaseExportFile) { f.Data.Attachments[0].SHA256 = "sha256:abc" },
		"cover bad sha256":             func(f *dto.BaseExportFile) { f.Base.Cover.SHA256 = "md5:abc" },
		"cover filename too long":      func(f *dto.BaseExportFile) { f.Base.Cover.Filename = long },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			cp := roundTrip(t, file)
			mutate(cp)
			if len(transfer.ValidateBaseFile(cp)) == 0 {
				t.Fatal("expected a problem")
			}
		})
	}
	// the exporter's own values are within every limit
	if problems := transfer.ValidateBaseFile(roundTrip(t, file)); len(problems) > 0 {
		t.Fatalf("problems = %v", problems)
	}
}

func TestBaseHasNoTypeAndOlderFilesWithOneStillImport(t *testing.T) {
	c := seedCRM(t)
	svc := newService(c.db)
	file, err := svc.svc.ExportBase(context.Background(), "s", c.baseID, dto.ExportBaseOptions{IncludeData: true})
	must(t, err)
	raw, _ := json.Marshal(file)
	var generic struct {
		Base map[string]interface{} `json:"base"`
	}
	must(t, json.Unmarshal(raw, &generic))
	if _, has := generic.Base["type"]; has {
		t.Fatalf("base must not have a type: %v", generic.Base)
	}

	// an older file has base.type; it is read and ignored
	var withType map[string]interface{}
	must(t, json.Unmarshal(raw, &withType))
	withType["base"].(map[string]interface{})["type"] = "default"
	old, _ := json.Marshal(withType)
	var parsed dto.BaseExportFile
	must(t, json.Unmarshal(old, &parsed))
	if problems := transfer.ValidateBaseFile(&parsed); len(problems) > 0 {
		t.Fatalf("problems = %v", problems)
	}
	if _, err := svc.svc.ImportBase(context.Background(), "s", workspace, "u", &parsed, dto.ImportBaseOptions{IncludeData: true}); err != nil {
		t.Fatal(err)
	}
}

func TestFieldsCarryTheDatabaseTypeButImportIgnoresIt(t *testing.T) {
	c := seedCRM(t)
	svc := newService(c.db)
	ctx := context.Background()
	file, err := svc.svc.ExportBase(ctx, "s", c.baseID, dto.ExportBaseOptions{IncludeData: true})
	must(t, err)

	// the exporter writes the type the field had in the database
	want := map[string]string{"Name": "TEXT", "Amount": "NUMERIC", "Status": "TEXT", "Orders": "JSONB", "Files": "JSONB[]", "Labels": "TEXT[]", "Customer name": "lookup", "Id": "BIGSERIAL"}
	seen := 0
	for _, tb := range file.Schema.Tables {
		for _, f := range tb.Fields {
			if dt, ok := want[f.Title]; ok {
				seen++
				if f.DT != dt {
					t.Errorf("field %s (%s): dt = %q, want %q", f.Title, f.UIDT, f.DT, dt)
				}
			}
		}
	}
	if seen < len(want) {
		t.Fatalf("only %d of the expected fields were found", seen)
	}

	// import does not read it: a wrong or made-up dt changes nothing, because the app derives it from uidt
	wrong := roundTrip(t, file)
	for i := range wrong.Schema.Tables {
		for j := range wrong.Schema.Tables[i].Fields {
			wrong.Schema.Tables[i].Fields[j].DT = "BOGUS"
		}
	}
	if problems := transfer.ValidateBaseFile(wrong); len(problems) > 0 {
		t.Fatalf("a dt value must not be validated: %v", problems)
	}
	res, err := svc.svc.ImportBase(ctx, "s", workspace, "u", wrong, dto.ImportBaseOptions{IncludeData: true})
	must(t, err)
	for id, m := range c.db.models {
		if m.BaseID.String() != res.BaseID {
			continue
		}
		for _, col := range c.db.columns[id] {
			if col.DT == "BOGUS" {
				t.Fatalf("field %s took the dt from the file", col.Title)
			}
		}
	}
	again, err := svc.svc.ExportBase(ctx, "s", res.BaseID, dto.ExportBaseOptions{IncludeData: true})
	must(t, err)
	if a, b := normalized(t, file), normalized(t, again); a != b {
		t.Fatalf("the copy differs from the original: %s", firstDifference(a, b))
	}
}

func exportAutomations(t *testing.T, c *crm, baseID string) *dto.BaseExportFile {
	t.Helper()
	file, err := newService(c.db).svc.ExportBase(context.Background(), "s", baseID, dto.ExportBaseOptions{IncludeData: true, IncludeAutomations: true})
	must(t, err)
	return file
}

func TestExportWritesTriggersAndFunctionsWithPlaceholders(t *testing.T) {
	c := seedCRM(t)
	file := exportAutomations(t, c, c.baseID)
	orders := findTable(file, "Orders")
	if len(orders.Functions) != 1 || len(orders.Triggers) != 1 {
		t.Fatalf("functions %d, triggers %d", len(orders.Functions), len(orders.Triggers))
	}
	fn, tr := orders.Functions[0], orders.Triggers[0]
	if fn.Key != "fn_round_amount" || fn.Title != "Round amount" {
		t.Fatalf("function = %+v", fn)
	}
	if tr.Title != "Round on write" || tr.Event != "BEFORE INSERT, BEFORE UPDATE" {
		t.Fatalf("trigger = %+v", tr)
	}

	// names that differ in a copy are written as placeholders
	for _, part := range []string{"{schema}", "{fn_round_amount}", "{f_amount}", "{t_orders}"} {
		if !strings.Contains(fn.Context, part) && !strings.Contains(tr.Context, part) {
			t.Errorf("no %s in the SQL:\n%s\n%s", part, fn.Context, tr.Context)
		}
	}
	realAlias := c.db.models[c.orders].Alias
	realAmount := c.db.colByTitle(c.orders, "Amount").ColumnName
	for _, text := range []string{fn.Context, tr.Context} {
		for _, real := range []string{realAlias, realAmount, `"s".`, "round_amount()"} {
			if strings.Contains(text, real) {
				t.Errorf("the SQL still holds %q:\n%s", real, text)
			}
		}
	}
	if !strings.Contains(tr.Context, "{fn_round_amount}()") {
		t.Errorf("the trigger must call the function by its placeholder:\n%s", tr.Context)
	}
	if problems := transfer.ValidateBaseFile(roundTrip(t, file)); len(problems) > 0 {
		t.Fatalf("problems: %v", problems)
	}
}

func TestTriggersAndFunctionsAreOnlyExportedWhenAsked(t *testing.T) {
	c := seedCRM(t)
	file, err := newService(c.db).svc.ExportBase(context.Background(), "s", c.baseID, dto.ExportBaseOptions{IncludeData: true})
	must(t, err)
	for _, tb := range file.Schema.Tables {
		if len(tb.Functions) != 0 || len(tb.Triggers) != 0 {
			t.Fatalf("table %s has automations although they were not asked for", tb.Title)
		}
	}
}

// ownFunctionName is the name of the trigger function a trigger of the table calls.
func calledFunction(t *testing.T, c *crm, modelID string) string {
	t.Helper()
	for _, a := range c.db.automations[modelID] {
		if a.Type == "trigger" {
			if m := fakeExecuteFunction.FindStringSubmatch(a.Context); m != nil {
				return m[1]
			}
		}
	}
	t.Fatal("no trigger")
	return ""
}

func tableOf(c *crm, baseID, title string) string {
	for id, m := range c.db.models {
		if m.BaseID.String() == baseID && m.Title == title {
			return id
		}
	}
	return ""
}

func TestImportGivesEachCopyItsOwnFunctionAndRebuildsTheSQL(t *testing.T) {
	c := seedCRM(t)
	file := exportAutomations(t, c, c.baseID)
	functionsBefore := len(c.db.functions)

	res, err := newService(c.db).svc.ImportBase(context.Background(), "s", workspace, "u", roundTrip(t, file), dto.ImportBaseOptions{IncludeData: true, IncludeAutomations: true})
	must(t, err)
	if res.Functions != 1 || res.Triggers != 1 {
		t.Fatalf("result = %+v", res)
	}
	// the original's function is still there, and the copy has one of its own (the fake refuses a name that exists)
	if len(c.db.functions) != functionsBefore+1 || !c.db.functions["round_amount"] {
		t.Fatalf("functions = %v", c.db.functions)
	}
	newOrders := tableOf(c, res.BaseID, "Orders")
	name := calledFunction(t, c, newOrders)
	if !regexp.MustCompile(`^round_amount_[0-9a-f]{8}$`).MatchString(name) || !c.db.functions[name] {
		t.Fatalf("the copy's trigger calls %q", name)
	}
	if calledFunction(t, c, c.orders) != "round_amount" {
		t.Fatal("the original's trigger must still call the original's function")
	}

	// the SQL was rebuilt with the copy's own table and column names
	newAlias := c.db.models[newOrders].Alias
	newAmount := c.db.colByTitle(newOrders, "Amount").ColumnName
	for _, a := range c.db.automations[newOrders] {
		if a.Type == "trigger" && !strings.Contains(a.Context, `"`+newAlias+`"`) {
			t.Errorf("trigger is not on the new table %s:\n%s", newAlias, a.Context)
		}
		if a.Type == "function" && (!strings.Contains(a.Context, "NEW."+newAmount) || strings.Contains(a.Context, c.db.colByTitle(c.orders, "Amount").ColumnName)) {
			t.Errorf("function does not use the new column %s:\n%s", newAmount, a.Context)
		}
		if strings.ContainsAny(a.Context, "{}") {
			t.Errorf("placeholder left:\n%s", a.Context)
		}
	}

	// importing the copy again does not pile up suffixes
	again := exportAutomations(t, c, res.BaseID)
	res2, err := newService(c.db).svc.ImportBase(context.Background(), "s", workspace, "u", roundTrip(t, again), dto.ImportBaseOptions{IncludeData: true, IncludeAutomations: true})
	must(t, err)
	if name2 := calledFunction(t, c, tableOf(c, res2.BaseID, "Orders")); !regexp.MustCompile(`^round_amount_[0-9a-f]{8}$`).MatchString(name2) || name2 == name {
		t.Fatalf("second copy calls %q (first copy %q)", name2, name)
	}
}

func TestImportDoesNotCreateTriggersAndFunctionsUnlessAsked(t *testing.T) {
	c := seedCRM(t)
	file := exportAutomations(t, c, c.baseID)
	functionsBefore := len(c.db.functions)
	res, err := newService(c.db).svc.ImportBase(context.Background(), "s", workspace, "u", roundTrip(t, file), dto.ImportBaseOptions{IncludeData: true})
	must(t, err)
	if res.Functions != 0 || res.Triggers != 0 || len(c.db.functions) != functionsBefore {
		t.Fatalf("result = %+v, functions = %v", res, c.db.functions)
	}
	if len(c.db.automations[tableOf(c, res.BaseID, "Orders")]) != 0 {
		t.Fatal("nothing may be created without include_automations")
	}
	if !strings.Contains(strings.Join(res.Warnings, " "), "include_automations") {
		t.Fatalf("warnings = %v", res.Warnings)
	}
}

func TestFailedTriggerRemovesTheFunctionsAndTheBase(t *testing.T) {
	c := seedCRM(t)
	file := exportAutomations(t, c, c.baseID)
	bases, models, functions := len(c.db.bases), len(c.db.models), len(c.db.functions)
	c.db.failTrigger = true // the function is created first, then the trigger fails

	_, err := newService(c.db).svc.ImportBase(context.Background(), "s", workspace, "u", roundTrip(t, file), dto.ImportBaseOptions{IncludeData: true, IncludeAutomations: true})
	if err == nil || !strings.Contains(err.Error(), "trigger failed") {
		t.Fatalf("err = %v", err)
	}
	if len(c.db.bases) != bases || len(c.db.models) != models || len(c.db.functions) != functions {
		t.Fatalf("left behind: %d bases (want %d), %d tables (want %d), functions %v", len(c.db.bases), bases, len(c.db.models), models, c.db.functions)
	}
}

func TestExportThenImportKeepsTheSameTriggersAndFunctions(t *testing.T) {
	c := seedCRM(t)
	first := exportAutomations(t, c, c.baseID)
	res, err := newService(c.db).svc.ImportBase(context.Background(), "s", workspace, "u", roundTrip(t, first), dto.ImportBaseOptions{IncludeData: true, IncludeAutomations: true})
	must(t, err)
	second := exportAutomations(t, c, res.BaseID)

	// the function of the copy has a name of its own, so its key differs; everything else is the same
	keys := regexp.MustCompile(`fn_[a-z0-9_]+`)
	a := keys.ReplaceAllString(normalized(t, first), "fn_X")
	b := keys.ReplaceAllString(normalized(t, second), "fn_X")
	if a != b {
		t.Fatalf("the copy differs from the original: %s", firstDifference(a, b))
	}
}

func TestValidationOfTriggersAndFunctions(t *testing.T) {
	c := seedCRM(t)
	file := exportAutomations(t, c, c.baseID)

	cases := map[string]func(f *dto.BaseExportFile){
		"unknown function placeholder": func(f *dto.BaseExportFile) { f.Schema.Tables[1].Triggers[0].Context += " -- {fn_ghost}" },
		"unknown table placeholder":    func(f *dto.BaseExportFile) { f.Schema.Tables[1].Triggers[0].Context += " -- {t_ghost}" },
		"unknown field placeholder":    func(f *dto.BaseExportFile) { f.Schema.Tables[1].Functions[0].Context += " -- {f_ghost}" },
		"function key used twice": func(f *dto.BaseExportFile) {
			f.Schema.Tables[0].Functions = append(f.Schema.Tables[0].Functions, f.Schema.Tables[1].Functions[0])
		},
		"function key is not a key": func(f *dto.BaseExportFile) { f.Schema.Tables[1].Functions[0].Key = "Not A Key" },
		"function without CREATE":   func(f *dto.BaseExportFile) { f.Schema.Tables[1].Functions[0].Context = "SELECT 1" },
		"function without a title":  func(f *dto.BaseExportFile) { f.Schema.Tables[1].Functions[0].Title = " " },
		"trigger without an event":  func(f *dto.BaseExportFile) { f.Schema.Tables[1].Triggers[0].Event = "" },
		"trigger without SQL":       func(f *dto.BaseExportFile) { f.Schema.Tables[1].Triggers[0].Context = "" },
		"timing written twice": func(f *dto.BaseExportFile) {
			c := f.Schema.Tables[1].Triggers[0].Context
			f.Schema.Tables[1].Triggers[0].Context = strings.Replace(c, "BEFORE INSERT OR UPDATE", "BEFORE INSERT OR BEFORE UPDATE", 1)
		},
		"same trigger title twice": func(f *dto.BaseExportFile) {
			tb := &f.Schema.Tables[1]
			tb.Triggers = append(tb.Triggers, tb.Triggers[0])
		},
		"SQL too long": func(f *dto.BaseExportFile) { f.Schema.Tables[1].Triggers[0].Context = strings.Repeat("x", 100_001) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			cp := roundTrip(t, file)
			mutate(cp)
			if len(transfer.ValidateBaseFile(cp)) == 0 {
				t.Fatal("expected a problem")
			}
		})
	}
}

func TestAutomationsOfAFieldThatIsNotExportedAreSkipped(t *testing.T) {
	c := seedCRM(t)
	// an unsupported field whose column name the function uses
	margin := c.db.addColumn(dto.AddColumnRequest{ModelID: uuid.MustParse(c.orders), BaseID: c.db.models[c.orders].BaseID, UIDT: "rollup", OrderIndex: ptr(9.0)}, "Margin", map[string]interface{}{})
	a := fakeAutomations{db: c.db}
	_, err := a.Create(context.Background(), "s", dto.CreateAutomationRequest{ModelID: c.orders, Title: "Uses margin", Type: "function",
		Context: `CREATE FUNCTION "s".uses_margin() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN NEW.` + margin.ColumnName + ` := 1; RETURN NEW; END; $$`})
	must(t, err)

	file := exportAutomations(t, c, c.baseID)
	for _, fn := range findTable(file, "Orders").Functions {
		if fn.Title == "Uses margin" {
			t.Fatal("a function that uses a field that is not exported cannot be rebuilt and must be skipped")
		}
	}
	if !strings.Contains(strings.Join(file.Skipped, " "), "Uses margin") {
		t.Fatalf("skipped = %v", file.Skipped)
	}
	if problems := transfer.ValidateBaseFile(roundTrip(t, file)); len(problems) > 0 {
		t.Fatalf("problems: %v", problems)
	}
}

// The import result is what the API returns, so its JSON names are part of the contract.
func TestImportResultFieldNames(t *testing.T) {
	raw, err := json.Marshal(dto.ImportBaseResult{BaseID: "b", Title: "T", Tables: 1, Fields: 2, Views: 3, Rows: 4, Files: 5, Cover: true, Constraints: 6, Functions: 7, Triggers: 8, Warnings: []string{"w"}})
	must(t, err)
	var got map[string]interface{}
	must(t, json.Unmarshal(raw, &got))
	for _, name := range []string{"base_id", "title", "tables", "fields", "views", "rows", "attachments", "cover_image", "constraints", "functions", "triggers", "warnings"} {
		if _, ok := got[name]; !ok {
			t.Errorf("the result has no %q: %s", name, raw)
		}
	}
}

func TestAutoUUIDValuesAreNotExportedAndNewRowsGetTheirOwn(t *testing.T) {
	c := seedCRM(t)
	svc := newService(c.db)
	ctx := context.Background()
	file, err := svc.svc.ExportBase(ctx, "s", c.baseID, dto.ExportBaseOptions{IncludeData: true})
	must(t, err)

	// the system makes the value of an auto uuid field, so it is not in the file; a manual uuid travels like any value
	row := file.Data.Tables["t_tags"].Rows[0]
	if _, has := row.Values["f_ref"]; has {
		t.Fatalf("an auto uuid value must not be exported: %v", row.Values)
	}
	if row.Values["f_external_id"] != "6d1f2c3e-1a2b-4c3d-8e9f-0a1b2c3d4e5f" {
		t.Fatalf("a manual uuid must be exported: %v", row.Values)
	}

	// importing works (the real service refuses a written auto uuid), and each new row gets its own value
	res, err := svc.svc.ImportBase(ctx, "s", workspace, "u", roundTrip(t, file), dto.ImportBaseOptions{IncludeData: true})
	must(t, err)
	newTags := tableOf(c, res.BaseID, "Tags")
	ref := c.db.cell(newTags, 0, "Ref")
	if ref == nil || ref == "018f0000-0000-7000-8000-000000000001" {
		t.Fatalf("the copy must get a new uuid, got %v", ref)
	}
	if got := c.db.cell(newTags, 0, "External id"); got != "6d1f2c3e-1a2b-4c3d-8e9f-0a1b2c3d4e5f" {
		t.Fatalf("a manual uuid must be copied, got %v", got)
	}

	// a file that does hold a value for an auto uuid field (an older export) still imports: the value is ignored
	withValue := roundTrip(t, file)
	r := withValue.Data.Tables["t_tags"].Rows[0]
	r.Values["f_ref"] = "018f0000-0000-7000-8000-000000000001"
	withValue.Data.Tables["t_tags"].Rows[0] = r
	if _, err := svc.svc.ImportBase(ctx, "s", workspace, "u", withValue, dto.ImportBaseOptions{IncludeData: true}); err != nil {
		t.Fatalf("a value for an auto uuid field must be ignored, not fail the import: %v", err)
	}
}

func TestTheSQLFormOfAnEventNamesTheTimingOnce(t *testing.T) {
	for event, want := range map[string]string{
		"BEFORE INSERT, BEFORE UPDATE": "BEFORE INSERT OR UPDATE",
		"BEFORE UPDATE":                "BEFORE UPDATE",
		"AFTER INSERT, AFTER DELETE":   "AFTER INSERT OR DELETE",
	} {
		if got := triggerEventSQL(event); got != want {
			t.Errorf("%s -> %q, want %q", event, got, want)
		}
	}
	// every trigger of the complete example is written that way
	db, baseID := completeBase(t)
	file, _ := exportComplete(t, db, baseID)
	for _, tb := range file.Schema.Tables {
		for _, tr := range tb.Triggers {
			if regexp.MustCompile(`(?i)\bOR\s+(BEFORE|AFTER)\b`).MatchString(tr.Context) {
				t.Errorf("trigger %q repeats its timing: %s", tr.Title, tr.Context)
			}
		}
	}
}

func TestExportBaseTableFilter(t *testing.T) {
	c := seedCRM(t)
	svc := newService(c.db)
	ctx := context.Background()

	// 1. Export only Customers table
	file, err := svc.svc.ExportBase(ctx, "s", c.baseID, dto.ExportBaseOptions{
		IncludeData:        true,
		IncludeAttachments: true,
		IncludeAutomations: true,
		Tables:             []string{"Customers"},
	})
	must(t, err)

	if len(file.Schema.Tables) != 1 || file.Schema.Tables[0].Title != "Customers" {
		t.Fatalf("expected only Customers table, got %d tables: %+v", len(file.Schema.Tables), file.Schema.Tables)
	}
	if file.Data == nil || len(file.Data.Tables) != 1 {
		t.Fatalf("expected data for only 1 table, got: %+v", file.Data)
	}

	// Validate the exported file - it should be completely valid even though Orders was excluded
	if problems := transfer.ValidateBaseFile(roundTrip(t, file)); len(problems) > 0 {
		t.Fatalf("validation failed for table-filtered export: %v", problems)
	}

	// 2. Exporting an unknown table returns TableNotFound
	_, err = svc.svc.ExportBase(ctx, "s", c.baseID, dto.ExportBaseOptions{
		Tables: []string{"NonExistentTable"},
	})
	if !errors.Is(err, app_errors.TableNotFound) {
		t.Fatalf("expected TableNotFound, got %v", err)
	}
}

func TestExportBaseFiltersDataAttachmentsAndAutomations(t *testing.T) {
	c := seedCRM(t)
	svc := newService(c.db)
	ctx := context.Background()

	// 1. IncludeData: true, IncludeAttachments: false
	noAtt, err := svc.svc.ExportBase(ctx, "s", c.baseID, dto.ExportBaseOptions{
		IncludeData:        true,
		IncludeAttachments: false,
	})
	must(t, err)
	if noAtt.Data == nil || len(noAtt.Data.Tables) == 0 {
		t.Fatal("data should be present")
	}
	if len(noAtt.Data.Attachments) != 0 {
		t.Fatalf("attachments should be empty when IncludeAttachments is false, got %d", len(noAtt.Data.Attachments))
	}
	if noAtt.Base.Cover != nil {
		t.Fatal("cover should be nil when IncludeAttachments is false")
	}

	// 2. IncludeAutomations: false
	db, baseID := completeBase(t)
	fullSvc := newService(db)
	noAuto, err := fullSvc.svc.ExportBase(ctx, "s", baseID, dto.ExportBaseOptions{
		IncludeAutomations: false,
	})
	must(t, err)
	for _, tbl := range noAuto.Schema.Tables {
		if len(tbl.Functions) > 0 || len(tbl.Triggers) > 0 {
			t.Fatalf("automations should be empty when IncludeAutomations is false in %s", tbl.Title)
		}
	}

	// 3. IncludeAutomations: true
	withAuto, err := fullSvc.svc.ExportBase(ctx, "s", baseID, dto.ExportBaseOptions{
		IncludeAutomations: true,
	})
	must(t, err)
	hasFnOrTrig := false
	for _, tbl := range withAuto.Schema.Tables {
		if len(tbl.Functions) > 0 || len(tbl.Triggers) > 0 {
			hasFnOrTrig = true
			break
		}
	}
	if !hasFnOrTrig {
		t.Fatal("expected automations (functions or triggers) to be exported")
	}
}

// ExportDependencies tells which table needs which, so the dialog can warn before a table is left out.
func TestExportDependenciesFindsLinks(t *testing.T) {
	c := seedCRM(t)
	got, err := newService(c.db).svc.ExportDependencies(context.Background(), "s", c.baseID, true)
	must(t, err)
	if len(got.Tables) != 3 {
		t.Fatalf("tables = %+v", got.Tables)
	}
	found := false
	for _, d := range got.Dependencies {
		if d.Kind == dto.ExportDependencyLink && d.DependsOnID != d.TableID {
			found = true
		}
		if d.TableID == d.DependsOnID {
			t.Fatalf("a table never depends on itself: %+v", d)
		}
	}
	if !found {
		t.Fatalf("expected a link dependency between Customers and Orders, got %+v", got.Dependencies)
	}
}
