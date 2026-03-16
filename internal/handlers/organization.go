// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package handlers

import (
	"fmt"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/handlers/validators"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

const (
	OrganizationRetrievedMessage = "Organization retrieved successfully"
)

type OrganizationHandler struct {
	organizationService interfaces.OrganizationService
}

func NewOrganizationHandler(organizationService interfaces.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{organizationService: organizationService}
}

// @Summary      Create an organization
// @Description  Persists a new organization record with contacts and metadata for future workspaces and RBAC.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.CreateOrganizationRequest  true  "Organization creation payload"
// @Success      201      {object}  dto.OrganizationResponse       "Organization created"
// @Failure      400      {object}  models.ErrorResponse            "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse            "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse            "Forbidden"
// @Failure      409      {object}  models.ErrorResponse            "Conflict — organization already exists"
// @Failure      500      {object}  models.ErrorResponse            "Internal Server Error"
// @Security     BearerAuth
// @Router       /organization [post]
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

// @Summary      Get organization by ID
// @Description  Fetches the organization record identified by the path parameter.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string                  true  "Organization ID"
// @Success      200  {object}  dto.OrganizationResponse  "Organization returned"
// @Failure      400  {object}  models.ErrorResponse      "Bad Request — invalid ID"
// @Failure      401  {object}  models.ErrorResponse      "Unauthorized"
// @Failure      403  {object}  models.ErrorResponse      "Forbidden"
// @Failure      404  {object}  models.ErrorResponse      "Not Found — organization missing"
// @Failure      500  {object}  models.ErrorResponse      "Internal Server Error"
// @Security     BearerAuth
// @Router       /organization/{id} [get]
func (h *OrganizationHandler) GetOrganizationByID(c *gin.Context) {
	id := c.Param("id")

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	organization, err := h.organizationService.GetOrganizationByID(c.Request.Context(), schemaName, id)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, OrganizationRetrievedMessage, organization)
}

// @Summary      List organizations
// @Description  Retrieves all organizations within the tenant schema for administrative dashboards.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Success      200  {array}   dto.OrganizationResponse  "Organizations returned"
// @Failure      401  {object}  models.ErrorResponse      "Unauthorized"
// @Failure      403  {object}  models.ErrorResponse      "Forbidden"
// @Failure      500  {object}  models.ErrorResponse      "Internal Server Error"
// @Security     BearerAuth
// @Router       /organization [get]
func (h *OrganizationHandler) GetAllOrganizations(c *gin.Context) {
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)
	fmt.Printf("DEBUG: Fetching organization in schema: %s\n", schemaName)
	organization, err := h.organizationService.GetOrganization(c.Request.Context(), schemaName)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, OrganizationRetrievedMessage, organization)
}

// @Summary      Update organization
// @Description  Updates the organization profile and settings fields.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id       path      string                    true  "Organization ID"
// @Param        request  body      dto.UpdateOrganizationRequest  true  "Fields to update"
// @Success      200      {object}  dto.OrganizationResponse       "Organization updated"
// @Failure      400      {object}  models.ErrorResponse            "Bad Request — invalid payload"
// @Failure      401      {object}  models.ErrorResponse            "Unauthorized"
// @Failure      403      {object}  models.ErrorResponse            "Forbidden"
// @Failure      404      {object}  models.ErrorResponse            "Not Found — organization missing"
// @Failure      500      {object}  models.ErrorResponse            "Internal Server Error"
// @Security     BearerAuth
// @Router       /organization/{id} [put]
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

// @Summary      Delete organization
// @Description  Deletes the organization from the schema.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string  true  "Organization ID"
// @Success      200  {object}  models.SuccessResponse  "Organization deleted"
// @Failure      400  {object}  models.ErrorResponse    "Bad Request — invalid ID"
// @Failure      401  {object}  models.ErrorResponse    "Unauthorized"
// @Failure      403  {object}  models.ErrorResponse    "Forbidden"
// @Failure      404  {object}  models.ErrorResponse    "Not Found — organization missing"
// @Failure      500  {object}  models.ErrorResponse    "Internal Server Error"
// @Security     BearerAuth
// @Router       /organization/{id} [delete]
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

// @Summary      Get organization by email
// @Description  Looks up an organization using the provided email address.
// @Tags         Admin
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        email  query     string  true  "Organization contact email"
// @Success      200    {object}  dto.OrganizationResponse  "Organization returned"
// @Failure      400    {object}  models.ErrorResponse      "Bad Request — email missing"
// @Failure      401    {object}  models.ErrorResponse      "Unauthorized"
// @Failure      403    {object}  models.ErrorResponse      "Forbidden"
// @Failure      404    {object}  models.ErrorResponse      "Not Found — organization missing"
// @Failure      500    {object}  models.ErrorResponse      "Internal Server Error"
// @Security     BearerAuth
// @Router       /organization/by-email [get]
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

	response.SendSuccess(c, OrganizationRetrievedMessage, organization)
}
