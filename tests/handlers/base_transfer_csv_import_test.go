package handlers_test

import (
	"archive/zip"
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aptlogica/sereni-base/internal/handlers"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// uploadCSV posts a zip of files as the file of an import; fields are the other form fields
func uploadCSV(t *testing.T, jobs *fakeJobs, svc *capturingTransfer, fields map[string]string, files map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	var payload bytes.Buffer
	mw := multipart.NewWriter(&payload)
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	part, _ := mw.CreateFormFile("file", "Sales CRM.csv.zip")
	zw := zip.NewWriter(part)
	for name, content := range files {
		w, _ := zw.Create(name)
		_, _ = io.WriteString(w, content)
	}
	_ = zw.Close()
	_ = mw.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", &payload)
	c.Request.Header.Set("Content-Type", mw.FormDataContentType())
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	c.Set("schema", "s")
	c.Set("user_id", "user-1")
	var h *handlers.BaseTransferHandler
	if jobs != nil {
		h = handlers.NewBaseTransferHandlerWithJobs(svc, jobs)
	} else {
		h = handlers.NewBaseTransferHandler(svc)
	}
	h.ImportBase(c)
	return w
}

// format=csv reads the zip with the csv adapter and starts the job with that format; triggers and system columns
// are never imported from a csv, whatever is asked
func TestImportCSVStartsAJobWithTheCsvFormat(t *testing.T) {
	jobs, svc := &fakeJobs{}, &capturingTransfer{}
	w := uploadCSV(t, jobs, svc,
		map[string]string{"format": "csv", "title": " Copy ", "include_automations": "true", "include_system_columns": "true"},
		map[string]string{"Customers.csv": "Name,Age\nAcme,3\n"})
	if w.Code != http.StatusAccepted {
		t.Fatalf("status %d, body %s", w.Code, w.Body.String())
	}
	got := jobs.imported.opts
	if got.Format != "csv" || got.Title != "Copy" || got.IncludeAutomations || got.IncludeSystemColumns || !got.IncludeData || !got.SkipFailedRows {
		t.Fatalf("options: %+v", got)
	}
	if svc.called {
		t.Fatal("the base must not be created during the request")
	}
}

// without the job service the import runs during the request, on the model the csv adapter read
func TestImportCSVWithoutJobsRunsOnTheModelOfTheFiles(t *testing.T) {
	svc := &capturingTransfer{}
	w := uploadCSV(t, nil, svc, map[string]string{"format": "csv"}, map[string]string{"Customers.csv": "Name,Age\nAcme,3\n", "Orders.csv": "No\nO-1\n"})
	if w.Code != http.StatusOK || !svc.called {
		t.Fatalf("status %d, body %s", w.Code, w.Body.String())
	}
	if svc.got.Format != "csv" || svc.got.Title != "" || !svc.got.SkipFailedRows {
		t.Fatalf("options: %+v", svc.got)
	}
	// where each row came from reaches the import, to name a row the database refuses
	if svc.got.Source == nil || len(svc.got.Source.Rows) != 2 {
		t.Fatalf("source: %+v", svc.got.Source)
	}
}

// a zip with no .csv or .tsv file is refused at once, in words for the user
func TestImportCSVRefusesAZipWithoutCsvFiles(t *testing.T) {
	jobs, svc := &fakeJobs{}, &capturingTransfer{}
	w := uploadCSV(t, jobs, svc, map[string]string{"format": "csv"}, map[string]string{"readme.txt": "hello"})
	if w.Code != http.StatusBadRequest || !bytes.Contains(w.Body.Bytes(), []byte("no .csv or .tsv file")) {
		t.Fatalf("status %d, body %s", w.Code, w.Body.String())
	}
	if jobs.imported.filename != "" {
		t.Fatal("no job may start")
	}
}

func TestImportRefusesAnUnknownFormat(t *testing.T) {
	jobs, svc := &fakeJobs{}, &capturingTransfer{}
	w := uploadCSV(t, jobs, svc, map[string]string{"format": "pdf"}, map[string]string{"Customers.csv": "a\n1\n"})
	if w.Code != http.StatusBadRequest || jobs.imported.filename != "" {
		t.Fatalf("status %d, body %s", w.Code, w.Body.String())
	}
}

// no format is still a SereniBase file: a zip of CSV files is then refused, as before
func TestImportWithoutAFormatIsStillSbase(t *testing.T) {
	jobs, svc := &fakeJobs{}, &capturingTransfer{}
	w := uploadCSV(t, jobs, svc, nil, map[string]string{"Customers.csv": "a\n1\n"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("a CSV zip without format=csv must be refused as a base file: status %d", w.Code)
	}
}

// a SereniBase import never skips rows: its file promised the values would fit
func TestImportSbaseDoesNotSkipFailedRows(t *testing.T) {
	jobs, svc := &fakeJobs{}, &capturingTransfer{}
	w := importWithJobs(t, jobs, svc, nil, "crm.sbase.zip", importFileJSON)
	if w.Code != http.StatusAccepted || jobs.imported.opts.SkipFailedRows || jobs.imported.opts.Source != nil {
		t.Fatalf("status %d, options %+v", w.Code, jobs.imported.opts)
	}
}

// uploadFile posts one file, as it is, as the file of an import
func uploadFile(t *testing.T, jobs *fakeJobs, svc *capturingTransfer, fields map[string]string, filename string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	var payload bytes.Buffer
	mw := multipart.NewWriter(&payload)
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	part, _ := mw.CreateFormFile("file", filename)
	_, _ = part.Write(content)
	_ = mw.Close()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", &payload)
	c.Request.Header.Set("Content-Type", mw.FormDataContentType())
	c.Params = gin.Params{{Key: "id", Value: "ws-1"}}
	c.Set("schema", "s")
	c.Set("user_id", "user-1")
	handlers.NewBaseTransferHandlerWithJobs(svc, jobs).ImportBase(c)
	return w
}

func workbook(t *testing.T) []byte {
	t.Helper()
	f := excelize.NewFile()
	_ = f.SetSheetRow("Sheet1", "A1", &[]interface{}{"Name", "Age"})
	_ = f.SetSheetRow("Sheet1", "A2", &[]interface{}{"Acme", 3})
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// format=xlsx reads a plain .xlsx with the xlsx adapter and starts the job with that format; triggers and system
// columns are never imported from a workbook, whatever is asked
func TestImportXLSXStartsAJobWithTheXlsxFormat(t *testing.T) {
	jobs, svc := &fakeJobs{}, &capturingTransfer{}
	w := uploadFile(t, jobs, svc, map[string]string{"format": "xlsx", "include_automations": "true", "include_system_columns": "true"}, "Sales CRM.xlsx", workbook(t))
	if w.Code != http.StatusAccepted {
		t.Fatalf("status %d, body %s", w.Code, w.Body.String())
	}
	got := jobs.imported.opts
	if got.Format != "xlsx" || got.IncludeAutomations || got.IncludeSystemColumns || !got.IncludeData || !got.SkipFailedRows {
		t.Fatalf("options: %+v", got)
	}
	if jobs.imported.filename != "Sales CRM.xlsx" {
		t.Fatalf("the job must keep the file name: %q", jobs.imported.filename)
	}
}

// an xlsx import takes an .xlsx or a .zip; a csv or an old workbook is refused at once
func TestImportXLSXRefusesOtherFiles(t *testing.T) {
	for _, name := range []string{"book.csv", "book.xls", "book.xlsm", "book"} {
		jobs, svc := &fakeJobs{}, &capturingTransfer{}
		w := uploadFile(t, jobs, svc, map[string]string{"format": "xlsx"}, name, workbook(t))
		if w.Code != http.StatusBadRequest || !bytes.Contains(w.Body.Bytes(), []byte("only .xlsx or .zip files are supported")) || jobs.imported.filename != "" {
			t.Fatalf("%s: status %d, body %s", name, w.Code, w.Body.String())
		}
	}
}

// a file that is called .xlsx but is not a workbook is refused at once, in words for the user
func TestImportXLSXRefusesAFileThatIsNotAWorkbook(t *testing.T) {
	jobs, svc := &fakeJobs{}, &capturingTransfer{}
	w := uploadFile(t, jobs, svc, map[string]string{"format": "xlsx"}, "book.xlsx", []byte("hello"))
	if w.Code != http.StatusBadRequest || !bytes.Contains(w.Body.Bytes(), []byte("not a valid .xlsx")) || jobs.imported.filename != "" {
		t.Fatalf("status %d, body %s", w.Code, w.Body.String())
	}
}

// an .xlsx is not a SereniBase file: without format=xlsx it is still refused as before
func TestImportWorkbookWithoutAFormatIsRefused(t *testing.T) {
	jobs, svc := &fakeJobs{}, &capturingTransfer{}
	w := uploadFile(t, jobs, svc, nil, "book.xlsx", workbook(t))
	if w.Code != http.StatusBadRequest || !bytes.Contains(w.Body.Bytes(), []byte("only .zip files are supported")) {
		t.Fatalf("status %d, body %s", w.Code, w.Body.String())
	}
}

// the zip of a workbook and its attachments/ is a file an xlsx import takes
func TestImportXLSXTakesAZipOfTheWorkbookAndFiles(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("Sales CRM.xlsx")
	_, _ = w.Write(workbook(t))
	_ = zw.Close()
	jobs, svc := &fakeJobs{}, &capturingTransfer{}
	rec := uploadFile(t, jobs, svc, map[string]string{"format": "xlsx"}, "Sales CRM.xlsx.zip", buf.Bytes())
	if rec.Code != http.StatusAccepted || jobs.imported.opts.Format != "xlsx" || jobs.imported.filename != "Sales CRM.xlsx.zip" {
		t.Fatalf("status %d, body %s, options %+v", rec.Code, rec.Body.String(), jobs.imported.opts)
	}
}

// jsonZip is a zip with one .json file per table
func jsonZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, text := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(text))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// format=json reads a zip of .json files with the json adapter and starts the job with that format; triggers and
// system columns are never imported from it, whatever is asked
func TestImportJSONStartsAJobWithTheJSONFormat(t *testing.T) {
	jobs, svc := &fakeJobs{}, &capturingTransfer{}
	data := jsonZip(t, map[string]string{"Customers.json": `[{"Name": "Asha", "Age": 31}]`, "Products.json": `[]`})
	w := uploadFile(t, jobs, svc, map[string]string{"format": "json", "include_automations": "true", "include_system_columns": "true"}, "Sales CRM.json.zip", data)
	if w.Code != http.StatusAccepted {
		t.Fatalf("status %d, body %s", w.Code, w.Body.String())
	}
	got := jobs.imported.opts
	if got.Format != "json" || got.IncludeAutomations || got.IncludeSystemColumns || !got.IncludeData || !got.SkipFailedRows {
		t.Fatalf("options: %+v", got)
	}
	if jobs.imported.filename != "Sales CRM.json.zip" {
		t.Fatalf("the job must keep the file name: %q", jobs.imported.filename)
	}
}

// a json import takes a .zip or one .json file; any other file is refused at once
func TestImportJSONRefusesOtherFiles(t *testing.T) {
	for _, name := range []string{"book.xlsx", "book.csv", "book"} {
		jobs, svc := &fakeJobs{}, &capturingTransfer{}
		w := uploadFile(t, jobs, svc, map[string]string{"format": "json"}, name, []byte(`[{"a": 1}]`))
		if w.Code != http.StatusBadRequest || !bytes.Contains(w.Body.Bytes(), []byte("only .zip or .json files are supported")) || jobs.imported.filename != "" {
			t.Fatalf("%s: status %d, body %s", name, w.Code, w.Body.String())
		}
	}
}

// a single .json file starts a job with the json format
func TestImportJSONTakesASingleFile(t *testing.T) {
	jobs, svc := &fakeJobs{}, &capturingTransfer{}
	w := uploadFile(t, jobs, svc, map[string]string{"format": "json"}, "Customers.json", []byte(`[{"Name": "Asha"}]`))
	if w.Code != http.StatusAccepted {
		t.Fatalf("status %d, body %s", w.Code, w.Body.String())
	}
	if jobs.imported.filename != "Customers.json" || jobs.imported.opts.Format != "json" {
		t.Fatalf("job: %q %+v", jobs.imported.filename, jobs.imported.opts)
	}
}

// a file that is not the rows of a table is refused at once
func TestImportJSONRefusesASingleFileThatIsNotTheFormat(t *testing.T) {
	jobs, svc := &fakeJobs{}, &capturingTransfer{}
	w := uploadFile(t, jobs, svc, map[string]string{"format": "json"}, "Customers.json", []byte(`{"a": 1}`))
	if w.Code != http.StatusBadRequest || !bytes.Contains(w.Body.Bytes(), []byte("Customers.json must be an array of objects")) || jobs.imported.filename != "" {
		t.Fatalf("status %d, body %s", w.Code, w.Body.String())
	}
}

// a zip whose files are not what the format writes is refused at once, in words for the user
func TestImportJSONRefusesAZipThatIsNotTheFormat(t *testing.T) {
	for _, tc := range []struct {
		files map[string]string
		want  string
	}{
		{map[string]string{"a.json": "hello"}, "a.json is not valid JSON"},
		{map[string]string{"a.json": `{"A": []}`}, "a.json must be an array of objects"},
		{map[string]string{"readme.txt": "x"}, "the zip has no .json file"},
		{map[string]string{"base.sbase.json": `{}`}, "import it with the sbase format"},
	} {
		jobs, svc := &fakeJobs{}, &capturingTransfer{}
		w := uploadFile(t, jobs, svc, map[string]string{"format": "json"}, "data.zip", jsonZip(t, tc.files))
		if w.Code != http.StatusBadRequest || !bytes.Contains(w.Body.Bytes(), []byte(tc.want)) || jobs.imported.filename != "" {
			t.Fatalf("%v: status %d, body %s", tc.files, w.Code, w.Body.String())
		}
	}
}
