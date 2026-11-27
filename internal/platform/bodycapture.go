// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// Body capture levels (PUBLIC_API_LOG_BODIES)
const (
	BodiesOff    = "off"    // no bodies
	BodiesErrors = "errors" // bodies of requests that end with status 400 or higher
	BodiesAll    = "all"    // every body
)

const (
	// DefaultLogBodyBytes is the cap of one captured body
	DefaultLogBodyBytes = 8 << 10
	// MaxLogBodyBytes is the highest cap: two bodies plus fields must stay under the log service's 32 KB entry limit
	MaxLogBodyBytes = 12 << 10

	redactedValue  = "***"
	ctxRecordCount = "platform.recordCount" // set by respondList: the size of the list answered
)

// builtinRedactFields are always masked; PUBLIC_API_LOG_REDACT_FIELDS adds to them
var builtinRedactFields = []string{"password", "token", "secret", "authorization", "api_key", "access_key"}

// bodyPolicy says which bodies are captured, how much of them, and what is masked
type bodyPolicy struct {
	level    string
	maxBytes int
	redactor *redactor
}

// newBodyPolicy returns the policy for level, or nil when no body is captured
func newBodyPolicy(level string, maxBytes int, redactFields []string) *bodyPolicy {
	if level != BodiesErrors && level != BodiesAll {
		return nil
	}
	if maxBytes <= 0 {
		maxBytes = DefaultLogBodyBytes
	}
	if maxBytes > MaxLogBodyBytes {
		maxBytes = MaxLogBodyBytes
	}
	return &bodyPolicy{level: level, maxBytes: maxBytes, redactor: newRedactor(redactFields)}
}

// wants says whether the bodies of a request that ended with status are kept
func (p *bodyPolicy) wants(status int) bool {
	return p.level == BodiesAll || status >= http.StatusBadRequest
}

// ---- Tapping the request and the response without reading ahead ----

// bodyBuffer keeps the first limit bytes that pass through a tap and counts all of them
type bodyBuffer struct {
	buf   []byte
	limit int
	total int64
}

func (b *bodyBuffer) write(p []byte) {
	b.total += int64(len(p))
	if room := b.limit - len(b.buf); room > 0 {
		if len(p) < room {
			room = len(p)
		}
		b.buf = append(b.buf, p[:room]...)
	}
}

// tapReader copies what the handler reads from the request body, up to the limit. It never reads ahead,
// so a large upload is not held in memory and the handler sees the same bytes as without it.
type tapReader struct {
	io.ReadCloser
	b *bodyBuffer
}

func (r *tapReader) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if n > 0 {
		r.b.write(p[:n])
	}
	return n, err
}

// tapWriter copies what the handler writes to the response, up to the limit
type tapWriter struct {
	gin.ResponseWriter
	b *bodyBuffer
}

func (w *tapWriter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	if n > 0 {
		w.b.write(p[:n])
	}
	return n, err
}

func (w *tapWriter) WriteString(s string) (int, error) {
	n, err := w.ResponseWriter.WriteString(s)
	if n > 0 {
		w.b.write([]byte(s[:n]))
	}
	return n, err
}

// bodyTap is the pair of taps of one request
type bodyTap struct {
	req, resp *bodyBuffer
	writer    *tapWriter
}

// start puts the taps on the request body and the response writer of c
func (p *bodyPolicy) start(c *gin.Context) *bodyTap {
	t := &bodyTap{req: &bodyBuffer{limit: p.maxBytes}, resp: &bodyBuffer{limit: p.maxBytes}}
	if c.Request.Body != nil && c.Request.Body != http.NoBody {
		c.Request.Body = &tapReader{ReadCloser: c.Request.Body, b: t.req}
	}
	t.writer = &tapWriter{ResponseWriter: c.Writer, b: t.resp}
	c.Writer = t.writer
	return t
}

// ---- Turning captured bytes into the fields.request / fields.response objects ----

// object returns the captured body as a JSON object, and whether it was cut. It returns nil when there was
// no body. The result is always an object so the field type never changes:
//   - a JSON object: as is, with secret fields masked
//   - another JSON value (an array of a bulk request): {"_json": value}
//   - a body cut at the cap, or not JSON: {"_raw": text, "_size": bytes, "_truncated": true when cut}
func (p *bodyPolicy) object(b *bodyBuffer, contentType string) (map[string]any, bool) {
	if b == nil || b.total == 0 {
		return nil, false
	}
	truncated := b.total > int64(len(b.buf))
	if !truncated && isJSONContentType(contentType) {
		dec := json.NewDecoder(bytes.NewReader(b.buf))
		dec.UseNumber()
		var v any
		if err := dec.Decode(&v); err == nil && !hasMoreJSON(dec) {
			v = p.redactor.mask(v)
			if obj, ok := v.(map[string]any); ok {
				return obj, false
			}
			return map[string]any{"_json": v}, false
		}
	}
	raw := p.redactor.maskRaw(strings.ToValidUTF8(string(b.buf), "�"))
	obj := map[string]any{"_raw": raw, "_size": b.total}
	if truncated {
		obj["_truncated"] = true
	}
	return obj, truncated
}

func hasMoreJSON(dec *json.Decoder) bool {
	_, err := dec.Token()
	return err != io.EOF
}

func isJSONContentType(contentType string) bool {
	mt, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	return mt == "application/json" || strings.HasSuffix(mt, "+json")
}

// ---- Redaction ----

// redactor masks the values of secret fields. Names are compared in lower case with - read as _.
type redactor struct {
	names map[string]bool
	raw   *regexp.Regexp // "name": "value" in text that is not valid JSON
}

func newRedactor(extra []string) *redactor {
	r := &redactor{names: map[string]bool{}}
	var quoted []string
	for _, name := range append(append([]string{}, builtinRedactFields...), extra...) {
		name = normalizeFieldName(name)
		if name == "" || r.names[name] {
			continue
		}
		r.names[name] = true
		quoted = append(quoted, regexp.QuoteMeta(name))
	}
	// "name" : "value" or "name" : 123 / true / null; the value of a cut body may stop anywhere
	r.raw = regexp.MustCompile(`(?i)("(?:` + strings.Join(quoted, "|") + `)"\s*:\s*)("(?:[^"\\]|\\.)*"?|[^,}\]\s]+)`)
	return r
}

func normalizeFieldName(s string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(s)), "-", "_")
}

// mask replaces the value of every secret field, at any depth
func (r *redactor) mask(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if r.names[normalizeFieldName(k)] {
				t[k] = redactedValue
			} else {
				t[k] = r.mask(val)
			}
		}
		return t
	case []any:
		for i, val := range t {
			t[i] = r.mask(val)
		}
		return t
	default:
		return v
	}
}

// maskRaw masks secret values in text that is not valid JSON, such as a body cut at the cap
func (r *redactor) maskRaw(s string) string {
	return r.raw.ReplaceAllString(s, `${1}"`+redactedValue+`"`)
}
