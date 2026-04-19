// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/handlers/validators"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/response"
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type AuthHandler struct {
	authManagementService interfaces.AuthManagementService
}

func NewAuthHandler(authManagementService interfaces.AuthManagementService) *AuthHandler {
	return &AuthHandler{authManagementService: authManagementService}
}

// @Summary      Authenticate with email and password
// @Description  Verifies the provided email/password pair and returns the refreshed login tokens plus the user profile when the credentials are correct.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.LoginRequest  true  "Login payload with email and password"
// @Success      200      {object}  dto.LoginResponse  "Login successful, returns user details and tokens"
// @Failure      400      {object}  models.ErrorResponse "Bad Request — malformed JSON or missing required fields"
// @Failure      401      {object}  models.ErrorResponse "Unauthorized — invalid credentials"
// @Failure      422      {object}  models.ErrorResponse "Unprocessable Entity — validation failed on one or more fields"
// @Failure      500      {object}  models.ErrorResponse "Internal Server Error"
// @Router       /auth/login [post]
func (h *AuthHandler) LoginUser(c *gin.Context) {
	var req dto.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.LoginValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	user, err := h.authManagementService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.AuthSuccess.UserLogin, user)
}

// @Summary      Verify email via OTP
// @Description  Consumes the OTP and token from the user to confirm ownership of the mailbox, then issues the standard login payload.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.VerifyEmailRequest  true  "Token and OTP used for email verification"
// @Success      200      {object}  dto.LoginResponse       "Email verified and login response returned"
// @Failure      400      {object}  models.ErrorResponse    "Bad Request — malformed token/OTP or missing parameters"
// @Failure      401      {object}  models.ErrorResponse    "Unauthorized — OTP invalid or expired"
// @Failure      422      {object}  models.ErrorResponse    "Unprocessable Entity — validation failed"
// @Failure      500      {object}  models.ErrorResponse    "Internal Server Error"
// @Router       /auth/otp/verify [post]
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req dto.VerifyEmailRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.VerifyEmailRequestError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	loginResp, err := h.authManagementService.VerifyEmail(c.Request.Context(), req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.AuthSuccess.EmailVerified, loginResp)
}

// @Summary      Resend throttled OTP
// @Description  Reissues one-time password SMS/email for the given verification token when the previous code expired or was lost.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.ResendOTPRequest  true  "Existing verification token"
// @Success      200      {object}  models.SuccessResponse  "OTP resent successfully"
// @Failure      400      {object}  models.ErrorResponse    "Bad Request — missing or invalid token"
// @Failure      401      {object}  models.ErrorResponse    "Unauthorized — token already consumed"
// @Failure      429      {object}  models.ErrorResponse    "Too Many Requests — rate limit exceeded for OTP resends"
// @Failure      500      {object}  models.ErrorResponse    "Internal Server Error"
// @Router       /auth/otp/resend [post]
func (h *AuthHandler) ResendOTP(c *gin.Context) {
	var req dto.ResendOTPRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.VerifyResendOtpRequestError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	err := h.authManagementService.ResendOTP(c.Request.Context(), req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.AuthSuccess.ResendOTP, nil)
}

// @Summary      Refresh authentication tokens
// @Description  Exchanges a refresh token for a new access/refresh token pair so ongoing sessions stay authenticated.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.RefreshTokenRequest  true  "Refresh token payload"
// @Success      200      {object}  dto.TokenResponse        "New access and refresh tokens"
// @Failure      400      {object}  models.ErrorResponse    "Bad Request — missing refresh token"
// @Failure      401      {object}  models.ErrorResponse    "Unauthorized — refresh token invalid or revoked"
// @Failure      422      {object}  models.ErrorResponse    "Unprocessable Entity — validation failed"
// @Failure      500      {object}  models.ErrorResponse    "Internal Server Error"
// @Router       /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.RefreshTokenRequestError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	refreshResp, err := h.authManagementService.RefreshToken(c.Request.Context(), req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.AuthSuccess.RefreshToken, refreshResp)

}

