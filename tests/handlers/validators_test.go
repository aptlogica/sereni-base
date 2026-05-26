package handlers_test

import (
	"reflect"
	"testing"

	ut "github.com/go-playground/universal-translator"
	validator "github.com/go-playground/validator/v10"

	"github.com/aptlogica/sereni-base/internal/dto"
	handlersValidators "github.com/aptlogica/sereni-base/internal/handlers/validators"
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"
	"github.com/google/uuid"
)

type fakeFieldError struct {
	field string
	tag   string
}

var _ validator.FieldError = (*fakeFieldError)(nil)

func (f fakeFieldError) Field() string {
	return f.field
}

func (f fakeFieldError) Error() string {
	return ""
}

func (f fakeFieldError) Tag() string {
	return f.tag
}

func (f fakeFieldError) ActualTag() string {
	return f.tag
}

func (f fakeFieldError) Namespace() string {
	return f.field
}

func (f fakeFieldError) StructNamespace() string {
	return f.field
}

func (f fakeFieldError) StructField() string {
	return f.field
}

func (f fakeFieldError) Value() interface{} {
	return nil
}

func (f fakeFieldError) Param() string {
	return ""
}

func (f fakeFieldError) Kind() reflect.Kind {
	return reflect.Invalid
}

func (f fakeFieldError) Type() reflect.Type {
	return reflect.TypeOf("")
}

func (f fakeFieldError) Translate(ut ut.Translator) string {
	return ""
}

type validationCase struct {
	name  string
	fn    func(validator.FieldError) responseConst.ResponseCode
	field string
	tag   string
	want  responseConst.ResponseCode
}

func runValidationCases(t *testing.T, cases []validationCase) {
	t.Helper()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			fe := fakeFieldError{field: tc.field, tag: tc.tag}
			if got := tc.fn(fe); got != tc.want {
				t.Fatalf("%s: got %s want %s", tc.name, got, tc.want)
			}
		})
	}
}

func TestBaseValidators(t *testing.T) {
	cases := []validationCase{
		{
			name:  "BaseCreationTitleRequired",
			fn:    handlersValidators.BaseCreationValidationError,
			field: "Title",
			tag:   "required",
			want:  responseConst.BaseError.NameRequired,
		},
		{
			name:  "BaseCreationTitleInvalid",
			fn:    handlersValidators.BaseCreationValidationError,
			field: "Title",
			tag:   "max",
			want:  responseConst.BaseError.NameInvalid,
		},
		{
			name:  "BaseCreationDescriptionInvalid",
			fn:    handlersValidators.BaseCreationValidationError,
			field: "Description",
			tag:   "pattern",
			want:  responseConst.BaseError.DescriptionInvalid,
		},
		{
			name:  "BaseCreationUnknownField",
			fn:    handlersValidators.BaseCreationValidationError,
			field: "Unknown",
			tag:   "required",
			want:  responseConst.Error.ValidationFailed,
		},
		{
			name:  "BaseUpdateIDRequired",
			fn:    handlersValidators.BaseUpdateValidationError,
			field: "ID",
			tag:   "required",
			want:  responseConst.BaseError.IdRequired,
		},
		{
			name:  "BaseUpdateIDInvalid",
			fn:    handlersValidators.BaseUpdateValidationError,
			field: "ID",
			tag:   "uuid",
			want:  responseConst.BaseError.IdInvalid,
		},
		{
			name:  "BaseUpdateTitleRequired",
			fn:    handlersValidators.BaseUpdateValidationError,
			field: "Title",
			tag:   "required",
			want:  responseConst.BaseError.NameRequired,
		},
		{
			name:  "BaseUpdateTitleInvalid",
			fn:    handlersValidators.BaseUpdateValidationError,
			field: "Title",
			tag:   "len",
			want:  responseConst.BaseError.NameInvalid,
		},
		{
			name:  "BaseUpdateDescriptionInvalid",
			fn:    handlersValidators.BaseUpdateValidationError,
			field: "Description",
			tag:   "pattern",
			want:  responseConst.BaseError.DescriptionInvalid,
		},
		{
			name:  "BaseUpdateUnknownField",
			fn:    handlersValidators.BaseUpdateValidationError,
			field: "Missing",
			tag:   "required",
			want:  responseConst.Error.ValidationFailed,
		},
	}

	runValidationCases(t, cases)
}

func TestAssetValidators(t *testing.T) {
	cases := []validationCase{
		{
			name:  "AssetBulkIdsRequired",
			fn:    handlersValidators.BulkInsertValidationError,
			field: "IDs",
			tag:   "required",
			want:  responseConst.AssetError.IdsRequired,
		},
		{
			name:  "AssetBulkIdsInvalid",
			fn:    handlersValidators.BulkInsertValidationError,
			field: "IDs",
			tag:   "uuid",
			want:  responseConst.AssetError.IdsInvalid,
		},
		{
			name:  "AssetBulkUnknown",
			fn:    handlersValidators.BulkInsertValidationError,
			field: "Other",
			tag:   "required",
			want:  responseConst.Error.ValidationFailed,
		},
		{
			name:  "AssetUpdateTitleRequired",
			fn:    handlersValidators.UpdateValidationError,
			field: "Title",
			tag:   "required",
			want:  responseConst.AssetError.TitleRequired,
		},
		{
			name:  "AssetUpdateTitleInvalid",
			fn:    handlersValidators.UpdateValidationError,
			field: "Title",
			tag:   "pattern",
			want:  responseConst.AssetError.TitleInvalid,
		},
		{
			name:  "AssetUpdateUnknownField",
			fn:    handlersValidators.UpdateValidationError,
			field: "Name",
			tag:   "required",
			want:  responseConst.Error.ValidationFailed,
		},
	}

	runValidationCases(t, cases)
}

