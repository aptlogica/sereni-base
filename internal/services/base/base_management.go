// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/aptlogica/go-postgres-rest/pkg"
	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"

	"github.com/google/uuid"
)

type baseManagementService struct {
	repo                   *pkg.DatabaseService
	baseService            interfaces.BaseService
	tableManagementService interfaces.TableManagementService
	modelService           interfaces.ModelService
	assetManagementService interfaces.AssetManagementService
}

func NewBaseManagementService(
	repo *pkg.DatabaseService,
	baseService interfaces.BaseService,
	tableManagementService interfaces.TableManagementService,
	modelService interfaces.ModelService,
	assetManagementService interfaces.AssetManagementService,
) interfaces.BaseManagementService {
	return &baseManagementService{
		repo:                   repo,
		baseService:            baseService,
		tableManagementService: tableManagementService,
		modelService:           modelService,
		assetManagementService: assetManagementService,
	}
}

// insertBase is a helper method that handles the common base insertion logic
func (s baseManagementService) insertBase(ctx context.Context, req dto.CreateBaseRequest, schemaName string, userId string) (tenant.Base, error) {
	if req.CreatedBy == "" {
		req.CreatedBy = userId
	}
	id, err := uuid.Parse(req.WorkspaceID)
	if err != nil {
		return tenant.Base{}, app_errors.InvalidPayload
	}
	insertionReq := dto.BaseInsertion{
		WorkspaceID: id,
		Title:       req.Title,
		Description: req.Description,
		CreatedBy:   req.CreatedBy,
		UpdatedBy:   req.CreatedBy,
	}
	return s.baseService.BaseInsertion(ctx, insertionReq, schemaName)
}

func (s baseManagementService) CreateBase(ctx context.Context, req dto.CreateBaseRequest, schemaName string, userId string) (tenant.Base, error) {
	insertedBase, err := s.insertBase(ctx, req, schemaName, userId)
	if err != nil {
		return tenant.Base{}, err
	}

	id, _ := uuid.Parse(req.WorkspaceID)
	tableInsertionData := dto.CreateTableRequest{
		BaseID:      insertedBase.ID.String(),
		WorkspaceID: id.String(),
		Title:       "Default Table",
		Description: "",
		OrderIndex:  0,
		CreatedBy:   req.CreatedBy,
	}

	_, err = s.tableManagementService.CreateTableWithDefaults(ctx, tableInsertionData, schemaName)
	if err != nil {
		return tenant.Base{}, err
	}

	return insertedBase, nil
}

func (s baseManagementService) CreateBaseWithoutTable(ctx context.Context, req dto.CreateBaseRequest, schemaName string, userId string) (tenant.Base, error) {
	return s.insertBase(ctx, req, schemaName, userId)
}

func (s baseManagementService) CreateBaseWithImage(ctx context.Context, req dto.CreateBaseRequest, schemaName string, userId string, fileHeader *multipart.FileHeader) (tenant.Base, error) {
	// First create the base
	insertedBase, err := s.CreateBase(ctx, req, schemaName, userId)
	if err != nil {
		return tenant.Base{}, err
	}

	// If image file is provided, upload it
	if fileHeader != nil {
		filename := fileHeader.Filename
		ext := strings.ToLower(filepath.Ext(filename))
		allowedExtensions := map[string]bool{
			".jpg":  true,
			".jpeg": true,
			".png":  true,
		}
		if !allowedExtensions[ext] {
			return insertedBase, nil // Return base without image if extension not allowed
		}

		uploadReq := dto.UploadAssetRequest{
			Files: []*multipart.FileHeader{fileHeader},
		}
		assets, err := s.assetManagementService.Upload(ctx, uploadReq, schemaName)
		if err != nil {
			return tenant.Base{}, err
		}
		if len(assets) == 0 {
			return tenant.Base{}, app_errors.StorageUploadFailed
		}
		imagePath := assets[0].Url

		updateReq := dto.BaseUpdate{
			Image:     &imagePath,
			UpdatedBy: userId,
		}

		updatedBase, err := s.baseService.UpdateBase(ctx, schemaName, insertedBase.ID.String(), updateReq)
		if err != nil {
			return insertedBase, nil // Return base without image if update fails
		}

		return updatedBase, nil
	}

	return insertedBase, nil
}

func (s baseManagementService) GetBaseByID(ctx context.Context, schemaName string, id string) (tenant.Base, error) {
	base, err := s.baseService.GetBaseByID(ctx, schemaName, id)
	if err != nil {
		return tenant.Base{}, err
	}
	return base, nil
}

func (s baseManagementService) GetAllBasesWithAccess(ctx context.Context, schemaName string, workspaceMember *tenant.WorkspaceMember) ([]tenant.Base, error) {
	if workspaceMember.BasesIds == "*" {
		return s.GetBasesByWorkspace(ctx, schemaName, workspaceMember.WorkspaceID)
	} else {
		baseIDs := strings.Split(workspaceMember.BasesIds, ",")
		for i := range baseIDs {
			baseIDs[i] = strings.TrimSpace(baseIDs[i])
		}
		return s.baseService.GetBulkbases(ctx, schemaName, baseIDs)
	}
}

func (s baseManagementService) GetAllBases(ctx context.Context, schemaName string, workspaceId string) ([]tenant.Base, error) {
	return s.baseService.GetBasesByWorkspace(ctx, schemaName, workspaceId)
}