// @Summary      Start password reset flow
// @Description  Accepts a registered email, sends a reset token, and records the request to prevent spam.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.ForgotPasswordRequest  true  "Email address that owns the account"
// @Success      200      {object}  models.SuccessResponse     "Password reset token issued"
// @Failure      400      {object}  models.ErrorResponse       "Bad Request — missing or invalid email"
// @Failure      404      {object}  models.ErrorResponse       "Not Found — email not registered"
// @Failure      500      {object}  models.ErrorResponse       "Internal Server Error"
// @Router       /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.ForgotPasswordRequestError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	err := h.authManagementService.ForgotPassword(c.Request.Context(), req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.AuthSuccess.ForgotPassword, nil)
}

// @Summary      Reset password using token
// @Description  Validates the reset token and new password, updates the stored credentials, and invalidates the token.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.ResetPasswordRequest  true  "Reset token and new password payload"
// @Success      200      {object}  models.SuccessResponse    "Password updated successfully"
// @Failure      400      {object}  models.ErrorResponse      "Bad Request — missing fields or invalid token format"
// @Failure      401      {object}  models.ErrorResponse      "Unauthorized — token expired or already used"
// @Failure      422      {object}  models.ErrorResponse      "Unprocessable Entity — validation failed"
// @Failure      500      {object}  models.ErrorResponse      "Internal Server Error"
// @Router       /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.ResetPasswordRequestError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	err := h.authManagementService.ResetPassword(c.Request.Context(), req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.AuthSuccess.ResetPassword, nil)
}

// @Summary      Health probe
// @Description  Returns 200 to signal the authentication handler is running.
// @Tags         System
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Success      200      {object}  models.SuccessResponse  "Service is healthy"
// @Router       /health [get]
func (h *AuthHandler) Health(c *gin.Context) {
	c.Status(http.StatusOK)
}

// @Summary      Liveness probe
// @Description  Quick check that the auth handler process is still alive.
// @Tags         System
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Success      200      {object}  models.SuccessResponse  "Process is alive"
// @Router       /health/live [get]
func (h *AuthHandler) HealthLive(c *gin.Context) {
	c.Status(http.StatusOK)
}

// @Summary      Readiness probe
// @Description  Indicates the authentication handler has finished startup routines and is ready to accept requests.
// @Tags         System
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Success      200      {object}  models.SuccessResponse  "Service is ready"
// @Router       /health/ready [get]
func (h *AuthHandler) HealthReady(c *gin.Context) {
	c.Status(http.StatusOK)
}

// @Summary      Validate a token
// @Description  Checks whether a token is still valid and returns metadata about the owner.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.TokenValidationRequest  true  "Token to validate"
// @Success      200      {object}  dto.TokenValidationResponse   "Token validity and metadata"
// @Failure      400      {object}  models.ErrorResponse          "Bad Request — malformed token"
// @Failure      401      {object}  models.ErrorResponse          "Unauthorized — token invalid"
// @Failure      422      {object}  models.ErrorResponse          "Unprocessable Entity — validation failed"
// @Failure      500      {object}  models.ErrorResponse          "Internal Server Error"
// @Router       /auth/validate-token [post]
func (h *AuthHandler) ValidateToken(c *gin.Context) {
	var req dto.TokenValidationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.ValidateTokenRequestError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	resp, err := h.authManagementService.ValidateToken(c.Request.Context(), req.Token)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.AuthSuccess.ValidateToken, resp)
}

// @Summary      Verify a token issued by the system
// @Description  Ensures the supplied token is authentic and returns the user metadata embedded within it.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.TokenValidationRequest  true  "Token to verify"
// @Success      200      {object}  dto.TokenValidationResponse   "Token signature verified"
// @Failure      400      {object}  models.ErrorResponse          "Bad Request — missing token"
// @Failure      401      {object}  models.ErrorResponse          "Unauthorized — token invalid or expired"
// @Failure      422      {object}  models.ErrorResponse          "Unprocessable Entity — validation failed"
// @Failure      500      {object}  models.ErrorResponse          "Internal Server Error"
// @Router       /auth/verify-token [post]
func (h *AuthHandler) VerifyToken(c *gin.Context) {
	var req dto.TokenValidationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.ValidateTokenRequestError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	resp, err := h.authManagementService.VerifyToken(c.Request.Context(), req.Token)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.AuthSuccess.VerifyToken, resp)
}

