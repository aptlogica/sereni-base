package adapter_test

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter/xlsx"
	"github.com/aptlogica/sereni-base/internal/services/transfer/formats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func xlsxModel() *dto.BaseExportFile {
	return &dto.BaseExportFile{
		Base: dto.BaseExportBase{Title: "Sales CRM"},
		Schema: dto.BaseExportSchema{Tables: []dto.BaseExportTable{
			{Key: "t_customers", Title: "Customers", Fields: []dto.BaseExportField{
				{Key: "f_id", Title: "Id", UIDT: "number", System: true},
				{Key: "f_name", Title: "Name", UIDT: "text", OrderIndex: 1},
				{Key: "f_joined", Title: "Joined", UIDT: "date", OrderIndex: 2},
				{Key: "f_seen", Title: "Last seen", UIDT: "datetime", OrderIndex: 3},
				{Key: "f_active", Title: "Active", UIDT: "boolean", OrderIndex: 4},
				{Key: "f_score", Title: "Score", UIDT: "decimal", OrderIndex: 5},
				{Key: "f_age", Title: "Age", UIDT: "number", OrderIndex: 6},
				{Key: "f_tags", Title: "Tags", UIDT: "multiSelect", OrderIndex: 7},
				{Key: "f_docs", Title: "Invoice", UIDT: "attachment", OrderIndex: 8},
				{Key: "f_note", Title: "Note", UIDT: "longText", OrderIndex: 9},
			}},
			{Key: "t_orders", Title: "Orders", Fields: []dto.BaseExportField{{Key: "f_no", Title: "Order no", UIDT: "text", OrderIndex: 1}}},
			{Key: "t_empty", Title: "Products", Fields: []dto.BaseExportField{
				{Key: "f_sku", Title: "Sku", UIDT: "text", OrderIndex: 1},
				{Key: "f_price", Title: "Price", UIDT: "text", OrderIndex: 2},
			}},
		}},
		Data: &dto.BaseExportData{
			Tables: map[string]dto.BaseExportTableRow{
				"t_customers": {Rows: []dto.BaseExportRow{
					{RID: "r1", Values: map[string]interface{}{"f_name": "Asha Rao", "f_joined": "2024-03-01", "f_seen": "2024-03-01T10:30:00Z",
						"f_active": true, "f_score": 83.73333333333333, "f_age": float64(41), "f_tags": []interface{}{"vip", "new"},
						"f_docs": []interface{}{"a1", "a2"}, "f_note": "line1\nline2"}},
					{RID: "r2", Values: map[string]interface{}{"f_name": "=SUM(A1)", "f_active": false, "f_age": float64(30)}},
					{RID: "r3", Values: map[string]interface{}{"f_name": "é日本語😀"}},
				}},
				"t_orders": {Rows: []dto.BaseExportRow{{RID: "o1", Values: map[string]interface{}{"f_no": "O-1"}}}},
			},
			Attachments: []dto.BaseExportAttachment{
				{Ref: "a1", Filename: "invoice-1.pdf", Path: "attachments/a1_invoice-1.pdf", StoragePath: "s/a1"},
				{Ref: "a2", Filename: "gone.pdf", Path: "attachments/a2_gone.pdf", StoragePath: "s/missing"},
			},
		},
	}
}

func exportXLSX(t *testing.T, m *dto.BaseExportFile) []byte {
	t.Helper()
	a, ok := formats.Registry().Get(xlsx.Name)
	require.True(t, ok, "xlsx is a registered format")
	var buf bytes.Buffer
	require.NoError(t, a.Export(context.Background(), m, adapter.ExportOptions{IncludeData: true}, &buf))
	return buf.Bytes()
}

func importXLSX(t *testing.T, data []byte, filename string) (*adapter.ImportResult, error) {
	t.Helper()
	a, _ := formats.Registry().Get(xlsx.Name)
	return a.Import(context.Background(), bytes.NewReader(data), int64(len(data)), adapter.ImportOptions{Filename: filename})
}

