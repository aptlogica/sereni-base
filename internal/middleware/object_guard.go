// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"

	appConstant "github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/response"
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"
	"github.com/gin-gonic/gin"
)

// ScopeTarget is a workspace or base that a request acts on. A target with a BaseID can be
// reached through a membership of that base; a workspace-level target (no BaseID) only through
// a system-scope membership or a membership of the workspace.
type ScopeTarget struct {
	WorkspaceID string
	BaseID      string
}

// TargetResolver extracts the scope targets of a request. Every returned target must pass.
type TargetResolver func(c *gin.Context, schema string) ([]ScopeTarget, error)

// ScopeLookups resolve stored objects to the scope that owns them.
type ScopeLookups struct {
	BaseWorkspace BaseWorkspaceResolver
	ModelBase     func(ctx context.Context, schema, modelID string) (string, error)
	ColumnModel   func(ctx context.Context, schema, columnID string) (string, error)
	ViewModel     func(ctx context.Context, schema, viewID string) (string, error)
	AccessMember  func(ctx context.Context, schema, id string) (dto.AccessMemberDTO, error)
}

var errOutOfScope = errors.New("request references an object outside its target scope")

// NewObjectPermissionGuard binds a permission check to the objects a request targets. Unlike
// NewPermissionGuard, which passes if the user holds the permission in any scope, the permission
// is only evaluated on a membership that covers each target, so access to one workspace or base
// never grants access to another. An empty resourceCode only requires a covering membership.
func NewObjectPermissionGuard(resourceCode, actionCode string, accessMemberSvc interfaces.AccessMemberService, resolve TargetResolver) gin.HandlerFunc {
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
		targets, err := resolve(c, userInfo.Schema)
		if err != nil || len(targets) == 0 {
			deny(c)
			return
		}
		ctx := c.Request.Context()
		members, err := accessMemberSvc.GetUserAccessMembers(ctx, userInfo.Schema, userInfo.UserID)
		if err != nil {
			deny(c)
			return
		}

		checked := make(map[ScopeTarget]bool, len(targets))
		for _, t := range targets {
			if checked[t] {
				continue
			}
			if !hasScopedPermission(ctx, accessMemberSvc, userInfo, members, t, resourceCode, actionCode) {
				deny(c)
				return
			}
			checked[t] = true
		}
		c.Next()
	}
}

func hasScopedPermission(ctx context.Context, svc interfaces.AccessMemberService, userInfo *UserInfo, members []dto.AccessMemberDTO, t ScopeTarget, resourceCode, actionCode string) bool {
	for _, m := range members {
		var scopeID string
		if m.ScopeID != nil {
			scopeID = *m.ScopeID
		}
		covers := m.ScopeType == appConstant.ScopeLevels.System ||
			(m.ScopeType == appConstant.ScopeLevels.Workspace && t.WorkspaceID != "" && scopeID == t.WorkspaceID) ||
			(m.ScopeType == appConstant.ScopeLevels.Base && t.BaseID != "" && scopeID == t.BaseID)
		if !covers {
			continue
		}
		if resourceCode == "" {
			return true
		}
		allowed, err := svc.CheckUserPermission(ctx, userInfo.Schema, userInfo.UserID, m.ScopeType, &scopeID, resourceCode, actionCode)
		if err == nil && allowed {
			return true
		}
	}
	return false
}

// --- target lookups ---

func (l ScopeLookups) baseTarget(ctx context.Context, schema, baseID string) (ScopeTarget, error) {
	if baseID == "" || l.BaseWorkspace == nil {
		return ScopeTarget{}, errOutOfScope
	}
	workspaceID, err := l.BaseWorkspace(ctx, schema, baseID)
	if err != nil || workspaceID == "" {
		return ScopeTarget{}, errOutOfScope
	}
	return ScopeTarget{WorkspaceID: workspaceID, BaseID: baseID}, nil
}

func (l ScopeLookups) modelTarget(ctx context.Context, schema, modelID string) (ScopeTarget, error) {
	if modelID == "" || l.ModelBase == nil {
		return ScopeTarget{}, errOutOfScope
	}
	baseID, err := l.ModelBase(ctx, schema, modelID)
	if err != nil {
		return ScopeTarget{}, errOutOfScope
	}
	return l.baseTarget(ctx, schema, baseID)
}

