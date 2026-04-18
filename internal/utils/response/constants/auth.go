// Copyright (c) 2026 Aptlogica Technologies Private Limited
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package constants

import "net/http"

// Expanding AuthError to include additional error keys from errorMapping.go (56-77).
var AuthError = struct {
	FirstNameRequired        ResponseCode
	LastNameRequired         ResponseCode
	FirstNameInvalid         ResponseCode
	LastNameInvalid          ResponseCode
	EmailRequired            ResponseCode
	EmailInvalidFormat       ResponseCode
	EmailInvalid             ResponseCode
	PasswordRequired         ResponseCode
	PasswordTooShort         ResponseCode
	PasswordInvalid          ResponseCode
	OrganizationNameRequired ResponseCode
	OrganizationNameInvalid  ResponseCode
	InvalidOTP               ResponseCode
	UserIDRequired           ResponseCode
	UserIDInvalidFormat      ResponseCode
	UserIDInvalid            ResponseCode
	OTPRequired              ResponseCode
	OTPInvalid               ResponseCode
	SubscriptionIDRequired   ResponseCode
	SubscriptionIDInvalid    ResponseCode
	RoleIDRequired           ResponseCode
	RoleIDInvalid            ResponseCode
	SchemaRequired           ResponseCode
	InvalidSchema            ResponseCode
	RefreshTokenRequired     ResponseCode
	RefreshTokenInvalid      ResponseCode
	TokenRequired            ResponseCode
	TokenInvalidFormat       ResponseCode
	TokenInvalid             ResponseCode
	NewPasswordRequired      ResponseCode
	NewPasswordInvalid       ResponseCode
	DateOfBirthRequired      ResponseCode
	DateOfBirthInvalid       ResponseCode
	CountryRequired          ResponseCode
	CountryInvalid           ResponseCode
	TimezoneRequired         ResponseCode
	TimezoneInvalid          ResponseCode
	WorkspaceRequired        ResponseCode
	BaseRequired             ResponseCode
	AccountLocked            ResponseCode
	TooManyRequests          ResponseCode
	Unauthorized             ResponseCode
	Forbidden                ResponseCode
	InternalServerError      ResponseCode
	NotFound                 ResponseCode
	// ---- Added below ----
	AuthProviderLoginFailed          ResponseCode
	AuthProviderRefreshTokenFailed   ResponseCode
	AuthProviderTokenInvalid         ResponseCode
	AuthProviderPingFailed           ResponseCode
	AuthProviderAuthHeaderRequired   ResponseCode
	AuthProviderTokenDecodeFailed    ResponseCode
	AuthProviderClaimsNotFound       ResponseCode
	AuthProviderUserIDNotFound       ResponseCode
	TokenUserIdNotFound              ResponseCode
	TokenAccessTokenSignFailed       ResponseCode
	TokenRefreshTokenSignFailed      ResponseCode
	TokenRefreshTokenInvalid         ResponseCode
	TokenRefreshTokenClaimsInvalid   ResponseCode
	TokenClaimsInvalid               ResponseCode
	TokenAuthorizationHeaderRequired ResponseCode
	TokenClaimsNotFound              ResponseCode
	AuthProviderAdminLoginFailed     ResponseCode
	AuthProviderUserCreateFailed     ResponseCode
	AuthProviderSetPasswordFailed    ResponseCode
	TokenExpired                     ResponseCode
	AuthProviderTokenExpired         ResponseCode
	TokenUnauthorized                ResponseCode
}{
	FirstNameRequired:        "AUTH_VAL_1001",
	LastNameRequired:         "AUTH_VAL_1002",
	FirstNameInvalid:         "AUTH_VAL_1003",
	LastNameInvalid:          "AUTH_VAL_1004",
	EmailRequired:            "AUTH_VAL_1005",
	EmailInvalidFormat:       "AUTH_VAL_1006",
	EmailInvalid:             "AUTH_VAL_1007",
	PasswordRequired:         "AUTH_VAL_1008",
	PasswordTooShort:         "AUTH_VAL_1009",
	PasswordInvalid:          "AUTH_VAL_1010",
	OrganizationNameRequired: "AUTH_VAL_1011",
	OrganizationNameInvalid:  "AUTH_VAL_1012",
	InvalidOTP:               "AUTH_VAL_1013",
	UserIDRequired:           "AUTH_VAL_1014",
	UserIDInvalidFormat:      "AUTH_VAL_1015",
	UserIDInvalid:            "AUTH_VAL_1016",
	OTPRequired:              "AUTH_VAL_1017",
	OTPInvalid:               "AUTH_VAL_1018",
	SubscriptionIDRequired:   "AUTH_VAL_1019",
	SubscriptionIDInvalid:    "AUTH_VAL_1020",
	RoleIDRequired:           "AUTH_VAL_1021",
	RoleIDInvalid:            "AUTH_VAL_1022",
	SchemaRequired:           "AUTH_VAL_1023",
	InvalidSchema:            "AUTH_VAL_1044",
	RefreshTokenRequired:     "AUTH_VAL_1047",
	RefreshTokenInvalid:      "AUTH_VAL_1048",
	TokenRequired:            "AUTH_VAL_1049",
	TokenInvalidFormat:       "AUTH_VAL_1050",
	TokenInvalid:             "AUTH_VAL_1051",
	NewPasswordRequired:      "AUTH_VAL_1052",
	NewPasswordInvalid:       "AUTH_VAL_1053",
	DateOfBirthRequired:      "AUTH_VAL_1054",
	DateOfBirthInvalid:       "AUTH_VAL_1055",
	CountryRequired:          "AUTH_VAL_1056",
	CountryInvalid:           "AUTH_VAL_1057",
	TimezoneRequired:         "AUTH_VAL_1058",
	TimezoneInvalid:          "AUTH_VAL_1059",
	WorkspaceRequired:        "AUTH_VAL_1060",
	BaseRequired:             "AUTH_VAL_1061",
	AccountLocked:            "AUTH_VAL_2001",
	TooManyRequests:          "AUTH_VAL_2002",
	Unauthorized:             "AUTH_VAL_2003",
	Forbidden:                "AUTH_VAL_2004",
	InternalServerError:      "AUTH_VAL_2005",
	NotFound:                 "AUTH_VAL_2006",
	TokenUnauthorized:        "AUTH_VAL_2007",

	// ---- Added below ----
	AuthProviderLoginFailed:          "AUTH_ERR_3001",
	AuthProviderRefreshTokenFailed:   "AUTH_ERR_3002",
	AuthProviderTokenInvalid:         "AUTH_ERR_3003",
	AuthProviderPingFailed:           "AUTH_ERR_3004",
	AuthProviderAuthHeaderRequired:   "AUTH_ERR_3005",
	AuthProviderTokenDecodeFailed:    "AUTH_ERR_3006",
	AuthProviderClaimsNotFound:       "AUTH_ERR_3007",
	AuthProviderUserIDNotFound:       "AUTH_ERR_3008",
	TokenUserIdNotFound:              "AUTH_ERR_3009",
	TokenAccessTokenSignFailed:       "AUTH_ERR_3010",
	TokenRefreshTokenSignFailed:      "AUTH_ERR_3011",
	TokenRefreshTokenInvalid:         "AUTH_ERR_3012",
	TokenRefreshTokenClaimsInvalid:   "AUTH_ERR_3013",
	TokenClaimsInvalid:               "AUTH_ERR_3014",
	TokenAuthorizationHeaderRequired: "AUTH_ERR_3015",
	TokenClaimsNotFound:              "AUTH_ERR_3016",
	AuthProviderAdminLoginFailed:     "AUTH_ERR_3017",
	AuthProviderUserCreateFailed:     "AUTH_ERR_3018",
	AuthProviderSetPasswordFailed:    "AUTH_ERR_3019",
	TokenExpired:                     "AUTH_ERR_3020",
	AuthProviderTokenExpired:         "AUTH_ERR_3021",
}

