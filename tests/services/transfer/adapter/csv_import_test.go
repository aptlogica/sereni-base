package adapter_test

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/aptlogica/sereni-base/internal/dto"
	transfer "github.com/aptlogica/sereni-base/internal/services/transfer"
	"github.com/aptlogica/sereni-base/internal/services/transfer/adapter"
	csvformat "github.com/aptlogica/sereni-base/internal/services/transfer/adapter/csv"
	"github.com/aptlogica/sereni-base/internal/services/transfer/formats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// zipOf makes a zip from name -> content
func zipOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = io.WriteString(w, content)
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func importCSV(t *testing.T, data []byte, filename string) (*adapter.ImportResult, error) {
	t.Helper()
	a, ok := formats.Registry().Get(csvformat.Name)
	require.True(t, ok)
	return a.Import(context.Background(), bytes.NewReader(data), int64(len(data)), adapter.ImportOptions{Filename: filename})
}

func fieldByTitle(t *testing.T, tbl dto.BaseExportTable, title string) dto.BaseExportField {
	t.Helper()
	for _, f := range tbl.Fields {
		if f.Title == title {
			return f
		}
	}
	t.Fatalf("no field %q in %+v", title, tbl.Fields)
	return dto.BaseExportField{}
}

func TestCSVImportBuildsATablePerFileNamedAfterTheZip(t *testing.T) {
	res, err := importCSV(t, zipOf(t, map[string]string{
		"Customers.csv": "Name,Age\nAcme,3\n",
		"Orders.csv":    "No\nO-1\n",
		"readme.txt":    "ignored",
	}), "Sales CRM.csv.zip")
	require.NoError(t, err)
	assert.Equal(t, "Sales CRM", res.File.Base.Title)
	require.Len(t, res.File.Schema.Tables, 2)
	assert.Equal(t, "Customers", res.File.Schema.Tables[0].Title)
	assert.Equal(t, "Orders", res.File.Schema.Tables[1].Title)
	assert.Nil(t, res.Attachments, "no files in the zip")
	assert.Empty(t, transfer.ValidateBaseFile(res.File), "the model passes the same validation as any file")
}

func TestCSVImportInfersTheTypeOfEachColumn(t *testing.T) {
	csv := strings.Join([]string{
		"Count,Price,Mixed,Paid,Day,Moment,Zip,Note,Long,Empty",
		"1,1.5,1,true,2026-01-31,2026-01-31T10:00:00Z,007,hi,\"a\nb\",",
		"-20,2,2.25,FALSE,2026-02-01,2026-02-01 09:30:00,12345,there,x,",
	}, "\n")
	res, err := importCSV(t, zipOf(t, map[string]string{"T.csv": csv}), "x.zip")
	require.NoError(t, err)
	tbl := res.File.Schema.Tables[0]
	want := map[string]string{
		"Count": "number", "Price": "decimal", "Mixed": "decimal", "Paid": "boolean", "Day": "date", "Moment": "datetime",
		"Zip": "text", "Note": "text", "Long": "longText", "Empty": "text",
	}
	for title, uidt := range want {
		assert.Equal(t, uidt, fieldByTitle(t, tbl, title).UIDT, title)
	}
	row := res.File.Data.Tables[tbl.Key].Rows[1]
	assert.Equal(t, float64(-20), row.Values[fieldByTitle(t, tbl, "Count").Key])
	assert.Equal(t, false, row.Values[fieldByTitle(t, tbl, "Paid").Key])
	assert.Equal(t, "12345", row.Values[fieldByTitle(t, tbl, "Zip").Key])
	_, hasEmpty := row.Values[fieldByTitle(t, tbl, "Empty").Key]
	assert.False(t, hasEmpty, "an empty cell is left out of the row")
	assert.Equal(t, "007", res.File.Data.Tables[tbl.Key].Rows[0].Values[fieldByTitle(t, tbl, "Zip").Key], "a leading zero keeps the text")
	assert.Empty(t, transfer.ValidateBaseFile(res.File))
}

func TestCSVImportFindsTheDelimiterOfEachFile(t *testing.T) {
	files := map[string]string{
		"comma.csv": "a,b\n1,2\n", "semi.csv": "a;b\n1;2\n", "tab.csv": "a\tb\n1\t2\n", "pipe.csv": "a|b\n1|2\n",
	}
	res, err := importCSV(t, zipOf(t, files), "d.zip")
	require.NoError(t, err)
	require.Len(t, res.File.Schema.Tables, 4)
	for _, tbl := range res.File.Schema.Tables {
		require.Len(t, tbl.Fields, 2, tbl.Title)
		assert.Equal(t, []string{"a", "b"}, []string{tbl.Fields[0].Title, tbl.Fields[1].Title}, tbl.Title)
		assert.Equal(t, float64(2), res.File.Data.Tables[tbl.Key].Rows[0].Values[tbl.Fields[1].Key], tbl.Title)
	}
}

