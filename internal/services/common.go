package services

import (
	"go-postgres-rest/pkg"
	dbModels "go-postgres-rest/pkg/models"
	app_errors "serenibase/internal/app-errors"
	"serenibase/internal/utils/helpers"
)

// getSingleRecord fetches a single record and maps it to type T
func getSingleRecord[T any](repo *pkg.DatabaseService, tableName string, query dbModels.QueryParams, errorMsg string) (T, error) {
	data, err := repo.TableService.GetTableData(tableName, query)
	if err != nil {
		var zero T
		return zero, app_errors.LogDatabaseError(err, errorMsg)
	}

	if len(data) == 0 {
		var zero T
		return zero, app_errors.ErrRecordNotFound
	}

	var result T
	if err := helpers.MapToStruct(data[0], &result); err != nil {
		var zero T
		return zero, app_errors.ErrMapToStruct
	}
	return result, nil
}

// listRecords lists records and maps them to []T
func listRecords[T any](repo *pkg.DatabaseService, tableName string, query dbModels.QueryParams, errorMsg string) ([]T, error) {
	data, err := repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, errorMsg)
	}

	results := make([]T, 0, len(data))
	for _, item := range data {
		var result T
		if err := helpers.MapToStruct(item, &result); err != nil {
			return nil, app_errors.ErrMapToStruct
		}
		results = append(results, result)
	}
	return results, nil
}

// createSingleFilterQuery helper to create a simple filter query
func createSingleFilterQuery(column, operator, value string, limit int) dbModels.QueryParams {
	return dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   column,
				Operator: operator,
				Value:    value,
			},
		},
		Limit: &limit,
	}
}

// countRecords returns the total count of records in a table
func countRecords(repo *pkg.DatabaseService, tableName string, errorMsg string) (int64, error) {
	countQuery := dbModels.QueryParams{
		Aggregates: []dbModels.AggregateFunction{
			{
				Function: "COUNT",
				Column:   "id",
				Alias:    "total",
			},
		},
	}

	countData, err := repo.TableService.GetTableData(tableName, countQuery)
	if err != nil {
		return 0, app_errors.LogDatabaseError(err, errorMsg)
	}

	count := int64(0)
	if len(countData) > 0 {
		if total, ok := countData[0]["total"]; ok {
			count = int64(total.(float64))
		}
	}
	return count, nil
}
