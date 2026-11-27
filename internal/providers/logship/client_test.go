package logship

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeLogService records the pushes it gets and answers with the next status of its script (202 after that)
type fakeLogService struct {
	mu      sync.Mutex
	script  []int
	pushes  [][]map[string]any
	headers []http.Header
	paths   []string
}

func (f *fakeLogService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var req struct {
		Logs []map[string]any `json:"logs"`
	}
	_ = json.Unmarshal(body, &req)
	f.mu.Lock()
	status := http.StatusAccepted
	if len(f.script) > 0 {
		status, f.script = f.script[0], f.script[1:]
	}
	f.pushes = append(f.pushes, req.Logs)
	f.headers = append(f.headers, r.Header.Clone())
	f.paths = append(f.paths, r.URL.Path)
	f.mu.Unlock()
	w.WriteHeader(status)
}

func (f *fakeLogService) pushCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.pushes)
}

func (f *fakeLogService) entryCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, p := range f.pushes {
		n += len(p)
	}
	return n
}

type drops struct {
	mu    sync.Mutex
	count int
	why   []string
}

func (d *drops) on(n int, why string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.count += n
	d.why = append(d.why, why)
}

func (d *drops) total() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.count
}

func entry(msg string) Entry {
	return Entry{Timestamp: time.Now(), Level: "info", Message: msg, Service: "sereni-base",
		Labels: map[string]string{"env": "test"}, Fields: map[string]any{"status": 200}}
}

func newTestClient(t *testing.T, svc *fakeLogService, mutate func(*Config)) (*Client, *drops) {
	t.Helper()
	srv := httptest.NewServer(svc)
	t.Cleanup(srv.Close)
	d := &drops{}
	cfg := Config{URL: srv.URL + "/api/v1", Backoff: []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond},
		FlushInterval: time.Hour, WarnEvery: -1, OnDrop: d.on}
	if mutate != nil {
		mutate(&cfg)
	}
	c := New(cfg)
	c.Start()
	return c, d
}

func shutdown(t *testing.T, c *Client) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestBatchesBySizeAndShutdownFlushesTheRest(t *testing.T) {
	svc := &fakeLogService{}
	c, _ := newTestClient(t, svc, func(cfg *Config) { cfg.BatchSize = 3 })
	for i := 0; i < 7; i++ {
		c.Enqueue(entry("e"))
	}
	waitFor(t, "two full batches", func() bool { return svc.pushCount() >= 2 })
	shutdown(t, c)
	if svc.pushCount() != 3 || svc.entryCount() != 7 {
		t.Errorf("pushes = %d, entries = %d, want 3 and 7", svc.pushCount(), svc.entryCount())
	}
	if got := c.Stats().Sent; got != 7 {
		t.Errorf("sent = %d", got)
	}
}

func TestBatchesByTime(t *testing.T) {
	svc := &fakeLogService{}
	c, _ := newTestClient(t, svc, func(cfg *Config) { cfg.FlushInterval = 20 * time.Millisecond })
	c.Enqueue(entry("a"))
	waitFor(t, "timed flush", func() bool { return svc.entryCount() == 1 })
	shutdown(t, c)
}

func TestBatchesByBytes(t *testing.T) {
	svc := &fakeLogService{}
	c, _ := newTestClient(t, svc, func(cfg *Config) { cfg.MaxBatchBytes = 600 })
	big := entry(strings.Repeat("x", 300))
	for i := 0; i < 4; i++ {
		c.Enqueue(big)
	}
	shutdown(t, c)
	if svc.entryCount() != 4 || svc.pushCount() < 2 {
		t.Errorf("pushes = %d, entries = %d, want 4 entries in 2 or more pushes", svc.pushCount(), svc.entryCount())
	}
}

