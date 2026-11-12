package platform_test

// Runs the workspace public API against a real Postgres, with the real table services, in a throwaway schema
// that is dropped afterwards. Skipped unless PLATFORM_TEST_PG_PASSWORD is set; PLATFORM_TEST_PG_HOST,
// PLATFORM_TEST_PG_PORT, PLATFORM_TEST_PG_USER and PLATFORM_TEST_PG_DB default to localhost, 5432, postgres
// and serenibase. The database must already have SereniBase's public functions (any database the app has
// started against does).

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aptlogica/go-postgres-rest/pkg"
	dbConfig "github.com/aptlogica/go-postgres-rest/pkg/config"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	appConstant "github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/middleware"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/platform"
	"github.com/aptlogica/sereni-base/internal/scripts"
	"github.com/aptlogica/sereni-base/internal/services"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const e2eSchema = "papi_e2e_tmp"

type apiEnvelope struct {
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details []struct {
			Index *int   `json:"index"`
			Field string `json:"field"`
			ID    string `json:"id"`
			Issue string `json:"issue"`
		} `json:"details"`
	} `json:"error"`
	Meta map[string]interface{} `json:"meta"`
}

type apiRecord struct {
	ID        string                 `json:"id"`
	Key       map[string]interface{} `json:"key"`
	Fields    map[string]interface{} `json:"fields"`
	CreatedAt string                 `json:"created_at"`
	UpdatedAt string                 `json:"updated_at"`
}