var AuthErrorCodes = map[ResponseCode]MetaResponse{
	AuthError.FirstNameRequired:        {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "First name is required"},
	AuthError.LastNameRequired:         {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "Last name is required"},
	AuthError.FirstNameInvalid:         {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "First name is invalid"},
	AuthError.LastNameInvalid:          {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "Last name is invalid"},
	AuthError.EmailRequired:            {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "Email is required"},
	AuthError.EmailInvalidFormat:       {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "Email format is invalid"},
	AuthError.EmailInvalid:             {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "Email is invalid"},
	AuthError.PasswordRequired:         {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "Password is required"},
	AuthError.PasswordTooShort:         {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "Password is too short"},
	AuthError.PasswordInvalid:          {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "Password is invalid"},
	AuthError.OrganizationNameRequired: {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "Organization name is required"},
	AuthError.OrganizationNameInvalid:  {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "Organization name is invalid"},
	AuthError.InvalidOTP:               {HTTPStatus: http.StatusBadRequest, Message: "Invalid OTP", Description: "Invalid OTP"},
	AuthError.UserIDRequired:           {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "User ID is required"},
	AuthError.UserIDInvalidFormat:      {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "User ID format is invalid"},
	AuthError.UserIDInvalid:            {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "User ID is invalid"},
	AuthError.OTPRequired:              {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "OTP is required"},
	AuthError.OTPInvalid:               {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "OTP is invalid"},
	AuthError.SubscriptionIDRequired:   {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "Subscription ID is required"},
	AuthError.SubscriptionIDInvalid:    {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "Subscription ID is invalid"},
	AuthError.RoleIDRequired:           {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "Role ID is required"},
	AuthError.RoleIDInvalid:            {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "Role ID is invalid"},
	AuthError.SchemaRequired:           {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "Header 'schema' is required"},
	AuthError.InvalidSchema:            {HTTPStatus: http.StatusBadRequest, Message: "Invalid schema", Description: "Provided schema is invalid"},
	AuthError.RefreshTokenRequired:     {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "Refresh token is required"},
	AuthError.RefreshTokenInvalid:      {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "Refresh token is invalid"},
	AuthError.TokenRequired:            {HTTPStatus: http.StatusUnauthorized, Message: "Token required", Description: "Authentication token is required"},
	AuthError.TokenInvalidFormat:       {HTTPStatus: http.StatusUnauthorized, Message: "Token invalid format", Description: "Submitted token format is invalid"},
	AuthError.TokenInvalid:             {HTTPStatus: http.StatusUnauthorized, Message: "Token invalid", Description: "The provided token is invalid"},
	AuthError.NewPasswordRequired:      {HTTPStatus: http.StatusBadRequest, Message: "New password required", Description: "New password is required"},
	AuthError.NewPasswordInvalid:       {HTTPStatus: http.StatusBadRequest, Message: "New password invalid", Description: "Provided new password is invalid"},
	AuthError.DateOfBirthRequired:      {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "Date of birth is required"},
	AuthError.DateOfBirthInvalid:       {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "Date of birth is invalid"},
	AuthError.CountryRequired:          {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "Country is required"},
	AuthError.CountryInvalid:           {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "Country is invalid"},
	AuthError.TimezoneRequired:         {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "Timezone is required"},
	AuthError.TimezoneInvalid:          {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "Timezone is invalid"},
	AuthError.WorkspaceRequired:        {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "Header 'workspace' is required"},
	AuthError.BaseRequired:             {HTTPStatus: http.StatusBadRequest, Message: MsgInvalidRequestPayload, Description: "Header 'base' is required"},
	AuthError.AccountLocked:            {HTTPStatus: http.StatusForbidden, Message: "Account locked", Description: "Your account has been locked"},
	AuthError.TooManyRequests:          {HTTPStatus: http.StatusTooManyRequests, Message: "Too many requests", Description: "Too many authentication attempts, try again later"},
	AuthError.Unauthorized:             {HTTPStatus: http.StatusUnauthorized, Message: "Unauthorized", Description: "You are not authorized to perform this action"},
	AuthError.Forbidden:                {HTTPStatus: http.StatusForbidden, Message: "Forbidden", Description: "You do not have permission to access this resource"},
	AuthError.InternalServerError:      {HTTPStatus: http.StatusInternalServerError, Message: MsgInternalServerError, Description: "An error occurred on the server"},
	AuthError.NotFound:                 {HTTPStatus: http.StatusNotFound, Message: "Not found", Description: "The requested resource was not found"},
	AuthError.TokenUnauthorized:        {HTTPStatus: http.StatusUnauthorized, Message: "Unauthorized", Description: "Token is unauthorized"},

	// ---- Added below (matching errorMapping.go 56-77) ----
	AuthError.AuthProviderLoginFailed:          {HTTPStatus: http.StatusUnauthorized, Message: "Authentication provider login failed", Description: "Unable to login using authentication provider"},
	AuthError.AuthProviderRefreshTokenFailed:   {HTTPStatus: http.StatusUnauthorized, Message: "Authentication provider refresh token failed", Description: "Refresh token failed for authentication provider"},
	AuthError.AuthProviderTokenInvalid:         {HTTPStatus: http.StatusUnauthorized, Message: "Authentication provider token invalid", Description: "Authentication provider token is invalid"},
	AuthError.AuthProviderPingFailed:           {HTTPStatus: http.StatusServiceUnavailable, Message: "Authentication provider ping failed", Description: "Unable to ping authentication provider"},
	AuthError.AuthProviderAuthHeaderRequired:   {HTTPStatus: http.StatusUnauthorized, Message: "Authentication provider auth header required", Description: "Authentication header required by provider"},
	AuthError.AuthProviderTokenDecodeFailed:    {HTTPStatus: http.StatusUnauthorized, Message: "Authentication provider token decode failed", Description: "Failed to decode authentication provider token"},
	AuthError.AuthProviderClaimsNotFound:       {HTTPStatus: http.StatusUnauthorized, Message: "Authentication provider claims not found", Description: "Claims not found in authentication provider token"},
	AuthError.AuthProviderUserIDNotFound:       {HTTPStatus: http.StatusUnauthorized, Message: "Authentication provider user ID not found", Description: "User ID not found in authentication provider claims"},
	AuthError.TokenUserIdNotFound:              {HTTPStatus: http.StatusUnauthorized, Message: "User ID not found in token", Description: "User ID not available in token"},
	AuthError.TokenAccessTokenSignFailed:       {HTTPStatus: http.StatusInternalServerError, Message: "Access token signing failed", Description: "Signing access token failed"},
	AuthError.TokenRefreshTokenSignFailed:      {HTTPStatus: http.StatusInternalServerError, Message: "Refresh token signing failed", Description: "Signing refresh token failed"},
	AuthError.TokenRefreshTokenInvalid:         {HTTPStatus: http.StatusUnauthorized, Message: "Refresh token invalid", Description: "Supplied refresh token is invalid"},
	AuthError.TokenRefreshTokenClaimsInvalid:   {HTTPStatus: http.StatusUnauthorized, Message: "Refresh token claims invalid", Description: "Claims in refresh token are invalid"},
	AuthError.TokenClaimsInvalid:               {HTTPStatus: http.StatusUnauthorized, Message: "Token claims invalid", Description: "Supplied token contains invalid claims"},
	AuthError.TokenAuthorizationHeaderRequired: {HTTPStatus: http.StatusUnauthorized, Message: "Token authorization header required", Description: "Authorization header is required"},
	AuthError.TokenClaimsNotFound:              {HTTPStatus: http.StatusUnauthorized, Message: "Token claims not found", Description: "Token claims were not found"},
	AuthError.AuthProviderAdminLoginFailed:     {HTTPStatus: http.StatusUnauthorized, Message: "Authentication provider admin login failed", Description: "Admin login in authentication provider failed"},
	AuthError.AuthProviderUserCreateFailed:     {HTTPStatus: http.StatusInternalServerError, Message: "Authentication provider user create failed", Description: "User creation in authentication provider failed"},
	AuthError.AuthProviderSetPasswordFailed:    {HTTPStatus: http.StatusInternalServerError, Message: "Authentication provider set password failed", Description: "Setting password in authentication provider failed"},
	AuthError.TokenExpired:                     {HTTPStatus: http.StatusUnauthorized, Message: "Token expired", Description: "The authentication token has expired"},
	AuthError.AuthProviderTokenExpired:         {HTTPStatus: http.StatusUnauthorized, Message: "Authentication provider token expired", Description: "Authentication provider's token has expired"},
}

