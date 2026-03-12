// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

// This file provides backward compatibility after restructuring services into subdirectories.
// It re-exports all service constructors and types from subdirectories so existing code
// that imports "serenibase/internal/services" continues to work without changes.

import (
	asset "serenibase/internal/services/asset"
	auth "serenibase/internal/services/auth"
	base "serenibase/internal/services/base"
	core "serenibase/internal/services/core"
	rbac "serenibase/internal/services/rbac"
	table "serenibase/internal/services/table"
	workspace "serenibase/internal/services/workspace"
)

// ============================================================================
// Auth Services - Re-exports from auth subdirectory
// ============================================================================

var NewAuthManagementService = auth.NewAuthManagementService
var NewUserManagementService = auth.NewUserManagementService
var NewUserResetTokenService = auth.NewUserResetTokenService
var NewUserService = auth.NewUserService

// Auth service type exports
type AuthManagementServiceDeps = auth.AuthManagementServiceDeps
type AuthManagementProviderDeps = auth.AuthManagementProviderDeps

// ============================================================================
// RBAC Services - Re-exports from rbac subdirectory
// ============================================================================

var NewAccessMemberService = rbac.NewAccessMemberService
var NewRBACAccessRoleService = rbac.NewAccessRoleService
var NewRBACPermissionService = rbac.NewPermissionService
var NewRolePermissionService = rbac.NewRolePermissionService
var NewRBACManagementService = rbac.NewRBACManagementService

// RBAC service type exports
type RBACManagementServiceDeps = rbac.RBACManagementServiceDeps

// RBAC constants - Direct from rbac package
const AccessMembersTableFormat = rbac.AccessMembersTableFormat

// ============================================================================
// Workspace Services - Re-exports from workspace subdirectory
// ============================================================================

var NewWorkspaceService = workspace.NewWorkspaceService
var NewWorkspaceMemberService = workspace.NewWorkspaceMemberService
var NewWorkspaceManagementService = workspace.NewWorkspaceManagementService

// ============================================================================
// Base Services - Re-exports from base subdirectory
// ============================================================================

var NewBaseService = base.NewBaseService
var NewBaseManagementService = base.NewBaseManagementService

// ============================================================================
// Table Services - Re-exports from table subdirectory
// ============================================================================

var NewTableManagementService = table.NewTableManagementService
var NewColumnService = table.NewColumnService
var NewImportService = table.NewImportService
var NewModelService = table.NewModelService

// ============================================================================
// Asset Services - Re-exports from asset subdirectory
// ============================================================================

var NewAssetsService = asset.NewAssetsService
var NewAssetManagementService = asset.NewAssetManagementService

// ============================================================================
// Core Services - Re-exports from core subdirectory
// ============================================================================

var NewOrganizationService = core.NewOrganizationService
var NewCoreResourceService = core.NewResourceService
var NewCoreActionService = core.NewActionService
var NewViewService = core.NewViewService
var NewRelationshipService = core.NewRelationshipService