// @Summary      Log out current session
// @Description  Invalidates the supplied refresh token so the session cannot be reused.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.LogoutRequest  true  "Refresh token to invalidate"
// @Success      200      {object}  models.SuccessResponse  "Logout succeeded"
// @Failure      400      {object}  models.ErrorResponse    "Bad Request — missing field"
// @Failure      401      {object}  models.ErrorResponse    "Unauthorized — token already invalid or missing"
// @Failure      500      {object}  models.ErrorResponse    "Internal Server Error"
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req dto.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.LogoutRequestError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	err := h.authManagementService.Logout(c.Request.Context(), req.Token)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.AuthSuccess.UserLogout, nil)
}

// @Summary      Invite or onboard a new user
// @Description  Accepts multipart data to create a workspace member, including optional membership JSON and profile picture.
// @Tags         Users
// @Accept       multipart/form-data
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request     body      dto.AddUserRequest  true   "Form payload describing the user and memberships"
// @Param        profile_pic formData  file               false  "Optional profile picture"
// @Success      201         {object}  models.SuccessResponse "User added successfully"
// @Failure      400         {object}  models.ErrorResponse   "Bad Request — missing required fields"
// @Failure      401         {object}  models.ErrorResponse   "Unauthorized — invalid Bearer token"
// @Failure      403         {object}  models.ErrorResponse   "Forbidden — insufficient permissions"
// @Failure      409         {object}  models.ErrorResponse   "Conflict — user already exists"
// @Failure      422         {object}  models.ErrorResponse   "Unprocessable Entity — validation failed"
// @Failure      500         {object}  models.ErrorResponse   "Internal Server Error"
// @Security     BearerAuth
// @Router       /user/create [post]
func (h *AuthHandler) AddUser(c *gin.Context) {
	var req dto.AddUserRequest

	// Bind form data (firstname, lastname, email)
	if err := c.ShouldBind(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.AddUserRequestError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	// Handle profile picture file upload
	if req.ProfilePic != nil {
		// You now have the file header directly from binding
		// Can save the file or process it
		// Example: c.SaveUploadedFile(req.ProfilePic, "./uploads/"+req.ProfilePic.Filename)
		fmt.Println("File uploaded:", req.ProfilePic.Filename, "Size:", req.ProfilePic.Size)
	}

	// Parse membership JSON array from form field
	membershipStr := c.PostForm("membership")
	if membershipStr != "" && membershipStr != "[]" {
		var membership []dto.MembershipRequest
		if err := json.Unmarshal([]byte(membershipStr), &membership); err != nil {
			response.CheckAndSendError(c, fmt.Errorf("invalid membership format: %v", err))
			return
		}
		req.Membership = membership
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	reqBy, _ := userIdVal.(string)

	_, err := h.authManagementService.AddUser(c.Request.Context(), schemaName, req, reqBy)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.UserSuccess.UserAdded, nil)
}

// @Summary      Update an existing user
// @Description  Applies partial updates, optional membership adjustments, and profile picture replacements to the specified user.
// @Tags         Users
// @Accept       multipart/form-data
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request     body      dto.EditUserRequest  true   "Payload with user_id plus optional fields to patch"
// @Param        profile_pic formData  file               false  "Replace the profile picture"
// @Success      200         {object}  dto.UserResponse     "User returned after update"
// @Failure      400         {object}  models.ErrorResponse "Bad Request — missing user_id or invalid format"
// @Failure      401         {object}  models.ErrorResponse "Unauthorized — invalid Bearer token"
// @Failure      403         {object}  models.ErrorResponse "Forbidden — cannot edit another tenant"
// @Failure      404         {object}  models.ErrorResponse "Not Found — user does not exist"
// @Failure      422         {object}  models.ErrorResponse "Unprocessable Entity — validation error on fields"
// @Failure      500         {object}  models.ErrorResponse "Internal Server Error"
// @Security     BearerAuth
// @Router       /user/edit [post]
func (h *AuthHandler) EditUser(c *gin.Context) {
	var req dto.EditUserRequest

	// Get user_id from form
	userID := c.PostForm("user_id")
	if userID == "" {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}
	req.UserID = userID

	// Get optional firstname from form
	firstname := c.PostForm("firstname")
	if firstname != "" {
		req.FirstName = &firstname
	}

	// Get optional lastname from form
	lastname := c.PostForm("lastname")
	if lastname != "" {
		req.LastName = &lastname
	}

	// Get optional is_coowner from form
	isCoOwnerStr := c.PostForm("is_coowner")
	if isCoOwnerStr != "" {
		// Handle various boolean representations (case-insensitive, yes/no, true/false, 1/0)
		isCoOwner := strings.ToLower(isCoOwnerStr) == "true" ||
			strings.ToLower(isCoOwnerStr) == "yes" ||
			isCoOwnerStr == "1"
		req.IsCoOwner = &isCoOwner
	}

	// Handle profile picture file upload if provided
	fileHeader, err := c.FormFile("profile_pic")
	if err == nil && fileHeader != nil {
		req.ProfilePic = fileHeader
	}

	// Parse membership JSON array from form field if provided
	membershipStr := c.PostForm("membership")
	if membershipStr != "" && membershipStr != "[]" {
		var membership []dto.MembershipRequest
		if err := json.Unmarshal([]byte(membershipStr), &membership); err != nil {
			response.CheckAndSendError(c, fmt.Errorf("invalid membership format: %v", err))
			return
		}
		req.Membership = membership
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	reqBy, _ := userIdVal.(string)

	updatedUser, err := h.authManagementService.EditUser(c.Request.Context(), schemaName, req, reqBy)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.UserSuccess.UserUpdated, updatedUser)
}

// @Summary      Fully remove a user
// @Description  Permanently deletes the user with the supplied ID from the tenant schema.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.RemoveUserRequest  true  "User ID to delete"
// @Success      200      {object}  models.SuccessResponse    "User removed successfully"
// @Failure      400      {object}  models.ErrorResponse      "Bad Request — missing user_id"
// @Failure      401      {object}  models.ErrorResponse      "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse      "Forbidden — insufficient privileges"
// @Failure      404      {object}  models.ErrorResponse      "Not Found — user already removed"
// @Failure      500      {object}  models.ErrorResponse      "Internal Server Error"
// @Security     BearerAuth
// @Router       /user/remove [post]
func (h *AuthHandler) RemoveUser(c *gin.Context) {
	var req dto.RemoveUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.RemoveUserRequestError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	err := h.authManagementService.DeleteUserCompletely(c.Request.Context(), schemaName, req.UserID)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.UserSuccess.UserRemoved, nil)
}

