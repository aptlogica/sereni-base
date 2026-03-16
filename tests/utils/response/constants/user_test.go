package tests

import (
	"net/http"
	"testing"

	"github.com/aptlogica/sereni-base/internal/utils/response/constants"
)

func TestUserErrorCodes(t *testing.T) {
	// Test that all UserError fields are properly initialized
	userErrorFields := map[string]constants.ResponseCode{
		"ErrNotFound":          constants.UserError.ErrNotFound,
		"UserAlreadyExists":    constants.UserError.UserAlreadyExists,
		"UserNotCreated":       constants.UserError.UserNotCreated,
		"UserNotUpdated":       constants.UserError.UserNotUpdated,
		"UserNotDeleted":       constants.UserError.UserNotDeleted,
		"EmailAlreadyVerified": constants.UserError.EmailAlreadyVerified,
		"InvalidOldPassword":   constants.UserError.InvalidOldPassword,
		"NewPasswordRequired":  constants.UserError.NewPasswordRequired,
		"EmailRequired":        constants.UserError.EmailRequired,
		"FirstNameRequired":    constants.UserError.FirstNameRequired,
		"UserIDRequired":       constants.UserError.UserIDRequired,
		"UserIDInvalid":        constants.UserError.UserIDInvalid,
	}

	// Test that all fields have non-empty values
	for fieldName, code := range userErrorFields {
		if code == "" {
			t.Errorf("UserError.%s is empty", fieldName)
		}
		if string(code) == "" {
			t.Errorf("UserError.%s string conversion is empty", fieldName)
		}
	}

	// Test that all user error codes exist in ErrorCodes map
	for fieldName, code := range userErrorFields {
		if _, exists := constants.ErrorCodes[code]; !exists {
			t.Errorf("UserError.%s code %s not found in ErrorCodes map", fieldName, code)
		}
	}
}

func TestUserSuccessCodes(t *testing.T) {
	// Test that all UserSuccess fields are properly initialized
	userSuccessFields := map[string]constants.ResponseCode{
		"UserCreated":              constants.UserSuccess.UserCreated,
		"UserUpdated":              constants.UserSuccess.UserUpdated,
		"UserDeleted":              constants.UserSuccess.UserDeleted,
		"UserFetched":              constants.UserSuccess.UserFetched,
		"PasswordUpdated":          constants.UserSuccess.PasswordUpdated,
		"AvatarAdded":              constants.UserSuccess.AvatarAdded,
		"AvatarRemoved":            constants.UserSuccess.AvatarRemoved,
		"UserAdded":                constants.UserSuccess.UserAdded,
		"UserRemoved":              constants.UserSuccess.UserRemoved,
		"UsersFetched":             constants.UserSuccess.UsersFetched,
		"UserAssignedToWorkspace":  constants.UserSuccess.UserAssignedToWorkspace,
		"WorkspaceFetched":         constants.UserSuccess.WorkspaceFetched,
		"UserRemovedFromWorkspace": constants.UserSuccess.UserRemovedFromWorkspace,
		"UserAccessDetailsFetched": constants.UserSuccess.UserAccessDetailsFetched,
	}

	// Test that all fields have non-empty values
	for fieldName, code := range userSuccessFields {
		if code == "" {
			t.Errorf("UserSuccess.%s is empty", fieldName)
		}
		if string(code) == "" {
			t.Errorf("UserSuccess.%s string conversion is empty", fieldName)
		}
	}

	// Test that all user success codes exist in SuccessCodes map
	for fieldName, code := range userSuccessFields {
		if _, exists := constants.SuccessCodes[code]; !exists {
			t.Errorf("UserSuccess.%s code %s not found in SuccessCodes map", fieldName, code)
		}
	}
}

func TestUserErrorCodesMap(t *testing.T) {
	// Test that UserErrorCodes map has expected entries
	expectedUserErrorCodes := []constants.ResponseCode{
		"USR_2006", // ErrNotFound
		"USR_2005", // UserAlreadyExists
		"USR_2007", // UserNotCreated
		"USR_2008", // UserNotUpdated
		"USR_2009", // UserNotDeleted
		"USR_2010", // EmailAlreadyVerified
		"USR_2011", // InvalidOldPassword
		"USR_2014", // NewPasswordRequired
		"USR_2016", // EmailRequired
		"USR_2018", // FirstNameRequired
		"USR_2024", // UserIDRequired
		"USR_2025", // UserIDInvalid
	}

	for _, code := range expectedUserErrorCodes {
		if _, exists := constants.ErrorCodes[code]; !exists {
			t.Errorf("Expected user error code %s not found in ErrorCodes", code)
		}
	}
}

