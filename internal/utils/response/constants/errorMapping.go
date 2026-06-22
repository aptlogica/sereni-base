// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package constants

import (
	"errors"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
)

// ErrorMappingEntry defines a single mapping from a sentinel error to a response code.
type ErrorMappingEntry struct {
	Err  error
	Code ResponseCode
}

// errorMappings preserves ordering so the first match wins when using errors.Is.
var errorMappings = []ErrorMappingEntry{
	{app_errors.ErrInternal, Error.InternalError},
	{app_errors.DatabaseError, Error.DatabaseError},
	{app_errors.ErrMapToStruct, Error.MapToStructError},
	{app_errors.ErrStructToStruct, Error.StructToStructError},
	{app_errors.ErrHashed, Error.HashingError},
	{app_errors.InvalidCredentials, Error.InvalidCredentials},
	{app_errors.FileNotFound, Error.FileNotFound},
	{app_errors.FileAlreadyExists, Error.FileAlreadyExists},
	{app_errors.FileReadFailed, Error.FileReadFailed},
	{app_errors.FileWriteFailed, Error.FileWriteFailed},
	{app_errors.FileDeleteFailed, Error.FileDeleteFailed},
	{app_errors.FilePermissionDenied, Error.FilePermissionDenied},
	{app_errors.FileInvalidPath, Error.FileInvalidPath},
	{app_errors.FolderNotFound, Error.FolderNotFound},
	{app_errors.FolderAlreadyExists, Error.FolderAlreadyExists},
	{app_errors.FolderCreateFailed, Error.FolderCreateFailed},
	{app_errors.FolderDeleteFailed, Error.FolderDeleteFailed},
	{app_errors.FolderPermissionDenied, Error.FolderPermissionDenied},
	{app_errors.FolderInvalidPath, Error.FolderInvalidPath},
	{app_errors.InvalidPayload, Error.InvalidPayload},
	{app_errors.InvalidDriver, Error.InvalidDriver},
	{app_errors.ErrRecordNotFound, Error.ErrNotFound},
	{app_errors.ErrJSONMarshal, Error.JSONMarshalError},
	{app_errors.ErrHTTPRequestCreation, Error.HTTPRequestCreationError},
	{app_errors.ErrHTTPDoRequest, Error.HTTPDoRequestError},
	{app_errors.UserNotActive, Error.UserNotActive},

	// user management
	{app_errors.UserAlreadyExists, UserError.UserAlreadyExists},
	{app_errors.UserNotFound, UserError.ErrNotFound},
	{app_errors.EmailAlreadyVerified, UserError.EmailAlreadyVerified},
	{app_errors.InvalidOldPassword, UserError.InvalidOldPassword},
	{app_errors.NewPasswordInvalid, UserError.NewPasswordInvalid},
	{app_errors.EmailVerificationPending, UserError.EmailVerificationPending},

	// role management (aliases point to RBAC codes for consistency)
	{app_errors.RoleAlreadyExists, RBACError.RoleAlreadyExists},
	{app_errors.RoleNotFound, RBACError.RoleNotFound},

	// tenant management
	{app_errors.TenantAlreadyExists, TenantError.TenantAlreadyExists},
	{app_errors.TenantNotFound, TenantError.TenantNotFound},

	// auth management
	{app_errors.InvalidOTP, AuthError.InvalidOTP},
	{app_errors.AuthProviderLoginFailed, AuthError.AuthProviderLoginFailed},
	{app_errors.AuthProviderRefreshTokenFailed, AuthError.AuthProviderRefreshTokenFailed},
	{app_errors.AuthProviderTokenInvalid, AuthError.AuthProviderTokenInvalid},
	{app_errors.AuthProviderPingFailed, AuthError.AuthProviderPingFailed},
	{app_errors.AuthProviderAuthHeaderRequired, AuthError.AuthProviderAuthHeaderRequired},
	{app_errors.AuthProviderTokenDecodeFailed, AuthError.AuthProviderTokenDecodeFailed},
	{app_errors.AuthProviderClaimsNotFound, AuthError.AuthProviderClaimsNotFound},
	{app_errors.AuthProviderUserIDNotFound, AuthError.AuthProviderUserIDNotFound},
	{app_errors.TokenUserIdNotFound, AuthError.TokenUserIdNotFound},
	{app_errors.TokenAccessTokenSignFailed, AuthError.TokenAccessTokenSignFailed},
	{app_errors.TokenRefreshTokenSignFailed, AuthError.TokenRefreshTokenSignFailed},
	{app_errors.TokenRefreshTokenInvalid, AuthError.TokenRefreshTokenInvalid},
	{app_errors.TokenRefreshTokenClaimsInvalid, AuthError.TokenRefreshTokenClaimsInvalid},
	{app_errors.TokenInvalid, AuthError.TokenInvalid},
	{app_errors.TokenClaimsInvalid, AuthError.TokenClaimsInvalid},
	{app_errors.TokenAuthorizationHeaderRequired, AuthError.TokenAuthorizationHeaderRequired},
	{app_errors.TokenClaimsNotFound, AuthError.TokenClaimsNotFound},
	{app_errors.AuthProviderAdminLoginFailed, AuthError.AuthProviderAdminLoginFailed},
	{app_errors.AuthProviderUserCreateFailed, AuthError.AuthProviderUserCreateFailed},
	{app_errors.AuthProviderSetPasswordFailed, AuthError.AuthProviderSetPasswordFailed},
	{app_errors.TokenExpired, AuthError.TokenExpired},
	{app_errors.AuthProviderTokenExpired, AuthError.AuthProviderTokenExpired},
	{app_errors.TokenUnauthorized, AuthError.TokenUnauthorized},

	// workspace management
	{app_errors.ErrWorkspaceInsertion, WorkspaceError.ErrWorkspaceInsertion},
	{app_errors.WorkspaceMemberNotFound, WorkspaceError.WorkspaceMemberNotFound},
	{app_errors.ErrUserAlreadyInWorkspace, WorkspaceError.ErrUserAlreadyInWorkspace},

	// base management
	{app_errors.BaseNotFound, BaseError.BaseNotFound},

	// asset management
	{app_errors.VirusDetected, AssetError.VirusDetected},
	{app_errors.StorageFileOpenFailed, AssetError.StorageFileOpenFailed},
	{app_errors.StorageUploadFailed, AssetError.StorageUploadFailed},
	{app_errors.AssetNotFound, AssetError.AssetNotFound},
	{app_errors.MultipartFormNotFound, AssetError.MultipartFormNotFound},
	{app_errors.FileTooLargeError, AssetError.FileTooLargeError},
	{app_errors.MultipleFilesTooLargeError, AssetError.MultipleFilesTooLargeError},
	{app_errors.TooManyFilesError, AssetError.TooManyFilesError},

	// table management
	{app_errors.ViewNotFound, TableError.ViewNotFound},
	{app_errors.ViewUploadFailed, TableError.ViewUploadFailed},
	{app_errors.UpdateNotAllowed, TableError.UpdateNotAllowed},
	{app_errors.DeleteNotAllowed, TableError.DeleteNotAllowed},
	{app_errors.ColumnNotFound, TableError.ColumnNotFound},
	{app_errors.ColumnUpdateFailed, TableError.ColumnUpdateFailed},
	{app_errors.TableNotFound, TableError.TableNotFound},
	{app_errors.InvalidUIDT, TableError.UIDTInvalid},
	{app_errors.InvalidColumnMetaForLinkType, TableError.InvalidColumnMetaForLinkType},
	{app_errors.RowNotFound, TableError.RowNotFound},
	{app_errors.InvalidColumnMetaForLookupType, TableError.InvalidColumnMetaForLookupType},
	{app_errors.SplitNotPossible, TableError.SplitNotPossible},

	// New mappings
	{app_errors.ErrInvalidDateOfBirth, Error.InvalidDateOfBirth},
	{app_errors.ErrRoleCreation, Error.RoleCreationError},
	{app_errors.ErrRoleNotFound, Error.RoleNotFound},
	{app_errors.ErrUserDisableFailed, Error.UserDisableFailed},
	{app_errors.ErrInvalidWorkspaceMemberData, Error.InvalidWorkspaceMemberData},
	{app_errors.ErrUserContextNotFound, Error.UserContextNotFound},

	// RBAC (Role-Based Access Control) errors
	{app_errors.RoleDeleteFailed, RBACError.RoleDeleteFailed},
	{app_errors.RoleUpdateFailed, RBACError.RoleUpdateFailed},
	{app_errors.InvalidRolePriority, RBACError.InvalidRolePriority},
	{app_errors.RoleAssignmentFailed, RBACError.RoleAssignmentFailed},
	{app_errors.RoleRemovalFailed, RBACError.RoleRemovalFailed},

	// Resource errors
	{app_errors.ResourceNotFound, RBACError.ResourceNotFound},
	{app_errors.ResourceAlreadyExists, RBACError.ResourceAlreadyExists},
	{app_errors.ResourceCreateFailed, RBACError.ResourceCreateFailed},
	{app_errors.ResourceDeleteFailed, RBACError.ResourceDeleteFailed},
	{app_errors.InvalidResourceCode, RBACError.InvalidResourceCode},

	// Action errors
	{app_errors.ActionNotFound, RBACError.ActionNotFound},
	{app_errors.ActionAlreadyExists, RBACError.ActionAlreadyExists},
	{app_errors.ActionCreateFailed, RBACError.ActionCreateFailed},
	{app_errors.ActionDeleteFailed, RBACError.ActionDeleteFailed},
	{app_errors.InvalidActionCode, RBACError.InvalidActionCode},

	// Permission errors
	{app_errors.PermissionNotFound, RBACError.PermissionNotFound},
	{app_errors.PermissionAlreadyExists, RBACError.PermissionAlreadyExists},
	{app_errors.PermissionCreateFailed, RBACError.PermissionCreateFailed},
	{app_errors.PermissionDeleteFailed, RBACError.PermissionDeleteFailed},
	{app_errors.InvalidPermissionCombo, RBACError.InvalidPermissionCombo},

	// Role-Permission errors
	{app_errors.RolePermissionNotFound, RBACError.RolePermissionNotFound},
	{app_errors.RolePermissionExists, RBACError.RolePermissionExists},
	{app_errors.RolePermissionCreateFailed, RBACError.RolePermissionCreateFailed},
	{app_errors.RolePermissionDeleteFailed, RBACError.RolePermissionDeleteFailed},

	// Access Member errors
	{app_errors.AccessMemberNotFound, RBACError.AccessMemberNotFound},
	{app_errors.AccessMemberAlreadyExists, RBACError.AccessMemberAlreadyExists},
	{app_errors.AccessMemberCreateFailed, RBACError.AccessMemberCreateFailed},
	{app_errors.AccessMemberDeleteFailed, RBACError.AccessMemberDeleteFailed},
	{app_errors.InvalidAccessScope, RBACError.InvalidAccessScope},
	{app_errors.MissingScopeID, RBACError.MissingScopeID},
	{app_errors.UserNotInScope, RBACError.UserNotInScope},

	// Permission check errors
	{app_errors.PermissionDenied, RBACError.PermissionDenied},
	{app_errors.AccessDenied, RBACError.AccessDenied},
	{app_errors.InsufficientPrivileges, RBACError.InsufficientPrivileges},

	// Bulk operation errors
	{app_errors.BulkAssignmentFailed, RBACError.BulkAssignmentFailed},
	{app_errors.BulkRemovalFailed, RBACError.BulkRemovalFailed},
	{app_errors.EmptyUserList, RBACError.EmptyUserList},

	// Scope errors
	{app_errors.InvalidScopeType, RBACError.InvalidScopeType},
	{app_errors.ScopeNotFound, RBACError.ScopeNotFound},
}

// MapError returns the response code for a given error using errors.Is to support wrapped errors.
func MapError(err error) ResponseCode {
	for _, m := range errorMappings {
		if errors.Is(err, m.Err) {
			return m.Code
		}
	}
	return ""
}

// AllErrorMappings returns the mapping table for validation/testing.
func AllErrorMappings() []ErrorMappingEntry {
	return errorMappings
}
