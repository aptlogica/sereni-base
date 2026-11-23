package adapter_test

import (
	"archive/zip"
	"bytes"
	"context"
	stdcsv "encoding/csv"
	"strings"
	"testing"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter"
	csvformat "github.com/aptlogica/sereni-base/internal/services/transfer/adapter/csv"
	"github.com/aptlogica/sereni-base/internal/services/transfer/formats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// csvModel is a base with two tables, one of each kind of cell the adapter has to write
func csvModel() *dto.BaseExportFile {
	return &dto.BaseExportFile{
		Base: dto.BaseExportBase{Title: "Sales CRM"},
		Schema: dto.BaseExportSchema{Tables: []dto.BaseExportTable{
			{Key: "t_orders", Title: "Orders", Fields: []dto.BaseExportField{
				{Key: "f_id", Title: "Id", UIDT: "number", System: true, ColumnName: "id"},
				{Key: "f_no", Title: "Order no", UIDT: "text", OrderIndex: 1},
				{Key: "f_customer", Title: "Customer", UIDT: "longText", OrderIndex: 2},
				{Key: "f_amount", Title: "Amount", UIDT: "decimal", OrderIndex: 3},
				{Key: "f_paid", Title: "Paid", UIDT: "boolean", OrderIndex: 4},
				{Key: "f_tags", Title: "Tags", UIDT: "multiSelect", OrderIndex: 5},
				{Key: "f_docs", Title: "Documents", UIDT: "attachment", OrderIndex: 6},
				{Key: "f_note", Title: "Name", UIDT: "text", OrderIndex: 7},
				{Key: "f_note2", Title: "Name", UIDT: "text", OrderIndex: 8},
			}},
			{Key: "t_orders2", Title: "Orders", Fields: []dto.BaseExportField{{Key: "f_x", Title: "X", UIDT: "text"}}},
			{Key: "t_bad", Title: `a/b:c|d`, Fields: []dto.BaseExportField{{Key: "f_y", Title: "Y", UIDT: "text"}}},
		}},
		Data: &dto.BaseExportData{
			Tables: map[string]dto.BaseExportTableRow{
				"t_orders": {Rows: []dto.BaseExportRow{
					{RID: "r1", Values: map[string]interface{}{"f_no": "O-1", "f_customer": "Acme, Inc", "f_amount": 2500.5, "f_paid": true,
						"f_tags": []interface{}{"vip", "new"}, "f_docs": []interface{}{"a1", "a2"}, "f_note": "line1\nline2 \"q\""}},
					{RID: "r2", Values: map[string]interface{}{"f_no": "é日本語😀", "f_amount": float64(120)}},
				}},
			},
			Attachments: []dto.BaseExportAttachment{
				{Ref: "a1", Filename: "contract.pdf", Path: "attachments/a1/contract.pdf", StoragePath: "s/a1"},
				{Ref: "a2", Filename: "gone.pdf", Path: "attachments/a2/gone.pdf", StoragePath: "s/missing"},
			},
		},
	}
}

func csvFiles() memoryFiles { return memoryFiles{"s/a1": "PDF"} }

func exportCSV(t *testing.T, m *dto.BaseExportFile, delim rune) ([]byte, error) {
	t.Helper()
	a, ok := formats.Registry().Get(csvformat.Name)
	require.True(t, ok, "csv is a registered format")
	var buf bytes.Buffer
	err := a.Export(context.Background(), m, adapter.ExportOptions{IncludeData: true, OpenFile: csvFiles().OpenFile, Delimiter: delim}, &buf)
	return buf.Bytes(), err
}

func parse(t *testing.T, text string, delim rune) [][]string {
	t.Helper()
	r := stdcsv.NewReader(strings.NewReader(text))
	r.Comma = delim
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	require.NoError(t, err)
	return rows
}

func zipNames(t *testing.T, data []byte) []string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	return names
}

func TestCSVDescribeIsAlwaysACsvZip(t *testing.T) {
	a, _ := formats.Registry().Get(csvformat.Name)
	for _, d := range []rune{0, ',', ';', '\t', '|'} {
		name, ct := a.Describe(adapter.ExportOptions{Delimiter: d}, "My Base!")
		assert.Equal(t, "My_Base.csv.zip", name)
		assert.Equal(t, "application/zip", ct)
	}
	name, _ := a.Describe(adapter.ExportOptions{}, "***")
	assert.Equal(t, "base.csv.zip", name)
}

func TestCSVAllDelimitersRoundTrip(t *testing.T) {
	for _, d := range []rune{',', ';', '\t', '|'} {
		data, err := exportCSV(t, csvModel(), d)
		require.NoError(t, err)
		text := zipEntry(t, data, "Orders.csv")
		rows := parse(t, text, d)
		require.Len(t, rows, 3, "delimiter %q: header and two rows", string(d))
		assert.Equal(t, []string{"Order no", "Customer", "Amount", "Paid", "Tags", "Documents", "Name", "Name (2)"}, rows[0], "system fields are left out, duplicate titles are numbered")
		assert.Equal(t, "Acme, Inc", rows[1][1])
		assert.Equal(t, "2500.5", rows[1][2], "a decimal point and no thousands separator")
		assert.Equal(t, "true", rows[1][3])
		assert.Equal(t, "line1\nline2 \"q\"", rows[1][6], "a line break and quotes survive")
		assert.Equal(t, "120", rows[2][2])
		assert.Contains(t, text, "\r\n", "CRLF line endings")
	}
}