// @Summary      List all users
// @Description  Returns every user in the tenant schema along with their role definitions.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Success      200      {array}   dto.UserWithRole  "Users retrieved successfully"
// @Failure      401      {object}  models.ErrorResponse  "Unauthorized — invalid bearer token"
// @Failure      403      {object}  models.ErrorResponse  "Forbidden — insufficient access to list users"
// @Failure      500      {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /user/list [get]
func (h *AuthHandler) GetUsers(c *gin.Context) {
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	users, err := h.authManagementService.GetUsers(c.Request.Context(), schemaName)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.UserSuccess.UsersFetched, users)
}

// @Summary      List assignable users
// @Description  Retrieves only users marked as active so they can be assigned to new workspaces or bases.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Success      200      {array}   dto.UserWithRole  "Assignable users returned"
// @Failure      401      {object}  models.ErrorResponse  "Unauthorized — invalid token"
// @Failure      403      {object}  models.ErrorResponse  "Forbidden — needs higher privileges"
// @Failure      500      {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /user/list-for-assign [get]
func (h *AuthHandler) GetActiveUsersForAssign(c *gin.Context) {
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	users, err := h.authManagementService.GetActiveUsersForAssign(c.Request.Context(), schemaName)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.UserSuccess.UsersFetched, users)
}

