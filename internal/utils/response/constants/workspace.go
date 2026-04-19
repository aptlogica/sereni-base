// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package constants

import "net/http"

var WorkspaceError = struct {
	ErrNotFound ResponseCode

	WorkspaceAlreadyExists    ResponseCode
	WorkspaceNotCreated       ResponseCode
	WorkspaceNotUpdated       ResponseCode
	WorkspaceNotDeleted       ResponseCode
	NameRequired              ResponseCode
	NameInvalid               ResponseCode
	DescriptionInvalid        ResponseCode
	IdRequired                ResponseCode
	IdInvalid                 ResponseCode
	ErrWorkspaceInsertion     ResponseCode
	WorkspaceMemberNotFound   ResponseCode
	ErrUserAlreadyInWorkspace ResponseCode
}{
	ErrNotFound:               "WSP_3006",
	WorkspaceAlreadyExists:    "WSP_3005",
	WorkspaceNotCreated:       "WSP_3007",
	WorkspaceNotUpdated:       "WSP_3008",
	WorkspaceNotDeleted:       "WSP_3009",
	NameRequired:              "WSP_3010",
	NameInvalid:               "WSP_3011",
	DescriptionInvalid:        "WSP_3012",
	IdRequired:                "WSP_3013",
	IdInvalid:                 "WSP_3014",
	ErrWorkspaceInsertion:     "WSP_3017",
	WorkspaceMemberNotFound:   "WSP_3015",
	ErrUserAlreadyInWorkspace: "WSP_3016",
}

var WorkspaceErrorCodes = map[ResponseCode]MetaResponse{
	WorkspaceError.ErrNotFound: {
		HTTPStatus:  http.StatusNotFound,
		Message:     "Workspace not found",
		Description: "The specified workspace could not be found",
	},
	WorkspaceError.WorkspaceAlreadyExists: {
		HTTPStatus:  http.StatusConflict,
		Message:     "Workspace already exists",
		Description: "A workspace with the given information already exists",
	},
	WorkspaceError.WorkspaceNotCreated: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Workspace not created",
		Description: "The workspace could not be created due to an internal error",
	},
	WorkspaceError.WorkspaceNotUpdated: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Workspace not updated",
		Description: "The workspace could not be updated due to an internal error",
	},
	WorkspaceError.WorkspaceNotDeleted: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Workspace not deleted",
		Description: "The workspace could not be deleted due to an internal error",
	},
	WorkspaceError.NameRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Workspace name is required",
		Description: "The workspace name field is required and was not provided",
	},
	WorkspaceError.NameInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Workspace name is invalid",
		Description: "The workspace name provided is invalid",
	},
	WorkspaceError.DescriptionInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Workspace description is invalid",
		Description: "The workspace description provided is invalid",
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
	WorkspaceError.ErrWorkspaceInsertion: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Workspace insertion failed",
		Description: "Failed to insert workspace due to an internal error",
	},
	WorkspaceError.WorkspaceMemberNotFound: {
		HTTPStatus:  http.StatusNotFound,
		Message:     "Workspace member not found",
		Description: "The specified workspace member does not exist",
	},
	WorkspaceError.ErrUserAlreadyInWorkspace: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "User already in workspace",
		Description: "The user is already a member of the specified workspace",
	},
}

var WorkspaceSuccess = struct {
	WorkspaceCreated ResponseCode
	WorkspaceUpdated ResponseCode
	WorkspaceDeleted ResponseCode
}{
	WorkspaceCreated: "WSP_SUCCESS_3001",
	WorkspaceUpdated: "WSP_SUCCESS_3002",
	WorkspaceDeleted: "WSP_SUCCESS_3003",
}

var WorkspaceSuccessCodes = map[ResponseCode]MetaResponse{
	WorkspaceSuccess.WorkspaceCreated: {
		HTTPStatus:  http.StatusCreated,
		Message:     "Workspace created successfully",
		Description: "The workspace has been created successfully",
	},
	WorkspaceSuccess.WorkspaceUpdated: {
		HTTPStatus:  http.StatusOK,
		Message:     "Workspace updated successfully",
		Description: "The workspace has been updated successfully",
	},
	WorkspaceSuccess.WorkspaceDeleted: {
		HTTPStatus:  http.StatusOK,
		Message:     "Workspace deleted successfully",
		Description: "The workspace has been deleted successfully",
	},
}
