// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package handlers

import (
	"encoding/json"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/response"
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"

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

// bindUpdateProfileFields binds form fields to the update payload
func (h *UserHandler) bindUpdateProfileFields(c *gin.Context) dto.UpdateUserProfileRequest {
	var updatePayload dto.UpdateUserProfileRequest

	// Manually bind form fields
	if firstName := c.PostForm("first_name"); firstName != "" {
		updatePayload.FirstName = &firstName
	}
	if lastName := c.PostForm("last_name"); lastName != "" {
		updatePayload.LastName = &lastName
	}
	if displayName := c.PostForm("display_name"); displayName != "" {
		updatePayload.DisplayName = &displayName
	}
	if activityData := c.PostForm("activity_data"); activityData != "" {
		// Parse activity_data JSON string
		var activityMap map[string]interface{}
		if err := json.Unmarshal([]byte(activityData), &activityMap); err == nil {
			updatePayload.ActivityData = &activityMap
		}
	}
	if dob := c.PostForm("dob"); dob != "" {
		updatePayload.DateOfBirth = &dob
	}
	if country := c.PostForm("country"); country != "" {
		updatePayload.Country = &country
	}
	if timezone := c.PostForm("timezone"); timezone != "" {
		updatePayload.Timezone = &timezone
	}
	if locale := c.PostForm("locale"); locale != "" {
		updatePayload.Locale = &locale
	}

	return updatePayload
}

// handleAvatarUpdate handles avatar upload and profile update logic
func (h *UserHandler) handleAvatarUpdate(c *gin.Context, schemaName, id string, updatePayload dto.UpdateUserProfileRequest) (dto.UserResponse, error) {
	// Handle avatar upload
	updatedProfile, err := h.userManagementService.AddAvatar(c.Request.Context(), schemaName, id, updatePayload.ProfilePic)
	if err != nil {
		return dto.UserResponse{}, err
	}

	// Update other profile fields if provided
	updatePayload.ProfilePic = nil
	updateFields := updatePayload.Map()
	if len(updateFields) > 0 {
		updatedProfile, err = h.userManagementService.UpdateUserProfile(c.Request.Context(), schemaName, id, updatePayload)
		if err != nil {
			return dto.UserResponse{}, err
		}
	}

	return updatedProfile, nil
}

// @Summary      Get user profile by ID
// @Description  Returns the detailed user profile for the supplied ID.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string          true  "User ID"
// @Success      200  {object}  dto.UserResponse  "User profile retrieved"
// @Failure      400  {object}  models.ErrorResponse  "Bad Request — invalid or missing ID"
// @Failure      401  {object}  models.ErrorResponse  "Unauthorized"
// @Failure      403  {object}  models.ErrorResponse  "Forbidden"
// @Failure      404  {object}  models.ErrorResponse  "Not Found — user missing"
// @Failure      500  {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /user/profile/{id} [get]
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

// @Summary      Update user profile
// @Description  Applies form fields and optional avatar upload to update the user's profile.
// @Tags         Users
// @Accept       multipart/form-data
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id     path      string  true   "User ID"
// @Param        avatar formData  file    false  "Avatar file to upload"
// @Param        request body      dto.UpdateUserProfileRequest  true  "Profile fields to update"
// @Success      200    {object}  dto.UserResponse               "Updated user profile"
// @Failure      400    {object}  models.ErrorResponse           "Bad Request — invalid payload"
// @Failure      401    {object}  models.ErrorResponse           "Unauthorized"
// @Failure      403    {object}  models.ErrorResponse           "Forbidden"
// @Failure      422    {object}  models.ErrorResponse           "Unprocessable Entity — invalid field"
// @Failure      500    {object}  models.ErrorResponse           "Internal Server Error"
// @Security     BearerAuth
// @Router       /user/profile/{id} [patch]
func (h *UserHandler) UpdateUserProfile(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.SendError(c, responseConst.Error.InvalidPayload)
		return
	}

	updatePayload := h.bindUpdateProfileFields(c)

	// Bind file if present
	if file, err := c.FormFile("avatar"); err == nil {
		updatePayload.ProfilePic = file
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	var updatedProfile dto.UserResponse
	var err error

	if updatePayload.ProfilePic != nil {
		updatedProfile, err = h.handleAvatarUpdate(c, schemaName, id, updatePayload)
	} else {
		// Update profile fields only
		updatedProfile, err = h.userManagementService.UpdateUserProfile(c.Request.Context(), schemaName, id, updatePayload)
	}

	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.UserSuccess.UserUpdated, updatedProfile)
}

