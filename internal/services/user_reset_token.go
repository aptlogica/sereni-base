package services

import (
	"context"
	"godbgrest/pkg"
	app_errors "serenibase/internal/app-errors"
	"serenibase/internal/constant"
	"serenibase/internal/dto"
	"serenibase/internal/models/master"
	"serenibase/internal/providers/logger"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/helpers"

	dbModels "godbgrest/pkg/models"
)

type userResetTokenService struct {
	repo *pkg.DatabaseService
}

func NewUserResetTokenService(repo *pkg.DatabaseService) interfaces.UserResetTokenService {
	return &userResetTokenService{repo: repo}
}

func (s *userResetTokenService) CreateUserResetToken(ctx context.Context, req dto.UserResetTokenInsertion) (master.UserResetToken, error) {
	lg := logger.Get()
	tableName := master.UserResetToken{}.TableName(constant.MasterDatabase)

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

	existing, err := s.repo.TableService.GetTableData(ctx, tableName, query)
	if err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to fetch existing reset tokens")
		return master.UserResetToken{}, app_errors.DatabaseError
	}

	// Always insert new record, delete any existing record for this user_id first
	if len(existing) > 0 {

		for _, record := range existing {
			idVal, ok := record["id"]
			if !ok {
				lg.Warn().Msg("ID field not found in reset token record")
				return master.UserResetToken{}, app_errors.DatabaseError
			}
			if err := s.repo.TableService.DeleteRecord(ctx, tableName, idVal); err != nil {
				lg.Error().Stack().Err(err).Msg("Failed to delete existing reset token")
				return master.UserResetToken{}, app_errors.DatabaseError
			}
		}
	}
	recordData, err := s.repo.TableService.CreateRecord(ctx, tableName, req.Map())
	if err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to create reset token record")
		return master.UserResetToken{}, app_errors.DatabaseError
	}

	var out master.UserResetToken
	if err := helpers.MapToStruct(recordData, &out); err != nil {
		return master.UserResetToken{}, app_errors.ErrMapToStruct
	}
	return out, nil
}

func (s *userResetTokenService) GetUserResetToken(ctx context.Context, token string) (master.UserResetToken, error) {
	limit := 1
	tableName := master.UserResetToken{}.TableName(constant.MasterDatabase)
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

	tokensData, err := s.repo.TableService.GetTableData(ctx, tableName, query)
	if err != nil {
		return master.UserResetToken{}, app_errors.DatabaseError
	}

	if len(tokensData) == 0 {
		return master.UserResetToken{}, app_errors.ErrRecordNotFound
	}

	tokenData := tokensData[0]

	var outToken master.UserResetToken
	if err := helpers.MapToStruct(tokenData, &outToken); err != nil {
		return master.UserResetToken{}, app_errors.ErrMapToStruct
	}
	return outToken, nil
}

func (s *userResetTokenService) DeleteTokensByUserId(ctx context.Context, userId string) error {
	lg := logger.Get()
	tableName := master.UserResetToken{}.TableName(constant.MasterDatabase)
	// Build filter for user_id
	filter := dbModels.QueryFilter{
		Column:   "user_id",
		Operator: "eq",
		Value:    userId,
	}

	// First, get records by user_id
	query := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{filter},
	}
	records, err := s.repo.TableService.GetTableData(ctx, tableName, query)
	if err != nil {
		return app_errors.DatabaseError
	}
	if len(records) > 0 {

		for _, record := range records {
			idVal, ok := record["id"]
			if !ok {
				lg.Warn().Msg("ID field not found in reset token record")
				return app_errors.DatabaseError
			}
			if err := s.repo.TableService.DeleteRecord(ctx, tableName, idVal); err != nil {
				lg.Error().Stack().Err(err).Msg("Failed to delete reset token by user ID")
				return app_errors.DatabaseError
			}
		}
	}
	return nil
}
