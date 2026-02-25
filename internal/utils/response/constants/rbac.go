package constants

type RBACResponseCode struct {
	// Role errors
	RoleNotFound         ResponseCode
	RoleAlreadyExists    ResponseCode
	RoleDeleteFailed     ResponseCode
	RoleUpdateFailed     ResponseCode
	InvalidRolePriority  ResponseCode
	RoleAssignmentFailed ResponseCode
	RoleRemovalFailed    ResponseCode

	// Resource errors
	ResourceNotFound      ResponseCode
	ResourceAlreadyExists ResponseCode
	ResourceCreateFailed  ResponseCode
	ResourceDeleteFailed  ResponseCode
	InvalidResourceCode   ResponseCode

	// Action errors
	ActionNotFound      ResponseCode
	ActionAlreadyExists ResponseCode
	ActionCreateFailed  ResponseCode
	ActionDeleteFailed  ResponseCode
	InvalidActionCode   ResponseCode

	// Permission errors
	PermissionNotFound      ResponseCode
	PermissionAlreadyExists ResponseCode
	PermissionCreateFailed  ResponseCode
	PermissionDeleteFailed  ResponseCode
	InvalidPermissionCombo  ResponseCode

	// Role-Permission errors
	RolePermissionNotFound     ResponseCode
	RolePermissionExists       ResponseCode
	RolePermissionCreateFailed ResponseCode
	RolePermissionDeleteFailed ResponseCode

	// Access Member errors
	AccessMemberNotFound      ResponseCode
	AccessMemberAlreadyExists ResponseCode
	AccessMemberCreateFailed  ResponseCode
	AccessMemberDeleteFailed  ResponseCode
	InvalidAccessScope        ResponseCode
	MissingScopeID            ResponseCode
	UserNotInScope            ResponseCode

	// Permission check errors
	PermissionDenied       ResponseCode
	AccessDenied           ResponseCode
	InsufficientPrivileges ResponseCode

	// Bulk operation errors
	BulkAssignmentFailed ResponseCode
	BulkRemovalFailed    ResponseCode
	EmptyUserList        ResponseCode

	// Scope errors
	InvalidScopeType ResponseCode
	ScopeNotFound    ResponseCode
}

var RBACError = RBACResponseCode{
	// Role errors
	RoleNotFound:         "RBAC_ERR_7001",
	RoleAlreadyExists:    "RBAC_ERR_7002",
	RoleDeleteFailed:     "RBAC_ERR_7003",
	RoleUpdateFailed:     "RBAC_ERR_7004",
	InvalidRolePriority:  "RBAC_ERR_7005",
	RoleAssignmentFailed: "RBAC_ERR_7006",
	RoleRemovalFailed:    "RBAC_ERR_7007",

	// Resource errors
	ResourceNotFound:      "RBAC_ERR_7008",
	ResourceAlreadyExists: "RBAC_ERR_7009",
	ResourceCreateFailed:  "RBAC_ERR_7010",
	ResourceDeleteFailed:  "RBAC_ERR_7011",
	InvalidResourceCode:   "RBAC_ERR_7012",

	// Action errors
	ActionNotFound:      "RBAC_ERR_7013",
	ActionAlreadyExists: "RBAC_ERR_7014",
	ActionCreateFailed:  "RBAC_ERR_7015",
	ActionDeleteFailed:  "RBAC_ERR_7016",
	InvalidActionCode:   "RBAC_ERR_7017",

	// Permission errors
	PermissionNotFound:      "RBAC_ERR_7018",
	PermissionAlreadyExists: "RBAC_ERR_7019",
	PermissionCreateFailed:  "RBAC_ERR_7020",
	PermissionDeleteFailed:  "RBAC_ERR_7021",
	InvalidPermissionCombo:  "RBAC_ERR_7022",

	// Role-Permission errors
	RolePermissionNotFound:     "RBAC_ERR_7023",
	RolePermissionExists:       "RBAC_ERR_7024",
	RolePermissionCreateFailed: "RBAC_ERR_7025",
	RolePermissionDeleteFailed: "RBAC_ERR_7026",

	// Access Member errors
	AccessMemberNotFound:      "RBAC_ERR_7027",
	AccessMemberAlreadyExists: "RBAC_ERR_7028",
	AccessMemberCreateFailed:  "RBAC_ERR_7029",
	AccessMemberDeleteFailed:  "RBAC_ERR_7030",
	InvalidAccessScope:        "RBAC_ERR_7031",
	MissingScopeID:            "RBAC_ERR_7032",
	UserNotInScope:            "RBAC_ERR_7033",

	// Permission check errors
	PermissionDenied:       "RBAC_ERR_7034",
	AccessDenied:           "RBAC_ERR_7035",
	InsufficientPrivileges: "RBAC_ERR_7036",

	// Bulk operation errors
	BulkAssignmentFailed: "RBAC_ERR_7037",
	BulkRemovalFailed:    "RBAC_ERR_7038",
	EmptyUserList:        "RBAC_ERR_7039",

	// Scope errors
	InvalidScopeType: "RBAC_ERR_7040",
	ScopeNotFound:    "RBAC_ERR_7041",
}