// @Summary      Upload avatar
// @Description  Stores the provided avatar file and returns the updated user profile.
// @Tags         Users
// @Accept       multipart/form-data
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id    path      string  true  "User ID"
// @Param        file  formData  file    true  "Avatar file"
// @Success      200   {object}  dto.UserResponse  "Profile returned with avatar URL"
// @Failure      400   {object}  models.ErrorResponse  "Bad Request — missing file"
// @Failure      401   {object}  models.ErrorResponse  "Unauthorized"
// @Failure      403   {object}  models.ErrorResponse  "Forbidden"
// @Failure      500   {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /user/profile/{id}/avatar [post]
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

// @Summary      Remove avatar
// @Description  Clears the avatar for the user and returns the updated profile.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string          true  "User ID"
// @Success      200  {object}  dto.UserResponse  "Avatar removed"
// @Failure      400  {object}  models.ErrorResponse  "Bad Request — invalid user id"
// @Failure      401  {object}  models.ErrorResponse  "Unauthorized"
// @Failure      403  {object}  models.ErrorResponse  "Forbidden"
// @Failure      404  {object}  models.ErrorResponse  "Not Found — user missing"
// @Failure      500  {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /user/profile/{id}/avatar [delete]
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

// @Summary      List user workspaces
// @Description  Returns every workspace and base combination the authenticated user has membership in.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Success      200  {array}   dto.UserWorkspaceResponse  "Workspaces user participates in"
// @Failure      401  {object}  models.ErrorResponse      "Unauthorized"
// @Failure      403  {object}  models.ErrorResponse      "Forbidden"
// @Failure      500  {object}  models.ErrorResponse      "Internal Server Error"
// @Security     BearerAuth
// @Router       /user/workspaces [get]
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

// @Summary      Get user access details
// @Description  Returns expanded workspace/base access details for a user; restricted to Admin roles and allows optional workspace filtering.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        user_id      query     string  true   "Target user ID"
// @Param        workspace_id query     string  false  "Optional workspace filter"
// @Success      200          {object}  dto.UserAccessDetailsResponse  "Access details returned"
// @Failure      400          {object}  models.ErrorResponse           "Bad Request — missing user_id"
// @Failure      401          {object}  models.ErrorResponse           "Unauthorized"
// @Failure      403          {object}  models.ErrorResponse           "Forbidden — requires Admin role"
// @Failure      500          {object}  models.ErrorResponse           "Internal Server Error"
// @Security     BearerAuth
// @Router       /user/access-details [get]
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

	accessDetails, err := h.userManagementService.GetUserAccessDetails(c.Request.Context(), schemaName, userId, roles, workspaceId)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.UserSuccess.UserAccessDetailsFetched, accessDetails)
}

// GetUserRolesAndAccess retrieves user's roles and access information organized by workspace and base
// Supports optional query parameter: scope_id to filter by specific scope (workspace or base)
// @Summary      Get user roles and access
// @Description  Returns workspaces and bases with role assignments for the requested user, optionally scoped by workspace/base id.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id       path      string  true   "User ID"
// @Param        scope_id query     string  false  "Optional workspace or base scope ID"
// @Success      200      {array}   dto.UserRolesAccessResponse  "Roles and access list"
// @Failure      400      {object}  models.ErrorResponse        "Bad Request — invalid user ID"
// @Failure      401      {object}  models.ErrorResponse        "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse        "Forbidden — insufficient access"
// @Failure      500      {object}  models.ErrorResponse        "Internal Server Error"
// @Security     BearerAuth
// @Router       /user/roles-and-access/{id} [get]
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
