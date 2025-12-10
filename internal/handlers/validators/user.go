package validators

import (
	responseConst "serenibase/internal/utils/response/constants"

	"github.com/go-playground/validator"
)


func UpdateUserPasswordValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "OldPassword":
		switch tag {
		case "required":
			return responseConst.UserError.OldPasswordRequired
		default:
			return responseConst.UserError.OldPasswordInvalid
		}
	case "NewPassword":
		switch tag {
		case "required":
			return responseConst.UserError.NewPasswordRequired
		default:
			return responseConst.UserError.NewPasswordInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}