// RBAC error codes mapping - Added to existing ErrorCodes map in constants.go
var RBACErrorCodeDetails = map[ResponseCode]MetaResponse{
	// Role errors
	RBACError.RoleNotFound: {
		HTTPStatus:  404,
		Message:     "role not found",
		Description: "The requested role does not exist",
	},
	RBACError.RoleAlreadyExists: {
		HTTPStatus:  409,
		Message:     "role already exists",
		Description: "A role with this name already exists in the system",
	},
	RBACError.RoleDeleteFailed: {
		HTTPStatus:  500,
		Message:     "failed to delete role",
		Description: "An error occurred while deleting the role",
	},
	RBACError.RoleUpdateFailed: {
		HTTPStatus:  500,
		Message:     "failed to update role",
		Description: "An error occurred while updating the role",
	},
	RBACError.InvalidRolePriority: {
		HTTPStatus:  400,
		Message:     "invalid role priority value",
		Description: "Role priority must be a valid number",
	},
	RBACError.RoleAssignmentFailed: {
		HTTPStatus:  400,
		Message:     "failed to assign role to user",
		Description: "An error occurred while assigning the role to the user",
	},
	RBACError.RoleRemovalFailed: {
		HTTPStatus:  400,
		Message:     "failed to remove role from user",
		Description: "An error occurred while removing the role from the user",
	},

	// Resource errors
	RBACError.ResourceNotFound: {
		HTTPStatus:  404,
		Message:     "resource not found",
		Description: "The requested resource does not exist",
	},
	RBACError.ResourceAlreadyExists: {
		HTTPStatus:  409,
		Message:     "resource already exists",
		Description: "A resource with this code already exists",
	},
	RBACError.ResourceCreateFailed: {
		HTTPStatus:  500,
		Message:     "failed to create resource",
		Description: "An error occurred while creating the resource",
	},
	RBACError.ResourceDeleteFailed: {
		HTTPStatus:  500,
		Message:     "failed to delete resource",
		Description: "An error occurred while deleting the resource",
	},
	RBACError.InvalidResourceCode: {
		HTTPStatus:  400,
		Message:     "invalid resource code",
		Description: "Resource code must be alphanumeric and unique",
	},

	// Action errors
	RBACError.ActionNotFound: {
		HTTPStatus:  404,
		Message:     "action not found",
		Description: "The requested action does not exist",
	},
	RBACError.ActionAlreadyExists: {
		HTTPStatus:  409,
		Message:     "action already exists",
		Description: "An action with this code already exists",
	},
	RBACError.ActionCreateFailed: {
		HTTPStatus:  500,
		Message:     "failed to create action",
		Description: "An error occurred while creating the action",
	},
	RBACError.ActionDeleteFailed: {
		HTTPStatus:  500,
		Message:     "failed to delete action",
		Description: "An error occurred while deleting the action",
	},
	RBACError.InvalidActionCode: {
		HTTPStatus:  400,
		Message:     "invalid action code",
		Description: "Action code must be alphanumeric and unique",
	},

	// Permission errors
	RBACError.PermissionNotFound: {
		HTTPStatus:  404,
		Message:     "permission not found",
		Description: "The requested permission does not exist",
	},
	RBACError.PermissionAlreadyExists: {
		HTTPStatus:  409,
		Message:     "permission already exists",
		Description: "This resource-action combination already exists",
	},
	RBACError.PermissionCreateFailed: {
		HTTPStatus:  500,
		Message:     "failed to create permission",
		Description: "An error occurred while creating the permission",
	},
	RBACError.PermissionDeleteFailed: {
		HTTPStatus:  500,
		Message:     "failed to delete permission",
		Description: "An error occurred while deleting the permission",
	},
	RBACError.InvalidPermissionCombo: {
		HTTPStatus:  400,
		Message:     "invalid resource-action combination",
		Description: "The provided resource and action combination is invalid",
	},

	// Role-Permission errors
	RBACError.RolePermissionNotFound: {
		HTTPStatus:  404,
		Message:     "role permission mapping not found",
		Description: "The role permission mapping does not exist",
	},
	RBACError.RolePermissionExists: {
		HTTPStatus:  409,
		Message:     "role permission mapping already exists",
		Description: "This role already has this permission",
	},
	RBACError.RolePermissionCreateFailed: {
		HTTPStatus:  500,
		Message:     "failed to create role permission",
		Description: "An error occurred while assigning permission to role",
	},
	RBACError.RolePermissionDeleteFailed: {
		HTTPStatus:  500,
		Message:     "failed to delete role permission",
		Description: "An error occurred while removing permission from role",
	},

	// Access Member errors
	RBACError.AccessMemberNotFound: {
		HTTPStatus:  404,
		Message:     "access member record not found",
		Description: "The user does not have a role in the specified scope",
	},
	RBACError.AccessMemberAlreadyExists: {
		HTTPStatus:  409,
		Message:     "user already has this role in the scope",
		Description: "The user already has this role assignment",
	},
	RBACError.AccessMemberCreateFailed: {
		HTTPStatus:  400,
		Message:     "failed to assign role to user",
		Description: "An error occurred while assigning the role to the user",
	},
	RBACError.AccessMemberDeleteFailed: {
		HTTPStatus:  400,
		Message:     "failed to remove role from user",
		Description: "An error occurred while removing the role from the user",
	},
	RBACError.InvalidAccessScope: {
		HTTPStatus:  400,
		Message:     "invalid access scope",
		Description: "The access scope type is invalid",
	},
	RBACError.MissingScopeID: {
		HTTPStatus:  400,
		Message:     "scope ID is required for workspace or base scope",
		Description: "Scope ID must be provided for workspace and base level scopes",
	},
	RBACError.UserNotInScope: {
		HTTPStatus:  403,
		Message:     "user does not have access to this scope",
		Description: "The user does not have access to the specified scope",
	},

	// Permission check errors
	RBACError.PermissionDenied: {
		HTTPStatus:  403,
		Message:     "user does not have permission to perform this action",
		Description: "The user lacks the required permission for this operation",
	},
	RBACError.AccessDenied: {
		HTTPStatus:  403,
		Message:     "access denied",
		Description: "You do not have access to this resource",
	},
	RBACError.InsufficientPrivileges: {
		HTTPStatus:  403,
		Message:     "insufficient privileges for this operation",
		Description: "Your current role does not have sufficient privileges",
	},

	// Bulk operation errors
	RBACError.BulkAssignmentFailed: {
		HTTPStatus:  400,
		Message:     "failed to assign roles to one or more users",
		Description: "One or more role assignments failed during the bulk operation",
	},
	RBACError.BulkRemovalFailed: {
		HTTPStatus:  400,
		Message:     "failed to remove roles from one or more users",
		Description: "One or more role removals failed during the bulk operation",
	},
	RBACError.EmptyUserList: {
		HTTPStatus:  400,
		Message:     "user list cannot be empty for bulk operations",
		Description: "At least one user must be specified for bulk operations",
	},

	// Scope errors
	RBACError.InvalidScopeType: {
		HTTPStatus:  400,
		Message:     "invalid scope type. Must be 'system', 'workspace', or 'base'",
		Description: "Valid scope types are: system, workspace, base",
	},
	RBACError.ScopeNotFound: {
		HTTPStatus:  404,
		Message:     "scope not found",
		Description: "The specified scope does not exist",
	},
}

// RBAC codes are merged into ErrorCodes in constants.go to keep a single merge point.
