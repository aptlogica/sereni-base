package tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	appErrors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/utils/response"
	responseConstants "github.com/aptlogica/sereni-base/internal/utils/response/constants"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSendError_Cases(t *testing.T) {
	tests := []struct {
		name         string
		code         responseConstants.ResponseCode
		expectedCode int
	}{
		{
			name:         "unauthorized access",
			code:         responseConstants.Error.UnauthorizedAccess,
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "invalid payload",
			code:         responseConstants.Error.InvalidPayload,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "internal error",
			code:         responseConstants.Error.InternalError,
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			response.SendError(c, tt.code)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, w.Code)
			}

			var resp response.StandardResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if resp.Success {
				t.Error("Expected success to be false")
			}

			if resp.Error == nil {
				t.Error("Expected error info to be present")
			}
		})
	}
}

func TestSendError_InvalidCode_Extended(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

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

	if resp.Error.Message != "Internal server error" {
		t.Errorf("Expected message 'Internal server error', got '%s'", resp.Error.Message)
	}
}

func TestSendErrorWithMessage_Cases(t *testing.T) {
	tests := []struct {
		name         string
		code         responseConstants.ResponseCode
		message      string
		expectedCode int
	}{
		{
			name:         "custom error message",
			code:         responseConstants.Error.InvalidPayload,
			message:      "Custom error message",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "another custom message",
			code:         responseConstants.Error.UnauthorizedAccess,
			message:      "Access denied for this resource",
			expectedCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			response.SendErrorWithMessage(c, tt.code, tt.message)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, w.Code)
			}

			var resp response.StandardResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if resp.Success {
				t.Error("Expected success to be false")
			}

			if resp.Error.Message != tt.message {
				t.Errorf("Expected message '%s', got '%s'", tt.message, resp.Error.Message)
			}
		})
	}
}

func TestSendErrorWithMessage_InvalidCode_Extended(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	invalidCode := responseConstants.ResponseCode("INVALID_CODE")
	response.SendErrorWithMessage(c, invalidCode, "custom message")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestCheckAndSendError_GenericError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	err := errors.New("some generic error")
	response.CheckAndSendError(c, err)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestCheckAndSendError_APIError_Extended(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	apiErr := &appErrors.APIError{
		Code:       "TEST_ERROR",
		Message:    "Test error message",
		StatusCode: http.StatusBadRequest,
		Details:    "Additional details",
	}

	response.CheckAndSendError(c, apiErr)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp response.StandardResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Error.Code != "TEST_ERROR" {
		t.Errorf("Expected code 'TEST_ERROR', got '%s'", resp.Error.Code)
	}

	if resp.Error.Message != "Test error message" {
		t.Errorf("Expected message 'Test error message', got '%s'", resp.Error.Message)
	}
}

func TestCheckAndSendError_APIErrorWithZeroStatus_Extended(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	apiErr := &appErrors.APIError{
		Code:       "TEST_ERROR",
		Message:    "Test error message",
		StatusCode: 0,
	}

	response.CheckAndSendError(c, apiErr)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestSendSuccess_WithRequestID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("request_id", "test-request-id")

	data := map[string]string{"key": "value"}
	response.SendSuccess(c, responseConstants.AuthSuccess.UserLogin, data)

	var resp response.StandardResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.RequestID != "test-request-id" {
		t.Errorf("Expected request_id 'test-request-id', got '%s'", resp.RequestID)
	}
}

func TestCreatedResponse_WithRequestID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("request_id", "test-request-id")

	response.CreatedResponse(c, "test data")

	var resp response.StandardResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.RequestID != "test-request-id" {
		t.Errorf("Expected request_id 'test-request-id', got '%s'", resp.RequestID)
	}
}

func TestSendError_WithRequestID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("request_id", "test-request-id")

	response.SendError(c, responseConstants.Error.InvalidPayload)

	var resp response.StandardResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.RequestID != "test-request-id" {
		t.Errorf("Expected request_id 'test-request-id', got '%s'", resp.RequestID)
	}
}

func TestFormatValidationError_NonValidationError(t *testing.T) {
	err := errors.New("regular error")
	result := response.FormatValidationError(err)

	if len(result) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result))
	}

	if result[0] != "regular error" {
		t.Errorf("Expected 'regular error', got '%s'", result[0])
	}
}

func TestStandardResponse_Structure(t *testing.T) {
	resp := response.StandardResponse{
		Success:   true,
		Message:   "Test message",
		Data:      map[string]string{"key": "value"},
		RequestID: "req-123",
		Meta: &response.MetaInfo{
			Code:       "TEST_CODE",
			HTTPStatus: 200,
		},
	}

	jsonBytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	var unmarshaled response.StandardResponse
	if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if unmarshaled.Success != resp.Success {
		t.Errorf("Success mismatch")
	}
	if unmarshaled.Message != resp.Message {
		t.Errorf("Message mismatch")
	}
	if unmarshaled.RequestID != resp.RequestID {
		t.Errorf("RequestID mismatch")
	}
}

func TestErrorInfo_Structure(t *testing.T) {
	errInfo := response.ErrorInfo{
		Code:    "ERR_001",
		Message: "Error message",
		Details: "Error details",
	}

	jsonBytes, err := json.Marshal(errInfo)
	if err != nil {
		t.Fatalf("Failed to marshal error info: %v", err)
	}

	var unmarshaled response.ErrorInfo
	if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal error info: %v", err)
	}

	if unmarshaled.Code != errInfo.Code {
		t.Errorf("Code mismatch")
	}
	if unmarshaled.Message != errInfo.Message {
		t.Errorf("Message mismatch")
	}
	if unmarshaled.Details != errInfo.Details {
		t.Errorf("Details mismatch")
	}
}

func TestMetaInfo_Structure(t *testing.T) {
	meta := response.MetaInfo{
		Code:       "META_001",
		HTTPStatus: 200,
	}

	jsonBytes, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("Failed to marshal meta info: %v", err)
	}

	var unmarshaled response.MetaInfo
	if err := json.Unmarshal(jsonBytes, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal meta info: %v", err)
	}

	if unmarshaled.Code != meta.Code {
		t.Errorf("Code mismatch")
	}
	if unmarshaled.HTTPStatus != meta.HTTPStatus {
		t.Errorf("HTTPStatus mismatch")
	}
}

func TestMultipleSendSuccess(t *testing.T) {
	successCodes := []responseConstants.ResponseCode{
		responseConstants.AuthSuccess.UserLogin,
		responseConstants.AuthSuccess.UserRegister,
		responseConstants.AuthSuccess.EmailVerified,
		responseConstants.AuthSuccess.ResendOTP,
		responseConstants.AuthSuccess.RefreshToken,
		responseConstants.AuthSuccess.ForgotPassword,
		responseConstants.AuthSuccess.ResetPassword,
	}

	for _, code := range successCodes {
		t.Run(fmt.Sprintf("success_code_%s", code), func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			response.SendSuccess(c, code, nil)

			var resp response.StandardResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if !resp.Success {
				t.Error("Expected success to be true")
			}
		})
	}
}
