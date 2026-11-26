package handlers_test

import (
	"archive/zip"
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aptlogica/sereni-base/internal/handlers"
	"github.com/gin-gonic/gin"
)

type upFile struct {
	name    string
	content []byte
}

// uploadMany posts several parts named file, as the import dialog does for loose CSV files
func uploadMany(t *testing.T, jobs *fakeJobs, svc *capturingTransfer, format string, files []upFile) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	var payload bytes.Buffer
	mw := multipart.NewWriter(&payload)
	_ = mw.WriteField("format", format)
	for _, f := range files {
		part, _ := mw.CreateFormFile("file", f.name)
		_, _ = part.Write(f.content)
	}
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

func zipNames(t *testing.T, data string) []string {
	t.Helper()
	zr, err := zip.NewReader(strings.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("not a zip: %v", err)
	}
	var names []string
	for _, f := range zr.File {
		rc, _ := f.Open()
		_, _ = io.Copy(io.Discard, rc)
		rc.Close()
		names = append(names, f.Name)
	}
	return names
}

func TestImportSeveralCSVFilesStartsOneJobNamedImportedCSV(t *testing.T) {
	jobs, svc := &fakeJobs{}, &capturingTransfer{}
	w := uploadMany(t, jobs, svc, "csv", []upFile{{"Customers.csv", []byte("Name\nAcme\n")}, {"Orders.csv", []byte("No\nO-1\n")}})
	if w.Code != http.StatusAccepted {
		t.Fatalf("status %d, body %s", w.Code, w.Body.String())
	}
	if jobs.imported.filename != "Imported CSV.zip" || jobs.imported.opts.Format != "csv" || !jobs.imported.opts.SkipFailedRows {
		t.Fatalf("job: %+v", jobs.imported)
	}
	names := zipNames(t, jobs.imported.data)
	if len(names) != 2 || names[0] != "Customers.csv" || names[1] != "Orders.csv" {
		t.Fatalf("files in the import: %v", names)
	}
}

func TestImportOneCSVFileIsNamedAfterTheFile(t *testing.T) {
	jobs, svc := &fakeJobs{}, &capturingTransfer{}
	w := uploadMany(t, jobs, svc, "csv", []upFile{{"Sales 2026.csv", []byte("Name\nAcme\n")}})
	if w.Code != http.StatusAccepted || jobs.imported.filename != "Sales 2026.zip" {
		t.Fatalf("status %d, filename %q, body %s", w.Code, jobs.imported.filename, w.Body.String())
	}
}

func TestImportCSVRefusesBadFiles(t *testing.T) {
	zipBytes := func() []byte {
		var b bytes.Buffer
		zw := zip.NewWriter(&b)
		_, _ = zw.Create("a.csv")
		_ = zw.Close()
		return b.Bytes()
	}()
	good := upFile{"good.csv", []byte("Name\nAcme\n")}
	cases := []struct {
		name  string
		files []upFile
		want  string
	}{
		{"not a csv", []upFile{good, {"report.xlsx", []byte("x")}}, "report.xlsx is not a CSV file"},
		{"empty", []upFile{{"empty.csv", nil}}, "empty.csv is empty"},
		{"zip renamed csv", []upFile{{"fake.csv", zipBytes}}, "fake.csv is not a valid CSV file"},
		{"binary renamed csv", []upFile{{"image.csv", []byte("ab\x00cd")}}, "image.csv is not a valid CSV file"},
		{"same name twice", []upFile{good, {"GOOD.csv", []byte("a\n1\n")}}, "was added twice"},
		{"csv and zip", []upFile{good, {"all.zip", zipBytes}}, "either CSV files or one .zip"},
		{"two zips", []upFile{{"a.zip", zipBytes}, {"b.zip", zipBytes}}, "only one .zip"},
		{"hidden name", []upFile{{".hidden.csv", []byte("a\n1\n")}}, "not a valid CSV file name"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			jobs, svc := &fakeJobs{}, &capturingTransfer{}
			w := uploadMany(t, jobs, svc, "csv", tc.files)
			if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), tc.want) {
				t.Fatalf("status %d, body %s", w.Code, w.Body.String())
			}
			if jobs.imported.filename != "" {
				t.Fatal("no job may start")
			}
		})
	}
}

func TestImportOtherFormatsTakeOnlyOneFile(t *testing.T) {
	jobs, svc := &fakeJobs{}, &capturingTransfer{}
	w := uploadMany(t, jobs, svc, "sbase", []upFile{{"a.zip", []byte("x")}, {"b.zip", []byte("x")}})
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "only one file is supported") {
		t.Fatalf("status %d, body %s", w.Code, w.Body.String())
	}
}
