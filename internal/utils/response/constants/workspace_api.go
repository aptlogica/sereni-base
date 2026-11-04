// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package constants

import "net/http"

var WorkspaceAPIError = struct {
	PublicIDFailed ResponseCode
}{
	PublicIDFailed: "WSA_1001",
}

var WorkspaceAPIErrorCodes = map[ResponseCode]MetaResponse{
	WorkspaceAPIError.PublicIDFailed: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "The workspace API could not be turned on",
		Description: "No unused workspace API ID was found after several attempts; try again",
	},
}

var WorkspaceAPISuccess = struct {
	Fetched ResponseCode
	Enabled ResponseCode
}{
	Fetched: "WSA_SUCCESS_2001",
	Enabled: "WSA_SUCCESS_2002",
}

var WorkspaceAPISuccessCodes = map[ResponseCode]MetaResponse{
	WorkspaceAPISuccess.Fetched: {
		HTTPStatus:  http.StatusOK,
		Message:     "Workspace API status fetched successfully",
		Description: "Whether the workspace API is on, and its address once it has been turned on",
	},
	WorkspaceAPISuccess.Enabled: {
		HTTPStatus:  http.StatusOK,
		Message:     "Workspace API enabled successfully",
		Description: "The workspace API is on and the workspace has its permanent API ID",
	},
}
