package services

import (
	"context"
	"godbgrest/pkg"
	"serenibase/internal/constant"
	"serenibase/internal/models/master"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/helpers"

	dbModels "godbgrest/pkg/models"
	app_errors "serenibase/internal/app-errors"

)

type subscriptionPlanService struct {
	repo *pkg.DatabaseService
}

func NewSubscriptionPlanService(repo *pkg.DatabaseService) interfaces.SubscriptionPlanService {
	return &subscriptionPlanService{repo: repo}
}

func (s *subscriptionPlanService) GetSubscriptionPlanByName(ctx context.Context, name string) (master.SubscriptionPlan, error) {
	limit := 1
	tableName := master.SubscriptionPlan{}.TableName(constant.MasterDatabase)
	query := dbModels.QueryParams{
		Select: []string{"id"},
		Filters: []dbModels.QueryFilter{
			{
				Column:   "name",
				Operator: "eq",
				Value:    name,
			},
		},
		Limit: &limit,
	}

	plansData, err := s.repo.TableService.GetTableData(ctx, tableName, query)
	if err != nil {
		return master.SubscriptionPlan{}, app_errors.DatabaseError
	}

	if len(plansData) == 0 {
		return master.SubscriptionPlan{}, app_errors.SubscriptionPlanNotFound
	}

	planData := plansData[0]

	var plan master.SubscriptionPlan
	if err := helpers.MapToStruct(planData, &plan); err != nil {
		return master.SubscriptionPlan{}, app_errors.ErrMapToStruct
	}
	return plan, nil
}


func (s *subscriptionPlanService) GetSubscriptionPlanById(ctx context.Context, id string) (master.SubscriptionPlan, error) {
	limit := 1
	tableName := master.SubscriptionPlan{}.TableName(constant.MasterDatabase)
	query := dbModels.QueryParams{
		Select: []string{"id"},
		Filters: []dbModels.QueryFilter{
			{
				Column:   "id",
				Operator: "eq",
				Value:    id,
			},
		},
		Limit: &limit,
	}

	plansData, err := s.repo.TableService.GetTableData(ctx, tableName, query)
	if err != nil {
		return master.SubscriptionPlan{}, app_errors.DatabaseError
	}

	if len(plansData) == 0 {
		return master.SubscriptionPlan{}, app_errors.SubscriptionPlanNotFound
	}

	planData := plansData[0]

	var plan master.SubscriptionPlan
	if err := helpers.MapToStruct(planData, &plan); err != nil {
		return master.SubscriptionPlan{}, app_errors.ErrMapToStruct
	}
	return plan, nil
}