// add into user handler
// @Summary      Grant workspace access
// @Description  Links a user to a workspace using the membership definition provided in the payload.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.CreateMemberRequest  true  "Membership payload for workspace assignment"
// @Success      200      {object}  models.SuccessResponse    "User assigned to workspace"
// @Failure      400      {object}  models.ErrorResponse      "Bad Request — missing or invalid payload"
// @Failure      401      {object}  models.ErrorResponse      "Unauthorized — invalid token"
// @Failure      403      {object}  models.ErrorResponse      "Forbidden — insufficient role"
// @Failure      500      {object}  models.ErrorResponse      "Internal Server Error"
// @Security     BearerAuth
// @Router       /user/assign [post]
func (h *AuthHandler) AssignUserToWorkspace(c *gin.Context) {
	var req dto.CreateMemberRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.CreateMemberRequestError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	reqBy, _ := userIdVal.(string)

	// NOTE: You should implement this method on your authManagementService!
	err := h.authManagementService.AssignUserToWorkspace(c.Request.Context(), schemaName, req, reqBy)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.UserSuccess.UserAssignedToWorkspace, nil)
}

// @Summary      Update user workspace access
// @Description  Adjusts workspace or base memberships for the specified user and returns success once changes are persisted.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.CreateMemberRequest  true  "Membership payload for updating access"
// @Success      200      {object}  models.SuccessResponse    "User access updated"
// @Failure      400      {object}  models.ErrorResponse      "Bad Request — missing or invalid payload"
// @Failure      401      {object}  models.ErrorResponse      "Unauthorized — invalid bearer token"
// @Failure      403      {object}  models.ErrorResponse      "Forbidden — cannot escalate role"
// @Failure      500      {object}  models.ErrorResponse      "Internal Server Error"
// @Security     BearerAuth
// @Router       /user/access/update [put]
func (h *AuthHandler) UpdateUserAccess(c *gin.Context) {
	var req dto.CreateMemberRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.CreateMemberRequestError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	reqBy, _ := userIdVal.(string)

	// NOTE: UpdateUserAccess uses the same service method as AssignUserToWorkspace
	// It will detect if user already has access and update accordingly
	err := h.authManagementService.AssignUserToWorkspace(c.Request.Context(), schemaName, req, reqBy)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "User access updated successfully", nil)
}

// @Summary      Remove a member from a workspace
// @Description  Deletes the specified user from the workspace and optionally cleans up linked access entries.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id       path      string                 true  "Workspace ID"
// @Param        request  body      dto.RemoveMemberRequest  true  "User to remove from workspace"
// @Success      200      {object}  models.SuccessResponse    "Member removed successfully"
// @Failure      400      {object}  models.ErrorResponse      "Bad Request — invalid workspace id or payload"
// @Failure      401      {object}  models.ErrorResponse      "Unauthorized — invalid token"
// @Failure      403      {object}  models.ErrorResponse      "Forbidden — insufficient role to remove member"
// @Failure      404      {object}  models.ErrorResponse      "Not Found — member or workspace missing"
// @Failure      500      {object}  models.ErrorResponse      "Internal Server Error"
// @Security     BearerAuth
// @Router       /workspace/{id}/remove [post]
func (h *AuthHandler) RemoveUserFromWorkspace(c *gin.Context) {
	var req dto.RemoveMemberRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.RemoveMemberRequestError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	// Get workspaceID from URL parameter "id"
	workspaceID := c.Param("id")

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	reqBy, _ := userIdVal.(string)

	err := h.authManagementService.RemoveUserFromWorkspace(c.Request.Context(), schemaName, workspaceID, req.UserID, reqBy)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.UserSuccess.UserRemovedFromWorkspace, nil)
}

// @Summary      Remove a member from a base
// @Description  Removes access for the supplied user ID on the specific base.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id       path      string                 true  "Base ID"
// @Param        request  body      dto.RemoveMemberRequest  true  "User to revoke base membership from"
// @Success      200      {object}  models.SuccessResponse    "User removed from base"
// @Failure      400      {object}  models.ErrorResponse      "Bad Request — invalid base or payload"
// @Failure      401      {object}  models.ErrorResponse      "Unauthorized — invalid token"
// @Failure      403      {object}  models.ErrorResponse      "Forbidden — insufficient permissions"
// @Failure      404      {object}  models.ErrorResponse      "Not Found — base or user missing"
// @Failure      500      {object}  models.ErrorResponse      "Internal Server Error"
// @Security     BearerAuth
// @Router       /base/{id}/remove [post]
func (h *AuthHandler) RemoveUserFromBase(c *gin.Context) {
	var req dto.RemoveMemberRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.RemoveMemberRequestError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	// Get baseID from URL parameter "id"
	baseID := c.Param("id")

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	reqBy, _ := userIdVal.(string)

	err := h.authManagementService.RemoveUserFromBase(c.Request.Context(), schemaName, baseID, req.UserID, reqBy)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.UserSuccess.UserRemovedFromWorkspace, nil)
}

