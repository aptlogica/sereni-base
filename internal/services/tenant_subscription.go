package services

import (
	"context"

	"godbgrest/pkg"
	"serenibase/internal/dto"
	"serenibase/internal/models/master"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/helpers"
	"time"

	"serenibase/internal/constant"

	app_errors "serenibase/internal/app-errors"

	dbModels "godbgrest/pkg/models"

	"github.com/google/uuid"
)

type tenantSubscriptionService struct {
	tableName string
	repo      *pkg.DatabaseService
}

func NewTenantSubscriptionService(repo *pkg.DatabaseService) interfaces.TenantSubscriptionService {
	tableName := master.TenantSubscription{}.TableName(constant.MasterDatabase)
	return &tenantSubscriptionService{tableName: tableName, repo: repo}
}

func (t *tenantSubscriptionService) CreateTenantSubscription(ctx context.Context, tenantSubscriptionData dto.TenantSubscriptionInsertion) (master.TenantSubscription, error) {
	if tenantSubscriptionData.ID == uuid.Nil {
		tenantSubscriptionData.ID = uuid.New()
	}
	tenantSubscriptionData.CreatedAt = time.Now()
	tenantSubscriptionData.UpdatedAt = time.Now()
	tenantSubscriptionData.Status = "active"

	subscriptionMap := tenantSubscriptionData.Map()

	insertedData, err := t.repo.TableService.CreateRecord(ctx, t.tableName, subscriptionMap)
	if err != nil {
		return master.TenantSubscription{}, app_errors.DatabaseError
	}

	var insertedSubscription master.TenantSubscription
	if err := helpers.MapToStruct(insertedData, &insertedSubscription); err != nil {
		return master.TenantSubscription{}, app_errors.ErrMapToStruct
	}

	return insertedSubscription, nil
}

func (t *tenantSubscriptionService) GetTenantSubscription(ctx context.Context, tenantID string) (master.TenantSubscription, error) {
	limit := 1
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "tenant_id",
				Operator: "eq",
				Value:    tenantID,
			},
		},
		Limit: &limit,
	}

	data, err := t.repo.TableService.GetTableData(ctx, t.tableName, query)
	if err != nil {
		return master.TenantSubscription{}, app_errors.DatabaseError
	}

	if len(data) == 0 {
		return master.TenantSubscription{}, app_errors.SubscriptionPlanNotFound
	}

	var subscription master.TenantSubscription
	if err := helpers.MapToStruct(data[0], &subscription); err != nil {
		return master.TenantSubscription{}, app_errors.ErrMapToStruct
	}
	return subscription, nil
}

// func (t *tenantSubscriptionService) GetTenantSubscriptionByID(ctx context.Context, subscriptionID uuid.UUID) (master.TenantSubscription, error) {
// 	conditions := map[string]interface{}{
// 		"id": subscriptionID,
// 	}

// 	data, err := t.repo.TableService.GetRecord(ctx, t.tableName, conditions)
// 	if err != nil {
// 		return master.TenantSubscription{}, app_errors.DatabaseError
// 	}

// 	var subscription master.TenantSubscription
// 	if err := helpers.MapToStruct(data, &subscription); err != nil {
// 		return master.TenantSubscription{}, app_errors.ErrMapToStruct
// 	}

// 	return subscription, nil
// }

// func (t *tenantSubscriptionService) UpdateTenantSubscription(ctx context.Context, subscriptionID uuid.UUID, updates map[string]interface{}) (master.TenantSubscription, error) {
// 	updates["last_modified_time"] = time.Now()

// 	conditions := map[string]interface{}{
// 		"id": subscriptionID,
// 	}

// 	updatedData, err := t.repo.TableService.UpdateRecord(ctx, t.tableName, conditions, updates)
// 	if err != nil {
// 		return master.TenantSubscription{}, app_errors.DatabaseError
// 	}

// 	var subscription master.TenantSubscription
// 	if err := helpers.MapToStruct(updatedData, &subscription); err != nil {
// 		return master.TenantSubscription{}, app_errors.ErrMapToStruct
// 	}

// 	return subscription, nil
// }

// func (t *tenantSubscriptionService) CancelTenantSubscription(ctx context.Context, subscriptionID uuid.UUID) error {
// 	updates := map[string]interface{}{
// 		"status":      "canceled",
// 		"canceled_at": time.Now(),
// 		"last_modified_time":  time.Now(),
// 	}

