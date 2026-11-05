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
	Fields    map[string]interface{} `json:"fields"`
	CreatedAt string                 `json:"created_at"`
	UpdatedAt string                 `json:"updated_at"`
}

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
	createTable("In/Out")

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
}
