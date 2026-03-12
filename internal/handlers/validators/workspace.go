// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package validators

import (
	responseConst "serenibase/internal/utils/response/constants"

	"github.com/go-playground/validator"
)

// WorkspaceCreationValidationError maps validation errors for dto.CreateWorkspaceRequest to response codes.
func WorkspaceCreationValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "Title":
		switch tag {
		case "required":
			return responseConst.WorkspaceError.NameRequired
		default:
			return responseConst.WorkspaceError.NameInvalid
		}
	case "Description":
		return responseConst.WorkspaceError.DescriptionInvalid
	default:
		return responseConst.Error.ValidationFailed
	}
}

func WorkspaceUpdateValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "ID":
		switch tag {
		case "required":
			return responseConst.WorkspaceError.IdRequired
		default:
			return responseConst.WorkspaceError.IdInvalid
		}
	case "Title":
		switch tag {
		case "required":
			return responseConst.WorkspaceError.NameRequired
		default:
			return responseConst.WorkspaceError.NameInvalid
		}
	case "Description":
		return responseConst.WorkspaceError.DescriptionInvalid
	default:
		return responseConst.Error.ValidationFailed
	}
}
