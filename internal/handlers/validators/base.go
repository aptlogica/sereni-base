package validators

import (
	responseConst "serenibase/internal/utils/response/constants"

	"github.com/go-playground/validator"
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
