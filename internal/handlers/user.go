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
	if err := c.ShouldBind(&updatePayload); err != nil {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	var updatedProfile dto.UserResponse
	var err error

	if updatePayload.ProfilePic != nil {
		// Handle avatar upload
		updatedProfile, err = h.userManagementService.AddAvatar(c.Request.Context(), schemaName, id, updatePayload.ProfilePic)
		if err != nil {
			response.CheckAndSendError(c, err)
			return
		}
		// Update other profile fields if provided
		updatePayload.ProfilePic = nil
		updateFields := updatePayload.Map()
		if len(updateFields) > 0 {
			updatedProfile, err = h.userManagementService.UpdateUserProfile(c.Request.Context(), schemaName, id, updatePayload)
			if err != nil {
				response.CheckAndSendError(c, err)
				return
			}
		}
	} else {
		// Update profile fields only
		updatedProfile, err = h.userManagementService.UpdateUserProfile(c.Request.Context(), schemaName, id, updatePayload)
		if err != nil {
			response.CheckAndSendError(c, err)
			return
		}
	}

	response.SendSuccess(c, responseConst.UserSuccess.UserUpdated, updatedProfile)
}

func (h *UserHandler) AddAvatar(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	file, err := c.FormFile("file")
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
// Supports optional query parameter: scope_id to filter by specific scope (workspace or base)
func (h *UserHandler) GetUserRolesAndAccess(c *gin.Context) {
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userID := c.Param("id")
	if userID == "" {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	// Get optional scope_id query parameter
	scopeID := c.Query("scope_id")
	var scopeIDPtr *string
	if scopeID != "" {
		scopeIDPtr = &scopeID
	}

	rolesAndAccess, err := h.userManagementService.GetUserRolesAndAccess(c.Request.Context(), schemaName, userID, scopeIDPtr)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.UserSuccess.UserFetched, rolesAndAccess)
}
