// Copyright (c) 2026 Aptlogica Technologies Private Limited
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package constants

import "net/http"

var AssetError = struct {
	ErrAssetUpload             ResponseCode
	AssetNotFound              ResponseCode
	AssetAlreadyExists         ResponseCode
	AssetNotCreated            ResponseCode
	AssetNotUpdated            ResponseCode
	AssetNotDeleted            ResponseCode
	MultipartFormNotFound      ResponseCode
	FilesNotFound              ResponseCode
	IdsRequired                ResponseCode
	IdsInvalid                 ResponseCode
	InvalidRequest             ResponseCode
	TitleRequired              ResponseCode
	TitleInvalid               ResponseCode
	StorageFileOpenFailed      ResponseCode
	StorageUploadFailed        ResponseCode
	FileTooLargeError          ResponseCode
	MultipleFilesTooLargeError ResponseCode
	TooManyFilesError          ResponseCode
	VirusDetected              ResponseCode
	InvalidFileFormat          ResponseCode
}{
	ErrAssetUpload:             "AST_5000",
	AssetNotFound:              "AST_5001",
	AssetAlreadyExists:         "AST_5002",
	AssetNotCreated:            "AST_5003",
	AssetNotUpdated:            "AST_5004",
	AssetNotDeleted:            "AST_5005",
	MultipartFormNotFound:      "AST_5006",
	FilesNotFound:              "AST_5007",
	IdsRequired:                "AST_5008",
	IdsInvalid:                 "AST_5009",
	InvalidRequest:             "AST_5010",
	TitleRequired:              "AST_5011",
	TitleInvalid:               "AST_5012",
	StorageFileOpenFailed:      "AST_5013",
	StorageUploadFailed:        "AST_5014",
	FileTooLargeError:          "AST_5015",
	MultipleFilesTooLargeError: "AST_5016",
	TooManyFilesError:          "AST_5017",
	VirusDetected:              "AST_5018",
	InvalidFileFormat:          "AST_5019",
}

var AssetErrorCodes = map[ResponseCode]MetaResponse{
	AssetError.ErrAssetUpload: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Asset upload failed",
		Description: "The asset could not be uploaded due to an internal error",
	},
	AssetError.AssetNotFound: {
		HTTPStatus:  http.StatusNotFound,
		Message:     "Asset not found",
		Description: "The specified asset could not be found",
	},
	AssetError.AssetAlreadyExists: {
		HTTPStatus:  http.StatusConflict,
		Message:     "Asset already exists",
		Description: "An asset with the given information already exists",
	},
	AssetError.AssetNotCreated: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Asset not created",
		Description: "The asset could not be created due to an internal error",
	},
	AssetError.AssetNotUpdated: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Asset not updated",
		Description: "The asset could not be updated due to an internal error",
	},
	AssetError.AssetNotDeleted: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Asset not deleted",
		Description: "The asset could not be deleted due to an internal error",
	},
	AssetError.MultipartFormNotFound: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid payload",
		Description: "The multipart form data was not found in the request",
	},
	AssetError.FilesNotFound: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid payload",
		Description: "No files were found in the multipart form data of the request",
	},
	AssetError.IdsRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "IDs required",
		Description: "One or more IDs are required in the request",
	},
	AssetError.IdsInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid IDs",
		Description: "The provided IDs are invalid or malformed",
	},
	AssetError.InvalidRequest: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid request",
		Description: "The request is invalid or malformed",
	},
	AssetError.TitleRequired: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Title required",
		Description: "The title field is required",
	},
	AssetError.TitleInvalid: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid title",
		Description: "The provided title is invalid or malformed",
	},
	AssetError.StorageFileOpenFailed: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Failed to open file for storage",
		Description: "The file could not be opened for storage due to an internal error",
	},
	AssetError.StorageUploadFailed: {
		HTTPStatus:  http.StatusInternalServerError,
		Message:     "Failed to upload file to storage",
		Description: "The file could not be uploaded to storage due to an internal error",
	},
	AssetError.FileTooLargeError: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "File too large",
		Description: "The uploaded file exceeds the maximum allowed size",
	},
	AssetError.MultipleFilesTooLargeError: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "One or more files too large",
		Description: "One or more uploaded files exceed the maximum allowed size",
	},
	AssetError.TooManyFilesError: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Too many files",
		Description: "The number of uploaded files exceeds the allowed limit",
	},
	AssetError.VirusDetected: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Virus detected",
		Description: "The uploaded file contains a virus and was rejected",
	},
	AssetError.InvalidFileFormat: {
		HTTPStatus:  http.StatusBadRequest,
		Message:     "Invalid file format",
		Description: "Only image files are allowed",
	},
}

var AssetSuccess = struct {
	AssetUpload  ResponseCode
	AssetCreated ResponseCode
	AssetUpdated ResponseCode
	AssetDeleted ResponseCode
	AssetFetch   ResponseCode
}{
	AssetUpload:  "AST_SUCCESS_5000",
	AssetCreated: "AST_SUCCESS_5001",
	AssetUpdated: "AST_SUCCESS_5002",
	AssetDeleted: "AST_SUCCESS_5003",
	AssetFetch:   "AST_SUCCESS_5004",
}

var AssetSuccessCodes = map[ResponseCode]MetaResponse{
	AssetSuccess.AssetUpload: {
		HTTPStatus:  http.StatusCreated,
		Message:     "Asset uploaded successfully",
		Description: "The asset has been uploaded successfully",
	},
	AssetSuccess.AssetCreated: {
		HTTPStatus:  http.StatusCreated,
		Message:     "Asset created successfully",
		Description: "The asset has been created successfully",
	},
	AssetSuccess.AssetUpdated: {
		HTTPStatus:  http.StatusOK,
		Message:     "Asset updated successfully",
		Description: "The asset has been updated successfully",
	},
	AssetSuccess.AssetDeleted: {
		HTTPStatus:  http.StatusOK,
		Message:     "Asset deleted successfully",
		Description: "The asset has been deleted successfully",
	},
	AssetSuccess.AssetFetch: {
		HTTPStatus:  http.StatusOK,
		Message:     "Assets fetched successfully",
		Description: "The assets have been fetched successfully",
	},
}
