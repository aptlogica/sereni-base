// Copyright (c) 2026 Aptlogica Technologies Private Limited
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package validators

import (
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"

	"github.com/go-playground/validator/v10"
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
