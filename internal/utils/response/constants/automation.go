// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package constants

import "net/http"

var AutomationError = struct {
	AutomationNotFound   ResponseCode
	InvalidTriggerQuery  ResponseCode
	TriggerFailed        ResponseCode
	TitleRequired        ResponseCode
	TitleTooLong         ResponseCode
	InvalidFunctionQuery ResponseCode
	TypeInvalid          ResponseCode
}{
	AutomationNotFound:   "AUT_1001",
	InvalidTriggerQuery:  "AUT_1002",
	TriggerFailed:        "AUT_1003",
	TitleRequired:        "AUT_1005",
	TitleTooLong:         "AUT_1006",
	InvalidFunctionQuery: "AUT_1007",
	TypeInvalid:          "AUT_1008",
}

var AutomationErrorCodes = map[ResponseCode]MetaResponse{
	AutomationError.AutomationNotFound: {
		HTTPStatus:  http.StatusNotFound,
		Message:     "Automation not found",
		Description: "The requested trigger or webhook could not be found",
	},
	AutomationError.InvalidTriggerQuery: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid trigger query",
		Description: "Context must be a single CREATE TRIGGER statement on this table",
	},
	AutomationError.TriggerFailed: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Query failed",
		Description: "Postgres could not run the query",
	},
	AutomationError.InvalidFunctionQuery: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid function query",
		Description: "Context must be a single CREATE FUNCTION name() RETURNS trigger statement",
	},
	AutomationError.TypeInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid type",
		Description: "Type must be trigger, webhook or function",
	},
	AutomationError.TitleRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Title is required",
		Description: "The title field is required",
	},
	AutomationError.TitleTooLong: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Title too long",
		Description: "The title exceeds the maximum allowed length",
	},
}

var AutomationSuccess = struct {
	AutomationCreated  ResponseCode
	AutomationsFetched ResponseCode
	AutomationDeleted  ResponseCode
}{
	AutomationCreated:  "AUT_SUCCESS_2001",
	AutomationsFetched: "AUT_SUCCESS_2002",
	AutomationDeleted:  "AUT_SUCCESS_2003",
}

var AutomationSuccessCodes = map[ResponseCode]MetaResponse{
	AutomationSuccess.AutomationCreated: {
		HTTPStatus:  http.StatusCreated,
		Message:     "Automation created successfully",
		Description: "The trigger or webhook has been created successfully",
	},
	AutomationSuccess.AutomationsFetched: {
		HTTPStatus:  http.StatusOK,
		Message:     "Automations fetched successfully",
		Description: "The triggers and webhooks have been fetched successfully",
	},
	AutomationSuccess.AutomationDeleted: {
		HTTPStatus:  http.StatusOK,
		Message:     "Automation deleted successfully",
		Description: "The trigger or webhook has been deleted successfully",
	},
}
