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
