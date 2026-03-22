package tests

import (
	"errors"
	"testing"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/stretchr/testify/assert"
)

// TestRBACErrors tests all RBAC related errors
func TestRBACErrors(t *testing.T) {
	// Role errors
	t.Run("RoleErrors", func(t *testing.T) {
		assert.NotNil(t, app_errors.RoleDeleteFailed)
		assert.NotNil(t, app_errors.RoleUpdateFailed)
		assert.NotNil(t, app_errors.InvalidRolePriority)
		assert.NotNil(t, app_errors.RoleAssignmentFailed)
		assert.NotNil(t, app_errors.RoleRemovalFailed)

		assert.Equal(t, "failed to delete role", app_errors.RoleDeleteFailed.Error())
		assert.Equal(t, "failed to update role", app_errors.RoleUpdateFailed.Error())
		assert.Equal(t, "invalid role priority value", app_errors.InvalidRolePriority.Error())
		assert.Equal(t, "failed to assign role to user", app_errors.RoleAssignmentFailed.Error())
		assert.Equal(t, "failed to remove role from user", app_errors.RoleRemovalFailed.Error())
	})

	// Resource errors
	t.Run("ResourceErrors", func(t *testing.T) {
		assert.NotNil(t, app_errors.ResourceNotFound)
		assert.NotNil(t, app_errors.ResourceAlreadyExists)
		assert.NotNil(t, app_errors.ResourceCreateFailed)
		assert.NotNil(t, app_errors.ResourceDeleteFailed)
		assert.NotNil(t, app_errors.InvalidResourceCode)

		assert.Equal(t, "resource not found", app_errors.ResourceNotFound.Error())
		assert.Equal(t, "resource already exists", app_errors.ResourceAlreadyExists.Error())
		assert.Equal(t, "failed to create resource", app_errors.ResourceCreateFailed.Error())
		assert.Equal(t, "failed to delete resource", app_errors.ResourceDeleteFailed.Error())
		assert.Equal(t, "invalid resource code", app_errors.InvalidResourceCode.Error())
	})

	// Action errors
	t.Run("ActionErrors", func(t *testing.T) {
		assert.NotNil(t, app_errors.ActionNotFound)
		assert.NotNil(t, app_errors.ActionAlreadyExists)
		assert.NotNil(t, app_errors.ActionCreateFailed)
		assert.NotNil(t, app_errors.ActionDeleteFailed)
		assert.NotNil(t, app_errors.InvalidActionCode)

		assert.Equal(t, "action not found", app_errors.ActionNotFound.Error())
		assert.Equal(t, "action already exists", app_errors.ActionAlreadyExists.Error())
		assert.Equal(t, "failed to create action", app_errors.ActionCreateFailed.Error())
		assert.Equal(t, "failed to delete action", app_errors.ActionDeleteFailed.Error())
		assert.Equal(t, "invalid action code", app_errors.InvalidActionCode.Error())
	})

	// Permission errors
	t.Run("PermissionErrors", func(t *testing.T) {
		assert.NotNil(t, app_errors.PermissionNotFound)
		assert.NotNil(t, app_errors.PermissionAlreadyExists)
		assert.NotNil(t, app_errors.PermissionCreateFailed)
		assert.NotNil(t, app_errors.PermissionDeleteFailed)
		assert.NotNil(t, app_errors.InvalidPermissionCombo)

		assert.Equal(t, "permission not found", app_errors.PermissionNotFound.Error())
		assert.Equal(t, "permission already exists", app_errors.PermissionAlreadyExists.Error())
		assert.Equal(t, "failed to create permission", app_errors.PermissionCreateFailed.Error())
		assert.Equal(t, "failed to delete permission", app_errors.PermissionDeleteFailed.Error())
		assert.Equal(t, "invalid resource-action combination", app_errors.InvalidPermissionCombo.Error())
	})

	// Role-Permission errors
	t.Run("RolePermissionErrors", func(t *testing.T) {
		assert.NotNil(t, app_errors.RolePermissionNotFound)
		assert.NotNil(t, app_errors.RolePermissionExists)
		assert.NotNil(t, app_errors.RolePermissionCreateFailed)
		assert.NotNil(t, app_errors.RolePermissionDeleteFailed)

		assert.Equal(t, "role permission mapping not found", app_errors.RolePermissionNotFound.Error())
		assert.Equal(t, "role permission mapping already exists", app_errors.RolePermissionExists.Error())
		assert.Equal(t, "failed to create role permission", app_errors.RolePermissionCreateFailed.Error())
		assert.Equal(t, "failed to delete role permission", app_errors.RolePermissionDeleteFailed.Error())
	})

	// Access Member errors
	t.Run("AccessMemberErrors", func(t *testing.T) {
		assert.NotNil(t, app_errors.AccessMemberNotFound)
		assert.NotNil(t, app_errors.AccessMemberAlreadyExists)
		assert.NotNil(t, app_errors.AccessMemberCreateFailed)
		assert.NotNil(t, app_errors.AccessMemberDeleteFailed)
		assert.NotNil(t, app_errors.InvalidAccessScope)
		assert.NotNil(t, app_errors.MissingScopeID)
		assert.NotNil(t, app_errors.UserNotInScope)

		assert.Equal(t, "access member record not found", app_errors.AccessMemberNotFound.Error())
		assert.Equal(t, "user already has this role in the scope", app_errors.AccessMemberAlreadyExists.Error())
		assert.Equal(t, "failed to assign role to user", app_errors.AccessMemberCreateFailed.Error())
		assert.Equal(t, "failed to remove role from user", app_errors.AccessMemberDeleteFailed.Error())
		assert.Equal(t, "invalid access scope type", app_errors.InvalidAccessScope.Error())
		assert.Equal(t, "scope ID required for workspace or base scope", app_errors.MissingScopeID.Error())
		assert.Equal(t, "user does not have access to this scope", app_errors.UserNotInScope.Error())
	})

	// Permission check errors
	t.Run("PermissionCheckErrors", func(t *testing.T) {
		assert.NotNil(t, app_errors.PermissionDenied)
		assert.NotNil(t, app_errors.AccessDenied)
		assert.NotNil(t, app_errors.InsufficientPrivileges)

		assert.Equal(t, "user does not have permission to perform this action", app_errors.PermissionDenied.Error())
		assert.Equal(t, "access denied", app_errors.AccessDenied.Error())
		assert.Equal(t, "insufficient privileges for this operation", app_errors.InsufficientPrivileges.Error())
	})

	// Bulk operation errors
	t.Run("BulkOperationErrors", func(t *testing.T) {
		assert.NotNil(t, app_errors.BulkAssignmentFailed)
		assert.NotNil(t, app_errors.BulkRemovalFailed)
		assert.NotNil(t, app_errors.EmptyUserList)

		assert.Equal(t, "failed to assign roles to one or more users", app_errors.BulkAssignmentFailed.Error())
		assert.Equal(t, "failed to remove roles from one or more users", app_errors.BulkRemovalFailed.Error())
		assert.Equal(t, "user list cannot be empty for bulk operations", app_errors.EmptyUserList.Error())
	})

	// Scope errors
	t.Run("ScopeErrors", func(t *testing.T) {
		assert.NotNil(t, app_errors.InvalidScopeType)
		assert.NotNil(t, app_errors.ScopeNotFound)

		assert.Equal(t, "invalid scope type. Must be 'system', 'workspace', or 'base'", app_errors.InvalidScopeType.Error())
		assert.Equal(t, "scope not found", app_errors.ScopeNotFound.Error())
	})
}

