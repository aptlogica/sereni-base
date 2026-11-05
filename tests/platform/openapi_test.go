package platform_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	specURL  = "/api/v1/" + mockWS + "/openapi.json"
	mockTID  = "tbl_33333333333333333333333333333333"
	ordersAt = "/bases/Main/tables/Orders"
)

// expectLinkLookups expects the reads of Customer's linked table and its fields, for the link objects' lookups
func (m *mockAPI) expectLinkLookups() {
	m.expectLinkTarget()
	m.mock.ExpectQuery(q(`FROM "public".columns WHERE model_id = $1`)).WithArgs(mockTargetID).WillReturnRows(
		sqlmock.NewRows([]string{"id", "title", "column_name", "uidt", "dt", "meta", "system", "order_index"}).
			AddRow("bbbbbbbb-0000-0000-0000-000000000003", "Email", "email_x", "email", "", "{}", false, 1))
}

func getSpec(t *testing.T, m *mockAPI, url string) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	w, _ := m.do("GET", url, nil)
	var spec map[string]interface{}
	if w.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &spec), w.Body.String())
	}
	return w, spec
}

// obj walks nested objects of a decoded spec
func obj(t *testing.T, v interface{}, keys ...string) map[string]interface{} {
	t.Helper()
	for _, k := range keys {
		m, ok := v.(map[string]interface{})
		require.True(t, ok, "no object at %q", k)
		v = m[k]
	}
	m, ok := v.(map[string]interface{})
	require.True(t, ok, "not an object: %v", keys)
	return m
}

func TestOpenAPISpecDescribesTheTables(t *testing.T) {
	m := newMockAPI(t, 600)
	m.expectSchema(lookupColumn("aaaaaaaa-0000-0000-0000-000000000010", "Customer email", "bbbbbbbb-0000-0000-0000-000000000003"))
	m.expectLinkLookups()

	w, spec := getSpec(t, m, specURL)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NoError(t, m.mock.ExpectationsWereMet())
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	// A valid OpenAPI document
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(w.Body.Bytes())
	require.NoError(t, err)
	require.NoError(t, doc.Validate(context.Background()))

	assert.Equal(t, []interface{}{map[string]interface{}{"url": "/api/v1/" + mockWS}}, spec["servers"])
	paths := obj(t, spec, "paths")
	for _, p := range []string{"/bases", "/bases/{base}/tables/{table}", ordersAt + "/records", ordersAt + "/records/{record}", ordersAt + "/records/delete"} {
		assert.Contains(t, paths, p)
	}
	assert.NotContains(t, paths, "/bases/{base}/tables/{table}/records", "the generic records paths give way to the tables'")

	list := obj(t, paths, ordersAt+"/records", "get")
	assert.Equal(t, "listRecords_"+mockTID, list["operationId"])
	assert.Equal(t, []interface{}{"Main / Orders"}, list["tags"])
	assert.Empty(t, obj(t, paths, ordersAt+"/records")["parameters"], "no base or table parameters")

	schemas := obj(t, spec, "components", "schemas")
	fields := obj(t, schemas, "Fields_"+mockTID, "properties")
	assert.Equal(t, map[string]interface{}{"type": "string", "nullable": true, "enum": []interface{}{"Open", "Done"},
		"description": "single_select field, fld_aaaaaaaa000000000000000000000005."}, fields["Status"])
	assert.Equal(t, []interface{}{"red", "blue"}, obj(t, fields, "Tags", "items")["enum"])
	assert.Equal(t, "number", obj(t, fields, "Price")["type"])
	assert.Equal(t, true, obj(t, fields, "Files")["readOnly"])
	assert.Contains(t, fields, "fld_aaaaaaaa00000000000000000000000d", "fields sharing a name are keyed by ID")
	assert.NotContains(t, fields, "Customer email", "lookups live in the link objects")
	assert.NotContains(t, fields, "Go", "buttons are not data")

	linkRef := "LinkRef_" + mockTID + "_fld_aaaaaaaa00000000000000000000000c"
	assert.Equal(t, "#/components/schemas/"+linkRef, obj(t, fields, "Customer", "items")["$ref"])
	assert.Equal(t, "email", obj(t, schemas, linkRef, "properties", "Email")["format"])

	write := obj(t, schemas, "FieldsWrite_"+mockTID, "properties")
	assert.NotContains(t, write, "Files", "read-only fields can't be written")
	assert.Contains(t, obj(t, write, "Customer", "items"), "oneOf")
	assert.NotContains(t, obj(t, schemas, "FieldsBulkWrite_"+mockTID, "properties"), "Customer", "no links in bulk writes")

	// Reading it again is cached: no queries, and the ETag answers 304
	etag := w.Header().Get("ETag")
	require.NotEmpty(t, etag)
	req := httptest.NewRequest("GET", specURL, nil)
	req.Header.Set("If-None-Match", etag)
	w = httptest.NewRecorder()
	m.r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotModified, w.Code)
	require.NoError(t, m.mock.ExpectationsWereMet())
}

func TestOpenAPISpecForOneBase(t *testing.T) {
	m := newMockAPI(t, 600)
	m.expectSchema()

	w, spec := getSpec(t, m, specURL+"?base=Main")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, obj(t, spec, "paths"), ordersAt+"/records")

	w, res := m.do("GET", specURL+"?base=Nope", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "not_found", res.Error.Code)
	require.NoError(t, m.mock.ExpectationsWereMet())
}

func TestDocsPagesLoadTheSpec(t *testing.T) {
	m := newMockAPI(t, 600)
	m.mock.ExpectQuery(q(`FROM "public".workspace_api WHERE public_id = $1 AND enabled = true`)).WithArgs(mockWS).
		WillReturnRows(sqlmock.NewRows([]string{"workspace_id", "schema_version"}).AddRow(mockWSID, 1))

	w, _ := m.do("GET", "/api/v1/"+mockWS+"/docs", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), `SwaggerUIBundle({url: "openapi.json"`)
	assert.Contains(t, w.Body.String(), `/swagger/swagger-ui-bundle.js`)

	w, _ = m.do("GET", "/api/v1/"+mockWS+"/redoc?base=Main", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `<redoc spec-url="openapi.json?base=Main">`)
	assert.Contains(t, w.Body.String(), `redoc@2.1.5/bundles/redoc.standalone.js`)
	require.NoError(t, m.mock.ExpectationsWereMet())
}
