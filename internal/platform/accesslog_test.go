package platform

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aptlogica/sereni-base/internal/middleware"
	"github.com/gin-gonic/gin"
)

type fakeSink struct{ entries []LogEntry }

func (f *fakeSink) Write(e LogEntry) { f.entries = append(f.entries, e) }

// newLogRouter serves /api/v1/:bs/... through the access log. resolved says whether the test handlers
// behave as if resolveBase found the base.
func newLogRouter(a *API, resolved bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(middleware.RequestIDKey, "req_test"); c.Next() })
	bs := r.Group("/api/v1/:bs", a.accessLog(), func(c *gin.Context) {
		if resolved {
			c.Set(ctxBase, baseRef{})
		}
		c.Next()
	})
	bs.GET("/tables/:table/records", func(c *gin.Context) {
		switch c.Query("case") {
		case "400":
			respondError(c, badRequest("bad"))
		case "500":
			respondError(c, NewError(CodeInternalError, "boom"))
		case "429":
			respondError(c, NewError(CodeRateLimited, "slow down"))
		case "404":
			respondError(c, notFound("nope"))
		default:
			respond(c, http.StatusOK, gin.H{"ok": true})
		}
	})
	return r
}

func do(r *gin.Engine, target string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
	return w
}

func newTestAPI(sink LogSink, logAll ...bool) *API {
	a := &API{now: time.Now}
	if sink != nil {
		a.WithLogSink(sink, "test", len(logAll) > 0 && logAll[0])
	}
	return a
}

func TestAccessLogClientError(t *testing.T) {
	sink := &fakeSink{}
	r := newLogRouter(newTestAPI(sink), true)
	w := do(r, "/api/v1/bs_abcdefghij/tables/orders/records?case=400")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", w.Code)
	}
	if len(sink.entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(sink.entries))
	}
	e := sink.entries[0]
	f := e.Fields
	if e.Level != "warn" || e.Service != "sereni-base" || e.Labels["env"] != "test" {
		t.Errorf("entry head = %+v", e)
	}
	if e.Message != "GET /api/v1/:bs/tables/:table/records returned 400" {
		t.Errorf("message = %q", e.Message)
	}
	if f.Event != eventClientError || f.Status != 400 || f.ErrorCode != "bad_request" ||
		f.BaseID != "bs_abcdefghij" || f.Table != "orders" || f.RequestID != "req_test" ||
		f.API != "/api/v1/:bs/tables/:table/records" ||
		f.Path != "/api/v1/bs_abcdefghij/tables/orders/records" || f.Method != "GET" {
		t.Errorf("fields = %+v", f)
	}
}

func TestAccessLogServerError(t *testing.T) {
	sink := &fakeSink{}
	do(newLogRouter(newTestAPI(sink), true), "/api/v1/bs_abcdefghij/tables/orders/records?case=500")
	if len(sink.entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(sink.entries))
	}
	if e := sink.entries[0]; e.Level != "error" || e.Fields.Event != eventServerError || e.Fields.Status != 500 {
		t.Errorf("entry = %+v", e)
	}
}

func TestAccessLogSuccessIsNotLogged(t *testing.T) {
	sink := &fakeSink{}
	w := do(newLogRouter(newTestAPI(sink), true), "/api/v1/bs_abcdefghij/tables/orders/records")
	if w.Code != http.StatusOK || len(sink.entries) != 0 {
		t.Errorf("status = %d, entries = %d", w.Code, len(sink.entries))
	}
}

func TestAccessLogAllLogsSuccess(t *testing.T) {
	sink := &fakeSink{}
	w := do(newLogRouter(newTestAPI(sink, true), true), "/api/v1/bs_abcdefghij/tables/orders/records")
	if w.Code != http.StatusOK || len(sink.entries) != 1 {
		t.Fatalf("status = %d, entries = %d", w.Code, len(sink.entries))
	}
	e := sink.entries[0]
	if e.Level != "info" || e.Fields.Event != eventRequest || e.Fields.Status != 200 || e.Fields.ErrorCode != "" ||
		e.Message != "GET /api/v1/:bs/tables/:table/records returned 200" {
		t.Errorf("entry = %+v", e)
	}
}

func TestAccessLogAllStillLogsFailuresOnce(t *testing.T) {
	sink := &fakeSink{}
	do(newLogRouter(newTestAPI(sink, true), true), "/api/v1/bs_abcdefghij/tables/orders/records?case=500")
	if len(sink.entries) != 1 || sink.entries[0].Fields.Event != eventServerError {
		t.Errorf("entries = %+v", sink.entries)
	}
}

