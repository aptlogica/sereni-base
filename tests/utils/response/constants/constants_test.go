package tests

import (
	"errors"
	"net/http"
	"testing"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/utils/response/constants"
)

// TestErrorStruct tests that the Error struct is properly defined with all fields
func TestErrorStruct(t *testing.T) {
	errorTests := []struct {
		name     string
		code     constants.ResponseCode
		expected string
	}{
		{"InvalidID", constants.Error.InvalidID, "ERR_0001"},
		{"UnauthorizedAccess", constants.Error.UnauthorizedAccess, "ERR_0002"},
		{"Forbidden", constants.Error.Forbidden, "ERR_0003"},
		{"SessionExpired", constants.Error.SessionExpired, "ERR_0004"},
		{"InvalidPayload", constants.Error.InvalidPayload, "ERR_0005"},
		{"ValidationFailed", constants.Error.ValidationFailed, "ERR_0006"},
		{"DatabaseError", constants.Error.DatabaseError, "ERR_0007"},
		{"RecordNotFound", constants.Error.RecordNotFound, "ERR_0008"},
		{"InternalError", constants.Error.InternalError, "ERR_0009"},
		{"ServiceUnavailable", constants.Error.ServiceUnavailable, "ERR_0010"},
		{"GatewayTimeout", constants.Error.GatewayTimeout, "ERR_0011"},
		{"TooManyRequests", constants.Error.TooManyRequests, "ERR_0012"},
		{"UserAlreadyExists", constants.Error.UserAlreadyExists, "ERR_0013"},
		{"RecordAlreadyExists", constants.Error.RecordAlreadyExists, "ERR_0014"},
		{"RecordNotInserted", constants.Error.RecordNotInserted, "ERR_0015"},
		{"BadRequest", constants.Error.BadRequest, "ERR_0016"},
		{"Conflict", constants.Error.Conflict, "ERR_0017"},
		{"NotImplemented", constants.Error.NotImplemented, "ERR_0018"},
		{"Timeout", constants.Error.Timeout, "ERR_0019"},
		{"DependencyFailed", constants.Error.DependencyFailed, "ERR_0020"},
		{"MapToStructError", constants.Error.MapToStructError, "ERR_0021"},
		{"StructToStructError", constants.Error.StructToStructError, "ERR_0022"},
		{"HashingError", constants.Error.HashingError, "ERR_0023"},
		{"InvalidCredentials", constants.Error.InvalidCredentials, "ERR_0024"},
		{"InvalidDriver", constants.Error.InvalidDriver, "ERR_0025"},
		{"ErrNotFound", constants.Error.ErrNotFound, "ERR_0026"},
		{"JSONMarshalError", constants.Error.JSONMarshalError, "ERR_0027"},
		{"HTTPRequestCreationError", constants.Error.HTTPRequestCreationError, "ERR_0028"},
		{"HTTPDoRequestError", constants.Error.HTTPDoRequestError, "ERR_0029"},
		{"UserNotActive", constants.Error.UserNotActive, "ERR_0030"},
		{"FileNotFound", constants.Error.FileNotFound, "ERR_1001"},
		{"FileAlreadyExists", constants.Error.FileAlreadyExists, "ERR_1002"},
		{"FileReadFailed", constants.Error.FileReadFailed, "ERR_1003"},
		{"FileWriteFailed", constants.Error.FileWriteFailed, "ERR_1004"},
		{"FileDeleteFailed", constants.Error.FileDeleteFailed, "ERR_1005"},
		{"FilePermissionDenied", constants.Error.FilePermissionDenied, "ERR_1006"},
		{"FileInvalidPath", constants.Error.FileInvalidPath, "ERR_1007"},
		{"FolderNotFound", constants.Error.FolderNotFound, "ERR_1008"},
		{"FolderAlreadyExists", constants.Error.FolderAlreadyExists, "ERR_1009"},
		{"FolderCreateFailed", constants.Error.FolderCreateFailed, "ERR_1010"},
		{"FolderDeleteFailed", constants.Error.FolderDeleteFailed, "ERR_1011"},
		{"FolderPermissionDenied", constants.Error.FolderPermissionDenied, "ERR_1012"},
		{"FolderInvalidPath", constants.Error.FolderInvalidPath, "ERR_1013"},
		{"InvalidDateOfBirth", constants.Error.InvalidDateOfBirth, "ERR_1014"},
		{"RoleCreationError", constants.Error.RoleCreationError, "ERR_1015"},
		{"RoleNotFound", constants.Error.RoleNotFound, "ERR_1017"},
		{"UserDisableFailed", constants.Error.UserDisableFailed, "ERR_1018"},
		{"InvalidWorkspaceMemberData", constants.Error.InvalidWorkspaceMemberData, "ERR_1019"},
		{"UserContextNotFound", constants.Error.UserContextNotFound, "ERR_1020"},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.code) != tt.expected {
				t.Errorf("%s: expected code %s, got %s", tt.name, tt.expected, string(tt.code))
			}
		})
	}
}

