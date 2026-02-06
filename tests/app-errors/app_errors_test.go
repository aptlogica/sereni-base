package tests

import (
	"errors"
	"testing"

	appErrors "serenibase/internal/app-errors"
)

// TestAllErrorConstants verifies all error constants are defined and not nil
func TestAllErrorConstants(t *testing.T) {
	tests := []struct {
		name  string
		error error
	}{
		// File errors
		{"FileNotFound", appErrors.FileNotFound},
		{"FileAlreadyExists", appErrors.FileAlreadyExists},
		{"FileReadFailed", appErrors.FileReadFailed},
		{"FileWriteFailed", appErrors.FileWriteFailed},
		{"FileDeleteFailed", appErrors.FileDeleteFailed},
		{"FilePermissionDenied", appErrors.FilePermissionDenied},
		{"FileInvalidPath", appErrors.FileInvalidPath},
		{"FolderNotFound", appErrors.FolderNotFound},
		{"FolderAlreadyExists", appErrors.FolderAlreadyExists},
		{"FolderCreateFailed", appErrors.FolderCreateFailed},
		{"FolderDeleteFailed", appErrors.FolderDeleteFailed},
		{"FolderPermissionDenied", appErrors.FolderPermissionDenied},
		{"FolderInvalidPath", appErrors.FolderInvalidPath},

		// New refactoring errors
		{"ErrInvalidDateOfBirth", appErrors.ErrInvalidDateOfBirth},
		{"ErrRoleCreation", appErrors.ErrRoleCreation},
		{"ErrSubscriptionPlanNotFound", appErrors.ErrSubscriptionPlanNotFound},
		{"ErrRoleNotFound", appErrors.ErrRoleNotFound},
		{"ErrUserDisableFailed", appErrors.ErrUserDisableFailed},
		{"ErrInvalidWorkspaceMemberData", appErrors.ErrInvalidWorkspaceMemberData},
		{"ErrUserContextNotFound", appErrors.ErrUserContextNotFound},

		// Database and general errors
		{"DatabaseError", appErrors.DatabaseError},
		{"ErrInternal", appErrors.ErrInternal},
		{"ErrMapToStruct", appErrors.ErrMapToStruct},
		{"ErrStructToStruct", appErrors.ErrStructToStruct},
		{"ErrHashed", appErrors.ErrHashed},
		{"InvalidCredentials", appErrors.InvalidCredentials},
		{"InvalidPayload", appErrors.InvalidPayload},
		{"InvalidDriver", appErrors.InvalidDriver},
		{"InvalidOldPassword", appErrors.InvalidOldPassword},
		{"ErrRecordNotFound", appErrors.ErrRecordNotFound},
		{"ErrJSONMarshal", appErrors.ErrJSONMarshal},
		{"ErrHTTPRequestCreation", appErrors.ErrHTTPRequestCreation},
		{"ErrHTTPDoRequest", appErrors.ErrHTTPDoRequest},
		{"ErrServiceNotInitialized", appErrors.ErrServiceNotInitialized},
		{"UserNotActive", appErrors.UserNotActive},

		// User management errors
		{"UserAlreadyExists", appErrors.UserAlreadyExists},
		{"UserNotFound", appErrors.UserNotFound},
		{"EmailAlreadyVerified", appErrors.EmailAlreadyVerified},
		{"NewPasswordInvalid", appErrors.NewPasswordInvalid},
		{"OwnerCannotBeDeactivated", appErrors.OwnerCannotBeDeactivated},
		{"OnlyPendingUsersCanBeDeleted", appErrors.OnlyPendingUsersCanBeDeleted},

		// Role management errors
		{"RoleAlreadyExists", appErrors.RoleAlreadyExists},
		{"RoleNotFound", appErrors.RoleNotFound},

		// Subscription plan errors
		{"SubscriptionPlanAlreadyExists", appErrors.SubscriptionPlanAlreadyExists},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.error == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
			if tt.error.Error() == "" {
				t.Errorf("%s should have a non-empty error message", tt.name)
			}
		})
	}
}

