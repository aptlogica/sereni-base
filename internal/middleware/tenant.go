package middleware

import (
	"fmt"
	app_errors "serenibase/internal/app-errors"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/response"
	responseConst "serenibase/internal/utils/response/constants"

	"github.com/gin-gonic/gin"
)

// TenantSchemaMiddleware checks for the presence of the "schema" header and aborts if missing.
func TenantSchemaMiddleware(tenantManagement interfaces.TenantManagementService) gin.HandlerFunc {
	return func(c *gin.Context) {
		schemaName := c.GetHeader("schema")
		if schemaName == "" {
			response.SendError(c, responseConst.AuthError.SchemaRequired)
			c.Abort()
			return
		}
		tenant, err := tenantManagement.GetTenant(c.Request.Context(), schemaName)
		if err != nil {
			if err == app_errors.TenantNotFound {
				response.SendError(c, responseConst.AuthError.InvalidSchema)
			} else {
				response.CheckAndSendError(c, err)
			}
			c.Abort()
			return
		}
		fmt.Println("tenant.Schema", tenant.Schema)
		c.Set("schema", tenant.Schema)
		c.Next()
	}
}
