package platform

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aptlogica/sereni-base/internal/middleware"
	"github.com/aptlogica/sereni-base/internal/providers/logship"
	"github.com/gin-gonic/gin"
)

func TestBodyObjectJSONObjectIsMasked(t *testing.T) {
	p := newBodyPolicy(BodiesAll, 0, []string{"card_number"})
	b := &bodyBuffer{limit: 1024}
	b.write([]byte(`{"name":"John","password":"x","nested":{"Token":"t","list":[{"api-key":"k"}]},"card_number":"1","n":12.50}`))
	obj, cut := p.object(b, "application/json; charset=utf-8")
	if cut {
		t.Fatal("not cut")
	}
	if obj["name"] != "John" || obj["password"] != "***" || obj["card_number"] != "***" {
		t.Errorf("obj = %v", obj)
	}
	nested := obj["nested"].(map[string]any)
	if nested["Token"] != "***" || nested["list"].([]any)[0].(map[string]any)["api-key"] != "***" {
		t.Errorf("nested = %v", nested)
	}
	if obj["n"].(interface{ String() string }).String() != "12.50" {
		t.Errorf("number lost its form: %v", obj["n"])
	}
}

func TestBodyObjectArrayIsWrapped(t *testing.T) {
	p := newBodyPolicy(BodiesAll, 0, nil)
	b := &bodyBuffer{limit: 1024}
	b.write([]byte(`[{"a":1},{"password":"x"}]`))
	obj, _ := p.object(b, "application/json")
	items, ok := obj["_json"].([]any)
	if !ok || len(items) != 2 || items[1].(map[string]any)["password"] != "***" {
		t.Errorf("obj = %v", obj)
	}
}

func TestBodyObjectTruncatedIsRawAndMasked(t *testing.T) {
	p := newBodyPolicy(BodiesAll, 60, nil)
	b := &bodyBuffer{limit: 60}
	body := `{"name":"John","password":"hunter2","notes":"` + strings.Repeat("x", 200) + `"}`
	b.write([]byte(body))
	obj, cut := p.object(b, "application/json")
	if !cut || obj["_truncated"] != true || obj["_size"] != int64(len(body)) {
		t.Fatalf("obj = %v, cut = %v", obj, cut)
	}
	raw := obj["_raw"].(string)
	if strings.Contains(raw, "hunter2") || !strings.Contains(raw, `"password":"***"`) {
		t.Errorf("raw = %q", raw)
	}
	if len(raw) > 60+20 {
		t.Errorf("raw is %d bytes, want it capped", len(raw))
	}
}

func TestBodyObjectNonJSON(t *testing.T) {
	p := newBodyPolicy(BodiesAll, 0, nil)
	b := &bodyBuffer{limit: 1024}
	b.write([]byte("name,email\nJohn,j@x.com\n"))
	obj, cut := p.object(b, "text/csv")
	if cut || obj["_raw"] != "name,email\nJohn,j@x.com\n" || obj["_size"] != int64(24) {
		t.Errorf("obj = %v", obj)
	}
	if _, has := obj["_truncated"]; has {
		t.Error("_truncated must be left out when the body was not cut")
	}
}

func TestBodyObjectInvalidJSONIsRaw(t *testing.T) {
	p := newBodyPolicy(BodiesAll, 0, nil)
	b := &bodyBuffer{limit: 1024}
	b.write([]byte(`{"a":1} trailing`))
	obj, _ := p.object(b, "application/json")
	if obj["_raw"] == nil {
		t.Errorf("obj = %v", obj)
	}
}

func TestBodyObjectEmptyIsNil(t *testing.T) {
	p := newBodyPolicy(BodiesAll, 0, nil)
	if obj, _ := p.object(&bodyBuffer{limit: 10}, "application/json"); obj != nil {
		t.Errorf("obj = %v", obj)
	}
	if obj, _ := p.object(nil, "application/json"); obj != nil {
		t.Errorf("obj = %v", obj)
	}
}

func TestBodyObjectInvalidUTF8(t *testing.T) {
	p := newBodyPolicy(BodiesAll, 0, nil)
	b := &bodyBuffer{limit: 100}
	b.write([]byte{'a', 0xff, 'b'})
	obj, _ := p.object(b, "application/octet-stream")
	if obj["_raw"] != "a�b" {
		t.Errorf("raw = %q", obj["_raw"])
	}
}

