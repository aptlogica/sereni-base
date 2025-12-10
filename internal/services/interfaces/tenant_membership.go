package interfaces

import (
	"context"
	"serenibase/internal/dto"
	"serenibase/internal/models/master"

	"github.com/google/uuid"
)

type TenantMembershipService interface {
	CreateTenantMembership(ctx context.Context, tenantMembershipData dto.TenantMembershipInsertion) (master.TenantMembership, error)
	GetTenantMembership(ctx context.Context, tenantID string, userID uuid.UUID) (master.TenantMembership, error)
	GetTenantMembershipByUser(ctx context.Context, userID uuid.UUID) (master.TenantMembership, error)
	// GetTenantMembershipByID(ctx context.Context, membershipID uuid.UUID) (master.TenantMembership, error)
	GetTenantMembers(ctx context.Context, tenantID string) ([]master.TenantMembership, error)
	// GetUserTenants(ctx context.Context, userID uuid.UUID) ([]master.TenantMembership, error)
	// UpdateTenantMembership(ctx context.Context, membershipID uuid.UUID, updates map[string]interface{}) (master.TenantMembership, error)
	// UpdateUserRole(ctx context.Context, tenantID, userID, roleID uuid.UUID) error
	// UpdateUserPermissions(ctx context.Context, tenantID, userID uuid.UUID, permissions string) error
	// RemoveTenantMember(ctx context.Context, tenantID, userID uuid.UUID) error
	// InviteUserToTenant(ctx context.Context, tenantID, invitedBy uuid.UUID, email, roleID string) (master.TenantMembership, error)
	// AcceptInvitation(ctx context.Context, invitationToken string) error
	// RejectInvitation(ctx context.Context, invitationToken string) error
	// GetPendingInvitations(ctx context.Context, tenantID uuid.UUID) ([]master.TenantMembership, error)
	// UpdateLastAccess(ctx context.Context, tenantID, userID uuid.UUID) error
	// CheckUserPermission(ctx context.Context, tenantID, userID uuid.UUID, permission string) (bool, error)
}
