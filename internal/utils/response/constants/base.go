// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
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
	NameTooLong        ResponseCode
	NameInvalid        ResponseCode
	DescriptionInvalid ResponseCode
	IdRequired         ResponseCode
	IdInvalid          ResponseCode
	BaseNotFound       ResponseCode
	ImageTooLarge      ResponseCode
	NameTooShort       ResponseCode
	TitleAlreadyExists ResponseCode
}{
	ErrNotFound:        "BAS_6001",
	BaseAlreadyExists:  "BAS_6002",
	BaseNotCreated:     "BAS_6003",
	BaseNotUpdated:     "BAS_6004",
	BaseNotDeleted:     "BAS_6005",
	NameRequired:       "BAS_6006",
	NameTooLong:        "BAS_6012",
	NameInvalid:        "BAS_6007",
	DescriptionInvalid: "BAS_6008",
	IdRequired:         "BAS_6009",
	IdInvalid:          "BAS_6010",
	BaseNotFound:       "BAS_6011",
	ImageTooLarge:      "BAS_6013",
	NameTooShort:       "BAS_6014",
	TitleAlreadyExists: "BAS_6015",
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
	BaseError.NameTooShort: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Base name must be at least 3 characters. Please lengthen the name and try again",
		Description: "The base name is less than the required 3 characters limit. Use a longer base name",
	},
	BaseError.NameTooLong: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Base name must be 50 characters or fewer. Please shorten the name and try again",
		Description: "The base name exceeds the allowed limit of 50 characters. Use a shorter base name with 50 characters or fewer",
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
		Message:     "Base ID is required",
		Description: "The Base ID field is required and was not provided",
	},
	BaseError.IdInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Base ID is invalid",
		Description: "The workspace ID provided is invalid",
	},
	BaseError.BaseNotFound: {
		HTTPStatus:  http.StatusNotFound,
		Message:     "Base not found",
		Description: "The specified Base could not be found",
	},
	BaseError.ImageTooLarge: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Image size exceeds maximum allowed limit",
		Description: "The uploaded image exceeds the maximum allowed size of 5MB",
	},
	BaseError.TitleAlreadyExists: {
		HTTPStatus:  http.StatusConflict,
		Message:     "Title already exists",
		Description: "A Base with the given title already exists in this workspace",
	},
}

var BaseSuccess = struct {
	BaseCreated  ResponseCode
	BaseUpdated  ResponseCode
	BaseDeleted  ResponseCode
	BasesFetched ResponseCode
}{
	BaseCreated:  "BAS_SUCCESS_6001",
	BaseUpdated:  "BAS_SUCCESS_6002",
	BaseDeleted:  "BAS_SUCCESS_6003",
	BasesFetched: "BAS_SUCCESS_6004",
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
	BaseSuccess.BasesFetched: {
		HTTPStatus:  http.StatusOK,
		Message:     "Bases retrieved successfully",
		Description: "The bases have been retrieved successfully",
	},
}
