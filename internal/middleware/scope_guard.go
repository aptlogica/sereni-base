// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package middleware

import (
	"context"

	appConstant "github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/response"
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"
	"github.com/gin-gonic/gin"
)

// BaseWorkspaceResolver returns the ID of the workspace that owns a base.
type BaseWorkspaceResolver func(ctx context.Context, schema, baseID string) (string, error)

// NewScopeAccessGuard enforces object-level access on routes whose :id is a workspace or base.
// Unlike NewPermissionGuard/NewRoleGuard, which pass if the user holds the permission in any
// scope, this requires a membership on the specific object:
//   - system-scope members (owner, co-owner) always pass
//   - workspace members pass for that workspace and its bases
//   - base members pass for that base and (read-only context) its parent workspace
//
// resolveBaseWorkspace is required when scopeType is base and ignored otherwise.
func NewScopeAccessGuard(scopeType string, accessMemberSvc interfaces.AccessMemberService, resolveBaseWorkspace BaseWorkspaceResolver) gin.HandlerFunc {
	deny := func(c *gin.Context) {
		response.SendError(c, responseConst.Error.UnauthorizedAccess)
		c.Abort()
	}

	return func(c *gin.Context) {
		userInfo, err := ExtractUserInfo(c)
		if err != nil {
			deny(c)
			return
		}
		id := c.Param("id")
		if id == "" {
			deny(c)
			return
		}
		ctx := c.Request.Context()

		members, err := accessMemberSvc.GetUserAccessMembers(ctx, userInfo.Schema, userInfo.UserID)
		if err != nil {
			deny(c)
			return
		}

		// Resolve the owning workspace lazily: system members never need it.
		workspaceID := id
		resolved := scopeType != appConstant.ScopeLevels.Base

		for _, m := range members {
			var scopeID, memberWorkspaceID string
			if m.ScopeID != nil {
				scopeID = *m.ScopeID
			}
			if m.WorkspaceID != nil {
				memberWorkspaceID = *m.WorkspaceID
			}

			switch m.ScopeType {
			case appConstant.ScopeLevels.System:
				c.Next()
				return
			case appConstant.ScopeLevels.Base:
				if (scopeType == appConstant.ScopeLevels.Base && scopeID == id) ||
					(scopeType == appConstant.ScopeLevels.Workspace && memberWorkspaceID == id) {
					c.Next()
					return
				}
			case appConstant.ScopeLevels.Workspace:
				if !resolved {
					if resolveBaseWorkspace == nil {
						continue
					}
					if workspaceID, err = resolveBaseWorkspace(ctx, userInfo.Schema, id); err != nil {
						deny(c)
						return
					}
					resolved = true
				}
				if scopeID == workspaceID {
					c.Next()
					return
				}
			}
		}

		deny(c)
	}
}
