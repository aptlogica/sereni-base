package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/handlers"
)

// fakePrimaryKeyService records the calls and returns the configured result
type fakePrimaryKeyService struct {
	out     dto.PrimaryKeyResponse
	err     error
	schema  string
	modelID string
	req     dto.UpdatePrimaryKeyRequest
	updated bool
}

func (f *fakePrimaryKeyService) Get(_ context.Context, schemaName, modelID string) (dto.PrimaryKeyResponse, error) {
	f.schema, f.modelID = schemaName, modelID
	return f.out, f.err
}

func (f *fakePrimaryKeyService) Update(_ context.Context, schemaName, modelID string, req dto.UpdatePrimaryKeyRequest) (dto.PrimaryKeyResponse, error) {
	f.schema, f.modelID, f.req, f.updated = schemaName, modelID, req, true
	return f.out, f.err
}

func (f *fakePrimaryKeyService) EnsureColumnNotInKey(context.Context, string, string, string) error {
	return f.err
}

func primaryKeyContext(method string, body any) (*gin.Context, func() automationResponse, *int) {
	c, w := newAutomationContext(method, "/table/m1/primary-key", body)
	c.Params = gin.Params{{Key: "id", Value: "m1"}}
	status := new(int)
	return c, func() automationResponse {
		*status = w.Code
		var res automationResponse
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		return res
	}, status
}

func TestPrimaryKeyHandler_Get(t *testing.T) {
	t.Run("returns the key", func(t *testing.T) {
		svc := &fakePrimaryKeyService{out: dto.PrimaryKeyResponse{ModelID: "m1", ColumnNames: []string{"id"}}}
		c, read, status := primaryKeyContext(http.MethodGet, nil)

		handlers.NewPrimaryKeyHandler(svc).GetPrimaryKey(c)

		res := read()
		assert.Equal(t, http.StatusOK, *status)
		assert.True(t, res.Success)
		assert.Equal(t, "tenant_1", svc.schema)
		assert.Equal(t, "m1", svc.modelID)
		assert.JSONEq(t, `["id"]`, extract(t, res.Data, "column_names"))
	})

	t.Run("table not found", func(t *testing.T) {
		svc := &fakePrimaryKeyService{err: app_errors.TableNotFound}
		c, read, status := primaryKeyContext(http.MethodGet, nil)

		handlers.NewPrimaryKeyHandler(svc).GetPrimaryKey(c)

		res := read()
		assert.False(t, res.Success)
		assert.NotEqual(t, http.StatusOK, *status)
	})
}

func TestPrimaryKeyHandler_Update(t *testing.T) {
	t.Run("passes the column IDs in order", func(t *testing.T) {
		svc := &fakePrimaryKeyService{out: dto.PrimaryKeyResponse{ColumnNames: []string{"code", "dept"}}}
		c, read, status := primaryKeyContext(http.MethodPut, map[string]any{"column_ids": []string{"c2", "c1"}})

		handlers.NewPrimaryKeyHandler(svc).UpdatePrimaryKey(c)

		res := read()
		assert.Equal(t, http.StatusOK, *status)
		assert.True(t, res.Success)
		assert.Equal(t, []string{"c2", "c1"}, svc.req.ColumnIDs)
	})

	t.Run("an empty key is rejected before the service", func(t *testing.T) {
		svc := &fakePrimaryKeyService{}
		c, read, status := primaryKeyContext(http.MethodPut, map[string]any{"column_ids": []string{}})

		handlers.NewPrimaryKeyHandler(svc).UpdatePrimaryKey(c)

		res := read()
		assert.Equal(t, http.StatusBadRequest, *status)
		assert.Equal(t, "PKY_1001", res.Error.Code)
		assert.False(t, svc.updated)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		svc := &fakePrimaryKeyService{}
		c, read, _ := primaryKeyContext(http.MethodPut, "{")

		handlers.NewPrimaryKeyHandler(svc).UpdatePrimaryKey(c)

		assert.False(t, read().Success)
		assert.False(t, svc.updated)
	})

	t.Run("an invalid field is named in the message", func(t *testing.T) {
		svc := &fakePrimaryKeyService{err: fmt.Errorf("%w: Skills can't be part of the primary key: holds multiple or long values.", app_errors.PrimaryKeyColumnInvalid)}
		c, read, status := primaryKeyContext(http.MethodPut, map[string]any{"column_ids": []string{"c1"}})

		handlers.NewPrimaryKeyHandler(svc).UpdatePrimaryKey(c)

		res := read()
		assert.Equal(t, http.StatusBadRequest, *status)
		assert.Equal(t, "PKY_1002", res.Error.Code)
		assert.Equal(t, "Skills can't be part of the primary key: holds multiple or long values.", res.Error.Message)
	})

	t.Run("locked by links", func(t *testing.T) {
		svc := &fakePrimaryKeyService{err: app_errors.PrimaryKeyLocked}
		c, read, status := primaryKeyContext(http.MethodPut, map[string]any{"column_ids": []string{"c1"}})

		handlers.NewPrimaryKeyHandler(svc).UpdatePrimaryKey(c)

		res := read()
		assert.Equal(t, http.StatusConflict, *status)
		assert.Equal(t, "PKY_1003", res.Error.Code)
	})

	t.Run("offending records are returned in data", func(t *testing.T) {
		details := dto.PrimaryKeyViolationDetails{Columns: []string{"code"}, NullCount: 1, NullRecords: []dto.PrimaryKeyRecord{{ID: 4, Title: "Acme"}}}
		message := "Can't set the primary key to Code. Code is empty in 1 record (#4 Acme). Fill in or change these records, then save again."
		svc := &fakePrimaryKeyService{err: &app_errors.PrimaryKeyViolation{Message: message, Details: details}}
		c, read, status := primaryKeyContext(http.MethodPut, map[string]any{"column_ids": []string{"c1"}})

		handlers.NewPrimaryKeyHandler(svc).UpdatePrimaryKey(c)

		res := read()
		assert.Equal(t, http.StatusConflict, *status)
		assert.Equal(t, "PKY_1004", res.Error.Code)
		assert.Equal(t, message, res.Error.Message)
		assert.JSONEq(t, `1`, extract(t, res.Data, "null_count"))
		// Also in error.details for clients that keep only the error object
		assert.JSONEq(t, `1`, extract(t, json.RawMessage(res.Error.Details), "null_count"))
	})

	t.Run("Postgres message is shown", func(t *testing.T) {
		svc := &fakePrimaryKeyService{err: fmt.Errorf("%w: %v", app_errors.PrimaryKeyUpdateFailed, errors.New("pq: could not create unique index"))}
		c, read, status := primaryKeyContext(http.MethodPut, map[string]any{"column_ids": []string{"c1"}})

		handlers.NewPrimaryKeyHandler(svc).UpdatePrimaryKey(c)

		res := read()
		assert.Equal(t, http.StatusBadRequest, *status)
		assert.Equal(t, "could not create unique index", res.Error.Message)
	})
}

func extract(t *testing.T, raw json.RawMessage, field string) string {
	t.Helper()
	var obj map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &obj))
	return string(obj[field])
}
