// Copyright (c) 2026 Aptlogica Technologies Private Limited
// Licensed under the Apache License, Version 2.0
package models_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aptlogica/sereni-base/internal/utils/response"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupGinContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	return c, w
}

func TestSuccessResponseWithValidData(t *testing.T) {
	c, w := setupGinContext()

	testData := map[string]interface{}{
		"id":   "123",
		"name": "Test",
	}

	response.SendSuccess(c, "Success", testData)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Body.String())
}

func TestSuccessResponseWithNilData(t *testing.T) {
	c, w := setupGinContext()

	response.SendSuccess(c, "Success", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Body.String())
}

func TestSuccessResponseWithString(t *testing.T) {
	c, w := setupGinContext()

	response.SendSuccess(c, "Success", "test string")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Body.String())
}

func TestSuccessResponseWithArray(t *testing.T) {
	c, w := setupGinContext()

	testArray := []map[string]interface{}{
		{"id": "1", "value": "a"},
		{"id": "2", "value": "b"},
	}

	response.SendSuccess(c, "Success", testArray)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Body.String())
}

func TestErrorResponseWithMessage(t *testing.T) {
	c, w := setupGinContext()

	response.SendError(c, "Error message")

	assert.NotEqual(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Body.String())
}

func TestCheckAndSendErrorWithValidError(t *testing.T) {
	c, w := setupGinContext()

	err := errors.New("test error")
	response.CheckAndSendError(c, err)

	assert.NotEqual(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Body.String())
}

func TestCheckAndSendErrorWithNilError(t *testing.T) {
	c, _ := setupGinContext()

	assert.Panics(t, func() {
		response.CheckAndSendError(c, nil)
	})
}

func TestSuccessResponseStructure(t *testing.T) {
	c, w := setupGinContext()

	testData := map[string]string{"key": "value"}
	response.SendSuccess(c, "Test Success", testData)

	// Verify response contains message and data
	body := w.Body.String()
	assert.Contains(t, body, "message")
	assert.Contains(t, body, "data")
}

func TestErrorResponseStructure(t *testing.T) {
	c, w := setupGinContext()

	response.SendError(c, "Test Error")

	// Verify error response structure
	body := w.Body.String()
	assert.Contains(t, body, "error")
}

func TestMultipleSuccessResponses(t *testing.T) {
	c1, w1 := setupGinContext()
	response.SendSuccess(c1, "First", map[string]interface{}{"id": 1})

	c2, w2 := setupGinContext()
	response.SendSuccess(c2, "Second", map[string]interface{}{"id": 2})

	// Both should succeed independently
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestSuccessResponseWithNestedData(t *testing.T) {
	c, w := setupGinContext()

	nestedData := map[string]interface{}{
		"user": map[string]interface{}{
			"id":    "123",
			"name":  "John",
			"email": "john@example.com",
		},
		"metadata": map[string]interface{}{
			"timestamp": "2024-01-01",
			"version":   "1.0",
		},
	}

	response.SendSuccess(c, "Success", nestedData)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Body.String())
}

func TestResponseWithEmptyMessage(t *testing.T) {
	c, w := setupGinContext()

	response.SendSuccess(c, "", map[string]interface{}{})

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestResponseWithSpecialCharactersInData(t *testing.T) {
	c, w := setupGinContext()

	testData := map[string]interface{}{
		"message": "Hello \"World\" with 'quotes'",
		"symbol":  "©®™€",
	}

	response.SendSuccess(c, "Success", testData)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Body.String())
}

func TestResponseWithBooleanValues(t *testing.T) {
	c, w := setupGinContext()

	testData := map[string]interface{}{
		"success": true,
		"active":  false,
	}

	response.SendSuccess(c, "Success", testData)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "true")
	assert.Contains(t, body, "false")
}

func TestResponseWithNumericValues(t *testing.T) {
	c, w := setupGinContext()

	testData := map[string]interface{}{
		"integer":  42,
		"decimal":  3.14,
		"zero":     0,
		"negative": -10,
	}

	response.SendSuccess(c, "Success", testData)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Body.String())
}