func openWorkbook(t *testing.T, data []byte) *excelize.File {
	t.Helper()
	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func TestXLSXDescribeIsAPlainWorkbook(t *testing.T) {
	a, _ := formats.Registry().Get(xlsx.Name)
	name, contentType := a.Describe(adapter.ExportOptions{IncludeData: true}, "Sales: CRM/2024")
	assert.Equal(t, "Sales_CRM_2024.xlsx", name)
	assert.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", contentType)
	// with attachment files it is a zip of the workbook and attachments/
	name, contentType = a.Describe(adapter.ExportOptions{IncludeData: true, IncludeFiles: true}, "Sales: CRM/2024")
	assert.Equal(t, "Sales_CRM_2024.xlsx.zip", name)
	assert.Equal(t, "application/zip", contentType)
	name, _ = a.Describe(adapter.ExportOptions{}, "///")
	assert.Equal(t, "base.xlsx", name)
	assert.True(t, a.Capabilities().Files)
}

func TestXLSXExportMakesOneSheetPerTable(t *testing.T) {
	f := openWorkbook(t, exportXLSX(t, xlsxModel()))
	assert.Equal(t, []string{"Customers", "Orders", "Products"}, f.GetSheetList())

	rows, err := f.GetRows("Customers")
	require.NoError(t, err)
	require.Len(t, rows, 4)
	// system fields are left out, the others come in field order
	assert.Equal(t, []string{"Name", "Joined", "Last seen", "Active", "Score", "Age", "Tags", "Invoice", "Note"}, rows[0])
	assert.Equal(t, []string{"Asha Rao", "2024-03-01", "2024-03-01 10:30:00", "TRUE", "83.7333333333333", "41", "vip|new", "", "line1\nline2"}, rows[1])
	assert.Equal(t, "é日本語😀", rows[3][0])

	// a table with no rows still has its header
	rows, err = f.GetRows("Products")
	require.NoError(t, err)
	assert.Equal(t, [][]string{{"Sku", "Price"}}, rows)
}

func TestXLSXExportWritesRealTypes(t *testing.T) {
	f := openWorkbook(t, exportXLSX(t, xlsxModel()))
	for cell, want := range map[string]excelize.CellType{
		"A2": excelize.CellTypeInlineString, // text
		"B2": excelize.CellTypeUnset,        // a date is a number with a date format
		"D2": excelize.CellTypeBool,
		"E2": excelize.CellTypeUnset, // a number
		"F2": excelize.CellTypeUnset,
	} {
		got, err := f.GetCellType("Customers", cell)
		require.NoError(t, err, cell)
		assert.Equal(t, want, got, cell)
	}
}

// a text that looks like a formula stays text
func TestXLSXExportNeverWritesAFormula(t *testing.T) {
	f := openWorkbook(t, exportXLSX(t, xlsxModel()))
	formula, err := f.GetCellFormula("Customers", "A3")
	require.NoError(t, err)
	assert.Empty(t, formula)
	value, err := f.GetCellValue("Customers", "A3")
	require.NoError(t, err)
	assert.Equal(t, "=SUM(A1)", value)
}

func TestXLSXSheetNames(t *testing.T) {
	tables := []dto.BaseExportTable{
		{Title: "Customers"},
		{Title: "customers"}, // the same ignoring case
		{Title: "Q1/Q2 Sales: North [Region]"},
		{Title: "Quarterly Customer Satisfaction Survey 2024"},
		{Title: "Quarterly Customer Satisfaction Survey 2025"},
		{Title: "   "},
		{Title: "'quoted'"},
		{Title: "History"},
		{Title: "😀😀😀😀😀😀😀😀😀😀😀😀😀😀😀😀😀😀"}, // 18 emoji = 36 UTF-16 units
	}
	names := xlsx.SheetNames(tables)
	assert.Equal(t, "Customers", names[0])
	assert.Equal(t, "customers (2)", names[1])
	assert.Equal(t, "Q1_Q2 Sales_ North _Region_", names[2])
	assert.Equal(t, "Quarterly Customer Satisfaction", names[3])
	assert.Equal(t, "Quarterly Customer Satisfac (2)", names[4])
	assert.Equal(t, "Sheet6", names[5])
	assert.Equal(t, "quoted", names[6])
	assert.Equal(t, "History (2)", names[7])
	seen := map[string]bool{}
	for _, n := range names {
		assert.LessOrEqual(t, len(utf16Units(n)), 31, n)
		assert.False(t, seen[strings.ToLower(n)], "%s is used twice", n)
		seen[strings.ToLower(n)] = true
		assert.NotContains(t, n, "/")
	}
}

func utf16Units(s string) []uint16 {
	var out []uint16
	for _, r := range s {
		if r >= 0x10000 {
			out = append(out, 0, 0)
		} else {
			out = append(out, 0)
		}
	}
	return out
}

// every sheet name the adapter makes is one Excel (and the library) accepts, so the file opens
func TestXLSXExportWithAwkwardTableTitlesIsAValidWorkbook(t *testing.T) {
	m := &dto.BaseExportFile{
		Base: dto.BaseExportBase{Title: "x"},
		Schema: dto.BaseExportSchema{Tables: []dto.BaseExportTable{
			{Key: "a", Title: "Quarterly Customer Satisfaction Survey 2024", Fields: []dto.BaseExportField{{Key: "f", Title: "A", UIDT: "text"}}},
			{Key: "b", Title: "a/b:c*d?e[f]", Fields: []dto.BaseExportField{{Key: "f", Title: "A", UIDT: "text"}}},
		}},
	}
	f := openWorkbook(t, exportXLSX(t, m))
	assert.Len(t, f.GetSheetList(), 2)
}

func TestXLSXRoundTrip(t *testing.T) {
	res, err := importXLSX(t, exportXLSX(t, xlsxModel()), "Sales CRM.xlsx")
	require.NoError(t, err)
	m := res.File
	assert.Equal(t, "Sales CRM", m.Base.Title)
	require.Len(t, m.Schema.Tables, 3)
	assert.Nil(t, res.Attachments)

	customers := m.Schema.Tables[0]
	assert.Equal(t, "Customers", customers.Title)
	types := map[string]string{}
	for _, f := range customers.Fields {
		types[f.Title] = f.UIDT
	}
	assert.Equal(t, map[string]string{
		"Name": "text", "Joined": "date", "Last seen": "datetime", "Active": "boolean",
		"Score": "decimal", "Age": "number", "Tags": "text", "Invoice": "text", "Note": "longText",
	}, types)

	rows := m.Data.Tables[customers.Key].Rows
	require.Len(t, rows, 3)
	byTitle := map[string]string{}
	for _, f := range customers.Fields {
		byTitle[f.Title] = f.Key
	}
	assert.Equal(t, "Asha Rao", rows[0].Values[byTitle["Name"]])
	assert.Equal(t, "2024-03-01", rows[0].Values[byTitle["Joined"]])
	assert.Equal(t, true, rows[0].Values[byTitle["Active"]])
	// Excel keeps 15 significant digits
	assert.InDelta(t, 83.73333333333333, rows[0].Values[byTitle["Score"]], 1e-12)
	assert.Equal(t, "=SUM(A1)", rows[1].Values[byTitle["Name"]])
	assert.Equal(t, "é日本語😀", rows[2].Values[byTitle["Name"]])
	// an empty cell has no value
	_, has := rows[1].Values[byTitle["Joined"]]
	assert.False(t, has)

	// the sheet with only a header is still a table
	assert.Equal(t, "Products", m.Schema.Tables[2].Title)
	assert.Empty(t, m.Data.Tables[m.Schema.Tables[2].Key].Rows)
	// each row names its sheet and line, for the error log
	require.NotNil(t, res.Source)
	assert.Len(t, res.Source.Rows, 4)
}

func TestXLSXImportSkipsHiddenAndEmptySheets(t *testing.T) {
	f := excelize.NewFile()
	require.NoError(t, f.SetSheetName("Sheet1", "Data"))
	require.NoError(t, f.SetSheetRow("Data", "A1", &[]interface{}{"Name", "Age"}))
	require.NoError(t, f.SetSheetRow("Data", "A2", &[]interface{}{"Acme", 3}))
	_, err := f.NewSheet("Hidden")
	require.NoError(t, err)
	require.NoError(t, f.SetSheetRow("Hidden", "A1", &[]interface{}{"x"}))
	require.NoError(t, f.SetSheetVisible("Hidden", false))
	_, err = f.NewSheet("Blank")
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, f.Write(&buf))

	res, err := importXLSX(t, buf.Bytes(), "book.xlsx")
	require.NoError(t, err)
	require.Len(t, res.File.Schema.Tables, 1)
	assert.Equal(t, "Data", res.File.Schema.Tables[0].Title)
}

