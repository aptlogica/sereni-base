// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package constants

import "net/http"

var TableConstraintError = struct {
	ConstraintNotFound     ResponseCode
	TitleRequired          ResponseCode
	TitleTooLong           ResponseCode
	ColumnNotFound         ResponseCode
	ColumnNotAllowed       ResponseCode
	RepeatedColumn         ResponseCode
	AlreadyExists          ResponseCode
	DuplicateValues        ResponseCode
	ConstraintFailed       ResponseCode
	ColumnUsedInConstraint ResponseCode
	UniqueKeyViolation     ResponseCode
	TypeRequired           ResponseCode
	DefaultConflict        ResponseCode
	CheckConditionInvalid  ResponseCode
	CheckCategoryUnknown   ResponseCode
	CheckExistingRows      ResponseCode
	CheckViolation         ResponseCode
}{
	ConstraintNotFound:     "TCN_1001",
	TitleRequired:          "TCN_1002",
	TitleTooLong:           "TCN_1003",
	ColumnNotFound:         "TCN_1004",
	ColumnNotAllowed:       "TCN_1005",
	RepeatedColumn:         "TCN_1007",
	AlreadyExists:          "TCN_1008",
	DuplicateValues:        "TCN_1009",
	ConstraintFailed:       "TCN_1010",
	ColumnUsedInConstraint: "TCN_1011",
	UniqueKeyViolation:     "TCN_1012",
	TypeRequired:           "TCN_1013",
	DefaultConflict:        "TCN_1014",
	CheckConditionInvalid:  "TCN_1015",
	CheckCategoryUnknown:   "TCN_1016",
	CheckExistingRows:      "TCN_1017",
	CheckViolation:         "TCN_1018",
}

var TableConstraintErrorCodes = map[ResponseCode]MetaResponse{
	TableConstraintError.ConstraintNotFound: {
		HTTPStatus:  http.StatusNotFound,
		Message:     "Constraint not found",
		Description: "The requested constraint could not be found",
	},
	TableConstraintError.TitleRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Name is required",
		Description: "The name of the constraint is required",
	},
	TableConstraintError.TitleTooLong: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Name too long",
		Description: "The name of the constraint exceeds the maximum allowed length",
	},
	TableConstraintError.ColumnNotFound: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Column not found",
		Description: "A column of the key does not exist in this table",
	},
	TableConstraintError.ColumnNotAllowed: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Column can't be part of a key",
		Description: "Links, lookups, formulas, attachments, JSON, multi select and system fields can't be part of a key",
	},
	TableConstraintError.RepeatedColumn: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Column repeated",
		Description: "Each column can be in a key only once",
	},
	TableConstraintError.AlreadyExists: {
		HTTPStatus:  http.StatusConflict,
		Message:     "A unique key on these columns already exists",
		Description: "The table already has a unique key on the same columns",
	},
	TableConstraintError.DuplicateValues: {
		HTTPStatus:  http.StatusConflict,
		Message:     "Records repeat values in these columns",
		Description: "Fix the duplicate records before adding the unique key",
	},
	TableConstraintError.ConstraintFailed: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Constraint could not be applied",
		Description: "Postgres could not apply the constraint",
	},
	TableConstraintError.ColumnUsedInConstraint: {
		HTTPStatus:  http.StatusConflict,
		Message:     "Column is used by a unique key",
		Description: "Remove the column from the unique key first",
	},
	TableConstraintError.UniqueKeyViolation: {
		HTTPStatus:  http.StatusConflict,
		Message:     "A record with these values already exists",
		Description: "The record repeats the values of a unique key",
	},
	TableConstraintError.DefaultConflict: {
		HTTPStatus:  http.StatusConflict,
		Message:     "Default value conflicts with a unique key",
		Description: "A column of a unique key can't have a default value, since it would repeat in every new record",
	},
	TableConstraintError.TypeRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Type is required",
		Description: "The constraint type is required, e.g. unique",
	},
	TableConstraintError.CheckConditionInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Condition not allowed",
		Description: "The check condition is empty, too long or contains text that isn't allowed",
	},
	TableConstraintError.CheckCategoryUnknown: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Unknown category",
		Description: "The check category is not one of the known categories",
	},
	TableConstraintError.CheckExistingRows: {
		HTTPStatus:  http.StatusConflict,
		Message:     "Records break this condition",
		Description: "Fix the records or change the condition before adding the check constraint",
	},
	TableConstraintError.CheckViolation: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Value not allowed",
		Description: "The record breaks a check constraint of the table",
	},
}

var TableConstraintSuccess = struct {
	ConstraintCreated  ResponseCode
	ConstraintsFetched ResponseCode
	ConstraintUpdated  ResponseCode
	ConstraintDeleted  ResponseCode
	ConstraintChecked  ResponseCode
}{
	ConstraintCreated:  "TCN_SUCCESS_2001",
	ConstraintsFetched: "TCN_SUCCESS_2002",
	ConstraintUpdated:  "TCN_SUCCESS_2003",
	ConstraintDeleted:  "TCN_SUCCESS_2004",
	ConstraintChecked:  "TCN_SUCCESS_2005",
}

var TableConstraintSuccessCodes = map[ResponseCode]MetaResponse{
	TableConstraintSuccess.ConstraintCreated: {
		HTTPStatus:  http.StatusCreated,
		Message:     "Constraint created successfully",
		Description: "The constraint has been added to the table",
	},
	TableConstraintSuccess.ConstraintsFetched: {
		HTTPStatus:  http.StatusOK,
		Message:     "Constraints fetched successfully",
		Description: "The constraints of the table have been fetched",
	},
	TableConstraintSuccess.ConstraintUpdated: {
		HTTPStatus:  http.StatusOK,
		Message:     "Constraint updated successfully",
		Description: "The constraint has been updated",
	},
	TableConstraintSuccess.ConstraintDeleted: {
		HTTPStatus:  http.StatusOK,
		Message:     "Constraint deleted successfully",
		Description: "The constraint has been removed from the table",
	},
	TableConstraintSuccess.ConstraintChecked: {
		HTTPStatus:  http.StatusOK,
		Message:     "Records checked",
		Description: "The records have been checked against the columns; nothing was changed",
	},
}
