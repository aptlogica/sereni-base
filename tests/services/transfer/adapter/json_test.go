package adapter_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter/jsonfmt"
	"github.com/aptlogica/sereni-base/internal/services/transfer/formats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func exportJSON(t *testing.T, m *dto.BaseExportFile) ([]byte, error) {
	t.Helper()
	a, ok := formats.Registry().Get(jsonfmt.Name)
	require.True(t, ok, "json is a registered format")
	var buf bytes.Buffer
	err := a.Export(context.Background(), m, adapter.ExportOptions{IncludeData: true, IncludeFiles: true, OpenFile: memoryFiles{"s/a1": "PDF"}.OpenFile}, &buf)
	return buf.Bytes(), err
}

func mustExportJSON(t *testing.T, m *dto.BaseExportFile) []byte {
	t.Helper()
	data, err := exportJSON(t, m)
	require.NoError(t, err)
	return data
}

func importJSON(t *testing.T, data []byte, filename string) (*adapter.ImportResult, error) {
	t.Helper()
	a, _ := formats.Registry().Get(jsonfmt.Name)
	return a.Import(context.Background(), bytes.NewReader(data), int64(len(data)), adapter.ImportOptions{Filename: filename})
}

// jsonZip makes a zip with the given files (name -> text)
func jsonZip(t *testing.T, files map[string]string, order ...string) []byte {
	t.Helper()
	if len(order) == 0 {
		for name := range files {
			order = append(order, name)
		}
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range order {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, _ = w.Write([]byte(files[name]))
	}
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

// decodeRows reads a table file of an export into rows, keeping the digits of the numbers
func decodeRows(t *testing.T, text string) []map[string]interface{} {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(text))
	dec.UseNumber()
	var out []map[string]interface{}
	require.NoError(t, dec.Decode(&out), text)
	return out
}

func TestJSONDescribeIsAlwaysAZip(t *testing.T) {
	a, _ := formats.Registry().Get(jsonfmt.Name)
	for _, opt := range []adapter.ExportOptions{{IncludeData: true}, {IncludeData: true, IncludeFiles: true}, {}} {
		name, contentType := a.Describe(opt, "Sales: CRM/2024")
		assert.Equal(t, "Sales_CRM_2024.json.zip", name)
		assert.Equal(t, "application/zip", contentType)
	}
	name, _ := a.Describe(adapter.ExportOptions{}, "///")
	assert.Equal(t, "base.json.zip", name)
	assert.True(t, a.Capabilities().Files)
}

func TestJSONExportHasOneFilePerTableAndTheAttachments(t *testing.T) {
	m := xlsxModel()
	data := mustExportJSON(t, m)
	// one .json per table, then the files that could be read (the other one is left out and reported)
	assert.ElementsMatch(t, []string{"attachments/a1_invoice-1.pdf", "Customers.json", "Orders.json", "Products.json"}, zipNames(t, data))
	require.Len(t, m.Skipped, 1)
	assert.Contains(t, m.Skipped[0], "gone.pdf")
}

func TestJSONExportTableFileIsAnArrayOfRows(t *testing.T) {
	data := mustExportJSON(t, xlsxModel())
	customers := zipEntry(t, data, "Customers.json")
	rows := decodeRows(t, customers)
	require.Len(t, rows, 3)

	// system fields are left out; every row has every key; a cell with no value is null
	first := rows[0]
	assert.NotContains(t, first, "Id")
	assert.Len(t, first, 9)
	assert.Equal(t, "Asha Rao", first["Name"])
	assert.Equal(t, "2024-03-01", first["Joined"])
	assert.Equal(t, "2024-03-01T10:30:00Z", first["Last seen"])
	assert.Equal(t, true, first["Active"])
	assert.Equal(t, json.Number("83.73333333333333"), first["Score"])
	assert.Equal(t, json.Number("41"), first["Age"])
	assert.Equal(t, []interface{}{"vip", "new"}, first["Tags"])
	// the packed file is named by its path in the zip
	assert.Equal(t, []interface{}{"attachments/a1_invoice-1.pdf"}, first["Invoice"])
	assert.Equal(t, "line1\nline2", first["Note"])
	assert.Nil(t, rows[1]["Joined"])
	assert.Equal(t, "=SUM(A1)", rows[1]["Name"])
	assert.Equal(t, false, rows[1]["Active"])
	assert.Nil(t, rows[1]["Invoice"])
	assert.Equal(t, "é日本語😀", rows[2]["Name"])

	// the fields come in field order
	last := -1
	for _, k := range []string{`"Name"`, `"Joined"`, `"Last seen"`, `"Active"`, `"Score"`, `"Age"`, `"Tags"`, `"Invoice"`, `"Note"`} {
		i := strings.Index(customers, k)
		require.Greater(t, i, last, k)
		last = i
	}

	// a table with no rows is an empty array
	assert.Equal(t, "[]\n", zipEntry(t, data, "Products.json"))
}

func TestJSONExportNumbersTextAndNames(t *testing.T) {
	m := &dto.BaseExportFile{
		Base: dto.BaseExportBase{Title: "T"},
		Schema: dto.BaseExportSchema{Tables: []dto.BaseExportTable{
			{Key: "t1", Title: "A/B: C", Fields: []dto.BaseExportField{
				{Key: "f1", Title: "Big", UIDT: "number", OrderIndex: 1},
				{Key: "f2", Title: "Dup", UIDT: "text", OrderIndex: 2},
				{Key: "f3", Title: "Dup", UIDT: "text", OrderIndex: 3},
				{Key: "f4", Title: "Doc", UIDT: "json", OrderIndex: 4},
			}},
			{Key: "t2", Title: "a/b: c", Fields: []dto.BaseExportField{{Key: "g1", Title: "A", UIDT: "text", OrderIndex: 1}}},
		}},
		Data: &dto.BaseExportData{Tables: map[string]dto.BaseExportTableRow{
			"t1": {Rows: []dto.BaseExportRow{{RID: "r1", Values: map[string]interface{}{
				"f1": json.Number("12345678901234567"), "f2": "<b>&", "f3": "x\x01y", "f4": map[string]interface{}{"a": float64(1)},
			}}}},
		}},
	}
	data := mustExportJSON(t, m)
	// the file names are safe, and unique when two titles differ only in case or in forbidden characters
	assert.ElementsMatch(t, []string{"A_B_ C.json", "a_b_ c (2).json"}, zipNames(t, data))

	text := zipEntry(t, data, "A_B_ C.json")
	assert.Contains(t, text, "<b>&", "text is not escaped as HTML")
	row := decodeRows(t, text)[0]
	// a number too big to be exact is written as text, so no reader rounds it
	assert.Equal(t, "12345678901234567", row["Big"])
	// two fields with one title get two keys, and a control character is dropped
	assert.Equal(t, "<b>&", row["Dup"])
	assert.Equal(t, "xy", row["Dup (2)"])
	assert.Equal(t, map[string]interface{}{"a": json.Number("1")}, row["Doc"])
}

func TestJSONRoundTrip(t *testing.T) {
	res, err := importJSON(t, mustExportJSON(t, xlsxModel()), "Sales CRM.json.zip")
	require.NoError(t, err)
	m := res.File
	assert.Equal(t, "Sales CRM", m.Base.Title)
	require.Len(t, m.Schema.Tables, 3)

	byName := map[string]dto.BaseExportTable{}
	for _, tb := range m.Schema.Tables {
		byName[tb.Title] = tb
	}
	customers := byName["Customers"]
	types, keys := map[string]string{}, map[string]string{}
	for _, f := range customers.Fields {
		types[f.Title] = f.UIDT
		keys[f.Title] = f.Key
	}
	assert.Equal(t, map[string]string{
		"Name": "text", "Joined": "date", "Last seen": "datetime", "Active": "boolean",
		"Score": "decimal", "Age": "number", "Tags": "text", "Invoice": "attachment", "Note": "longText",
	}, types)

	rows := m.Data.Tables[customers.Key].Rows
	require.Len(t, rows, 3)
	assert.Equal(t, "Asha Rao", rows[0].Values[keys["Name"]])
	assert.Equal(t, "2024-03-01", rows[0].Values[keys["Joined"]])
	assert.Equal(t, true, rows[0].Values[keys["Active"]])
	assert.InDelta(t, 83.73333333333333, rows[0].Values[keys["Score"]], 1e-12)
	assert.Equal(t, "vip,new", rows[0].Values[keys["Tags"]])
	assert.Equal(t, "=SUM(A1)", rows[1].Values[keys["Name"]])
	assert.Equal(t, "é日本語😀", rows[2].Values[keys["Name"]])
	// null is no value
	_, has := rows[1].Values[keys["Joined"]]
	assert.False(t, has)

	// the attachment came back with its file
	require.Len(t, m.Data.Attachments, 1)
	assert.Equal(t, "invoice-1.pdf", m.Data.Attachments[0].Filename)
	require.NotNil(t, res.Attachments)
	assert.True(t, res.Attachments.Has("attachments/a1_invoice-1.pdf"))

	// a table with no rows has no fields in its file; it is made with the default field of an empty file
	products := byName["Products"]
	assert.Empty(t, m.Data.Tables[products.Key].Rows)
	require.Len(t, products.Fields, 1)

	// each row names its file and line, for the error log
	require.NotNil(t, res.Source)
	assert.Len(t, res.Source.Rows, 4)
}

func TestJSONImportReadsTheKeysInOrderAndKeepsNumbers(t *testing.T) {
	doc := "\xef\xbb\xbf" + `[
		{"b": 1, "a": "x"},
		{"c": 1.10, "a": "y", "big": 12345678901234567890},
		{"b": null, "a": true, "obj": {"k": [1, 2]}, "list": ["p", "q"]}
	]`
	res, err := importJSON(t, jsonZip(t, map[string]string{"things.json": doc}), "x.zip")
	require.NoError(t, err)
	require.Len(t, res.File.Schema.Tables, 1)
	table := res.File.Schema.Tables[0]
	assert.Equal(t, "things", table.Title)
	var titles []string
	keys := map[string]string{}
	for _, f := range table.Fields {
		titles = append(titles, f.Title)
		keys[f.Title] = f.Key
	}
	// the union of the keys, in order of first appearance
	assert.Equal(t, []string{"b", "a", "c", "big", "obj", "list"}, titles)

	rows := res.File.Data.Tables[table.Key].Rows
	require.Len(t, rows, 3)
	// a list is joined, an object stays JSON text, a number keeps its own digits
	assert.Equal(t, "p,q", rows[2].Values[keys["list"]])
	assert.Equal(t, `{"k":[1,2]}`, rows[2].Values[keys["obj"]])
	assert.Equal(t, "12345678901234567890", rows[1].Values[keys["big"]])
	assert.InDelta(t, 1.1, rows[1].Values[keys["c"]], 1e-12)
}

func TestJSONImportMakesOneTablePerFile(t *testing.T) {
	data := jsonZip(t, map[string]string{
		"Customers.json":       `[{"Name": "Asha"}]`,
		"Orders.json":          `[{"No": 1}, {"No": 2}]`,
		"readme.txt":           "not a table",
		"attachments/doc.json": `[{"ignored": true}]`,
	})
	res, err := importJSON(t, data, "x.json.zip")
	require.NoError(t, err)
	var titles []string
	for _, tb := range res.File.Schema.Tables {
		titles = append(titles, tb.Title)
	}
	assert.ElementsMatch(t, []string{"Customers", "Orders"}, titles)
}

func TestJSONImportRefusesWhatIsNotTheFormat(t *testing.T) {
	for name, tc := range map[string]struct{ file, want string }{
		"not an array":   {`{"a": 1}`, "Orders.json must be an array of objects"},
		"row not object": {`[{"a": 1}, 5]`, "row 2 of Orders.json must be an object"},
		"invalid json":   {`[{"a": `, "Orders.json is not valid JSON"},
		"syntax error":   {`[}`, "Orders.json is not valid JSON"},
		"data after":     {`[] []`, "data after the array of rows"},
		"plain text":     {`hello`, "Orders.json is not valid JSON"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := importJSON(t, jsonZip(t, map[string]string{"Orders.json": tc.file}), "x.zip")
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}

	_, err := importJSON(t, jsonZip(t, map[string]string{"base.sbase.json": `{}`}), "x.zip")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "import it with the sbase format")

	_, err = importJSON(t, jsonZip(t, map[string]string{"readme.txt": "x"}), "x.zip")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no .json file")

	_, err = importJSON(t, []byte("not a zip"), "x.zip")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a valid zip")

	for _, name := range []string{"x.xlsx", "x.csv", "x"} {
		_, err = importJSON(t, []byte("[]"), name)
		require.Error(t, err, name)
		assert.Contains(t, err.Error(), "only .zip or .json files are supported", name)
	}

	// a lone .sbase.json is the other format, and a lone .json must be the rows of a table
	_, err = importJSON(t, []byte("{}"), "x.sbase.json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "import it with the sbase format")
	_, err = importJSON(t, []byte(`{"a": 1}`), "Orders.json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Orders.json must be an array of objects")
}

// one .json file is one table, named after the file; the base is named after the file too
func TestJSONImportTakesASingleFile(t *testing.T) {
	res, err := importJSON(t, []byte(`[{"Name": "Asha", "Age": 31}, {"Name": "Ben", "Age": 25}]`), "My Customers.json")
	require.NoError(t, err)
	require.Len(t, res.File.Schema.Tables, 1)
	tb := res.File.Schema.Tables[0]
	assert.Equal(t, "My Customers", tb.Title)
	assert.Equal(t, "My Customers", res.File.Base.Title)
	require.Len(t, tb.Fields, 2)
	assert.Equal(t, "Name", tb.Fields[0].Title)
	assert.Equal(t, "Age", tb.Fields[1].Title)
	assert.Len(t, res.File.Data.Tables[tb.Key].Rows, 2)
	assert.Nil(t, res.Attachments)

	// an empty array is a table with the default field
	res, err = importJSON(t, []byte("[]"), "Empty.json")
	require.NoError(t, err)
	require.Len(t, res.File.Schema.Tables, 1)
	assert.Equal(t, "Empty", res.File.Schema.Tables[0].Title)
}

func TestJSONImportRefusesAFileOverTheLimit(t *testing.T) {
	a, _ := formats.Registry().Get(jsonfmt.Name)
	data := jsonZip(t, map[string]string{"A.json": `[{"a": "` + strings.Repeat("x", 200) + `"}]`})
	_, err := a.Import(context.Background(), bytes.NewReader(data), int64(len(data)), adapter.ImportOptions{Filename: "a.zip", MaxBytes: 100})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too large")
}

func TestJSONExportWithoutAStorageProviderRefusesFiles(t *testing.T) {
	a, _ := formats.Registry().Get(jsonfmt.Name)
	var buf bytes.Buffer
	err := a.Export(context.Background(), xlsxModel(), adapter.ExportOptions{IncludeData: true}, &buf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no storage provider")
}
