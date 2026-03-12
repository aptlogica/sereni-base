// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package constants

// app_errors "serenibase/internal/app-errors"

type ResponseCode string

type MetaResponse struct {
	HTTPStatus  int
	Message     string
	Description string
}

// CreateMetaResponse is a helper function to create MetaResponse instances
func CreateMetaResponse(httpStatus int, message, description string) MetaResponse {
	return MetaResponse{
		HTTPStatus:  httpStatus,
		Message:     message,
		Description: description,
	}
}

// MergeMaps merges multiple maps of type map[ResponseCode]MetaResponse into one.
func MergeMaps(maps ...map[ResponseCode]MetaResponse) map[ResponseCode]MetaResponse {
	merged := make(map[ResponseCode]MetaResponse)
	for _, m := range maps {
		for k, v := range m {
			merged[k] = v
		}
	}
	return merged
}

// ErrorCodes is the merged map of all error codes.
var ErrorCodes = MergeMaps(
	AuthErrorCodes,
	UserErrorCodes,
	CoreErrorCodes,
	TenantErrorCodes,
	WorkspaceErrorCodes,
	BaseErrorCodes,
	RoleErrorCodes,
	AssetErrorCodes,
	TableErrorCodes,
	SubscriptionPlanErrorCodes,
	RBACErrorCodeDetails,
)

var SuccessCodes = MergeMaps(
	AuthSuccessCodes,
	UserSuccessCodes,
	CoreSuccessCodes,
	TenantSuccessCodes,
	WorkspaceSuccessCodes,
	BaseSuccessCodes,
	RoleSuccessCodes,
	AssetSuccessCodes,
	TableSuccessCodes,
	SubscriptionPlanSuccessCodes,
)

// 	CoreErrorCodes,
// 	UserErrorCodes,
// 	TableErrorCodes,
// 	UserValidationErrorCodes,
// 	TableValidationErrorCodes,
// 	MigrationValidationErrorCodes,
// 	BulkValidationErrorCodes,
// 	KeycloakValidationErrorCodes,
// 	fileErrorCodes)

// Generic error codes for common error scenarios
