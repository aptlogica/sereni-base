// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package constants

import "net/http"

var UserError = struct {
	ErrNotFound          ResponseCode
	UserAlreadyExists    ResponseCode
	UserNotCreated       ResponseCode
	UserNotUpdated       ResponseCode
	UserNotDeleted       ResponseCode
	EmailAlreadyVerified ResponseCode
	InvalidOldPassword   ResponseCode
	OldPasswordRequired  ResponseCode
	OldPasswordInvalid   ResponseCode
	NewPasswordRequired  ResponseCode
	NewPasswordInvalid   ResponseCode
	EmailRequired        ResponseCode
	EmailInvalid         ResponseCode
	FirstNameRequired    ResponseCode
	FirstNameInvalid     ResponseCode
	LastNameRequired     ResponseCode
	LastNameInvalid      ResponseCode
	RoleIDRequired       ResponseCode
	RoleIDInvalid        ResponseCode
	UserIDRequired       ResponseCode
	UserIDInvalid        ResponseCode
}{
	ErrNotFound:          "USR_2006",
	UserAlreadyExists:    "USR_2005",
	UserNotCreated:       "USR_2007",
	UserNotUpdated:       "USR_2008",
	UserNotDeleted:       "USR_2009",
	EmailAlreadyVerified: "USR_2010",
	InvalidOldPassword:   "USR_2011",
	OldPasswordRequired:  "USR_2012",
	OldPasswordInvalid:   "USR_2013",
	NewPasswordRequired:  "USR_2014",
	NewPasswordInvalid:   "USR_2015",
	// AddUserRequest validation error codes
	EmailRequired:     "USR_2016",
	EmailInvalid:      "USR_2017",
	FirstNameRequired: "USR_2018",
	FirstNameInvalid:  "USR_2019",
	LastNameRequired:  "USR_2020",
	LastNameInvalid:   "USR_2021",
	RoleIDRequired:    "USR_2022",
	RoleIDInvalid:     "USR_2023",
	UserIDRequired:    "USR_2024",
	UserIDInvalid:     "USR_2025",
}

var UserErrorCodes = map[ResponseCode]MetaResponse{
	UserError.ErrNotFound: {
		HTTPStatus:  http.StatusNotFound,
		Message:     "User not found",
		Description: "The specified user could not be found",
	},
	UserError.UserAlreadyExists: {
		HTTPStatus:  http.StatusConflict,
		Message:     "User already exists",
		Description: "A user with the given information already exists",
	},
	UserError.UserNotCreated: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "User not created",
		Description: "The user could not be created due to an internal error",
	},
	UserError.UserNotUpdated: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "User not updated",
		Description: "The user could not be updated due to an internal error",
	},
	UserError.UserNotDeleted: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "User not deleted",
		Description: "The user could not be deleted due to an internal error",
	},
	UserError.EmailAlreadyVerified: {
		HTTPStatus:  http.StatusConflict,
		Message:     "Email already verified",
		Description: "The user's email address has already been verified",
	},
	UserError.InvalidOldPassword: {
		HTTPStatus:  http.StatusUnauthorized,
		Message:     "Invalid old password",
		Description: "The provided old password is incorrect",
	},
	UserError.OldPasswordRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Old password is required",
		Description: "You must provide your old password to proceed",
	},
	UserError.OldPasswordInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Old password is invalid",
		Description: "The old password provided does not meet requirements or is invalid",
	},
	UserError.NewPasswordRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "New password is required",
		Description: "You must provide a new password to update your credentials",
	},
	UserError.NewPasswordInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "New password is invalid",
		Description: "The new password provided does not meet the required criteria",
	},
	UserError.EmailRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Email is required",
		Description: "An email address must be provided",
	},
	UserError.EmailInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid email address",
		Description: "The provided email address does not match the required format",
	},
	UserError.FirstNameRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "First name is required",
		Description: "A first name must be provided",
	},
	UserError.FirstNameInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid first name",
		Description: "The provided first name is invalid",
	},
	UserError.LastNameRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Last name is required",
		Description: "A last name must be provided",
	},
	UserError.LastNameInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid last name",
		Description: "The provided last name is invalid",
	},
	UserError.RoleIDRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Role ID is required",
		Description: "A role identifier must be provided",
	},
	UserError.RoleIDInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid role ID",
		Description: "The provided role identifier is not valid or not a proper UUID",
	},
	UserError.UserIDRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "User ID is required",
		Description: "A user identifier must be provided",
	},
	UserError.UserIDInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid user ID",
		Description: "The provided user identifier is not valid or not a proper UUID",
	},
}

