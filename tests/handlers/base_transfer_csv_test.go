package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/handlers"
	"github.com/gin-gonic/gin"
)

const csvBaseID = "33cc3ba3-9fac-43d8-866f-0bd1e44548eb"

// A csv export is a zip named .csv.zip; links become text and triggers and system columns are switched off
func TestExportBaseCSVIsADirectCsvZip(t *testing.T) {
	svc, w := exportAs(t, "format=csv&include_system_columns=true&include_automations=true", nil, true)
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "application/zip" ||
		!strings.Contains(w.Header().Get("Content-Disposition"), "CRM.csv.zip") {
		t.Fatalf("status %d, headers %v", w.Code, w.Header())
	}
	got := svc.exported
	if got.Format != "csv" || got.Delimiter != ',' || !got.LinksAsText || got.IncludeAutomations || got.IncludeSystemColumns {
		t.Fatalf("csv options: %+v", got)
	}
}

func TestExportBaseDefaultsToSbase(t *testing.T) {
	svc, w := exportAs(t, "", nil, true)
	if w.Code != http.StatusOK || !strings.Contains(w.Header().Get("Content-Disposition"), ".sbase.zip") {
		t.Fatalf("status %d, headers %v", w.Code, w.Header())
	}
	if svc.exported.Format != "sbase" || svc.exported.LinksAsText || svc.exported.Delimiter != 0 {
		t.Fatalf("sbase options: %+v", svc.exported)
	}
	// a delimiter means nothing to sbase
	if svc, _ = exportAs(t, "format=sbase&delimiter=tab", nil, true); svc.exported.Delimiter != 0 {
		t.Fatalf("sbase must ignore the delimiter: %+v", svc.exported)
	}
}

func TestExportBaseCSVDelimiters(t *testing.T) {
	for word, want := range map[string]rune{"comma": ',', "semicolon": ';', "tab": '\t', "pipe": '|', "PIPE": '|'} {
		svc, w := exportWith(t, "format=csv&delimiter="+word, nil)
		if w.Code != http.StatusOK || svc.exported.Delimiter != want {
			t.Fatalf("delimiter=%s: status %d, options %+v", word, w.Code, svc.exported)
		}
	}
}

func TestExportBaseRefusesAnUnknownFormatOrDelimiter(t *testing.T) {
	for _, query := range []string{"format=pdf", "format=csv&delimiter=space", "format=csv&delimiter=%2C", "format=csv&delimiter=colon"} {
		svc, w := exportWith(t, query, nil)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("%s: status %d", query, w.Code)
		}
		if svc.exported.Format != "" {
			t.Fatalf("%s: nothing may be exported, got %+v", query, svc.exported)
		}
	}
}

func TestExportBaseCSVWithDataStartsAJobWithTheFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jobs := &fakeJobs{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?include_data=true&format=csv&delimiter=tab", nil)
	c.Params = gin.Params{{Key: "id", Value: csvBaseID}}
	c.Set("schema", "s")
	c.Set("user_id", "user-1")
	c.Set(handlers.AutomationsAllowedKey, true)
	handlers.NewBaseTransferHandlerWithJobs(&capturingTransfer{}, jobs).ExportBase(c)
	if w.Code != http.StatusAccepted {
		t.Fatalf("status %d, body %s", w.Code, w.Body.String())
	}
	got := jobs.started
	if got.Format != "csv" || got.Delimiter != '\t' || !got.LinksAsText || got.IncludeAutomations || !got.IncludeData {
		t.Fatalf("job started with %+v", got)
	}
}

func TestExportBaseWithBodyTakesTheFormatAndDelimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	post := func(body string) (*capturingTransfer, *httptest.ResponseRecorder) {
		svc := &capturingTransfer{file: &dto.BaseExportFile{Base: dto.BaseExportBase{Title: "CRM"}}}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: csvBaseID}}
		c.Set("schema", "s")
		handlers.NewBaseTransferHandler(svc).ExportBaseWithBody(c)
		return svc, w
	}
	svc, w := post(`{"format":"csv","delimiter":"pipe","include_data":false}`)
	if w.Code != http.StatusOK || svc.exported.Format != "csv" || svc.exported.Delimiter != '|' || !svc.exported.LinksAsText {
		t.Fatalf("status %d, options %+v", w.Code, svc.exported)
	}
	if _, w = post(`{"format":"csv","delimiter":"space"}`); w.Code != http.StatusBadRequest {
		t.Fatalf("a bad delimiter in the body: status %d", w.Code)
	}
	if svc, w = post(`{"include_data":false}`); w.Code != http.StatusOK || svc.exported.Format != "sbase" {
		t.Fatalf("no format is sbase: status %d, %+v", w.Code, svc.exported)
	}
}
