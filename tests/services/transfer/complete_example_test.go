// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package transfer_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/aptlogica/sereni-base/internal/dto"
	transfer "github.com/aptlogica/sereni-base/internal/services/transfer"
	"github.com/google/uuid"
)

const completeZipPath = "../../../docs/import-export-schema/examples/complete.sbase.zip"

// completeBase builds one base that uses every feature of the file format:
//
//	6 tables, every field type, links of every kind (has-many, one-to-one, many-to-many, and a self-link of each kind),
//	lookups through each kind of link (the app does not allow a lookup to read a link or another lookup), a formula, a custom primary key, unique and check constraints,
//	trigger functions and triggers, every view type, attachments and a cover image, and rows that fill all of it.
func completeBase(t *testing.T) (db *fakeDB, baseID string) {
	t.Helper()
	ctx := context.Background()
	db = newFakeDB()
	base, _ := db.CreateBaseWithoutTable(ctx, dto.CreateBaseRequest{Title: "Everything", Description: ptr("One base that uses every part of the export format"), WorkspaceID: workspace}, "s", "u")
	bid := base.ID

	tables := map[string]dto.TableResponse{}
	for i, title := range []string{"Customers", "Orders", "Tags", "Profiles", "Projects", "All field types"} {
		tb, _ := db.CreateTableWithDefaultsImport(ctx, dto.CreateTableRequest{BaseID: bid.String(), Title: title, Description: "The " + title + " table", OrderIndex: float64(i + 1)}, "s")
		tables[title] = tb
	}
	model := func(title string) uuid.UUID { return tables[title].Model.ID }
	next := map[string]float64{}
	add := func(table, title, uidt string, meta map[string]interface{}) dto.ColumnResponse {
		next[table]++
		col, err := db.AddColumn(ctx, "s", dto.AddColumnRequest{ModelID: model(table), BaseID: bid, Title: title, UIDT: uidt, OrderIndex: ptr(next[table]), Meta: meta, Description: ""})
		must(t, err)
		return col
	}
	opts := func(labels ...string) map[string]interface{} {
		colors := []string{"#9ca3af", "#22c55e", "#3b82f6", "#f59e0b"}
		var list []interface{}
		for i, l := range labels {
			list = append(list, map[string]interface{}{"option": l, "color": colors[i%len(colors)]})
		}
		return map[string]interface{}{"options": list}
	}
	link := func(table, title, with, kind, inverse string) dto.ColumnResponse {
		rel := map[string]interface{}{"with": model(with).String(), "type": kind}
		if inverse != "" {
			rel["inverse_title"] = inverse
		}
		return add(table, title, "links", map[string]interface{}{"relation": rel})
	}
	lookup := func(table, title string, via dto.ColumnResponse, shown dto.ColumnResponse) dto.ColumnResponse {
		return add(table, title, "lookup", map[string]interface{}{
			"lookup_column_id": shown.ID.String(),
			"relation_id":      via.Meta["relation_id"].(uuid.UUID).String(),
			"link_column_id":   via.ID.String(),
		})
	}
	find := func(table, title string) dto.ColumnResponse { return db.colByTitle(model(table).String(), title) }

	// ---- Customers: a custom primary key first (the app refuses to change a key once links exist)
	add("Customers", "Name", "text", nil)
	code := add("Customers", "Code", "text", nil)
	_, err := fakePK{db: db}.Update(ctx, "s", model("Customers").String(), dto.UpdatePrimaryKeyRequest{ColumnIDs: []string{code.ID.String()}})
	must(t, err)
	add("Customers", "Email", "email", nil)
	add("Customers", "Tier", "select", opts("Basic", "Pro", "Enterprise"))
	add("Customers", "Interests", "multiSelect", opts("Sports", "Music", "Travel"))
	add("Customers", "Credit", "currency", nil)
	add("Customers", "Score", "number", nil)
	add("Customers", "Active", "boolean", map[string]interface{}{"defaultValue": true})
	add("Customers", "Joined", "date", nil)
	add("Customers", "Documents", "attachment", nil)
	add("Customers", "Credit plus score", "formula", map[string]interface{}{"formula": "ADD({Credit},{Score})"})

	// ---- Orders, Tags, Profiles, Projects: plain fields
	add("Orders", "Amount", "decimal", nil)
	add("Orders", "Due", "date", nil)
	add("Orders", "Status", "select", opts("New", "Paid", "Shipped"))
	add("Orders", "Invoice", "attachment", nil)
	add("Orders", "Total with tax", "formula", map[string]interface{}{"formula": "MULTIPLY({Amount},1.18)"})
	add("Tags", "Label", "text", nil)
	add("Profiles", "Bio", "longText", nil)
	add("Profiles", "Avatar", "attachment", nil)
	add("Profiles", "Website", "url", nil)
	add("Projects", "Title", "text", nil)
	add("Projects", "Start", "date", nil)
	add("Projects", "End", "date", nil)
	add("Projects", "Status", "select", opts("Planned", "Active", "Done"))
	add("Projects", "Progress", "percent", nil)
	add("Projects", "Owner", "user", nil)
	add("Projects", "Pictures", "attachment", nil)
	add("Projects", "Budget", "currency", nil)

	// ---- one field of every other type
	const all = "All field types"
	add(all, "Text", "text", nil)
	add(all, "Long text", "longText", nil)
	add(all, "Number", "number", nil)
	add(all, "Decimal", "decimal", nil)
	add(all, "Checkbox", "boolean", nil)
	add(all, "Currency", "currency", nil)
	add(all, "Percent", "percent", nil)
	add(all, "Duration", "duration", nil)
	add(all, "Year", "year", nil)
	add(all, "Date", "date", nil)
	add(all, "Date time", "datetime", nil)
	add(all, "Time", "time", nil)
	add(all, "Email", "email", nil)
	add(all, "Phone number", "phoneNumber", nil)
	add(all, "URL", "url", nil)
	add(all, "Single select", "select", opts("One", "Two", "Three"))
	add(all, "Multi select", "multiSelect", opts("Red", "Green", "Blue"))
	add(all, "Rating", "rating", nil)
	add(all, "User", "user", nil)
	add(all, "Button", "button", nil)
	add(all, "Json", "json", nil)
	add(all, "Auto uuid", "uuid", map[string]interface{}{"idGeneration": map[string]interface{}{"mode": "auto", "version": "v7"}})
	add(all, "Manual uuid", "uuid", map[string]interface{}{"idGeneration": map[string]interface{}{"mode": "manual"}})
	add(all, "Attachment", "attachment", nil)
	add(all, "Formula", "formula", map[string]interface{}{"formula": "ADD({Number},{Decimal})"})
	add(all, "Created time", "createdTime", nil)
	add(all, "Last modified time", "lastModifiedTime", nil)
	add(all, "Created by", "createdBy", nil)
	add(all, "Last modified by", "lastModifiedBy", nil)

	// ---- links: every kind between tables, and every kind from a table to itself
	orders := link("Customers", "Orders", "Orders", "has-many", "Customer")
	profile := link("Customers", "Profile", "Profiles", "one-to-one", "Customer")
	referrals := link("Customers", "Referrals", "Customers", "has-many", "")
	friends := link("Customers", "Friends", "Customers", "many-to-many", "")
	partner := link("Customers", "Partner", "Customers", "one-to-one", "")
	orderTags := link("Orders", "Tags", "Tags", "many-to-many", "Tagged orders")

	// ---- lookups: through each kind of link, and a lookup of a lookup
	lookup("Customers", "Profile bio", profile, find("Profiles", "Bio"))
	lookup("Customers", "Latest order status", orders, find("Orders", "Status"))
	lookup("Orders", "Customer name", find("Orders", "Customer"), find("Customers", "Name"))
	lookup("Orders", "Customer tier", find("Orders", "Customer"), find("Customers", "Tier"))
	lookup("Profiles", "Customer name", find("Profiles", "Customer"), find("Customers", "Name"))
	lookup("Tags", "Order statuses", find("Tags", "Tagged orders"), find("Orders", "Status"))

	// ---- views: every type, with the settings the app stores
	view := func(table, title, kind string, order float64, meta map[string]interface{}) {
		_, err := db.CreateView(ctx, "s", dto.CreateViewRequest{ModelID: model(table), BaseID: bid, Title: title, Type: kind, OrderIndex: ptr(order), Meta: &meta})
		must(t, err)
	}
	hide := func(table string, titles ...string) []interface{} {
		var cfg []interface{}
		for _, ti := range titles {
			cfg = append(cfg, map[string]interface{}{"id": find(table, ti).ID.String(), "isHidden": true})
		}
		return cfg
	}
	view("Customers", "Active customers", "grid", 1, map[string]interface{}{
		"filters":     []interface{}{map[string]interface{}{"column": find("Customers", "Active").ID.String(), "operator": "equals", "value": "true"}},
		"fieldConfig": hide("Customers", "Documents", "Credit plus score"),
	})
	view("Customers", "New customer", "form", 2, map[string]interface{}{
		"fieldConfig": append(hide("Customers", "Score", "Credit", "Joined"), map[string]interface{}{"id": find("Customers", "Name").ID.String(), "isHidden": false}),
	})
	view("Orders", "Order board", "kanban", 1, map[string]interface{}{"view_target_field": find("Orders", "Status").ID.String(), "fieldConfig": hide("Orders", "Total with tax")})
	view("Orders", "Due dates", "calendar", 2, map[string]interface{}{"date_field_id": find("Orders", "Due").ID.String()})
	view("Tags", "Tags by label", "grid", 1, map[string]interface{}{"groupBy": map[string]interface{}{"column": find("Tags", "Label").ID.String()}})
	view("Profiles", "Profile pictures", "gallery", 1, map[string]interface{}{"attachment_field_id": find("Profiles", "Avatar").ID.String()})
	view("Projects", "Timeline", "ganttChart", 1, map[string]interface{}{
		"start_date_field_id": find("Projects", "Start").ID.String(),
		"end_date_field_id":   find("Projects", "End").ID.String(),
		"title_field_id":      find("Projects", "Title").ID.String(),
	})
	view("Projects", "Project board", "kanban", 2, map[string]interface{}{"view_target_field": find("Projects", "Status").ID.String(), "groupBy": map[string]interface{}{"column": find("Projects", "Status").ID.String()}})
	view("Projects", "Project gallery", "gallery", 3, map[string]interface{}{"attachment_field_id": find("Projects", "Pictures").ID.String()})
	view("Projects", "Project calendar", "calendar", 4, map[string]interface{}{"date_field_id": find("Projects", "Start").ID.String()})

	// ---- rows
	row := func(table string, values map[string]interface{}) int {
		return db.addRow(model(table).String(), "", values)
	}
	pdf := func(name string) []interface{} {
		title := strings.TrimSuffix(strings.ReplaceAll(strings.ReplaceAll(name, "-", " "), "_", " "), ".pdf")
		return []interface{}{db.storeFile(name, "application/pdf", realPDF(strings.ToUpper(title[:1])+title[1:], "Part of the SereniBase complete export example.", "File: "+name, "Every attachment in this base is a real file."))}
	}
	png := func(name string) []interface{} {
		return []interface{}{db.storeFile(name, "image/png", realPNG(240, 240, name))}
	}
	c1 := row("Customers", map[string]interface{}{"Name": "Acme Corp", "Code": "C-001", "Email": "hello@acme.io", "Tier": "Enterprise", "Interests": []interface{}{"Sports", "Travel"}, "Credit": 2500.5, "Score": 90, "Active": true, "Joined": "2025-01-15", "Documents": pdf("contract.pdf"), "Credit plus score": "2590.5"})
	c2 := row("Customers", map[string]interface{}{"Name": "Globex", "Code": "C-002", "Email": "sales@globex.io", "Tier": "Pro", "Interests": []interface{}{"Music"}, "Credit": 800.0, "Score": 70, "Active": true, "Joined": "2025-03-02", "Credit plus score": "870"})
	c3 := row("Customers", map[string]interface{}{"Name": "Initech", "Code": "C-003", "Email": "info@initech.com", "Tier": "Basic", "Credit": 120.0, "Score": 40, "Active": false, "Joined": "2025-06-30", "Credit plus score": "160"})
	o1 := row("Orders", map[string]interface{}{"Amount": 120.5, "Due": "2026-12-01", "Status": "New", "Invoice": pdf("invoice-1.pdf"), "Total with tax": "142.19"})
	o2 := row("Orders", map[string]interface{}{"Amount": 80.0, "Due": "2026-12-15", "Status": "Paid", "Invoice": pdf("invoice-2.pdf"), "Total with tax": "94.4"})
	o3 := row("Orders", map[string]interface{}{"Amount": 310.25, "Due": "2027-01-10", "Status": "Shipped", "Total with tax": "366.1"})
	t1 := row("Tags", map[string]interface{}{"Label": "VIP"})
	t2 := row("Tags", map[string]interface{}{"Label": "Wholesale"})
	p1 := row("Profiles", map[string]interface{}{"Bio": "Founder and CEO of Acme.", "Avatar": png("acme.png"), "Website": "https://acme.io"})
	p2 := row("Profiles", map[string]interface{}{"Bio": "Runs partnerships at Globex.", "Website": "https://globex.io"})
	row("Projects", map[string]interface{}{"Title": "Website redesign", "Start": "2026-11-01", "End": "2027-01-31", "Status": "Active", "Progress": 40.0, "Budget": 15000.0, "Pictures": png("mockup.png")})
	row("Projects", map[string]interface{}{"Title": "Mobile app", "Start": "2027-02-01", "End": "2027-06-30", "Status": "Planned", "Progress": 0.0, "Budget": 42000.0})
	row("Projects", map[string]interface{}{"Title": "Data migration", "Start": "2026-09-01", "End": "2026-10-15", "Status": "Done", "Progress": 100.0, "Budget": 8000.0})
	row(all, map[string]interface{}{
		"Text": "Plain text", "Long text": "A longer text\nwith two lines.", "Number": 7, "Decimal": 12.345, "Checkbox": true, "Currency": 99.95, "Percent": 45.5,
		"Duration": "01:30:00", "Year": 2026, "Date": "2026-11-03", "Date time": "2026-11-03T09:00:00Z", "Time": "09:00:00",
		"Email": "someone@example.com", "Phone number": "+911234567890", "URL": "https://www.serenibase.com", "Single select": "Two",
		"Multi select": []interface{}{"Red", "Blue"}, "Rating": 4, "Json": `{"theme":"dark","size":3}`,
		"Auto uuid": "018f0000-0000-7000-8000-000000000001", "Manual uuid": "6d1f2c3e-1a2b-4c3d-8e9f-0a1b2c3d4e5f",
		"Attachment": pdf("all-types.pdf"), "Formula": "19.345",
	})

	linkRows := func(table string, col dto.ColumnResponse, pairs [][2]int) {
		for _, p := range pairs {
			_, err := db.UpdateRawDataForLinks(ctx, "s", dto.UpdateRowDataLinksRequest{ModelID: model(table).String(), ColumnId: col.ID.String(), SourceRowId: p[0], TargetRowId: p[1], Action: "link"})
			must(t, err)
		}
	}
	linkRows("Customers", orders, [][2]int{{c1, o1}, {c1, o2}, {c2, o3}})
	linkRows("Customers", profile, [][2]int{{c1, p1}, {c2, p2}})
	linkRows("Customers", referrals, [][2]int{{c1, c2}, {c1, c3}})
	linkRows("Customers", friends, [][2]int{{c1, c2}, {c2, c3}})
	linkRows("Customers", partner, [][2]int{{c1, c2}})
	linkRows("Orders", orderTags, [][2]int{{o1, t1}, {o1, t2}, {o2, t1}, {o3, t2}})

	// ---- constraints
	cons := fakeConstraints{db: db}
	constraint := func(table string, req dto.CreateTableConstraintRequest) {
		req.ModelID = model(table).String()
		_, err := cons.Create(ctx, "s", req)
		must(t, err)
	}
	constraint("Customers", dto.CreateTableConstraintRequest{Type: "unique", Title: "Unique email", ColumnIDs: []string{find("Customers", "Email").ID.String()}})
	constraint("Projects", dto.CreateTableConstraintRequest{Type: "unique", Title: "One project per title and status", ColumnIDs: []string{find("Projects", "Title").ID.String(), find("Projects", "Status").ID.String()}})
	constraint("Orders", dto.CreateTableConstraintRequest{Type: "check", Title: "Amount is not negative", CheckInput: dto.CheckInput{Category: "number_date", Condition: find("Orders", "Amount").ColumnName + " >= 0"}})
	constraint("Customers", dto.CreateTableConstraintRequest{Type: "check", Title: "Credit is not negative", CheckInput: dto.CheckInput{Category: "number_date", Condition: find("Customers", "Credit").ColumnName + " >= 0"}})
	constraint("Projects", dto.CreateTableConstraintRequest{Type: "check", Title: "Ends after it starts", CheckInput: dto.CheckInput{Category: "compare", Condition: find("Projects", "End").ColumnName + " >= " + find("Projects", "Start").ColumnName}})

	// ---- trigger functions and triggers, written with this base's real names
	autos := fakeAutomations{db: db}
	automation := func(table, fn, fnTitle, body, trigger, trigTitle, event string) {
		alias := db.models[model(table).String()].Alias
		_, err := autos.Create(ctx, "s", dto.CreateAutomationRequest{ModelID: model(table).String(), Title: fnTitle, Type: "function",
			Context: `CREATE FUNCTION "s".` + fn + `() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN ` + body + ` RETURN NEW; END; $$`})
		must(t, err)
		_, err = autos.Create(ctx, "s", dto.CreateAutomationRequest{ModelID: model(table).String(), Title: trigTitle, Type: "trigger", Event: event,
			Context: `CREATE TRIGGER ` + trigger + ` ` + triggerEventSQL(event) + ` ON "s"."` + alias + `" FOR EACH ROW EXECUTE FUNCTION "s".` + fn + `()`})
		must(t, err)
	}
	amount := find("Orders", "Amount").ColumnName
	automation("Orders", "round_amount", "Round amount", "NEW."+amount+" := ROUND(NEW."+amount+", 2);",
		"round_on_write", "Round on write", "BEFORE INSERT, BEFORE UPDATE")
	automation("Customers", "touch_customer", "Touch customer", "NEW.last_modified_time := now();",
		"touch_on_update", "Touch on update", "BEFORE UPDATE")
	progress := find("Projects", "Progress").ColumnName
	automation("Projects", "clamp_progress", "Clamp progress", "NEW."+progress+" := LEAST(GREATEST(NEW."+progress+", 0), 100);",
		"clamp_on_insert", "Clamp on insert", "BEFORE INSERT")

	db.setCover(bid.String(), "cover.png", realPNG(640, 320, "complete cover")) // within the 800x400 limit of a cover
	return db, bid.String()
}

