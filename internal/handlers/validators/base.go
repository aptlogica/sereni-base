// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package validators

import (
	"unicode/utf8"

	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"

	"github.com/go-playground/validator/v10"
)

const maxNameOrTitleLength = 50

// ValidateMaxNameOrTitleLength ensures names/titles do not exceed 50 characters.
func ValidateMaxNameOrTitleLength(value string, tooLongErr responseConst.ResponseCode) (responseConst.ResponseCode, bool) {
	if utf8.RuneCountInString(value) > maxNameOrTitleLength {
		return tooLongErr, true
	}
	return "", false
}

// WorkspaceCreationValidationError maps validation errors for dto.CreateWorkspaceRequest to response codes.
func BaseCreationValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "Title":
		switch tag {
		case "required":
			return responseConst.BaseError.NameRequired
		default:
			return responseConst.BaseError.NameInvalid
		}
	case "Description":
		return responseConst.BaseError.DescriptionInvalid
	default:
		return responseConst.Error.ValidationFailed
	}
}

func BaseUpdateValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "ID":
		switch tag {
		case "required":
			return responseConst.BaseError.IdRequired
		default:
			return responseConst.BaseError.IdInvalid
		}
	case "Title":
		switch tag {
		case "required":
			return responseConst.BaseError.NameRequired
		default:
			return responseConst.BaseError.NameInvalid
		}
	case "Description":
		return responseConst.BaseError.DescriptionInvalid
	default:
		return responseConst.Error.ValidationFailed
	}
}