var AuthSuccess = struct {
	UserRegister   ResponseCode
	UserLogin      ResponseCode
	UserLogout     ResponseCode
	EmailVerified  ResponseCode
	ResendOTP      ResponseCode
	RefreshToken   ResponseCode
	ForgotPassword ResponseCode
	ResetPassword  ResponseCode
	ValidateToken  ResponseCode
	VerifyToken    ResponseCode
}{
	UserRegister:   "AUTH_SUCCESS_1001",
	UserLogin:      "AUTH_SUCCESS_1002",
	EmailVerified:  "AUTH_SUCCESS_1003",
	ResendOTP:      "AUTH_SUCCESS_1004",
	RefreshToken:   "AUTH_SUCCESS_1005",
	ForgotPassword: "AUTH_SUCCESS_1006",
	ResetPassword:  "AUTH_SUCCESS_1007",
	UserLogout:     "AUTH_SUCCESS_1008",
	ValidateToken:  "AUTH_SUCCESS_1009",
	VerifyToken:    "AUTH_SUCCESS_1010",
}

var AuthSuccessCodes = map[ResponseCode]MetaResponse{
	AuthSuccess.UserRegister:   {HTTPStatus: http.StatusCreated, Message: "User registered successfully", Description: "The user has been registered successfully"},
	AuthSuccess.UserLogin:      {HTTPStatus: http.StatusOK, Message: "Login successful", Description: "The user has logged in successfully"},
	AuthSuccess.EmailVerified:  {HTTPStatus: http.StatusOK, Message: "Email verified successfully", Description: "The user's email has been verified successfully"},
	AuthSuccess.ResendOTP:      {HTTPStatus: http.StatusOK, Message: "OTP resent successfully", Description: "A new OTP has been sent successfully"},
	AuthSuccess.RefreshToken:   {HTTPStatus: http.StatusOK, Message: "Token refreshed successfully", Description: "The access token has been refreshed successfully"},
	AuthSuccess.ForgotPassword: {HTTPStatus: http.StatusOK, Message: "Forgot password request successful", Description: "Password recovery instructions have been sent successfully"},
	AuthSuccess.ResetPassword:  {HTTPStatus: http.StatusOK, Message: "Password reset successful", Description: "The user's password has been reset successfully"},
	AuthSuccess.UserLogout:     {HTTPStatus: http.StatusOK, Message: "Logout successful", Description: "The user has been logged out successfully"},
	AuthSuccess.ValidateToken:  {HTTPStatus: http.StatusOK, Message: "Token valid", Description: "The provided token is valid"},
	AuthSuccess.VerifyToken:    {HTTPStatus: http.StatusOK, Message: "Token verified", Description: "The provided token has been verified"},
}