// TestCoreErrorCodesMap tests that CoreErrorCodes map is properly initialized
func TestCoreErrorCodesMap(t *testing.T) {
	if constants.CoreErrorCodes == nil {
		t.Fatal("CoreErrorCodes map should not be nil")
	}

	// Test specific error code mappings
	errorMappingTests := []struct {
		name       string
		code       constants.ResponseCode
		httpStatus int
	}{
		{"UnauthorizedAccess", constants.Error.UnauthorizedAccess, http.StatusUnauthorized},
		{"Forbidden", constants.Error.Forbidden, http.StatusForbidden},
		{"RecordNotFound", constants.Error.RecordNotFound, http.StatusNotFound},
		{"BadRequest", constants.Error.BadRequest, http.StatusBadRequest},
		{"InternalError", constants.Error.InternalError, http.StatusInternalServerError},
		{"Conflict", constants.Error.Conflict, http.StatusConflict},
		{"ValidationFailed", constants.Error.ValidationFailed, http.StatusUnprocessableEntity},
		{"TooManyRequests", constants.Error.TooManyRequests, http.StatusTooManyRequests},
	}

	for _, tt := range errorMappingTests {
		t.Run(tt.name, func(t *testing.T) {
			meta, exists := constants.CoreErrorCodes[tt.code]
			if !exists {
				t.Errorf("%s not found in CoreErrorCodes map", tt.name)
				return
			}
			if meta.HTTPStatus != tt.httpStatus {
				t.Errorf("%s: expected HTTP status %d, got %d", tt.name, tt.httpStatus, meta.HTTPStatus)
			}
			if meta.Message == "" {
				t.Errorf("%s: message should not be empty", tt.name)
			}
		})
	}
}

// TestErrorCodesMap tests that the ErrorCodes merged map is properly initialized
func TestErrorCodesMap(t *testing.T) {
	if constants.ErrorCodes == nil {
		t.Fatal("ErrorCodes map should not be nil")
	}

	if len(constants.ErrorCodes) == 0 {
		t.Error("ErrorCodes map should have entries")
	}

	// Verify that core error codes are present in the merged map
	if _, exists := constants.ErrorCodes[constants.Error.UnauthorizedAccess]; !exists {
		t.Error("UnauthorizedAccess not found in merged ErrorCodes map")
	}
	if _, exists := constants.ErrorCodes[constants.Error.RecordNotFound]; !exists {
		t.Error("RecordNotFound not found in merged ErrorCodes map")
	}
}

// TestSuccessCodesMap tests that the SuccessCodes merged map is properly initialized
func TestSuccessCodesMap(t *testing.T) {
	if constants.SuccessCodes == nil {
		t.Fatal("SuccessCodes map should not be nil")
	}

	if len(constants.SuccessCodes) == 0 {
		t.Error("SuccessCodes map should have entries")
	}
}

// TestResponseCodeType tests that ResponseCode type works correctly
func TestResponseCodeType(t *testing.T) {
	var code constants.ResponseCode = "TEST_001"

	if string(code) != "TEST_001" {
		t.Errorf("ResponseCode string conversion failed: expected TEST_001, got %s", string(code))
	}
}

// TestMetaResponse tests MetaResponse struct
func TestMetaResponse(t *testing.T) {
	meta, exists := constants.CoreErrorCodes[constants.Error.BadRequest]
	if !exists {
		t.Fatal("BadRequest should exist in CoreErrorCodes")
	}

	if meta.HTTPStatus == 0 {
		t.Error("HTTPStatus should be set")
	}
	if meta.Message == "" {
		t.Error("Message should not be empty")
	}
	if meta.Description == "" {
		t.Error("Description should not be empty")
	}
}

func TestMapError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected constants.ResponseCode
	}{
		{"DatabaseError", app_errors.DatabaseError, constants.Error.DatabaseError},
		{"InvalidCredentials", app_errors.InvalidCredentials, constants.Error.InvalidCredentials},
		{"UnmappedError", errors.New("unmapped"), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := constants.MapError(tt.err)
			if result != tt.expected {
				t.Errorf("MapError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestAllErrorMappings(t *testing.T) {
	mappings := constants.AllErrorMappings()
	if len(mappings) == 0 {
		t.Error("AllErrorMappings should return non-empty slice")
	}

	// Check that mappings contain expected entries
	found := false
	for _, m := range mappings {
		if m.Code == constants.Error.DatabaseError {
			found = true
			break
		}
	}
	if !found {
		t.Error("AllErrorMappings should contain DatabaseError mapping")
	}
}