func TestCSVImportAcceptsABOMQuotesAndRaggedRows(t *testing.T) {
	csv := "\xEF\xBB\xBFName,Note\r\n\"Acme, Inc\",\"say \"\"hi\"\"\"\r\nshort\r\n\r\n,\r\n"
	res, err := importCSV(t, zipOf(t, map[string]string{"T.csv": csv}), "x.zip")
	require.NoError(t, err)
	tbl := res.File.Schema.Tables[0]
	assert.Equal(t, "Name", tbl.Fields[0].Title, "the byte-order mark is not part of the title")
	rows := res.File.Data.Tables[tbl.Key].Rows
	require.Len(t, rows, 2, "the blank line and the row of empty cells are dropped")
	assert.Equal(t, "Acme, Inc", rows[0].Values[tbl.Fields[0].Key])
	assert.Equal(t, `say "hi"`, rows[0].Values[tbl.Fields[1].Key])
	assert.Equal(t, "short", rows[1].Values[tbl.Fields[0].Key], "a short row is padded")
}

func TestCSVImportCleansTitlesAndNames(t *testing.T) {
	res, err := importCSV(t, zipOf(t, map[string]string{
		"A/Orders.csv": "id,,Name,name,Created time\n1,2,3,4,5\n",
		"B/Orders.csv": "x\n1\n",
	}), "")
	require.NoError(t, err)
	assert.Equal(t, "Imported CSV", res.File.Base.Title, "no file name: a default")
	require.Len(t, res.File.Schema.Tables, 2)
	assert.NotEqual(t, strings.ToLower(res.File.Schema.Tables[0].Title), strings.ToLower(res.File.Schema.Tables[1].Title), "two tables never share a title")
	var titles []string
	for _, f := range res.File.Schema.Tables[0].Fields {
		titles = append(titles, f.Title)
	}
	assert.Equal(t, []string{"id (1)", "Column 2", "Name", "name (2)", "Created time (1)"}, titles,
		"a system title is numbered, an empty one is named, and a repeated one is numbered")
	assert.Empty(t, transfer.ValidateBaseFile(res.File))
}

func TestCSVImportReadsWhatExportWrote(t *testing.T) {
	m := csvModel()
	// the core service gives a file a path directly in attachments/
	m.Data.Attachments[0].Path = "attachments/a1_contract.pdf"
	exported, err := exportCSV(t, m, ';')
	require.NoError(t, err)
	res, err := importCSV(t, exported, "Sales CRM.csv.zip")
	require.NoError(t, err)

	var orders dto.BaseExportTable
	for _, tbl := range res.File.Schema.Tables {
		if tbl.Title == "Orders" {
			orders = tbl
		}
	}
	require.NotEmpty(t, orders.Key)
	assert.Equal(t, "decimal", fieldByTitle(t, orders, "Amount").UIDT)
	assert.Equal(t, "boolean", fieldByTitle(t, orders, "Paid").UIDT)
	assert.Equal(t, "attachment", fieldByTitle(t, orders, "Documents").UIDT, "the paths of files in the zip make an attachment field")
	require.NotNil(t, res.Attachments)
	require.Len(t, res.File.Data.Attachments, 1)
	att := res.File.Data.Attachments[0]
	assert.Equal(t, "contract.pdf", att.Filename)
	assert.Equal(t, "attachments/a1_contract.pdf", att.Path, "the exported path is used as is")
	assert.Empty(t, transfer.ValidateBaseFile(res.File))
}

// An empty cell has no value, whatever the type of its column: it is left out of the row (NULL). A date column once
// got "" for it, which the database refuses and which failed the whole import; a true / false column got false.
func TestCSVImportLeavesOutEveryEmptyCell(t *testing.T) {
	csv := strings.Join([]string{
		"Day,Moment,Paid,Count,Price,Note,Docs",
		"2026-01-31,2026-01-31T10:00:00Z,true,1,1.5,hi,attachments/a1_x.pdf",
		",,,,,,",
		",,,,, ,",
		"2026-02-01,,false,,2,,",
	}, "\n")
	res, err := importCSV(t, zipOf(t, map[string]string{"T.csv": csv, "attachments/a1_x.pdf": "PDF"}), "x.zip")
	require.NoError(t, err)
	tbl := res.File.Schema.Tables[0]
	assert.Equal(t, "date", fieldByTitle(t, tbl, "Day").UIDT, "the type is still chosen from the cells that have a value")
	assert.Equal(t, "boolean", fieldByTitle(t, tbl, "Paid").UIDT)
	assert.Equal(t, "attachment", fieldByTitle(t, tbl, "Docs").UIDT)

	rows := res.File.Data.Tables[tbl.Key].Rows
	require.Len(t, rows, 2, "the rows of only empty cells are dropped, as before")
	for _, title := range []string{"Day", "Moment", "Paid", "Count", "Price", "Note", "Docs"} {
		key := fieldByTitle(t, tbl, title).Key
		for _, row := range rows {
			v, has := row.Values[key]
			if has {
				assert.NotEqual(t, "", v, "%s: an empty cell must never be written as an empty string", title)
			}
		}
	}
	last := rows[1].Values
	_, hasMoment := last[fieldByTitle(t, tbl, "Moment").Key]
	assert.False(t, hasMoment, "an empty datetime is left out")
	assert.Equal(t, false, last[fieldByTitle(t, tbl, "Paid").Key], "a cell that says false is still false")
	_, hasCount := last[fieldByTitle(t, tbl, "Count").Key]
	assert.False(t, hasCount, "an empty number is left out")
	assert.Equal(t, "2026-02-01", last[fieldByTitle(t, tbl, "Day").Key])
	assert.Empty(t, transfer.ValidateBaseFile(res.File))
}

