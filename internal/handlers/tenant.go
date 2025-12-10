package handlers

import (
	"serenibase/internal/dto"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/response"
	responseConst "serenibase/internal/utils/response/constants"

	"github.com/gin-gonic/gin"
)

type TenantHandler struct {
	tenantManagementService interfaces.TenantManagementService
}

func NewTenantHandler(tenantManagementService interfaces.TenantManagementService) *TenantHandler {
	return &TenantHandler{tenantManagementService: tenantManagementService}
}

func (h *TenantHandler) GetTenantInfo(c *gin.Context) {
	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	tenant, err := h.tenantManagementService.GetTenantInfoBySchema(c.Request.Context(), schemaName)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TenantSuccess.TenantFetched, tenant)

}

func (h *TenantHandler) UpdateTenantInfo(c *gin.Context) {
	var req dto.UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	updatedTenant, err := h.tenantManagementService.UpdateTenantBySchema(c.Request.Context(), schemaName, req)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.TenantSuccess.TenantUpdated, updatedTenant)
}