func exportComplete(t *testing.T, db *fakeDB, baseID string) (*dto.BaseExportFile, []byte) {
	t.Helper()
	svc := newService(db)
	file, err := svc.svc.ExportBase(context.Background(), "s", baseID, dto.ExportBaseOptions{IncludeData: true, IncludeAttachments: true, IncludeAutomations: true})
	must(t, err)
	var buf bytes.Buffer
	must(t, writeArchive(svc.svc, file, &buf))
	return file, buf.Bytes()
}

func TestCompleteBaseUsesEveryPartOfTheFormat(t *testing.T) {
	db, baseID := completeBase(t)
	file, raw := exportComplete(t, db, baseID)
	if len(file.Skipped) != 0 {
		t.Fatalf("nothing should be left out: %v", file.Skipped)
	}
	if problems := transfer.ValidateBaseFile(roundTrip(t, file)); len(problems) > 0 {
		t.Fatalf("the complete base must be valid: %v", problems)
	}

	// every field type of the schema
	var schemaTypes []string
	_, defs := loadSchema(t)
	schemaTypes = append(schemaTypes, defs["uidt"].Enum...)
	used := map[string]bool{}
	relationTypes := map[string]bool{}
	selfLinks := map[string]bool{}
	viewTypes := map[string]bool{}
	var functions, triggers, uniques, checks, withPrimaryKey, lookups int
	for _, tb := range file.Schema.Tables {
		for _, f := range tb.Fields {
			used[f.UIDT] = true
			if f.UIDT == "lookup" {
				lookups++
			}
			if f.UIDT == "links" && f.Meta["entity_role"] == "source" {
				rel := f.Meta["relation"].(map[string]interface{})
				relationTypes[rel["type"].(string)] = true
				if rel["with"] == "@"+tb.Key {
					selfLinks[rel["type"].(string)] = true
				}
			}
		}
		for _, v := range tb.Views {
			viewTypes[v.Type] = true
		}
		for _, c := range tb.Constraints {
			if c.Type == "unique" {
				uniques++
			} else {
				checks++
			}
		}
		if len(tb.PrimaryKey) > 0 {
			withPrimaryKey++
		}
		functions += len(tb.Functions)
		triggers += len(tb.Triggers)
	}
	for _, uidt := range schemaTypes {
		if !used[uidt] {
			t.Errorf("field type %q is not in the complete base", uidt)
		}
	}
	for _, kind := range []string{"has-many", "one-to-one", "many-to-many"} {
		if !relationTypes[kind] {
			t.Errorf("no %s link", kind)
		}
		if !selfLinks[kind] {
			t.Errorf("no self-link of kind %s", kind)
		}
	}
	for _, kind := range []string{"grid", "kanban", "calendar", "gallery", "ganttChart", "form"} {
		if !viewTypes[kind] {
			t.Errorf("no %s view", kind)
		}
	}
	if len(file.Schema.Tables) != 6 || lookups < 5 || uniques < 2 || checks < 3 || functions < 3 || triggers < 3 || withPrimaryKey != 1 {
		t.Errorf("tables %d, lookups %d, unique %d, check %d, functions %d, triggers %d, primary keys %d", len(file.Schema.Tables), lookups, uniques, checks, functions, triggers, withPrimaryKey)
	}
	if file.Base.Cover == nil || len(file.Data.Attachments) < 6 {
		t.Errorf("cover %v, attachments %d", file.Base.Cover, len(file.Data.Attachments))
	}
	rows := 0
	for _, n := range file.Data.RowCounts {
		rows += n
	}
	if rows != 14 {
		t.Errorf("only %d rows", rows)
	}

	// the zip holds the JSON and every file, and nothing else
	zr := openZip(t, raw)
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	sort.Strings(names)
	if names[0] != "attachments/a1_contract.pdf" || !contains(names, "base.sbase.json") || !contains(names, "attachments/cover_cover.png") {
		t.Errorf("zip entries = %v", names)
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// The complete base imports into an empty system and exports to the same file again.
func TestCompleteBaseRoundTrips(t *testing.T) {
	db, baseID := completeBase(t)
	first, raw := exportComplete(t, db, baseID)

	zr := openZip(t, raw)
	var parsed dto.BaseExportFile
	must(t, json.Unmarshal(readZipFile(t, zr, "base.sbase.json"), &parsed))
	fresh := newFakeDB() // an empty system: nothing of the original exists there
	res, err := newService(fresh).svc.ImportBase(context.Background(), "s", workspace, "u", &parsed, dto.ImportBaseOptions{IncludeData: true, IncludeAutomations: true, IncludeAttachments: true, Attachments: transfer.NewZipSource(zr)})
	must(t, err)
	if res.Tables != 6 || res.Functions != 3 || res.Triggers != 3 || !res.Cover || res.Files < 6 || len(res.Warnings) != 0 {
		t.Fatalf("result = %+v", res)
	}

	second, err := newService(fresh).svc.ExportBase(context.Background(), "s", res.BaseID, dto.ExportBaseOptions{IncludeData: true, IncludeAttachments: true, IncludeAutomations: true})
	must(t, err)
	keys := regexp.MustCompile(`fn_[a-z0-9_]+`)
	a := keys.ReplaceAllString(normalized(t, first), "fn_X")
	b := keys.ReplaceAllString(normalized(t, second), "fn_X")
	// checksums are filled in when a zip is packed; the second export was not packed, so leave them out of the comparison
	a, b = withoutKey(t, a, "sha256"), withoutKey(t, b, "sha256")
	if a != b {
		t.Fatalf("the copy differs from the original: %s", firstDifference(a, b))
	}
}

// UPDATE_EXAMPLE=1 go test ./tests/services/transfer -run TestWriteCompleteExample rewrites examples/complete.sbase.zip
func TestWriteCompleteExample(t *testing.T) {
	if os.Getenv("UPDATE_EXAMPLE") == "" {
		t.Skip("set UPDATE_EXAMPLE=1 to rewrite docs/import-export-schema/examples/complete.sbase.zip")
	}
	db, baseID := completeBase(t)
	_, raw := exportComplete(t, db, baseID)
	must(t, os.WriteFile(completeZipPath, raw, 0o644))
}

// The committed complete.sbase.zip stays valid and importable.
func TestCommittedCompleteZipImports(t *testing.T) {
	raw, err := os.ReadFile(completeZipPath)
	if err != nil {
		t.Skip("complete.sbase.zip not present")
	}
	zr := openZip(t, raw)
	var parsed dto.BaseExportFile
	must(t, json.Unmarshal(readZipFile(t, zr, "base.sbase.json"), &parsed))
	if problems := transfer.ValidateBaseFile(&parsed); len(problems) > 0 {
		t.Fatalf("complete.sbase.zip is not valid: %v", problems)
	}
	res, err := newService(newFakeDB()).svc.ImportBase(context.Background(), "s", workspace, "u", &parsed, dto.ImportBaseOptions{IncludeData: true, IncludeAutomations: true, IncludeAttachments: true, Attachments: transfer.NewZipSource(zr)})
	must(t, err)
	if res.Tables != 6 || len(res.Warnings) != 0 {
		t.Fatalf("result = %+v", res)
	}
}

// withoutKey removes every property with the given name from a JSON text.
func withoutKey(t *testing.T, text, key string) string {
	t.Helper()
	var doc interface{}
	must(t, json.Unmarshal([]byte(text), &doc))
	var strip func(v interface{})
	strip = func(v interface{}) {
		switch x := v.(type) {
		case map[string]interface{}:
			delete(x, key)
			for _, child := range x {
				strip(child)
			}
		case []interface{}:
			for _, child := range x {
				strip(child)
			}
		}
	}
	strip(doc)
	out, err := json.MarshalIndent(doc, "", " ")
	must(t, err)
	return string(out)
}

// The app refuses a lookup that reads a link, another lookup or a rollup, or a field that is not on the other side of
// its link (TBL_1079, TBL_1077). The import must refuse it up front, instead of the import failing halfway.
func TestValidationRefusesLookupsTheAppWouldRefuse(t *testing.T) {
	db, baseID := completeBase(t)
	file, _ := exportComplete(t, db, baseID)

	lookupOf := func(f *dto.BaseExportFile, table, title string) *dto.BaseExportField {
		for i := range f.Schema.Tables {
			if f.Schema.Tables[i].Title != table {
				continue
			}
			for j := range f.Schema.Tables[i].Fields {
				if f.Schema.Tables[i].Fields[j].Title == title && f.Schema.Tables[i].Fields[j].UIDT == "lookup" {
					return &f.Schema.Tables[i].Fields[j]
				}
			}
		}
		t.Fatalf("no lookup %s.%s", table, title)
		return nil
	}
	fieldKey := func(f *dto.BaseExportFile, table, title string) string {
		for _, tb := range f.Schema.Tables {
			if tb.Title == table {
				for _, fl := range tb.Fields {
					if fl.Title == title {
						return "@" + fl.Key
					}
				}
			}
		}
		t.Fatalf("no field %s.%s", table, title)
		return ""
	}

	cases := map[string]struct {
		point func(f *dto.BaseExportFile) string
		want  string
	}{
		"reads another lookup":             {func(f *dto.BaseExportFile) string { return fieldKey(f, "Customers", "Profile bio") }, "cannot read the lookup field"},
		"reads a link":                     {func(f *dto.BaseExportFile) string { return fieldKey(f, "Customers", "Orders") }, "cannot read the links field"},
		"reads a field of the wrong table": {func(f *dto.BaseExportFile) string { return fieldKey(f, "Orders", "Amount") }, "not on the table its link points at"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			cp := roundTrip(t, file)
			lookupOf(cp, "Orders", "Customer name").Meta["lookup_column_id"] = c.point(cp)
			problems := strings.Join(transfer.ValidateBaseFile(cp), " | ")
			if !strings.Contains(problems, c.want) {
				t.Fatalf("problems = %q, want one that contains %q", problems, c.want)
			}
		})
	}
	// what the app allows still validates
	if problems := transfer.ValidateBaseFile(roundTrip(t, file)); len(problems) > 0 {
		t.Fatalf("problems = %v", problems)
	}
}

// A reader sees the file in a sensible order: system fields first, the cells of a row in the order of the fields, and
// the main key of a field's meta first.
func TestTheFileIsWrittenInAProperOrder(t *testing.T) {
	db, baseID := completeBase(t)
	file, _ := exportComplete(t, db, baseID)
	raw, err := marshalLikeTheWriter(file)
	must(t, err)

	var doc struct {
		Schema struct {
			Tables []struct {
				Key    string `json:"key"`
				Fields []struct {
					Key        string          `json:"key"`
					UIDT       string          `json:"uidt"`
					System     bool            `json:"system"`
					OrderIndex float64         `json:"order_index"`
					Meta       json.RawMessage `json:"meta"`
				} `json:"fields"`
			} `json:"tables"`
		} `json:"schema"`
		Data struct {
			Tables map[string]struct {
				Rows []struct {
					Values json.RawMessage `json:"values"`
				} `json:"rows"`
			} `json:"tables"`
		} `json:"data"`
	}
	must(t, json.Unmarshal(raw, &doc))

	for _, tb := range doc.Schema.Tables {
		// system fields first, then the table's own fields, each group by order_index
		seenOwn, last := false, map[bool]float64{}
		var own []string
		for _, f := range tb.Fields {
			if f.System && seenOwn {
				t.Errorf("table %s: system field %s comes after an own field", tb.Key, f.Key)
			}
			if !f.System {
				seenOwn = true
				own = append(own, f.Key)
			}
			if f.OrderIndex < last[f.System] {
				t.Errorf("table %s: field %s (order %v) is out of order", tb.Key, f.Key, f.OrderIndex)
			}
			last[f.System] = f.OrderIndex

			// the main key of the meta comes first
			switch f.UIDT {
			case "links":
				if keys := keysInOrder(t, f.Meta); keys[0] != "relation" || keys[1] != "relation_id" || keys[2] != "entity_role" {
					t.Errorf("link %s: meta keys %v", f.Key, keys)
				}
			case "lookup":
				if keys := keysInOrder(t, f.Meta); keys[0] != "lookup_column_id" || keys[1] != "relation_id" {
					t.Errorf("lookup %s: meta keys %v", f.Key, keys)
				}
			}
		}
		// the cells of a row follow the order of the fields
		position := map[string]int{}
		for i, k := range own {
			position[k] = i
		}
		for r, row := range doc.Data.Tables[tb.Key].Rows {
			last := -1
			for _, k := range keysInOrder(t, row.Values) {
				if position[k] < last {
					t.Errorf("table %s row %d: cell %s is out of field order", tb.Key, r, k)
				}
				last = position[k]
			}
		}
	}
}

func TestOrderedWritingKeepsEverythingAndReadsBack(t *testing.T) {
	field := dto.BaseExportField{
		Key: "f_x", Title: "X", Description: "d", UIDT: "links", DT: "JSONB", OrderIndex: 2, System: false, ColumnName: "",
		Meta: map[string]interface{}{
			"relation_id": "@r_a", "is_self_link": false, "entity_role": "source", "display_column_id": "@f_id",
			"relation": map[string]interface{}{"with": "@t_a", "type": "has-many"},
			"note":     "a > b & c < d",
		},
	}
	raw, err := marshalLikeTheWriter(field)
	must(t, err)
	var back dto.BaseExportField
	must(t, json.Unmarshal(raw, &back))
	if !reflect.DeepEqual(field, back) {
		t.Fatalf("a field changed on the way:\n%+v\n%+v", field, back)
	}
	// SQL and text with < > & stay readable
	if !strings.Contains(string(raw), "a > b & c < d") {
		t.Fatalf("characters were escaped: %s", raw)
	}
	if keys := keysInOrder(t, json.RawMessage(raw)); keys[0] != "key" || keys[3] != "uidt" || keys[4] != "dt" {
		t.Fatalf("field keys %v", keys)
	}

	row := dto.BaseExportRow{RID: "r1", Values: map[string]interface{}{"f_b": 1, "f_a": "x", "f_c": []string{"r2"}}}
	row.OrderBy([]string{"f_c", "f_a"}) // f_b is not listed: it follows, alphabetically
	raw, err = marshalLikeTheWriter(row)
	must(t, err)
	var values struct {
		Values json.RawMessage `json:"values"`
	}
	must(t, json.Unmarshal(raw, &values))
	if keys := keysInOrder(t, values.Values); !reflect.DeepEqual(keys, []string{"f_c", "f_a", "f_b"}) {
		t.Fatalf("values written as %v", keys)
	}
	var rowBack dto.BaseExportRow
	must(t, json.Unmarshal(raw, &rowBack))
	if rowBack.RID != "r1" || len(rowBack.Values) != 3 {
		t.Fatalf("row read back as %+v", rowBack)
	}
}

// marshalLikeTheWriter encodes the way the export writes files: without escaping <, > and &.
func marshalLikeTheWriter(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// triggerEventSQL turns the app's event ("BEFORE INSERT, BEFORE UPDATE") into the SQL form, where the timing is written
// once: "BEFORE INSERT OR UPDATE". Postgres rejects "BEFORE INSERT OR BEFORE UPDATE".
func triggerEventSQL(event string) string {
	var timing string
	var operations []string
	for _, part := range strings.Split(event, ",") {
		words := strings.Fields(strings.ToUpper(part))
		timing = strings.Join(words[:len(words)-1], " ")
		operations = append(operations, words[len(words)-1])
	}
	return timing + " " + strings.Join(operations, " OR ")
}
