package tests

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	appErrors "serenibase/internal/app-errors"
	"serenibase/internal/utils/response"
	responseConstants "serenibase/internal/utils/response/constants"
)

func TestCreatedResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]string{"key": "value"}
	response.CreatedResponse(c, data)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var resp response.StandardResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success to be true")
	}
	if resp.Message != "Resource created successfully" {
		t.Errorf("Expected message 'Resource created successfully', got '%s'", resp.Message)
	}
	if resp.Data.(map[string]interface{})["key"] != "value" {
		t.Error("Expected data to contain the provided data")
	}
}

func TestCreatedResponseWithCustomMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := "test data"
	response.CreatedResponse(c, data, "Custom message")

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var resp response.StandardResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Message != "Custom message" {
		t.Errorf("Expected message 'Custom message', got '%s'", resp.Message)
	}
}

func TestSendSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := "success data"
	response.SendSuccess(c, responseConstants.AuthSuccess.UserRegister, data)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var resp response.StandardResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success to be true")
	}
	if resp.Data != "success data" {
		t.Errorf("Expected data 'success data', got '%v'", resp.Data)
	}
	if resp.Meta.Code != string(responseConstants.AuthSuccess.UserRegister) {
		t.Errorf("Expected meta code '%s', got '%s'", responseConstants.AuthSuccess.UserRegister, resp.Meta.Code)
	}
}

func TestSendSuccess_InvalidCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := "success data"
	invalidCode := responseConstants.ResponseCode("INVALID_SUCCESS_CODE")
	response.SendSuccess(c, invalidCode, data)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp response.StandardResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success to be true")
	}
	if resp.Message != "success" {
		t.Errorf("Expected message 'success', got '%s'", resp.Message)
	}
	if resp.Data != "success data" {
		t.Errorf("Expected data 'success data', got '%v'", resp.Data)
	}
	if resp.Meta.Code != string(invalidCode) {
		t.Errorf("Expected meta code '%s', got '%s'", invalidCode, resp.Meta.Code)
	}
}

func TestFormatValidationError_EmptyErrors(t *testing.T) {
	// Create a mock validation error
	ve := validator.ValidationErrors{}
	err := ve

	result := response.FormatValidationError(err)

	if len(result) != 0 {
		t.Errorf("Expected empty slice, got %v", result)
	}
}

func TestFormatValidationError_WithErrors(t *testing.T) {
	// Create a mock validation error
	err := errors.New("some error")

	result := response.FormatValidationError(err)

	expected := []string{"some error"}
	if len(result) != len(expected) || result[0] != expected[0] {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestSendError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	response.SendError(c, responseConstants.Error.InvalidCredentials)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	var resp response.StandardResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Success {
		t.Error("Expected success to be false")
	}
	if resp.Error.Code != string(responseConstants.Error.InvalidCredentials) {
		t.Errorf("Expected error code '%s', got '%s'", responseConstants.Error.InvalidCredentials, resp.Error.Code)
	}
}

func TestSendError_InvalidCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Use an invalid response code
	invalidCode := responseConstants.ResponseCode("INVALID_CODE")
	response.SendError(c, invalidCode)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var resp response.StandardResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Success {
		t.Error("Expected success to be false")
	}
	if resp.Error.Code != string(responseConstants.Error.InternalError) {
		t.Errorf("Expected error code '%s', got '%s'", responseConstants.Error.InternalError, resp.Error.Code)
	}
	if resp.Error.Message != "Internal server error" {
		t.Errorf("Expected error message 'Internal server error', got '%s'", resp.Error.Message)
	}
}

func TestSendErrorWithMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	customMessage := "Custom error message"
	response.SendErrorWithMessage(c, responseConstants.Error.InvalidPayload, customMessage)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp response.StandardResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Success {
		t.Error("Expected success to be false")
	}
	if resp.Error.Message != customMessage {
		t.Errorf("Expected error message '%s', got '%s'", customMessage, resp.Error.Message)
	}
}

func TestCheckAndSendError_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a mock validation error
	ve := validator.ValidationErrors{}
	err := error(ve)
	response.CheckAndSendError(c, err)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Expected status %d, got %d", http.StatusUnprocessableEntity, w.Code)
	}

	var resp response.StandardResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Success {
		t.Error("Expected success to be false")
	}
	if resp.Error.Code != string(responseConstants.Error.ValidationFailed) {
		t.Errorf("Expected error code '%s', got '%s'", responseConstants.Error.ValidationFailed, resp.Error.Code)
	}
}