func TestNewBodyPolicy(t *testing.T) {
	if newBodyPolicy(BodiesOff, 0, nil) != nil || newBodyPolicy("nonsense", 0, nil) != nil {
		t.Error("off and unknown levels capture nothing")
	}
	if p := newBodyPolicy(BodiesErrors, 0, nil); p.maxBytes != DefaultLogBodyBytes {
		t.Errorf("default cap = %d", p.maxBytes)
	}
	if p := newBodyPolicy(BodiesErrors, 1<<20, nil); p.maxBytes != MaxLogBodyBytes {
		t.Errorf("cap = %d, want it limited to %d", p.maxBytes, MaxLogBodyBytes)
	}
	e, a := newBodyPolicy(BodiesErrors, 0, nil), newBodyPolicy(BodiesAll, 0, nil)
	if e.wants(200) || !e.wants(400) || !e.wants(500) || !a.wants(200) {
		t.Error("wants")
	}
}

func TestBodyBufferKeepsOnlyTheLimit(t *testing.T) {
	b := &bodyBuffer{limit: 5}
	b.write([]byte("abc"))
	b.write([]byte("defgh"))
	b.write([]byte("ij"))
	if string(b.buf) != "abcde" || b.total != 10 {
		t.Errorf("buf = %q, total = %d", b.buf, b.total)
	}
}

// ---- Through the middleware ----

type bodyRouter struct {
	engine      *gin.Engine
	sink        *fakeSink
	handlerBody string // what the handler read from the request
}

func newBodyRouter(level string, maxBytes int) *bodyRouter {
	gin.SetMode(gin.TestMode)
	br := &bodyRouter{sink: &fakeSink{}}
	a := newTestAPI(br.sink, true).WithLogBodies(level, maxBytes, nil)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(middleware.RequestIDKey, "req_test"); c.Next() })
	bs := r.Group("/api/v1/:bs", a.accessLog())
	bs.POST("/tables/:table/records", func(c *gin.Context) {
		data, _ := io.ReadAll(c.Request.Body)
		br.handlerBody = string(data)
		if c.Query("fail") != "" {
			respondError(c, NewError(CodeValidationFailed, "Some fields have invalid values. Nothing was saved.",
				ErrorDetail{Field: "Price", Issue: "Price must be a number."}))
			return
		}
		respond(c, http.StatusCreated, gin.H{"id": "r_1", "created": true, "token": "abc"})
	})
	bs.GET("/tables/:table/records", func(c *gin.Context) {
		respondList(c, []gin.H{{"id": 1}, {"id": 2}, {"id": 3}}, "", 50, nil)
	})
	bs.GET("/openapi.json", func(c *gin.Context) { respondError(c, notFound("x")) })
	bs.GET("/docs", func(c *gin.Context) { respondError(c, notFound("x")) })
	br.engine = r
	return br
}

func (br *bodyRouter) post(target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	br.engine.ServeHTTP(w, req)
	return w
}

const createURL = "/api/v1/bs_abcdefghij/tables/orders/records"

func TestBodiesAllCapturesBothAndHandlerStillReadsTheBody(t *testing.T) {
	br := newBodyRouter(BodiesAll, 0)
	body := `{"name":"John","email":"j@x.com","password":"p"}`
	w := br.post(createURL, body)
	if w.Code != http.StatusCreated || br.handlerBody != body {
		t.Fatalf("status = %d, handler read %q", w.Code, br.handlerBody)
	}
	if len(br.sink.entries) != 1 {
		t.Fatalf("entries = %d", len(br.sink.entries))
	}
	f := br.sink.entries[0].Fields
	if f.Request["name"] != "John" || f.Request["password"] != "***" {
		t.Errorf("request = %v", f.Request)
	}
	data := f.Response["data"].(map[string]any)
	if data["id"] != "r_1" || data["token"] != "***" {
		t.Errorf("response = %v", f.Response)
	}
	if f.RequestTruncated || f.ResponseTruncated {
		t.Error("nothing was cut")
	}
}

func TestBodiesErrorsOnlyForFailedCalls(t *testing.T) {
	br := newBodyRouter(BodiesErrors, 0)
	br.post(createURL, `{"name":"John"}`)
	if f := br.sink.entries[0].Fields; f.Request != nil || f.Response != nil {
		t.Errorf("a success must carry no bodies: %+v", f)
	}
	br.post(createURL+"?fail=1", `{"Price":"abc"}`)
	f := br.sink.entries[1].Fields
	if f.Request["Price"] != "abc" {
		t.Errorf("request = %v", f.Request)
	}
	errBody := f.Response["error"].(map[string]any)
	if errBody["code"] != "validation_failed" || f.Status != 422 {
		t.Errorf("response = %v", f.Response)
	}
}

func TestBodiesOffOrUnsetCaptureNothing(t *testing.T) {
	for _, level := range []string{BodiesOff, ""} {
		br := newBodyRouter(level, 0)
		br.post(createURL+"?fail=1", `{"a":1}`)
		if f := br.sink.entries[0].Fields; f.Request != nil || f.Response != nil {
			t.Errorf("level %q: %+v", level, f)
		}
	}
}