// TestRBACErrorConstants verifies all RBAC error constants
func TestRBACErrorConstants(t *testing.T) {
	tests := []struct {
		name  string
		error error
	}{
		// Role errors
		{"RoleDeleteFailed", appErrors.RoleDeleteFailed},
		{"RoleUpdateFailed", appErrors.RoleUpdateFailed},
		{"InvalidRolePriority", appErrors.InvalidRolePriority},
		{"RoleAssignmentFailed", appErrors.RoleAssignmentFailed},
		{"RoleRemovalFailed", appErrors.RoleRemovalFailed},

		// Resource errors
		{"ResourceNotFound", appErrors.ResourceNotFound},
		{"ResourceAlreadyExists", appErrors.ResourceAlreadyExists},
		{"ResourceCreateFailed", appErrors.ResourceCreateFailed},
		{"ResourceDeleteFailed", appErrors.ResourceDeleteFailed},
		{"InvalidResourceCode", appErrors.InvalidResourceCode},

		// Action errors
		{"ActionNotFound", appErrors.ActionNotFound},
		{"ActionAlreadyExists", appErrors.ActionAlreadyExists},
		{"ActionCreateFailed", appErrors.ActionCreateFailed},
		{"ActionDeleteFailed", appErrors.ActionDeleteFailed},
		{"InvalidActionCode", appErrors.InvalidActionCode},

		// Permission errors
		{"PermissionNotFound", appErrors.PermissionNotFound},
		{"PermissionAlreadyExists", appErrors.PermissionAlreadyExists},
		{"PermissionCreateFailed", appErrors.PermissionCreateFailed},
		{"PermissionDeleteFailed", appErrors.PermissionDeleteFailed},
		{"InvalidPermissionCombo", appErrors.InvalidPermissionCombo},

		// Role-Permission errors
		{"RolePermissionNotFound", appErrors.RolePermissionNotFound},
		{"RolePermissionExists", appErrors.RolePermissionExists},
		{"RolePermissionCreateFailed", appErrors.RolePermissionCreateFailed},
		{"RolePermissionDeleteFailed", appErrors.RolePermissionDeleteFailed},

		// Access Member errors
		{"AccessMemberNotFound", appErrors.AccessMemberNotFound},
		{"AccessMemberAlreadyExists", appErrors.AccessMemberAlreadyExists},
		{"AccessMemberCreateFailed", appErrors.AccessMemberCreateFailed},
		{"AccessMemberDeleteFailed", appErrors.AccessMemberDeleteFailed},
		{"InvalidAccessScope", appErrors.InvalidAccessScope},
		{"MissingScopeID", appErrors.MissingScopeID},
		{"UserNotInScope", appErrors.UserNotInScope},

		// Permission check errors
		{"PermissionDenied", appErrors.PermissionDenied},
		{"AccessDenied", appErrors.AccessDenied},
		{"InsufficientPrivileges", appErrors.InsufficientPrivileges},

		// Bulk operation errors
		{"BulkAssignmentFailed", appErrors.BulkAssignmentFailed},
		{"BulkRemovalFailed", appErrors.BulkRemovalFailed},
		{"EmptyUserList", appErrors.EmptyUserList},

		// Scope errors
		{"InvalidScopeType", appErrors.InvalidScopeType},
		{"ScopeNotFound", appErrors.ScopeNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.error == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
			if tt.error.Error() == "" {
				t.Errorf("%s should have a non-empty error message", tt.name)
			}
		})
	}
}

// TestLogDatabaseError tests the LogDatabaseError function
func TestLogDatabaseError(t *testing.T) {
	tests := []struct {
		name        string
		inputErr    error
		message     string
		expectedErr error
	}{
		{
			name:        "nil error returns DatabaseError",
			inputErr:    nil,
			message:     "test message",
			expectedErr: appErrors.DatabaseError,
		},
		{
			name:        "non-nil error returns DatabaseError",
			inputErr:    errors.New("original error"),
			message:     "database operation failed",
			expectedErr: appErrors.DatabaseError,
		},
		{
			name:        "empty message with error",
			inputErr:    errors.New("test error"),
			message:     "",
			expectedErr: appErrors.DatabaseError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := appErrors.LogDatabaseError(tt.inputErr, tt.message)
			if result != tt.expectedErr {
				t.Errorf("LogDatabaseError() = %v, want %v", result, tt.expectedErr)
			}
		})
	}
}

// TestErrorEquality verifies errors can be compared using errors.Is
func TestErrorEquality(t *testing.T) {
	tests := []struct {
		name   string
		err1   error
		err2   error
		expect bool
	}{
		{
			name:   "same error should be equal",
			err1:   appErrors.UserNotFound,
			err2:   appErrors.UserNotFound,
			expect: true,
		},
		{
			name:   "different errors should not be equal",
			err1:   appErrors.UserNotFound,
			err2:   appErrors.UserAlreadyExists,
			expect: false,
		},
		{
			name:   "DatabaseError should be equal to itself",
			err1:   appErrors.DatabaseError,
			err2:   appErrors.DatabaseError,
			expect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errors.Is(tt.err1, tt.err2)
			if result != tt.expect {
				t.Errorf("errors.Is(%v, %v) = %v, want %v", tt.err1, tt.err2, result, tt.expect)
			}
		})
	}
}