func TestAuthValidationErrors(t *testing.T) {
	t.Run("Register", func(t *testing.T) {
		cases := []validationCase{
			{name: "RegisterFirstNameRequired", fn: handlersValidators.RegisterValidationError, field: "FirstName", tag: "required", want: responseConst.AuthError.FirstNameRequired},
			{name: "RegisterFirstNameInvalid", fn: handlersValidators.RegisterValidationError, field: "FirstName", tag: "min", want: responseConst.AuthError.FirstNameInvalid},
			{name: "RegisterLastNameRequired", fn: handlersValidators.RegisterValidationError, field: "LastName", tag: "required", want: responseConst.AuthError.LastNameRequired},
			{name: "RegisterEmailRequired", fn: handlersValidators.RegisterValidationError, field: "Email", tag: "required", want: responseConst.AuthError.EmailRequired},
			{name: "RegisterEmailInvalidFormat", fn: handlersValidators.RegisterValidationError, field: "Email", tag: "email", want: responseConst.AuthError.EmailInvalidFormat},
			{name: "RegisterEmailInvalid", fn: handlersValidators.RegisterValidationError, field: "Email", tag: "regex", want: responseConst.AuthError.EmailInvalid},
			{name: "RegisterPasswordRequired", fn: handlersValidators.RegisterValidationError, field: "Password", tag: "required", want: responseConst.AuthError.PasswordRequired},
			{name: "RegisterPasswordTooShort", fn: handlersValidators.RegisterValidationError, field: "Password", tag: "min", want: responseConst.AuthError.PasswordTooShort},
			{name: "RegisterPasswordInvalid", fn: handlersValidators.RegisterValidationError, field: "Password", tag: "len", want: responseConst.AuthError.PasswordInvalid},
			{name: "RegisterDateOfBirthInvalid", fn: handlersValidators.RegisterValidationError, field: "DateOfBirth", tag: "format", want: responseConst.AuthError.DateOfBirthInvalid},
			{name: "RegisterCountryInvalid", fn: handlersValidators.RegisterValidationError, field: "Country", tag: "alpha", want: responseConst.AuthError.CountryInvalid},
			{name: "RegisterTimezoneInvalid", fn: handlersValidators.RegisterValidationError, field: "Timezone", tag: "timezone", want: responseConst.AuthError.TimezoneInvalid},
			{name: "RegisterUnknown", fn: handlersValidators.RegisterValidationError, field: "Unknown", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("Login", func(t *testing.T) {
		cases := []validationCase{
			{name: "LoginEmailRequired", fn: handlersValidators.LoginValidationError, field: "Email", tag: "required", want: responseConst.AuthError.EmailRequired},
			{name: "LoginEmailInvalidFormat", fn: handlersValidators.LoginValidationError, field: "Email", tag: "email", want: responseConst.AuthError.EmailInvalidFormat},
			{name: "LoginPasswordRequired", fn: handlersValidators.LoginValidationError, field: "Password", tag: "required", want: responseConst.AuthError.PasswordRequired},
			{name: "LoginPasswordInvalid", fn: handlersValidators.LoginValidationError, field: "Password", tag: "len", want: responseConst.AuthError.PasswordInvalid},
			{name: "LoginUnknown", fn: handlersValidators.LoginValidationError, field: "Token", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("VerifyEmail", func(t *testing.T) {
		cases := []validationCase{
			{name: "VerifyEmailTokenRequired", fn: handlersValidators.VerifyEmailRequestError, field: "Token", tag: "required", want: responseConst.AuthError.TokenRequired},
			{name: "VerifyEmailOtpInvalid", fn: handlersValidators.VerifyEmailRequestError, field: "OTP", tag: "digits", want: responseConst.AuthError.OTPInvalid},
			{name: "VerifyEmailUnknown", fn: handlersValidators.VerifyEmailRequestError, field: "Foo", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("VerifyResendOtp", func(t *testing.T) {
		cases := []validationCase{
			{name: "VerifyResendTokenRequired", fn: handlersValidators.VerifyResendOtpRequestError, field: "Token", tag: "required", want: responseConst.AuthError.TokenRequired},
			{name: "VerifyResendTokenInvalidFormat", fn: handlersValidators.VerifyResendOtpRequestError, field: "Token", tag: "jwt", want: responseConst.AuthError.TokenInvalidFormat},
			{name: "VerifyResendUnknown", fn: handlersValidators.VerifyResendOtpRequestError, field: "OTP", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("RefreshToken", func(t *testing.T) {
		cases := []validationCase{
			{name: "RefreshTokenRequired", fn: handlersValidators.RefreshTokenRequestError, field: "RefreshToken", tag: "required", want: responseConst.AuthError.RefreshTokenRequired},
			{name: "RefreshTokenInvalid", fn: handlersValidators.RefreshTokenRequestError, field: "RefreshToken", tag: "format", want: responseConst.AuthError.RefreshTokenInvalid},
			{name: "RefreshTokenUnknown", fn: handlersValidators.RefreshTokenRequestError, field: "Token", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("ForgotPassword", func(t *testing.T) {
		cases := []validationCase{
			{name: "ForgotPasswordEmailRequired", fn: handlersValidators.ForgotPasswordRequestError, field: "Email", tag: "required", want: responseConst.AuthError.EmailRequired},
			{name: "ForgotPasswordEmailInvalidFormat", fn: handlersValidators.ForgotPasswordRequestError, field: "Email", tag: "email", want: responseConst.AuthError.EmailInvalidFormat},
			{name: "ForgotPasswordUnknown", fn: handlersValidators.ForgotPasswordRequestError, field: "UserID", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("ResetPassword", func(t *testing.T) {
		cases := []validationCase{
			{name: "ResetTokenRequired", fn: handlersValidators.ResetPasswordRequestError, field: "Token", tag: "required", want: responseConst.AuthError.TokenRequired},
			{name: "ResetTokenInvalidFormat", fn: handlersValidators.ResetPasswordRequestError, field: "Token", tag: "uuid", want: responseConst.AuthError.TokenInvalidFormat},
			{name: "ResetNewPasswordRequired", fn: handlersValidators.ResetPasswordRequestError, field: "NewPassword", tag: "required", want: responseConst.AuthError.NewPasswordRequired},
			{name: "ResetNewPasswordInvalid", fn: handlersValidators.ResetPasswordRequestError, field: "NewPassword", tag: "len", want: responseConst.AuthError.NewPasswordInvalid},
			{name: "ResetUnknown", fn: handlersValidators.ResetPasswordRequestError, field: "Email", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("ValidateToken", func(t *testing.T) {
		cases := []validationCase{
			{name: "ValidateTokenRequired", fn: handlersValidators.ValidateTokenRequestError, field: "Token", tag: "required", want: responseConst.AuthError.TokenRequired},
			{name: "ValidateTokenInvalid", fn: handlersValidators.ValidateTokenRequestError, field: "Token", tag: "invalid", want: responseConst.AuthError.TokenInvalid},
			{name: "ValidateTokenUnknown", fn: handlersValidators.ValidateTokenRequestError, field: "Foo", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("Logout", func(t *testing.T) {
		cases := []validationCase{
			{name: "LogoutTokenRequired", fn: handlersValidators.LogoutRequestError, field: "Token", tag: "required", want: responseConst.AuthError.TokenRequired},
			{name: "LogoutTokenInvalidFormat", fn: handlersValidators.LogoutRequestError, field: "Token", tag: "uuid", want: responseConst.AuthError.TokenInvalidFormat},
			{name: "LogoutTokenInvalid", fn: handlersValidators.LogoutRequestError, field: "Token", tag: "len", want: responseConst.AuthError.TokenInvalid},
			{name: "LogoutUnknown", fn: handlersValidators.LogoutRequestError, field: "OTP", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("UserLifecycle", func(t *testing.T) {
		cases := []validationCase{
			{name: "AddUserEmailRequired", fn: handlersValidators.AddUserRequestError, field: "Email", tag: "required", want: responseConst.UserError.EmailRequired},
			{name: "AddUserEmailInvalid", fn: handlersValidators.AddUserRequestError, field: "Email", tag: "regex", want: responseConst.UserError.EmailInvalid},
			{name: "AddUserFirstNameRequired", fn: handlersValidators.AddUserRequestError, field: "FirstName", tag: "required", want: responseConst.UserError.FirstNameRequired},
			{name: "AddUserLastNameInvalid", fn: handlersValidators.AddUserRequestError, field: "LastName", tag: "len", want: responseConst.UserError.LastNameInvalid},
			{name: "AddUserUnknown", fn: handlersValidators.AddUserRequestError, field: "Role", tag: "required", want: responseConst.Error.ValidationFailed},
			{name: "RemoveUserIDRequired", fn: handlersValidators.RemoveUserRequestError, field: "UserID", tag: "required", want: responseConst.UserError.UserIDRequired},
			{name: "RemoveUserIDInvalid", fn: handlersValidators.RemoveUserRequestError, field: "UserID", tag: "uuid", want: responseConst.UserError.UserIDInvalid},
			{name: "RemoveUserUnknown", fn: handlersValidators.RemoveUserRequestError, field: "Email", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("Membership", func(t *testing.T) {
		cases := []validationCase{
			{name: "CreateMemberWorkspaceRequired", fn: handlersValidators.CreateMemberRequestError, field: "WorkspaceID", tag: "required", want: responseConst.WorkspaceError.IdRequired},
			{name: "CreateMemberUserInvalid", fn: handlersValidators.CreateMemberRequestError, field: "UserID", tag: "uuid", want: responseConst.UserError.UserIDInvalid},
			{name: "CreateMemberAccessLevelRequired", fn: handlersValidators.CreateMemberRequestError, field: "AccessLevel", tag: "required", want: responseConst.RoleError.RoleRequired},
			{name: "CreateMemberBasesInvalid", fn: handlersValidators.CreateMemberRequestError, field: "BasesIds", tag: "uuid", want: responseConst.BaseError.IdInvalid},
			{name: "RemoveMemberBaseInvalid", fn: handlersValidators.RemoveMemberRequestError, field: "BaseID", tag: "uuid", want: responseConst.BaseError.IdInvalid},
			{name: "RemoveMemberUserRequired", fn: handlersValidators.RemoveMemberRequestError, field: "UserID", tag: "required", want: responseConst.UserError.UserIDRequired},
			{name: "AddMultipleWorkspaceRequired", fn: handlersValidators.AddMultipleMembersRequestError, field: "WorkspaceID", tag: "required", want: responseConst.WorkspaceError.IdRequired},
			{name: "AddMultipleUserIDsMin", fn: handlersValidators.AddMultipleMembersRequestError, field: "UserIDs", tag: "min", want: responseConst.Error.ValidationFailed},
			{name: "AddMultipleAccessLevelInvalid", fn: handlersValidators.AddMultipleMembersRequestError, field: "AccessLevel", tag: "regex", want: responseConst.RoleError.RoleInvalid},
			{name: "MembershipUnknown", fn: handlersValidators.AddMultipleMembersRequestError, field: "Foo", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("UserActivation", func(t *testing.T) {
		cases := []validationCase{
			{name: "ActivateRequired", fn: handlersValidators.ActivateUserRequestError, field: "UserID", tag: "required", want: responseConst.UserError.UserIDRequired},
			{name: "ActivateInvalid", fn: handlersValidators.ActivateUserRequestError, field: "UserID", tag: "uuid", want: responseConst.UserError.UserIDInvalid},
			{name: "DeactivateRequired", fn: handlersValidators.DeactivateUserRequestError, field: "UserID", tag: "required", want: responseConst.UserError.UserIDRequired},
			{name: "DeactivateInvalid", fn: handlersValidators.DeactivateUserRequestError, field: "UserID", tag: "uuid", want: responseConst.UserError.UserIDInvalid},
			{name: "ActivationUnknown", fn: handlersValidators.ActivateUserRequestError, field: "Foo", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})
}

func TestOrganizationValidators(t *testing.T) {
	t.Run("Creation", func(t *testing.T) {
		cases := []validationCase{
			{name: "OrganizationCreationNameRequired", fn: handlersValidators.OrganizationCreationValidationError, field: "Name", tag: "required", want: responseConst.Error.ValidationFailed},
			{name: "OrganizationCreationEmailInvalid", fn: handlersValidators.OrganizationCreationValidationError, field: "Email", tag: "email", want: responseConst.Error.ValidationFailed},
			{name: "OrganizationCreationUnknown", fn: handlersValidators.OrganizationCreationValidationError, field: "Foo", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("Update", func(t *testing.T) {
		cases := []validationCase{
			{name: "OrganizationUpdateEmailInvalid", fn: handlersValidators.OrganizationUpdateValidationError, field: "Email", tag: "email", want: responseConst.Error.ValidationFailed},
			{name: "OrganizationUpdateUnknown", fn: handlersValidators.OrganizationUpdateValidationError, field: "Name", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})
}

func TestTableValidators(t *testing.T) {
	t.Run("CreateTable", func(t *testing.T) {
		cases := []validationCase{
			{name: "CreateTableBaseIDRequired", fn: handlersValidators.CreateTableValidationErrors, field: "BaseID", tag: "required", want: responseConst.TableError.BaseIDRequired},
			{name: "CreateTableBaseIDInvalid", fn: handlersValidators.CreateTableValidationErrors, field: "BaseID", tag: "uuid", want: responseConst.TableError.BaseIDInvalid},
			{name: "CreateTableWorkspaceRequired", fn: handlersValidators.CreateTableValidationErrors, field: "WorkspaceID", tag: "required", want: responseConst.TableError.WorkspaceIDRequired},
			{name: "CreateTableWorkspaceInvalid", fn: handlersValidators.CreateTableValidationErrors, field: "WorkspaceID", tag: "uuid", want: responseConst.TableError.WorkspaceIDInvalid},
			{name: "CreateTableTitleRequired", fn: handlersValidators.CreateTableValidationErrors, field: "Title", tag: "required", want: responseConst.TableError.TitleRequired},
			{name: "CreateTableTitleInvalid", fn: handlersValidators.CreateTableValidationErrors, field: "Title", tag: "alpha", want: responseConst.TableError.TitleInvalid},
			{name: "CreateTableDescriptionRequired", fn: handlersValidators.CreateTableValidationErrors, field: "Description", tag: "required", want: responseConst.TableError.DescriptionRequired},
			{name: "CreateTableDescriptionInvalid", fn: handlersValidators.CreateTableValidationErrors, field: "Description", tag: "max", want: responseConst.TableError.DescriptionInvalid},
			{name: "CreateTableOrderIndexRequired", fn: handlersValidators.CreateTableValidationErrors, field: "OrderIndex", tag: "required", want: responseConst.TableError.OrderIndexRequired},
			{name: "CreateTableOrderIndexInvalid", fn: handlersValidators.CreateTableValidationErrors, field: "OrderIndex", tag: "min", want: responseConst.TableError.OrderIndexInvalid},
			{name: "CreateTableUnknown", fn: handlersValidators.CreateTableValidationErrors, field: "Foo", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("AddColumn", func(t *testing.T) {
		cases := []validationCase{
			{name: "AddColumnModelIDRequired", fn: handlersValidators.AddColumnValidator, field: "ModelID", tag: "required", want: responseConst.TableError.ModelIDRequired},
			{name: "AddColumnModelIDInvalid", fn: handlersValidators.AddColumnValidator, field: "ModelID", tag: "uuid", want: responseConst.TableError.ModelIDInvalid},
			{name: "AddColumnMetaRequired", fn: handlersValidators.AddColumnValidator, field: "Meta", tag: "required", want: responseConst.TableError.DTRequired},
			{name: "AddColumnMetaInvalid", fn: handlersValidators.AddColumnValidator, field: "Meta", tag: "json", want: responseConst.TableError.DTInvalid},
			{name: "AddColumnVirtualRequired", fn: handlersValidators.AddColumnValidator, field: "Virtual", tag: "required", want: responseConst.TableError.VirtualRequired},
			{name: "AddColumnVirtualInvalid", fn: handlersValidators.AddColumnValidator, field: "Virtual", tag: "bool", want: responseConst.TableError.VirtualInvalid},
			{name: "AddColumnSystemRequired", fn: handlersValidators.AddColumnValidator, field: "System", tag: "required", want: responseConst.TableError.SystemRequired},
			{name: "AddColumnSystemInvalid", fn: handlersValidators.AddColumnValidator, field: "System", tag: "bool", want: responseConst.TableError.SystemInvalid},
			{name: "AddColumnOrderIndexInvalid", fn: handlersValidators.AddColumnValidator, field: "OrderIndex", tag: "min", want: responseConst.TableError.OrderIndexInvalid},
			{name: "AddColumnUnknown", fn: handlersValidators.AddColumnValidator, field: "Foo", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("CreateView", func(t *testing.T) {
		cases := []validationCase{
			{name: "CreateViewModelIDRequired", fn: handlersValidators.CreateViewValidationError, field: "ModelID", tag: "required", want: responseConst.TableError.ModelIDRequired},
			{name: "CreateViewModelIDInvalid", fn: handlersValidators.CreateViewValidationError, field: "ModelID", tag: "uuid", want: responseConst.TableError.ModelIDInvalid},
			{name: "CreateViewMetaRequired", fn: handlersValidators.CreateViewValidationError, field: "Meta", tag: "required", want: responseConst.TableError.MetaRequired},
			{name: "CreateViewMetaInvalid", fn: handlersValidators.CreateViewValidationError, field: "Meta", tag: "json", want: responseConst.TableError.MetaInvalid},
			{name: "CreateViewBaseIDRequired", fn: handlersValidators.CreateViewValidationError, field: "BaseID", tag: "required", want: responseConst.TableError.BaseIDRequired},
			{name: "CreateViewBaseIDInvalid", fn: handlersValidators.CreateViewValidationError, field: "BaseID", tag: "uuid", want: responseConst.TableError.BaseIDInvalid},
			{name: "CreateViewTitleRequired", fn: handlersValidators.CreateViewValidationError, field: "Title", tag: "required", want: responseConst.TableError.TitleRequired},
			{name: "CreateViewTitleInvalid", fn: handlersValidators.CreateViewValidationError, field: "Title", tag: "alpha", want: responseConst.TableError.TitleInvalid},
			{name: "CreateViewDescriptionRequired", fn: handlersValidators.CreateViewValidationError, field: "Description", tag: "required", want: responseConst.TableError.DescriptionRequired},
			{name: "CreateViewDescriptionInvalid", fn: handlersValidators.CreateViewValidationError, field: "Description", tag: "max", want: responseConst.TableError.DescriptionInvalid},
			{name: "CreateViewTypeRequired", fn: handlersValidators.CreateViewValidationError, field: "Type", tag: "required", want: responseConst.TableError.TypeRequired},
			{name: "CreateViewTypeInvalid", fn: handlersValidators.CreateViewValidationError, field: "Type", tag: "oneof", want: responseConst.TableError.TypeInvalid},
			{name: "CreateViewOrderIndexRequired", fn: handlersValidators.CreateViewValidationError, field: "OrderIndex", tag: "required", want: responseConst.TableError.OrderIndexRequired},
			{name: "CreateViewOrderIndexInvalid", fn: handlersValidators.CreateViewValidationError, field: "OrderIndex", tag: "min", want: responseConst.TableError.OrderIndexInvalid},
			{name: "CreateViewUnknown", fn: handlersValidators.CreateViewValidationError, field: "Foo", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("CreateRow", func(t *testing.T) {
		cases := []validationCase{
			{name: "CreateRowModelIDRequired", fn: handlersValidators.CreateRowRequestValidationError, field: "ModelID", tag: "required", want: responseConst.TableError.ModelIDRequired},
			{name: "CreateRowModelIDInvalid", fn: handlersValidators.CreateRowRequestValidationError, field: "ModelID", tag: "uuid", want: responseConst.TableError.ModelIDInvalid},
			{name: "CreateRowUnknown", fn: handlersValidators.CreateRowRequestValidationError, field: "Foo", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("InsertRow", func(t *testing.T) {
		cases := []validationCase{
			{name: "InsertRowModelIDRequired", fn: handlersValidators.InsertRowDataRequestValidationError, field: "ModelID", tag: "required", want: responseConst.TableError.ModelIDRequired},
			{name: "InsertRowModelIDInvalid", fn: handlersValidators.InsertRowDataRequestValidationError, field: "ModelID", tag: "uuid", want: responseConst.TableError.ModelIDInvalid},
			{name: "InsertRowColumnIdRequired", fn: handlersValidators.InsertRowDataRequestValidationError, field: "ColumnId", tag: "required", want: responseConst.TableError.ColumnIdRequired},
			{name: "InsertRowColumnIdInvalid", fn: handlersValidators.InsertRowDataRequestValidationError, field: "ColumnId", tag: "uuid", want: responseConst.TableError.ColumnIdInvalid},
			{name: "InsertRowRowIdRequired", fn: handlersValidators.InsertRowDataRequestValidationError, field: "RowId", tag: "required", want: responseConst.TableError.RowIdRequired},
			{name: "InsertRowRowIdInvalid", fn: handlersValidators.InsertRowDataRequestValidationError, field: "RowId", tag: "uuid", want: responseConst.TableError.RowIdInvalid},
			{name: "InsertRowValueRequired", fn: handlersValidators.InsertRowDataRequestValidationError, field: "Value", tag: "required", want: responseConst.TableError.ValueRequired},
			{name: "InsertRowValueInvalid", fn: handlersValidators.InsertRowDataRequestValidationError, field: "Value", tag: "json", want: responseConst.TableError.ValueInvalid},
			{name: "InsertRowUnknown", fn: handlersValidators.InsertRowDataRequestValidationError, field: "Foo", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("DeleteRow", func(t *testing.T) {
		cases := []validationCase{
			{name: "DeleteRowModelIDRequired", fn: handlersValidators.DeleteRowDataRequestValidationError, field: "ModelID", tag: "required", want: responseConst.TableError.ModelIDRequired},
			{name: "DeleteRowModelIDInvalid", fn: handlersValidators.DeleteRowDataRequestValidationError, field: "ModelID", tag: "uuid", want: responseConst.TableError.ModelIDInvalid},
			{name: "DeleteRowRowIdRequired", fn: handlersValidators.DeleteRowDataRequestValidationError, field: "RowId", tag: "required", want: responseConst.TableError.RowIdRequired},
			{name: "DeleteRowRowIdInvalid", fn: handlersValidators.DeleteRowDataRequestValidationError, field: "RowId", tag: "uuid", want: responseConst.TableError.RowIdInvalid},
			{name: "DeleteRowUnknown", fn: handlersValidators.DeleteRowDataRequestValidationError, field: "Foo", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("BulkDeleteRows", func(t *testing.T) {
		cases := []validationCase{
			{name: "BulkDeleteModelIDRequired", fn: handlersValidators.BulkDeleteRowsRequestValidationError, field: "ModelID", tag: "required", want: responseConst.TableError.ModelIDRequired},
			{name: "BulkDeleteModelIDInvalid", fn: handlersValidators.BulkDeleteRowsRequestValidationError, field: "ModelID", tag: "uuid", want: responseConst.TableError.ModelIDInvalid},
			{name: "BulkDeleteRowIdsRequired", fn: handlersValidators.BulkDeleteRowsRequestValidationError, field: "RowIds", tag: "required", want: responseConst.TableError.RowIdRequired},
			{name: "BulkDeleteRowIdsInvalid", fn: handlersValidators.BulkDeleteRowsRequestValidationError, field: "RowIds", tag: "min", want: responseConst.TableError.RowIdRequired},
			{name: "BulkDeleteRowIdsInvalidValue", fn: handlersValidators.BulkDeleteRowsRequestValidationError, field: "RowIds", tag: "json", want: responseConst.TableError.RowIdInvalid},
			{name: "BulkDeleteUnknown", fn: handlersValidators.BulkDeleteRowsRequestValidationError, field: "Foo", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("UpdateRowDataLinks", func(t *testing.T) {
		cases := []validationCase{
			{name: "UpdateRowLinksModelIDRequired", fn: handlersValidators.UpdateRowDataLinksRequestValidationError, field: "ModelID", tag: "required", want: responseConst.TableError.ModelIDRequired},
			{name: "UpdateRowLinksModelIDInvalid", fn: handlersValidators.UpdateRowDataLinksRequestValidationError, field: "ModelID", tag: "uuid", want: responseConst.TableError.ModelIDInvalid},
			{name: "UpdateRowLinksColumnIdRequired", fn: handlersValidators.UpdateRowDataLinksRequestValidationError, field: "ColumnId", tag: "required", want: responseConst.TableError.ColumnIdRequired},
			{name: "UpdateRowLinksColumnIdInvalid", fn: handlersValidators.UpdateRowDataLinksRequestValidationError, field: "ColumnId", tag: "uuid", want: responseConst.TableError.ColumnIdInvalid},
			{name: "UpdateRowLinksSourceRowIdRequired", fn: handlersValidators.UpdateRowDataLinksRequestValidationError, field: "SourceRowId", tag: "required", want: responseConst.TableError.RowIdRequired},
			{name: "UpdateRowLinksSourceRowIdInvalid", fn: handlersValidators.UpdateRowDataLinksRequestValidationError, field: "SourceRowId", tag: "uuid", want: responseConst.TableError.RowIdInvalid},
			{name: "UpdateRowLinksTargetRowIdInvalid", fn: handlersValidators.UpdateRowDataLinksRequestValidationError, field: "TargetRowId", tag: "uuid", want: responseConst.TableError.RowIdInvalid},
			{name: "UpdateRowLinksActionRequired", fn: handlersValidators.UpdateRowDataLinksRequestValidationError, field: "Action", tag: "required", want: responseConst.TableError.ActionRequired},
			{name: "UpdateRowLinksActionInvalid", fn: handlersValidators.UpdateRowDataLinksRequestValidationError, field: "Action", tag: "oneof", want: responseConst.TableError.ActionInvalid},
			{name: "UpdateRowLinksUnknown", fn: handlersValidators.UpdateRowDataLinksRequestValidationError, field: "Foo", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("RemoveAttachments", func(t *testing.T) {
		cases := []validationCase{
			{name: "RemoveAttachmentsModelIDRequired", fn: handlersValidators.RemoveAttachmentsRequestValidationError, field: "ModelID", tag: "required", want: responseConst.TableError.ModelIDRequired},
			{name: "RemoveAttachmentsModelIDInvalid", fn: handlersValidators.RemoveAttachmentsRequestValidationError, field: "ModelID", tag: "uuid", want: responseConst.TableError.ModelIDInvalid},
			{name: "RemoveAttachmentsColumnIdRequired", fn: handlersValidators.RemoveAttachmentsRequestValidationError, field: "ColumnId", tag: "required", want: responseConst.TableError.ColumnIdRequired},
			{name: "RemoveAttachmentsColumnIdInvalid", fn: handlersValidators.RemoveAttachmentsRequestValidationError, field: "ColumnId", tag: "uuid", want: responseConst.TableError.ColumnIdInvalid},
			{name: "RemoveAttachmentsRowIdRequired", fn: handlersValidators.RemoveAttachmentsRequestValidationError, field: "RowId", tag: "required", want: responseConst.TableError.RowIdRequired},
			{name: "RemoveAttachmentsRowIdInvalid", fn: handlersValidators.RemoveAttachmentsRequestValidationError, field: "RowId", tag: "uuid", want: responseConst.TableError.RowIdInvalid},
			{name: "RemoveAttachmentsRequired", fn: handlersValidators.RemoveAttachmentsRequestValidationError, field: "Attachments", tag: "required", want: responseConst.TableError.AttachmentRequired},
			{name: "RemoveAttachmentsInvalid", fn: handlersValidators.RemoveAttachmentsRequestValidationError, field: "Attachments", tag: "json", want: responseConst.TableError.AttachmentInvalid},
			{name: "RemoveAttachmentsUnknown", fn: handlersValidators.RemoveAttachmentsRequestValidationError, field: "Foo", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("Pagination", func(t *testing.T) {
		cases := []validationCase{
			{name: "PaginationModelIDInvalid", fn: handlersValidators.PaginationRequestValidationError, field: "ModelID", tag: "uuid", want: responseConst.TableError.ModelIDInvalid},
			{name: "PaginationPageSizeRequired", fn: handlersValidators.PaginationRequestValidationError, field: "PageSize", tag: "required", want: responseConst.TableError.LimitRequired},
			{name: "PaginationPageSizeInvalid", fn: handlersValidators.PaginationRequestValidationError, field: "PageSize", tag: "min", want: responseConst.TableError.LimitInvalid},
			{name: "PaginationPageNumberRequired", fn: handlersValidators.PaginationRequestValidationError, field: "PageNumber", tag: "required", want: responseConst.TableError.PageRequired},
			{name: "PaginationPageNumberInvalid", fn: handlersValidators.PaginationRequestValidationError, field: "PageNumber", tag: "uuid", want: responseConst.TableError.PageInvalid},
			{name: "PaginationUnknown", fn: handlersValidators.PaginationRequestValidationError, field: "Foo", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("ReorderColumn", func(t *testing.T) {
		cases := []validationCase{
			{name: "ReorderSourceColumnIDRequired", fn: handlersValidators.ReorderColumnRequestValidationError, field: "SourceColumnID", tag: "required", want: responseConst.TableError.ColumnIdRequired},
			{name: "ReorderSourceColumnIDInvalid", fn: handlersValidators.ReorderColumnRequestValidationError, field: "SourceColumnID", tag: "uuid", want: responseConst.TableError.ColumnIdInvalid},
			{name: "ReorderTargetColumnIDInvalid", fn: handlersValidators.ReorderColumnRequestValidationError, field: "TargetColumnID", tag: "uuid", want: responseConst.TableError.ColumnIdInvalid},
			{name: "ReorderColumnUnknown", fn: handlersValidators.ReorderColumnRequestValidationError, field: "Foo", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("UpdateRow", func(t *testing.T) {
		cases := []validationCase{
			{name: "UpdateRowModelIDRequired", fn: handlersValidators.UpdateRowRequestValidationError, field: "ModelID", tag: "required", want: responseConst.TableError.ModelIDRequired},
			{name: "UpdateRowModelIDInvalid", fn: handlersValidators.UpdateRowRequestValidationError, field: "ModelID", tag: "uuid", want: responseConst.TableError.ModelIDInvalid},
			{name: "UpdateRowRowIdRequired", fn: handlersValidators.UpdateRowRequestValidationError, field: "RowId", tag: "required", want: responseConst.TableError.RowIdRequired},
			{name: "UpdateRowRowIdInvalid", fn: handlersValidators.UpdateRowRequestValidationError, field: "RowId", tag: "uuid", want: responseConst.TableError.RowIdInvalid},
			{name: "UpdateRowUnknown", fn: handlersValidators.UpdateRowRequestValidationError, field: "Foo", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("UpdateAttachment", func(t *testing.T) {
		cases := []validationCase{
			{name: "UpdateAttachmentModelIDRequired", fn: handlersValidators.UpdateAttachmentRequestValidationError, field: "ModelID", tag: "required", want: responseConst.TableError.ModelIDRequired},
			{name: "UpdateAttachmentModelIDInvalid", fn: handlersValidators.UpdateAttachmentRequestValidationError, field: "ModelID", tag: "uuid", want: responseConst.TableError.ModelIDInvalid},
			{name: "UpdateAttachmentColumnIdRequired", fn: handlersValidators.UpdateAttachmentRequestValidationError, field: "ColumnId", tag: "required", want: responseConst.TableError.ColumnIdRequired},
			{name: "UpdateAttachmentColumnIdInvalid", fn: handlersValidators.UpdateAttachmentRequestValidationError, field: "ColumnId", tag: "uuid", want: responseConst.TableError.ColumnIdInvalid},
			{name: "UpdateAttachmentRowIdRequired", fn: handlersValidators.UpdateAttachmentRequestValidationError, field: "RowId", tag: "required", want: responseConst.TableError.RowIdRequired},
			{name: "UpdateAttachmentRowIdInvalid", fn: handlersValidators.UpdateAttachmentRequestValidationError, field: "RowId", tag: "uuid", want: responseConst.TableError.RowIdInvalid},
			{name: "UpdateAttachmentAssetIdRequired", fn: handlersValidators.UpdateAttachmentRequestValidationError, field: "AssetId", tag: "required", want: responseConst.TableError.AssetIdRequired},
			{name: "UpdateAttachmentAssetIdInvalid", fn: handlersValidators.UpdateAttachmentRequestValidationError, field: "AssetId", tag: "uuid", want: responseConst.TableError.AssetIdInvalid},
			{name: "UpdateAttachmentContentRequired", fn: handlersValidators.UpdateAttachmentRequestValidationError, field: "Content", tag: "required", want: responseConst.TableError.ContentRequired},
			{name: "UpdateAttachmentContentInvalid", fn: handlersValidators.UpdateAttachmentRequestValidationError, field: "Content", tag: "json", want: responseConst.TableError.ContentInvalid},
			{name: "UpdateAttachmentUnknown", fn: handlersValidators.UpdateAttachmentRequestValidationError, field: "Foo", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("BulkUpdateColumns", func(t *testing.T) {
		cases := []validationCase{
			{name: "BulkUpdateModelIDRequired", fn: handlersValidators.BulkUpdateColumnsRequestValidationError, field: "ModelID", tag: "required", want: responseConst.TableError.ModelIDRequired},
			{name: "BulkUpdateModelIDInvalid", fn: handlersValidators.BulkUpdateColumnsRequestValidationError, field: "ModelID", tag: "uuid", want: responseConst.TableError.ModelIDInvalid},
			{name: "BulkUpdateUpdatesRequired", fn: handlersValidators.BulkUpdateColumnsRequestValidationError, field: "Updates", tag: "required", want: responseConst.TableError.UpdatesRequired},
			{name: "BulkUpdateUpdatesInvalid", fn: handlersValidators.BulkUpdateColumnsRequestValidationError, field: "Updates", tag: "json", want: responseConst.TableError.UpdatesInvalid},
			{name: "BulkUpdateUnknown", fn: handlersValidators.BulkUpdateColumnsRequestValidationError, field: "Foo", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("ResetColumnValues", func(t *testing.T) {
		cases := []validationCase{
			{name: "ResetColumnModelIDRequired", fn: handlersValidators.ResetColumnValuesRequestValidationError, field: "ModelID", tag: "required", want: responseConst.TableError.ModelIDRequired},
			{name: "ResetColumnModelIDInvalid", fn: handlersValidators.ResetColumnValuesRequestValidationError, field: "ModelID", tag: "uuid", want: responseConst.TableError.ModelIDInvalid},
			{name: "ResetColumnColumnIdRequired", fn: handlersValidators.ResetColumnValuesRequestValidationError, field: "ColumnId", tag: "required", want: responseConst.TableError.ColumnIdRequired},
			{name: "ResetColumnColumnIdInvalid", fn: handlersValidators.ResetColumnValuesRequestValidationError, field: "ColumnId", tag: "uuid", want: responseConst.TableError.ColumnIdInvalid},
			{name: "ResetColumnUnknown", fn: handlersValidators.ResetColumnValuesRequestValidationError, field: "Foo", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})
}

func TestUserValidators(t *testing.T) {
	cases := []validationCase{
		{name: "UpdateUserOldPasswordRequired", fn: handlersValidators.UpdateUserPasswordValidationError, field: "OldPassword", tag: "required", want: responseConst.UserError.OldPasswordRequired},
		{name: "UpdateUserOldPasswordInvalid", fn: handlersValidators.UpdateUserPasswordValidationError, field: "OldPassword", tag: "len", want: responseConst.UserError.OldPasswordInvalid},
		{name: "UpdateUserNewPasswordRequired", fn: handlersValidators.UpdateUserPasswordValidationError, field: "NewPassword", tag: "required", want: responseConst.UserError.NewPasswordRequired},
		{name: "UpdateUserNewPasswordInvalid", fn: handlersValidators.UpdateUserPasswordValidationError, field: "NewPassword", tag: "min", want: responseConst.UserError.NewPasswordInvalid},
		{name: "UpdateUserUnknown", fn: handlersValidators.UpdateUserPasswordValidationError, field: "Email", tag: "required", want: responseConst.Error.ValidationFailed},
	}

	runValidationCases(t, cases)
}

func TestWorkspaceValidators(t *testing.T) {
	t.Run("Creation", func(t *testing.T) {
		cases := []validationCase{
			{name: "WorkspaceCreationTitleRequired", fn: handlersValidators.WorkspaceCreationValidationError, field: "Title", tag: "required", want: responseConst.WorkspaceError.NameRequired},
			{name: "WorkspaceCreationTitleInvalid", fn: handlersValidators.WorkspaceCreationValidationError, field: "Title", tag: "min", want: responseConst.WorkspaceError.NameInvalid},
			{name: "WorkspaceCreationDescriptionInvalid", fn: handlersValidators.WorkspaceCreationValidationError, field: "Description", tag: "pattern", want: responseConst.WorkspaceError.DescriptionInvalid},
			{name: "WorkspaceCreationUnknown", fn: handlersValidators.WorkspaceCreationValidationError, field: "Foo", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})

	t.Run("Update", func(t *testing.T) {
		cases := []validationCase{
			{name: "WorkspaceUpdateIDRequired", fn: handlersValidators.WorkspaceUpdateValidationError, field: "ID", tag: "required", want: responseConst.WorkspaceError.IdRequired},
			{name: "WorkspaceUpdateIDInvalid", fn: handlersValidators.WorkspaceUpdateValidationError, field: "ID", tag: "uuid", want: responseConst.WorkspaceError.IdInvalid},
			{name: "WorkspaceUpdateTitleInvalid", fn: handlersValidators.WorkspaceUpdateValidationError, field: "Title", tag: "min", want: responseConst.WorkspaceError.NameInvalid},
			{name: "WorkspaceUpdateDescriptionInvalid", fn: handlersValidators.WorkspaceUpdateValidationError, field: "Description", tag: "pattern", want: responseConst.WorkspaceError.DescriptionInvalid},
			{name: "WorkspaceUpdateUnknown", fn: handlersValidators.WorkspaceUpdateValidationError, field: "Foo", tag: "required", want: responseConst.Error.ValidationFailed},
		}

		runValidationCases(t, cases)
	})
}

func TestValidateCreateViewMeta(t *testing.T) {
	t.Run("nil meta", func(t *testing.T) {
		req := dto.CreateViewRequest{Meta: nil}
		code, hasError := handlersValidators.ValidateCreateViewMeta(req)
		if !hasError || code != responseConst.TableError.MetaRequired {
			t.Errorf("expected MetaRequired error, got code=%v hasError=%v", code, hasError)
		}
	})

	t.Run("gallery view with valid attachment_field_id", func(t *testing.T) {
		fieldID := uuid.New().String()
		req := dto.CreateViewRequest{
			Type: "gallery",
			Meta: &map[string]interface{}{
				"attachment_field_id": fieldID,
			},
		}
		code, hasError := handlersValidators.ValidateCreateViewMeta(req)
		if hasError || code != "" {
			t.Errorf("expected no error, got code=%v hasError=%v", code, hasError)
		}
	})

	t.Run("gallery view missing attachment_field_id", func(t *testing.T) {
		req := dto.CreateViewRequest{
			Type: "gallery",
			Meta: &map[string]interface{}{},
		}
		code, hasError := handlersValidators.ValidateCreateViewMeta(req)
		if !hasError || code != responseConst.TableError.ViewAttachmentFieldIDRequired {
			t.Errorf("expected ViewAttachmentFieldIDRequired error, got code=%v hasError=%v", code, hasError)
		}
	})

	t.Run("gallery view with invalid attachment_field_id uuid", func(t *testing.T) {
		req := dto.CreateViewRequest{
			Type: "gallery",
			Meta: &map[string]interface{}{
				"attachment_field_id": "invalid-uuid",
			},
		}
		code, hasError := handlersValidators.ValidateCreateViewMeta(req)
		if !hasError || code != responseConst.TableError.ViewAttachmentFieldIDInvalid {
			t.Errorf("expected ViewAttachmentFieldIDInvalid error, got code=%v hasError=%v", code, hasError)
		}
	})

	t.Run("gallery view with empty attachment_field_id", func(t *testing.T) {
		req := dto.CreateViewRequest{
			Type: "gallery",
			Meta: &map[string]interface{}{
				"attachment_field_id": "",
			},
		}
		code, hasError := handlersValidators.ValidateCreateViewMeta(req)
		if !hasError || code != responseConst.TableError.ViewAttachmentFieldIDInvalid {
			t.Errorf("expected ViewAttachmentFieldIDInvalid error, got code=%v hasError=%v", code, hasError)
		}
	})

	t.Run("kanban view with valid view_target_field", func(t *testing.T) {
		fieldID := uuid.New().String()
		req := dto.CreateViewRequest{
			Type: "kanban",
			Meta: &map[string]interface{}{
				"view_target_field": fieldID,
			},
		}
		code, hasError := handlersValidators.ValidateCreateViewMeta(req)
		if hasError || code != "" {
			t.Errorf("expected no error, got code=%v hasError=%v", code, hasError)
		}
	})

	t.Run("calendar view with valid date_field_id", func(t *testing.T) {
		fieldID := uuid.New().String()
		req := dto.CreateViewRequest{
			Type: "calendar",
			Meta: &map[string]interface{}{
				"date_field_id": fieldID,
			},
		}
		code, hasError := handlersValidators.ValidateCreateViewMeta(req)
		if hasError || code != "" {
			t.Errorf("expected no error, got code=%v hasError=%v", code, hasError)
		}
	})

	t.Run("calender view with valid date_field_id", func(t *testing.T) {
		fieldID := uuid.New().String()
		req := dto.CreateViewRequest{
			Type: "calender",
			Meta: &map[string]interface{}{
				"date_field_id": fieldID,
			},
		}
		code, hasError := handlersValidators.ValidateCreateViewMeta(req)
		if hasError || code != "" {
			t.Errorf("expected no error, got code=%v hasError=%v", code, hasError)
		}
	})

	t.Run("ganttchart view with valid dates", func(t *testing.T) {
		startID := uuid.New().String()
		endID := uuid.New().String()
		req := dto.CreateViewRequest{
			Type: "ganttchart",
			Meta: &map[string]interface{}{
				"start_date_field_id": startID,
				"end_date_field_id":   endID,
			},
		}
		code, hasError := handlersValidators.ValidateCreateViewMeta(req)
		if hasError || code != "" {
			t.Errorf("expected no error, got code=%v hasError=%v", code, hasError)
		}
	})

	t.Run("ganttchart view missing end_date_field_id", func(t *testing.T) {
		startID := uuid.New().String()
		req := dto.CreateViewRequest{
			Type: "ganttchart",
			Meta: &map[string]interface{}{
				"start_date_field_id": startID,
			},
		}
		code, hasError := handlersValidators.ValidateCreateViewMeta(req)
		if !hasError || code != responseConst.TableError.ViewEndDateFieldIDRequired {
			t.Errorf("expected ViewEndDateFieldIDRequired error, got code=%v hasError=%v", code, hasError)
		}
	})

	t.Run("ganttchart view with invalid start_date", func(t *testing.T) {
		endID := uuid.New().String()
		req := dto.CreateViewRequest{
			Type: "ganttchart",
			Meta: &map[string]interface{}{
				"start_date_field_id": "invalid",
				"end_date_field_id":   endID,
			},
		}
		code, hasError := handlersValidators.ValidateCreateViewMeta(req)
		if !hasError || code != responseConst.TableError.ViewStartDateFieldIDInvalid {
			t.Errorf("expected ViewStartDateFieldIDInvalid error, got code=%v hasError=%v", code, hasError)
		}
	})

	t.Run("grid view no meta validation needed", func(t *testing.T) {
		req := dto.CreateViewRequest{
			Type: "grid",
			Meta: &map[string]interface{}{},
		}
		code, hasError := handlersValidators.ValidateCreateViewMeta(req)
		if hasError || code != "" {
			t.Errorf("expected no error, got code=%v hasError=%v", code, hasError)
		}
	})

	t.Run("non-string attachment_field_id", func(t *testing.T) {
		req := dto.CreateViewRequest{
			Type: "gallery",
			Meta: &map[string]interface{}{
				"attachment_field_id": 123,
			},
		}
		code, hasError := handlersValidators.ValidateCreateViewMeta(req)
		if !hasError || code != responseConst.TableError.ViewAttachmentFieldIDInvalid {
			t.Errorf("expected ViewAttachmentFieldIDInvalid error for non-string, got code=%v hasError=%v", code, hasError)
		}
	})
}
