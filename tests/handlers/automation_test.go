package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/handlers"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/tests/handlers/mocks"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type automationResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details string `json:"details"`
	} `json:"error"`
}

// newAutomationContext builds a gin context with the schema and user set, as the auth middleware does
func newAutomationContext(method, target string, body any) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	var reader *bytes.Buffer
	switch b := body.(type) {
	case nil:
		reader = bytes.NewBuffer(nil)
	case string:
		reader = bytes.NewBufferString(b)
	default:
		raw, _ := json.Marshal(b)
		reader = bytes.NewBuffer(raw)
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, target, reader)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("schema", "tenant_1")
	c.Set("user_id", "user-1")
	return c, w
}

func decodeAutomationResponse(t *testing.T, w *httptest.ResponseRecorder) automationResponse {
	t.Helper()
	var res automationResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
	return res
}

func newAutomationHandler(t *testing.T) (*handlers.AutomationHandler, *mocks.MockAutomationService) {
	ctrl := gomock.NewController(t)
	svc := mocks.NewMockAutomationService(ctrl)
	return handlers.NewAutomationHandler(svc), svc
}

var sampleAutomation = tenant.Automation{ID: uuid.New(), ModelID: "m1", Title: "Log salary", Type: tenant.AutomationTypeTrigger}

func TestNewAutomationHandler(t *testing.T) {
	assert.NotNil(t, handlers.NewAutomationHandler(nil))
}