// a formula is never kept: the cell is read as the value it holds, not as the text of the formula
func TestXLSXImportDoesNotKeepAFormula(t *testing.T) {
	f := excelize.NewFile()
	require.NoError(t, f.SetSheetRow("Sheet1", "A1", &[]interface{}{"A", "B", "Sum"}))
	require.NoError(t, f.SetSheetRow("Sheet1", "A2", &[]interface{}{2, 3}))
	require.NoError(t, f.SetCellFormula("Sheet1", "C2", "A2+B2"))
	var buf bytes.Buffer
	require.NoError(t, f.Write(&buf))

	res, err := importXLSX(t, buf.Bytes(), "calc.xlsx")
	require.NoError(t, err)
	table := res.File.Schema.Tables[0]
	require.Len(t, table.Fields, 3)
	for _, row := range res.File.Data.Tables[table.Key].Rows {
		for _, v := range row.Values {
			assert.NotContains(t, fmt.Sprint(v), "A2+B2")
		}
	}
}

func TestXLSXImportRefusesWhatIsNotAWorkbook(t *testing.T) {
	_, err := importXLSX(t, []byte("not a workbook"), "x.xlsx")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a valid .xlsx")

	for _, name := range []string{"x.csv", "x.xls", "x.xlsm", "x"} {
		_, err = importXLSX(t, []byte("whatever"), name)
		require.Error(t, err, name)
		assert.Contains(t, err.Error(), "only .xlsx or .zip files are supported", name)
	}
	// a .zip that is not a zip
	_, err = importXLSX(t, []byte("whatever"), "x.zip")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a valid zip")

	empty := excelize.NewFile()
	var buf bytes.Buffer
	require.NoError(t, empty.Write(&buf))
	_, err = importXLSX(t, buf.Bytes(), "empty.xlsx")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no sheet with data")
}

