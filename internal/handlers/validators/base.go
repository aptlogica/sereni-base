// Copyright (c) 2026 Aptlogica Technologies Private Limited
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package validators

import (
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"

	"github.com/go-playground/validator/v10"
)

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
