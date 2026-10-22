// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package constants

import "net/http"

var UUIDError = struct {
	Invalid       ResponseCode
	ReadOnly      ResponseCode
	ConfigInvalid ResponseCode
}{
	Invalid:       "UID_1001",
	ReadOnly:      "UID_1002",
	ConfigInvalid: "UID_1003",
}

var UUIDErrorCodes = map[ResponseCode]MetaResponse{
	UUIDError.Invalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Not a valid UUID",
		Description: "Use the format 8-4-4-4-12, for example 3f6c2a9e-1b47-4d8a-9e05-7c2b4f1a8d63",
	},
	UUIDError.ReadOnly: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "This field is generated automatically",
		Description: "Auto-generated UUID fields are read-only; their value is created for every new record",
	},
	UUIDError.ConfigInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid UUID settings",
		Description: "Mode must be auto or manual, and an auto field's version must be v7, v4 or v1",
	},
}
