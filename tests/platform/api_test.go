package platform_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/middleware"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/platform"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	mockWS       = "ws_abcdefghij"
	mockWSID     = "11111111-1111-1111-1111-111111111111"
	mockBaseID   = "22222222-2222-2222-2222-222222222222"
	mockTableID  = "33333333-3333-3333-3333-333333333333"
	mockTargetID = "44444444-4444-4444-4444-444444444444"
	mockAlias    = "orders_1762164900_ab12"
)

// fakeWriter records the writes the API asks for and returns rows built from them
type fakeWriter struct {
	created [][]map[string]interface{}
	err     error
}

func (f *fakeWriter) CreateRecordsForAPI(_ context.Context, _ string, _ tenant.Model, records []map[string]interface{}) ([]map[string]interface{}, error) {
	f.created = append(f.created, records)
	if f.err != nil {
		return nil, f.err
	}
	out := make([]map[string]interface{}, len(records))
	for i := range records {
		out[i] = map[string]interface{}{"id": json.Number("7"), "created_time": "2026-11-03T10:15:00", "last_modified_time": "2026-11-03T10:15:00"}
		for k, v := range records[i] {
			out[i][k] = v
		}
	}
	return out, nil
}

func (f *fakeWriter) UpdateRecordsForAPI(context.Context, string, tenant.Model, []dto.APIRecordUpdate) ([]map[string]interface{}, error) {
	return nil, f.err
}

func (f *fakeWriter) DeleteRecordsForAPI(context.Context, string, tenant.Model, []int64) ([]int64, error) {
	return nil, f.err
}

func (f *fakeWriter) UpdateRawDataForLinks(context.Context, string, dto.UpdateRowDataLinksRequest) (dto.RecordResponse, error) {
	return dto.RecordResponse{}, nil
}

type mockAPI struct {
	r      *gin.Engine
	mock   sqlmock.Sqlmock
	writer *fakeWriter
}

func newMockAPI(t *testing.T, rateLimit int) *mockAPI {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	writer := &fakeWriter{}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.UseRawPath = true
	r.Use(middleware.RequestID())
	platform.NewAPI(db, "public", writer, rateLimit).Register(r.Group("/api/v1"))
	return &mockAPI{r: r, mock: mock, writer: writer}
}

func q(s string) string { return regexp.QuoteMeta(s) }

// expectSchema expects the workspace lookup and the reads of the base, the table and its fields; extra
// columns are added after the table's own
func (m *mockAPI) expectSchema(extra ...[3]string) {
	m.mock.ExpectQuery(q(`FROM "public".workspace_api WHERE public_id = $1 AND enabled = true`)).WithArgs(mockWS).
		WillReturnRows(sqlmock.NewRows([]string{"workspace_id", "schema_version"}).AddRow(mockWSID, 1))
	m.mock.ExpectQuery(q(`FROM "public".bases WHERE workspace_id = $1`)).WithArgs(mockWSID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).AddRow(mockBaseID, "Main"))
	m.mock.ExpectQuery(q(`FROM "public".models WHERE base_id = $1`)).WithArgs(mockBaseID, mockWSID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "base_id", "title", "alias"}).AddRow(mockTableID, mockBaseID, "Orders", mockAlias))
	cols := sqlmock.NewRows([]string{"id", "title", "column_name", "uidt", "dt", "meta", "system", "order_index"})
	add := func(id, title, column, uidt, meta string, system bool) {
		cols.AddRow(id, title, column, uidt, "", meta, system, 0)
	}
	add("aaaaaaaa-0000-0000-0000-000000000001", "Id", "id", "number", "{}", true)
	add("aaaaaaaa-0000-0000-0000-000000000002", "Created Time", "created_time", "datetime", "{}", true)
	add("aaaaaaaa-0000-0000-0000-000000000003", "Title", "title_x", "text", "{}", false)
	add("aaaaaaaa-0000-0000-0000-000000000004", "Price", "price_x", "decimal", "{}", false)
	add("aaaaaaaa-0000-0000-0000-000000000005", "Status", "status_x", "select", `{"options":[{"option":"Open"},{"option":"Done"}]}`, false)
	add("aaaaaaaa-0000-0000-0000-000000000006", "Tags", "tags_x", "multiSelect", `{"options":["red","blue"]}`, false)
	add("aaaaaaaa-0000-0000-0000-000000000007", "At", "at_x", "datetime", "{}", false)
	add("aaaaaaaa-0000-0000-0000-000000000008", "Took", "took_x", "duration", "{}", false)
	add("aaaaaaaa-0000-0000-0000-000000000009", "Extra", "extra_x", "json", "{}", false)
	add("aaaaaaaa-0000-0000-0000-00000000000a", "Owners", "owners_x", "user", `{"allowMultiple":true}`, false)
	add("aaaaaaaa-0000-0000-0000-00000000000b", "Files", "files_x", "attachment", "{}", false)
	add("aaaaaaaa-0000-0000-0000-00000000000c", "Customer", "customer_x", "links",
		`{"entity_role":"source","relation":{"with":"`+mockTargetID+`","type":"has-many"}}`, false)
	add("aaaaaaaa-0000-0000-0000-00000000000d", "Dup", "dup_1", "text", "{}", false)
	add("aaaaaaaa-0000-0000-0000-00000000000e", "Dup", "dup_2", "text", "{}", false)
	add("aaaaaaaa-0000-0000-0000-00000000000f", "Go", "button_x", "button", "{}", false)
	for _, c := range extra {
		add(c[0], c[1], "", "lookup", c[2], false)
	}
	m.mock.ExpectQuery(q(`FROM "public".columns WHERE model_id = $1`)).WithArgs(mockTableID).WillReturnRows(cols)
}