func (l ScopeLookups) columnModelID(ctx context.Context, schema, columnID string) (string, error) {
	if columnID == "" || l.ColumnModel == nil {
		return "", errOutOfScope
	}
	modelID, err := l.ColumnModel(ctx, schema, columnID)
	if err != nil || modelID == "" {
		return "", errOutOfScope
	}
	return modelID, nil
}

// requireColumnsInModel rejects column IDs that belong to a different table than modelID, so a
// caller authorised on one table cannot address another table's columns through it.
func (l ScopeLookups) requireColumnsInModel(ctx context.Context, schema, modelID string, columnIDs ...string) error {
	for _, id := range columnIDs {
		owner, err := l.columnModelID(ctx, schema, id)
		if err != nil || owner != modelID {
			return errOutOfScope
		}
	}
	return nil
}

// jsonBody decodes the request body as a JSON object and restores it for the handler.
func jsonBody(c *gin.Context) (map[string]interface{}, error) {
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(raw))
	var body map[string]interface{}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	return body, nil
}

func str(m map[string]interface{}, key string) string {
	s, _ := m[key].(string)
	return s
}

// --- resolvers ---

// WorkspaceParam targets the workspace named by a path parameter.
func (l ScopeLookups) WorkspaceParam(name string) TargetResolver {
	return func(c *gin.Context, _ string) ([]ScopeTarget, error) {
		if id := c.Param(name); id != "" {
			return []ScopeTarget{{WorkspaceID: id}}, nil
		}
		return nil, errOutOfScope
	}
}

// WorkspaceForm targets the workspace named by a form field.
func (l ScopeLookups) WorkspaceForm(field string) TargetResolver {
	return func(c *gin.Context, _ string) ([]ScopeTarget, error) {
		if id := c.PostForm(field); id != "" {
			return []ScopeTarget{{WorkspaceID: id}}, nil
		}
		return nil, errOutOfScope
	}
}

// BaseParam targets the base named by a path parameter.
func (l ScopeLookups) BaseParam(name string) TargetResolver {
	return func(c *gin.Context, schema string) ([]ScopeTarget, error) {
		t, err := l.baseTarget(c.Request.Context(), schema, c.Param(name))
		return []ScopeTarget{t}, err
	}
}

// ModelParam targets the base of the table named by a path parameter.
func (l ScopeLookups) ModelParam(name string) TargetResolver {
	return func(c *gin.Context, schema string) ([]ScopeTarget, error) {
		t, err := l.modelTarget(c.Request.Context(), schema, c.Param(name))
		return []ScopeTarget{t}, err
	}
}

// ColumnParam targets the base of the column named by a path parameter.
func (l ScopeLookups) ColumnParam(name string) TargetResolver {
	return func(c *gin.Context, schema string) ([]ScopeTarget, error) {
		ctx := c.Request.Context()
		modelID, err := l.columnModelID(ctx, schema, c.Param(name))
		if err != nil {
			return nil, err
		}
		t, err := l.modelTarget(ctx, schema, modelID)
		return []ScopeTarget{t}, err
	}
}

// ViewParam targets the base of the view named by a path parameter.
func (l ScopeLookups) ViewParam(name string) TargetResolver {
	return func(c *gin.Context, schema string) ([]ScopeTarget, error) {
		ctx := c.Request.Context()
		if l.ViewModel == nil {
			return nil, errOutOfScope
		}
		modelID, err := l.ViewModel(ctx, schema, c.Param(name))
		if err != nil {
			return nil, errOutOfScope
		}
		t, err := l.modelTarget(ctx, schema, modelID)
		return []ScopeTarget{t}, err
	}
}

