package handlers

import (
	"fmt"
	"serenibase/internal/dto"
	"serenibase/internal/handlers/validators"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type OrganizationHandler struct {
	organizationService interfaces.OrganizationService
}

func NewOrganizationHandler(organizationService interfaces.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{organizationService: organizationService}
}

func (h *OrganizationHandler) CreateOrganization(c *gin.Context) {
	var req dto.CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.OrganizationCreationValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)

	req.Settings = map[string]interface{}{"created_by_user": userId}

	organization, err := h.organizationService.CreateOrganization(c.Request.Context(), schemaName, req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}
	response.SendSuccess(c, "Organization created successfully", organization)
}

func (h *OrganizationHandler) GetOrganizationByID(c *gin.Context) {
	id := c.Param("id")

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	organization, err := h.organizationService.GetOrganizationByID(c.Request.Context(), schemaName, id)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "Organization retrieved successfully", organization)
}

func (h *OrganizationHandler) GetAllOrganizations(c *gin.Context) {
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)
	fmt.Printf("DEBUG: Fetching organization in schema: %s\n", schemaName)
	organization, err := h.organizationService.GetOrganization(c.Request.Context(), schemaName)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "Organization retrieved successfully", organization)
}

func (h *OrganizationHandler) UpdateOrganization(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.OrganizationUpdateValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	organization, err := h.organizationService.UpdateOrganization(c.Request.Context(), schemaName, id, req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "Organization updated successfully", organization)
}

func (h *OrganizationHandler) DeleteOrganization(c *gin.Context) {
	id := c.Param("id")

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	err := h.organizationService.DeleteOrganization(c.Request.Context(), schemaName, id)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "Organization deleted successfully", nil)
}

func (h *OrganizationHandler) GetOrganizationByEmail(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		response.SendError(c, "Email parameter is required")
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	organization, err := h.organizationService.GetOrganizationByEmail(c.Request.Context(), schemaName, email)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "Organization retrieved successfully", organization)
}
