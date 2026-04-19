// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"fmt"
	"time"

	"github.com/aptlogica/go-postgres-rest/pkg"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"

	"github.com/aptlogica/sereni-base/internal/utils/helpers"

	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"

	"github.com/google/uuid"
)

type userService struct {
	repo *pkg.DatabaseService
}

func NewUserService(repo *pkg.DatabaseService) interfaces.UserService {
	return &userService{repo: repo}
}

func (u *userService) CreateUser(ctx context.Context, schema string, req dto.RegisterRequest) (tenant.User, error) {
	lg := logger.Get()
	tableName := tenant.User{}.TableName(schema)

	// Parse DateOfBirth if present
	// var dateOfBirth *time.Time

	// if req.DateOfBirth != "" {
	// 	// example format: "2025-11-03"
	// 	parsed, err := time.Parse("2006-01-02", req.DateOfBirth)
	// 	if err != nil {
	// 		return tenant.User{}, fmt.Errorf("invalid date of birth format (expected YYYY-MM-DD): %v", err)
	// 	}
	// 	onlyDate := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
	// 	dateOfBirth = &onlyDate
	// }

	userData := dto.UserInsertion{
		ID:            uuid.New(),
		Email:         req.Email,
		Password:      req.Password,
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		DisplayName:   fmt.Sprintf("%s %s", req.FirstName, req.LastName),
		EmailVerified: req.EmailVerified,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
		DateOfBirth:   req.DateOfBirth,
		Country:       req.Country,
		Timezone:      req.Timezone,
		Status:        req.Status,
		AuthProvider:  req.AuthProvider,
		Roles:         req.Roles,
	}

	// Set fields conditionally
	if req.Status != "" {
		userData.Status = req.Status
	}
	userData.EmailVerified = req.EmailVerified
	if req.ID != uuid.Nil {
		userData.ID = req.ID
	}
	if req.AuthProvider != "" {
		userData.AuthProvider = req.AuthProvider
	}

	insertedUserData, err := u.repo.TableService.CreateRecord(tableName, userData.Map())
	if err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to create user record")
		return tenant.User{}, app_errors.LogDatabaseError(err, "failed to create user record")
	}
	lg.Debug().Interface("userData", insertedUserData).Msg("User record created successfully")

	var insertedUser tenant.User
	if err := helpers.MapToStruct(insertedUserData, &insertedUser); err != nil {
		return tenant.User{}, app_errors.ErrMapToStruct
	}
	return insertedUser, nil
}

func (u *userService) GetUserByEmail(ctx context.Context, schema string, email string) (tenant.User, error) {
	tableName := tenant.User{}.TableName(schema)
	limit := 1
	query := dbModels.QueryParams{
		Select: []string{"*"},
		Filters: []dbModels.QueryFilter{
			{
				Column:   "email",
				Operator: "eq",
				Value:    email,
			},
		},
		Limit: &limit,
	}

	usersData, err := u.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return tenant.User{}, app_errors.LogDatabaseError(err, "failed to get user by email")
	}
	if len(usersData) == 0 {
		return tenant.User{}, app_errors.UserNotFound
	}

	userData := usersData[0]

	var user tenant.User
	if err := helpers.MapToStruct(userData, &user); err != nil {
		return tenant.User{}, app_errors.ErrMapToStruct
	}
	// if user.DateOfBirth != nil {
	// 	*user.DateOfBirth = user.DateOfBirth
	// }

	fmt.Printf("Retrieved user by email %s: %+v\n", email, user)
	return user, nil
}

