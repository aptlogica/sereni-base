// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package constants

import "net/http"

var BaseAPIError = struct {
	PublicIDFailed    ResponseCode
	LogFilterInvalid  ResponseCode
	LogsNotConfigured ResponseCode
	LogsUnavailable   ResponseCode
	LogsTimeout       ResponseCode
}{
	PublicIDFailed:    "BSA_1001",
	LogFilterInvalid:  "BSA_1002",
	LogsNotConfigured: "BSA_1003",
	LogsUnavailable:   "BSA_1004",
	LogsTimeout:       "BSA_1005",
}

var BaseAPIErrorCodes = map[ResponseCode]MetaResponse{
	BaseAPIError.LogFilterInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid log filter",
		Description: "A filter of the base API log request has a bad value",
	},
	BaseAPIError.LogsNotConfigured: {
		HTTPStatus:  http.StatusServiceUnavailable,
		Message:     "Log service is not configured",
		Description: "LOG_SERVICE_URL is empty, so API logs cannot be read",
	},
	BaseAPIError.LogsUnavailable: {
		HTTPStatus:  http.StatusBadGateway,
		Message:     "Could not read the API logs",
		Description: "The log service could not be reached or failed",
	},
	BaseAPIError.LogsTimeout: {
		HTTPStatus:  http.StatusGatewayTimeout,
		Message:     "Reading the API logs timed out",
		Description: "The log service did not answer in time",
	},
	BaseAPIError.PublicIDFailed: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "The base API could not be turned on",
		Description: "No unused base API ID was found after several attempts; try again",
	},
}

var BaseAPISuccess = struct {
	Fetched ResponseCode
	Enabled ResponseCode
}{
	Fetched: "BSA_SUCCESS_2001",
	Enabled: "BSA_SUCCESS_2002",
}

var BaseAPISuccessCodes = map[ResponseCode]MetaResponse{
	BaseAPISuccess.Fetched: {
		HTTPStatus:  http.StatusOK,
		Message:     "Base API status fetched successfully",
		Description: "Whether the base API is on, and its address once it has been turned on",
	},
	BaseAPISuccess.Enabled: {
		HTTPStatus:  http.StatusOK,
		Message:     "Base API enabled successfully",
		Description: "The base API is on and the base has its permanent API ID",
	},
}
