// Copyright (c) 2026 Aptlogica Technologies Private Limited
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package constants

import "net/http"

var BaseError = struct {
	ErrNotFound        ResponseCode
	BaseAlreadyExists  ResponseCode
	BaseNotCreated     ResponseCode
	BaseNotUpdated     ResponseCode
	BaseNotDeleted     ResponseCode
	NameRequired       ResponseCode
	NameInvalid        ResponseCode
	DescriptionInvalid ResponseCode
	IdRequired         ResponseCode
	IdInvalid          ResponseCode
	BaseNotFound       ResponseCode
}{
	ErrNotFound:        "BAS_6001",
	BaseAlreadyExists:  "BAS_6002",
	BaseNotCreated:     "BAS_6003",
	BaseNotUpdated:     "BAS_6004",
	BaseNotDeleted:     "BAS_6005",
	NameRequired:       "BAS_6006",
	NameInvalid:        "BAS_6007",
	DescriptionInvalid: "BAS_6008",
	IdRequired:         "BAS_6009",
	IdInvalid:          "BAS_6010",
	BaseNotFound:       "BAS_6011",
}

var BaseErrorCodes = map[ResponseCode]MetaResponse{
	BaseError.ErrNotFound: {
		HTTPStatus:  http.StatusNotFound,
		Message:     "Base not found",
		Description: "The specified Base could not be found",
	},
	BaseError.BaseAlreadyExists: {
		HTTPStatus:  http.StatusConflict,
		Message:     "Base already exists",
		Description: "A Base with the given information already exists",
	},
	BaseError.BaseNotCreated: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Base not created",
		Description: "The Base could not be created due to an internal error",
	},
	BaseError.BaseNotUpdated: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Base not updated",
		Description: "The Base could not be updated due to an internal error",
	},
	BaseError.BaseNotDeleted: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Base not deleted",
		Description: "The Base could not be deleted due to an internal error",
	},
	BaseError.NameRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Base name is required",
		Description: "The Base name field is required and was not provided",
	},
	BaseError.NameInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Base name is invalid",
		Description: "The Base name provided is invalid",
	},
	BaseError.DescriptionInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Base description is invalid",
		Description: "The Base description provided is invalid",
	},
	BaseError.IdRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Workspace ID is required",
		Description: "The workspace ID field is required and was not provided",
	},
	BaseError.IdInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Workspace ID is invalid",
		Description: "The workspace ID provided is invalid",
	},
	BaseError.BaseNotFound: {
		HTTPStatus:  http.StatusNotFound,
		Message:     "Base not found",
		Description: "The specified Base could not be found",
	},
}

var BaseSuccess = struct {
	BaseCreated ResponseCode
	BaseUpdated ResponseCode
	BaseDeleted ResponseCode
}{
	BaseCreated: "BAS_SUCCESS_6001",
	BaseUpdated: "BAS_SUCCESS_6002",
	BaseDeleted: "BAS_SUCCESS_6003",
}

var BaseSuccessCodes = map[ResponseCode]MetaResponse{
	BaseSuccess.BaseCreated: {
		HTTPStatus:  http.StatusCreated,
		Message:     "Base created successfully",
		Description: "The Base has been created successfully",
	},
	BaseSuccess.BaseUpdated: {
		HTTPStatus:  http.StatusOK,
		Message:     "Base updated successfully",
		Description: "The Base has been updated successfully",
	},
	BaseSuccess.BaseDeleted: {
		HTTPStatus:  http.StatusOK,
		Message:     "Base deleted successfully",
		Description: "The Base has been deleted successfully",
	},
}
