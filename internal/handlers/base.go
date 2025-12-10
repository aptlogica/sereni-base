package handlers

import (
	"serenibase/internal/dto"
	"serenibase/internal/handlers/validators"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type BaseHandler struct {
	baseManagementService interfaces.BaseManagementService
}

func NewBaseHandler(baseManagementService interfaces.BaseManagementService) *BaseHandler {
	return &BaseHandler{baseManagementService: baseManagementService}
}

func (h *BaseHandler) CreateBase(c *gin.Context) {
	var req dto.CreateBaseRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.BaseCreationValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)

	base, err := h.baseManagementService.CreateBase(c.Request.Context(), req, schemaName, userId)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "base created successfully", base)
}

func (h *BaseHandler) GetBaseByID(c *gin.Context) {
	id := c.Param("id")

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	base, err := h.baseManagementService.GetBaseByID(c.Request.Context(), schemaName, id)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "base retrieved successfully", base)
}


func (h *BaseHandler) UpdateBase(c *gin.Context) {
	var req dto.BaseUpdate

	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.BaseUpdateValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	id := c.Param("id")

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	userIdVal, _ := c.Get("user_id")
	userId, _ := userIdVal.(string)

	updatedBase, err := h.baseManagementService.UpdateBase(c.Request.Context(), schemaName, id, req, userId)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "base updated successfully", updatedBase)
}

func (h *BaseHandler) DeleteBase(c *gin.Context) {
	id := c.Param("id")

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	if err := h.baseManagementService.DeleteBase(c.Request.Context(), schemaName, id); err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "base deleted successfully", nil)
}

func (h *BaseHandler) GetTablesByBaseId(c *gin.Context) {
	id := c.Param("id")

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	tables, err := h.baseManagementService.GetTablesByBaseId(c.Request.Context(), schemaName, id)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, "tables retrieved successfully", tables)
}
