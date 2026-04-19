// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"github.com/aptlogica/go-postgres-rest/pkg"
	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"
)

// GetSingleRecord fetches a single record and maps it to type T
// Works with both *pkg.DatabaseService and *BaseService
func GetSingleRecord[T any](service interface {
	GetSingleRecord(ctx interface{}, tableName string, query dbModels.QueryParams, errorMsg string) (map[string]interface{}, error)
}, tableName string, query dbModels.QueryParams, errorMsg string) (T, error) {
	data, err := service.GetSingleRecord(nil, tableName, query, errorMsg)
	if err != nil {
		var zero T
		return zero, err
	}

	var result T
	if err := helpers.MapToStruct(data, &result); err != nil {
		var zero T
		return zero, app_errors.ErrMapToStruct
	}
	return result, nil
}

// GetSingleRecordWithRepo fetches a single record from repository and maps it to type T
func GetSingleRecordWithRepo[T any](repo *pkg.DatabaseService, tableName string, query dbModels.QueryParams, errorMsg string) (T, error) {
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

// ListRecords lists records and maps them to []T
// Works with both *pkg.DatabaseService and *BaseService
func ListRecords[T any](service interface {
	GetMultipleRecords(ctx interface{}, tableName string, query dbModels.QueryParams, errorMsg string) ([]map[string]interface{}, error)
}, tableName string, query dbModels.QueryParams, errorMsg string) ([]T, error) {
	data, err := service.GetMultipleRecords(nil, tableName, query, errorMsg)
	if err != nil {
		return nil, err
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

// ListRecordsWithRepo lists records from repository and maps them to []T
func ListRecordsWithRepo[T any](repo *pkg.DatabaseService, tableName string, query dbModels.QueryParams, errorMsg string) ([]T, error) {
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

// CountRecords returns the total count of records in a table
// Works with both *pkg.DatabaseService and *BaseService
func CountRecords(service interface {
	CountRecords(ctx interface{}, tableName string, errorMsg string) (int64, error)
}, tableName string, errorMsg string) (int64, error) {
	return service.CountRecords(nil, tableName, errorMsg)
}

// CountRecordsWithRepo returns the total count of records in a table from repository
func CountRecordsWithRepo(repo *pkg.DatabaseService, tableName string, errorMsg string) (int64, error) {
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

// CreateSingleFilterQuery helper to create a simple filter query
func CreateSingleFilterQuery(column, operator, value string, limit int) dbModels.QueryParams {
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

// AddFilter adds a filter to an existing QueryParams if value is not empty
func AddFilter(query *dbModels.QueryParams, column, operator, value string) {
	if value != "" {
		query.Filters = append(query.Filters, dbModels.QueryFilter{
			Column:   column,
			Operator: operator,
			Value:    value,
		})
	}
}

// CreateMultiFilterQuery creates a query with multiple filters
func CreateMultiFilterQuery(filters []dbModels.QueryFilter, limit, offset *int) dbModels.QueryParams {
	return dbModels.QueryParams{
		Filters: filters,
		Limit:   limit,
		Offset:  offset,
	}
}

// MapToStructList converts a slice of maps to a slice of type T
func MapToStructList[T any](data []map[string]interface{}) ([]T, error) {
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
