// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package middleware

import (
	"context"

	appConstant "github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
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
		id := c.Param("id")
		if err != nil || id == "" {
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
		workspace := lazyWorkspace{id: id, resolved: scopeType != appConstant.ScopeLevels.Base}

		for _, m := range members {
			granted, err := memberGrantsAccess(ctx, userInfo.Schema, m, scopeType, id, resolveBaseWorkspace, &workspace)
			if err != nil {
				deny(c)
				return
			}
			if granted {
				c.Next()
				return
			}
		}

		deny(c)
	}
}

// lazyWorkspace is the workspace that owns the object guarded by NewScopeAccessGuard,
// resolved on first use when that object is a base.
type lazyWorkspace struct {
	id       string
	resolved bool
}

// memberGrantsAccess reports whether membership m grants access to the workspace or base id.
// An error means the owning workspace could not be resolved, and access must be denied.
func memberGrantsAccess(ctx context.Context, schema string, m dto.AccessMemberDTO, scopeType, id string, resolveBaseWorkspace BaseWorkspaceResolver, workspace *lazyWorkspace) (bool, error) {
	switch m.ScopeType {
	case appConstant.ScopeLevels.System:
		return true, nil
	case appConstant.ScopeLevels.Base:
		return (scopeType == appConstant.ScopeLevels.Base && strValue(m.ScopeID) == id) ||
			(scopeType == appConstant.ScopeLevels.Workspace && strValue(m.WorkspaceID) == id), nil
	case appConstant.ScopeLevels.Workspace:
		if !workspace.resolved {
			if resolveBaseWorkspace == nil {
				return false, nil
			}
			var err error
			if workspace.id, err = resolveBaseWorkspace(ctx, schema, id); err != nil {
				return false, err
			}
			workspace.resolved = true
		}
		return strValue(m.ScopeID) == workspace.id, nil
	}
	return false, nil
}

// strValue dereferences an optional string, treating nil as empty.
func strValue(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// NewWorkspaceFormPermissionGuard enforces a permission on create routes that take the target
// workspace from a form field instead of :id. The permission is only evaluated on the user's
// system-scope membership or their membership of that workspace, so a maintainer of one workspace
// cannot create content in another. Base-only membership never passes.
func NewWorkspaceFormPermissionGuard(field, resourceCode, actionCode string, accessMemberSvc interfaces.AccessMemberService) gin.HandlerFunc {
	return NewObjectPermissionGuard(resourceCode, actionCode, accessMemberSvc, ScopeLookups{}.WorkspaceForm(field))
}