func (m *mockAPI) expectLinkTarget() {
	m.mock.ExpectQuery(q(`FROM "public".models WHERE id = $1 AND workspace_id = $2`)).WithArgs(mockTargetID, mockWSID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "base_id", "title", "alias"}).AddRow(mockTargetID, mockBaseID, "Customers", "customers_x"))
}

func (m *mockAPI) do(method, path string, body interface{}) (*httptest.ResponseRecorder, apiEnvelope) {
	var reader *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	w := httptest.NewRecorder()
	m.r.ServeHTTP(w, httptest.NewRequest(method, path, reader))
	var out apiEnvelope
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

const ordersURL = "/api/v1/" + mockWS + "/bases/Main/tables/Orders"

func TestAPIUnknownWorkspaceIsCachedAsMiss(t *testing.T) {
	m := newMockAPI(t, 600)
	m.mock.ExpectQuery(q(`FROM "public".workspace_api`)).WithArgs(mockWS).WillReturnRows(sqlmock.NewRows([]string{"workspace_id", "schema_version"}))

	for i := 0; i < 2; i++ {
		w, res := m.do("GET", "/api/v1/"+mockWS+"/bases", nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
		require.NotNil(t, res.Error)
		assert.Equal(t, "not_found", res.Error.Code)
		assert.Equal(t, "null", string(res.Data), "errors have data: null")
		assert.Equal(t, w.Header().Get("X-Request-Id"), res.Meta["request_id"])
	}
	assert.NoError(t, m.mock.ExpectationsWereMet(), "the second request is answered from the cache")

	// Not a public ID at all: no query
	w, res := m.do("GET", "/api/v1/workspace-x/bases", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "not_found", res.Error.Code)
}

func TestAPIListRecordsConvertsValuesAndPages(t *testing.T) {
	m := newMockAPI(t, 600)
	m.expectSchema()
	row := `{"id":5,"created_time":"2026-11-03T10:15:00.5","last_modified_time":"2026-11-04T08:00:00",` +
		`"title_x":"Hello","price_x":12.50,"status_x":"Open","tags_x":["red"],"at_x":"2026-11-03T04:45:00",` +
		`"took_x":"90","extra_x":"{\"a\":1}","owners_x":"u1, u2","files_x":[{"id":"f1","title":"a.png","url":"http://x/a.png","size":10,"mime_type":"image/png","width":3}],` +
		`"customer_x":[{"id":12},{"id":13}],"dup_1":"one","dup_2":"two","button_x":"ignored"}`
	m.mock.ExpectQuery(q(`SELECT row_to_json(t)::text FROM "public"."`+mockAlias+`" AS t WHERE t."status_x" = $1 ORDER BY t."price_x" DESC, t.id LIMIT $2 OFFSET $3`)).
		WithArgs("Open", 2, 0).
		WillReturnRows(sqlmock.NewRows([]string{"row"}).AddRow(row).AddRow(strings.Replace(row, `"id":5`, `"id":6`, 1)))
	m.expectLinkTarget()

	w, res := m.do("GET", ordersURL+"/records?limit=1&filter[Status]=Open&sort=-Price", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, m.mock.ExpectationsWereMet())

	var recs []apiRecord
	require.NoError(t, json.Unmarshal(res.Data, &recs))
	require.Len(t, recs, 1, "limit+1 rows are read to know there is another page")
	rec := recs[0]
	assert.Equal(t, "rec_5", rec.ID)
	assert.Equal(t, "2026-11-03T10:15:00.5Z", rec.CreatedAt)
	assert.Equal(t, "2026-11-04T08:00:00Z", rec.UpdatedAt)
	f := rec.Fields
	assert.Equal(t, "Hello", f["Title"])
	assert.Equal(t, 12.5, f["Price"])
	assert.Equal(t, []interface{}{"red"}, f["Tags"])
	assert.Equal(t, "2026-11-03T04:45:00Z", f["At"])
	assert.Equal(t, float64(90), f["Took"], "duration minutes are a number")
	assert.Equal(t, map[string]interface{}{"a": float64(1)}, f["Extra"], "json text is returned as JSON")
	assert.Equal(t, []interface{}{"u1", "u2"}, f["Owners"])
	assert.Equal(t, []interface{}{map[string]interface{}{"id": "f1", "url": "http://x/a.png", "name": "a.png", "size": float64(10), "type": "image/png"}}, f["Files"])
	assert.Equal(t, []interface{}{map[string]interface{}{"id": "rec_12"}, map[string]interface{}{"id": "rec_13"}}, f["Customer"],
		"links are objects with the record's id")
	assert.Equal(t, "one", f["fld_aaaaaaaa00000000000000000000000d"], "fields sharing a name are keyed by ID")
	assert.Equal(t, "two", f["fld_aaaaaaaa00000000000000000000000e"])
	assert.NotContains(t, f, "Go", "buttons are not data")
	assert.NotContains(t, f, "Id")

	assert.Equal(t, true, res.Meta["has_more"])
	assert.Equal(t, float64(1), res.Meta["limit"])
	next, _ := res.Meta["next_cursor"].(string)
	require.NotEmpty(t, next)

	// Field order in "fields" follows the table
	raw := string(res.Data)
	assert.Less(t, strings.Index(raw, `"Title"`), strings.Index(raw, `"Price"`))

	// The next page uses the offset from the cursor; the schema comes from the cache
	m.mock.ExpectQuery(q(`ORDER BY t."price_x" DESC, t.id LIMIT $2 OFFSET $3`)).WithArgs("Open", 2, 1).
		WillReturnRows(sqlmock.NewRows([]string{"row"}))
	w, res = m.do("GET", ordersURL+"/records?limit=1&filter[Status]=Open&sort=-Price&cursor="+url.QueryEscape(next), nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, false, res.Meta["has_more"])
	assert.Nil(t, res.Meta["next_cursor"])
	assert.NoError(t, m.mock.ExpectationsWereMet())
}

func TestAPIListRecordsBadQueries(t *testing.T) {
	cases := map[string]string{
		"limit too large":        "limit=201",
		"unknown select field":   "select=Nope",
		"ambiguous select name":  "select=Dup",
		"unknown operator":       "filter[Price][like]=1",
		"not a number":           "filter[Price][gt]=abc",
		"json can't be filtered": "filter[Extra]=1",
		"bad link id":            "filter[Customer][contains]=12",
		"unsortable field":       "sort=Tags",
		"malformed filter key":   "filter[Price=1",
		"bad keys value":         "keys=title",
		"garbage cursor":         "cursor=abc",
	}
	for name, query := range cases {
		t.Run(name, func(t *testing.T) {
			m := newMockAPI(t, 600)
			m.expectSchema()
			if strings.Contains(query, "Customer") {
				m.expectLinkTarget()
			}
			w, res := m.do("GET", ordersURL+"/records?"+query, nil)
			require.NotNil(t, res.Error, w.Body.String())
			if name == "ambiguous select name" {
				assert.Equal(t, http.StatusConflict, w.Code)
				assert.Equal(t, "ambiguous_name", res.Error.Code)
				assert.Len(t, res.Error.Details, 2)
				return
			}
			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Equal(t, "bad_request", res.Error.Code)
		})
	}
}

func TestAPICreateValidatesBeforeWriting(t *testing.T) {
	m := newMockAPI(t, 600)
	m.expectSchema()
	w, res := m.do("POST", ordersURL+"/records", map[string]interface{}{"fields": map[string]interface{}{
		"Status": "Maybe", "Tags": []string{"green"}, "At": "yesterday", "Took": -5, "Files": []string{}, "Nope": 1, "Dup": "x",
	}})
	require.Equal(t, http.StatusUnprocessableEntity, w.Code, w.Body.String())
	assert.Equal(t, "validation_failed", res.Error.Code)
	issues := map[string]string{}
	for _, d := range res.Error.Details {
		issues[d.Field] = d.Issue
		assert.Nil(t, d.Index, "single-record errors have no index")
	}
	assert.Contains(t, issues["Status"], "not one of the field's options")
	assert.Contains(t, issues["Tags"], "not one of the field's options")
	assert.Contains(t, issues["At"], "date and time")
	assert.Contains(t, issues["Took"], "minutes")
	assert.Equal(t, "is read-only", issues["Files"])
	assert.Equal(t, "unknown field", issues["Nope"])
	assert.Contains(t, issues["Dup"], "use the field ID")
	assert.Empty(t, m.writer.created, "nothing is written")
}

func TestAPICreateManyConvertsValues(t *testing.T) {
	m := newMockAPI(t, 600)
	m.expectSchema()
	m.expectLinkTarget()
	w, res := m.do("POST", ordersURL+"/records", map[string]interface{}{"records": []interface{}{
		map[string]interface{}{"fields": map[string]interface{}{
			"Title": "A", "Price": 10.25, "Tags": []string{"red", "blue"}, "Took": 45, "Extra": []int{1, 2},
			"At": "2026-11-03T10:15:00+05:30", "fld_aaaaaaaa00000000000000000000000d": "by id"}},
	}})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	require.Len(t, m.writer.created, 1)
	values := m.writer.created[0][0]
	assert.Equal(t, "A", values["title_x"])
	assert.Equal(t, "10.25", values["price_x"], "decimals keep their exact digits")
	assert.Equal(t, "45", values["took_x"], "duration is stored as text minutes")
	assert.Equal(t, "[1,2]", values["extra_x"], "json is stored as text")
	assert.Equal(t, "2026-11-03 04:45:00 +0000 UTC", values["at_x"].(interface{ String() string }).String())
	assert.Equal(t, "by id", values["dup_1"])

	var recs []apiRecord
	require.NoError(t, json.Unmarshal(res.Data, &recs))
	assert.Equal(t, "rec_7", recs[0].ID)
}

func TestAPIBulkRules(t *testing.T) {
	m := newMockAPI(t, 600)
	m.expectSchema()

	// Links can't be written in bulk; the detail carries the record's index
	w, res := m.do("POST", ordersURL+"/records", map[string]interface{}{"records": []interface{}{
		map[string]interface{}{"fields": map[string]interface{}{}},
		map[string]interface{}{"fields": map[string]interface{}{"Customer": []string{"rec_1"}}},
	}})
	require.Equal(t, http.StatusUnprocessableEntity, w.Code, w.Body.String())
	require.NotNil(t, res.Error.Details[0].Index)
	assert.Equal(t, 1, *res.Error.Details[0].Index)

	// Over 100 records
	many := make([]interface{}, 101)
	for i := range many {
		many[i] = map[string]interface{}{"fields": map[string]interface{}{}}
	}
	w, res = m.do("POST", ordersURL+"/records", map[string]interface{}{"records": many})
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	assert.Equal(t, "payload_too_large", res.Error.Code)

	// Typos in the body are rejected, not ignored
	w, res = m.do("POST", ordersURL+"/records", map[string]interface{}{"field": map[string]interface{}{}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "bad_request", res.Error.Code)

	// The same ID twice in a bulk delete
	w, res = m.do("POST", ordersURL+"/records/delete", map[string]interface{}{"ids": []string{"rec_1", "rec_1"}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, res.Error.Message, "appears twice")

	// Not JSON
	w, res = m.do("PATCH", ordersURL+"/records", "not an object")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "bad_request", res.Error.Code)
}

func TestAPIWriteErrorsKeepTheAppMessage(t *testing.T) {
	m := newMockAPI(t, 600)
	m.expectSchema()
	m.writer.err = &app_errors.RecordWriteError{Index: 0, Err: app_errors.WithMessage(app_errors.UniqueKeyViolation, "Duplicate value not allowed. Two records can't have the same Code.")}
	w, res := m.do("POST", ordersURL+"/records", map[string]interface{}{"fields": map[string]interface{}{"Title": "A"}})
	require.Equal(t, http.StatusUnprocessableEntity, w.Code, w.Body.String())
	assert.Equal(t, "Duplicate value not allowed. Two records can't have the same Code.", res.Error.Details[0].Issue)

	m.writer.err = &app_errors.RecordWriteError{Index: 0, Err: app_errors.DatabaseError}
	w, res = m.do("POST", ordersURL+"/records", map[string]interface{}{"fields": map[string]interface{}{"Title": "A"}})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "internal_error", res.Error.Code)
	assert.NotContains(t, w.Body.String(), "database", "database errors are not shown")
}

func TestAPIRateLimitPerWorkspace(t *testing.T) {
	m := newMockAPI(t, 2)
	m.mock.ExpectQuery(q(`FROM "public".workspace_api`)).WithArgs(mockWS).
		WillReturnRows(sqlmock.NewRows([]string{"workspace_id", "schema_version"}).AddRow(mockWSID, 1))
	m.mock.ExpectQuery(q(`FROM "public".bases`)).WillReturnRows(sqlmock.NewRows([]string{"id", "title"}))

	for i := 0; i < 2; i++ {
		w, _ := m.do("GET", "/api/v1/"+mockWS+"/bases", nil)
		require.Equal(t, http.StatusOK, w.Code)
	}
	w, res := m.do("GET", "/api/v1/"+mockWS+"/bases", nil)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "rate_limited", res.Error.Code)
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
}

func TestAPISchemaEndpoints(t *testing.T) {
	m := newMockAPI(t, 600)
	m.expectSchema()
	w, res := m.do("GET", ordersURL, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	body := string(res.Data)
	assert.Contains(t, body, `"id":"tbl_33333333333333333333333333333333"`)
	assert.Contains(t, body, `"base_id":"bse_22222222222222222222222222222222"`)
	assert.Contains(t, body, `{"id":"fld_aaaaaaaa000000000000000000000005","name":"Status","type":"single_select","read_only":false,"options":["Open","Done"]}`)
	assert.Contains(t, body, `"name":"Customer","type":"links","read_only":false,"linked_table_id":"tbl_44444444444444444444444444444444","multiple":true`)
	assert.Contains(t, body, `"name":"Files","type":"attachment","read_only":true`)
	assert.NotContains(t, body, `"name":"Created Time"`)

	// By ID, and an ID that doesn't exist
	w, _ = m.do("GET", "/api/v1/"+mockWS+"/bases/bse_22222222222222222222222222222222/tables/tbl_33333333333333333333333333333333", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	w, res = m.do("GET", "/api/v1/"+mockWS+"/bases/Main/tables/tbl_99999999999999999999999999999999", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "not_found", res.Error.Code)
}

// lookupColumn is an extra column for expectSchema: a lookup through Customer of a Customers field
func lookupColumn(id, title, targetColumnID string) [3]string {
	return [3]string{id, title, `{"link_column_id":"aaaaaaaa-0000-0000-0000-00000000000c","lookup_column_id":"` + targetColumnID + `"}`}
}

func TestAPILinkObjectsCarryLookups(t *testing.T) {
	const (
		targetName  = "bbbbbbbb-0000-0000-0000-000000000002"
		targetEmail = "bbbbbbbb-0000-0000-0000-000000000003"
	)
	m := newMockAPI(t, 600)
	m.expectSchema(
		lookupColumn("aaaaaaaa-0000-0000-0000-000000000010", "Customer name", targetName),
		lookupColumn("aaaaaaaa-0000-0000-0000-000000000011", "Customer email", targetEmail),
		lookupColumn("aaaaaaaa-0000-0000-0000-000000000012", "Name again", targetName),
	)
	listRows := func() *sqlmock.Rows {
		return sqlmock.NewRows([]string{"row"}).
			AddRow(`{"id":5,"customer_x":[{"id":12},{"id":13}]}`).
			AddRow(`{"id":6,"customer_x":null}`)
	}
	values := func(id12, id13 interface{}) *sqlmock.Rows {
		return sqlmock.NewRows([]string{"id", "v"}).AddRow(12, id12).AddRow(13, id13)
	}
	m.mock.ExpectQuery(q(`FROM "public"."` + mockAlias + `" AS t ORDER BY t.id`)).WillReturnRows(listRows())
	m.expectLinkTarget()
	m.mock.ExpectQuery(q(`FROM "public".columns WHERE model_id = $1`)).WithArgs(mockTargetID).WillReturnRows(
		sqlmock.NewRows([]string{"id", "title", "column_name", "uidt", "dt", "meta", "system", "order_index"}).
			AddRow("bbbbbbbb-0000-0000-0000-000000000001", "Id", "id", "number", "", "{}", true, 0).
			AddRow(targetName, "Name", "name_x", "text", "", "{}", false, 1).
			AddRow(targetEmail, "Email", "email_x", "email", "", "{}", false, 2))
	m.mock.ExpectQuery(q(`SELECT x.id, (to_jsonb(x) -> $1)::text FROM "public"."customers_x"`)).
		WithArgs("name_x", sqlmock.AnyArg()).WillReturnRows(values(`"Asha"`, `"Ravi"`))
	m.mock.ExpectQuery(q(`SELECT x.id, (to_jsonb(x) -> $1)::text FROM "public"."customers_x"`)).
		WithArgs("email_x", sqlmock.AnyArg()).WillReturnRows(values(`"asha@example.com"`, nil))

	w, res := m.do("GET", ordersURL+"/records?select=Title,Customer", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, m.mock.ExpectationsWereMet())
	var recs []apiRecord
	require.NoError(t, json.Unmarshal(res.Data, &recs))
	require.Len(t, recs, 2)
	assert.Equal(t, []interface{}{
		map[string]interface{}{"id": "rec_12", "Name": "Asha", "Email": "asha@example.com"},
		map[string]interface{}{"id": "rec_13", "Name": "Ravi", "Email": nil},
	}, recs[0].Fields["Customer"], "a selected link carries all its lookups, each looked-up field once")
	assert.Equal(t, []interface{}{}, recs[1].Fields["Customer"])
	assert.NotContains(t, recs[0].Fields, "Customer name", "lookups are not keys of their own")
	assert.Contains(t, string(res.Data), `{"id":"rec_12","Name":"Asha","Email":"asha@example.com"}`, "id first, then the lookups in order")

	// A selected lookup brings its link with only that lookup, keyed by ID with keys=id
	m.mock.ExpectQuery(q(`FROM "public"."` + mockAlias + `" AS t ORDER BY t.id`)).WillReturnRows(listRows())
	m.mock.ExpectQuery(q(`SELECT x.id, (to_jsonb(x) -> $1)::text FROM "public"."customers_x"`)).
		WithArgs("email_x", sqlmock.AnyArg()).WillReturnRows(values(`"asha@example.com"`, nil))
	w, res = m.do("GET", ordersURL+"/records?select=Customer%20email&keys=id", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, m.mock.ExpectationsWereMet())
	recs = nil
	require.NoError(t, json.Unmarshal(res.Data, &recs))
	assert.Equal(t, map[string]interface{}{
		"fld_aaaaaaaa00000000000000000000000c": []interface{}{
			map[string]interface{}{"id": "rec_12", "fld_bbbbbbbb000000000000000000000003": "asha@example.com"},
			map[string]interface{}{"id": "rec_13", "fld_bbbbbbbb000000000000000000000003": nil},
		},
	}, recs[0].Fields)

	// The table lists the lookups under the link field, not as fields
	w, res = m.do("GET", ordersURL, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	body := string(res.Data)
	assert.Contains(t, body, `"lookups":[{"key":"Name","id":"fld_bbbbbbbb000000000000000000000002","name":"Name","type":"text"},`+
		`{"key":"Email","id":"fld_bbbbbbbb000000000000000000000003","name":"Email","type":"email"}]`)
	assert.NotContains(t, body, `"name":"Customer name"`)

	// Lookups stay read-only, and link values may be objects with an id
	w, res = m.do("POST", ordersURL+"/records", map[string]interface{}{"fields": map[string]interface{}{
		"Customer name": "x", "Customer": []interface{}{map[string]interface{}{"name": "no id"}}}})
	require.Equal(t, http.StatusUnprocessableEntity, w.Code, w.Body.String())
	issues := map[string]string{}
	for _, d := range res.Error.Details {
		issues[d.Field] = d.Issue
	}
	assert.Equal(t, "is read-only", issues["Customer name"])
	assert.Equal(t, "must be an array of record IDs like rec_12, or of objects with an id", issues["Customer"])
}