// A column that has a blank in a true / false column is left empty, not false
func TestCSVImportBlankInATrueFalseColumnIsNotFalse(t *testing.T) {
	res, err := importCSV(t, zipOf(t, map[string]string{"T.csv": "Name,Paid\na,true\nb,\nc,FALSE\n"}), "x.zip")
	require.NoError(t, err)
	tbl := res.File.Schema.Tables[0]
	paid := fieldByTitle(t, tbl, "Paid").Key
	rows := res.File.Data.Tables[tbl.Key].Rows
	assert.Equal(t, true, rows[0].Values[paid])
	_, has := rows[1].Values[paid]
	assert.False(t, has)
	assert.Equal(t, false, rows[2].Values[paid])
}

// A number field is a 32 bit integer: a whole number that does not fit is a decimal, never a value the database
// refuses ("value 100021000410003 is out of range for type integer")
func TestCSVImportTypesWholeNumbersByTheRangeOfAnInteger(t *testing.T) {
	csv := strings.Join([]string{
		"Small,Edge,Big,Huge,Negative",
		"1,2147483647,100021000410003,1234567890123456789,-2147483648",
		"2,0,5,1,-1",
	}, "\n")
	res, err := importCSV(t, zipOf(t, map[string]string{"T.csv": csv}), "x.zip")
	require.NoError(t, err)
	tbl := res.File.Schema.Tables[0]
	assert.Equal(t, "number", fieldByTitle(t, tbl, "Small").UIDT)
	assert.Equal(t, "number", fieldByTitle(t, tbl, "Edge").UIDT, "the largest 32 bit integer still fits")
	assert.Equal(t, "number", fieldByTitle(t, tbl, "Negative").UIDT, "so does the smallest")
	assert.Equal(t, "decimal", fieldByTitle(t, tbl, "Big").UIDT, "a 15 digit whole number does not fit an integer")
	assert.Equal(t, "text", fieldByTitle(t, tbl, "Huge").UIDT, "a number of more than 15 digits is not exact as a number")
	assert.Equal(t, float64(100021000410003), res.File.Data.Tables[tbl.Key].Rows[0].Values[fieldByTitle(t, tbl, "Big").Key])
	assert.Equal(t, "1234567890123456789", res.File.Data.Tables[tbl.Key].Rows[0].Values[fieldByTitle(t, tbl, "Huge").Key])
	assert.Empty(t, transfer.ValidateBaseFile(res.File))
}

// Each row knows its file, its line in the file (the header is line 1) and its cells, for the error log; blank and
// repeated rows are counted
func TestCSVImportRemembersWhereEachRowCameFrom(t *testing.T) {
	csv := "No,Note\r\n1,\"two\nlines\"\r\n\r\n2,b\r\n,\r\n2,b\r\n"
	res, err := importCSV(t, zipOf(t, map[string]string{"Orders.csv": csv}), "x.zip")
	require.NoError(t, err)
	require.NotNil(t, res.Source)
	tbl := res.File.Schema.Tables[0]
	rows := res.File.Data.Tables[tbl.Key].Rows
	require.Len(t, rows, 3)

	first := res.Source.Rows[rows[0].RID]
	assert.Equal(t, "Orders.csv", first.File)
	assert.Equal(t, "Orders", first.Table)
	assert.Equal(t, 2, first.Line, "line 1 is the header")
	assert.Equal(t, []string{"No", "Note"}, first.Headers)
	assert.Equal(t, []string{"1", "two\nlines"}, first.Cells)

	// the quoted cell spans two lines, and the blank line is not a row: the next row starts on line 5
	assert.Equal(t, 5, res.Source.Rows[rows[1].RID].Line)
	assert.Equal(t, 7, res.Source.Rows[rows[2].RID].Line)
	assert.Equal(t, 1, res.Source.EmptyRows, "the row of empty cells")
	assert.Equal(t, 1, res.Source.DuplicateRows, "the second 2,b")
}
