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
	ErrNotFound:        "WSP_3006",
	BaseAlreadyExists:  "WSP_3005",
	BaseNotCreated:     "WSP_3007",
	BaseNotUpdated:     "WSP_3008",
	BaseNotDeleted:     "WSP_3009",
	NameRequired:       "WSP_3010",
	NameInvalid:        "WSP_3011",
	DescriptionInvalid: "WSP_3012",
	IdRequired:         "WSP_3013",
	IdInvalid:          "WSP_3014",
	BaseNotFound:       "WSP_3015",
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
	WorkspaceError.IdRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Workspace ID is required",
		Description: "The workspace ID field is required and was not provided",
	},
	WorkspaceError.IdInvalid: {
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
	BaseCreated: "WSP_SUCCESS_3001",
	BaseUpdated: "WSP_SUCCESS_3002",
	BaseDeleted: "WSP_SUCCESS_3003",
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