func TestCheckAndSendError_APIError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	apiErr := &appErrors.APIError{
		Code:       "TEST_ERROR",
		Message:    "Test API error",
		Details:    "Test details",
		StatusCode: http.StatusBadGateway,
	}
	response.CheckAndSendError(c, apiErr)

	if w.Code != http.StatusBadGateway {
		t.Errorf("Expected status %d, got %d", http.StatusBadGateway, w.Code)
	}

	var resp response.StandardResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Success {
		t.Error("Expected success to be false")
	}
	if resp.Error.Code != "TEST_ERROR" {
		t.Errorf("Expected error code 'TEST_ERROR', got '%s'", resp.Error.Code)
	}
	if resp.Error.Message != "Test API error" {
		t.Errorf("Expected error message 'Test API error', got '%s'", resp.Error.Message)
	}
}

func TestCheckAndSendError_MappedError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	err := appErrors.DatabaseError
	response.CheckAndSendError(c, err)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var resp response.StandardResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Success {
		t.Error("Expected success to be false")
	}
	if resp.Error.Code != string(responseConstants.Error.DatabaseError) {
		t.Errorf("Expected error code '%s', got '%s'", responseConstants.Error.DatabaseError, resp.Error.Code)
	}
}

func TestCheckAndSendError_UnmappedError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	err := errors.New("unmapped error")
	response.CheckAndSendError(c, err)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var resp response.StandardResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Success {
		t.Error("Expected success to be false")
	}
	if resp.Error.Message != "unmapped error" {
		t.Errorf("Expected error message 'unmapped error', got '%s'", resp.Error.Message)
	}
}

func TestFormatValidationError(t *testing.T) {
	// Mock validation error
	ve := validator.ValidationErrors{}
	err := error(ve)
	result := response.FormatValidationError(err)

	if len(result) != 0 {
		t.Errorf("Expected empty slice for ValidationErrors, got %v", result)
	}

	// Test with non-validation error
	err = errors.New("non-validation error")
	result = response.FormatValidationError(err)

	if len(result) != 1 {
		t.Errorf("Expected slice with 1 element, got %d", len(result))
	}
	if result[0] != "non-validation error" {
		t.Errorf("Expected 'non-validation error', got '%s'", result[0])
	}
}

// TestStandardResponse tests the StandardResponse structure
func TestStandardResponse(t *testing.T) {
	tests := []struct {
		name     string
		response response.StandardResponse
	}{
		{
			name: "success response with data",
			response: response.StandardResponse{
				Success: true,
				Message: "Test message",
				Data:    map[string]string{"key": "value"},
				Meta: &response.MetaInfo{
					Code:       "TEST_CODE",
					HTTPStatus: 200,
				},
			},
		},
		{
			name: "error response",
			response: response.StandardResponse{
				Success: false,
				Error: &response.ErrorInfo{
					Code:    "ERROR_CODE",
					Message: "Error message",
					Details: "Error details",
				},
				Meta: &response.MetaInfo{
					Code:       "ERROR_CODE",
					HTTPStatus: 400,
				},
			},
		},
		{
			name: "minimal response",
			response: response.StandardResponse{
				Success: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the structure can be marshaled to JSON
			_, err := json.Marshal(tt.response)
			if err != nil {
				t.Errorf("Failed to marshal StandardResponse: %v", err)
			}
		})
	}
}

// TestErrorInfo tests the ErrorInfo structure
func TestErrorInfo(t *testing.T) {
	errorInfo := response.ErrorInfo{
		Code:    "TEST_ERROR",
		Message: "Test error message",
		Details: "Additional details",
	}

	if errorInfo.Code != "TEST_ERROR" {
		t.Errorf("Expected code 'TEST_ERROR', got '%s'", errorInfo.Code)
	}
	if errorInfo.Message != "Test error message" {
		t.Errorf("Expected message 'Test error message', got '%s'", errorInfo.Message)
	}
	if errorInfo.Details != "Additional details" {
		t.Errorf("Expected details 'Additional details', got '%s'", errorInfo.Details)
	}

	// Test JSON marshaling
	_, err := json.Marshal(errorInfo)
	if err != nil {
		t.Errorf("Failed to marshal ErrorInfo: %v", err)
	}
}

// TestMetaInfo tests the MetaInfo structure
func TestMetaInfo(t *testing.T) {
	metaInfo := response.MetaInfo{
		Code:       "TEST_META",
		HTTPStatus: 201,
	}

	if metaInfo.Code != "TEST_META" {
		t.Errorf("Expected code 'TEST_META', got '%s'", metaInfo.Code)
	}
	if metaInfo.HTTPStatus != 201 {
		t.Errorf("Expected HTTPStatus 201, got %d", metaInfo.HTTPStatus)
	}

	// Test JSON marshaling
	_, err := json.Marshal(metaInfo)
	if err != nil {
		t.Errorf("Failed to marshal MetaInfo: %v", err)
	}
}

