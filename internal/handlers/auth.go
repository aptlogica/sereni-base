package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"serenibase/internal/dto"
	"serenibase/internal/handlers/validators"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/response"
	responseConst "serenibase/internal/utils/response/constants"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type AuthHandler struct {
	authManagementService interfaces.AuthManagementService
}

func NewAuthHandler(authManagementService interfaces.AuthManagementService) *AuthHandler {
	return &AuthHandler{authManagementService: authManagementService}
}

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
		fmt.Println("err--->", err)
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.AuthSuccess.EmailVerified, loginResp)
}

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

func (h *AuthHandler) Health(c *gin.Context) {
	c.Status(http.StatusOK)
}

func (h *AuthHandler) HealthLive(c *gin.Context) {
	c.Status(http.StatusOK)
}

func (h *AuthHandler) HealthReady(c *gin.Context) {
	c.Status(http.StatusOK)
}

// TODO: Implement proper logic for these
func (h *AuthHandler) ValidateToken(c *gin.Context) {
	c.Status(http.StatusOK)
}

func (h *AuthHandler) VerifyToken(c *gin.Context) {
	c.Status(http.StatusOK)
}

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

func (h *AuthHandler) GetUsers(c *gin.Context) {
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	fmt.Println("schemaName: ", schemaName)

	users, err := h.authManagementService.GetUsers(c.Request.Context(), schemaName)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.UserSuccess.UsersFetched, users)
}

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

	// Set workspaceID from URL parameter "id"
	workspaceID := c.Param("id")
	req.WorkspaceID = workspaceID

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	reqBy, _ := userIdVal.(string)

	err := h.authManagementService.RemoveUserFromWorkspace(c.Request.Context(), schemaName, req, reqBy)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.UserSuccess.UserRemovedFromWorkspace, nil)
}

// add into workspace handler
func (h *AuthHandler) InviteUser(c *gin.Context) {
	var req dto.CreateMemberRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	reqBy, _ := userIdVal.(string)

	if err := h.authManagementService.InviteMemberToWorkspace(c.Request.Context(), schemaName, req, reqBy); err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "User invited to workspace successfully", nil)
}

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

func (h *AuthHandler) GetBaseMembers(c *gin.Context) {
	baseID := c.Param("id")
	fmt.Println("baseID...", baseID)
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)
	fmt.Println("schemaName...", schemaName)
	baseMembers, err := h.authManagementService.GetBaseMembers(c.Request.Context(), schemaName, baseID)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "Base members retrieved successfully", baseMembers)
}

// GetWorkspaceMembersWithRole retrieves workspace members with their roles
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
