package platform

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestEmbeddedMessagesLoad(t *testing.T) {
	if _, err := loadMessages(enUSMessages); err != nil {
		t.Fatal(err)
	}
}

func TestLogKeysMatchTheMessageFile(t *testing.T) {
	inFile := map[string]bool{}
	for key, m := range defaultMessages.messages {
		if !strings.HasPrefix(key, "log.") {
			continue
		}
		if m.Type != messageTypeLog {
			t.Errorf("%s: type = %q, want log", key, m.Type)
		}
		if m.Phase == 0 {
			inFile[key] = true
		}
	}
	used := map[string]bool{}
	for _, key := range usedLogEvents {
		used[key] = true
		if _, ok := defaultMessages.get(key); !ok {
			t.Errorf("%s is used in code but missing from the message file", key)
		}
	}
	for key := range inFile {
		if !used[key] {
			t.Errorf("%s is in the message file but not used in code", key)
		}
	}
}

// Every public error code has a general message and a general suggestion, so no failed request is logged
// without a note next to it.
func TestEveryErrorCodeHasAMessageAndSuggestion(t *testing.T) {
	for code := range errorStatus {
		key := errorMessageKey(code)
		m, ok := defaultMessages.get(key)
		if !ok {
			t.Errorf("%s: missing from the message file", key)
			continue
		}
		if m.Type != messageTypeError || m.Code != string(code) {
			t.Errorf("%s: type %q code %q", key, m.Type, m.Code)
		}
		if m.Action == "" {
			t.Errorf("%s: no action", key)
		}
	}
	if !defaultMessages.has(errorKeyUnknownBase) {
		t.Errorf("%s: missing from the message file", errorKeyUnknownBase)
	}
}

func TestLoadMessagesRejectsBadFiles(t *testing.T) {
	cases := map[string]string{
		"unknown type":    `{"locale":"en-US","messages":{"a.b":{"type":"x","message":"m"}}}`,
		"bad log level":   `{"locale":"en-US","messages":{"a.b":{"type":"log","level":"loud","message":"m"}}}`,
		"error no code":   `{"locale":"en-US","messages":{"a.b":{"type":"error","message":"m"}}}`,
		"empty message":   `{"locale":"en-US","messages":{"a.b":{"type":"log","level":"warn"}}}`,
		"undeclared":      `{"locale":"en-US","messages":{"a.b":{"type":"log","level":"warn","message":"{x}"}}}`,
		"unused param":    `{"locale":"en-US","messages":{"a.b":{"type":"log","level":"warn","message":"m","params":["x"]}}}`,
		"no locale":       `{"messages":{"a.b":{"type":"log","level":"warn","message":"m"}}}`,
		"not json":        `nope`,
		"no messages":     `{"locale":"en-US","messages":{}}`,
		"action mismatch": `{"locale":"en-US","messages":{"a.b":{"type":"error","code":"c","message":"m","action":"wait {s}"}}}`,
	}
	for name, data := range cases {
		if _, err := loadMessages([]byte(data)); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestRender(t *testing.T) {
	c, err := loadMessages([]byte(`{"locale":"en-US","messages":{"a.b":{"type":"log","level":"warn","message":"{m} {n}","params":["m","n"]}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := c.render("a.b", map[string]any{"m": "GET", "n": 7}); got != "GET 7" {
		t.Errorf("got %q", got)
	}
	if got := c.render("a.b", map[string]any{"m": "GET"}); got != "GET {n}" {
		t.Errorf("missing value: got %q", got)
	}
	if got := c.render("no.such.key", nil); got != "no.such.key" {
		t.Errorf("missing key: got %q", got)
	}
	long := strings.Repeat("x", 200)
	if got := c.render("a.b", map[string]any{"m": "a\nb\x00", "n": long}); got != "ab "+long[:maxParamLength]+"..." {
		t.Errorf("clean: got %q", got)
	}
}

func TestZerologSinkWritesFlatFields(t *testing.T) {
	var buf bytes.Buffer
	lg := zerolog.New(&buf)
	ZerologSink{Logger: &lg}.Write(LogEntry{
		Level:   "warn",
		Message: "GET /x returned 400",
		Fields: LogFields{Event: eventClientError, API: "/x", Path: "/x", Method: "GET", BaseID: "bs_a",
			Status: 400, DurationMS: 3, RequestID: "req_1", ErrorCode: "bad_request", Table: "t"},
	})
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"level": "warn", "message": "GET /x returned 400", "event": eventClientError,
		"api": "/x", "path": "/x", "method": "GET", "base_id": "bs_a", "status": float64(400),
		"duration_ms": float64(3), "request_id": "req_1", "error_code": "bad_request", "table": "t"}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %v, want %v", k, got[k], v)
		}
	}
	if _, ok := got["key"]; ok {
		t.Error("key must be left out when empty")
	}
}
