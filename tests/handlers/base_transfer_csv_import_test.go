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
	w := uploadCSV(t, jobs, svc, map[string]string{"format": "xlsx"}, map[string]string{"Customers.csv": "a\n1\n"})
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
