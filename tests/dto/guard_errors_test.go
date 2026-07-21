package tests

import (
	"testing"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/stretchr/testify/assert"
)

func TestGuardErrorError(t *testing.T) {
	t.Run("error returns message", func(t *testing.T) {
		ge := &dto.GuardError{
			Code:       "TEST_ERROR",
			Message:    "Test error message",
			StatusCode: 400,
		}

		assert.Equal(t, "Test error message", ge.Error())
	})

	t.Run("error implements error interface", func(t *testing.T) {
		ge := &dto.GuardError{
			Code:       "UNAUTHORIZED",
			Message:    "Unauthorized access",
			StatusCode: 401,
		}

		var err error = ge
		assert.NotNil(t, err)
		assert.Equal(t, "Unauthorized access", err.Error())
	})

	t.Run("error with details", func(t *testing.T) {
		ge := &dto.GuardError{
			Code:       "PERMISSION_DENIED",
			Message:    "Permission denied",
			StatusCode: 403,
			Details: map[string]interface{}{
				"resource": "base",
				"action":   "edit",
			},
		}

		assert.Equal(t, "Permission denied", ge.Error())
		assert.NotNil(t, ge.Details)
		assert.Equal(t, "base", ge.Details["resource"])
	})

	t.Run("error with empty message", func(t *testing.T) {
		ge := &dto.GuardError{
			Code:       "EMPTY_ERROR",
			Message:    "",
			StatusCode: 500,
		}

		assert.Equal(t, "", ge.Error())
	})

	t.Run("error with special characters in message", func(t *testing.T) {
		message := "Error: Invalid request\nDetails: Missing required field 'id'"
		ge := &dto.GuardError{
			Code:       "INVALID_REQUEST",
			Message:    message,
			StatusCode: 400,
		}

		assert.Equal(t, message, ge.Error())
	})
}

func TestGuardErrorFields(t *testing.T) {
	t.Run("guard error structure", func(t *testing.T) {
		ge := &dto.GuardError{
			Code:       "TEST_CODE",
			Message:    "Test message",
			StatusCode: 418,
			Details: map[string]interface{}{
				"extra": "data",
			},
		}

		assert.Equal(t, "TEST_CODE", ge.Code)
		assert.Equal(t, "Test message", ge.Message)
		assert.Equal(t, 418, ge.StatusCode)
		assert.NotNil(t, ge.Details)
	})

	t.Run("nil details", func(t *testing.T) {
		ge := &dto.GuardError{
			Code:       "NO_DETAILS",
			Message:    "No details",
			StatusCode: 400,
			Details:    nil,
		}

		assert.Nil(t, ge.Details)
		assert.Equal(t, "No details", ge.Error())
	})

	t.Run("empty details map", func(t *testing.T) {
		ge := &dto.GuardError{
			Code:       "EMPTY_DETAILS",
			Message:    "Empty details",
			StatusCode: 400,
			Details:    map[string]interface{}{},
		}

		assert.NotNil(t, ge.Details)
		assert.Empty(t, ge.Details)
	})
}