// TestErrorMessages verifies specific error messages
func TestErrorMessages(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectedMsg string
	}{
		{"UserNotFound message", appErrors.UserNotFound, "user not found"},
		{"RoleAlreadyExists message", appErrors.RoleAlreadyExists, "role already exists"},
		{"PermissionDenied message", appErrors.PermissionDenied, "user does not have permission to perform this action"},
		{"InvalidCredentials message", appErrors.InvalidCredentials, "invalid credentials"},
		{"DatabaseError message", appErrors.DatabaseError, "database error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expectedMsg {
				t.Errorf("%s error message = %q, want %q", tt.name, tt.err.Error(), tt.expectedMsg)
			}
		})
	}
}

// TestAPIError_Error tests the Error method of APIError
func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		apiError appErrors.APIError
		expected string
	}{
		{
			name: "with message",
			apiError: appErrors.APIError{
				Code:    "TEST_ERROR",
				Message: "Test error message",
			},
			expected: "Test error message",
		},
		{
			name: "without message",
			apiError: appErrors.APIError{
				Code: "TEST_ERROR",
			},
			expected: "TEST_ERROR",
		},
		{
			name: "empty message",
			apiError: appErrors.APIError{
				Code:    "TEST_ERROR",
				Message: "",
			},
			expected: "TEST_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.apiError.Error()
			if result != tt.expected {
				t.Errorf("APIError.Error() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestAPIError_WithDetails tests APIError with details
func TestAPIError_WithDetails(t *testing.T) {
	details := map[string]string{"field": "username"}
	apiError := appErrors.APIError{
		Code:       "VALIDATION_ERROR",
		Message:    "Validation failed",
		Details:    details,
		StatusCode: 400,
	}

	if apiError.Code != "VALIDATION_ERROR" {
		t.Errorf("Code = %q, want %q", apiError.Code, "VALIDATION_ERROR")
	}
	if apiError.StatusCode != 400 {
		t.Errorf("StatusCode = %d, want %d", apiError.StatusCode, 400)
	}
	if apiError.Details == nil {
		t.Error("Details should not be nil")
	}
}

// TestAllAuthErrors tests all auth-related errors
func TestAllAuthErrors(t *testing.T) {
	authErrors := []struct {
		name  string
		error error
	}{
		{"InvalidOTP", appErrors.InvalidOTP},
		{"AuthProviderLoginFailed", appErrors.AuthProviderLoginFailed},
		{"AuthProviderRefreshTokenFailed", appErrors.AuthProviderRefreshTokenFailed},
		{"AuthProviderTokenInvalid", appErrors.AuthProviderTokenInvalid},
		{"AuthProviderPingFailed", appErrors.AuthProviderPingFailed},
		{"AuthProviderAuthHeaderRequired", appErrors.AuthProviderAuthHeaderRequired},
		{"AuthProviderTokenDecodeFailed", appErrors.AuthProviderTokenDecodeFailed},
		{"AuthProviderClaimsNotFound", appErrors.AuthProviderClaimsNotFound},
		{"AuthProviderUserIDNotFound", appErrors.AuthProviderUserIDNotFound},
		{"TokenUserIdNotFound", appErrors.TokenUserIdNotFound},
		{"TokenAccessTokenSignFailed", appErrors.TokenAccessTokenSignFailed},
		{"TokenRefreshTokenSignFailed", appErrors.TokenRefreshTokenSignFailed},
		{"TokenRefreshTokenInvalid", appErrors.TokenRefreshTokenInvalid},
		{"TokenRefreshTokenClaimsInvalid", appErrors.TokenRefreshTokenClaimsInvalid},
		{"TokenInvalid", appErrors.TokenInvalid},
		{"TokenClaimsInvalid", appErrors.TokenClaimsInvalid},
		{"TokenAuthorizationHeaderRequired", appErrors.TokenAuthorizationHeaderRequired},
		{"TokenClaimsNotFound", appErrors.TokenClaimsNotFound},
		{"AuthProviderAdminLoginFailed", appErrors.AuthProviderAdminLoginFailed},
		{"AuthProviderUserCreateFailed", appErrors.AuthProviderUserCreateFailed},
		{"AuthProviderSetPasswordFailed", appErrors.AuthProviderSetPasswordFailed},
		{"TokenExpired", appErrors.TokenExpired},
		{"AuthProviderTokenExpired", appErrors.AuthProviderTokenExpired},
		{"TokenUnauthorized", appErrors.TokenUnauthorized},
	}

	for _, tt := range authErrors {
		t.Run(tt.name, func(t *testing.T) {
			if tt.error == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
			if tt.error.Error() == "" {
				t.Errorf("%s should have a non-empty error message", tt.name)
			}
		})
	}
}

// TestAllWorkspaceErrors tests workspace-related errors
func TestAllWorkspaceErrors(t *testing.T) {
	workspaceErrors := []struct {
		name  string
		error error
	}{
		{"ErrWorkspaceInsertion", appErrors.ErrWorkspaceInsertion},
		{"WorkspaceMemberNotFound", appErrors.WorkspaceMemberNotFound},
		{"ErrUserAlreadyInWorkspace", appErrors.ErrUserAlreadyInWorkspace},
	}

	for _, tt := range workspaceErrors {
		t.Run(tt.name, func(t *testing.T) {
			if tt.error == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
		})
	}
}

// TestAllAssetErrors tests asset-related errors
func TestAllAssetErrors(t *testing.T) {
	assetErrors := []struct {
		name  string
		error error
	}{
		{"VirusDetected", appErrors.VirusDetected},
		{"StorageFileOpenFailed", appErrors.StorageFileOpenFailed},
		{"StorageUploadFailed", appErrors.StorageUploadFailed},
		{"AssetNotFound", appErrors.AssetNotFound},
		{"FileTooLargeError", appErrors.FileTooLargeError},
		{"MultipleFilesTooLargeError", appErrors.MultipleFilesTooLargeError},
		{"TooManyFilesError", appErrors.TooManyFilesError},
		{"MultipartFormNotFound", appErrors.MultipartFormNotFound},
	}

	for _, tt := range assetErrors {
		t.Run(tt.name, func(t *testing.T) {
			if tt.error == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
		})
	}
}

// TestAllTableErrors tests table-related errors
func TestAllTableErrors(t *testing.T) {
	tableErrors := []struct {
		name  string
		error error
	}{
		{"UpdateNotAllowed", appErrors.UpdateNotAllowed},
		{"DeleteNotAllowed", appErrors.DeleteNotAllowed},
		{"ViewNotFound", appErrors.ViewNotFound},
		{"ViewUploadFailed", appErrors.ViewUploadFailed},
		{"ColumnNotFound", appErrors.ColumnNotFound},
		{"ColumnUpdateFailed", appErrors.ColumnUpdateFailed},
		{"TableNotFound", appErrors.TableNotFound},
		{"InvalidUIDT", appErrors.InvalidUIDT},
		{"InvalidColumnMetaForLinkType", appErrors.InvalidColumnMetaForLinkType},
		{"RowNotFound", appErrors.RowNotFound},
		{"InvalidColumnMetaForLookupType", appErrors.InvalidColumnMetaForLookupType},
	}

	for _, tt := range tableErrors {
		t.Run(tt.name, func(t *testing.T) {
			if tt.error == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
		})
	}
}

// TestAllTenantErrors tests tenant-related errors
func TestAllTenantErrors(t *testing.T) {
	tenantErrors := []struct {
		name  string
		error error
	}{
		{"TenantAlreadyExists", appErrors.TenantAlreadyExists},
		{"TenantNotFound", appErrors.TenantNotFound},
		{"TenantSubscriptionAlreadyExists", appErrors.TenantSubscriptionAlreadyExists},
		{"TenantSubscriptionNotFound", appErrors.TenantSubscriptionNotFound},
	}

	for _, tt := range tenantErrors {
		t.Run(tt.name, func(t *testing.T) {
			if tt.error == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
		})
	}
}

// TestBaseErrors tests base-related errors
func TestBaseErrors(t *testing.T) {
	if appErrors.BaseNotFound == nil {
		t.Error("BaseNotFound should not be nil")
	}
	if appErrors.BaseNotFound.Error() != "base not found" {
		t.Errorf("BaseNotFound message = %q, want %q", appErrors.BaseNotFound.Error(), "base not found")
	}
}

// TestAllScopeErrors tests scope-related errors
func TestAllScopeErrors(t *testing.T) {
	scopeErrors := []struct {
		name  string
		error error
	}{
		{"InvalidScopeType", appErrors.InvalidScopeType},
		{"ScopeNotFound", appErrors.ScopeNotFound},
		{"InvalidAccessScope", appErrors.InvalidAccessScope},
		{"MissingScopeID", appErrors.MissingScopeID},
		{"UserNotInScope", appErrors.UserNotInScope},
	}

	for _, tt := range scopeErrors {
		t.Run(tt.name, func(t *testing.T) {
			if tt.error == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
		})
	}
}

// TestBulkOperationErrors tests bulk operation errors
func TestBulkOperationErrors(t *testing.T) {
	bulkErrors := []struct {
		name  string
		error error
	}{
		{"BulkAssignmentFailed", appErrors.BulkAssignmentFailed},
		{"BulkRemovalFailed", appErrors.BulkRemovalFailed},
		{"EmptyUserList", appErrors.EmptyUserList},
	}

	for _, tt := range bulkErrors {
		t.Run(tt.name, func(t *testing.T) {
			if tt.error == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
		})
	}
}
