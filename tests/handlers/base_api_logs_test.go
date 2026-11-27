package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/handlers"
	"github.com/aptlogica/sereni-base/internal/providers/logquery"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
)

type fakeLogService struct {
	res    interfaces.BaseAPILogResult
	err    error
	method string
	schema string
	baseID string
	req    dto.BaseAPILogsRequest
}

func (f *fakeLogService) List(_ context.Context, schema, baseID string, req dto.BaseAPILogsRequest) (interfaces.BaseAPILogResult, error) {
	f.method, f.schema, f.baseID, f.req = "list", schema, baseID, req
	return f.res, f.err
}

func (f *fakeLogService) Count(_ context.Context, schema, baseID string, req dto.BaseAPILogsRequest) (interfaces.BaseAPILogResult, error) {
	f.method, f.schema, f.baseID, f.req = "count", schema, baseID, req
	return f.res, f.err
}

func TestBaseAPILogsHandler(t *testing.T) {
	raw := `{"success":true,"code":"OK","message":"ok","data":{"query":"q","count":1,"entries":[{"level":"error"}]}}`

	t.Run("list writes the answer of the log service as it is", func(t *testing.T) {
		svc := &fakeLogService{res: interfaces.BaseAPILogResult{Status: 200, Body: []byte(raw)}}
		c, w := newAutomationContext(http.MethodGet, "/base/base-1/api/logs?level=error&status=500&limit=20&request_id=r1", nil)
		c.Params = append(c.Params, gin.Param{Key: "id", Value: "base-1"})

		handlers.NewBaseAPILogsHandler(svc).ListLogs(c)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, raw, w.Body.String())
		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
		assert.Equal(t, "list", svc.method)
		assert.Equal(t, "tenant_1", svc.schema)
		assert.Equal(t, "base-1", svc.baseID)
		assert.Equal(t, "error", svc.req.Level)
		assert.Equal(t, "500", svc.req.Status)
		assert.Equal(t, "20", svc.req.Limit)
		assert.Equal(t, "r1", svc.req.RequestID)
	})

	t.Run("count", func(t *testing.T) {
		svc := &fakeLogService{res: interfaces.BaseAPILogResult{Status: 200, Body: []byte(`{"data":{"count":7}}`)}}
		c, w := newAutomationContext(http.MethodGet, "/base/base-1/api/logs/count?level=warn", nil)
		c.Params = append(c.Params, gin.Param{Key: "id", Value: "base-1"})

		handlers.NewBaseAPILogsHandler(svc).CountLogs(c)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, `{"data":{"count":7}}`, w.Body.String())
		assert.Equal(t, "count", svc.method)
	})

	t.Run("a 400 of the log service is passed as it is", func(t *testing.T) {
		body := `{"success":false,"code":"BAD_REQUEST","message":"invalid query"}`
		svc := &fakeLogService{res: interfaces.BaseAPILogResult{Status: 400, Body: []byte(body)}}
		c, w := newAutomationContext(http.MethodGet, "/base/base-1/api/logs", nil)
		c.Params = append(c.Params, gin.Param{Key: "id", Value: "base-1"})

		handlers.NewBaseAPILogsHandler(svc).ListLogs(c)

		assert.Equal(t, 400, w.Code)
		assert.Equal(t, body, w.Body.String())
	})

	errorCases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"bad filter", &interfaces.InvalidLogFilterError{Message: "limit must be a number from 1 to 500"}, 400, "BSA_1002"},
		{"not configured", logquery.ErrNotConfigured, 503, "BSA_1003"},
		{"unavailable", logquery.ErrUnavailable, 502, "BSA_1004"},
		{"timeout", logquery.ErrTimeout, 504, "BSA_1005"},
	}
	for _, tt := range errorCases {
		t.Run("error: "+tt.name, func(t *testing.T) {
			svc := &fakeLogService{err: tt.err}
			c, w := newAutomationContext(http.MethodGet, "/base/base-1/api/logs", nil)
			c.Params = append(c.Params, gin.Param{Key: "id", Value: "base-1"})

			handlers.NewBaseAPILogsHandler(svc).ListLogs(c)

			require.Equal(t, tt.wantStatus, w.Code)
			var res automationResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
			assert.False(t, res.Success)
			require.NotNil(t, res.Error)
			assert.Equal(t, tt.wantCode, res.Error.Code)
		})
	}

	t.Run("bad filter message is shown", func(t *testing.T) {
		svc := &fakeLogService{err: &interfaces.InvalidLogFilterError{Message: "limit must be a number from 1 to 500"}}
		c, w := newAutomationContext(http.MethodGet, "/base/base-1/api/logs?limit=501", nil)
		c.Params = append(c.Params, gin.Param{Key: "id", Value: "base-1"})

		handlers.NewBaseAPILogsHandler(svc).ListLogs(c)

		var res automationResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
		assert.Equal(t, "limit must be a number from 1 to 500", res.Error.Message)
	})

	t.Run("an unknown error is a normal error answer", func(t *testing.T) {
		svc := &fakeLogService{err: errors.New("boom")}
		c, w := newAutomationContext(http.MethodGet, "/base/base-1/api/logs", nil)
		c.Params = append(c.Params, gin.Param{Key: "id", Value: "base-1"})

		handlers.NewBaseAPILogsHandler(svc).ListLogs(c)

		assert.GreaterOrEqual(t, w.Code, 400)
	})
}