func (s baseManagementService) UpdateBase(ctx context.Context, schemaName string, id string, req dto.BaseUpdate, userId string, fileHeader *multipart.FileHeader, removeImage string) (tenant.Base, error) {
	if req.UpdatedBy == "" {
		req.UpdatedBy = userId
	}
	// First update base fields
	updatedBase, err := s.baseService.UpdateBase(ctx, schemaName, id, req)
	if err != nil {
		return tenant.Base{}, err
	}

	// If image file provided, handle upload and update
	if fileHeader != nil {
		updatedBase, err = s.AddBaseImage(ctx, schemaName, id, fileHeader, userId)
		if err != nil {
			return tenant.Base{}, err
		}
		return updatedBase, nil
	}

	// If remove image requested, handle removal
	if removeImage == "true" {
		updatedBase, err = s.RemoveBaseImage(ctx, schemaName, id, userId)
		if err != nil {
			return tenant.Base{}, err
		}
		return updatedBase, nil
	}

	return updatedBase, nil
}

func (s baseManagementService) DeleteBase(ctx context.Context, schemaName string, id string) error {
	err := s.baseService.DeleteBase(ctx, schemaName, id)
	if err != nil {
		return err
	}
	return nil
}

func (s baseManagementService) GetTablesByBaseId(ctx context.Context, schemaName string, baseID string) ([]dto.TableResponse, error) {
	models, err := s.modelService.GetModelByBaseID(ctx, schemaName, baseID)
	if err != nil {
		return nil, err
	}

	var tableResponses []dto.TableResponse
	for _, m := range models {
		var modelResponse dto.ModelResponse
		if err := helpers.StructToStruct(m, &modelResponse); err != nil {
			return nil, app_errors.ErrStructToStruct
		}
		tableResponses = append(tableResponses, dto.TableResponse{
			Model: modelResponse,
		})
	}

	return tableResponses, nil
}

func (s baseManagementService) GetBasesByWorkspace(ctx context.Context, schemaName string, workspaceID string) ([]tenant.Base, error) {
	return s.baseService.GetBasesByWorkspace(ctx, schemaName, workspaceID)
}

func (s baseManagementService) AddBaseImage(ctx context.Context, schema string, baseID string, fileHeader *multipart.FileHeader, userId string) (tenant.Base, error) {
	// Delete existing image if any
	err := s.deleteBaseImageIfExists(ctx, schema, baseID)
	if err != nil {
		return tenant.Base{}, err
	}

	if fileHeader == nil {
		return tenant.Base{}, app_errors.InvalidPayload
	}

	filename := fileHeader.Filename
	ext := strings.ToLower(filepath.Ext(filename))
	allowedExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
	}
	if !allowedExtensions[ext] {
		return tenant.Base{}, app_errors.InvalidPayload
	}

	uploadReq := dto.UploadAssetRequest{
		Files: []*multipart.FileHeader{fileHeader},
	}
	assets, err := s.assetManagementService.Upload(ctx, uploadReq, schema)
	if err != nil {
		return tenant.Base{}, err
	}
	if len(assets) == 0 {
		return tenant.Base{}, app_errors.StorageUploadFailed
	}
	imagePath := assets[0].Url

	updateReq := dto.BaseUpdate{
		Image:     &imagePath,
		UpdatedBy: userId,
	}

	updatedBase, err := s.baseService.UpdateBase(ctx, schema, baseID, updateReq)
	if err != nil {
		return tenant.Base{}, err
	}

	return updatedBase, nil
}

func (s baseManagementService) deleteBaseImageIfExists(ctx context.Context, schema string, baseID string) error {
	base, err := s.baseService.GetBaseByID(ctx, schema, baseID)
	if err != nil {
		return err
	}

	if base.Image != "" {
		asset, err := s.assetManagementService.GetAssetByURL(ctx, schema, base.Image)
		if err == nil {
			if asset.Url == base.Image {
				imageAssetId := asset.ID.String()
				err = s.assetManagementService.DeleteAsset(ctx, imageAssetId, schema)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (s baseManagementService) RemoveBaseImage(ctx context.Context, schema string, baseID string, userId string) (tenant.Base, error) {
	err := s.deleteBaseImageIfExists(ctx, schema, baseID)
	if err != nil {
		return tenant.Base{}, err
	}

	emptyImage := ""
	updateReq := dto.BaseUpdate{
		Image:     &emptyImage,
		UpdatedBy: userId,
	}

	updatedBase, err := s.baseService.UpdateBase(ctx, schema, baseID, updateReq)
	if err != nil {
		return tenant.Base{}, err
	}

	return updatedBase, nil
}

// RemoveUserFromBase removes a user from a base by deleting their access_members record
func (s baseManagementService) RemoveUserFromBase(ctx context.Context, schemaName string, baseID string, userID string) error {
	// Query access_members table to find the record for this user and base
	tableName := fmt.Sprintf("\"%s\".access_members", schemaName)

	params := dbModels.QueryParams{
		Select: []string{"id"},
		Filters: []dbModels.QueryFilter{
			{
				Column:   "user_id",
				Operator: "eq",
				Value:    userID,
			},
			{
				Column:   "scope_type",
				Operator: "eq",
				Value:    "base",
			},
			{
				Column:   "scope_id",
				Operator: "eq",
				Value:    baseID,
			},
		},
	}

	records, err := s.repo.TableService.GetTableData(tableName, params)
	if err != nil {
		return err
	}

	if len(records) == 0 {
		return app_errors.ErrRecordNotFound
	}

	// Extract the ID from the first record
	accessMemberID := records[0]["id"].(string)

	// Delete the access_members record
	return s.repo.TableService.DeleteRecord(tableName, accessMemberID)
}