func TestCSVJoinsManyValuesWithAPipeExceptForThePipeDelimiter(t *testing.T) {
	data, err := exportCSV(t, csvModel(), ',')
	require.NoError(t, err)
	rows := parse(t, zipEntry(t, data, "Orders.csv"), ',')
	assert.Equal(t, "vip|new", rows[1][4])
	assert.Equal(t, "attachments/a1/contract.pdf", rows[1][5], "the file that could not be read is not in the cell")

	data, err = exportCSV(t, csvModel(), '|')
	require.NoError(t, err)
	rows = parse(t, zipEntry(t, data, "Orders.csv"), '|')
	assert.Equal(t, "vip;new", rows[1][4], "with the pipe delimiter a cell is joined with a semicolon")
	assert.Equal(t, 8, len(rows[1]), "the cell does not split into columns")
}

func TestCSVEncodingIsUTF8WithoutABOM(t *testing.T) {
	data, err := exportCSV(t, csvModel(), ',')
	require.NoError(t, err)
	text := zipEntry(t, data, "Orders.csv")
	assert.False(t, strings.HasPrefix(text, "\xEF\xBB\xBF"), "no byte-order mark")
	assert.Contains(t, text, "é日本語😀")
}

func TestCSVPacksAttachmentsAndSkipsAFileItCannotRead(t *testing.T) {
	m := csvModel()
	data, err := exportCSV(t, m, ',')
	require.NoError(t, err)
	names := zipNames(t, data)
	assert.Contains(t, names, "attachments/a1/contract.pdf")
	assert.NotContains(t, names, "attachments/a2/gone.pdf")
	assert.Equal(t, "PDF", zipEntry(t, data, "attachments/a1/contract.pdf"))
	require.Len(t, m.Skipped, 1)
	assert.Contains(t, m.Skipped[0], "gone.pdf")
}

func TestCSVAttachmentPathNeverHoldsTheJoiner(t *testing.T) {
	m := csvModel()
	m.Data.Attachments = []dto.BaseExportAttachment{{Ref: "a1", Filename: "a;b.pdf", Path: "attachments/a1/a;b.pdf", StoragePath: "s/a1"}}
	data, err := exportCSV(t, m, '|')
	require.NoError(t, err)
	assert.Contains(t, zipNames(t, data), "attachments/a1/a_b.pdf", "the zip entry and the cell use the same path")
	rows := parse(t, zipEntry(t, data, "Orders.csv"), '|')
	assert.Equal(t, "attachments/a1/a_b.pdf", rows[1][5])
}

func TestCSVWithoutAttachmentsHasNoFolderAndEmptyCells(t *testing.T) {
	m := csvModel()
	m.Data.Attachments = nil
	data, err := exportCSV(t, m, ',')
	require.NoError(t, err)
	for _, n := range zipNames(t, data) {
		assert.False(t, strings.HasPrefix(n, "attachments/"), n)
	}
	assert.Equal(t, "", parse(t, zipEntry(t, data, "Orders.csv"), ',')[1][5])
}

func TestCSVFilesNeedAnOpenFile(t *testing.T) {
	a, _ := formats.Registry().Get(csvformat.Name)
	err := a.Export(context.Background(), csvModel(), adapter.ExportOptions{IncludeData: true}, &bytes.Buffer{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no storage provider")
}

func TestCSVTableNamesAreSafeAndUnique(t *testing.T) {
	data, err := exportCSV(t, csvModel(), ',')
	require.NoError(t, err)
	names := zipNames(t, data)
	assert.Contains(t, names, "Orders.csv")
	assert.Contains(t, names, "Orders (2).csv")
	assert.Contains(t, names, "a_b_c_d.csv")
}

func TestCSVWithoutRowsIsHeaderOnly(t *testing.T) {
	m := csvModel()
	m.Data = nil
	data, err := exportCSV(t, m, ',')
	require.NoError(t, err)
	assert.Equal(t, "Order no,Customer,Amount,Paid,Tags,Documents,Name,Name (2)\r\n", zipEntry(t, data, "Orders.csv"))
}

func TestCSVRefusesAnUnknownDelimiter(t *testing.T) {
	_, err := exportCSV(t, csvModel(), ' ')
	require.Error(t, err)

	a, _ := formats.Registry().Get(csvformat.Name)
	assert.True(t, a.Capabilities().Rows && a.Capabilities().Files)
	assert.False(t, a.Capabilities().Links || a.Capabilities().Views || a.Capabilities().Constraints)
}

func TestCSVDelimiterWords(t *testing.T) {
	assert.Equal(t, map[string]rune{"comma": ',', "semicolon": ';', "tab": '\t', "pipe": '|'}, csvformat.Delimiters)
	assert.Equal(t, "|", csvformat.Joiner(','))
	assert.Equal(t, ";", csvformat.Joiner('|'))
	assert.Equal(t, ".csv", csvformat.Extension('\t'))
	assert.Equal(t, ".csv", csvformat.Extension(';'))
}
