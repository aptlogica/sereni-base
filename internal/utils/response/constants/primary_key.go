// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package constants

import "net/http"

var PrimaryKeyError = struct {
	Required      ResponseCode
	ColumnInvalid ResponseCode
	Locked        ResponseCode
	DataInvalid   ResponseCode
	UpdateFailed  ResponseCode
	ColumnInKey   ResponseCode
	// Writing a record: a key field left empty, or a key already used by another record
	EmptyValue     ResponseCode
	DuplicateValue ResponseCode
}{
	Required:       "PKY_1001",
	ColumnInvalid:  "PKY_1002",
	Locked:         "PKY_1003",
	DataInvalid:    "PKY_1004",
	UpdateFailed:   "PKY_1005",
	ColumnInKey:    "PKY_1006",
	EmptyValue:     "PKY_1007",
	DuplicateValue: "PKY_1008",
}

var PrimaryKeyErrorCodes = map[ResponseCode]MetaResponse{
	PrimaryKeyError.Required: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Select at least one primary key column",
		Description: "A table must always have a primary key",
	},
	PrimaryKeyError.ColumnInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "This field cannot be part of the primary key",
		Description: "Only single-value fields of this table (text, number, date, email, select and similar) can be key fields",
	},
	PrimaryKeyError.Locked: {
		HTTPStatus:  http.StatusConflict,
		Message:     "The primary key cannot be changed while this table has link fields",
		Description: "Link fields store the primary key values of linked records; remove the link fields first",
	},
	PrimaryKeyError.DataInvalid: {
		HTTPStatus:  http.StatusConflict,
		Message:     "Existing records do not satisfy the new primary key",
		Description: "Key fields must be filled in for every record and their combined values must be unique",
	},
	PrimaryKeyError.UpdateFailed: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "The primary key could not be updated",
		Description: "Postgres rejected the primary key change",
	},
	PrimaryKeyError.EmptyValue: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "A primary key field can't be empty",
		Description: "Every primary key field must be filled in; the message names the field",
	},
	PrimaryKeyError.DuplicateValue: {
		HTTPStatus:  http.StatusConflict,
		Message:     "This primary key value is already used by another record",
		Description: "Each record needs a unique primary key; the message names the values and the record that holds them",
	},
	PrimaryKeyError.ColumnInKey: {
		HTTPStatus:  http.StatusConflict,
		Message:     "This field is part of the primary key",
		Description: "Remove the field from the primary key in table settings before deleting it or changing its type",
	},
}

var PrimaryKeySuccess = struct {
	Fetched ResponseCode
	Updated ResponseCode
}{
	Fetched: "PKY_SUCCESS_2001",
	Updated: "PKY_SUCCESS_2002",
}

var PrimaryKeySuccessCodes = map[ResponseCode]MetaResponse{
	PrimaryKeySuccess.Fetched: {
		HTTPStatus:  http.StatusOK,
		Message:     "Primary key fetched successfully",
		Description: "The primary key of the table has been fetched",
	},
	PrimaryKeySuccess.Updated: {
		HTTPStatus:  http.StatusOK,
		Message:     "Primary key updated successfully",
		Description: "The primary key of the table has been updated",
	},
}
