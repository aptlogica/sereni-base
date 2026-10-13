package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/handlers"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// atomicRowService supports saving all fields of a record in one statement
type atomicRowService struct {
	interfaces.TableManagementService
	values map[string]interface{}
	err    error
}

func (s *atomicRowService) UpdateRowValues(_ context.Context, _ string, _ string, rowID int, values map[string]interface{}, _ string) (dto.RecordResponse, error) {
	s.values = values
	return dto.RecordResponse{Record: map[string]interface{}{"id": rowID}}, s.err
}

func updateRowRequest(values map[string]interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	body, _ := json.Marshal(dto.UpdateRowRequest{ModelID: "m1", RowId: 7, Values: values})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPatch, "/row/update", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")
	return c, w
}

func TestTableHandler_UpdateRow_SavesAllFieldsAtOnce(t *testing.T) {
	svc := &atomicRowService{}
	c, w := updateRowRequest(map[string]interface{}{"col-code": "HR", "col-year": 2026.0})

	handlers.NewTableHandler(svc, nil).UpdateRow(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, map[string]interface{}{"col-code": "HR", "col-year": 2026.0}, svc.values)
}

func TestTableHandler_UpdateRow_ReturnsTheExactKeyError(t *testing.T) {
	message := `The combination Code = "HR", Year = 2026 is already used by record #4 (Acme). Each record needs a unique primary key (Code + Year).`
	svc := &atomicRowService{err: &app_errors.APIError{Code: "PKY_1008", Message: message, StatusCode: http.StatusConflict}}
	c, w := updateRowRequest(map[string]interface{}{"col-code": "HR"})

	handlers.NewTableHandler(svc, nil).UpdateRow(c)

	assert.Equal(t, http.StatusConflict, w.Code)
	var res struct {
		Error struct{ Code, Message string } `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	assert.Equal(t, "PKY_1008", res.Error.Code)
	assert.Equal(t, message, res.Error.Message)
}

// upsertRowService supports POST /row/data/insert as an upsert
type upsertRowService struct {
	interfaces.TableManagementService
	rowID  int
	values map[string]interface{}
	err    error
}

func (s *upsertRowService) UpsertRowValues(_ context.Context, _ string, _ string, rowID int, values map[string]interface{}, _ string) (dto.RecordResponse, error) {
	s.rowID, s.values = rowID, values
	return dto.RecordResponse{Record: map[string]interface{}{"id": 9}}, s.err
}

func insertRequest(body map[string]interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	raw, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/row/data/insert", bytes.NewBuffer(raw))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "test")
	c.Set("user_id", "user123")
	return c, w
}

func TestTableHandler_InsertRowData_Upsert(t *testing.T) {
	t.Run("without row_id the record is created with all values", func(t *testing.T) {
		svc := &upsertRowService{}
		c, w := insertRequest(map[string]interface{}{"model_id": "m1", "values": map[string]interface{}{"col-code": "HR", "col-year": 2026}})

		handlers.NewTableHandler(svc, nil).InsertRowData(c)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Equal(t, 0, svc.rowID)
		assert.Equal(t, map[string]interface{}{"col-code": "HR", "col-year": 2026.0}, svc.values)
	})

	t.Run("with row_id a single cell is updated", func(t *testing.T) {
		svc := &upsertRowService{}
		c, _ := insertRequest(map[string]interface{}{"model_id": "m1", "row_id": 7, "column_id": "col-code", "value": nil})

		handlers.NewTableHandler(svc, nil).InsertRowData(c)

		assert.Equal(t, 7, svc.rowID)
		assert.Equal(t, map[string]interface{}{"col-code": nil}, svc.values)
	})

	t.Run("an update needs at least one value", func(t *testing.T) {
		svc := &upsertRowService{}
		c, w := insertRequest(map[string]interface{}{"model_id": "m1", "row_id": 7})

		handlers.NewTableHandler(svc, nil).InsertRowData(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Nil(t, svc.values)
	})

	t.Run("missing key fields come back exactly", func(t *testing.T) {
		message := "Code and Year can't be empty: they are part of the primary key (Code + Year)."
		svc := &upsertRowService{err: &app_errors.APIError{Code: "PKY_1007", Message: message, StatusCode: http.StatusBadRequest}}
		c, w := insertRequest(map[string]interface{}{"model_id": "m1", "values": map[string]interface{}{"col-notes": "n"}})

		handlers.NewTableHandler(svc, nil).InsertRowData(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), message)
	})
}