// quoteIdent quotes a Postgres identifier
func quoteIdent(name string) string { return `"` + strings.ReplaceAll(name, `"`, `""`) + `"` }

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func TestPublicAPIAgainstPostgres(t *testing.T) {
	password := os.Getenv("PLATFORM_TEST_PG_PASSWORD")
	if password == "" {
		t.Skip("set PLATFORM_TEST_PG_PASSWORD to run against Postgres")
	}
	port, _ := strconv.Atoi(env("PLATFORM_TEST_PG_PORT", "5432"))
	db, err := pkg.NewDatabaseServiceWithInit(&dbConfig.Config{Database: dbConfig.DatabaseConfig{
		Host: env("PLATFORM_TEST_PG_HOST", "localhost"), Driver: "postgres", Port: port,
		Username: env("PLATFORM_TEST_PG_USER", "postgres"), Password: password,
		DatabaseName: env("PLATFORM_TEST_PG_DB", "serenibase"), SSLMode: "disable", MaxOpenConns: 4, MaxIdleConns: 2,
	}})
	require.NoError(t, err)
	ctx := context.Background()
	s := e2eSchema

	exec := func(query string, args ...interface{}) {
		t.Helper()
		_, err := db.DB.Exec(query, args...)
		require.NoError(t, err, query)
	}
	_, _ = db.DB.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS "%s" CASCADE`, s))
	exec(fmt.Sprintf(`CREATE SCHEMA "%s"`, s))
	t.Cleanup(func() { _, _ = db.DB.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS "%s" CASCADE`, s)) })

	// Metadata tables as CreateMasterSchema creates them, in the throwaway schema
	tables := []struct {
		name string
		err  error
	}{
		{"workspaces", db.TableService.CreateTable(tenant.Workspace{}.TableSchema(s))},
		{"workspace_api", db.TableService.CreateTable(tenant.WorkspaceAPI{}.TableSchema(s))},
		{"api_idempotency_keys", db.TableService.CreateTable(tenant.APIIdempotencyKey{}.TableSchema(s))},
		{"bases", db.TableService.CreateTable(tenant.Base{}.TableSchema(s))},
		{"models", db.TableService.CreateTable(tenant.Model{}.TableSchema(s))},
		{"columns", db.TableService.CreateTable(tenant.Column{}.TableSchema(s))},
		{"views", db.TableService.CreateTable(tenant.View{}.TableSchema(s))},
		{"relations", db.TableService.CreateTable(tenant.Relation{}.TableSchema(s))},
		{"table_constraints", db.TableService.CreateTable(tenant.TableConstraint{}.TableSchema(s))},
	}
	for _, tb := range tables {
		require.NoError(t, tb.err, tb.name)
	}
	for _, stmt := range (tenant.WorkspaceAPI{}).CheckConstraints(s) {
		exec(stmt)
	}
	for _, fn := range appConstant.DefinedFunctions {
		if fn.FunctionName == "workspace_api_bump_schema_version" {
			exec(fmt.Sprintf(`CREATE OR REPLACE FUNCTION "%s".%s(%s) %s`, s, fn.FunctionName, fn.FunctionParams, fn.FunctionSQL))
		}
	}
	require.NoError(t, scripts.InstallSchemaVersionTriggers(db, s))

	wsID, baseID := uuid.NewString(), uuid.NewString()
	exec(fmt.Sprintf(`INSERT INTO "%s".workspaces (id, title, slug) VALUES ($1, 'E2E', $2)`, s), wsID, "e2e-"+wsID[:8])
	exec(fmt.Sprintf(`INSERT INTO "%s".bases (id, workspace_id, title) VALUES ($1, $2, 'Main')`, s), baseID, wsID)

	// Tables and fields through the app's own services, so values are stored exactly as the app stores them
	modelSvc := services.NewModelService(db)
	columnSvc := services.NewColumnService(db)
	tableSvc := services.NewTableManagementService("postgres", db, modelSvc, columnSvc,
		services.NewViewService(db), services.NewRelationshipService(db), nil)

	createTable := func(title string) dto.TableResponse {
		t.Helper()
		out, err := tableSvc.CreateTableWithDefaults(ctx, dto.CreateTableRequest{BaseID: baseID, WorkspaceID: wsID, Title: title}, s)
		require.NoError(t, err, title)
		return out
	}
	addColumn := func(table dto.TableResponse, title, uidt string, meta map[string]interface{}) dto.ColumnResponse {
		t.Helper()
		col, err := tableSvc.AddColumn(ctx, s, dto.AddColumnRequest{
			ModelID: table.Model.ID, BaseID: uuid.MustParse(baseID), Title: title, UIDT: uidt, Meta: meta,
		})
		require.NoError(t, err, title)
		return col
	}

	customers := createTable("Customers")
	orders := createTable("Orders")
	addColumn(orders, "Price", "decimal", nil)
	qty := addColumn(orders, "Qty", "number", nil)
	addColumn(orders, "Status", "select", map[string]interface{}{"options": []interface{}{
		map[string]interface{}{"option": "Open", "color": "#0f0"}, map[string]interface{}{"option": "Done", "color": "#00f"}}})
	addColumn(orders, "Tags", "multiSelect", map[string]interface{}{"options": []interface{}{"red", "blue"}})
	addColumn(orders, "Due", "date", nil)
	addColumn(orders, "At", "datetime", nil)
	addColumn(orders, "Paid", "boolean", nil)
	addColumn(orders, "Extra", "json", nil)
	addColumn(orders, "Start", "time", nil)
	addColumn(orders, "Took", "duration", nil)
	link := addColumn(orders, "Customer", "links", map[string]interface{}{
		"relation": map[string]interface{}{"with": customers.Model.ID.String(), "type": "has-many"}})
	customerTitle := customers.Columns[0]
	for _, c := range customers.Columns {
		if (c.System == nil || !*c.System) && c.UIDT == "text" {
			customerTitle = c
		}
	}
	relationID, _ := link.Meta["relation_id"].(string)
	addColumn(orders, "Customer name", "lookup", map[string]interface{}{
		"relation_id": relationID, "lookup_column_id": customerTitle.ID.String(), "link_column_id": link.ID.String()})
	// Orders -> Customer -> Company, for expand two levels deep
	companies := createTable("Companies")
	addColumn(customers, "Company", "links", map[string]interface{}{
		"relation": map[string]interface{}{"with": companies.Model.ID.String(), "type": "has-many"}})
	createTable("In/Out")

	// A table whose primary key was changed in the app (two fields), with a unique constraint and a NOT NULL
	// field, and a table linking to it. Made before the first request, so the API's schema cache sees them.
	products := createTable("Products")
	sku := addColumn(products, "SKU", "text", nil)
	region := addColumn(products, "Region", "text", nil)
	note := addColumn(products, "Note", "text", nil)
	size := addColumn(products, "Size", "text", nil)
	pkSvc := services.NewPrimaryKeyService(db, modelSvc, columnSvc)
	_, err = pkSvc.Update(ctx, s, products.Model.ID.String(), dto.UpdatePrimaryKeyRequest{ColumnIDs: []string{sku.ID.String(), region.ID.String()}})
	require.NoError(t, err)
	_, err = services.NewTableConstraintService(db, modelSvc, columnSvc).Create(ctx, s, dto.CreateTableConstraintRequest{
		ModelID: products.Model.ID.String(), Type: "unique", Title: "One product per note", ColumnIDs: []string{note.ID.String()}})
	require.NoError(t, err)
	exec(fmt.Sprintf(`ALTER TABLE "%s".%q ALTER COLUMN %q SET NOT NULL`, s, products.Model.Alias, size.ColumnName))
	shipments := createTable("Shipments")
	addColumn(shipments, "Product", "links", map[string]interface{}{
		"relation": map[string]interface{}{"with": products.Model.ID.String(), "type": "has-many"}})

	// Primary key changed from id to a UUID field: generated by the system (auto) or entered by people (manual)
	devices := createTable("Devices")
	deviceKey := addColumn(devices, "Device ID", "uuid", map[string]interface{}{"idGeneration": map[string]interface{}{"mode": "auto", "version": "v7"}})
	addColumn(devices, "Label", "text", nil)
	_, err = pkSvc.Update(ctx, s, devices.Model.ID.String(), dto.UpdatePrimaryKeyRequest{ColumnIDs: []string{deviceKey.ID.String()}})
	require.NoError(t, err)
	badges := createTable("Badges")
	badgeKey := addColumn(badges, "Badge ID", "uuid", map[string]interface{}{"idGeneration": map[string]interface{}{"mode": "manual"}})
	addColumn(badges, "Label", "text", nil)
	_, err = pkSvc.Update(ctx, s, badges.Model.ID.String(), dto.UpdatePrimaryKeyRequest{ColumnIDs: []string{badgeKey.ID.String()}})
	require.NoError(t, err)

	// A table with automations, as users write them in the app: one rejects bad values and deletes with its own
	// message, and one has a mistake in its code (a division by zero for the item "broken")
	stock := createTable("Stock")
	stockQty := addColumn(stock, "Qty", "number", nil)
	stockItem := addColumn(stock, "Item", "text", nil)
	stockTable := fmt.Sprintf(`"%s".%q`, s, stock.Model.Alias)
	exec(fmt.Sprintf(`CREATE FUNCTION "%s".stock_guard() RETURNS trigger AS $$ BEGIN
		IF TG_OP = 'DELETE' THEN
			IF OLD.%s > 100 THEN RAISE EXCEPTION 'Items in stock can''t be deleted'; END IF;
			RETURN OLD;
		END IF;
		IF NEW.%s < 0 THEN RAISE EXCEPTION 'Stock can''t go below zero'; END IF;
		IF NEW.%s = 'broken' THEN PERFORM 1 / 0; END IF;
		RETURN NEW;
	END $$ LANGUAGE plpgsql`, s, quoteIdent(stockQty.ColumnName), quoteIdent(stockQty.ColumnName), quoteIdent(stockItem.ColumnName)))
	exec(fmt.Sprintf(`CREATE TRIGGER stock_guard BEFORE INSERT OR UPDATE OR DELETE ON %s FOR EACH ROW EXECUTE FUNCTION "%s".stock_guard()`, stockTable, s))

	// Field settings the app's inputs follow: the email / phone / URL checks on and off, decimals, percent, time
	contacts := createTable("Contacts")
	addColumn(contacts, "Work email", "email", map[string]interface{}{"emailValid": true})
	addColumn(contacts, "Any email", "email", map[string]interface{}{"emailValid": false})
	addColumn(contacts, "Phone", "phoneNumber", map[string]interface{}{"phoneValid": true})
	addColumn(contacts, "Site", "url", map[string]interface{}{"urlValid": true})
	addColumn(contacts, "Any link", "url", nil)
	addColumn(contacts, "Score", "decimal", map[string]interface{}{"precision": "1.00"})
	addColumn(contacts, "Done", "percent", nil)
	addColumn(contacts, "Call at", "time", map[string]interface{}{"hourFormat": "12"})
	contactLevel := addColumn(contacts, "Level", "select", map[string]interface{}{"options": []interface{}{"Gold", "Silver"}})
	contactTags := addColumn(contacts, "Tags", "multiSelect", map[string]interface{}{"options": []interface{}{"vip", "new"}})

	workspaceAPI, err := platform.NewWorkspaceAPIService(db).Enable(ctx, s, wsID)
	require.NoError(t, err)

	// The router as the app builds it: request IDs, raw paths, the /:ws group and the 404 fallback
	gin.SetMode(gin.TestMode)
	recordWriter, ok := tableSvc.(platform.RecordWriter)
	require.True(t, ok)
	api := platform.NewAPI(db.DB, s, recordWriter, 600)
	r := gin.New()
	r.UseRawPath = true
	r.Use(middleware.RequestID())
	api.Register(r.Group("/api/v1"))
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/v1/ws_") {
			platform.NotFound(c)
		}
	})

	base := "/api/v1/" + workspaceAPI.PublicID
	ordersPath := base + "/bases/Main/tables/Orders"
	call := func(method, path string, body interface{}, headers ...string) (int, apiEnvelope, http.Header) {
		t.Helper()
		var reader *bytes.Reader
		if body != nil {
			raw, _ := json.Marshal(body)
			reader = bytes.NewReader(raw)
		} else {
			reader = bytes.NewReader(nil)
		}
		req := httptest.NewRequest(method, path, reader)
		req.Header.Set("Content-Type", "application/json")
		for i := 0; i+1 < len(headers); i += 2 {
			req.Header.Set(headers[i], headers[i+1])
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		var out apiEnvelope
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out), w.Body.String())
		assert.Equal(t, w.Header().Get("X-Request-Id"), out.Meta["request_id"], "meta.request_id matches X-Request-Id")
		return w.Code, out, w.Header()
	}
	records := func(raw json.RawMessage) []apiRecord {
		t.Helper()
		var out []apiRecord
		require.NoError(t, json.Unmarshal(raw, &out), string(raw))
		return out
	}
	record := func(raw json.RawMessage) apiRecord {
		t.Helper()
		var out apiRecord
		require.NoError(t, json.Unmarshal(raw, &out), string(raw))
		return out
	}

	// Two customers, created through the API, for the links
	code, res, _ := call("POST", base+"/bases/Main/tables/Customers/records", map[string]interface{}{
		"records": []interface{}{
			map[string]interface{}{"fields": map[string]interface{}{customerTitle.Title: "Asha"}},
			map[string]interface{}{"fields": map[string]interface{}{customerTitle.Title: "Ravi"}},
		}})
	require.Equal(t, http.StatusCreated, code, res.Error)
	cust := records(res.Data)
	require.Len(t, cust, 2)

	t.Run("schema endpoints", func(t *testing.T) {
		code, res, _ := call("GET", base+"/bases", nil)
		require.Equal(t, http.StatusOK, code)
		assert.Contains(t, string(res.Data), `"name":"Main"`)

		code, res, _ = call("GET", base+"/bases/Main/tables", nil)
		require.Equal(t, http.StatusOK, code)
		assert.Contains(t, string(res.Data), `"name":"Orders"`)
		assert.Contains(t, string(res.Data), `"name":"In/Out"`)

		code, res, _ = call("GET", ordersPath, nil)
		require.Equal(t, http.StatusOK, code)
		body := string(res.Data)
		assert.Contains(t, body, `"name":"Status","type":"single_select","read_only":false,"options":["Open","Done"]`)
		assert.Contains(t, body, `"type":"links"`)
		assert.Contains(t, body, `"lookups":[{"key":"`+customerTitle.Title+`","id":"fld_`)
		assert.NotContains(t, body, `"name":"Customer name"`, "lookups are listed under their link field")
		assert.NotContains(t, body, `"name":"Created Time"`, "system columns are not fields")

		// URL-encoded names reach the handler
		code, _, _ = call("GET", base+"/bases/Main/tables/In%2FOut", nil)
		assert.Equal(t, http.StatusOK, code)
	})

	var first apiRecord
	t.Run("create one with every writable type and a link", func(t *testing.T) {
		code, res, _ := call("POST", ordersPath+"/records", map[string]interface{}{"fields": map[string]interface{}{
			"Title": "First", "Price": 12.5, "Qty": 3, "Status": "Open", "Tags": []string{"red"},
			"Due": "2026-11-03", "At": "2026-11-03T10:15:00+05:30", "Paid": true,
			"Extra": map[string]interface{}{"a": 1}, "Start": "09:30", "Took": 90,
			"Customer": []string{cust[0].ID},
		}})
		require.Equal(t, http.StatusCreated, code, res.Error)
		first = record(res.Data)
		f := first.Fields
		assert.Regexp(t, `^rec_\d+$`, first.ID)
		assert.Equal(t, 12.5, f["Price"])
		assert.Equal(t, float64(3), f["Qty"])
		assert.Equal(t, "Open", f["Status"])
		assert.Equal(t, []interface{}{"red"}, f["Tags"])
		assert.Equal(t, "2026-11-03", f["Due"])
		assert.Equal(t, "2026-11-03T04:45:00Z", f["At"], "stored and returned in UTC")
		assert.Equal(t, true, f["Paid"])
		assert.Equal(t, map[string]interface{}{"a": float64(1)}, f["Extra"])
		assert.Equal(t, "09:30", f["Start"])
		assert.Equal(t, float64(90), f["Took"])
		assert.Equal(t, []interface{}{map[string]interface{}{"id": cust[0].ID, customerTitle.Title: "Asha"}}, f["Customer"],
			"the link carries the lookup through it")
		assert.NotContains(t, f, "Customer name")
		assert.True(t, strings.HasSuffix(first.CreatedAt, "Z"))
	})

	t.Run("openapi spec describes the real tables", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", base+"/openapi.json", nil))
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		doc, err := openapi3.NewLoader().LoadFromData(w.Body.Bytes())
		require.NoError(t, err)
		require.NoError(t, doc.Validate(ctx))
		assert.NotNil(t, doc.Paths.Value("/bases/Main/tables/In%2FOut/records"), "names are URL-encoded in paths")

		// Every key of a real record is a property of the table's record type
		fields := doc.Components.Schemas["Fields_"+platform.EncodeUUID(platform.PrefixTable, orders.Model.ID.String())]
		require.NotNil(t, fields)
		for key := range first.Fields {
			assert.Contains(t, fields.Value.Properties, key)
		}
		assert.Equal(t, []interface{}{"Open", "Done"}, fields.Value.Properties["Status"].Value.Enum)
	})

	t.Run("validation", func(t *testing.T) {
		code, res, _ := call("POST", ordersPath+"/records", map[string]interface{}{"fields": map[string]interface{}{
			"Nope": 1, "Status": "Maybe", "Qty": 1.5, "Customer name": "x"}})
		require.Equal(t, http.StatusUnprocessableEntity, code)
		require.Equal(t, "validation_failed", res.Error.Code)
		issues := map[string]string{}
		for _, d := range res.Error.Details {
			issues[d.Field] = d.Issue
		}
		assert.Equal(t, "unknown field", issues["Nope"])
		assert.Contains(t, issues["Status"], "not one of the field's options")
		assert.Equal(t, "must be a whole number", issues["Qty"])
		assert.Equal(t, "is read-only", issues["Customer name"])

		// Links can't be written in bulk
		code, res, _ = call("POST", ordersPath+"/records", map[string]interface{}{"records": []interface{}{
			map[string]interface{}{"fields": map[string]interface{}{"Customer": []string{cust[0].ID}}}}})
		require.Equal(t, http.StatusUnprocessableEntity, code)
		assert.Equal(t, 0, *res.Error.Details[0].Index)

		// A link to a record that doesn't exist
		code, res, _ = call("POST", ordersPath+"/records", map[string]interface{}{"fields": map[string]interface{}{"Customer": []string{"rec_999999"}}})
		require.Equal(t, http.StatusUnprocessableEntity, code)
		assert.Equal(t, "rec_999999", res.Error.Details[0].ID)

		// Over the bulk limit
		many := make([]interface{}, 101)
		for i := range many {
			many[i] = map[string]interface{}{"fields": map[string]interface{}{}}
		}
		code, res, _ = call("POST", ordersPath+"/records", map[string]interface{}{"records": many})
		assert.Equal(t, http.StatusRequestEntityTooLarge, code)
		assert.Equal(t, "payload_too_large", res.Error.Code)
	})

	t.Run("create many, list, page, filter and sort", func(t *testing.T) {
		in := make([]interface{}, 0, 6)
		for i := 1; i <= 6; i++ {
			status := "Open"
			if i%2 == 0 {
				status = "Done"
			}
			in = append(in, map[string]interface{}{"fields": map[string]interface{}{
				"Title": fmt.Sprintf("Bulk %d", i), "Price": float64(i * 10), "Status": status, "Took": i * 15}})
		}
		code, res, _ := call("POST", ordersPath+"/records", map[string]interface{}{"records": in})
		require.Equal(t, http.StatusCreated, code, res.Error)
		require.Len(t, records(res.Data), 6)

		// 7 records in pages of 3: every record exactly once
		seen := map[string]int{}
		path := ordersPath + "/records?limit=3"
		for pages := 0; ; pages++ {
			require.Less(t, pages, 5)
			code, res, _ := call("GET", path, nil)
			require.Equal(t, http.StatusOK, code, res.Error)
			for _, rec := range records(res.Data) {
				seen[rec.ID]++
			}
			next, _ := res.Meta["next_cursor"].(string)
			if next == "" {
				assert.Equal(t, false, res.Meta["has_more"])
				break
			}
			assert.Equal(t, true, res.Meta["has_more"])
			path = ordersPath + "/records?limit=3&cursor=" + url.QueryEscape(next)
		}
		assert.Len(t, seen, 7)
		for id, n := range seen {
			assert.Equal(t, 1, n, id)
		}

		code, res, _ = call("GET", ordersPath+"/records?filter[Status]=Done&filter[Price][gte]=40&sort=-Price&select=Title,Price", nil)
		require.Equal(t, http.StatusOK, code, res.Error)
		got := records(res.Data)
		require.Len(t, got, 2)
		assert.Equal(t, "Bulk 6", got[0].Fields["Title"])
		assert.Equal(t, "Bulk 4", got[1].Fields["Title"])
		assert.Len(t, got[0].Fields, 2, "select returns only the selected fields")

		// Duration is stored as text; comparisons are numeric
		code, res, _ = call("GET", ordersPath+"/records?filter[Took][gt]=60&filter[Title][contains]=bulk", nil)
		require.Equal(t, http.StatusOK, code, res.Error)
		assert.Len(t, records(res.Data), 2) // 75 and 90 minutes

		code, res, _ = call("GET", ordersPath+"/records?filter[Customer][contains]="+cust[0].ID, nil)
		require.Equal(t, http.StatusOK, code, res.Error)
		got = records(res.Data)
		require.Len(t, got, 1)
		assert.Equal(t, first.ID, got[0].ID)

		code, res, _ = call("GET", ordersPath+"/records?keys=id&limit=1", nil)
		require.Equal(t, http.StatusOK, code)
		for key := range records(res.Data)[0].Fields {
			assert.True(t, strings.HasPrefix(key, "fld_"), key)
		}

		// A cursor can't be reused with another filter
		code, res, _ = call("GET", ordersPath+"/records?limit=1", nil)
		require.Equal(t, http.StatusOK, code)
		next, _ := res.Meta["next_cursor"].(string)
		code, res, _ = call("GET", ordersPath+"/records?limit=1&filter[Status]=Open&cursor="+url.QueryEscape(next), nil)
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Equal(t, "bad_request", res.Error.Code)
	})

	t.Run("search across text fields", func(t *testing.T) {
		titles := func(raw json.RawMessage) []string {
			var out []string
			for _, rec := range records(raw) {
				out = append(out, fmt.Sprint(rec.Fields["Title"]))
			}
			return out
		}
		// Case-insensitive, in Title
		code, res, _ := call("GET", ordersPath+"/records?search=BULK&sort=Price", nil)
		require.Equal(t, http.StatusOK, code, res.Error)
		assert.Equal(t, []string{"Bulk 1", "Bulk 2", "Bulk 3", "Bulk 4", "Bulk 5", "Bulk 6"}, titles(res.Data))

		// In a select field too (Status), and combined with filters using AND
		code, res, _ = call("GET", ordersPath+"/records?search=open", nil)
		require.Equal(t, http.StatusOK, code, res.Error)
		assert.Len(t, records(res.Data), 4) // First, Bulk 1, 3, 5
		code, res, _ = call("GET", ordersPath+"/records?search=open&filter[Price][gte]=20", nil)
		require.Equal(t, http.StatusOK, code, res.Error)
		assert.Equal(t, []string{"Bulk 3", "Bulk 5"}, titles(res.Data))

		// In the body, paged with its cursor
		code, res, _ = call("POST", ordersPath+"/records/query", map[string]interface{}{"search": "bulk", "limit": 4, "sort": "Price"})
		require.Equal(t, http.StatusOK, code, res.Error)
		assert.Len(t, records(res.Data), 4)
		next, _ := res.Meta["next_cursor"].(string)
		require.NotEmpty(t, next)
		code, res, _ = call("GET", ordersPath+"/records?search=bulk&limit=4&sort=Price&cursor="+url.QueryEscape(next), nil)
		require.Equal(t, http.StatusOK, code, res.Error)
		assert.Equal(t, []string{"Bulk 5", "Bulk 6"}, titles(res.Data))
		code, _, _ = call("GET", ordersPath+"/records?search=first&limit=4&sort=Price&cursor="+url.QueryEscape(next), nil)
		assert.Equal(t, http.StatusBadRequest, code, "a cursor belongs to its search")

		code, res, _ = call("GET", ordersPath+"/records?search=nothing-matches-this", nil)
		require.Equal(t, http.StatusOK, code, res.Error)
		assert.Empty(t, records(res.Data))
	})

	t.Run("expand linked records", func(t *testing.T) {
		// Asha's company
		code, res, _ := call("POST", base+"/bases/Main/tables/Companies/records", map[string]interface{}{
			"fields": map[string]interface{}{"Title": "Acme Group"}})
		require.Equal(t, http.StatusCreated, code, res.Error)
		company := record(res.Data)
		code, res, _ = call("PATCH", base+"/bases/Main/tables/Customers/records/"+cust[0].ID, map[string]interface{}{
			"fields": map[string]interface{}{"Company": []string{company.ID}}})
		require.Equal(t, http.StatusOK, code, res.Error)

		linked := func(rec apiRecord, name string) []map[string]interface{} {
			t.Helper()
			list, ok := rec.Fields[name].([]interface{})
			require.True(t, ok, "%s: %v", name, rec.Fields[name])
			out := make([]map[string]interface{}, len(list))
			for i, item := range list {
				out[i] = item.(map[string]interface{})
			}
			return out
		}
		inner := func(obj map[string]interface{}) apiRecord {
			t.Helper()
			fields, ok := obj["fields"].(map[string]interface{})
			require.True(t, ok, "expanded object has fields: %v", obj)
			return apiRecord{Fields: fields}
		}
		byCustomer := ordersPath + "/records?filter[Customer][contains]=" + cust[0].ID

		// Without expand: only the id and the lookup
		code, res, _ = call("GET", byCustomer, nil)
		require.Equal(t, http.StatusOK, code, res.Error)
		obj := linked(records(res.Data)[0], "Customer")[0]
		assert.NotContains(t, obj, "fields")

		// One level: the customer's fields; its own link holds only the company's id
		code, res, _ = call("GET", byCustomer+"&expand=Customer", nil)
		require.Equal(t, http.StatusOK, code, res.Error)
		obj = linked(records(res.Data)[0], "Customer")[0]
		assert.Equal(t, cust[0].ID, obj["id"])
		assert.Equal(t, "Asha", obj[customerTitle.Title], "lookups are still there")
		customer := inner(obj)
		assert.Equal(t, "Asha", customer.Fields[customerTitle.Title])
		assert.Equal(t, []interface{}{map[string]interface{}{"id": company.ID}}, customer.Fields["Company"])

		// Two levels; Customer.Company alone expands Customer too
		for _, expand := range []string{"Customer,Customer.Company", "Customer.Company"} {
			code, res, _ = call("GET", byCustomer+"&expand="+url.QueryEscape(expand), nil)
			require.Equal(t, http.StatusOK, code, res.Error)
			customer = inner(linked(records(res.Data)[0], "Customer")[0])
			comp := linked(customer, "Company")[0]
			assert.Equal(t, company.ID, comp["id"])
			assert.Equal(t, "Acme Group", inner(comp).Fields["Title"], expand)
		}

		// An expanded field is returned even when select leaves it out
		code, res, _ = call("GET", byCustomer+"&select=Title&expand=Customer", nil)
		require.Equal(t, http.StatusOK, code, res.Error)
		got := records(res.Data)[0]
		assert.Len(t, got.Fields, 2)
		assert.Equal(t, "Asha", inner(linked(got, "Customer")[0]).Fields[customerTitle.Title])

		// In the query body, and on one record
		code, res, _ = call("POST", ordersPath+"/records/query", map[string]interface{}{
			"where":  map[string]interface{}{"field": "Customer", "op": "contains", "value": cust[0].ID},
			"expand": []string{"Customer.Company"}})
		require.Equal(t, http.StatusOK, code, res.Error)
		customer = inner(linked(records(res.Data)[0], "Customer")[0])
		assert.Equal(t, "Acme Group", inner(linked(customer, "Company")[0]).Fields["Title"])
		code, res, _ = call("GET", ordersPath+"/records/"+first.ID+"?expand=Customer", nil)
		require.Equal(t, http.StatusOK, code, res.Error)
		assert.Equal(t, "Asha", inner(linked(record(res.Data), "Customer")[0]).Fields[customerTitle.Title])

		// Records without links still list fine
		code, res, _ = call("GET", ordersPath+"/records?expand=Customer.Company", nil)
		require.Equal(t, http.StatusOK, code, res.Error)
		assert.Len(t, records(res.Data), 7)

		// Problems are 400s that say what to change
		for expand, msg := range map[string]string{
			"Price":                       `"Price" is a decimal field; only link fields can be expanded`,
			"Custmer":                     `Did you mean "Customer"?`,
			"Customer.Compny":             `Did you mean "Company"?`,
			"Customer.Company.Title.More": "goes deeper than 2 levels",
		} {
			code, res, _ = call("GET", ordersPath+"/records?expand="+url.QueryEscape(expand), nil)
			require.Equal(t, http.StatusBadRequest, code, expand)
			assert.Contains(t, res.Error.Message, msg, expand)
		}
	})

	t.Run("bulk create rolls back when the database rejects one record", func(t *testing.T) {
		// Passes the API's validation; only Postgres rejects it, on the third record
		exec(fmt.Sprintf(`ALTER TABLE "%s".%q ADD CONSTRAINT e2e_qty_not_13 CHECK (%q IS DISTINCT FROM 13)`,
			s, orders.Model.Alias, qty.ColumnName))
		defer exec(fmt.Sprintf(`ALTER TABLE "%s".%q DROP CONSTRAINT e2e_qty_not_13`, s, orders.Model.Alias))

		in := make([]interface{}, 0, 5)
		for i := 1; i <= 5; i++ {
			q := i
			if i == 3 {
				q = 13
			}
			in = append(in, map[string]interface{}{"fields": map[string]interface{}{"Title": fmt.Sprintf("Rollback %d", i), "Qty": q}})
		}
		code, res, _ := call("POST", ordersPath+"/records", map[string]interface{}{"records": in})
		require.Equal(t, http.StatusUnprocessableEntity, code, res.Error)
		assert.Equal(t, "validation_failed", res.Error.Code)
		require.NotEmpty(t, res.Error.Details)
		require.NotNil(t, res.Error.Details[0].Index)
		assert.Equal(t, 2, *res.Error.Details[0].Index)

		code, res, _ = call("GET", ordersPath+"/records?filter[Title][contains]=Rollback", nil)
		require.Equal(t, http.StatusOK, code, res.Error)
		assert.Empty(t, records(res.Data), "nothing written")
	})

	t.Run("update one and many, change links", func(t *testing.T) {
		code, res, _ := call("PATCH", ordersPath+"/records/"+first.ID, map[string]interface{}{"fields": map[string]interface{}{
			"Price": 99, "Customer": []string{cust[1].ID}}})
		require.Equal(t, http.StatusOK, code, res.Error)
		rec := record(res.Data)
		assert.Equal(t, float64(99), rec.Fields["Price"])
		assert.Equal(t, "Open", rec.Fields["Status"], "fields not sent are unchanged")
		assert.Equal(t, []interface{}{map[string]interface{}{"id": cust[1].ID, customerTitle.Title: "Ravi"}}, rec.Fields["Customer"])

		// The link value as read can be sent back as it is
		code, res, _ = call("PATCH", ordersPath+"/records/"+first.ID, map[string]interface{}{"fields": map[string]interface{}{
			"Customer": rec.Fields["Customer"]}})
		require.Equal(t, http.StatusOK, code, res.Error)
		assert.Equal(t, rec.Fields["Customer"], record(res.Data).Fields["Customer"])

		code, res, _ = call("GET", ordersPath+"/records?limit=2&sort=created_at", nil)
		require.Equal(t, http.StatusOK, code)
		two := records(res.Data)
		code, res, _ = call("PATCH", ordersPath+"/records", map[string]interface{}{"records": []interface{}{
			map[string]interface{}{"id": two[0].ID, "fields": map[string]interface{}{"Paid": false}},
			map[string]interface{}{"id": two[1].ID, "fields": map[string]interface{}{"Paid": true}},
		}})
		require.Equal(t, http.StatusOK, code, res.Error)
		assert.Len(t, records(res.Data), 2)

		// A missing record fails the whole bulk update
		code, res, _ = call("PATCH", ordersPath+"/records", map[string]interface{}{"records": []interface{}{
			map[string]interface{}{"id": two[0].ID, "fields": map[string]interface{}{"Title": "should not stick"}},
			map[string]interface{}{"id": "rec_999999", "fields": map[string]interface{}{"Paid": true}},
		}})
		require.Equal(t, http.StatusNotFound, code)
		assert.Equal(t, "rec_999999", res.Error.Details[0].ID)
		code, res, _ = call("GET", ordersPath+"/records/"+two[0].ID, nil)
		require.Equal(t, http.StatusOK, code)
		assert.NotEqual(t, "should not stick", record(res.Data).Fields["Title"], "rolled back")
		// With the default key (id), the plain number works as well as rec_
		code, res, _ = call("GET", ordersPath+"/records/"+strings.TrimPrefix(two[0].ID, "rec_"), nil)
		require.Equal(t, http.StatusOK, code, res.Error)
		assert.Equal(t, two[0].ID, record(res.Data).ID)
		code, _, _ = call("GET", ordersPath+"/records/abc", nil)
		assert.Equal(t, http.StatusNotFound, code)

		// The same ID twice
		code, _, _ = call("PATCH", ordersPath+"/records", map[string]interface{}{"records": []interface{}{
			map[string]interface{}{"id": two[0].ID, "fields": map[string]interface{}{}},
			map[string]interface{}{"id": two[0].ID, "fields": map[string]interface{}{}},
		}})
		assert.Equal(t, http.StatusBadRequest, code)
	})

	t.Run("idempotency", func(t *testing.T) {
		body := map[string]interface{}{"fields": map[string]interface{}{"Title": "Once"}}
		code1, res1, _ := call("POST", ordersPath+"/records", body, "Idempotency-Key", "key-1")
		require.Equal(t, http.StatusCreated, code1, res1.Error)
		code2, res2, h2 := call("POST", ordersPath+"/records", body, "Idempotency-Key", "key-1")
		require.Equal(t, http.StatusCreated, code2)
		assert.Equal(t, "true", h2.Get("Idempotent-Replayed"))
		assert.Equal(t, record(res1.Data).ID, record(res2.Data).ID, "one record, replayed")
		assert.NotEqual(t, res1.Meta["request_id"], res2.Meta["request_id"])

		code, res, _ := call("POST", ordersPath+"/records", map[string]interface{}{"fields": map[string]interface{}{"Title": "Other"}},
			"Idempotency-Key", "key-1")
		assert.Equal(t, http.StatusConflict, code)
		assert.Equal(t, "conflict", res.Error.Code)
	})

	t.Run("delete one and many", func(t *testing.T) {
		code, res, _ := call("GET", ordersPath+"/records?filter[Title][contains]=bulk&limit=3", nil)
		require.Equal(t, http.StatusOK, code)
		three := records(res.Data)
		require.Len(t, three, 3)

		code, res, _ = call("DELETE", ordersPath+"/records/"+three[0].ID, nil)
		require.Equal(t, http.StatusOK, code, res.Error)
		assert.JSONEq(t, fmt.Sprintf(`{"id":%q,"deleted":true}`, three[0].ID), string(res.Data))
		code, _, _ = call("GET", ordersPath+"/records/"+three[0].ID, nil)
		assert.Equal(t, http.StatusNotFound, code)

		// A missing ID: 404 and nothing deleted
		code, res, _ = call("POST", ordersPath+"/records/delete", map[string]interface{}{"ids": []string{three[1].ID, "rec_999999"}})
		require.Equal(t, http.StatusNotFound, code)
		assert.Equal(t, "rec_999999", res.Error.Details[0].ID)
		code, _, _ = call("GET", ordersPath+"/records/"+three[1].ID, nil)
		assert.Equal(t, http.StatusOK, code, "not deleted")

		code, res, _ = call("POST", ordersPath+"/records/delete", map[string]interface{}{"ids": []string{three[1].ID, three[2].ID}})
		require.Equal(t, http.StatusOK, code, res.Error)
		assert.Len(t, records(res.Data), 2)

		// Deleting a customer removes it from the orders that link to it
		code, _, _ = call("DELETE", base+"/bases/Main/tables/Customers/records/"+cust[1].ID, nil)
		require.Equal(t, http.StatusOK, code)
		code, res, _ = call("GET", ordersPath+"/records/"+first.ID, nil)
		require.Equal(t, http.StatusOK, code)
		assert.Equal(t, []interface{}{}, record(res.Data).Fields["Customer"])
	})

	t.Run("names, ambiguity and unknown paths", func(t *testing.T) {
		code, res, _ := call("GET", base+"/bases/Nope/tables", nil)
		assert.Equal(t, http.StatusNotFound, code)
		assert.Equal(t, "not_found", res.Error.Code)

		code, res, _ = call("GET", "/api/v1/ws_0000000000/bases", nil)
		assert.Equal(t, http.StatusNotFound, code)
		assert.Equal(t, "not_found", res.Error.Code)

		code, res, _ = call("GET", base+"/nothing/here", nil)
		assert.Equal(t, http.StatusNotFound, code)
		assert.Equal(t, "not_found", res.Error.Code)

		// A second table named Orders bumps schema_version; once the 10 s workspace cache expires, the name is ambiguous
		var before, after int64
		require.NoError(t, db.DB.QueryRow(fmt.Sprintf(`SELECT schema_version FROM "%s".workspace_api WHERE workspace_id = $1`, s), wsID).Scan(&before))
		dup := createTable("Orders")
		require.NoError(t, db.DB.QueryRow(fmt.Sprintf(`SELECT schema_version FROM "%s".workspace_api WHERE workspace_id = $1`, s), wsID).Scan(&after))
		assert.Greater(t, after, before, "creating a table bumps schema_version")

		// Turned off, the workspace is 404 once the 10 s cache expires; turned back on, it answers again
		setEnabled := func(on bool) {
			exec(fmt.Sprintf(`UPDATE "%s".workspace_api SET enabled = $1 WHERE workspace_id = $2`, s), on, wsID)
		}
		setEnabled(false)
		time.Sleep(11 * time.Second)
		code, res, _ = call("GET", base+"/bases", nil)
		assert.Equal(t, http.StatusNotFound, code)
		assert.Equal(t, "not_found", res.Error.Code)
		setEnabled(true)

		time.Sleep(11 * time.Second)
		code, res, _ = call("GET", ordersPath+"/records", nil)
		require.Equal(t, http.StatusConflict, code)
		assert.Equal(t, "ambiguous_name", res.Error.Code)
		assert.Len(t, res.Error.Details, 2)

		code, _, _ = call("GET", base+"/bases/Main/tables/"+platform.EncodeUUID(platform.PrefixTable, dup.Model.ID.String())+"/records", nil)
		assert.Equal(t, http.StatusOK, code, "IDs always work")
	})

	// A table whose primary key was changed in the app, with a unique constraint and a NOT NULL field, and a
	// table that links to it: every endpoint keeps working, and constraint problems are 422s, not 500s
	t.Run("custom primary key and constraints", func(t *testing.T) {
		productsPath := base + "/bases/Main/tables/Products"
		code, res, _ := call("POST", productsPath+"/records", map[string]interface{}{"records": []interface{}{
			map[string]interface{}{"fields": map[string]interface{}{"SKU": "A1", "Region": "IN", "Note": "first", "Size": "M"}},
			map[string]interface{}{"fields": map[string]interface{}{"SKU": "A1", "Region": "US", "Note": "second", "Size": "L"}},
		}})
		require.Equal(t, http.StatusCreated, code, res.Error)
		prods := records(res.Data)
		require.Len(t, prods, 2)

		rejected := map[string]map[string]interface{}{
			"key field missing":    {"SKU": "B1", "Note": "x1", "Size": "M"},
			"duplicate key":        {"SKU": "A1", "Region": "IN", "Note": "x2", "Size": "M"},
			"unique constraint":    {"SKU": "C1", "Region": "IN", "Note": "first", "Size": "M"},
			"NOT NULL field empty": {"SKU": "D1", "Region": "IN", "Note": "x3"},
		}
		for name, fields := range rejected {
			code, res, _ := call("POST", productsPath+"/records", map[string]interface{}{"fields": fields})
			assert.Equal(t, http.StatusUnprocessableEntity, code, "%s: %+v", name, res.Error)
			if assert.NotNil(t, res.Error, name) {
				assert.Equal(t, "validation_failed", res.Error.Code, name)
			}
		}

		// The key is SKU + Region, so a record is found by naming its key fields, not by its record ID
		assert.Equal(t, map[string]interface{}{"SKU": "A1", "Region": "IN"}, prods[0].Key, "records carry their key")
		code, res, _ = call("GET", productsPath+"/records/"+prods[0].ID, nil)
		require.Equal(t, http.StatusBadRequest, code, res.Error)
		assert.Equal(t, fmt.Sprintf(`%q is a record ID, but the primary key of this table is SKU + Region. Use records/by-key?key[SKU]=…&key[Region]=….`, prods[0].ID), res.Error.Message)
		code, res, _ = call("GET", productsPath+"/records/A1", nil)
		require.Equal(t, http.StatusBadRequest, code, res.Error)
		assert.Equal(t, "The primary key of this table is SKU + Region, so name each key field: use records/by-key?key[SKU]=…&key[Region]=….", res.Error.Message)
		code, res, _ = call("GET", productsPath+"/records/by-key?key[SKU]=A1&key[Region]=IN", nil)
		require.Equal(t, http.StatusOK, code, res.Error)
		assert.Equal(t, prods[0].ID, record(res.Data).ID)
		code, res, _ = call("POST", productsPath+"/records/query", map[string]interface{}{
			"where": map[string]interface{}{"match": "all", "conditions": []interface{}{
				map[string]interface{}{"field": "SKU", "value": "A1"},
				map[string]interface{}{"match": "none", "conditions": []interface{}{map[string]interface{}{"field": "Region", "value": "IN"}}},
			}}})
		require.Equal(t, http.StatusOK, code, res.Error)
		got := records(res.Data)
		require.Len(t, got, 1)
		assert.Equal(t, prods[1].ID, got[0].ID)
		// The HTTP QUERY method gives the same answer as POST
		code, res, _ = call("QUERY", productsPath+"/records/query", map[string]interface{}{
			"where": map[string]interface{}{"match": "all", "conditions": []interface{}{
				map[string]interface{}{"field": "SKU", "value": "A1"},
				map[string]interface{}{"match": "none", "conditions": []interface{}{map[string]interface{}{"field": "Region", "value": "IN"}}},
			}}})
		require.Equal(t, http.StatusOK, code, res.Error)
		got = records(res.Data)
		require.Len(t, got, 1)
		assert.Equal(t, prods[1].ID, got[0].ID)
		code, res, _ = call("PATCH", productsPath+"/records/by-key?key[SKU]=A1&key[Region]=IN", map[string]interface{}{"fields": map[string]interface{}{"Note": "renamed"}})
		require.Equal(t, http.StatusOK, code, res.Error)
		code, res, _ = call("PATCH", productsPath+"/records/by-key?key[SKU]=A1&key[Region]=US", map[string]interface{}{"fields": map[string]interface{}{"Note": "renamed"}})
		assert.Equal(t, http.StatusUnprocessableEntity, code, "unique constraint on update: %+v", res.Error)

		// Key fields in any order, by name or field ID; "_", "/" and spaces need only URL encoding
		code, res, _ = call("PATCH", productsPath+"/records/by-key?key[Region]=US&key[SKU]=A1", map[string]interface{}{"fields": map[string]interface{}{"Size": "XL"}})
		require.Equal(t, http.StatusOK, code, res.Error)
		assert.Equal(t, "XL", record(res.Data).Fields["Size"])
		assert.Equal(t, map[string]interface{}{"SKU": "A1", "Region": "US"}, record(res.Data).Key)
		code, res, _ = call("POST", productsPath+"/records", map[string]interface{}{"fields": map[string]interface{}{
			"SKU": "B_1/x y", "Region": "IN", "Note": "odd characters", "Size": "M"}})
		require.Equal(t, http.StatusCreated, code, res.Error)
		odd := record(res.Data)
		code, res, _ = call("GET", productsPath+"/records/by-key?"+url.Values{"key[" + platform.EncodeUUID(platform.PrefixField, sku.ID.String()) + "]": {"B_1/x y"}, "key[Region]": {"IN"}}.Encode(), nil)
		require.Equal(t, http.StatusOK, code, res.Error)
		assert.Equal(t, odd.ID, record(res.Data).ID)
		badKeys := map[string]int{
			"key[SKU]=A1&key[Region]=FR":              http.StatusNotFound,
			"key[SKU]=A1":                             http.StatusBadRequest, // Region missing
			"key[SKU]=A1&key[Region]=US&key[Size]=XL": http.StatusBadRequest, // Size isn't part of the key
			"key[Sku2]=A1&key[Region]=US":             http.StatusBadRequest, // no such field
			"":                                        http.StatusBadRequest, // no key fields
		}
		for query, want := range badKeys {
			code, res, _ = call("GET", productsPath+"/records/by-key?"+query, nil)
			assert.Equal(t, want, code, "%s: %+v", query, res.Error)
		}
		code, res, _ = call("GET", productsPath+"/records/by-key?key[SKU]=A1", nil)
		assert.Equal(t, "The primary key is SKU + Region; Region is missing.", res.Error.Message)
		code, res, _ = call("GET", productsPath+"/records/by-key?key[SKU]=A1&key[Region]=FR", nil)
		assert.Equal(t, `No record with SKU "A1" and Region "FR".`, res.Error.Message)

		// Bulk update takes "key" objects, bulk delete "keys"; record IDs are refused
		code, res, _ = call("PATCH", productsPath+"/records", map[string]interface{}{"records": []interface{}{
			map[string]interface{}{"key": map[string]interface{}{"SKU": "A1", "Region": "IN"}, "fields": map[string]interface{}{"Size": "S"}},
			map[string]interface{}{"key": map[string]interface{}{"SKU": "B_1/x y", "Region": "IN"}, "fields": map[string]interface{}{"Size": "S"}},
		}})
		require.Equal(t, http.StatusOK, code, res.Error)
		for _, r := range records(res.Data) {
			assert.Equal(t, "S", r.Fields["Size"])
		}
		code, res, _ = call("PATCH", productsPath+"/records", map[string]interface{}{"records": []interface{}{
			map[string]interface{}{"key": map[string]interface{}{"SKU": "A1", "Region": "IN"}, "fields": map[string]interface{}{"Size": "M"}},
			map[string]interface{}{"id": odd.ID, "fields": map[string]interface{}{"Size": "M"}},
		}})
		require.Equal(t, http.StatusBadRequest, code, res.Error)
		assert.Equal(t, fmt.Sprintf(`records[1]: %q is a record ID, but the primary key of this table is SKU + Region. Use "key": {"SKU": …, "Region": …} instead of "id".`, odd.ID), res.Error.Message)
		code, res, _ = call("PATCH", productsPath+"/records", map[string]interface{}{"records": []interface{}{
			map[string]interface{}{"key": map[string]interface{}{"SKU": "A1", "Region": "IN"}, "fields": map[string]interface{}{"Size": "M"}},
			map[string]interface{}{"key": map[string]interface{}{"SKU": "Z9", "Region": "XX"}, "fields": map[string]interface{}{"Size": "M"}},
		}})
		require.Equal(t, http.StatusNotFound, code, res.Error)
		require.Len(t, res.Error.Details, 1)
		assert.Equal(t, `{"Region":"XX","SKU":"Z9"}`, res.Error.Details[0].ID)
		code, res, _ = call("POST", productsPath+"/records/delete", map[string]interface{}{"ids": []interface{}{odd.ID}})
		require.Equal(t, http.StatusBadRequest, code, res.Error)
		assert.Contains(t, res.Error.Message, `Use "keys": [{"SKU": …, "Region": …}] instead of "ids".`)
		oddKey := map[string]interface{}{"SKU": "B_1/x y", "Region": "IN"}
		code, res, _ = call("POST", productsPath+"/records/delete", map[string]interface{}{"keys": []interface{}{oddKey, oddKey}})
		assert.Equal(t, http.StatusBadRequest, code, "the same record twice: %+v", res.Error)
		code, res, _ = call("POST", productsPath+"/records/delete", map[string]interface{}{"keys": []interface{}{oddKey}})
		require.Equal(t, http.StatusOK, code, res.Error)
		assert.JSONEq(t, fmt.Sprintf(`[{"id": %q, "key": {"SKU": "B_1/x y", "Region": "IN"}, "deleted": true}]`, odd.ID), string(res.Data))

		// Links to a table with a custom key: write, read back as rec_ IDs, and filter by the linked record
		shipmentsPath := base + "/bases/Main/tables/Shipments"
		code, res, _ = call("POST", shipmentsPath+"/records", map[string]interface{}{"fields": map[string]interface{}{"Product": []interface{}{prods[1].ID}}})
		require.Equal(t, http.StatusCreated, code, res.Error)
		shipment := record(res.Data)
		linked, _ := shipment.Fields["Product"].([]interface{})
		require.Len(t, linked, 1)
		assert.Equal(t, prods[1].ID, linked[0].(map[string]interface{})["id"])
		code, res, _ = call("GET", shipmentsPath+"/records?filter[Product][contains]="+prods[1].ID, nil)
		require.Equal(t, http.StatusOK, code, res.Error)
		got = records(res.Data)
		require.Len(t, got, 1)
		assert.Equal(t, shipment.ID, got[0].ID)
		code, res, _ = call("GET", shipmentsPath+"/records?filter[Product][contains]="+prods[0].ID, nil)
		require.Equal(t, http.StatusOK, code, res.Error)
		assert.Empty(t, records(res.Data))
		// expand follows links stored as key objects too
		code, res, _ = call("GET", shipmentsPath+"/records/"+shipment.ID+"?expand=Product", nil)
		require.Equal(t, http.StatusOK, code, res.Error)
		linked, _ = record(res.Data).Fields["Product"].([]interface{})
		require.Len(t, linked, 1)
		expandedProduct, _ := linked[0].(map[string]interface{})["fields"].(map[string]interface{})
		assert.Equal(t, prods[1].Fields["SKU"], expandedProduct["SKU"])
		assert.Equal(t, prods[1].Fields["Region"], expandedProduct["Region"])

		// Changing the key values of a linked record keeps the link (the app's key sync triggers rewrite it)
		code, res, _ = call("PATCH", productsPath+"/records/by-key?key[SKU]=A1&key[Region]=US", map[string]interface{}{"fields": map[string]interface{}{"SKU": "A2"}})
		require.Equal(t, http.StatusOK, code, res.Error)
		code, res, _ = call("GET", shipmentsPath+"/records/"+shipment.ID, nil)
		require.Equal(t, http.StatusOK, code, res.Error)
		linked, _ = record(res.Data).Fields["Product"].([]interface{})
		require.Len(t, linked, 1)
		assert.Equal(t, prods[1].ID, linked[0].(map[string]interface{})["id"])
		code, res, _ = call("GET", shipmentsPath+"/records?filter[Product][contains]="+prods[1].ID, nil)
		require.Equal(t, http.StatusOK, code, res.Error)
		assert.Len(t, records(res.Data), 1)

		// With a link in place, the app refuses to change the key again, so stored links can't break
		_, err := pkSvc.Update(ctx, s, products.Model.ID.String(), dto.UpdatePrimaryKeyRequest{ColumnIDs: []string{sku.ID.String()}})
		assert.Error(t, err)

		code, res, _ = call("DELETE", productsPath+"/records/by-key?key[SKU]=A1&key[Region]=IN", nil)
		assert.Equal(t, http.StatusOK, code, res.Error)
	})

	t.Run("automations: their own messages, a failed automation", func(t *testing.T) {
		stockPath := base + "/bases/Main/tables/Stock"
		issue := func(res apiEnvelope) string {
			t.Helper()
			require.NotNil(t, res.Error)
			if len(res.Error.Details) > 0 && res.Error.Details[0].Issue != "" {
				return res.Error.Details[0].Issue
			}
			return res.Error.Message
		}

		// RAISE EXCEPTION: 422 with the automation's own message, naming the record
		code, res, _ := call("POST", stockPath+"/records", map[string]interface{}{"fields": map[string]interface{}{"Qty": -1, "Item": "pen"}})
		require.Equal(t, http.StatusUnprocessableEntity, code, res.Error)
		assert.Equal(t, "validation_failed", res.Error.Code)
		assert.Equal(t, "Stock can't go below zero", issue(res))

		code, res, _ = call("POST", stockPath+"/records", map[string]interface{}{"records": []interface{}{
			map[string]interface{}{"fields": map[string]interface{}{"Qty": 5, "Item": "pen"}},
			map[string]interface{}{"fields": map[string]interface{}{"Qty": -2, "Item": "ink"}},
		}})
		require.Equal(t, http.StatusUnprocessableEntity, code, res.Error)
		require.NotNil(t, res.Error.Details[0].Index)
		assert.Equal(t, 1, *res.Error.Details[0].Index, "the second record was rejected")
		assert.Equal(t, "Stock can't go below zero", issue(res))

		// A mistake in the automation: not our generic error, and no Postgres text
		code, res, _ = call("POST", stockPath+"/records", map[string]interface{}{"fields": map[string]interface{}{"Qty": 1, "Item": "broken"}})
		require.Equal(t, http.StatusInternalServerError, code, res.Error)
		assert.Equal(t, "internal_error", res.Error.Code)
		assert.Equal(t, "An automation on this table failed, so nothing was saved. Check the table's automations.", res.Error.Message)
		assert.NotContains(t, res.Error.Message, "division")

		// Updates and deletes get the automation's message too
		code, res, _ = call("POST", stockPath+"/records", map[string]interface{}{"fields": map[string]interface{}{"Qty": 150, "Item": "box"}})
		require.Equal(t, http.StatusCreated, code, res.Error)
		boxID := record(res.Data).ID
		code, res, _ = call("PATCH", stockPath+"/records/"+boxID, map[string]interface{}{"fields": map[string]interface{}{"Qty": -5}})
		require.Equal(t, http.StatusUnprocessableEntity, code, res.Error)
		assert.Equal(t, "Stock can't go below zero", issue(res))
		code, res, _ = call("DELETE", stockPath+"/records/"+boxID, nil)
		require.Equal(t, http.StatusUnprocessableEntity, code, res.Error)
		assert.Equal(t, "Items in stock can't be deleted", issue(res))
		code, res, _ = call("POST", stockPath+"/records/delete", map[string]interface{}{"ids": []string{boxID}})
		require.Equal(t, http.StatusUnprocessableEntity, code, res.Error)
		assert.Equal(t, "Items in stock can't be deleted", issue(res))
		code, _, _ = call("GET", stockPath+"/records/"+boxID, nil)
		assert.Equal(t, http.StatusOK, code, "the record is still there")

		// The app's own row endpoint gets the same message
		upserter, ok := tableSvc.(interface {
			UpsertRowValues(ctx context.Context, schemaName, modelID string, rowID int, values map[string]interface{}, user string) (dto.RecordResponse, error)
		})
		require.True(t, ok)
		_, err := upserter.UpsertRowValues(ctx, s, stock.Model.ID.String(), 0, map[string]interface{}{stockQty.ID.String(): -3}, "")
		require.ErrorIs(t, err, app_errors.AutomationRejected)
		var detailed *app_errors.DetailedError
		require.ErrorAs(t, err, &detailed)
		assert.Equal(t, "Stock can't go below zero", detailed.Message)
	})

	t.Run("field settings: format checks on and off, decimals, time, options", func(t *testing.T) {
		contactsPath := base + "/bases/Main/tables/Contacts"
		issues := func(res apiEnvelope) map[string]string {
			t.Helper()
			require.NotNil(t, res.Error)
			out := map[string]string{}
			for _, d := range res.Error.Details {
				out[d.Field] = d.Issue
			}
			return out
		}

		// Checks that are on reject bad values with the app's own rules
		code, res, _ := call("POST", contactsPath+"/records", map[string]interface{}{"fields": map[string]interface{}{
			"Work email": "nope", "Phone": "0123", "Site": "https://Example.com", "Level": "Bronze", "Tags": []string{"vip", "old"}}})
		require.Equal(t, http.StatusUnprocessableEntity, code, res.Error)
		got := issues(res)
		assert.Equal(t, "must be a valid email address", got["Work email"])
		assert.Equal(t, "must be a valid phone number", got["Phone"])
		assert.Equal(t, "must be a valid URL", got["Site"])
		assert.Equal(t, `"Bronze" is not one of the field's options`, got["Level"])
		assert.Equal(t, `"old" is not one of the field's options`, got["Tags"])

		// Checks that are off accept any text, as in the app; decimals are rounded and 12-hour times converted
		code, res, _ = call("POST", contactsPath+"/records", map[string]interface{}{"fields": map[string]interface{}{
			"Work email": "asha@example.com", "Any email": "not an email", "Phone": "+91 98765 43210",
			"Site": "example.com/docs", "Any link": "see the wiki", "Score": 4.567, "Done": 33.36, "Call at": "2:15 PM",
			"Level": "Gold", "Tags": []string{"vip"}}})
		require.Equal(t, http.StatusCreated, code, res.Error)
		f := record(res.Data).Fields
		assert.Equal(t, "not an email", f["Any email"])
		assert.Equal(t, "see the wiki", f["Any link"])
		assert.Equal(t, 4.57, f["Score"], "kept to the field's 2 decimals")
		assert.Equal(t, 33.4, f["Done"], "percent keeps 1 decimal")
		assert.Equal(t, "14:15", f["Call at"])

		// The app's own row endpoint applies the same settings
		upserter, ok := tableSvc.(interface {
			UpsertRowValues(ctx context.Context, schemaName, modelID string, rowID int, values map[string]interface{}, user string) (dto.RecordResponse, error)
		})
		require.True(t, ok)
		_, err := upserter.UpsertRowValues(ctx, s, contacts.Model.ID.String(), 0, map[string]interface{}{contactLevel.ID.String(): "Bronze"}, "")
		require.ErrorIs(t, err, app_errors.InvalidFieldValue)
		var detailed *app_errors.DetailedError
		require.ErrorAs(t, err, &detailed)
		assert.Equal(t, `Level: "Bronze" is not one of the field's options`, detailed.Message)
		_, err = upserter.UpsertRowValues(ctx, s, contacts.Model.ID.String(), 0, map[string]interface{}{contactTags.ID.String(): []interface{}{"vip", "old"}}, "")
		require.ErrorIs(t, err, app_errors.InvalidFieldValue)
		_, err = upserter.UpsertRowValues(ctx, s, contacts.Model.ID.String(), 0, map[string]interface{}{
			contactLevel.ID.String(): "", contactTags.ID.String(): []interface{}{"new"}}, "")
		require.NoError(t, err, "an empty select and known options are fine")
	})

	t.Run("UUID primary key: auto needs no value, manual does", func(t *testing.T) {
		uuidField := func(tablePath, name string) map[string]interface{} {
			code, res, _ := call("GET", tablePath, nil)
			require.Equal(t, http.StatusOK, code, res.Error)
			var table struct {
				Fields []map[string]interface{} `json:"fields"`
			}
			require.NoError(t, json.Unmarshal(res.Data, &table))
			for _, f := range table.Fields {
				if f["name"] == name {
					return f
				}
			}
			t.Fatalf("no field %q", name)
			return nil
		}

		// Auto: the field is read-only; a create without it gets a generated v7 UUID
		devicesPath := base + "/bases/Main/tables/Devices"
		assert.Equal(t, true, uuidField(devicesPath, "Device ID")["read_only"])
		code, res, _ := call("POST", devicesPath+"/records", map[string]interface{}{"records": []interface{}{
			map[string]interface{}{"fields": map[string]interface{}{"Label": "Phone"}},
			map[string]interface{}{"fields": map[string]interface{}{"Label": "Tablet"}},
		}})
		require.Equal(t, http.StatusCreated, code, res.Error)
		devs := records(res.Data)
		require.Len(t, devs, 2)
		for _, d := range devs {
			generated, _ := d.Fields["Device ID"].(string)
			parsed, err := uuid.Parse(generated)
			require.NoError(t, err, "a UUID is generated for %s", d.ID)
			assert.Equal(t, uuid.Version(7), parsed.Version())
		}
		assert.NotEqual(t, devs[0].Fields["Device ID"], devs[1].Fields["Device ID"])
		code, res, _ = call("POST", devicesPath+"/records", map[string]interface{}{"fields": map[string]interface{}{
			"Device ID": uuid.NewString(), "Label": "Watch"}})
		assert.Equal(t, http.StatusUnprocessableEntity, code, "an auto UUID can't be written: %+v", res.Error)
		code, res, _ = call("GET", devicesPath+"/records/"+devs[0].ID, nil)
		require.Equal(t, http.StatusBadRequest, code, "the key is Device ID, not the record ID: %+v", res.Error)
		assert.Contains(t, res.Error.Message, "Use records/<Device ID> or records/by-key?key[Device ID]=….")
		// The UUID key in the path finds the record; a value that isn't a UUID finds nothing
		code, res, _ = call("GET", devicesPath+"/records/"+devs[1].Fields["Device ID"].(string), nil)
		require.Equal(t, http.StatusOK, code, res.Error)
		assert.Equal(t, devs[1].ID, record(res.Data).ID)
		code, res, _ = call("GET", devicesPath+"/records/not-a-uuid", nil)
		assert.Equal(t, http.StatusNotFound, code, res.Error)

		// Manual: the field is writable and, being the primary key, required
		badgesPath := base + "/bases/Main/tables/Badges"
		assert.Equal(t, false, uuidField(badgesPath, "Badge ID")["read_only"])
		code, res, _ = call("POST", badgesPath+"/records", map[string]interface{}{"fields": map[string]interface{}{"Label": "Visitor"}})
		assert.Equal(t, http.StatusUnprocessableEntity, code, "a manual UUID key must be sent: %+v", res.Error)
		code, res, _ = call("POST", badgesPath+"/records", map[string]interface{}{"fields": map[string]interface{}{
			"Badge ID": "not-a-uuid", "Label": "Visitor"}})
		assert.Equal(t, http.StatusUnprocessableEntity, code, "an invalid UUID is rejected: %+v", res.Error)
		badgeID := uuid.NewString()
		code, res, _ = call("POST", badgesPath+"/records", map[string]interface{}{"fields": map[string]interface{}{
			"Badge ID": badgeID, "Label": "Visitor"}})
		require.Equal(t, http.StatusCreated, code, res.Error)
		assert.Equal(t, badgeID, record(res.Data).Fields["Badge ID"])
		code, res, _ = call("POST", badgesPath+"/records", map[string]interface{}{"fields": map[string]interface{}{
			"Badge ID": badgeID, "Label": "Again"}})
		assert.Equal(t, http.StatusUnprocessableEntity, code, "the same manual UUID twice is a duplicate key: %+v", res.Error)
	})
}
