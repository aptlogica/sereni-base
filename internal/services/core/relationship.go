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
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"
)

type relationshipService struct {
	repo *pkg.DatabaseService
}

func NewRelationshipService(repo *pkg.DatabaseService) interfaces.RelationshipService {
	return &relationshipService{repo: repo}
}

func (s *relationshipService) Create(ctx context.Context, req dto.RelationInsertion, schemaName string) (tenant.Relation, error) {
	tableName := tenant.Relation{}.TableName(schemaName)
	insertedRelationshipData, err := s.repo.TableService.CreateRecord(tableName, req.Map())
	if err != nil {
		return tenant.Relation{}, app_errors.LogDatabaseError(err, "failed to create relation")
	}

	var insertedRelationship tenant.Relation
	if err := helpers.MapToStruct(insertedRelationshipData, &insertedRelationship); err != nil {
		return tenant.Relation{}, app_errors.ErrMapToStruct
	}
	return insertedRelationship, nil
}

func (s *relationshipService) GetRelationByID(ctx context.Context, id string, schemaName string) (tenant.Relation, error) {
	tableName := tenant.Relation{}.TableName(schemaName)
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

	relationsData, err := s.repo.TableService.GetTableData(tableName, query)
	if err != nil {
		return tenant.Relation{}, app_errors.LogDatabaseError(err, "failed to get relation by id")
	}

	if len(relationsData) == 0 {
		return tenant.Relation{}, app_errors.InvalidPayload
	}

	relationData := relationsData[0]
	var relation tenant.Relation
	if err := helpers.MapToStruct(relationData, &relation); err != nil {
		return tenant.Relation{}, app_errors.ErrMapToStruct
	}
	return relation, nil
}

func (s *relationshipService) DeleteRelation(ctx context.Context, relationId string, schemaName string) error {
	tableName := tenant.Relation{}.TableName(schemaName)

	err := s.repo.TableService.DeleteRecord(tableName, relationId)
	if err != nil {
		return app_errors.LogDatabaseError(err, "failed to delete relation")
	}

	return nil
}

func (s *relationshipService) UpdateRelation(ctx context.Context, relationId string, relationData dto.RelationUpdate, schemaName string) (tenant.Relation, error) {
	tableName := tenant.Relation{}.TableName(schemaName)
	updateData := relationData.Map()

	if len(updateData) == 0 {
		return tenant.Relation{}, app_errors.InvalidPayload
	}

	updatedData, err := s.repo.TableService.UpdateRecord(tableName, relationId, updateData)
	if err != nil {
		return tenant.Relation{}, app_errors.LogDatabaseError(err, "failed to update relation")
	}

	var relation tenant.Relation
	if err := helpers.MapToStruct(updatedData, &relation); err != nil {
		return tenant.Relation{}, app_errors.ErrMapToStruct
	}

	return relation, nil
}