func TestUserSuccessCodesMap(t *testing.T) {
	// Test that UserSuccessCodes map has expected entries
	expectedUserSuccessCodes := []constants.ResponseCode{
		"USR_SUCCESS_2001", // UserCreated
		"USR_SUCCESS_2002", // UserUpdated
		"USR_SUCCESS_2003", // UserDeleted
		"USR_SUCCESS_2004", // UserFetched
		"USR_SUCCESS_2005", // PasswordUpdated
		"USR_SUCCESS_2006", // AvatarAdded
		"USR_SUCCESS_2007", // AvatarRemoved
		"USR_SUCCESS_2008", // UserAdded
		"USR_SUCCESS_2009", // UserRemoved
		"USR_SUCCESS_2010", // UsersFetched
		"USR_SUCCESS_2011", // UserAssignedToWorkspace
		"USR_SUCCESS_2012", // WorkspaceFetched
		"USR_SUCCESS_2013", // UserRemovedFromWorkspace
		"USR_SUCCESS_2014", // UserAccessDetailsFetched
	}

	for _, code := range expectedUserSuccessCodes {
		if _, exists := constants.SuccessCodes[code]; !exists {
			t.Errorf("Expected user success code %s not found in SuccessCodes", code)
		}
	}
}

func TestUserErrorCodesHTTPStatus(t *testing.T) {
	// Test that user error codes have appropriate HTTP status codes
	testCases := []struct {
		code            constants.ResponseCode
		expectedStatus  int
		expectedMessage string
	}{
		{"USR_2006", http.StatusNotFound, "User not found"},
		{"USR_2005", http.StatusConflict, "User already exists"},
		{"USR_2007", http.StatusInternalServerError, "User not created"},
		{"USR_2008", http.StatusInternalServerError, "User not updated"},
		{"USR_2009", http.StatusInternalServerError, "User not deleted"},
		{"USR_2010", http.StatusConflict, "Email already verified"},
		{"USR_2011", http.StatusUnauthorized, "Invalid old password"},
		{"USR_2014", http.StatusBadRequest, "New password is required"},
		{"USR_2016", http.StatusBadRequest, "Email is required"},
		{"USR_2018", http.StatusBadRequest, "First name is required"},
		{"USR_2024", http.StatusBadRequest, "User ID is required"},
		{"USR_2025", http.StatusBadRequest, "Invalid user ID"},
	}

	for _, tc := range testCases {
		if meta, exists := constants.ErrorCodes[tc.code]; exists {
			if meta.HTTPStatus != tc.expectedStatus {
				t.Errorf("User error code %s has HTTP status %d, expected %d", tc.code, meta.HTTPStatus, tc.expectedStatus)
			}
			if meta.Message != tc.expectedMessage {
				t.Errorf("User error code %s has message '%s', expected '%s'", tc.code, meta.Message, tc.expectedMessage)
			}
		} else {
			t.Errorf("User error code %s not found in ErrorCodes", tc.code)
		}
	}
}

func TestUserSuccessCodesHTTPStatus(t *testing.T) {
	// Test that user success codes have appropriate HTTP status codes
	testCases := []struct {
		code            constants.ResponseCode
		expectedStatus  int
		expectedMessage string
	}{
		{"USR_SUCCESS_2001", http.StatusCreated, "User created successfully"},
		{"USR_SUCCESS_2002", http.StatusOK, "User updated successfully"},
		{"USR_SUCCESS_2003", http.StatusOK, "User deleted successfully"},
		{"USR_SUCCESS_2004", http.StatusOK, "User fetched successfully"},
		{"USR_SUCCESS_2005", http.StatusOK, "Password updated successfully"},
		{"USR_SUCCESS_2006", http.StatusOK, "Avatar added successfully"},
		{"USR_SUCCESS_2007", http.StatusOK, "Avatar removed successfully"},
		{"USR_SUCCESS_2008", http.StatusCreated, "User added successfully"},
		{"USR_SUCCESS_2009", http.StatusOK, "User removed successfully"},
		{"USR_SUCCESS_2010", http.StatusOK, "Users fetched successfully"},
		{"USR_SUCCESS_2011", http.StatusCreated, "User assigned to workspace successfully"},
		{"USR_SUCCESS_2012", http.StatusOK, "Workspaces fetched successfully"},
		{"USR_SUCCESS_2013", http.StatusOK, "User removed from workspace successfully"},
		{"USR_SUCCESS_2014", http.StatusOK, "User access details fetched successfully"},
	}

	for _, tc := range testCases {
		if meta, exists := constants.SuccessCodes[tc.code]; exists {
			if meta.HTTPStatus != tc.expectedStatus {
				t.Errorf("User success code %s has HTTP status %d, expected %d", tc.code, meta.HTTPStatus, tc.expectedStatus)
			}
			if meta.Message != tc.expectedMessage {
				t.Errorf("User success code %s has message '%s', expected '%s'", tc.code, meta.Message, tc.expectedMessage)
			}
		} else {
			t.Errorf("User success code %s not found in SuccessCodes", tc.code)
		}
	}
}

func TestUserErrorCodePatterns(t *testing.T) {
	// Test that user error codes follow expected patterns
	for code := range constants.ErrorCodes {
		if len(string(code)) > 0 && string(code)[:4] == "USR_" && string(code)[4:8] != "SUCCESS" {
			// This is a user error code, test it has proper structure
			if len(string(code)) < 8 {
				t.Errorf("User error code %s is too short", code)
			}
		}
	}
}

func TestUserSuccessCodePatterns(t *testing.T) {
	// Test that user success codes follow expected patterns
	for code := range constants.SuccessCodes {
		if len(string(code)) > 0 && string(code)[:12] == "USR_SUCCESS_" {
			// This is a user success code, test it has proper structure
			if len(string(code)) < 15 {
				t.Errorf("User success code %s is too short", code)
			}
		}
	}
}
