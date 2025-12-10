package interfaces

import (
	"context"
	"serenibase/internal/models/master"
)

type SubscriptionPlanService interface {
	GetSubscriptionPlanByName(ctx context.Context, name string) (master.SubscriptionPlan, error)
	GetSubscriptionPlanById(ctx context.Context, id string) (master.SubscriptionPlan, error)
}