// @Summary      List workspace members
// @Description  Retrieves every member assigned to the workspace along with their role metadata.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path   string  true  "Workspace ID"
// @Success      200  {array} dto.WorkspaceMemberResponse  "Workspace members retrieved"
// @Failure      401  {object} models.ErrorResponse        "Unauthorized"
// @Failure      403  {object} models.ErrorResponse        "Forbidden — insufficient workspace access"
// @Failure      404  {object} models.ErrorResponse        "Not Found — workspace not found"
// @Failure      500  {object} models.ErrorResponse        "Internal Server Error"
// @Security     BearerAuth
// @Router       /workspace/{id}/members [get]
func (h *AuthHandler) GetWorkspaceMembers(c *gin.Context) {
	workspaceID := c.Param("id")

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	members, err := h.authManagementService.GetWorkspaceMembers(c.Request.Context(), schemaName, workspaceID)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "Workspace members retrieved successfully", members)
}

// @Summary      List base members
// @Description  Returns all users attached to the base and their assigned roles.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path   string  true  "Base ID"
// @Success      200  {array}  dto.WorkspaceMemberResponse  "Base members retrieved"
// @Failure      401  {object} models.ErrorResponse         "Unauthorized"
// @Failure      403  {object} models.ErrorResponse         "Forbidden — insufficient base access"
// @Failure      404  {object} models.ErrorResponse         "Not Found — base missing"
// @Failure      500  {object} models.ErrorResponse         "Internal Server Error"
// @Security     BearerAuth
// @Router       /base/{id}/members [get]
func (h *AuthHandler) GetBaseMembers(c *gin.Context) {
	baseID := c.Param("id")
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)
	baseMembers, err := h.authManagementService.GetBaseMembers(c.Request.Context(), schemaName, baseID)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "Base members retrieved successfully", baseMembers)
}

// GetWorkspaceMembersWithRole retrieves workspace members with their roles
// @Summary      List workspace members with roles
// @Description  Returns workspace members annotated with their RBAC role summaries.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path   string  true  "Workspace ID"
// @Success      200  {array}  dto.UserWithRole  "Members with role metadata"
// @Failure      401  {object} models.ErrorResponse  "Unauthorized"
// @Failure      403  {object} models.ErrorResponse  "Forbidden"
// @Failure      404  {object} models.ErrorResponse  "Not Found"
// @Failure      500  {object} models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /workspace/{id}/members-with-roles [get]
func (h *AuthHandler) GetWorkspaceMembersWithRole(c *gin.Context) {
	workspaceID := c.Param("id")
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	members, err := h.authManagementService.GetWorkspaceMembersWithRole(c.Request.Context(), schemaName, workspaceID)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "Workspace members with roles retrieved successfully", members)
}

// GetBaseMembersWithRole retrieves base members with their roles
// @Summary      List base members with roles
// @Description  Returns base members annotated with their assigned roles for RBAC audits.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path   string  true  "Base ID"
// @Success      200  {array}  dto.UserWithRole  "Members with RBAC role data"
// @Failure      401  {object} models.ErrorResponse  "Unauthorized"
// @Failure      403  {object} models.ErrorResponse  "Forbidden"
// @Failure      404  {object} models.ErrorResponse  "Not Found"
// @Failure      500  {object} models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /base/{id}/members-with-roles [get]
func (h *AuthHandler) GetBaseMembersWithRole(c *gin.Context) {
	baseID := c.Param("id")
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	members, err := h.authManagementService.GetBaseMembersWithRole(c.Request.Context(), schemaName, baseID)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "Base members with roles retrieved successfully", members)
}