// 	conditions := map[string]interface{}{
// 		"id": subscriptionID,
// 	}

// 	_, err := t.repo.TableService.UpdateRecord(ctx, t.tableName, conditions, updates)
// 	return err
// }

// func (t *tenantSubscriptionService) ReactivateTenantSubscription(ctx context.Context, subscriptionID uuid.UUID) error {
// 	updates := map[string]interface{}{
// 		"status":      "active",
// 		"canceled_at": nil,
// 		"last_modified_time":  time.Now(),
// 	}

// 	conditions := map[string]interface{}{
// 		"id": subscriptionID,
// 	}

// 	_, err := t.repo.TableService.UpdateRecord(ctx, t.tableName, conditions, updates)
// 	return err
// }

// func (t *tenantSubscriptionService) GetActiveSubscriptions(ctx context.Context) ([]master.TenantSubscription, error) {
// 	conditions := map[string]interface{}{
// 		"status": "active",
// 	}

// 	data, err := t.repo.TableService.GetRecords(ctx, t.tableName, conditions)
// 	if err != nil {
// 		return nil, app_errors.DatabaseError
// 	}

// 	var subscriptions []master.TenantSubscription
// 	for _, record := range data {
// 		var subscription master.TenantSubscription
// 		if err := helpers.MapToStruct(record, &subscription); err != nil {
// 			return nil, app_errors.ErrMapToStruct
// 		}
// 		subscriptions = append(subscriptions, subscription)
// 	}

// 	return subscriptions, nil
// }

// func (t *tenantSubscriptionService) GetExpiredSubscriptions(ctx context.Context) ([]master.TenantSubscription, error) {
// 	// This would need a more complex query to check current_period_end < now()
// 	// For now, returning subscriptions with status "expired"
// 	conditions := map[string]interface{}{
// 		"status": "expired",
// 	}

// 	data, err := t.repo.TableService.GetRecords(ctx, t.tableName, conditions)
// 	if err != nil {
// 		return nil, app_errors.DatabaseError
// 	}

// 	var subscriptions []master.TenantSubscription
// 	for _, record := range data {
// 		var subscription master.TenantSubscription
// 		if err := helpers.MapToStruct(record, &subscription); err != nil {
// 			return nil, app_errors.ErrMapToStruct
// 		}
// 		subscriptions = append(subscriptions, subscription)
// 	}

// 	return subscriptions, nil
// }

// func (t *tenantSubscriptionService) GetTrialSubscriptions(ctx context.Context) ([]master.TenantSubscription, error) {
// 	conditions := map[string]interface{}{
// 		"status": "trial",
// 	}

// 	data, err := t.repo.TableService.GetRecords(ctx, t.tableName, conditions)
// 	if err != nil {
// 		return nil, app_errors.DatabaseError
// 	}

// 	var subscriptions []master.TenantSubscription
// 	for _, record := range data {
// 		var subscription master.TenantSubscription
// 		if err := helpers.MapToStruct(record, &subscription); err != nil {
// 			return nil, app_errors.ErrMapToStruct
// 		}
// 		subscriptions = append(subscriptions, subscription)
// 	}

// 	return subscriptions, nil
// }

// func (t *tenantSubscriptionService) UpdateBillingPeriod(ctx context.Context, subscriptionID uuid.UUID, startDate, endDate string) error {
// 	updates := map[string]interface{}{
// 		"current_period_start": startDate,
// 		"current_period_end":   endDate,
// 		"last_modified_time":           time.Now(),
// 	}

// 	conditions := map[string]interface{}{
// 		"id": subscriptionID,
// 	}

// 	_, err := t.repo.TableService.UpdateRecord(ctx, t.tableName, conditions, updates)
// 	return err
// }

// func (t *tenantSubscriptionService) SetPaymentProviderInfo(ctx context.Context, subscriptionID uuid.UUID, provider, subscriptionIDStr, customerID string) error {
// 	updates := map[string]interface{}{
// 		"payment_provider":                 provider,
// 		"payment_provider_subscription_id": subscriptionIDStr,
// 		"payment_provider_customer_id":     customerID,
// 		"last_modified_time":                       time.Now(),
// 	}

// 	conditions := map[string]interface{}{
// 		"id": subscriptionID,
// 	}

// 	_, err := t.repo.TableService.UpdateRecord(ctx, t.tableName, conditions, updates)
// 	return err
// }