var UserSuccess = struct {
	UserCreated              ResponseCode
	UserUpdated              ResponseCode
	UserDeleted              ResponseCode
	UserFetched              ResponseCode
	PasswordUpdated          ResponseCode
	AvatarAdded              ResponseCode
	AvatarRemoved            ResponseCode
	UserAdded                ResponseCode
	UserRemoved              ResponseCode
	UsersFetched             ResponseCode
	UserAssignedToWorkspace  ResponseCode
	WorkspaceFetched         ResponseCode
	UserRemovedFromWorkspace ResponseCode
	UserAccessDetailsFetched ResponseCode
}{
	UserCreated:              "USR_SUCCESS_2001",
	UserUpdated:              "USR_SUCCESS_2002",
	UserDeleted:              "USR_SUCCESS_2003",
	UserFetched:              "USR_SUCCESS_2004",
	PasswordUpdated:          "USR_SUCCESS_2005",
	AvatarAdded:              "USR_SUCCESS_2006",
	AvatarRemoved:            "USR_SUCCESS_2007",
	UserAdded:                "USR_SUCCESS_2008",
	UserRemoved:              "USR_SUCCESS_2009",
	UsersFetched:             "USR_SUCCESS_2010",
	UserAssignedToWorkspace:  "USR_SUCCESS_2011",
	WorkspaceFetched:         "USR_SUCCESS_2012",
	UserRemovedFromWorkspace: "USR_SUCCESS_2013",
	UserAccessDetailsFetched: "USR_SUCCESS_2014",
}

var UserSuccessCodes = map[ResponseCode]MetaResponse{
	UserSuccess.UserCreated: {
		HTTPStatus:  http.StatusCreated,
		Message:     "User created successfully",
		Description: "The user has been created successfully",
	},
	UserSuccess.UserUpdated: {
		HTTPStatus:  http.StatusOK,
		Message:     "User updated successfully",
		Description: "The user has been updated successfully",
	},
	UserSuccess.UserDeleted: {
		HTTPStatus:  http.StatusOK,
		Message:     "User deleted successfully",
		Description: "The user has been deleted successfully",
	},
	UserSuccess.UserFetched: {
		HTTPStatus:  http.StatusOK,
		Message:     "User fetched successfully",
		Description: "The user has been fetched successfully",
	},
	UserSuccess.PasswordUpdated: {
		HTTPStatus:  http.StatusOK,
		Message:     "Password updated successfully",
		Description: "The user's password has been updated successfully",
	},
	UserSuccess.AvatarAdded: {
		HTTPStatus:  http.StatusOK,
		Message:     "Avatar added successfully",
		Description: "The user's avatar has been added successfully",
	},
	UserSuccess.AvatarRemoved: {
		HTTPStatus:  http.StatusOK,
		Message:     "Avatar removed successfully",
		Description: "The user's avatar has been removed successfully",
	},
	UserSuccess.UserAdded: {
		HTTPStatus:  http.StatusCreated,
		Message:     "User added successfully",
		Description: "The user has been added to the tenant successfully",
	},
	UserSuccess.UserRemoved: {
		HTTPStatus:  http.StatusOK,
		Message:     "User removed successfully",
		Description: "The user has been removed from the tenant successfully",
	},
	UserSuccess.UsersFetched: {
		HTTPStatus:  http.StatusOK,
		Message:     "Users fetched successfully",
		Description: "The users have been fetched successfully",
	},
	UserSuccess.UserAssignedToWorkspace: {
		HTTPStatus:  http.StatusCreated,
		Message:     "User assigned to workspace successfully",
		Description: "The user has been successfully assigned to the workspace",
	},
	UserSuccess.WorkspaceFetched: {
		HTTPStatus:  http.StatusOK,
		Message:     "Workspaces fetched successfully",
		Description: "The user's workspaces have been fetched successfully",
	},
	UserSuccess.UserRemovedFromWorkspace: {
		HTTPStatus:  http.StatusOK,
		Message:     "User removed from workspace successfully",
		Description: "The user has been removed from the workspace successfully",
	},
	UserSuccess.UserAccessDetailsFetched: {
		HTTPStatus:  http.StatusOK,
		Message:     "User access details fetched successfully",
		Description: "The user's workspace and base access details have been fetched successfully",
	},
}