// @Summary      Update user password
// @Description  Validates the old password and applies the new password for the URL-specified user.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id       path      string                      true  "User ID"
// @Param        request  body      dto.UpdateUserPasswordRequest  true  "Old and new password pair"
// @Success      200      {object}  models.SuccessResponse          "Password updated"
// @Failure      400      {object}  models.ErrorResponse            "Bad Request — missing fields or invalid id"
// @Failure      401      {object}  models.ErrorResponse            "Unauthorized — invalid token"
// @Failure      403      {object}  models.ErrorResponse            "Forbidden — cannot change another user's password"
// @Failure      422      {object}  models.ErrorResponse            "Unprocessable Entity — validation error"
// @Failure      500      {object}  models.ErrorResponse            "Internal Server Error"
// @Security     BearerAuth
// @Router       /user/change-password/{id} [post]
func (h *AuthHandler) UpdatePassword(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	var updatePayload dto.UpdateUserPasswordRequest
	if err := c.ShouldBindJSON(&updatePayload); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.UpdateUserPasswordValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	err := h.authManagementService.UpdatePassword(c.Request.Context(), schemaName, id, updatePayload)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.UserSuccess.PasswordUpdated, nil)
}

// @Summary      Activate a disabled user
// @Description  Sets the user status to active and returns the refreshed profile.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.ActivateUserRequest  true  "User ID to activate"
// @Success      200      {object}  dto.UserResponse          "Activated user's profile"
// @Failure      400      {object}  models.ErrorResponse      "Bad Request — missing user_id"
// @Failure      401      {object}  models.ErrorResponse      "Unauthorized — invalid token"
// @Failure      403      {object}  models.ErrorResponse      "Forbidden — insufficient privileges"
// @Failure      404      {object}  models.ErrorResponse      "Not Found — user not found"
// @Failure      500      {object}  models.ErrorResponse      "Internal Server Error"
// @Security     BearerAuth
// @Router       /user/activate [post]
func (h *AuthHandler) ActivateUser(c *gin.Context) {
	var req dto.ActivateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.ActivateUserRequestError(ve[0]))
			return
		}
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	updatedProfile, err := h.authManagementService.ActivateUser(c.Request.Context(), schemaName, req.UserID)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.UserSuccess.UserUpdated, updatedProfile)
}

// @Summary      Deactivate a user account
// @Description  Marks the specified user as inactive and returns the updated profile.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.DeactivateUserRequest  true  "User ID to deactivate"
// @Success      200      {object}  dto.UserResponse            "User deactivated successfully"
// @Failure      400      {object}  models.ErrorResponse        "Bad Request — invalid request"
// @Failure      401      {object}  models.ErrorResponse        "Unauthorized — invalid token"
// @Failure      403      {object}  models.ErrorResponse        "Forbidden — insufficient privileges"
// @Failure      404      {object}  models.ErrorResponse        "Not Found — user missing"
// @Failure      500      {object}  models.ErrorResponse        "Internal Server Error"
// @Security     BearerAuth
// @Router       /user/deactivate [post]
func (h *AuthHandler) DeactivateUser(c *gin.Context) {
	var req dto.DeactivateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.DeactivateUserRequestError(ve[0]))
			return
		}
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	updatedProfile, err := h.authManagementService.DeactivateUser(c.Request.Context(), schemaName, req.UserID)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.UserSuccess.UserUpdated, updatedProfile)
}

// @Summary      Delete access member
// @Description  Deletes an access member entry by its ID, removing any inherited workspace/base privileges.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string  true  "Access member ID"
// @Success      200  {object}  models.SuccessResponse  "Access entry removed"
// @Failure      400  {object}  models.ErrorResponse    "Bad Request — missing lane identifier"
// @Failure      401  {object}  models.ErrorResponse    "Unauthorized — invalid bearer token"
// @Failure      403  {object}  models.ErrorResponse    "Forbidden — insufficient permissions"
// @Failure      404  {object}  models.ErrorResponse    "Not Found — access member already deleted"
// @Failure      500  {object}  models.ErrorResponse    "Internal Server Error"
// @Security     BearerAuth
// @Router       /workspace/access/{id} [delete]
func (h *AuthHandler) RemoveAccessMemberByID(c *gin.Context) {
	// Get access_member_id from URL parameter
	accessMemberID := c.Param("id")

	if accessMemberID == "" {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	reqBy, _ := userIdVal.(string)

	err := h.authManagementService.RemoveAccessMemberByID(c.Request.Context(), schemaName, accessMemberID, reqBy)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.UserSuccess.UserRemovedFromWorkspace, nil)
}