func TestXLSXImportRefusesAFileOverTheLimit(t *testing.T) {
	data := exportXLSX(t, xlsxModel())
	a, _ := formats.Registry().Get(xlsx.Name)
	_, err := a.Import(context.Background(), bytes.NewReader(data), int64(len(data)), adapter.ImportOptions{Filename: "x.xlsx", MaxBytes: 100})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too large")
}

// text longer than a cell can hold is cut and reported, and text with control characters still makes a valid file
func TestXLSXExportCutsLongTextAndCleansControlCharacters(t *testing.T) {
	m := &dto.BaseExportFile{
		Base: dto.BaseExportBase{Title: "x"},
		Schema: dto.BaseExportSchema{Tables: []dto.BaseExportTable{
			{Key: "t", Title: "T", Fields: []dto.BaseExportField{{Key: "f", Title: "A", UIDT: "longText"}}},
		}},
		Data: &dto.BaseExportData{Tables: map[string]dto.BaseExportTableRow{"t": {Rows: []dto.BaseExportRow{
			{RID: "r1", Values: map[string]interface{}{"f": strings.Repeat("a", xlsx.MaxCellLength+10)}},
			{RID: "r2", Values: map[string]interface{}{"f": "bell\x07 and nul\x00 stay readable"}},
		}}}},
	}
	f := openWorkbook(t, exportXLSX(t, m))
	v, err := f.GetCellValue("T", "A2")
	require.NoError(t, err)
	assert.Len(t, v, xlsx.MaxCellLength)
	v, err = f.GetCellValue("T", "A3")
	require.NoError(t, err)
	assert.Equal(t, "bell and nul stay readable", v)
	require.Len(t, m.Skipped, 1)
	assert.Contains(t, m.Skipped[0], "cut")
}

