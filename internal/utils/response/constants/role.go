// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package constants

import "net/http"

var RoleError = struct {
	RoleNotFound      ResponseCode
	RoleAlreadyExists ResponseCode
	RoleNotCreated    ResponseCode
	RoleNotUpdated    ResponseCode
	RoleNotDeleted    ResponseCode
	RoleRequired      ResponseCode
	RoleInvalid       ResponseCode
}{
	RoleNotFound:      "ROL_4001",
	RoleAlreadyExists: "ROL_4002",
	RoleNotCreated:    "ROL_4003",
	RoleNotUpdated:    "ROL_4004",
	RoleNotDeleted:    "ROL_4005",
	RoleRequired:      "ROL_4006",
	RoleInvalid:       "ROL_4007",
}

var RoleErrorCodes = map[ResponseCode]MetaResponse{
	RoleError.RoleNotFound:      CreateMetaResponse(http.StatusNotFound, "Role not found", "The specified role could not be found"),
	RoleError.RoleAlreadyExists: CreateMetaResponse(http.StatusConflict, "Role already exists", "A role with the given information already exists"),
	RoleError.RoleNotCreated:    CreateMetaResponse(http.StatusInternalServerError, "Role not created", "The role could not be created due to an internal error"),
	RoleError.RoleNotUpdated:    CreateMetaResponse(http.StatusInternalServerError, "Role not updated", "The role could not be updated due to an internal error"),
	RoleError.RoleNotDeleted:    CreateMetaResponse(http.StatusInternalServerError, "Role not deleted", "The role could not be deleted due to an internal error"),
	RoleError.RoleRequired:      CreateMetaResponse(http.StatusBadRequest, "Role is required", "A role must be provided in the request"),
	RoleError.RoleInvalid:       CreateMetaResponse(http.StatusBadRequest, "Invalid role", "The specified role is not valid"),
}

var RoleSuccess = struct {
	RoleCreated ResponseCode
	RoleUpdated ResponseCode
	RoleDeleted ResponseCode
}{
	RoleCreated: "ROL_SUCCESS_4001",
	RoleUpdated: "ROL_SUCCESS_4002",
	RoleDeleted: "ROL_SUCCESS_4003",
}

var RoleSuccessCodes = map[ResponseCode]MetaResponse{
	RoleSuccess.RoleCreated: CreateMetaResponse(http.StatusCreated, "Role created successfully", "The role has been created successfully"),
	RoleSuccess.RoleUpdated: CreateMetaResponse(http.StatusOK, "Role updated successfully", "The role has been updated successfully"),
	RoleSuccess.RoleDeleted: CreateMetaResponse(http.StatusOK, "Role deleted successfully", "The role has been deleted successfully"),
}
