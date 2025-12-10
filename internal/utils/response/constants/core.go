package constants

import "net/http"

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
}

var CoreErrorCodes = map[ResponseCode]MetaResponse{
	Error.UnauthorizedAccess:       {HTTPStatus: http.StatusUnauthorized, Message: "Unauthorized access", Description: "Unauthorized access"},
	Error.Forbidden:                {HTTPStatus: http.StatusForbidden, Message: "Forbidden", Description: "Forbidden"},
	Error.SessionExpired:           {HTTPStatus: http.StatusUnauthorized, Message: "Unauthorized access", Description: "Session expired"},
	Error.InvalidPayload:           {HTTPStatus: http.StatusBadRequest, Message: "Bad request", Description: "Invalid request payload"},
	Error.ValidationFailed:         {HTTPStatus: http.StatusUnprocessableEntity, Message: "Validation failed", Description: "Validation failed"},
	Error.DatabaseError:            {HTTPStatus: http.StatusInternalServerError, Message: "Internal server error", Description: "Database error"},
	Error.RecordNotFound:           {HTTPStatus: http.StatusNotFound, Message: "Record not found", Description: "Record not found"},
	Error.InternalError:            {HTTPStatus: http.StatusInternalServerError, Message: "Internal server error", Description: "Internal server error"},
	Error.ServiceUnavailable:       {HTTPStatus: http.StatusServiceUnavailable, Message: "Service unavailable", Description: "Service unavailable"},
	Error.GatewayTimeout:           {HTTPStatus: http.StatusGatewayTimeout, Message: "Gateway timeout", Description: "Gateway timeout"},
	Error.TooManyRequests:          {HTTPStatus: http.StatusTooManyRequests, Message: "Too many requests", Description: "Too many requests"},
	Error.Conflict:                 {HTTPStatus: http.StatusConflict, Message: "Conflict", Description: "Conflict"},
	Error.BadRequest:               {HTTPStatus: http.StatusBadRequest, Message: "Bad request", Description: "Bad request"},
	Error.InvalidID:                {HTTPStatus: http.StatusBadRequest, Message: "Invalid ID", Description: "The provided ID is invalid"},
	Error.UserAlreadyExists:        {HTTPStatus: http.StatusConflict, Message: "User already exists", Description: "User already exists"},
	Error.RecordAlreadyExists:      {HTTPStatus: http.StatusConflict, Message: "Record already exists", Description: "Record already exists"},
	Error.RecordNotInserted:        {HTTPStatus: http.StatusInternalServerError, Message: "Record not inserted", Description: "Failed to insert record"},
	Error.NotImplemented:           {HTTPStatus: http.StatusNotImplemented, Message: "Not implemented", Description: "This feature is not implemented"},
	Error.Timeout:                  {HTTPStatus: http.StatusRequestTimeout, Message: "Timeout", Description: "The request timed out"},
	Error.DependencyFailed:         {HTTPStatus: http.StatusFailedDependency, Message: "Dependency failed", Description: "A dependency failed to process the request"},
	Error.MapToStructError:         {HTTPStatus: http.StatusInternalServerError, Message: "Map to struct error", Description: "Failed to map data to struct"},
	Error.StructToStructError:      {HTTPStatus: http.StatusInternalServerError, Message: "Struct to struct error", Description: "Failed to map struct to struct"},
	Error.HashingError:             {HTTPStatus: http.StatusInternalServerError, Message: "Hashing error", Description: "Failed to hash data"},
	Error.InvalidCredentials:       {HTTPStatus: http.StatusUnauthorized, Message: "Invalid credentials", Description: "The provided credentials are invalid"},
	Error.InvalidDriver:            {HTTPStatus: http.StatusBadRequest, Message: "Invalid driver", Description: "The provided driver is invalid or not supported"},
	Error.ErrNotFound:              {HTTPStatus: http.StatusNotFound, Message: "Not found", Description: "The requested resource could not be found"},
	Error.JSONMarshalError:         {HTTPStatus: http.StatusInternalServerError, Message: "JSON marshal error", Description: "Failed to marshal data to JSON"},
	Error.HTTPRequestCreationError: {HTTPStatus: http.StatusInternalServerError, Message: "HTTP request creation error", Description: "Failed to create HTTP request"},
	Error.HTTPDoRequestError:       {HTTPStatus: http.StatusInternalServerError, Message: "HTTP do request error", Description: "Failed to execute HTTP request"},

	Error.FileNotFound:           {HTTPStatus: http.StatusNotFound, Message: "File not found", Description: "The requested file was not found"},
	Error.FileAlreadyExists:      {HTTPStatus: http.StatusConflict, Message: "File already exists", Description: "The file already exists"},
	Error.FileReadFailed:         {HTTPStatus: http.StatusInternalServerError, Message: "File read failed", Description: "Failed to read the file"},
	Error.FileWriteFailed:        {HTTPStatus: http.StatusInternalServerError, Message: "File write failed", Description: "Failed to write the file"},
	Error.FileDeleteFailed:       {HTTPStatus: http.StatusInternalServerError, Message: "File delete failed", Description: "Failed to delete the file"},
	Error.FilePermissionDenied:   {HTTPStatus: http.StatusForbidden, Message: "File permission denied", Description: "Permission denied for file operation"},
	Error.FileInvalidPath:        {HTTPStatus: http.StatusBadRequest, Message: "File invalid path", Description: "The file path provided is invalid"},
	Error.FolderNotFound:         {HTTPStatus: http.StatusNotFound, Message: "Folder not found", Description: "The requested folder was not found"},
	Error.FolderAlreadyExists:    {HTTPStatus: http.StatusConflict, Message: "Folder already exists", Description: "The folder already exists"},
	Error.FolderCreateFailed:     {HTTPStatus: http.StatusInternalServerError, Message: "Folder create failed", Description: "Failed to create the folder"},
	Error.FolderDeleteFailed:     {HTTPStatus: http.StatusInternalServerError, Message: "Folder delete failed", Description: "Failed to delete the folder"},
	Error.FolderPermissionDenied: {HTTPStatus: http.StatusForbidden, Message: "Folder permission denied", Description: "Permission denied for folder operation"},
	Error.FolderInvalidPath:      {HTTPStatus: http.StatusBadRequest, Message: "Folder invalid path", Description: "The folder path provided is invalid"},
}

var CoreSuccessCodes = map[ResponseCode]MetaResponse{}