func exportXLSXZip(t *testing.T, m *dto.BaseExportFile) ([]byte, error) {
	t.Helper()
	a, _ := formats.Registry().Get(xlsx.Name)
	var buf bytes.Buffer
	err := a.Export(context.Background(), m, adapter.ExportOptions{IncludeData: true, IncludeFiles: true, OpenFile: memoryFiles{"s/a1": "PDF"}.OpenFile}, &buf)
	return buf.Bytes(), err
}

// with files, an export is a zip of the workbook and the files in attachments/, and the cells hold the paths of the
// files that were packed (a file that cannot be read is left out and reported)
func TestXLSXExportWithFilesIsAZipOfTheWorkbookAndAttachments(t *testing.T) {
	m := xlsxModel()
	data, err := exportXLSXZip(t, m)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"attachments/a1_invoice-1.pdf", "Sales_CRM.xlsx"}, zipNames(t, data))
	require.Len(t, m.Skipped, 1)
	assert.Contains(t, m.Skipped[0], "gone.pdf")

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	for _, zf := range zr.File {
		if zf.Name != "Sales_CRM.xlsx" {
			continue
		}
		rc, err := zf.Open()
		require.NoError(t, err)
		book, err := io.ReadAll(rc)
		_ = rc.Close()
		require.NoError(t, err)
		f := openWorkbook(t, book)
		v, err := f.GetCellValue("Customers", "H2")
		require.NoError(t, err)
		assert.Equal(t, "attachments/a1_invoice-1.pdf", v)
	}
}

func TestXLSXZipRoundTripKeepsTheAttachments(t *testing.T) {
	data, err := exportXLSXZip(t, xlsxModel())
	require.NoError(t, err)
	res, err := importXLSX(t, data, "Sales CRM.xlsx.zip")
	require.NoError(t, err)
	assert.Equal(t, "Sales CRM", res.File.Base.Title)

	customers := res.File.Schema.Tables[0]
	var docs dto.BaseExportField
	for _, f := range customers.Fields {
		if f.Title == "Invoice" {
			docs = f
		}
	}
	assert.Equal(t, "attachment", docs.UIDT)
	require.Len(t, res.File.Data.Attachments, 1)
	assert.Equal(t, "invoice-1.pdf", res.File.Data.Attachments[0].Filename)
	require.NotNil(t, res.Attachments)
	assert.True(t, res.Attachments.Has("attachments/a1_invoice-1.pdf"))
}

func TestXLSXZipImportRefusesAZipWithoutOrWithManyWorkbooks(t *testing.T) {
	book := exportXLSX(t, xlsxModel())
	zipOf := func(names ...string) []byte {
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		for _, n := range names {
			w, _ := zw.Create(n)
			_, _ = w.Write(book)
		}
		require.NoError(t, zw.Close())
		return buf.Bytes()
	}
	_, err := importXLSX(t, zipOf("readme.txt"), "x.zip")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no .xlsx file")

	_, err = importXLSX(t, zipOf("a.xlsx", "b.xlsx"), "x.zip")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "more than one")

	// a workbook inside attachments/ is a file, not the workbook
	res, err := importXLSX(t, zipOf("a.xlsx", "attachments/b.xlsx"), "x.zip")
	require.NoError(t, err)
	assert.Len(t, res.File.Schema.Tables, 3)
}
