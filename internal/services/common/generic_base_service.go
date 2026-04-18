// Copyright (c) 2026 Aptlogica Technologies Private Limited
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"

	"github.com/aptlogica/go-postgres-rest/pkg"
	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"
)

// GenericBaseService provides common CRUD operations for any model
type GenericBaseService struct {
	repo *pkg.DatabaseService
}

// NewGenericBaseService creates a new generic base service
func NewGenericBaseService(repo *pkg.DatabaseService) *GenericBaseService {
	return &GenericBaseService{repo: repo}
}

// CreateRecord is a generic method for creating any record
func (bs *GenericBaseService) CreateRecord(tableName string, data map[string]interface{}) (map[string]interface{}, error) {
	return bs.repo.TableService.CreateRecord(tableName, data)
}

// GetSingleRecord fetches a single record by query and maps it to type T
func (bs *GenericBaseService) GetSingleRecord(ctx context.Context, tableName string, query dbModels.QueryParams, errorMsg string) (map[string]interface{}, error) {
	data, err := bs.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, errorMsg)
	}

	if len(data) == 0 {
		return nil, app_errors.ErrRecordNotFound
	}

	return data[0], nil
}

// GetMultipleRecords fetches multiple records and returns raw data
func (bs *GenericBaseService) GetMultipleRecords(ctx context.Context, tableName string, query dbModels.QueryParams, errorMsg string) ([]map[string]interface{}, error) {
	data, err := bs.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, errorMsg)
	}
	return data, nil
}

// UpdateRecord updates a record and returns the updated data
func (bs *GenericBaseService) UpdateRecord(tableName string, id interface{}, updateData map[string]interface{}) (map[string]interface{}, error) {
	return bs.repo.TableService.UpdateRecord(tableName, id, updateData)
}

// DeleteRecord deletes a record by ID or filter
func (bs *GenericBaseService) DeleteRecord(tableName string, filter interface{}) error {
	return bs.repo.TableService.DeleteRecord(tableName, filter)
}

// CountRecords returns total count of records matching query
func (bs *GenericBaseService) CountRecords(ctx context.Context, tableName string, errorMsg string) (int64, error) {
	countQuery := dbModels.QueryParams{
		Aggregates: []dbModels.AggregateFunction{
			{
				Function: "COUNT",
				Column:   "id",
				Alias:    "total",
			},
		},
	}

	countData, err := bs.repo.TableService.GetTableData(tableName, countQuery)
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

// CountRecordsWithFilter returns count of records matching specific filters
func (bs *GenericBaseService) CountRecordsWithFilter(ctx context.Context, tableName string, filters []dbModels.QueryFilter, errorMsg string) (int64, error) {
	countQuery := dbModels.QueryParams{
		Filters: filters,
		Aggregates: []dbModels.AggregateFunction{
			{
				Function: "COUNT",
				Column:   "id",
				Alias:    "total",
			},
		},
	}

	countData, err := bs.repo.TableService.GetTableData(tableName, countQuery)
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

// MapToStruct converts map data to struct type T
func (bs *GenericBaseService) MapToStruct(data map[string]interface{}, result interface{}) error {
	if err := helpers.MapToStruct(data, result); err != nil {
		return app_errors.ErrMapToStruct
	}
	return nil
}

// MapToStructList converts slice of maps to slice of type T
func (bs *GenericBaseService) MapToStructList(data []map[string]interface{}, results interface{}) error {
	resultSlice := make([]interface{}, 0, len(data))
	for _, item := range data {
		var result interface{}
		if err := helpers.MapToStruct(item, &result); err != nil {
			return app_errors.ErrMapToStruct
		}
		resultSlice = append(resultSlice, result)
	}
	return helpers.MapToStruct(map[string]interface{}{"items": resultSlice}, results)
}

// GetRepository returns the underlying database service
func (bs *GenericBaseService) GetRepository() *pkg.DatabaseService {
	return bs.repo
}
