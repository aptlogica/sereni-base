package response

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	appErrors "serenibase/internal/app-errors"
	"serenibase/internal/providers/logger"
	responseConstants "serenibase/internal/utils/response/constants"
)

// StandardResponse represents a standard API response format
type StandardResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Meta    *MetaInfo   `json:"meta,omitempty"`
}

// ErrorInfo represents error information
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// CreatedResponse sends a created response
func CreatedResponse(c *gin.Context, data interface{}, message ...string) {
	response := StandardResponse{
		Success: true,
		Data:    data,
	}

	if len(message) > 0 {
		response.Message = message[0]
	} else {
		response.Message = "Resource created successfully"
	}

	c.JSON(http.StatusCreated, response)
}

type MetaInfo struct {
	Code       string `json:"code"`
	HTTPStatus int    `json:"http_status"`
}

func SendSuccess(ctx *gin.Context, code responseConstants.ResponseCode, data interface{}) {
	meta := MetaInfo{
		Code:       string(code),
		HTTPStatus: responseConstants.SuccessCodes[code].HTTPStatus,
	}
	response := StandardResponse{
		Success: true,
		Message: responseConstants.SuccessCodes[code].Message,
		Data:    data,
		Meta:    &meta,
	}
	ctx.JSON(responseConstants.SuccessCodes[code].HTTPStatus, response)
}

func SendError(ctx *gin.Context, code responseConstants.ResponseCode) {
	meta := MetaInfo{
		Code:       string(code),
		HTTPStatus: responseConstants.ErrorCodes[code].HTTPStatus,
	}
	response := StandardResponse{
		Success: false,
		Error: &ErrorInfo{
			Code:    string(code),
			Message: responseConstants.ErrorCodes[code].Message,
		},
		Meta: &meta,
	}
	ctx.JSON(responseConstants.ErrorCodes[code].HTTPStatus, response)
}

func SendErrorWithMessage(ctx *gin.Context, code responseConstants.ResponseCode, message string) {
	meta := MetaInfo{
		Code:       string(code),
		HTTPStatus: responseConstants.ErrorCodes[code].HTTPStatus,
	}
	response := StandardResponse{
		Success: false,
		Error: &ErrorInfo{
			Code:    string(code),
			Message: message,
		},
		Meta: &meta,
	}
	ctx.JSON(responseConstants.ErrorCodes[code].HTTPStatus, response)
}

func CheckAndSendError(ctx *gin.Context, err error) {
	// Log the original error
	logger.Get().Error().Err(err).Msg("API Error")

	// Check if it's an APIError (from external API)
	var apiErr *appErrors.APIError
	if errors.As(err, &apiErr) {
		// Send the error message from the API directly to the frontend
		ctx.JSON(http.StatusBadRequest, StandardResponse{
			Success: false,
			Error: &ErrorInfo{
				Code:    apiErr.Code,
				Message: apiErr.Message,
				Details: fmt.Sprintf("%v", apiErr.Details),
			},
		})
		return
	}

	code, ok := responseConstants.ErrorMapping[err]
	if !ok || code == "" {
		code = responseConstants.Error.InternalError
	}
	SendError(ctx, code)
}

func FormatValidationError(err error) []string {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		var out []string
		for _, fe := range ve {
			msg := fmt.Sprintf("Field '%s' is %s", fe.Field(), fe.Tag())
			out = append(out, msg)
		}
		return out
	}
	// fallback for non-validation errors
	return []string{err.Error()}
}

type SuccessMetaInfoSwager struct {
	Code       string `json:"code" example:"USER_CREATED"`
	HTTPStatus int    `json:"http_status" example:"201"`
}

type ErrorMetaInfoSwager struct {
	Code       string `json:"code" example:"USER_VAL_2001"`
	HTTPStatus int    `json:"http_status" example:"400"`
}

type SuccessResponseSwager struct {
	Success bool                   `json:"success" example:"true"`
	Message string                 `json:"message,omitempty" example:"User created successfully"`
	Data    interface{}            `json:"data,omitempty"`
	Meta    *SuccessMetaInfoSwager `json:"meta,omitempty"`
}

type ErrorResponseSwager struct {
	Success bool                 `json:"success" example:"false"`
	Message string               `json:"message,omitempty" example:"Invalid input"`
	Error   *ErrorInfoSwager     `json:"error,omitempty"`
	Meta    *ErrorMetaInfoSwager `json:"meta,omitempty"`
}

type ErrorInfoSwager struct {
	Code    string `json:"code" example:"USER_VAL_2001"`
	Message string `json:"message" example:"Email is required"`
}
