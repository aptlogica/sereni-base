package tests

import (
	"fmt"
	"net/http"
	"testing"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/utils/response/constants"

	"github.com/stretchr/testify/assert"
)

func TestAutomationErrorCodes(t *testing.T) {
	cases := map[constants.ResponseCode]int{
		constants.AutomationError.AutomationNotFound:   http.StatusNotFound,
		constants.AutomationError.InvalidTriggerQuery:  http.StatusBadRequest,
		constants.AutomationError.TriggerFailed:        http.StatusBadRequest,
		constants.AutomationError.TitleRequired:        http.StatusBadRequest,
		constants.AutomationError.TitleTooLong:         http.StatusBadRequest,
		constants.AutomationError.InvalidFunctionQuery: http.StatusBadRequest,
		constants.AutomationError.TriggerHasFunction:   http.StatusBadRequest,
		constants.AutomationError.InvalidTriggerEvent:  http.StatusBadRequest,
	}
	for code, status := range cases {
		meta, ok := constants.ErrorCodes[code]
		assert.True(t, ok, "%s is not registered in ErrorCodes", code)
		assert.Equal(t, status, meta.HTTPStatus, code)
		assert.NotEmpty(t, meta.Message, code)
	}
}

func TestAutomationSuccessCodes(t *testing.T) {
	cases := map[constants.ResponseCode]int{
		constants.AutomationSuccess.AutomationCreated:  http.StatusCreated,
		constants.AutomationSuccess.AutomationsFetched: http.StatusOK,
		constants.AutomationSuccess.AutomationUpdated:  http.StatusOK,
		constants.AutomationSuccess.AutomationRun:      http.StatusOK,
		constants.AutomationSuccess.AutomationDeleted:  http.StatusOK,
	}
	for code, status := range cases {
		meta, ok := constants.SuccessCodes[code]
		assert.True(t, ok, "%s is not registered in SuccessCodes", code)
		assert.Equal(t, status, meta.HTTPStatus, code)
	}
}

func TestAutomationErrorMapping(t *testing.T) {
	cases := map[error]constants.ResponseCode{
		app_errors.AutomationNotFound:   constants.AutomationError.AutomationNotFound,
		app_errors.InvalidTriggerQuery:  constants.AutomationError.InvalidTriggerQuery,
		app_errors.InvalidFunctionQuery: constants.AutomationError.InvalidFunctionQuery,
		app_errors.TriggerFailed:        constants.AutomationError.TriggerFailed,
		app_errors.TriggerHasFunction:   constants.AutomationError.TriggerHasFunction,
		app_errors.InvalidTriggerEvent:  constants.AutomationError.InvalidTriggerEvent,
	}
	for err, code := range cases {
		assert.Equal(t, code, constants.MapError(err), err.Error())
	}

	// A Postgres failure is wrapped with its message and still maps to TriggerFailed
	wrapped := fmt.Errorf("%w: %v", app_errors.TriggerFailed, "syntax error")
	assert.Equal(t, constants.AutomationError.TriggerFailed, constants.MapError(wrapped))
}