// TestSendSuccess_MultipleCodes tests SendSuccess with various success codes
func TestSendSuccess_MultipleCodes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		code responseConstants.ResponseCode
	}{
		{"UserRegister", responseConstants.AuthSuccess.UserRegister},
		{"UserLogin", responseConstants.AuthSuccess.UserLogin},
		{"UserLogout", responseConstants.AuthSuccess.UserLogout},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			response.SendSuccess(c, tt.code, "test data")

			var resp response.StandardResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if !resp.Success {
				t.Error("Expected success to be true")
			}
			if resp.Meta == nil {
				t.Fatal("Expected meta to be non-nil")
			}
			if resp.Meta.Code != string(tt.code) {
				t.Errorf("Expected code '%s', got '%s'", tt.code, resp.Meta.Code)
			}
		})
	}
}

// TestSendError_MultipleCodes tests SendError with various error codes
func TestSendError_MultipleCodes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		code responseConstants.ResponseCode
	}{
		{"InvalidCredentials", responseConstants.Error.InvalidCredentials},
		{"InvalidPayload", responseConstants.Error.InvalidPayload},
		{"UnauthorizedAccess", responseConstants.Error.UnauthorizedAccess},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			response.SendError(c, tt.code)

			var resp response.StandardResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if resp.Success {
				t.Error("Expected success to be false")
			}
			if resp.Error == nil {
				t.Fatal("Expected error to be non-nil")
			}
			if resp.Error.Code != string(tt.code) {
				t.Errorf("Expected code '%s', got '%s'", tt.code, resp.Error.Code)
			}
		})
	}
}

// TestCheckAndSendError_NilError tests CheckAndSendError with nil (should not be called, but good to verify behavior)
func TestCheckAndSendError_WithDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create an APIError with details
	apiErr := &appErrors.APIError{
		Code:       "DETAILED_ERROR",
		Message:    "Error with details",
		Details:    map[string]string{"field": "value"},
		StatusCode: http.StatusBadRequest,
	}

	response.CheckAndSendError(c, apiErr)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp response.StandardResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Success {
		t.Error("Expected success to be false")
	}
	if resp.Error.Details == "" {
		t.Error("Expected error details to be present")
	}
}

// TestCreatedResponse_NilData tests CreatedResponse with nil data
func TestCreatedResponse_NilData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	response.CreatedResponse(c, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var resp response.StandardResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success to be true")
	}
}

// TestSendSuccess_NilData tests SendSuccess with nil data
func TestSendSuccess_NilData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	response.SendSuccess(c, responseConstants.AuthSuccess.UserLogout, nil)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp response.StandardResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !resp.Success {
		t.Error("Expected success to be true")
	}
}

// TestSendErrorWithMessage_EmptyMessage tests SendErrorWithMessage with empty message
func TestSendErrorWithMessage_EmptyMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	response.SendErrorWithMessage(c, responseConstants.Error.InvalidPayload, "")

	var resp response.StandardResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Success {
		t.Error("Expected success to be false")
	}
	if resp.Error.Message != "" {
		t.Error("Expected empty error message")
	}
}

// TestSendErrorWithMessage_InvalidCode tests SendErrorWithMessage with invalid code
func TestSendErrorWithMessage_InvalidCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	invalidCode := responseConstants.ResponseCode("TOTALLY_INVALID_CODE")
	response.SendErrorWithMessage(c, invalidCode, "custom message")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var resp response.StandardResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Success {
		t.Error("Expected success to be false")
	}
	if resp.Error.Message != "custom message" {
		t.Errorf("Expected message 'custom message', got '%s'", resp.Error.Message)
	}
}

// TestCheckAndSendError_APIErrorWithZeroStatus tests APIError with status 0
func TestCheckAndSendError_APIErrorWithZeroStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	apiErr := &appErrors.APIError{
		Code:       "TEST_ERROR",
		Message:    "Test message",
		StatusCode: 0, // Zero status should default to 400
	}

	response.CheckAndSendError(c, apiErr)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d (default), got %d", http.StatusBadRequest, w.Code)
	}
}

// TestFormatValidationError_WithActualValidationErrors tests with real validation errors
func TestFormatValidationError_WithActualValidationErrors(t *testing.T) {
	type TestStruct struct {
		Email string `validate:"required,email"`
		Age   int    `validate:"required,min=18"`
	}

	validate := validator.New()
	testData := TestStruct{} // Empty struct to trigger validation errors

	err := validate.Struct(testData)
	if err == nil {
		t.Fatal("Expected validation error")
	}

	result := response.FormatValidationError(err)
	if len(result) == 0 {
		t.Error("Expected validation error messages")
	}

	// Check that messages contain field names
	hasEmail := false
	hasAge := false
	for _, msg := range result {
		if len(msg) > 0 {
			if msg != "" {
				hasEmail = hasEmail || true
				hasAge = hasAge || true
			}
		}
	}

	if len(result) < 2 {
		t.Errorf("Expected at least 2 validation errors, got %d", len(result))
	}
}
