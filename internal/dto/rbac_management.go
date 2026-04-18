// Copyright (c) 2026 Aptlogica Technologies Private Limited
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

import (
	"github.com/google/uuid"
)

// RBACSystemStatus provides overall RBAC system status
type RBACSystemStatus struct {
	Initialized          bool   `json:"initialized"`
	TotalRoles           int64  `json:"total_roles"`
	TotalResources       int64  `json:"total_resources"`
	TotalActions         int64  `json:"total_actions"`
	TotalPermissions     int64  `json:"total_permissions"`
	TotalRoleAssignments int64  `json:"total_role_assignments"`
	DefaultRolesCreated  bool   `json:"default_roles_created"`
	Status               string `json:"status"` // healthy, degraded, error
}

// RBACAnalytics provides comprehensive RBAC analytics
type RBACAnalytics struct {
	SystemStatus         RBACSystemStatus     `json:"system_status"`
	RoleDistribution     map[string]int64     `json:"role_distribution"`     // role_name: count
	ResourceDistribution map[string]int64     `json:"resource_distribution"` // resource_code: permission_count
	TopRoles             []RoleUsageStats     `json:"top_roles"`
	TopResources         []ResourceUsageStats `json:"top_resources"`
	UnusedPermissions    []uuid.UUID          `json:"unused_permissions"`
}

// RoleUsageStats provides role usage statistics
type RoleUsageStats struct {
	RoleID          uuid.UUID `json:"role_id"`
	RoleName        string    `json:"role_name"`
	ScopeLevel      string    `json:"scope_level"`
	UserCount       int64     `json:"user_count"`       // number of users with this role
	PermissionCount int64     `json:"permission_count"` // number of permissions assigned
	LastAssigned    *string   `json:"last_assigned,omitempty"`
}

// PermissionUsageStats provides permission usage statistics
type PermissionUsageStats struct {
	PermissionID uuid.UUID `json:"permission_id"`
	ResourceCode string    `json:"resource_code"`
	ActionCode   string    `json:"action_code"`
	RoleCount    int64     `json:"role_count"`  // number of roles with this permission
	IsOrphaned   bool      `json:"is_orphaned"` // not assigned to any role
}

// ResourceUsageStats provides resource usage statistics
type ResourceUsageStats struct {
	ResourceID      uuid.UUID `json:"resource_id"`
	ResourceCode    string    `json:"resource_code"`
	PermissionCount int64     `json:"permission_count"` // number of permissions for this resource
	RoleCount       int64     `json:"role_count"`       // number of roles with access
}

// ResourceAccessMatrix provides a matrix view of resource-action-role mappings
type ResourceAccessMatrix struct {
	ResourceCode string            `json:"resource_code"`
	ResourceID   uuid.UUID         `json:"resource_id"`
	Actions      []ActionAccessMap `json:"actions"`
}

// ActionAccessMap maps actions to roles
type ActionAccessMap struct {
	ActionCode string      `json:"action_code"`
	ActionID   uuid.UUID   `json:"action_id"`
	Roles      []string    `json:"roles"` // role names with this permission
	RoleIDs    []uuid.UUID `json:"role_ids"`
}

// RoleValidationResult provides validation results for a role
type RoleValidationResult struct {
	RoleID          uuid.UUID `json:"role_id"`
	RoleName        string    `json:"role_name"`
	IsValid         bool      `json:"is_valid"`
	HasPermissions  bool      `json:"has_permissions"`
	PermissionCount int64     `json:"permission_count"`
	HasUsers        bool      `json:"has_users"`
	UserCount       int64     `json:"user_count"`
	Issues          []string  `json:"issues,omitempty"`
	Warnings        []string  `json:"warnings,omitempty"`
}

// UserAccessAudit provides comprehensive audit of user's access
type UserAccessAudit struct {
	UserID            string                    `json:"user_id"`
	TotalRoles        int                       `json:"total_roles"`
	SystemRoles       []AccessRoleDTO           `json:"system_roles"`
	WorkspaceRoles    []WorkspaceRoleAssignment `json:"workspace_roles"`
	BaseRoles         []BaseRoleAssignment      `json:"base_roles"`
	TotalPermissions  int                       `json:"total_permissions"`
	UniquePermissions []PermissionWithDetails   `json:"unique_permissions"`
	HighestPriority   int                       `json:"highest_priority"`
	AccessSummary     map[string][]string       `json:"access_summary"` // resource: [actions]
}

// WorkspaceRoleAssignment represents workspace-level role assignment
type WorkspaceRoleAssignment struct {
	WorkspaceID   string        `json:"workspace_id"`
	WorkspaceName string        `json:"workspace_name,omitempty"`
	Role          AccessRoleDTO `json:"role"`
	AssignedAt    string        `json:"assigned_at"`
}

// BaseRoleAssignment represents base-level role assignment
type BaseRoleAssignment struct {
	BaseID      string        `json:"base_id"`
	BaseName    string        `json:"base_name,omitempty"`
	WorkspaceID string        `json:"workspace_id,omitempty"`
	Role        AccessRoleDTO `json:"role"`
	AssignedAt  string        `json:"assigned_at"`
}
