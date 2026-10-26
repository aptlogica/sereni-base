package handlers_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/handlers"
	"github.com/aptlogica/sereni-base/tests/handlers/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestTableHandler_FuzzyDuplicates(t *testing.T) {
	modelID, colID, _ := enhancementColumnIDs()
	validReq := dto.FuzzyDuplicatesRequest{
		ModelID: modelID, Columns: []string{colID},
		Threshold: "medium", Duplicate: "remove_row", KeepRule: "keep_first",
	}

	t.Run("malformed JSON", func(t *testing.T) {
		handler := handlers.NewTableHandler(nil, nil)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("invalid"))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Set("schema", "test")
		handler.FuzzyDuplicates(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("validation error", func(t *testing.T) {
		handler := handlers.NewTableHandler(nil, nil)
		w, c := postJSONContext(map[string]interface{}{})
		handler.FuzzyDuplicates(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid threshold", func(t *testing.T) {
		handler := handlers.NewTableHandler(nil, nil)
		req := validReq
		req.Threshold = "extreme"
		w, c := postJSONContext(req)
		handler.FuzzyDuplicates(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validate columns error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockSvc := mocks.NewMockTableManagementService(ctrl)
		mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID}).Return(app_errors.ColumnNotFound)
		handler := handlers.NewTableHandler(mockSvc, nil)
		w, c := postJSONContext(validReq)
		handler.FuzzyDuplicates(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockSvc := mocks.NewMockTableManagementService(ctrl)
		mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID}).Return(nil)
		mockSvc.EXPECT().FuzzyDuplicates(gomock.Any(), "test", validReq).Return(dto.FuzzyDuplicatesResponse{}, errors.New("fuzzy failed"))
		handler := handlers.NewTableHandler(mockSvc, nil)
		w, c := postJSONContext(validReq)
		handler.FuzzyDuplicates(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockSvc := mocks.NewMockTableManagementService(ctrl)
		mockSvc.EXPECT().ValidateColumnsAllowed(gomock.Any(), "test", modelID, []string{colID}).Return(nil)
		mockSvc.EXPECT().FuzzyDuplicates(gomock.Any(), "test", validReq).Return(dto.FuzzyDuplicatesResponse{TotalRowsAffected: 2, TotalDuplicateRows: 2}, nil)
		handler := handlers.NewTableHandler(mockSvc, nil)
		w, c := postJSONContext(validReq)
		handler.FuzzyDuplicates(c)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
