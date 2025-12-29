package handlers

import (
	"fmt"
	"serenibase/internal/dto"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/response"
	responseConst "serenibase/internal/utils/response/constants"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userManagementService interfaces.UserManagementService
}

func NewUserHandler(
	userManagementService interfaces.UserManagementService,
) *UserHandler {
	return &UserHandler{
		userManagementService: userManagementService,
	}
}

func (h *UserHandler) GetUserProfileByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userProfile, err := h.userManagementService.GetUserProfileByID(c.Request.Context(), schemaName, id)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	if userProfile.DateOfBirth != nil {
		dateStr := *userProfile.DateOfBirth
		userProfile.DateOfBirth = &dateStr
	}
	response.SendSuccess(c, responseConst.UserSuccess.UserFetched, userProfile)
}

func (h *UserHandler) UpdateUserProfile(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	var updatePayload dto.UpdateUserProfileRequest
	if err := c.ShouldBindJSON(&updatePayload); err != nil {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	updatedProfile, err := h.userManagementService.UpdateUserProfile(c.Request.Context(), schemaName, id, updatePayload)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.UserSuccess.UserUpdated, updatedProfile)
}

func (h *UserHandler) AddAvatar(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	avatarURL, err := h.userManagementService.AddAvatar(c.Request.Context(), schemaName, id, file)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.UserSuccess.AvatarAdded, gin.H{"avatar_url": avatarURL})
}

func (h *UserHandler) RemoveAvatar(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	avatarURL, err := h.userManagementService.RemoveAvatar(c.Request.Context(), schemaName, id)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.UserSuccess.AvatarRemoved, avatarURL)
}

func (h *UserHandler) GetWorkspaces(c *gin.Context) {
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)

	rolesVal, _ := c.Get("roles")
	roles, _ := rolesVal.(string)

	workspaces, err := h.userManagementService.GetWorkspaces(c.Request.Context(), schemaName, userId, roles)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.UserSuccess.WorkspaceFetched, workspaces)
}

func (h *UserHandler) GetUserAccessDetails(c *gin.Context) {
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	// Check if user has Admin role
	rolesVal, _ := c.Get("roles")
	roles, _ := rolesVal.(string)

	if roles != "Admin" {
		response.SendError(c, responseConst.Error.UnauthorizedAccess)
		return
	}

	// Get user_id from query parameter
	userId := c.Query("user_id")
	if userId == "" {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	// Get optional workspace_id from query parameter
	workspaceId := c.Query("workspace_id")

	fmt.Println("role-->", rolesVal)

	accessDetails, err := h.userManagementService.GetUserAccessDetails(c.Request.Context(), schemaName, userId, roles, workspaceId)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.UserSuccess.UserAccessDetailsFetched, accessDetails)
}

// GetUserRolesAndAccess retrieves user's roles and access information organized by workspace and base
func (h *UserHandler) GetUserRolesAndAccess(c *gin.Context) {
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIDVal, _ := c.Get("user_id")
	userID, _ := userIDVal.(string)

	if userID == "" {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	rolesAndAccess, err := h.userManagementService.GetUserRolesAndAccess(c.Request.Context(), schemaName, userID)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.UserSuccess.UserFetched, rolesAndAccess)
}
