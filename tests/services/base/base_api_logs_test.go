package base_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/providers/logquery"
	"github.com/aptlogica/sereni-base/internal/services/base"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/tests/handlers/mocks"
)

type fakeLogs struct {
	calls  int
	path   string
	params url.Values
	res    logquery.Result
	err    error
}

func (f *fakeLogs) Query(_ context.Context, p url.Values) (logquery.Result, error) {
	f.calls++
	f.path, f.params = "query", p
	return f.res, f.err
}

func (f *fakeLogs) Count(_ context.Context, p url.Values) (logquery.Result, error) {
	f.calls++
	f.path, f.params = "count", p
	return f.res, f.err
}

func newLogService(t *testing.T, publicID string, logs *fakeLogs) interfaces.BaseAPILogService {
	ctrl := gomock.NewController(t)
	api := mocks.NewMockBaseAPIService(ctrl)
	api.EXPECT().Get(gomock.Any(), "tenant_1", "base-1").
		Return(dto.BaseAPIResponse{BaseID: "base-1", PublicID: publicID, Enabled: publicID != ""}, nil).AnyTimes()
	return services.NewBaseAPILogService(api, logs)
}

func TestBaseAPILogService_ForcesServiceAndBase(t *testing.T) {
	logs := &fakeLogs{res: logquery.Result{Status: 200, Body: []byte(`{"raw":true}`)}}
	svc := newLogService(t, "bs_8f3k2m9qx7", logs)

	res, err := svc.List(context.Background(), "tenant_1", "base-1", dto.BaseAPILogsRequest{
		Level: "ERROR", Status: "500", Method: "get", RequestID: "r1", Table: "tbl_1", ErrorCode: "internal_error",
		Search: "boom", Start: "2026-11-26T10:00:00+05:30", End: "2026-11-26T12:00:00Z", Limit: "50", Direction: "forward", Cursor: "c1",
	})

	require.NoError(t, err)
	assert.Equal(t, `{"raw":true}`, string(res.Body), "the answer of the log service is not changed")
	assert.Equal(t, "query", logs.path)
	p := logs.params
	assert.Equal(t, "sereni-base", p.Get("service"))
	assert.Equal(t, "bs_8f3k2m9qx7", p.Get("fields.base_id"))
	assert.Equal(t, "error", p.Get("level"))
	assert.Equal(t, "500", p.Get("fields.status"))
	assert.Equal(t, "GET", p.Get("fields.method"))
	assert.Equal(t, "r1", p.Get("fields.request_id"))
	assert.Equal(t, "tbl_1", p.Get("fields.table"))
	assert.Equal(t, "internal_error", p.Get("fields.error_code"))
	assert.Equal(t, "boom", p.Get("search"))
	assert.Equal(t, "2026-11-26T04:30:00Z", p.Get("start"))
	assert.Equal(t, "2026-11-26T12:00:00Z", p.Get("end"))
	assert.Equal(t, "50", p.Get("limit"))
	assert.Equal(t, "forward", p.Get("direction"))
	assert.Equal(t, "c1", p.Get("cursor"))
	assert.Empty(t, p.Get("query"), "the raw LogQL parameter is never sent")
}

func TestBaseAPILogService_CountHasNoPaging(t *testing.T) {
	logs := &fakeLogs{res: logquery.Result{Status: 200, Body: []byte(`{}`)}}
	svc := newLogService(t, "bs_8f3k2m9qx7", logs)

	_, err := svc.Count(context.Background(), "tenant_1", "base-1", dto.BaseAPILogsRequest{Limit: "50", Direction: "forward", Cursor: "c", Level: "warn"})

	require.NoError(t, err)
	assert.Equal(t, "count", logs.path)
	assert.Equal(t, "warn", logs.params.Get("level"))
	assert.Equal(t, "bs_8f3k2m9qx7", logs.params.Get("fields.base_id"))
	for _, name := range []string{"limit", "direction", "cursor"} {
		assert.Empty(t, logs.params.Get(name), name)
	}
}