func TestAccessLogUnknownBase(t *testing.T) {
	sink := &fakeSink{}
	do(newLogRouter(newTestAPI(sink), false), "/api/v1/bs_zzzzzzzzzz/tables/orders/records?case=404")
	if len(sink.entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(sink.entries))
	}
	f := sink.entries[0].Fields
	if f.Event != eventUnknownBase || f.BaseID != "bs_zzzzzzzzzz" {
		t.Errorf("fields = %+v", f)
	}
}

func TestAccessLogNotFoundInResolvedBaseIsClientError(t *testing.T) {
	sink := &fakeSink{}
	do(newLogRouter(newTestAPI(sink), true), "/api/v1/bs_abcdefghij/tables/orders/records?case=404")
	if len(sink.entries) != 1 || sink.entries[0].Fields.Event != eventClientError {
		t.Errorf("entries = %+v", sink.entries)
	}
}

func TestAccessLogRateLimited(t *testing.T) {
	sink := &fakeSink{}
	do(newLogRouter(newTestAPI(sink), true), "/api/v1/bs_abcdefghij/tables/orders/records?case=429")
	if len(sink.entries) != 1 || sink.entries[0].Fields.Event != eventRateLimited ||
		sink.entries[0].Message != "GET /api/v1/:bs/tables/:table/records rate limited" {
		t.Errorf("entries = %+v", sink.entries)
	}
}

func TestAccessLogOff(t *testing.T) {
	w := do(newLogRouter(newTestAPI(nil), true), "/api/v1/bs_abcdefghij/tables/orders/records?case=400")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestAccessLogKeyOnlyWhenPresent(t *testing.T) {
	sink := &fakeSink{}
	a := newTestAPI(sink)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/:bs/x", a.accessLog(), func(c *gin.Context) {
		c.Set(ctxErrorKey, "records.field_not_number")
		respondError(c, NewError(CodeValidationFailed, "bad"))
	})
	do(r, "/api/v1/bs_abcdefghij/x")
	if len(sink.entries) != 1 || sink.entries[0].Fields.Key != "records.field_not_number" {
		t.Errorf("entries = %+v", sink.entries)
	}
}

func TestAccessLogCarriesMessageAndSuggestion(t *testing.T) {
	sink := &fakeSink{}
	r := newLogRouter(newTestAPI(sink), true)
	do(r, "/api/v1/bs_abcdefghij/tables/orders/records?case=404")
	do(r, "/api/v1/bs_abcdefghij/tables/orders/records") // success: none of the three
	if len(sink.entries) != 1 {
		t.Fatalf("entries = %d, want 1 (the 2xx is not logged without logAll)", len(sink.entries))
	}
	f := sink.entries[0].Fields
	if f.ErrorMessage != "nope" || f.Key != "error.not_found" || f.Friendly == "" || f.Suggestion == "" {
		t.Errorf("fields = %+v", f)
	}

	sink = &fakeSink{}
	r = newLogRouter(newTestAPI(sink, true), true)
	do(r, "/api/v1/bs_abcdefghij/tables/orders/records")
	if f := sink.entries[0].Fields; f.ErrorMessage != "" || f.Friendly != "" || f.Suggestion != "" {
		t.Errorf("a 200 must carry none of them: %+v", f)
	}
}

func TestAccessLogUnknownBaseHasItsOwnSuggestion(t *testing.T) {
	sink := &fakeSink{}
	r := newLogRouter(newTestAPI(sink), false) // base not resolved
	do(r, "/api/v1/bs_abcdefghij/tables/orders/records?case=404")
	f := sink.entries[0].Fields
	if f.Key != errorKeyUnknownBase || f.Suggestion != defaultMessages.renderAction(errorKeyUnknownBase, nil) {
		t.Errorf("fields = %+v", f)
	}
}

func TestErrorMessageIsCut(t *testing.T) {
	sink := &fakeSink{}
	a := newTestAPI(sink)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/:bs/x", a.accessLog(), func(c *gin.Context) {
		respondError(c, badRequest(strings.Repeat("x", 2000)+"\n"))
	})
	do(r, "/api/v1/bs_abcdefghij/x")
	if got := len([]rune(sink.entries[0].Fields.ErrorMessage)); got != maxErrorMessage+3 {
		t.Errorf("length = %d", got)
	}
}

type panicSink struct{}

func (panicSink) Write(LogEntry) { panic("sink failed") }

func TestAccessLogSinkPanicDoesNotFailRequest(t *testing.T) {
	w := do(newLogRouter(newTestAPI(panicSink{}), true), "/api/v1/bs_abcdefghij/tables/orders/records?case=400")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want the handler's 400", w.Code)
	}
}
