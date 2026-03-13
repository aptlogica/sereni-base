// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package dto

// BaseRoleAccess represents a base with access level information
type BaseRoleAccess struct {
	BaseId   string `json:"base_id"`
	BaseName string `json:"base_name"`
	Access   string `json:"access"`
}

// UserRolesAccessResponse represents user's roles and access across workspaces and bases
type UserRolesAccessResponse struct {
	WorkspaceId   string           `json:"workspace_id"`
	WorkspaceName string           `json:"workspace_name"`
	Access        string           `json:"access"`
	Bases         []BaseRoleAccess `json:"bases"`
}

// UserRolesAccessList is a slice of UserRolesAccessResponse
type UserRolesAccessList []UserRolesAccessResponse
