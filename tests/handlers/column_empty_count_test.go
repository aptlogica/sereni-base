package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/handlers"
	"github.com/aptlogica/sereni-base/tests/handlers/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// emptyCountService is a table service that can also count empty cells, as the real one does
type emptyCountService struct {
	*mocks.MockTableManagementService
	counts dto.EmptyCountResponse
	err    error
	gotID  string
}

func (s *emptyCountService) CountEmptyCells(_ context.Context, _ string, columnID string) (dto.EmptyCountResponse, error) {
	s.gotID = columnID
	return s.counts, s.err
}

func callEmptyCount(t *testing.T, handler *handlers.TableHandler) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/column/c1/empty-count", nil)
	c.Params = gin.Params{{Key: "id", Value: "c1"}}
	c.Set("schema", "test")
	handler.GetColumnEmptyCount(c)
	return w
}

func TestTableHandler_GetColumnEmptyCount(t *testing.T) {
	t.Run("returns the counts", func(t *testing.T) {
		svc := &emptyCountService{
			MockTableManagementService: mocks.NewMockTableManagementService(gomock.NewController(t)),
			counts:                     dto.EmptyCountResponse{Empty: 128, Filled: 1872},
		}

		w := callEmptyCount(t, handlers.NewTableHandler(svc, nil))

		require.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "c1", svc.gotID)
		var body struct {
			Data dto.EmptyCountResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, dto.EmptyCountResponse{Empty: 128, Filled: 1872}, body.Data)
	})

	t.Run("passes on a service error", func(t *testing.T) {
		svc := &emptyCountService{
			MockTableManagementService: mocks.NewMockTableManagementService(gomock.NewController(t)),
			err:                        app_errors.ColumnNotFound,
		}

		w := callEmptyCount(t, handlers.NewTableHandler(svc, nil))

		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("fails when the service can't count", func(t *testing.T) {
		w := callEmptyCount(t, handlers.NewTableHandler(mocks.NewMockTableManagementService(gomock.NewController(t)), nil))

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
