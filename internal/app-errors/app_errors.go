package app_errors

import "errors"

// var (
// 	KeycloakAdminLoginFailed   = errors.New("keycloak admin login failed")
// 	KeycloakRegisterFailed     = errors.New("keycloak register failed")
// 	KeycloakLoginFailed        = errors.New("keycloak login failed")
// 	KeycloakRefreshTokenFailed = errors.New("keycloak refresh token failed")
// )

var (
	FileNotFound           = errors.New("file not found")
	FileAlreadyExists      = errors.New("file already exists")
	FileReadFailed         = errors.New("file read failed")
	FileWriteFailed        = errors.New("file write failed")
	FileDeleteFailed       = errors.New("file delete failed")
	FilePermissionDenied   = errors.New("file permission denied")
	FileInvalidPath        = errors.New("file invalid path")
	FolderNotFound         = errors.New("folder not found")
	FolderAlreadyExists    = errors.New("folder already exists")
	FolderCreateFailed     = errors.New("folder create failed")
	FolderDeleteFailed     = errors.New("folder delete failed")
	FolderPermissionDenied = errors.New("folder permission denied")
	FolderInvalidPath      = errors.New("folder invalid path")

	// New errors for refactoring
	ErrInvalidDateOfBirth         = errors.New("invalid date of birth format")
	ErrRoleCreation               = errors.New("failed to create role")
	ErrSubscriptionPlanNotFound   = errors.New("subscription plan not found")
	ErrRoleNotFound               = errors.New("role not found")
	ErrUserDisableFailed          = errors.New("failed to disable user")
	ErrInvalidWorkspaceMemberData = errors.New("invalid workspace member data")
	ErrUserContextNotFound        = errors.New("user context not found")
)

// var (
// 	StorageUploadFailed     = errors.New("storage upload failed")
// 	StorageDownloadFailed   = errors.New("storage download failed")
// 	StorageDeleteFailed     = errors.New("storage delete failed")
// 	FileNotReceived         = errors.New("no file is received")
// 	StorageExistsFailed     = errors.New("storage exists check failed")
// 	StorageInvalidPath      = errors.New("storage invalid path")
// 	StoragePermissionDenied = errors.New("storage permission denied")
// 	StorageFileOpenFailed   = errors.New("storage file open failed")
// )

var (
	DatabaseError            = errors.New("database error")
	ErrInternal              = errors.New("internal error")
	ErrMapToStruct           = errors.New("failed to map to struct")
	ErrStructToStruct        = errors.New("failed to struct to struct")
	ErrHashed                = errors.New("failed to hash value")
	InvalidCredentials       = errors.New("invalid credentials")
	InvalidPayload           = errors.New("invalid payload")
	InvalidDriver            = errors.New("invalid driver")
	InvalidOldPassword       = errors.New("invalid old password")
	ErrRecordNotFound        = errors.New("record not found")
	ErrJSONMarshal           = errors.New("failed to marshal JSON")
	ErrHTTPRequestCreation   = errors.New("failed to create HTTP request")
	ErrHTTPDoRequest         = errors.New("failed to execute HTTP request")
	ErrServiceNotInitialized = errors.New("service not initialized")
)

// user management
var (
	UserAlreadyExists    = errors.New("user already exists")
	UserNotFound         = errors.New("user not found")
	EmailAlreadyVerified = errors.New("email already verified")
	NewPasswordInvalid   = errors.New("new password is invalid")
)

// role management
var (
	RoleAlreadyExists = errors.New("role already exists")
	RoleNotFound      = errors.New("role not found")
)

// subscription plan management
var (
	SubscriptionPlanAlreadyExists = errors.New("subscription plan already exists")
	SubscriptionPlanNotFound      = errors.New("subscription plan not found")
)

// tenant management
var (
	TenantAlreadyExists = errors.New("tenant already exists")
	TenantNotFound      = errors.New("tenant not found")
)

