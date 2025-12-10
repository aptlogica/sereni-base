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
	RoleError.RoleNotFound: {
		HTTPStatus:  http.StatusNotFound,
		Message:     "Role not found",
		Description: "The specified role could not be found",
	},
	RoleError.RoleAlreadyExists: {
		HTTPStatus:  http.StatusConflict,
		Message:     "Role already exists",
		Description: "A role with the given information already exists",
	},
	RoleError.RoleNotCreated: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Role not created",
		Description: "The role could not be created due to an internal error",
	},
	RoleError.RoleNotUpdated: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Role not updated",
		Description: "The role could not be updated due to an internal error",
	},
	RoleError.RoleNotDeleted: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Role not deleted",
		Description: "The role could not be deleted due to an internal error",
	},
	RoleError.RoleRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Role is required",
		Description: "A role must be provided in the request",
	},
	RoleError.RoleInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid role",
		Description: "The specified role is not valid",
	},
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
	RoleSuccess.RoleCreated: {
		HTTPStatus:  http.StatusCreated,
		Message:     "Role created successfully",
		Description: "The role has been created successfully",
	},
	RoleSuccess.RoleUpdated: {
		HTTPStatus:  http.StatusOK,
		Message:     "Role updated successfully",
		Description: "The role has been updated successfully",
	},
	RoleSuccess.RoleDeleted: {
		HTTPStatus:  http.StatusOK,
		Message:     "Role deleted successfully",
		Description: "The role has been deleted successfully",
	},
}