func TestBaseAPILogService_APINeverTurnedOn(t *testing.T) {
	logs := &fakeLogs{}
	svc := newLogService(t, "", logs)

	list, err := svc.List(context.Background(), "tenant_1", "base-1", dto.BaseAPILogsRequest{})
	require.NoError(t, err)
	assert.Equal(t, 200, list.Status)
	var l struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int   `json:"count"`
			Entries []any `json:"entries"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(list.Body, &l))
	assert.True(t, l.Success)
	assert.Equal(t, 0, l.Data.Count)
	assert.NotNil(t, l.Data.Entries, "entries is [] not null")

	count, err := svc.Count(context.Background(), "tenant_1", "base-1", dto.BaseAPILogsRequest{})
	require.NoError(t, err)
	var c struct {
		Data struct {
			Count int    `json:"count"`
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(count.Body, &c))
	assert.Equal(t, 0, c.Data.Count)
	assert.NotEmpty(t, c.Data.Start)
	assert.NotEmpty(t, c.Data.End)

	assert.Zero(t, logs.calls, "the log service is not called for a base that never logged")
}

func TestBaseAPILogService_BadFilters(t *testing.T) {
	tests := []struct {
		name string
		req  dto.BaseAPILogsRequest
	}{
		{"limit above max", dto.BaseAPILogsRequest{Limit: "501"}},
		{"limit zero", dto.BaseAPILogsRequest{Limit: "0"}},
		{"limit text", dto.BaseAPILogsRequest{Limit: "abc"}},
		{"direction", dto.BaseAPILogsRequest{Direction: "sideways"}},
		{"level", dto.BaseAPILogsRequest{Level: "loud"}},
		{"status text", dto.BaseAPILogsRequest{Status: "ok"}},
		{"status range", dto.BaseAPILogsRequest{Status: "99"}},
		{"start", dto.BaseAPILogsRequest{Start: "yesterday"}},
		{"end", dto.BaseAPILogsRequest{End: "2026-11-26"}},
		{"end before start", dto.BaseAPILogsRequest{Start: "2026-11-26T12:00:00Z", End: "2026-11-26T10:00:00Z"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := &fakeLogs{}
			_, err := newLogService(t, "bs_8f3k2m9qx7", logs).List(context.Background(), "tenant_1", "base-1", tt.req)
			var bad *interfaces.InvalidLogFilterError
			require.ErrorAs(t, err, &bad)
			assert.Zero(t, logs.calls)
		})
	}

	t.Run("limit 500 is allowed", func(t *testing.T) {
		logs := &fakeLogs{res: logquery.Result{Status: 200, Body: []byte(`{}`)}}
		_, err := newLogService(t, "bs_8f3k2m9qx7", logs).List(context.Background(), "tenant_1", "base-1", dto.BaseAPILogsRequest{Limit: "500"})
		require.NoError(t, err)
		assert.Equal(t, "500", logs.params.Get("limit"))
	})
}

func TestBaseAPILogService_ProviderErrorsAreReturned(t *testing.T) {
	logs := &fakeLogs{err: logquery.ErrUnavailable}
	_, err := newLogService(t, "bs_8f3k2m9qx7", logs).Count(context.Background(), "tenant_1", "base-1", dto.BaseAPILogsRequest{})
	assert.True(t, errors.Is(err, logquery.ErrUnavailable))
}

func TestBaseAPILogService_BadStatusOfLogServiceIsPassed(t *testing.T) {
	logs := &fakeLogs{res: logquery.Result{Status: 400, Body: []byte(`{"success":false,"message":"window too long"}`)}}
	res, err := newLogService(t, "bs_8f3k2m9qx7", logs).List(context.Background(), "tenant_1", "base-1", dto.BaseAPILogsRequest{})
	require.NoError(t, err)
	assert.Equal(t, 400, res.Status)
	assert.Equal(t, `{"success":false,"message":"window too long"}`, string(res.Body))
}
