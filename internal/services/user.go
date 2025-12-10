package services

import (
	"context"
	"fmt"
	"godbgrest/pkg"
	app_errors "serenibase/internal/app-errors"
	"serenibase/internal/dto"
	"serenibase/internal/models/master"
	"serenibase/internal/services/interfaces"
	"time"

	"serenibase/internal/utils/helpers"

	dbModels "godbgrest/pkg/models"

	"github.com/google/uuid"
)

type userService struct {
	repo *pkg.DatabaseService
}

func NewUserService(repo *pkg.DatabaseService) interfaces.UserService {
	return &userService{repo: repo}
}

func (u *userService) CreateUser(ctx context.Context, schema string, req dto.RegisterRequest) (master.User, error) {
	tableName := master.User{}.TableName(schema)

	// Parse DateOfBirth if present
	// var dateOfBirth *time.Time

	// if req.DateOfBirth != "" {
	// 	// example format: "2025-11-03"
	// 	parsed, err := time.Parse("2006-01-02", req.DateOfBirth)
	// 	if err != nil {
	// 		return master.User{}, fmt.Errorf("invalid date of birth format (expected YYYY-MM-DD): %v", err)
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
		DeletedAt:     nil,
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

	insertedUserData, err := u.repo.TableService.CreateRecord(ctx, tableName, userData.Map())
	if err != nil {
		fmt.Println("CreateUser-->", err)
		return master.User{}, app_errors.DatabaseError
	}
	fmt.Println("insertedUserData--->", insertedUserData)

	var insertedUser master.User
	if err := helpers.MapToStruct(insertedUserData, &insertedUser); err != nil {
		return master.User{}, app_errors.ErrMapToStruct
	}
	return insertedUser, nil
}

func (u *userService) GetUserByEmail(ctx context.Context, schema string, email string) (master.User, error) {
	tableName := master.User{}.TableName(schema)
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

	usersData, err := u.repo.TableService.GetTableData(ctx, tableName, query)
	if err != nil {
		return master.User{}, app_errors.DatabaseError
	}

	if len(usersData) == 0 {
		return master.User{}, app_errors.UserNotFound
	}

	userData := usersData[0]

	var user master.User
	if err := helpers.MapToStruct(userData, &user); err != nil {
		return master.User{}, app_errors.ErrMapToStruct
	}
	// if user.DateOfBirth != nil {
	// 	*user.DateOfBirth = user.DateOfBirth
	// }
	return user, nil
}

func (u *userService) GetUserByID(ctx context.Context, schema string, id string) (master.User, error) {
	tableName := master.User{}.TableName(schema)
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

	usersData, err := u.repo.TableService.GetTableData(ctx, tableName, query)
	if err != nil {
		fmt.Println("errr====", err)
		fmt.Println(schema, id)
		return master.User{}, app_errors.DatabaseError
	}

	if len(usersData) == 0 {
		return master.User{}, app_errors.UserNotFound
	}

	userData := usersData[0]

	var user master.User
	if err := helpers.MapToStruct(userData, &user); err != nil {
		return master.User{}, app_errors.ErrMapToStruct
	}
	// if user.DateOfBirth != nil {
	// 	*user.DateOfBirth = time.Date(user.DateOfBirth.Year(), user.DateOfBirth.Month(), user.DateOfBirth.Day(), 0, 0, 0, 0, time.UTC)
	// }
	return user, nil
}

func (u *userService) UpdateUser(ctx context.Context, schema string, id string, updateData map[string]interface{}) (master.User, error) {
	tableName := master.User{}.TableName(schema)

	if dob, ok := updateData["DateOfBirth"]; ok {
		if strDob, isString := dob.(string); isString {
			parsed, err := time.Parse("2006-01-02", strDob)
			if err != nil {
				return master.User{}, fmt.Errorf("invalid dob format (expected YYYY-MM-DD): %v", err)
			}
			onlyDate := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
			updateData["date_of_birth"] = onlyDate
		}
		delete(updateData, "DateOfBirth")
	}

	updatedRecord, err := u.repo.TableService.UpdateRecord(ctx, tableName, id, updateData)
	fmt.Println("updatedRecord, err", updatedRecord, err)
	if err != nil {
		fmt.Println("UpdateUser err------>", err)
		return master.User{}, app_errors.DatabaseError
	}
	fmt.Println("updatedRecord--->", updatedRecord)

	var updatedUser master.User
	if err := helpers.MapToStruct(updatedRecord, &updatedUser); err != nil {
		return master.User{}, app_errors.ErrMapToStruct
	}
	// if updatedUser.DateOfBirth != nil {
	// 	*updatedUser.DateOfBirth = time.Date(updatedUser.DateOfBirth.Year(), updatedUser.DateOfBirth.Month(), updatedUser.DateOfBirth.Day(), 0, 0, 0, 0, time.UTC)
	// }
	return updatedUser, nil
}

func (u *userService) GetAllUsers(ctx context.Context, schema string) ([]master.User, error) {
	tableName := master.User{}.TableName(schema)
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
	usersData, err := u.repo.TableService.GetTableData(ctx, tableName, query)
	if err != nil {
		return nil, app_errors.DatabaseError
	}
	if len(usersData) == 0 {
		return []master.User{}, nil
	}
	var users []master.User
	for _, userData := range usersData {
		var user master.User
		if err := helpers.MapToStruct(userData, &user); err != nil {
			return nil, app_errors.ErrMapToStruct
		}
		users = append(users, user)
	}
	return users, nil
}

func (u *userService) GetBulkUsers(ctx context.Context, schema string, ids []string) ([]master.User, error) {
	if len(ids) == 0 {
		return []master.User{}, nil
	}

	tableName := master.User{}.TableName(schema)

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

	rows, err := u.repo.TableService.GetTableData(ctx, tableName, params)
	if err != nil {
		return nil, app_errors.DatabaseError
	}
	if len(rows) == 0 {
		return []master.User{}, nil
	}

	users := make([]master.User, 0, len(rows))
	for _, row := range rows {
		var user master.User
		if err := helpers.MapToStruct(row, &user); err != nil {
			return nil, app_errors.ErrMapToStruct
		}
		users = append(users, user)
	}

	return users, nil
}

func (u *userService) DeleteUser(ctx context.Context, schema string, id string) error {
	tableName := master.User{}.TableName(schema)
	return u.repo.TableService.DeleteRecord(ctx, tableName, id)
}
