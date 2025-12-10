package interfaces

import (
	"context"
	"serenibase/internal/dto"
	"serenibase/internal/models/master"

)

type TenantSubscriptionService interface {
	CreateTenantSubscription(ctx context.Context, tenantSubscriptionData dto.TenantSubscriptionInsertion) (master.TenantSubscription, error)
	GetTenantSubscription(ctx context.Context, tenantID string) (master.TenantSubscription, error)
	// GetTenantSubscriptionByID(ctx context.Context, subscriptionID uuid.UUID) (master.TenantSubscription, error)
	// UpdateTenantSubscription(ctx context.Context, subscriptionID uuid.UUID, updates map[string]interface{}) (master.TenantSubscription, error)
	// CancelTenantSubscription(ctx context.Context, subscriptionID uuid.UUID) error
	// ReactivateTenantSubscription(ctx context.Context, subscriptionID uuid.UUID) error
	// GetActiveSubscriptions(ctx context.Context) ([]master.TenantSubscription, error)
	// GetExpiredSubscriptions(ctx context.Context) ([]master.TenantSubscription, error)
	// GetTrialSubscriptions(ctx context.Context) ([]master.TenantSubscription, error)
	// UpdateBillingPeriod(ctx context.Context, subscriptionID uuid.UUID, startDate, endDate string) error
	// SetPaymentProviderInfo(ctx context.Context, subscriptionID uuid.UUID, provider, subscriptionID, customerID string) error
}