// TestErrorsAreDistinct tests that all errors are distinct
func TestErrorsAreDistinct(t *testing.T) {
	errors := []error{
		app_errors.RoleDeleteFailed,
		app_errors.RoleUpdateFailed,
		app_errors.InvalidRolePriority,
		app_errors.ResourceNotFound,
		app_errors.ResourceAlreadyExists,
		app_errors.ActionNotFound,
		app_errors.ActionAlreadyExists,
		app_errors.PermissionNotFound,
		app_errors.PermissionAlreadyExists,
		app_errors.AccessMemberNotFound,
		app_errors.PermissionDenied,
		app_errors.AccessDenied,
	}

	seen := make(map[string]bool)
	for _, err := range errors {
		msg := err.Error()
		if seen[msg] {
			t.Errorf("Duplicate error message: %s", msg)
		}
		seen[msg] = true
	}
}

// TestErrorIsChecks tests that error comparison works
func TestErrorIsChecks(t *testing.T) {
	t.Run("direct comparison", func(t *testing.T) {
		err := app_errors.RoleNotFound
		assert.True(t, errors.Is(err, app_errors.RoleNotFound))
		assert.False(t, errors.Is(err, app_errors.RoleDeleteFailed))
	})

	t.Run("wrapped error", func(t *testing.T) {
		wrapped := errors.New("wrapped: " + app_errors.PermissionDenied.Error())
		assert.False(t, errors.Is(wrapped, app_errors.PermissionDenied)) // Different error
	})
}

// TestErrorsNotEmpty tests that errors have non-empty messages
func TestErrorsNotEmpty(t *testing.T) {
	errors := []error{
		app_errors.RoleDeleteFailed,
		app_errors.RoleUpdateFailed,
		app_errors.InvalidRolePriority,
		app_errors.RoleAssignmentFailed,
		app_errors.RoleRemovalFailed,
		app_errors.ResourceNotFound,
		app_errors.ResourceAlreadyExists,
		app_errors.ResourceCreateFailed,
		app_errors.ResourceDeleteFailed,
		app_errors.InvalidResourceCode,
		app_errors.ActionNotFound,
		app_errors.ActionAlreadyExists,
		app_errors.ActionCreateFailed,
		app_errors.ActionDeleteFailed,
		app_errors.InvalidActionCode,
		app_errors.PermissionNotFound,
		app_errors.PermissionAlreadyExists,
		app_errors.PermissionCreateFailed,
		app_errors.PermissionDeleteFailed,
		app_errors.InvalidPermissionCombo,
		app_errors.RolePermissionNotFound,
		app_errors.RolePermissionExists,
		app_errors.RolePermissionCreateFailed,
		app_errors.RolePermissionDeleteFailed,
		app_errors.AccessMemberNotFound,
		app_errors.AccessMemberAlreadyExists,
		app_errors.AccessMemberCreateFailed,
		app_errors.AccessMemberDeleteFailed,
		app_errors.InvalidAccessScope,
		app_errors.MissingScopeID,
		app_errors.UserNotInScope,
		app_errors.PermissionDenied,
		app_errors.AccessDenied,
		app_errors.InsufficientPrivileges,
		app_errors.BulkAssignmentFailed,
		app_errors.BulkRemovalFailed,
		app_errors.EmptyUserList,
		app_errors.InvalidScopeType,
		app_errors.ScopeNotFound,
	}

	for _, err := range errors {
		msg := err.Error()
		if msg == "" {
			t.Errorf("Error has empty message: %v", err)
		}
	}
}