func TestAutomationHandler_CreateAutomation(t *testing.T) {
	validBody := dto.CreateAutomationRequest{ModelID: "m1", Title: "  Log salary  ", Type: "trigger", Context: "CREATE TRIGGER t", Event: "AFTER INSERT"}

	t.Run("creates with trimmed title and the logged-in user", func(t *testing.T) {
		h, svc := newAutomationHandler(t)
		svc.EXPECT().Create(gomock.Any(), "tenant_1", gomock.Any()).
			DoAndReturn(func(_ any, _ string, req dto.CreateAutomationRequest) (tenant.Automation, error) {
				assert.Equal(t, "Log salary", req.Title)
				assert.Equal(t, "user-1", req.CreatedBy)
				assert.Equal(t, "AFTER INSERT", req.Event)
				return sampleAutomation, nil
			})

		c, w := newAutomationContext(http.MethodPost, "/automation/create", validBody)
		h.CreateAutomation(c)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.True(t, decodeAutomationResponse(t, w).Success)
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		h, _ := newAutomationHandler(t)
		c, w := newAutomationContext(http.MethodPost, "/automation/create", "not json")
		h.CreateAutomation(c)
		assert.NotEqual(t, http.StatusOK, w.Code)
		assert.False(t, decodeAutomationResponse(t, w).Success)
	})

	t.Run("rejects an unknown type", func(t *testing.T) {
		h, _ := newAutomationHandler(t)
		body := validBody
		body.Type = "cron"
		c, w := newAutomationContext(http.MethodPost, "/automation/create", body)
		h.CreateAutomation(c)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("rejects a blank title", func(t *testing.T) {
		h, _ := newAutomationHandler(t)
		body := validBody
		body.Title = "   "
		c, w := newAutomationContext(http.MethodPost, "/automation/create", body)
		h.CreateAutomation(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "AUT_1005", decodeAutomationResponse(t, w).Error.Code)
	})

	t.Run("rejects a title over 50 characters", func(t *testing.T) {
		h, _ := newAutomationHandler(t)
		body := validBody
		body.Title = strings.Repeat("a", 51)
		c, w := newAutomationContext(http.MethodPost, "/automation/create", body)
		h.CreateAutomation(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "AUT_1006", decodeAutomationResponse(t, w).Error.Code)
	})

	t.Run("returns the Postgres message when the query fails", func(t *testing.T) {
		h, svc := newAutomationHandler(t)
		svc.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(tenant.Automation{}, fmt.Errorf("%w: %v", app_errors.TriggerFailed, errors.New(`pq: syntax error at or near "TRIGER"`)))

		c, w := newAutomationContext(http.MethodPost, "/automation/create", validBody)
		h.CreateAutomation(c)

		res := decodeAutomationResponse(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "AUT_1003", res.Error.Code)
		assert.Equal(t, `syntax error at or near "TRIGER"`, res.Error.Message)
	})

	t.Run("returns 409 for a title already in use", func(t *testing.T) {
		h, svc := newAutomationHandler(t)
		svc.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(tenant.Automation{}, app_errors.DuplicateAutomationTitle)

		c, w := newAutomationContext(http.MethodPost, "/automation/create", validBody)
		h.CreateAutomation(c)

		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Equal(t, "AUT_1011", decodeAutomationResponse(t, w).Error.Code)
	})

	t.Run("maps known service errors", func(t *testing.T) {
		h, svc := newAutomationHandler(t)
		svc.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(tenant.Automation{}, app_errors.TriggerHasFunction)

		c, w := newAutomationContext(http.MethodPost, "/automation/create", validBody)
		h.CreateAutomation(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "AUT_1009", decodeAutomationResponse(t, w).Error.Code)
	})
}

func TestAutomationHandler_GetAutomations(t *testing.T) {
	t.Run("lists the automations of a table", func(t *testing.T) {
		h, svc := newAutomationHandler(t)
		svc.EXPECT().List(gomock.Any(), "tenant_1", "m1").Return([]tenant.Automation{sampleAutomation}, nil)

		c, w := newAutomationContext(http.MethodGet, "/automation?model_id=%20m1%20", nil)
		h.GetAutomations(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var items []tenant.Automation
		require.NoError(t, json.Unmarshal(decodeAutomationResponse(t, w).Data, &items))
		assert.Len(t, items, 1)
	})

	t.Run("requires model_id", func(t *testing.T) {
		h, _ := newAutomationHandler(t)
		c, w := newAutomationContext(http.MethodGet, "/automation", nil)
		h.GetAutomations(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("passes service errors on", func(t *testing.T) {
		h, svc := newAutomationHandler(t)
		svc.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, app_errors.DatabaseError)

		c, w := newAutomationContext(http.MethodGet, "/automation?model_id=m1", nil)
		h.GetAutomations(c)
		assert.False(t, decodeAutomationResponse(t, w).Success)
		assert.NotEqual(t, http.StatusOK, w.Code)
	})
}

func TestAutomationHandler_UpdateAutomation(t *testing.T) {
	validBody := dto.UpdateAutomationRequest{Title: " Audit ", Context: "CREATE TRIGGER t", Event: "BEFORE UPDATE"}

	t.Run("updates with the id, trimmed title and the logged-in user", func(t *testing.T) {
		h, svc := newAutomationHandler(t)
		svc.EXPECT().Update(gomock.Any(), "tenant_1", "a1", gomock.Any()).
			DoAndReturn(func(_ any, _, _ string, req dto.UpdateAutomationRequest) (tenant.Automation, error) {
				assert.Equal(t, "Audit", req.Title)
				assert.Equal(t, "user-1", req.UpdatedBy)
				return sampleAutomation, nil
			})

		c, w := newAutomationContext(http.MethodPut, "/automation/a1", validBody)
		c.Params = gin.Params{{Key: "id", Value: "a1"}}
		h.UpdateAutomation(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 409 for a title already in use", func(t *testing.T) {
		h, svc := newAutomationHandler(t)
		svc.EXPECT().Update(gomock.Any(), "tenant_1", "a1", gomock.Any()).Return(tenant.Automation{}, app_errors.DuplicateAutomationTitle)

		c, w := newAutomationContext(http.MethodPut, "/automation/a1", validBody)
		c.Params = gin.Params{{Key: "id", Value: "a1"}}
		h.UpdateAutomation(c)

		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Equal(t, "AUT_1011", decodeAutomationResponse(t, w).Error.Code)
	})

	t.Run("rejects invalid JSON and missing context", func(t *testing.T) {
		h, _ := newAutomationHandler(t)
		c, w := newAutomationContext(http.MethodPut, "/automation/a1", "{")
		h.UpdateAutomation(c)
		assert.NotEqual(t, http.StatusOK, w.Code)

		c, w = newAutomationContext(http.MethodPut, "/automation/a1", dto.UpdateAutomationRequest{Title: "x"})
		h.UpdateAutomation(c)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("rejects a title over 50 characters", func(t *testing.T) {
		h, _ := newAutomationHandler(t)
		body := validBody
		body.Title = strings.Repeat("b", 51)
		c, w := newAutomationContext(http.MethodPut, "/automation/a1", body)
		h.UpdateAutomation(c)
		assert.Equal(t, "AUT_1006", decodeAutomationResponse(t, w).Error.Code)
	})

	t.Run("returns 404 for an unknown automation", func(t *testing.T) {
		h, svc := newAutomationHandler(t)
		svc.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(tenant.Automation{}, app_errors.AutomationNotFound)

		c, w := newAutomationContext(http.MethodPut, "/automation/a1", validBody)
		h.UpdateAutomation(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, "AUT_1001", decodeAutomationResponse(t, w).Error.Code)
	})
}

func TestAutomationHandler_RunAutomation(t *testing.T) {
	t.Run("runs the automation", func(t *testing.T) {
		h, svc := newAutomationHandler(t)
		svc.EXPECT().Run(gomock.Any(), "tenant_1", "a1").Return(sampleAutomation, nil)

		c, w := newAutomationContext(http.MethodPost, "/automation/a1/run", nil)
		c.Params = gin.Params{{Key: "id", Value: "a1"}}
		h.RunAutomation(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns the Postgres message when running fails", func(t *testing.T) {
		h, svc := newAutomationHandler(t)
		svc.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(tenant.Automation{}, fmt.Errorf("%w: %v", app_errors.TriggerFailed, errors.New("pq: function log_fn() does not exist")))

		c, w := newAutomationContext(http.MethodPost, "/automation/a1/run", nil)
		h.RunAutomation(c)

		res := decodeAutomationResponse(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "function log_fn() does not exist", res.Error.Message)
	})
}

func TestAutomationHandler_ValidateAutomation(t *testing.T) {
	body := dto.ValidateAutomationRequest{ModelID: "m1", Type: "function", Context: "CREATE FUNCTION f() RETURNS trigger AS $$ $$"}

	t.Run("returns the result from the service", func(t *testing.T) {
		h, svc := newAutomationHandler(t)
		svc.EXPECT().Validate(gomock.Any(), "tenant_1", body).
			Return(dto.AutomationValidationResult{Valid: false, Message: "syntax error at end of input"}, nil)

		c, w := newAutomationContext(http.MethodPost, "/automation/validate", body)
		h.ValidateAutomation(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var result dto.AutomationValidationResult
		require.NoError(t, json.Unmarshal(decodeAutomationResponse(t, w).Data, &result))
		assert.False(t, result.Valid)
		assert.Equal(t, "syntax error at end of input", result.Message)
	})

	t.Run("rejects a webhook", func(t *testing.T) {
		h, _ := newAutomationHandler(t)

		c, w := newAutomationContext(http.MethodPost, "/automation/validate", dto.ValidateAutomationRequest{ModelID: "m1", Type: "webhook", Context: "{}"})
		h.ValidateAutomation(c)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("returns a query format error", func(t *testing.T) {
		h, svc := newAutomationHandler(t)
		svc.EXPECT().Validate(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(dto.AutomationValidationResult{}, app_errors.InvalidFunctionQuery)

		c, w := newAutomationContext(http.MethodPost, "/automation/validate", body)
		h.ValidateAutomation(c)

		assert.Equal(t, "AUT_1007", decodeAutomationResponse(t, w).Error.Code)
	})
}

func TestAutomationHandler_DeleteAutomation(t *testing.T) {
	t.Run("deletes the automation", func(t *testing.T) {
		h, svc := newAutomationHandler(t)
		svc.EXPECT().Delete(gomock.Any(), "tenant_1", "a1").Return(nil)

		c, w := newAutomationContext(http.MethodDelete, "/automation/a1", nil)
		c.Params = gin.Params{{Key: "id", Value: "a1"}}
		h.DeleteAutomation(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns the Postgres message when the function is still in use", func(t *testing.T) {
		h, svc := newAutomationHandler(t)
		svc.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(fmt.Errorf("%w: %v", app_errors.TriggerFailed, errors.New("pq: cannot drop function f() because other objects depend on it")))

		c, w := newAutomationContext(http.MethodDelete, "/automation/a1", nil)
		h.DeleteAutomation(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, decodeAutomationResponse(t, w).Error.Message, "cannot drop function")
	})

	t.Run("returns 404 for an unknown automation", func(t *testing.T) {
		h, svc := newAutomationHandler(t)
		svc.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()).Return(app_errors.AutomationNotFound)

		c, w := newAutomationContext(http.MethodDelete, "/automation/a1", nil)
		h.DeleteAutomation(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