func (u *userService) GetUserByID(ctx context.Context, schema string, id string) (tenant.User, error) {
	lg := logger.Get()
	tableName := tenant.User{}.TableName(schema)
	limit := 1
	query := dbModels.QueryParams{
		Select: []string{"*"},
		Filters: []dbModels.QueryFilter{
			{
				Column:   "id",
				Operator: "eq",
				Value:    id,
			},
		},
		Limit: &limit,
	}

	usersData, err := u.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		lg.Error().Stack().Err(err).Str("schema", schema).Str("id", id).Msg("Failed to get user by ID")
		return tenant.User{}, app_errors.LogDatabaseError(err, "failed to get user by id")
	}

	if len(usersData) == 0 {
		return tenant.User{}, app_errors.UserNotFound
	}

	userData := usersData[0]

	var user tenant.User
	if err := helpers.MapToStruct(userData, &user); err != nil {
		return tenant.User{}, app_errors.ErrMapToStruct
	}
	// if user.DateOfBirth != nil {
	// 	*user.DateOfBirth = time.Date(user.DateOfBirth.Year(), user.DateOfBirth.Month(), user.DateOfBirth.Day(), 0, 0, 0, 0, time.UTC)
	// }
	return user, nil
}

func (u *userService) UpdateUser(ctx context.Context, schema string, id string, updateData map[string]interface{}) (tenant.User, error) {
	lg := logger.Get()
	tableName := tenant.User{}.TableName(schema)

	if dob, ok := updateData["DateOfBirth"]; ok {
		if strDob, isString := dob.(string); isString {
			parsed, err := time.Parse("2006-01-02", strDob)
			if err != nil {
				return tenant.User{}, fmt.Errorf("invalid dob format (expected YYYY-MM-DD): %v", err)
			}
			onlyDate := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
			updateData["date_of_birth"] = onlyDate
		}
		delete(updateData, "DateOfBirth")
	}

	updatedRecord, err := u.repo.TableService.UpdateRecord(tableName, id, updateData)
	lg.Debug().Interface("record", updatedRecord).Msg("Updated user record")
	if err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to update user record")
		return tenant.User{}, app_errors.LogDatabaseError(err, "failed to update user record")
	}
	lg.Debug().Interface("record", updatedRecord).Msg("User update completed")

	var updatedUser tenant.User
	if err := helpers.MapToStruct(updatedRecord, &updatedUser); err != nil {
		return tenant.User{}, app_errors.ErrMapToStruct
	}
	// if updatedUser.DateOfBirth != nil {
	// 	*updatedUser.DateOfBirth = time.Date(updatedUser.DateOfBirth.Year(), updatedUser.DateOfBirth.Month(), updatedUser.DateOfBirth.Day(), 0, 0, 0, 0, time.UTC)
	// }
	return updatedUser, nil
}

func (u *userService) GetAllUsers(ctx context.Context, schema string) ([]tenant.User, error) {
	tableName := tenant.User{}.TableName(schema)
	query := dbModels.QueryParams{
		Select: []string{"*"},
		Filters: []dbModels.QueryFilter{
			{
				Column:   "is_deleted",
				Operator: "eq",
				Value:    false,
			},
		},
	}
	usersData, err := u.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to get all users")
	}
	if len(usersData) == 0 {
		return []tenant.User{}, nil
	}
	var users []tenant.User
	for _, userData := range usersData {
		var user tenant.User
		if err := helpers.MapToStruct(userData, &user); err != nil {
			return nil, app_errors.ErrMapToStruct
		}
		users = append(users, user)
	}
	return users, nil
}

func (u *userService) GetBulkUsers(ctx context.Context, schema string, ids []string) ([]tenant.User, error) {
	if len(ids) == 0 {
		return []tenant.User{}, nil
	}

	tableName := tenant.User{}.TableName(schema)

	filters := []dbModels.QueryFilter{
		{
			Column:   "id",
			Operator: "in",
			Value:    ids,
		},
	}

	params := dbModels.QueryParams{
		Select:  []string{"*"},
		Filters: filters,
	}

	rows, err := u.repo.TableService.GetTableData(tableName, params)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to get bulk users")
	}
	if len(rows) == 0 {
		return []tenant.User{}, nil
	}

	users := make([]tenant.User, 0, len(rows))
	for _, row := range rows {
		var user tenant.User
		if err := helpers.MapToStruct(row, &user); err != nil {
			return nil, app_errors.ErrMapToStruct
		}
		users = append(users, user)
	}

	return users, nil
}

func (u *userService) DeleteUser(ctx context.Context, schema string, id string) error {
	tableName := tenant.User{}.TableName(schema)
	return u.repo.TableService.DeleteRecord(tableName, id)
}