// ModelBody targets the base of the table in the JSON body's model_id. It also requires that
// column_id, the keys of values (row updates) and base_id, when present, belong to that table,
// and adds the linked table of a links column (meta.relation.with) as a second target.
func (l ScopeLookups) ModelBody() TargetResolver {
	return func(c *gin.Context, schema string) ([]ScopeTarget, error) {
		body, err := jsonBody(c)
		if err != nil {
			return nil, errOutOfScope
		}
		ctx := c.Request.Context()
		modelID := str(body, "model_id")
		t, err := l.modelTarget(ctx, schema, modelID)
		if err != nil {
			return nil, err
		}
		if baseID := str(body, "base_id"); baseID != "" && baseID != t.BaseID {
			return nil, errOutOfScope
		}

		if err := l.requireColumnsInModel(ctx, schema, modelID, bodyColumnIDs(body)...); err != nil {
			return nil, err
		}

		targets := []ScopeTarget{t}
		if linked := linkedModelID(body); linked != "" {
			lt, err := l.modelTarget(ctx, schema, linked)
			if err != nil {
				return nil, err
			}
			targets = append(targets, lt)
		}
		return targets, nil
	}
}

// bodyColumnIDs returns the column IDs a body addresses: column_id and the keys of values.
func bodyColumnIDs(body map[string]interface{}) []string {
	var ids []string
	if id := str(body, "column_id"); id != "" {
		ids = append(ids, id)
	}
	values, _ := body["values"].(map[string]interface{})
	for id := range values {
		ids = append(ids, id)
	}
	return ids
}

// linkedModelID returns meta.relation.with, the table a links column points to.
func linkedModelID(body map[string]interface{}) string {
	meta, _ := body["meta"].(map[string]interface{})
	rel, _ := meta["relation"].(map[string]interface{})
	return str(rel, "with")
}

// ModelForm is ModelBody for multipart requests: model_id and column_id are form fields.
func (l ScopeLookups) ModelForm() TargetResolver {
	return func(c *gin.Context, schema string) ([]ScopeTarget, error) {
		ctx := c.Request.Context()
		modelID := c.PostForm("model_id")
		t, err := l.modelTarget(ctx, schema, modelID)
		if err != nil {
			return nil, err
		}
		if id := c.PostForm("column_id"); id != "" {
			if err := l.requireColumnsInModel(ctx, schema, modelID, id); err != nil {
				return nil, err
			}
		}
		return []ScopeTarget{t}, nil
	}
}

// ColumnPairBody targets the table of two columns in the JSON body and requires they share it.
func (l ScopeLookups) ColumnPairBody(first, second string) TargetResolver {
	return func(c *gin.Context, schema string) ([]ScopeTarget, error) {
		body, err := jsonBody(c)
		if err != nil {
			return nil, errOutOfScope
		}
		ctx := c.Request.Context()
		modelID, err := l.columnModelID(ctx, schema, str(body, first))
		if err != nil {
			return nil, err
		}
		if err := l.requireColumnsInModel(ctx, schema, modelID, str(body, second)); err != nil {
			return nil, err
		}
		t, err := l.modelTarget(ctx, schema, modelID)
		return []ScopeTarget{t}, err
	}
}

// requireBaseInWorkspace resolves a base and checks it belongs to workspaceID when one is given.
func (l ScopeLookups) requireBaseInWorkspace(ctx context.Context, schema, baseID, workspaceID string) (ScopeTarget, error) {
	t, err := l.baseTarget(ctx, schema, baseID)
	if err != nil {
		return ScopeTarget{}, err
	}
	if workspaceID != "" && workspaceID != t.WorkspaceID {
		return ScopeTarget{}, errOutOfScope
	}
	return t, nil
}

// TableCreateBody targets base_id of a JSON table-create body and requires it to be in workspace_id.
func (l ScopeLookups) TableCreateBody() TargetResolver {
	return func(c *gin.Context, schema string) ([]ScopeTarget, error) {
		body, err := jsonBody(c)
		if err != nil {
			return nil, errOutOfScope
		}
		t, err := l.requireBaseInWorkspace(c.Request.Context(), schema, str(body, "base_id"), str(body, "workspace_id"))
		return []ScopeTarget{t}, err
	}
}

