package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/handlers"
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"
	"github.com/aptlogica/sereni-base/tests/handlers/mocks"
)

func baseAPIContext(method, target string) (*gin.Context, func() automationResponse, *int) {
	c, w := newAutomationContext(method, target, nil)
	c.Params = gin.Params{{Key: "id", Value: "base-1"}}
	status := new(int)
	return c, func() automationResponse {
		*status = w.Code
		var res automationResponse
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		return res
	}, status
}

func TestBaseAPIHandler_Get(t *testing.T) {
	t.Run("returns the status", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		svc := mocks.NewMockBaseAPIService(ctrl)
		svc.EXPECT().Get(gomock.Any(), "tenant_1", "base-1").Return(dto.BaseAPIResponse{BaseID: "base-1"}, nil)
		c, read, status := baseAPIContext(http.MethodGet, "/base/base-1/api")

		handlers.NewBaseAPIHandler(svc).GetBaseAPI(c)

		res := read()
		assert.Equal(t, http.StatusOK, *status)
		assert.True(t, res.Success)
		assert.Equal(t, "false", extract(t, res.Data, "enabled"))
		assert.Empty(t, extract(t, res.Data, "public_id"), "public_id is left out until the API is turned on")
	})

	t.Run("database error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		svc := mocks.NewMockBaseAPIService(ctrl)
		svc.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(dto.BaseAPIResponse{}, app_errors.DatabaseError)
		c, read, status := baseAPIContext(http.MethodGet, "/base/base-1/api")

		handlers.NewBaseAPIHandler(svc).GetBaseAPI(c)

		res := read()
		assert.False(t, res.Success)
		assert.Equal(t, http.StatusInternalServerError, *status)
	})
}

func TestBaseAPIHandler_Enable(t *testing.T) {
	t.Run("returns the public ID and base path", func(t *testing.T) {
		enabledAt := time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)
		ctrl := gomock.NewController(t)
		svc := mocks.NewMockBaseAPIService(ctrl)
		svc.EXPECT().Enable(gomock.Any(), "tenant_1", "base-1").Return(dto.BaseAPIResponse{
			BaseID: "base-1", Enabled: true, PublicID: "bs_8f3k2m9qx7", BasePath: "/api/v1/bs_8f3k2m9qx7", EnabledAt: &enabledAt,
		}, nil)
		c, read, status := baseAPIContext(http.MethodPost, "/base/base-1/api/enable")

		handlers.NewBaseAPIHandler(svc).EnableBaseAPI(c)

		res := read()
		require.Equal(t, http.StatusOK, *status)
		assert.True(t, res.Success)
		assert.Equal(t, "true", extract(t, res.Data, "enabled"))
		assert.Equal(t, `"bs_8f3k2m9qx7"`, extract(t, res.Data, "public_id"))
		assert.Equal(t, `"/api/v1/bs_8f3k2m9qx7"`, extract(t, res.Data, "base_path"))
	})

	t.Run("missing workspace is 404", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		svc := mocks.NewMockBaseAPIService(ctrl)
		svc.EXPECT().Enable(gomock.Any(), gomock.Any(), gomock.Any()).Return(dto.BaseAPIResponse{}, app_errors.WorkspaceNotFound)
		c, read, status := baseAPIContext(http.MethodPost, "/base/base-1/api/enable")

		handlers.NewBaseAPIHandler(svc).EnableBaseAPI(c)

		res := read()
		assert.Equal(t, http.StatusNotFound, *status)
		require.NotNil(t, res.Error)
		assert.Equal(t, string(responseConst.WorkspaceError.ErrNotFound), res.Error.Code)
	})

	t.Run("no free public ID is 500", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		svc := mocks.NewMockBaseAPIService(ctrl)
		svc.EXPECT().Enable(gomock.Any(), gomock.Any(), gomock.Any()).Return(dto.BaseAPIResponse{}, app_errors.BaseAPIPublicIDFailed)
		c, read, status := baseAPIContext(http.MethodPost, "/base/base-1/api/enable")

		handlers.NewBaseAPIHandler(svc).EnableBaseAPI(c)

		res := read()
		assert.Equal(t, http.StatusInternalServerError, *status)
		require.NotNil(t, res.Error)
		assert.Equal(t, string(responseConst.BaseAPIError.PublicIDFailed), res.Error.Code)
	})
}
