package handlers_test

import (
	"encoding/json"
	"net/http"
	"strings"
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

func newConstraintHandler(t *testing.T) (*handlers.TableConstraintHandler, *mocks.MockTableConstraintService) {
	ctrl := gomock.NewController(t)
	svc := mocks.NewMockTableConstraintService(ctrl)
	return handlers.NewTableConstraintHandler(svc), svc
}

var sampleConstraint = dto.TableConstraintResponse{ID: "k1", Title: "Unique email", Status: "ok"}

func TestNewTableConstraintHandler(t *testing.T) {
	assert.NotNil(t, handlers.NewTableConstraintHandler(nil))
}

func TestTableConstraintHandler_GetTableConstraints(t *testing.T) {
	t.Run("lists unique keys by default", func(t *testing.T) {
		h, svc := newConstraintHandler(t)
		svc.EXPECT().List(gomock.Any(), "tenant_1", "m1", "unique").Return([]dto.TableConstraintResponse{sampleConstraint}, nil)

		c, w := newAutomationContext(http.MethodGet, "/table-constraint?model_id=m1", nil)
		h.GetTableConstraints(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var items []dto.TableConstraintResponse
		require.NoError(t, json.Unmarshal(decodeAutomationResponse(t, w).Data, &items))
		assert.Equal(t, "Unique email", items[0].Title)
	})

	t.Run("passes the requested type", func(t *testing.T) {
		h, svc := newConstraintHandler(t)
		svc.EXPECT().List(gomock.Any(), "tenant_1", "m1", "primary").Return([]dto.TableConstraintResponse{}, nil)

		c, w := newAutomationContext(http.MethodGet, "/table-constraint?model_id=m1&type=primary", nil)
		h.GetTableConstraints(c)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("requires model_id", func(t *testing.T) {
		h, _ := newConstraintHandler(t)
		c, w := newAutomationContext(http.MethodGet, "/table-constraint", nil)
		h.GetTableConstraints(c)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestTableConstraintHandler_CheckTableConstraint(t *testing.T) {
	t.Run("returns the duplicate report", func(t *testing.T) {
		h, svc := newConstraintHandler(t)
		svc.EXPECT().Check(gomock.Any(), "tenant_1", gomock.Any()).
			Return(dto.ConstraintCheckResult{Valid: false, Message: `Duplicate data in column "Email". Remove the duplicates and try again.`}, nil)

		c, w := newAutomationContext(http.MethodPost, "/table-constraint/check", dto.CheckTableConstraintRequest{ModelID: "m1", ColumnIDs: []string{"c1"}})
		h.CheckTableConstraint(c)

		assert.Equal(t, http.StatusOK, w.Code)
		var result dto.ConstraintCheckResult
		require.NoError(t, json.Unmarshal(decodeAutomationResponse(t, w).Data, &result))
		assert.False(t, result.Valid)
		assert.Equal(t, `Duplicate data in column "Email". Remove the duplicates and try again.`, result.Message)
	})

	t.Run("requires at least one column", func(t *testing.T) {
		h, _ := newConstraintHandler(t)
		c, w := newAutomationContext(http.MethodPost, "/table-constraint/check", dto.CheckTableConstraintRequest{ModelID: "m1", ColumnIDs: []string{}})
		h.CheckTableConstraint(c)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("shows why a column can't be used", func(t *testing.T) {
		h, svc := newConstraintHandler(t)
		svc.EXPECT().Check(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(dto.ConstraintCheckResult{}, app_errors.WithMessage(app_errors.ConstraintColumnNotAllowed, "Photo can't be part of a key: attachments can't be part of a key"))

		c, w := newAutomationContext(http.MethodPost, "/table-constraint/check", dto.CheckTableConstraintRequest{ModelID: "m1", ColumnIDs: []string{"c1"}})
		h.CheckTableConstraint(c)

		res := decodeAutomationResponse(t, w)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "TCN_1005", res.Error.Code)
		assert.Equal(t, "Photo can't be part of a key: attachments can't be part of a key", res.Error.Message)
	})
}

func TestTableConstraintHandler_CreateTableConstraint(t *testing.T) {
	body := dto.CreateTableConstraintRequest{ModelID: "m1", Type: "unique", Title: "  Unique email ", ColumnIDs: []string{"c1"}}

	t.Run("creates with the trimmed title and the logged-in user", func(t *testing.T) {
		h, svc := newConstraintHandler(t)
		svc.EXPECT().Create(gomock.Any(), "tenant_1", gomock.Any()).
			DoAndReturn(func(_ any, _ string, req dto.CreateTableConstraintRequest) (dto.TableConstraintResponse, error) {
				assert.Equal(t, "Unique email", req.Title)
				assert.Equal(t, "user-1", req.CreatedBy)
				return sampleConstraint, nil
			})

		c, w := newAutomationContext(http.MethodPost, "/table-constraint/create", body)
		h.CreateTableConstraint(c)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("rejects an unknown type", func(t *testing.T) {
		h, _ := newConstraintHandler(t)
		bad := body
		bad.Type = "primary"
		c, w := newAutomationContext(http.MethodPost, "/table-constraint/create", bad)
		h.CreateTableConstraint(c)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("a check constraint needs a condition", func(t *testing.T) {
		h, _ := newConstraintHandler(t)
		bad := body
		bad.Type = "check"
		c, w := newAutomationContext(http.MethodPost, "/table-constraint/create", bad)
		h.CreateTableConstraint(c)
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	})

	t.Run("creates a check constraint without columns", func(t *testing.T) {
		h, svc := newConstraintHandler(t)
		svc.EXPECT().Create(gomock.Any(), "tenant_1", gomock.Any()).
			DoAndReturn(func(_ any, _ string, req dto.CreateTableConstraintRequest) (dto.TableConstraintResponse, error) {
				assert.Equal(t, "age > 0", req.Condition)
				assert.Equal(t, "number_date", req.Category)
				return sampleConstraint, nil
			})

		check := dto.CreateTableConstraintRequest{ModelID: "m1", Type: "check", Title: "Age positive",
			CheckInput: dto.CheckInput{Category: "number_date", Condition: "age > 0"}}
		c, w := newAutomationContext(http.MethodPost, "/table-constraint/create", check)
		h.CreateTableConstraint(c)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("shows how many records break the condition", func(t *testing.T) {
		h, svc := newConstraintHandler(t)
		svc.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(dto.TableConstraintResponse{}, app_errors.WithMessage(app_errors.CheckExistingRows, "3 records break this condition. Fix them or change the condition."))

		check := dto.CreateTableConstraintRequest{ModelID: "m1", Type: "check", Title: "Age positive", CheckInput: dto.CheckInput{Condition: "age > 0"}}
		c, w := newAutomationContext(http.MethodPost, "/table-constraint/create", check)
		h.CreateTableConstraint(c)

		res := decodeAutomationResponse(t, w)
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Equal(t, "TCN_1016", res.Error.Code)
		assert.Equal(t, "3 records break this condition. Fix them or change the condition.", res.Error.Message)
	})

	t.Run("rejects a blank or long title", func(t *testing.T) {
		h, _ := newConstraintHandler(t)
		blank := body
		blank.Title = "  "
		c, w := newAutomationContext(http.MethodPost, "/table-constraint/create", blank)
		h.CreateTableConstraint(c)
		assert.Equal(t, "TCN_1002", decodeAutomationResponse(t, w).Error.Code)

		long := body
		long.Title = strings.Repeat("x", 51)
		c, w = newAutomationContext(http.MethodPost, "/table-constraint/create", long)
		h.CreateTableConstraint(c)
		assert.Equal(t, "TCN_1003", decodeAutomationResponse(t, w).Error.Code)
	})

	t.Run("returns 409 with the count when records repeat the values", func(t *testing.T) {
		h, svc := newConstraintHandler(t)
		svc.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(dto.TableConstraintResponse{}, app_errors.WithMessage(app_errors.ConstraintDuplicateValues, "Some records already have the same values in these columns. Remove the duplicates and try again."))

		c, w := newAutomationContext(http.MethodPost, "/table-constraint/create", body)
		h.CreateTableConstraint(c)

		res := decodeAutomationResponse(t, w)
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Equal(t, "TCN_1009", res.Error.Code)
		assert.Equal(t, "Some records already have the same values in these columns. Remove the duplicates and try again.", res.Error.Message)
	})
}

func TestTableConstraintHandler_UpdateTableConstraint(t *testing.T) {
	t.Run("updates with the id and the logged-in user", func(t *testing.T) {
		h, svc := newConstraintHandler(t)
		svc.EXPECT().Update(gomock.Any(), "tenant_1", "k1", gomock.Any()).
			DoAndReturn(func(_ any, _, _ string, req dto.UpdateTableConstraintRequest) (dto.TableConstraintResponse, error) {
				assert.Equal(t, "user-1", req.UpdatedBy)
				return sampleConstraint, nil
			})

		c, w := newAutomationContext(http.MethodPut, "/table-constraint/k1", dto.UpdateTableConstraintRequest{Title: "x", ColumnIDs: []string{"c1"}})
		c.Params = gin.Params{{Key: "id", Value: "k1"}}
		h.UpdateTableConstraint(c)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 404 for an unknown key", func(t *testing.T) {
		h, svc := newConstraintHandler(t)
		svc.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(dto.TableConstraintResponse{}, app_errors.ConstraintNotFound)

		c, w := newAutomationContext(http.MethodPut, "/table-constraint/k1", dto.UpdateTableConstraintRequest{Title: "x", ColumnIDs: []string{"c1"}})
		h.UpdateTableConstraint(c)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("rejects a blank title", func(t *testing.T) {
		h, _ := newConstraintHandler(t)
		c, w := newAutomationContext(http.MethodPut, "/table-constraint/k1", dto.UpdateTableConstraintRequest{Title: " ", ColumnIDs: []string{"c1"}})
		h.UpdateTableConstraint(c)
		assert.Equal(t, "TCN_1002", decodeAutomationResponse(t, w).Error.Code)
	})
}

func TestTableConstraintHandler_DeleteTableConstraint(t *testing.T) {
	t.Run("deletes the key", func(t *testing.T) {
		h, svc := newConstraintHandler(t)
		svc.EXPECT().Delete(gomock.Any(), "tenant_1", "k1").Return(nil)

		c, w := newAutomationContext(http.MethodDelete, "/table-constraint/k1", nil)
		c.Params = gin.Params{{Key: "id", Value: "k1"}}
		h.DeleteTableConstraint(c)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("shows the Postgres message when dropping fails", func(t *testing.T) {
		h, svc := newConstraintHandler(t)
		svc.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(app_errors.WithMessage(app_errors.ConstraintFailed, "constraint is referenced by a foreign key"))

		c, w := newAutomationContext(http.MethodDelete, "/table-constraint/k1", nil)
		h.DeleteTableConstraint(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "constraint is referenced by a foreign key", decodeAutomationResponse(t, w).Error.Message)
	})
}
