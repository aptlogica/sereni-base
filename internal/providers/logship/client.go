// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

// Package logship sends log entries to the log service (sereni-log) in the background.
//
// A request only does a non-blocking Enqueue into a buffered channel. One long-lived worker goroutine reads
// the channel, groups entries into batches and makes one POST per batch. The request path never waits for
// the log service: when the queue is full the entry is dropped, and a batch that cannot be delivered is
// dropped after the retry rules (api-platform-log-shipping-suggestion.md section 12).
//
// The package must not log its own failures through itself: drops are reported through Config.OnDrop.
package logship

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultQueueSize     = 10000
	defaultBatchSize     = 500
	maxBatchSize         = 1000 // the log service accepts 1 to 1000 entries per push
	defaultFlushInterval = 2 * time.Second
	defaultMaxBatchBytes = 4<<20 - 64<<10 // the log service refuses a push above 4 MB (413)
	defaultMaxEntryBytes = 32 << 10       // the log service refuses an entry above 32 KB (400)
	defaultTimeout       = 5 * time.Second
	defaultWarnEvery     = time.Minute
)

// defaultBackoff is the wait before each retry: 3 retries after the first attempt
var defaultBackoff = []time.Duration{200 * time.Millisecond, time.Second, 5 * time.Second}

// Entry is one log entry in the format the log service accepts. Fields is marshalled as the free-form
// `fields` object.
type Entry struct {
	Timestamp time.Time         `json:"timestamp"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Service   string            `json:"service"`
	Labels    map[string]string `json:"labels,omitempty"`
	Fields    any               `json:"fields,omitempty"`
}

// Config configures a Client. Zero values take the defaults.
type Config struct {
	URL           string          // log service URL including /api/v1; entries go to URL + "/logs"
	APIKey        string          // sent as X-API-Key when set
	QueueSize     int             // buffered entries before dropping (default 10000)
	BatchSize     int             // entries per push, capped at 1000 (default 500)
	FlushInterval time.Duration   // longest wait before a partial batch is sent (default 2s)
	MaxBatchBytes int             // flush early when a batch reaches this size (default just under 4 MB)
	MaxEntryBytes int             // an entry above this is dropped before sending (default 32 KB)
	Timeout       time.Duration   // per HTTP call (default 5s)
	Backoff       []time.Duration // wait before each retry; nil takes 200ms, 1s, 5s
	WarnEvery     time.Duration   // OnDrop is called at most this often (default 1 minute; negative: every drop)
	HTTPClient    *http.Client    // nil builds one with Timeout

	// OnDrop is told how many entries were dropped since the last call and why. It is called from the worker
	// or from a request, so it must be quick. It must not send through this Client.
	OnDrop func(count int, reason string)
}

// Stats are the counters of a Client
type Stats struct {
	Queued  int   // entries waiting in the queue now
	Sent    int64 // entries the log service accepted
	Dropped int64 // entries dropped: queue full, refused, too large or retries used up
}

// Client is the queue, the worker and the HTTP client
type Client struct {
	cfg   Config
	queue chan Entry
	url   string

	started  atomic.Bool
	closed   atomic.Bool
	stopOnce sync.Once
	stop     chan struct{}
	done     chan struct{}
	finalCtx context.Context // deadline of the last flush; set before stop is closed

	sent    atomic.Int64
	dropped atomic.Int64

	mu         sync.Mutex // guards the drop warning state
	pending    int
	lastReason string
	lastWarn   time.Time
}

// New returns a Client. Call Start to run the worker.
func New(cfg Config) *Client {
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = defaultQueueSize
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = defaultBatchSize
	}
	if cfg.BatchSize > maxBatchSize {
		cfg.BatchSize = maxBatchSize
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = defaultFlushInterval
	}
	if cfg.MaxBatchBytes <= 0 {
		cfg.MaxBatchBytes = defaultMaxBatchBytes
	}
	if cfg.MaxEntryBytes <= 0 {
		cfg.MaxEntryBytes = defaultMaxEntryBytes
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.Backoff == nil {
		cfg.Backoff = defaultBackoff
	}
	if cfg.WarnEvery == 0 {
		cfg.WarnEvery = defaultWarnEvery
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: cfg.Timeout}
	}
	return &Client{
		cfg:   cfg,
		queue: make(chan Entry, cfg.QueueSize),
		url:   strings.TrimRight(cfg.URL, "/") + "/logs",
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}
}

// Start runs the worker goroutine. It does nothing when called again.
func (c *Client) Start() {
	if c.started.CompareAndSwap(false, true) {
		go c.run()
	}
}

// Enqueue adds an entry without blocking and says whether it was accepted. A full queue or a Client that
// is shutting down drops the entry. It is safe to call from many goroutines and from request handlers.
func (c *Client) Enqueue(e Entry) bool {
	if c.closed.Load() {
		return false
	}
	select {
	case c.queue <- e:
		return true
	default:
		c.drop(1, "queue full")
		return false
	}
}

// Stats returns the current counters
func (c *Client) Stats() Stats {
	return Stats{Queued: len(c.queue), Sent: c.sent.Load(), Dropped: c.dropped.Load()}
}

// Shutdown stops taking entries, sends what is queued and waits for the worker, until ctx is done. Entries
// still unsent when ctx ends are dropped.
func (c *Client) Shutdown(ctx context.Context) error {
	if !c.started.Load() {
		return nil
	}
	c.stopOnce.Do(func() {
		c.closed.Store(true)
		c.finalCtx = ctx
		close(c.stop)
	})
	select {
	case <-c.done:
		c.reportPending()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ---- Worker ----

// batch holds the encoded entries of the next push
type batch struct {
	items [][]byte
	bytes int
}

func (b *batch) reset() { b.items, b.bytes = nil, 0 }

func (c *Client) run() {
	defer close(c.done)
	var b batch
	ticker := time.NewTicker(c.cfg.FlushInterval)
	defer ticker.Stop()
	for {
		select {
		case e := <-c.queue:
			c.add(context.Background(), &b, e)
		case <-ticker.C:
			c.flush(context.Background(), &b)
		case <-c.stop:
			for { // send everything still queued
				select {
				case e := <-c.queue:
					c.add(c.finalCtx, &b, e)
					continue
				default:
				}
				break
			}
			c.flush(c.finalCtx, &b)
			return
		}
	}
}

// add encodes e and puts it in the batch, flushing first when the batch would get too big and after when it
// is full. An entry that cannot be encoded or is above the entry limit is dropped here, because the log
// service would refuse the whole batch.
func (c *Client) add(ctx context.Context, b *batch, e Entry) {
	raw, err := json.Marshal(e)
	if err != nil {
		c.drop(1, "entry could not be encoded: "+err.Error())
		return
	}
	if len(raw) > c.cfg.MaxEntryBytes {
		c.drop(1, fmt.Sprintf("entry is %d bytes, above the %d byte limit", len(raw), c.cfg.MaxEntryBytes))
		return
	}
	if len(b.items) > 0 && b.bytes+len(raw) > c.cfg.MaxBatchBytes {
		c.flush(ctx, b)
	}
	b.items = append(b.items, raw)
	b.bytes += len(raw)
	if len(b.items) >= c.cfg.BatchSize {
		c.flush(ctx, b)
	}
}

// flush sends the batch, if it has entries, and empties it
func (c *Client) flush(ctx context.Context, b *batch) {
	if len(b.items) == 0 {
		return
	}
	payload := encodeBatch(b.items)
	n := len(b.items)
	b.reset()
	c.deliver(ctx, payload, n)
}

// encodeBatch builds {"logs":[...]} from already encoded entries
func encodeBatch(items [][]byte) []byte {
	var buf bytes.Buffer
	buf.WriteString(`{"logs":[`)
	for i, raw := range items {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.Write(raw)
	}
	buf.WriteString(`]}`)
	return buf.Bytes()
}

// deliver pushes one batch. Only a network error and 502 are retried (with Config.Backoff); any other
// answer other than success is final: the batch is dropped and counted.
func (c *Client) deliver(ctx context.Context, payload []byte, n int) {
	for attempt := 0; ; attempt++ {
		status, err := c.post(ctx, payload)
		var why string
		switch {
		case err == nil && status >= 200 && status < 300:
			c.sent.Add(int64(n))
			return
		case err == nil && status != http.StatusBadGateway:
			c.drop(n, fmt.Sprintf("log service answered %d", status))
			return
		case err != nil:
			why = err.Error()
		default:
			why = "log service answered 502"
		}
		if attempt >= len(c.cfg.Backoff) {
			c.drop(n, "retries used up: "+why)
			return
		}
		if !sleep(ctx, c.cfg.Backoff[attempt]) {
			c.drop(n, "stopped before a retry: "+why)
			return
		}
	}
}

// post makes one HTTP call and returns the status
func (c *Client) post(ctx context.Context, payload []byte) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("X-API-Key", c.cfg.APIKey)
	}
	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	return resp.StatusCode, nil
}

func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// ---- Drop reporting ----

// drop counts n lost entries and tells OnDrop, at most once per WarnEvery
func (c *Client) drop(n int, reason string) {
	c.dropped.Add(int64(n))
	if c.cfg.OnDrop == nil {
		return
	}
	c.mu.Lock()
	c.pending += n
	c.lastReason = reason
	if c.cfg.WarnEvery > 0 && time.Since(c.lastWarn) < c.cfg.WarnEvery {
		c.mu.Unlock()
		return
	}
	count, why := c.pending, c.lastReason
	c.pending, c.lastWarn = 0, time.Now()
	c.mu.Unlock()
	c.cfg.OnDrop(count, why)
}

// reportPending tells OnDrop about drops that were held back by the rate limit
func (c *Client) reportPending() {
	if c.cfg.OnDrop == nil {
		return
	}
	c.mu.Lock()
	count, why := c.pending, c.lastReason
	c.pending = 0
	c.mu.Unlock()
	if count > 0 {
		c.cfg.OnDrop(count, why)
	}
}
