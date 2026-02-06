package constants

import "net/http"

// Message constants for commonly used error messages
const (
	MsgInternalServerError   = "Internal server error"
	MsgUnauthorizedAccess    = "Unauthorized access"
	MsgBadRequest            = "Bad request"
	MsgInvalidRequestPayload = "Invalid request payload"
)

var Error = struct {
	InvalidID                ResponseCode
	UnauthorizedAccess       ResponseCode
	Forbidden                ResponseCode
	SessionExpired           ResponseCode
	InvalidPayload           ResponseCode
	ValidationFailed         ResponseCode
	DatabaseError            ResponseCode
	RecordNotFound           ResponseCode
	InternalError            ResponseCode
	ServiceUnavailable       ResponseCode
	GatewayTimeout           ResponseCode
	TooManyRequests          ResponseCode
	UserAlreadyExists        ResponseCode
	RecordAlreadyExists      ResponseCode
	RecordNotInserted        ResponseCode
	BadRequest               ResponseCode
	Conflict                 ResponseCode
	NotImplemented           ResponseCode
	Timeout                  ResponseCode
	DependencyFailed         ResponseCode
	MapToStructError         ResponseCode
	StructToStructError      ResponseCode
	HashingError             ResponseCode
	InvalidCredentials       ResponseCode
	InvalidDriver            ResponseCode
	ErrNotFound              ResponseCode
	JSONMarshalError         ResponseCode
	HTTPRequestCreationError ResponseCode
	HTTPDoRequestError       ResponseCode
	UserNotActive            ResponseCode

	// file handling
	FileNotFound           ResponseCode
	FileAlreadyExists      ResponseCode
	FileReadFailed         ResponseCode
	FileWriteFailed        ResponseCode
	FileDeleteFailed       ResponseCode
	FilePermissionDenied   ResponseCode
	FileInvalidPath        ResponseCode
	FolderNotFound         ResponseCode
	FolderAlreadyExists    ResponseCode
	FolderCreateFailed     ResponseCode
	FolderDeleteFailed     ResponseCode
	FolderPermissionDenied ResponseCode
	FolderInvalidPath      ResponseCode

	// New refactored codes
	InvalidDateOfBirth         ResponseCode
	RoleCreationError          ResponseCode
	SubscriptionPlanNotFound   ResponseCode
	RoleNotFound               ResponseCode
	UserDisableFailed          ResponseCode
	InvalidWorkspaceMemberData ResponseCode
	UserContextNotFound        ResponseCode
}{
	InvalidID:                "ERR_0001",
	UnauthorizedAccess:       "ERR_0002",
	Forbidden:                "ERR_0003",
	SessionExpired:           "ERR_0004",
	InvalidPayload:           "ERR_0005",
	ValidationFailed:         "ERR_0006",
	DatabaseError:            "ERR_0007",
	RecordNotFound:           "ERR_0008",
	InternalError:            "ERR_0009",
	ServiceUnavailable:       "ERR_0010",
	GatewayTimeout:           "ERR_0011",
	TooManyRequests:          "ERR_0012",
	UserAlreadyExists:        "ERR_0013",
	RecordAlreadyExists:      "ERR_0014",
	RecordNotInserted:        "ERR_0015",
	BadRequest:               "ERR_0016",
	Conflict:                 "ERR_0017",
	NotImplemented:           "ERR_0018",
	Timeout:                  "ERR_0019",
	DependencyFailed:         "ERR_0020",
	MapToStructError:         "ERR_0021",
	StructToStructError:      "ERR_0022",
	HashingError:             "ERR_0023",
	InvalidCredentials:       "ERR_0024",
	InvalidDriver:            "ERR_0025",
	ErrNotFound:              "ERR_0026",
	JSONMarshalError:         "ERR_0027",
	HTTPRequestCreationError: "ERR_0028",
	HTTPDoRequestError:       "ERR_0029",
	UserNotActive:            "ERR_0030",

	FileNotFound:           "ERR_1001",
	FileAlreadyExists:      "ERR_1002",
	FileReadFailed:         "ERR_1003",
	FileWriteFailed:        "ERR_1004",
	FileDeleteFailed:       "ERR_1005",
	FilePermissionDenied:   "ERR_1006",
	FileInvalidPath:        "ERR_1007",
	FolderNotFound:         "ERR_1008",
	FolderAlreadyExists:    "ERR_1009",
	FolderCreateFailed:     "ERR_1010",
	FolderDeleteFailed:     "ERR_1011",
	FolderPermissionDenied: "ERR_1012",
	FolderInvalidPath:      "ERR_1013",

	InvalidDateOfBirth:         "ERR_1014",
	RoleCreationError:          "ERR_1015",
	SubscriptionPlanNotFound:   "ERR_1016",
	RoleNotFound:               "ERR_1017",
	UserDisableFailed:          "ERR_1018",
	InvalidWorkspaceMemberData: "ERR_1019",
	UserContextNotFound:        "ERR_1020",
}

