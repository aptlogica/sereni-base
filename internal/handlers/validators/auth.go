// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package validators

import (
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"

	"github.com/go-playground/validator/v10"
)

func validateUserIDField(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "UserID":
		switch tag {
		case "required":
			return responseConst.UserError.UserIDRequired
		default:
			return responseConst.UserError.UserIDInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func RegisterValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "FirstName":
		switch tag {
		case "required":
			return responseConst.AuthError.FirstNameRequired
		default:
			return responseConst.AuthError.FirstNameInvalid
		}
	case "LastName":
		switch tag {
		case "required":
			return responseConst.AuthError.LastNameRequired
		default:
			return responseConst.AuthError.LastNameInvalid
		}
	case "Email":
		switch tag {
		case "required":
			return responseConst.AuthError.EmailRequired
		case "email":
			return responseConst.AuthError.EmailInvalidFormat
		default:
			return responseConst.AuthError.EmailInvalid
		}
	case "Password":
		switch tag {
		case "required":
			return responseConst.AuthError.PasswordRequired
		case "min":
			return responseConst.AuthError.PasswordTooShort
		default:
			return responseConst.AuthError.PasswordInvalid
		}
	case "DateOfBirth":
		switch tag {
		case "required":
			return responseConst.AuthError.DateOfBirthRequired
		default:
			return responseConst.AuthError.DateOfBirthInvalid
		}
	case "Country":
		switch tag {
		case "required":
			return responseConst.AuthError.CountryRequired
		default:
			return responseConst.AuthError.CountryInvalid
		}
	case "Timezone":
		switch tag {
		case "required":
			return responseConst.AuthError.TimezoneRequired
		default:
			return responseConst.AuthError.TimezoneInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func LoginValidationError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "Email":
		switch tag {
		case "required":
			return responseConst.AuthError.EmailRequired
		case "email":
			return responseConst.AuthError.EmailInvalidFormat
		default:
			return responseConst.AuthError.EmailInvalid
		}
	case "Password":
		switch tag {
		case "required":
			return responseConst.AuthError.PasswordRequired
		case "min":
			return responseConst.AuthError.PasswordTooShort
		default:
			return responseConst.AuthError.PasswordInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func VerifyEmailRequestError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "Token":
		switch tag {
		case "required":
			return responseConst.AuthError.TokenRequired
		default:
			return responseConst.AuthError.TokenInvalid
		}
	case "OTP":
		switch tag {
		case "required":
			return responseConst.AuthError.OTPRequired
		default:
			return responseConst.AuthError.OTPInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func VerifyResendOtpRequestError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "Token":
		switch tag {
		case "required":
			return responseConst.AuthError.TokenRequired
		case "jwt":
			return responseConst.AuthError.TokenInvalidFormat
		default:
			return responseConst.AuthError.TokenInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func RefreshTokenRequestError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "RefreshToken":
		switch tag {
		case "required":
			return responseConst.AuthError.RefreshTokenRequired
		default:
			return responseConst.AuthError.RefreshTokenInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func ForgotPasswordRequestError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "Email":
		switch tag {
		case "required":
			return responseConst.AuthError.EmailRequired
		case "email":
			return responseConst.AuthError.EmailInvalidFormat
		default:
			return responseConst.AuthError.EmailInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func ResetPasswordRequestError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "Token":
		switch tag {
		case "required":
			return responseConst.AuthError.TokenRequired
		case "uuid":
			return responseConst.AuthError.TokenInvalidFormat
		default:
			return responseConst.AuthError.TokenInvalid
		}
	case "NewPassword":
		switch tag {
		case "required":
			return responseConst.AuthError.NewPasswordRequired
		default:
			return responseConst.AuthError.NewPasswordInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func ValidateTokenRequestError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "Token":
		switch tag {
		case "required":
			return responseConst.AuthError.TokenRequired
		default:
			return responseConst.AuthError.TokenInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func LogoutRequestError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "Token":
		switch tag {
		case "required":
			return responseConst.AuthError.TokenRequired
		case "uuid":
			return responseConst.AuthError.TokenInvalidFormat
		default:
			return responseConst.AuthError.TokenInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func AddUserRequestError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "Email":
		switch tag {
		case "required":
			return responseConst.UserError.EmailRequired
		case "email":
			return responseConst.UserError.EmailInvalid
		default:
			return responseConst.UserError.EmailInvalid
		}
	case "FirstName":
		switch tag {
		case "required":
			return responseConst.UserError.FirstNameRequired
		default:
			return responseConst.UserError.FirstNameInvalid
		}
	case "LastName":
		switch tag {
		case "required":
			return responseConst.UserError.LastNameRequired
		default:
			return responseConst.UserError.LastNameInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func RemoveUserRequestError(e validator.FieldError) responseConst.ResponseCode {
	return validateUserIDField(e)
}

func CreateMemberRequestError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "WorkspaceID":
		switch tag {
		case "required":
			return responseConst.WorkspaceError.IdRequired
		default:
			return responseConst.WorkspaceError.IdInvalid
		}
	case "UserID":
		switch tag {
		case "required":
			return responseConst.UserError.UserIDRequired
		default:
			return responseConst.UserError.UserIDInvalid
		}
	case "AccessLevel":
		switch tag {
		case "required":
			return responseConst.RoleError.RoleRequired
		default:
			return responseConst.RoleError.RoleInvalid
		}
	case "BasesIds":
		switch tag {
		case "required":
			return responseConst.BaseError.IdRequired
		default:
			return responseConst.BaseError.IdInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func RemoveMemberRequestError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "WorkspaceID":
		switch tag {
		case "required":
			return responseConst.WorkspaceError.IdRequired
		default:
			return responseConst.WorkspaceError.IdInvalid
		}
	case "UserID":
		switch tag {
		case "required":
			return responseConst.UserError.UserIDRequired
		default:
			return responseConst.UserError.UserIDInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func AddMultipleMembersRequestError(e validator.FieldError) responseConst.ResponseCode {
	field := e.Field()
	tag := e.Tag()

	switch field {
	case "WorkspaceID":
		switch tag {
		case "required":
			return responseConst.WorkspaceError.IdRequired
		default:
			return responseConst.WorkspaceError.IdInvalid
		}
	case "UserIDs":
		switch tag {
		case "required":
			return responseConst.UserError.UserIDRequired
		case "min":
			return responseConst.Error.ValidationFailed
		default:
			return responseConst.UserError.UserIDInvalid
		}
	case "AccessLevel":
		switch tag {
		case "required":
			return responseConst.RoleError.RoleRequired
		default:
			return responseConst.RoleError.RoleInvalid
		}
	default:
		return responseConst.Error.ValidationFailed
	}
}

func ActivateUserRequestError(e validator.FieldError) responseConst.ResponseCode {
	return validateUserIDField(e)
}

func DeactivateUserRequestError(e validator.FieldError) responseConst.ResponseCode {
	return validateUserIDField(e)
}