func TestBodiesTruncatedFlagsAndHandlerSeesTheWholeBody(t *testing.T) {
	br := newBodyRouter(BodiesAll, 40)
	body := `{"notes":"` + strings.Repeat("x", 500) + `"}`
	br.post(createURL, body)
	if br.handlerBody != body {
		t.Fatal("the handler must read the whole body")
	}
	f := br.sink.entries[0].Fields
	if !f.RequestTruncated || f.Request["_truncated"] != true || f.Request["_size"] != int64(len(body)) {
		t.Errorf("request = %v, flag = %v", f.Request, f.RequestTruncated)
	}
	if !f.ResponseTruncated {
		t.Errorf("response of %d bytes under a 40 byte cap should be cut: %v", len(f.Response), f.Response)
	}
}

func TestListResponseStoresOnlyRecordCount(t *testing.T) {
	br := newBodyRouter(BodiesAll, 0)
	w := httptest.NewRecorder()
	br.engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, createURL, nil))
	f := br.sink.entries[0].Fields
	if f.RecordCount == nil || *f.RecordCount != 3 || f.Response != nil {
		t.Errorf("record_count = %v, response = %v", f.RecordCount, f.Response)
	}
	if !strings.Contains(w.Body.String(), `"id":3`) {
		t.Error("the client must still get the list")
	}
}

func TestDocsRoutesAreNotLogged(t *testing.T) {
	br := newBodyRouter(BodiesAll, 0)
	for _, p := range []string{"/openapi.json", "/docs"} {
		w := httptest.NewRecorder()
		br.engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/bs_abcdefghij"+p, nil))
		if w.Code != http.StatusNotFound {
			t.Errorf("%s: status = %d", p, w.Code)
		}
	}
	if len(br.sink.entries) != 0 {
		t.Errorf("entries = %d, want none", len(br.sink.entries))
	}
}

func TestBodylessRequestHasNoRequestField(t *testing.T) {
	br := newBodyRouter(BodiesAll, 0)
	br.engine.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, createURL, nil))
	if br.sink.entries[0].Fields.Request != nil {
		t.Error("request must be left out when there is no body")
	}
}

// ---- Sinks ----

type fakeShipper struct{ got []logship.Entry }

func (f *fakeShipper) Enqueue(e logship.Entry) bool { f.got = append(f.got, e); return true }

func TestLogServiceSinkConvertsEntry(t *testing.T) {
	fs := &fakeShipper{}
	ts := time.Now()
	LogServiceSink{Shipper: fs}.Write(LogEntry{Timestamp: ts, Level: "warn", Message: "m", Service: "sereni-base",
		Labels: map[string]string{"env": "x"}, Fields: LogFields{Event: eventClientError, Status: 400}})
	if len(fs.got) != 1 {
		t.Fatalf("got %d", len(fs.got))
	}
	e := fs.got[0]
	if !e.Timestamp.Equal(ts) || e.Level != "warn" || e.Message != "m" || e.Service != "sereni-base" ||
		e.Labels["env"] != "x" || e.Fields.(LogFields).Status != 400 {
		t.Errorf("entry = %+v", e)
	}
}

func TestMultiSinkWritesAllEvenIfOnePanics(t *testing.T) {
	a, b := &fakeSink{}, &fakeSink{}
	MultiSink{a, panicSink{}, b}.Write(LogEntry{Message: "m"})
	if len(a.entries) != 1 || len(b.entries) != 1 {
		t.Errorf("a = %d, b = %d", len(a.entries), len(b.entries))
	}
}

func TestEntryJSONMatchesTheLogServiceFormat(t *testing.T) {
	// marshal through logship exactly as the worker does
	e := LogEntry{Timestamp: time.Now(), Level: "info", Message: "m", Service: "sereni-base",
		Labels: map[string]string{"env": "dev"},
		Fields: LogFields{Event: eventRequest, API: "/a", Path: "/a", Method: "GET", BaseID: "bs_a", Status: 200, RequestID: "r"}}
	var sent logship.Entry
	LogServiceSink{Shipper: shipperFunc(func(x logship.Entry) bool { sent = x; return true })}.Write(e)
	b, err := json.Marshal(sent)
	if err != nil {
		t.Fatal(err)
	}
	raw := string(b)
	for _, want := range []string{`"service":"sereni-base"`, `"labels":{"env":"dev"}`, `"status":200`, `"duration_ms":0`, `"event":"log.api.request"`} {
		if !strings.Contains(raw, want) {
			t.Errorf("%s missing from %s", want, raw)
		}
	}
	for _, no := range []string{`"key"`, `"request"`, `"response"`, `"record_count"`, `"table"`} {
		if strings.Contains(raw, no) {
			t.Errorf("%s must be left out: %s", no, raw)
		}
	}
}

type shipperFunc func(logship.Entry) bool

func (f shipperFunc) Enqueue(e logship.Entry) bool { return f(e) }
