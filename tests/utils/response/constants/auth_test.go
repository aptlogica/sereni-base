package tests

import (
	"net/http"
	"testing"

	"serenibase/internal/utils/response/constants"
)

func TestAuthErrorCodes(t *testing.T) {
	// Test that all AuthError fields are properly initialized
	authErrorFields := map[string]constants.ResponseCode{
		"FirstNameRequired":           constants.AuthError.FirstNameRequired,
		"LastNameRequired":            constants.AuthError.LastNameRequired,
		"EmailRequired":               constants.AuthError.EmailRequired,
		"PasswordRequired":            constants.AuthError.PasswordRequired,
		"InvalidOTP":                  constants.AuthError.InvalidOTP,
		"TokenExpired":                constants.AuthError.TokenExpired,
		"TokenUnauthorized":           constants.AuthError.TokenUnauthorized,
		"AuthProviderLoginFailed":     constants.AuthError.AuthProviderLoginFailed,
		"TokenAccessTokenSignFailed":  constants.AuthError.TokenAccessTokenSignFailed,
		"TokenRefreshTokenSignFailed": constants.AuthError.TokenRefreshTokenSignFailed,
	}

	// Test that all fields have non-empty values
	for fieldName, code := range authErrorFields {
		if code == "" {
			t.Errorf("AuthError.%s is empty", fieldName)
		}
		if string(code) == "" {
			t.Errorf("AuthError.%s string conversion is empty", fieldName)
		}
	}

	// Test that all auth error codes exist in ErrorCodes map
	for fieldName, code := range authErrorFields {
		if _, exists := constants.ErrorCodes[code]; !exists {
			t.Errorf("AuthError.%s code %s not found in ErrorCodes map", fieldName, code)
		}
	}
}

func TestAuthSuccessCodes(t *testing.T) {
	// Test that all AuthSuccess fields are properly initialized
	authSuccessFields := map[string]constants.ResponseCode{
		"UserRegister":   constants.AuthSuccess.UserRegister,
		"UserLogin":      constants.AuthSuccess.UserLogin,
		"UserLogout":     constants.AuthSuccess.UserLogout,
		"EmailVerified":  constants.AuthSuccess.EmailVerified,
		"RefreshToken":   constants.AuthSuccess.RefreshToken,
		"ForgotPassword": constants.AuthSuccess.ForgotPassword,
		"ResetPassword":  constants.AuthSuccess.ResetPassword,
		"ValidateToken":  constants.AuthSuccess.ValidateToken,
		"VerifyToken":    constants.AuthSuccess.VerifyToken,
	}

	// Test that all fields have non-empty values
	for fieldName, code := range authSuccessFields {
		if code == "" {
			t.Errorf("AuthSuccess.%s is empty", fieldName)
		}
		if string(code) == "" {
			t.Errorf("AuthSuccess.%s string conversion is empty", fieldName)
		}
	}

	// Test that all auth success codes exist in SuccessCodes map
	for fieldName, code := range authSuccessFields {
		if _, exists := constants.SuccessCodes[code]; !exists {
			t.Errorf("AuthSuccess.%s code %s not found in SuccessCodes map", fieldName, code)
		}
	}
}

func TestAuthErrorCodesMap(t *testing.T) {
	// Test that AuthErrorCodes map has expected entries
	expectedAuthErrorCodes := []constants.ResponseCode{
		"AUTH_VAL_1001", // FirstNameRequired
		"AUTH_VAL_1005", // EmailRequired
		"AUTH_VAL_1008", // PasswordRequired
		"AUTH_VAL_1013", // InvalidOTP
		"AUTH_VAL_1047", // RefreshTokenRequired
	}

	for _, code := range expectedAuthErrorCodes {
		if _, exists := constants.ErrorCodes[code]; !exists {
			t.Errorf("Expected auth error code %s not found in ErrorCodes", code)
		}
	}
}

