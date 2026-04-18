// Copyright (c) 2026 Aptlogica Technologies Private Limited
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"fmt"

	"github.com/aptlogica/go-postgres-rest/pkg"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"

	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
)

type userResetTokenService struct {
	repo *pkg.DatabaseService
}

func NewUserResetTokenService(repo *pkg.DatabaseService) interfaces.UserResetTokenService {
	return &userResetTokenService{repo: repo}
}

func (s *userResetTokenService) CreateUserResetToken(ctx context.Context, req dto.UserResetTokenInsertion) (tenant.UserResetToken, error) {
	lg := logger.Get()
	tableName := tenant.UserResetToken{}.TableName(constant.MasterDatabase)

	// Check if a reset token already exists for this user_id
	limit := 1
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "user_id",
				Operator: "eq",
				Value:    req.UserID,
			},
		},
		Limit: &limit,
	}

	existing, err := s.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to fetch existing reset tokens")
		return tenant.UserResetToken{}, app_errors.LogDatabaseError(err, "failed to fetch existing reset tokens")
	}

	// Always insert new record, delete any existing record for this user_id first
	if len(existing) > 0 {

		for _, record := range existing {
			idVal, ok := record["id"]
			if !ok {
				lg.Warn().Msg("ID field not found in reset token record")
				errMissingID := fmt.Errorf("id field missing in reset token record")
				return tenant.UserResetToken{}, app_errors.LogDatabaseError(errMissingID, "reset token record missing id")
			}
			if err := s.repo.TableService.DeleteRecord(tableName, idVal); err != nil {
				lg.Error().Stack().Err(err).Msg("Failed to delete existing reset token")
				return tenant.UserResetToken{}, app_errors.LogDatabaseError(err, "failed to delete existing reset token")
			}
		}
	}
	recordData, err := s.repo.TableService.CreateRecord(tableName, req.Map())
	if err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to create reset token record")
		return tenant.UserResetToken{}, app_errors.LogDatabaseError(err, "failed to create reset token record")
	}

	var out tenant.UserResetToken
	if err := helpers.MapToStruct(recordData, &out); err != nil {
		return tenant.UserResetToken{}, app_errors.ErrMapToStruct
	}
	return out, nil
}

func (s *userResetTokenService) GetUserResetToken(ctx context.Context, token string) (tenant.UserResetToken, error) {
	limit := 1
	tableName := tenant.UserResetToken{}.TableName(constant.MasterDatabase)
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "token",
				Operator: "eq",
				Value:    token,
			},
		},
		Limit: &limit,
	}

	tokensData, err := s.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return tenant.UserResetToken{}, app_errors.LogDatabaseError(err, "failed to fetch reset token by token")
	}

	if len(tokensData) == 0 {
		return tenant.UserResetToken{}, app_errors.ErrRecordNotFound
	}

	tokenData := tokensData[0]

	var outToken tenant.UserResetToken
	if err := helpers.MapToStruct(tokenData, &outToken); err != nil {
		return tenant.UserResetToken{}, app_errors.ErrMapToStruct
	}
	return outToken, nil
}

func (s *userResetTokenService) DeleteTokensByUserId(ctx context.Context, userId string) error {
	lg := logger.Get()
	tableName := tenant.UserResetToken{}.TableName(constant.MasterDatabase)
	lg.Debug().Str("userId", userId).Str("tableName", tableName).Msg("Starting DeleteTokensByUserId")

	// Build filter for user_id
	filter := dbModels.QueryFilter{
		Column:   "user_id",
		Operator: "eq",
		Value:    userId,
	}
	lg.Debug().Interface("filter", filter).Msg("Built user_id filter")

	// First, get records by user_id
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{filter},
	}
	records, err := s.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		lg.Error().Stack().Err(err).Str("userId", userId).Msg("Error fetching reset tokens by user id")
		return app_errors.LogDatabaseError(err, "failed to fetch reset tokens by user id")
	}
	if len(records) == 0 {
		lg.Debug().Str("userId", userId).Msg("No reset token records found for user_id")
	}
	if len(records) > 0 {
		for _, record := range records {
			idVal, ok := record["id"]
			if !ok {
				errMissingID := fmt.Errorf("id field missing in reset token record")
				return app_errors.LogDatabaseError(errMissingID, "reset token record missing id during delete")
			}
			if err := s.repo.TableService.DeleteRecord(tableName, idVal); err != nil {
				return app_errors.LogDatabaseError(err, "failed to delete reset token by user id")
			}
		}
	}
	return nil
}