// TableImportForm targets base_id of a multipart import, or workspace_id when the import creates a new base.
func (l ScopeLookups) TableImportForm() TargetResolver {
	return func(c *gin.Context, schema string) ([]ScopeTarget, error) {
		baseID, workspaceID := c.PostForm("base_id"), c.PostForm("workspace_id")
		if baseID == "" {
			if workspaceID == "" {
				return nil, errOutOfScope
			}
			return []ScopeTarget{{WorkspaceID: workspaceID}}, nil
		}
		t, err := l.requireBaseInWorkspace(c.Request.Context(), schema, baseID, workspaceID)
		return []ScopeTarget{t}, err
	}
}

// MembershipBody targets every workspace a member-assignment body grants access in. Bases are
// mapped to workspace-level targets so only workspace or system administrators can grant them.
func (l ScopeLookups) MembershipBody() TargetResolver {
	return func(c *gin.Context, schema string) ([]ScopeTarget, error) {
		body, err := jsonBody(c)
		if err != nil {
			return nil, errOutOfScope
		}
		memberships, _ := body["membership"].([]interface{})
		if len(memberships) == 0 {
			return nil, errOutOfScope
		}
		ctx := c.Request.Context()
		var targets []ScopeTarget
		for _, raw := range memberships {
			m, _ := raw.(map[string]interface{})
			workspaceID := str(m, "workspace_id")
			if workspaceID == "" {
				return nil, errOutOfScope
			}
			targets = append(targets, ScopeTarget{WorkspaceID: workspaceID})
			bases, _ := m["bases"].([]interface{})
			for _, rawBase := range bases {
				b, _ := rawBase.(map[string]interface{})
				if _, err := l.requireBaseInWorkspace(ctx, schema, str(b, "base_id"), workspaceID); err != nil {
					return nil, err
				}
			}
		}
		return targets, nil
	}
}

// AccessMemberParam targets the scope of the access_members row named by a path parameter. The
// row must have scopeType (workspace or base); system-scope rows (owner, co-owner) are never
// removable this way. Base rows map to their workspace, so only workspace administrators pass.
func (l ScopeLookups) AccessMemberParam(name, scopeType string) TargetResolver {
	return func(c *gin.Context, schema string) ([]ScopeTarget, error) {
		if l.AccessMember == nil {
			return nil, errOutOfScope
		}
		ctx := c.Request.Context()
		row, err := l.AccessMember(ctx, schema, c.Param(name))
		if err != nil || row.ScopeType != scopeType || row.ScopeID == nil {
			return nil, errOutOfScope
		}
		switch scopeType {
		case appConstant.ScopeLevels.Workspace:
			return []ScopeTarget{{WorkspaceID: *row.ScopeID}}, nil
		case appConstant.ScopeLevels.Base:
			t, err := l.baseTarget(ctx, schema, *row.ScopeID)
			return []ScopeTarget{{WorkspaceID: t.WorkspaceID}}, err
		}
		return nil, errOutOfScope
	}
}

// --- list filtering ---

// ModelAccessFilterKey is the context key under which NewModelAccessFilter stores its filter.
const ModelAccessFilterKey = "modelAccessFilter"

// NewModelAccessFilter stores a func(modelID) bool in the context for list handlers that return
// objects from every table (e.g. GET /view/), so they can drop tables the caller has no
// membership on. Lookups are cached per request.
func NewModelAccessFilter(accessMemberSvc interfaces.AccessMemberService, l ScopeLookups) gin.HandlerFunc {
	return func(c *gin.Context) {
		userInfo, err := ExtractUserInfo(c)
		if err != nil {
			response.SendError(c, responseConst.Error.UnauthorizedAccess)
			c.Abort()
			return
		}
		ctx := c.Request.Context()
		members, err := accessMemberSvc.GetUserAccessMembers(ctx, userInfo.Schema, userInfo.UserID)
		if err != nil {
			members = nil
		}
		cache := map[string]bool{}
		c.Set(ModelAccessFilterKey, func(modelID string) bool {
			if allowed, ok := cache[modelID]; ok {
				return allowed
			}
			t, err := l.modelTarget(ctx, userInfo.Schema, modelID)
			allowed := err == nil && hasScopedPermission(ctx, accessMemberSvc, userInfo, members, t, "", "")
			cache[modelID] = allowed
			return allowed
		})
		c.Next()
	}
}