// tenant subscription management
var (
	TenantSubscriptionAlreadyExists = errors.New("tenant subscription already exists")
	TenantSubscriptionNotFound      = errors.New("tenant subscription not found")
)

// auth management
var (
	InvalidOTP                       = errors.New("invalid OTP")
	AuthProviderLoginFailed          = errors.New("authentication provider login failed")
	AuthProviderRefreshTokenFailed   = errors.New("authentication provider refresh token failed")
	AuthProviderTokenInvalid         = errors.New("authentication provider token invalid")
	AuthProviderPingFailed           = errors.New("authentication provider ping failed")
	AuthProviderAuthHeaderRequired   = errors.New("authentication provider authorization header required")
	AuthProviderTokenDecodeFailed    = errors.New("authentication provider token decode failed")
	AuthProviderClaimsNotFound       = errors.New("authentication provider claims not found")
	AuthProviderUserIDNotFound       = errors.New("authentication provider user id not found")
	TokenUserIdNotFound              = errors.New("token user id not found")
	TokenAccessTokenSignFailed       = errors.New("failed to sign access token")
	TokenRefreshTokenSignFailed      = errors.New("failed to sign refresh token")
	TokenRefreshTokenInvalid         = errors.New("invalid refresh token")
	TokenRefreshTokenClaimsInvalid   = errors.New("invalid claims in refresh token")
	TokenInvalid                     = errors.New("invalid token")
	TokenClaimsInvalid               = errors.New("invalid claims in token")
	TokenAuthorizationHeaderRequired = errors.New("authorization header required for token")
	TokenClaimsNotFound              = errors.New("token claims not found")
	AuthProviderAdminLoginFailed     = errors.New("failed to login as admin to authentication provider")
	AuthProviderUserCreateFailed     = errors.New("failed to create user in authentication provider")
	AuthProviderSetPasswordFailed    = errors.New("failed to set user password in authentication provider")
	TokenExpired                     = errors.New("token has expired")
	AuthProviderTokenExpired         = errors.New("authentication provider token has expired")
	TokenUnauthorized                = errors.New("unauthorized token")
)

// workspace management
var (
	ErrWorkspaceInsertion     = errors.New("failed to insert workspace")
	WorkspaceMemberNotFound   = errors.New("workspace member not found")
	ErrUserAlreadyInWorkspace = errors.New("user already in workspace")
)

// base management
var (
	BaseNotFound = errors.New("base not found")
)

// asset management
var (
	VirusDetected              = errors.New("virus detected in uploaded file")
	StorageFileOpenFailed      = errors.New("failed to open file for storage")
	StorageUploadFailed        = errors.New("failed to upload file to storage")
	AssetNotFound              = errors.New("asset not found")
	FileTooLargeError          = errors.New("File is too large. Limit is 5MB.")
	MultipleFilesTooLargeError = errors.New("One or more files exceed the 5MB size limit.")
	TooManyFilesError          = errors.New("Too many files uploaded. Only 5 files are allowed.")
	MultipartFormNotFound      = errors.New("multipart form not found")
)

// table management
var (
	UpdateNotAllowed               = errors.New("update not allowed")
	DeleteNotAllowed               = errors.New("delete not allowed")
	ViewNotFound                   = errors.New("view not found")
	ViewUploadFailed               = errors.New("failed to update view")
	ColumnNotFound                 = errors.New("column not found")
	ColumnUpdateFailed             = errors.New("failed to update column")
	TableNotFound                  = errors.New("table not found")
	InvalidUIDT                    = errors.New("invalid UI data type")
	InvalidColumnMetaForLinkType   = errors.New("invalid column meta for link type")
	RowNotFound                    = errors.New("row not found")
	InvalidColumnMetaForLookupType = errors.New("invalid column meta for lookup type")
)

// APIError represents an error response from an external API
type APIError struct {
	Code    string
	Message string
	Details interface{}
}

// Error implements the error interface for APIError
func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}