func TestRequestFormat(t *testing.T) {
	svc := &fakeLogService{}
	c, _ := newTestClient(t, svc, func(cfg *Config) { cfg.APIKey = "k1" })
	c.Enqueue(entry("hello"))
	shutdown(t, c)
	if svc.paths[0] != "/api/v1/logs" {
		t.Errorf("path = %q", svc.paths[0])
	}
	if svc.headers[0].Get("X-API-Key") != "k1" || svc.headers[0].Get("Content-Type") != "application/json" {
		t.Errorf("headers = %v", svc.headers[0])
	}
	e := svc.pushes[0][0]
	if e["message"] != "hello" || e["service"] != "sereni-base" || e["level"] != "info" || e["timestamp"] == nil {
		t.Errorf("entry = %v", e)
	}
	if e["fields"].(map[string]any)["status"] != float64(200) || e["labels"].(map[string]any)["env"] != "test" {
		t.Errorf("fields or labels = %v", e)
	}
}

func TestNoAPIKeyHeaderWhenUnset(t *testing.T) {
	svc := &fakeLogService{}
	c, _ := newTestClient(t, svc, nil)
	c.Enqueue(entry("a"))
	shutdown(t, c)
	if _, ok := svc.headers[0]["X-Api-Key"]; ok {
		t.Error("X-API-Key must not be sent when no key is set")
	}
}

func TestRetriesOn502ThenSucceeds(t *testing.T) {
	svc := &fakeLogService{script: []int{502, 502}}
	c, d := newTestClient(t, svc, nil)
	c.Enqueue(entry("a"))
	shutdown(t, c)
	if svc.pushCount() != 3 || c.Stats().Sent != 1 || d.total() != 0 {
		t.Errorf("pushes = %d, sent = %d, dropped = %d", svc.pushCount(), c.Stats().Sent, d.total())
	}
}

func TestDropsAfterRetriesUsedUp(t *testing.T) {
	svc := &fakeLogService{script: []int{502, 502, 502, 502, 502}}
	c, d := newTestClient(t, svc, nil)
	c.Enqueue(entry("a"))
	c.Enqueue(entry("b"))
	shutdown(t, c)
	if svc.pushCount() != 4 { // the first attempt and 3 retries
		t.Errorf("pushes = %d, want 4", svc.pushCount())
	}
	if d.total() != 2 || c.Stats().Dropped != 2 || c.Stats().Sent != 0 {
		t.Errorf("dropped = %d / %d, sent = %d", d.total(), c.Stats().Dropped, c.Stats().Sent)
	}
	if !strings.Contains(d.why[0], "retries used up") {
		t.Errorf("reason = %q", d.why[0])
	}
}

func TestNeverRetriesFinalAnswers(t *testing.T) {
	for _, status := range []int{400, 413, 422, 500} {
		svc := &fakeLogService{script: []int{status}}
		c, d := newTestClient(t, svc, nil)
		c.Enqueue(entry("a"))
		shutdown(t, c)
		if svc.pushCount() != 1 {
			t.Errorf("%d: pushes = %d, want 1 (no retry)", status, svc.pushCount())
		}
		if d.total() != 1 || c.Stats().Dropped != 1 {
			t.Errorf("%d: dropped = %d", status, d.total())
		}
	}
}

func TestRetriesNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // nothing listens any more
	d := &drops{}
	c := New(Config{URL: url, Backoff: []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond},
		FlushInterval: time.Hour, WarnEvery: -1, OnDrop: d.on})
	c.Start()
	c.Enqueue(entry("a"))
	shutdown(t, c)
	if d.total() != 1 || !strings.Contains(d.why[0], "retries used up") {
		t.Errorf("dropped = %d, reasons = %v", d.total(), d.why)
	}
}

