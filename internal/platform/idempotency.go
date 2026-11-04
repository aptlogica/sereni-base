// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package platform

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
	"github.com/gin-gonic/gin"
)

const (
	maxIdempotencyKeyLength = 255
	// idempotencyRetention is how long a key and its response are kept
	idempotencyRetention = "24 hours"
)

// captureWriter keeps a copy of the response so it can be replayed for a retry with the same key
type captureWriter struct {
	gin.ResponseWriter
	body bytes.Buffer
}

func (w *captureWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *captureWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

func (a *API) idempotencyTable() string {
	return tenant.APIIdempotencyKey{}.TableName(a.schema)
}

// idempotent makes a POST safe to retry with an Idempotency-Key header: the first request with a key runs and
// its response is stored; a retry with the same key and body gets that response again. The same key with a
// different body, or while the first request is still running, is a conflict.
func (a *API) idempotent() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			c.Next()
			return
		}
		if len(key) > maxIdempotencyKeyLength {
			respondError(c, badRequest(fmt.Sprintf("Idempotency-Key can be at most %d characters.", maxIdempotencyKeyLength)))
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes))
		if err != nil {
			respondError(c, NewError(CodePayloadTooLarge, "The request body is too large."))
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		sum := sha256.Sum256(append([]byte(c.Request.Method+" "+c.Request.URL.Path+"\n"), body...))
		hash := hex.EncodeToString(sum[:])
		ws := currentWorkspace(c)

		claimed, err := a.claimKey(ws.ID, key, hash)
		if err != nil {
			respondError(c, err)
			return
		}
		if !claimed {
			a.replay(c, ws.ID, key, hash)
			return
		}

		w := &captureWriter{ResponseWriter: c.Writer}
		c.Writer = w
		c.Next()

		status := w.Status()
		if status >= http.StatusInternalServerError {
			// Nothing was written, so a retry with the same key may run again
			if _, err := a.db.Exec(fmt.Sprintf(`DELETE FROM %s WHERE workspace_id = $1 AND "key" = $2`, a.idempotencyTable()), ws.ID, key); err != nil {
				logger.Get().Error().Err(err).Msg("failed to release idempotency key")
			}
			return
		}
		if _, err := a.db.Exec(fmt.Sprintf(`UPDATE %s SET status = $3, status_code = $4, response_body = $5 WHERE workspace_id = $1 AND "key" = $2`,
			a.idempotencyTable()), ws.ID, key, tenant.IdempotencyDone, status, w.body.String()); err != nil {
			logger.Get().Error().Err(err).Msg("failed to store idempotent response")
		}
	}
}

// claimKey stores the key as in progress; false when the workspace already used it. A key older than the
// retention period is treated as new, even before the cleanup removes it.
func (a *API) claimKey(workspaceID, key, hash string) (bool, error) {
	insert := fmt.Sprintf(`INSERT INTO %s (workspace_id, "key", request_hash, status) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`,
		a.idempotencyTable())
	for attempt := 0; attempt < 2; attempt++ {
		res, err := a.db.Exec(insert, workspaceID, key, hash, tenant.IdempotencyInProgress)
		if err != nil {
			return false, err
		}
		if n, _ := res.RowsAffected(); n == 1 {
			return true, nil
		}
		expired, err := a.db.Exec(fmt.Sprintf(`DELETE FROM %s WHERE workspace_id = $1 AND "key" = $2 AND created_time < CURRENT_TIMESTAMP - interval '%s'`,
			a.idempotencyTable(), idempotencyRetention), workspaceID, key)
		if err != nil {
			return false, err
		}
		if n, _ := expired.RowsAffected(); n == 0 {
			return false, nil
		}
	}
	return false, nil
}

// replay answers a retry: the stored response, or a conflict
func (a *API) replay(c *gin.Context, workspaceID, key, hash string) {
	var storedHash, status, body string
	var code int
	err := a.db.QueryRow(fmt.Sprintf(`SELECT request_hash, status, COALESCE(status_code, 0), COALESCE(response_body, '') FROM %s WHERE workspace_id = $1 AND "key" = $2`,
		a.idempotencyTable()), workspaceID, key).Scan(&storedHash, &status, &code, &body)
	if errors.Is(err, sql.ErrNoRows) {
		// Released by a failed first request between our insert and this read
		respondError(c, NewError(CodeConflict, "This Idempotency-Key is being used by another request. Try again."))
		return
	}
	if err != nil {
		respondError(c, err)
		return
	}
	if storedHash != hash {
		respondError(c, NewError(CodeConflict, "This Idempotency-Key was already used with a different request. Use a new key for a new request."))
		return
	}
	if status != tenant.IdempotencyDone || code == 0 {
		respondError(c, NewError(CodeConflict, "A request with this Idempotency-Key is still running. Try again shortly."))
		return
	}
	c.Header("Idempotent-Replayed", "true")
	c.Data(code, "application/json; charset=utf-8", withRequestID([]byte(body), requestID(c)))
	c.Abort()
}

// withRequestID sets meta.request_id in a stored response to this request's ID, so it matches X-Request-Id
func withRequestID(body []byte, id string) []byte {
	var env map[string]json.RawMessage
	if json.Unmarshal(body, &env) != nil {
		return body
	}
	var m map[string]interface{}
	if json.Unmarshal(env["meta"], &m) != nil || m == nil {
		m = map[string]interface{}{}
	}
	m["request_id"] = id
	raw, err := json.Marshal(m)
	if err != nil {
		return body
	}
	env["meta"] = raw
	out, err := json.Marshal(env)
	if err != nil {
		return body
	}
	return out
}

// StartIdempotencyCleanup deletes keys older than 24 hours every interval, until stop is called
func (a *API) StartIdempotencyCleanup(interval time.Duration) (stop func()) {
	done := make(chan struct{})
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if _, err := a.db.Exec(fmt.Sprintf(`DELETE FROM %s WHERE created_time < CURRENT_TIMESTAMP - interval '%s'`,
					a.idempotencyTable(), idempotencyRetention)); err != nil {
					logger.Get().Error().Err(err).Msg("failed to clean up idempotency keys")
				}
			}
		}
	}()
	return func() { close(done) }
}