var CoreErrorCodes = map[ResponseCode]MetaResponse{
	Error.UnauthorizedAccess:       CreateMetaResponse(http.StatusUnauthorized, MsgUnauthorizedAccess, MsgUnauthorizedAccess),
	Error.Forbidden:                CreateMetaResponse(http.StatusForbidden, "Forbidden", "Forbidden"),
	Error.SessionExpired:           {HTTPStatus: http.StatusUnauthorized, Message: MsgUnauthorizedAccess, Description: "Session expired"},
	Error.InvalidPayload:           {HTTPStatus: http.StatusBadRequest, Message: MsgBadRequest, Description: MsgInvalidRequestPayload},
	Error.ValidationFailed:         CreateMetaResponse(http.StatusUnprocessableEntity, "Validation failed", "Validation failed"),
	Error.DatabaseError:            CreateMetaResponse(http.StatusInternalServerError, "Database error", "Database error"),
	Error.RecordNotFound:           CreateMetaResponse(http.StatusNotFound, "Record not found", "Record not found"),
	Error.InternalError:            CreateMetaResponse(http.StatusInternalServerError, MsgInternalServerError, MsgInternalServerError),
	Error.ServiceUnavailable:       CreateMetaResponse(http.StatusServiceUnavailable, "Service unavailable", "Service unavailable"),
	Error.GatewayTimeout:           CreateMetaResponse(http.StatusGatewayTimeout, "Gateway timeout", "Gateway timeout"),
	Error.TooManyRequests:          CreateMetaResponse(http.StatusTooManyRequests, "Too many requests", "Too many requests"),
	Error.Conflict:                 CreateMetaResponse(http.StatusConflict, "Conflict", "Conflict"),
	Error.BadRequest:               CreateMetaResponse(http.StatusBadRequest, MsgBadRequest, MsgBadRequest),
	Error.InvalidID:                {HTTPStatus: http.StatusBadRequest, Message: "Invalid ID", Description: "The provided ID is invalid"},
	Error.UserAlreadyExists:        CreateMetaResponse(http.StatusConflict, "User already exists", "User already exists"),
	Error.RecordAlreadyExists:      CreateMetaResponse(http.StatusConflict, "Record already exists", "Record already exists"),
	Error.RecordNotInserted:        CreateMetaResponse(http.StatusInternalServerError, "Record not inserted", "Record not inserted"),
	Error.NotImplemented:           CreateMetaResponse(http.StatusNotImplemented, "Not implemented", "Not implemented"),
	Error.Timeout:                  CreateMetaResponse(http.StatusRequestTimeout, "Timeout", "Timeout"),
	Error.DependencyFailed:         CreateMetaResponse(http.StatusFailedDependency, "Dependency failed", "Dependency failed"),
	Error.MapToStructError:         CreateMetaResponse(http.StatusInternalServerError, "Map to struct error", "Map to struct error"),
	Error.StructToStructError:      CreateMetaResponse(http.StatusInternalServerError, "Struct to struct error", "Struct to struct error"),
	Error.HashingError:             CreateMetaResponse(http.StatusInternalServerError, "Hashing error", "Hashing error"),
	Error.InvalidCredentials:       CreateMetaResponse(http.StatusUnauthorized, "Invalid credentials", "Invalid credentials"),
	Error.InvalidDriver:            CreateMetaResponse(http.StatusBadRequest, "Invalid driver", "Invalid driver"),
	Error.ErrNotFound:              CreateMetaResponse(http.StatusNotFound, "Not found", "Not found"),
	Error.JSONMarshalError:         CreateMetaResponse(http.StatusInternalServerError, "JSON marshal error", "JSON marshal error"),
	Error.HTTPRequestCreationError: CreateMetaResponse(http.StatusInternalServerError, "HTTP request creation error", "HTTP request creation error"),
	Error.HTTPDoRequestError:       CreateMetaResponse(http.StatusInternalServerError, "HTTP do request error", "HTTP do request error"),
	Error.UserNotActive:            CreateMetaResponse(http.StatusForbidden, "User not active", "User not active"),

	Error.FileNotFound:           CreateMetaResponse(http.StatusNotFound, "File not found", "File not found"),
	Error.FileAlreadyExists:      CreateMetaResponse(http.StatusConflict, "File already exists", "File already exists"),
	Error.FileReadFailed:         CreateMetaResponse(http.StatusInternalServerError, "File read failed", "File read failed"),
	Error.FileWriteFailed:        CreateMetaResponse(http.StatusInternalServerError, "File write failed", "File write failed"),
	Error.FileDeleteFailed:       CreateMetaResponse(http.StatusInternalServerError, "File delete failed", "File delete failed"),
	Error.FilePermissionDenied:   CreateMetaResponse(http.StatusForbidden, "File permission denied", "File permission denied"),
	Error.FileInvalidPath:        CreateMetaResponse(http.StatusBadRequest, "File invalid path", "File invalid path"),
	Error.FolderNotFound:         CreateMetaResponse(http.StatusNotFound, "Folder not found", "Folder not found"),
	Error.FolderAlreadyExists:    CreateMetaResponse(http.StatusConflict, "Folder already exists", "Folder already exists"),
	Error.FolderCreateFailed:     CreateMetaResponse(http.StatusInternalServerError, "Folder create failed", "Folder create failed"),
	Error.FolderDeleteFailed:     CreateMetaResponse(http.StatusInternalServerError, "Folder delete failed", "Folder delete failed"),
	Error.FolderPermissionDenied: CreateMetaResponse(http.StatusForbidden, "Folder permission denied", "Folder permission denied"),
	Error.FolderInvalidPath:      CreateMetaResponse(http.StatusBadRequest, "Folder invalid path", "Folder invalid path"),

	Error.InvalidDateOfBirth:         CreateMetaResponse(http.StatusBadRequest, "Invalid date of birth", "Invalid date of birth"),
	Error.RoleCreationError:          CreateMetaResponse(http.StatusInternalServerError, "Role creation failed", "Role creation failed"),
	Error.SubscriptionPlanNotFound:   CreateMetaResponse(http.StatusNotFound, "Subscription plan not found", "Subscription plan not found"),
	Error.RoleNotFound:               CreateMetaResponse(http.StatusNotFound, "Role not found", "Role not found"),
	Error.UserDisableFailed:          CreateMetaResponse(http.StatusInternalServerError, "User disable failed", "User disable failed"),
	Error.InvalidWorkspaceMemberData: CreateMetaResponse(http.StatusInternalServerError, "Invalid workspace member data", "Invalid workspace member data"),
	Error.UserContextNotFound:        CreateMetaResponse(http.StatusUnauthorized, "User context not found", "User context not found"),
}

var CoreSuccessCodes = map[ResponseCode]MetaResponse{}