func TestQueueFullDropsWithoutBlocking(t *testing.T) {
	d := &drops{}
	c := New(Config{URL: "http://127.0.0.1:1", QueueSize: 2, WarnEvery: -1, OnDrop: d.on}) // worker not started
	start := time.Now()
	accepted := 0
	for i := 0; i < 5; i++ {
		if c.Enqueue(entry("a")) {
			accepted++
		}
	}
	if accepted != 2 || d.total() != 3 || c.Stats().Dropped != 3 || c.Stats().Queued != 2 {
		t.Errorf("accepted = %d, dropped = %d, stats = %+v", accepted, d.total(), c.Stats())
	}
	if time.Since(start) > time.Second {
		t.Error("Enqueue blocked")
	}
}

func TestOversizeEntryIsDroppedNotSent(t *testing.T) {
	svc := &fakeLogService{}
	c, d := newTestClient(t, svc, func(cfg *Config) { cfg.MaxEntryBytes = 500 })
	c.Enqueue(entry(strings.Repeat("x", 2000)))
	c.Enqueue(entry("small"))
	shutdown(t, c)
	if svc.entryCount() != 1 || d.total() != 1 || !strings.Contains(d.why[0], "limit") {
		t.Errorf("entries = %d, dropped = %d, reasons = %v", svc.entryCount(), d.total(), d.why)
	}
}

func TestUnencodableEntryIsDropped(t *testing.T) {
	svc := &fakeLogService{}
	c, d := newTestClient(t, svc, nil)
	bad := entry("a")
	bad.Fields = func() {} // json cannot encode a func
	c.Enqueue(bad)
	c.Enqueue(entry("ok"))
	shutdown(t, c)
	if svc.entryCount() != 1 || d.total() != 1 {
		t.Errorf("entries = %d, dropped = %d", svc.entryCount(), d.total())
	}
}

func TestDropWarningsAreRateLimited(t *testing.T) {
	d := &drops{}
	c := New(Config{URL: "http://127.0.0.1:1", QueueSize: 1, WarnEvery: time.Hour, OnDrop: d.on})
	c.Enqueue(entry("a")) // fills the queue
	for i := 0; i < 5; i++ {
		c.Enqueue(entry("b"))
	}
	d.mu.Lock()
	calls := len(d.why)
	d.mu.Unlock()
	if calls != 1 || c.Stats().Dropped != 5 {
		t.Errorf("warnings = %d, dropped = %d, want 1 warning and 5 drops", calls, c.Stats().Dropped)
	}
}

func TestEnqueueAfterShutdownIsRefused(t *testing.T) {
	svc := &fakeLogService{}
	c, _ := newTestClient(t, svc, nil)
	shutdown(t, c)
	if c.Enqueue(entry("late")) {
		t.Error("an entry was accepted after shutdown")
	}
}

func TestShutdownBeforeStartAndTwice(t *testing.T) {
	c := New(Config{URL: "http://127.0.0.1:1"})
	if err := c.Shutdown(context.Background()); err != nil {
		t.Error(err)
	}
	svc := &fakeLogService{}
	c2, _ := newTestClient(t, svc, nil)
	shutdown(t, c2)
	shutdown(t, c2)
}

func TestShutdownDeadlineEndsAHangingSend(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-block }))
	defer srv.Close()
	defer close(block)
	d := &drops{}
	c := New(Config{URL: srv.URL, FlushInterval: time.Hour, WarnEvery: -1, OnDrop: d.on})
	c.Start()
	c.Enqueue(entry("a"))
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_ = c.Shutdown(ctx)
	if time.Since(start) > 2*time.Second {
		t.Error("shutdown ignored its deadline")
	}
}

func TestBatchSizeIsCappedAt1000(t *testing.T) {
	if c := New(Config{URL: "x", BatchSize: 5000}); c.cfg.BatchSize != 1000 {
		t.Errorf("batch size = %d", c.cfg.BatchSize)
	}
	if c := New(Config{URL: "x"}); c.cfg.BatchSize != 500 || c.cfg.QueueSize != 10000 || c.cfg.FlushInterval != 2*time.Second {
		t.Errorf("defaults = %+v", c.cfg)
	}
}
