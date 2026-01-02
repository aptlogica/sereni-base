package validators

import (
	responseConst "serenibase/internal/utils/response/constants"

	"github.com/go-playground/validator"
)

// OrganizationCreationValidationError maps validation errors for dto.CreateOrganizationRequest to response codes.
func OrganizationCreationValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "Name":
		switch tag {
		case "required":
			return responseConst.Error.ValidationFailed
		default:
			return responseConst.Error.ValidationFailed
		}
	case "Email":
		switch tag {
		case "required":
			return responseConst.Error.ValidationFailed
		case "email":
			return responseConst.Error.ValidationFailed
		default:
			return responseConst.Error.ValidationFailed
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

// OrganizationUpdateValidationError maps validation errors for dto.UpdateOrganizationRequest to response codes.
func OrganizationUpdateValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "Email":
		switch tag {
		case "email":
			return responseConst.Error.ValidationFailed
		default:
			return responseConst.Error.ValidationFailed
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}
