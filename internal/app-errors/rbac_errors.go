// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package app_errors

import "errors"

// RBAC (Role-Based Access Control) specific errors
var (
	// Role errors (using existing RoleNotFound and RoleAlreadyExists from app_errors.go)
	// RoleNotFound, RoleAlreadyExists already defined
	RoleDeleteFailed     = errors.New("failed to delete role")
	RoleUpdateFailed     = errors.New("failed to update role")
	InvalidRolePriority  = errors.New("invalid role priority value")
	RoleAssignmentFailed = errors.New("failed to assign role to user")
	RoleRemovalFailed    = errors.New("failed to remove role from user")

	// Resource errors
	ResourceNotFound      = errors.New("resource not found")
	ResourceAlreadyExists = errors.New("resource already exists")
	ResourceCreateFailed  = errors.New("failed to create resource")
	ResourceDeleteFailed  = errors.New("failed to delete resource")
	InvalidResourceCode   = errors.New("invalid resource code")

	// Action errors
	ActionNotFound      = errors.New("action not found")
	ActionAlreadyExists = errors.New("action already exists")
	ActionCreateFailed  = errors.New("failed to create action")
	ActionDeleteFailed  = errors.New("failed to delete action")
	InvalidActionCode   = errors.New("invalid action code")

	// Permission errors
	PermissionNotFound      = errors.New("permission not found")
	PermissionAlreadyExists = errors.New("permission already exists")
	PermissionCreateFailed  = errors.New("failed to create permission")
	PermissionDeleteFailed  = errors.New("failed to delete permission")
	InvalidPermissionCombo  = errors.New("invalid resource-action combination")

	// Role-Permission errors
	RolePermissionNotFound     = errors.New("role permission mapping not found")
	RolePermissionExists       = errors.New("role permission mapping already exists")
	RolePermissionCreateFailed = errors.New("failed to create role permission")
	RolePermissionDeleteFailed = errors.New("failed to delete role permission")

	// Access Member errors
	AccessMemberNotFound      = errors.New("access member record not found")
	AccessMemberAlreadyExists = errors.New("user already has this role in the scope")
	AccessMemberCreateFailed  = errors.New("failed to assign role to user")
	AccessMemberDeleteFailed  = errors.New("failed to remove role from user")
	InvalidAccessScope        = errors.New("invalid access scope type")
	MissingScopeID            = errors.New("scope ID required for workspace or base scope")
	UserNotInScope            = errors.New("user does not have access to this scope")

	// Permission check errors
	PermissionDenied       = errors.New("user does not have permission to perform this action")
	AccessDenied           = errors.New("access denied")
	InsufficientPrivileges = errors.New("insufficient privileges for this operation")

	// Bulk operation errors
	BulkAssignmentFailed = errors.New("failed to assign roles to one or more users")
	BulkRemovalFailed    = errors.New("failed to remove roles from one or more users")
	EmptyUserList        = errors.New("user list cannot be empty for bulk operations")

	// Scope errors
	InvalidScopeType = errors.New("invalid scope type. Must be 'system', 'workspace', or 'base'")
	ScopeNotFound    = errors.New("scope not found")
)
