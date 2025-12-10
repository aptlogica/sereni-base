package middleware

import (
	"fmt"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/response"
	responseConst "serenibase/internal/utils/response/constants"
	"strings"

	appConstant "serenibase/internal/constant"

	"github.com/gin-gonic/gin"
)

// ScopeLevel constants
const (
	ScopeWorkspace = "workspace"
	ScopeBase      = "base"
)

func ScopeHeaderMiddleware(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("ScopeHeaderMiddleware-------------------")
		c.Request.Header.Set("Scope", scope)
		c.Next()
	}
}

func WorkspaceAndBaseAccessValidationMiddleware(workspaceMemberService interfaces.WorkspaceMemberService, allowedAccess []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, hasRole := c.Get("roles")
		if !hasRole {
			response.SendError(c, responseConst.Error.UnauthorizedAccess)
			c.Abort()
			return
		}

		roleStr, _ := role.(string)

		if roleStr == appConstant.RoleNames.Admin {
			c.Next()
			return
		}

		if roleStr == appConstant.RoleNames.User && len(allowedAccess) == 0 {
			response.SendError(c, responseConst.Error.UnauthorizedAccess)
			c.Abort()
			return
		}

		userId, hasUser := c.Get("user_id")
		schema, hasSchema := c.Get("schema")
		if !hasUser || !hasSchema {
			response.SendError(c, responseConst.Error.UnauthorizedAccess)
			c.Abort()
			return
		}

		workspaceID := c.GetHeader("workspace")
		if workspaceID == "" {
			response.SendError(c, responseConst.Error.UnauthorizedAccess)
			c.Abort()
			return
		}

		strSchema, _ := schema.(string)
		strUserId, _ := userId.(string)

		workspaceMemberData, err := workspaceMemberService.GetWorkspaceMemberByUserAndWorkspace(
			c.Request.Context(),
			strSchema,
			strUserId,
			workspaceID,
		)

		if err != nil {
			response.SendError(c, responseConst.Error.UnauthorizedAccess)
			c.Abort()
			return
		}

		// Check allowed access
		accessAllowed := false
		for _, a := range allowedAccess {
			if a == workspaceMemberData.AccessLevel {
				accessAllowed = true
				break
			}
		}
		if !accessAllowed {
			response.SendError(c, responseConst.Error.UnauthorizedAccess)
			c.Abort()
			return
		}

		c.Set("workspaceMemberData", workspaceMemberData)

		scope, hasScope := c.Get("scope")
		_ = hasScope

		if scope == ScopeBase {
			baseID := c.GetHeader("base")
			if baseID == "" {
				response.SendError(c, responseConst.Error.UnauthorizedAccess)
				c.Abort()
				return
			}

			if workspaceMemberData.BasesIds != "*" {
				baseIDs := strings.Split(workspaceMemberData.BasesIds, ",")
				baseAllowed := false
				for _, id := range baseIDs {
					if strings.TrimSpace(id) == baseID {
						baseAllowed = true
						break
					}
				}
				if !baseAllowed {
					response.SendError(c, responseConst.Error.UnauthorizedAccess)
					c.Abort()
					return
				}
			}
		}

		c.Next()
	}
}