func TestAuthSuccessCodesMap(t *testing.T) {
	// Test that AuthSuccessCodes map has expected entries
	expectedAuthSuccessCodes := []constants.ResponseCode{
		"AUTH_SUCCESS_1001", // UserRegister
		"AUTH_SUCCESS_1002", // UserLogin
		"AUTH_SUCCESS_1003", // EmailVerified
		"AUTH_SUCCESS_1005", // RefreshToken
		"AUTH_SUCCESS_1006", // ForgotPassword
		"AUTH_SUCCESS_1007", // ResetPassword
		"AUTH_SUCCESS_1008", // UserLogout
		"AUTH_SUCCESS_1009", // ValidateToken
		"AUTH_SUCCESS_1010", // VerifyToken
	}

	for _, code := range expectedAuthSuccessCodes {
		if _, exists := constants.SuccessCodes[code]; !exists {
			t.Errorf("Expected auth success code %s not found in SuccessCodes", code)
		}
	}
}

func TestAuthErrorCodesHTTPStatus(t *testing.T) {
	// Test that auth error codes have appropriate HTTP status codes
	testCases := []struct {
		code            constants.ResponseCode
		expectedStatus  int
		expectedMessage string
	}{
		{"AUTH_VAL_1001", http.StatusBadRequest, "Invalid request payload"},
		{"AUTH_VAL_1005", http.StatusBadRequest, "Invalid request payload"},
		{"AUTH_VAL_1008", http.StatusBadRequest, "Invalid request payload"},
		{"AUTH_VAL_1013", http.StatusBadRequest, "Invalid OTP"},
		{"AUTH_VAL_1047", http.StatusBadRequest, "Invalid request payload"},
	}

	for _, tc := range testCases {
		if meta, exists := constants.ErrorCodes[tc.code]; exists {
			if meta.HTTPStatus != tc.expectedStatus {
				t.Errorf("Auth error code %s has HTTP status %d, expected %d", tc.code, meta.HTTPStatus, tc.expectedStatus)
			}
			if meta.Message != tc.expectedMessage {
				t.Errorf("Auth error code %s has message '%s', expected '%s'", tc.code, meta.Message, tc.expectedMessage)
			}
		} else {
			t.Errorf("Auth error code %s not found in ErrorCodes", tc.code)
		}
	}
}

func TestAuthSuccessCodesHTTPStatus(t *testing.T) {
	// Test that auth success codes have appropriate HTTP status codes
	testCases := []struct {
		code            constants.ResponseCode
		expectedStatus  int
		expectedMessage string
	}{
		{"AUTH_SUCCESS_1001", http.StatusCreated, "User registered successfully"},
		{"AUTH_SUCCESS_1002", http.StatusOK, "Login successful"},
		{"AUTH_SUCCESS_1003", http.StatusOK, "Email verified successfully"},
		{"AUTH_SUCCESS_1005", http.StatusOK, "Token refreshed successfully"},
		{"AUTH_SUCCESS_1006", http.StatusOK, "Forgot password request successful"},
		{"AUTH_SUCCESS_1007", http.StatusOK, "Password reset successful"},
		{"AUTH_SUCCESS_1008", http.StatusOK, "Logout successful"},
		{"AUTH_SUCCESS_1009", http.StatusOK, "Token valid"},
		{"AUTH_SUCCESS_1010", http.StatusOK, "Token verified"},
	}

	for _, tc := range testCases {
		if meta, exists := constants.SuccessCodes[tc.code]; exists {
			if meta.HTTPStatus != tc.expectedStatus {
				t.Errorf("Auth success code %s has HTTP status %d, expected %d", tc.code, meta.HTTPStatus, tc.expectedStatus)
			}
			if meta.Message != tc.expectedMessage {
				t.Errorf("Auth success code %s has message '%s', expected '%s'", tc.code, meta.Message, tc.expectedMessage)
			}
		} else {
			t.Errorf("Auth success code %s not found in SuccessCodes", tc.code)
		}
	}
}

func TestAuthErrorCodePatterns(t *testing.T) {
	// Test that auth error codes follow expected patterns
	for code := range constants.ErrorCodes {
		if len(string(code)) > 0 && string(code)[:5] == "AUTH_" {
			// This is an auth-related code, test it has proper structure
			if len(string(code)) < 8 {
				t.Errorf("Auth error code %s is too short", code)
			}
		}
	}
}

func TestAuthSuccessCodePatterns(t *testing.T) {
	// Test that auth success codes follow expected patterns
	for code := range constants.SuccessCodes {
		if len(string(code)) > 0 && string(code)[:12] == "AUTH_SUCCESS" {
			// This is an auth success code, test it has proper structure
			if len(string(code)) < 15 {
				t.Errorf("Auth success code %s is too short", code)
			}
		}
	}
}
