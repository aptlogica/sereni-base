package services

import (
	"context"
	"godbgrest/pkg"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/helpers"
	"strings"

	app_errors "serenibase/internal/app-errors"

	"github.com/google/uuid"
)

type baseManagementService struct {
	repo         *pkg.DatabaseService
	baseService  interfaces.BaseService
	modelService interfaces.ModelService
}

func NewBaseManagementService(
	repo *pkg.DatabaseService,
	baseService interfaces.BaseService,
	modelService interfaces.ModelService,
) interfaces.BaseManagementService {
	return &baseManagementService{
		repo:         repo,
		baseService:  baseService,
		modelService: modelService,
	}
}

func (s baseManagementService) CreateBase(ctx context.Context, req dto.CreateBaseRequest, schemaName string, userId string) (tenant.Base, error) {
	if req.CreatedBy == "" {
		req.CreatedBy = userId
	}
	id, err := uuid.Parse(req.WorkspaceID)
	if err != nil {
		return tenant.Base{}, app_errors.InvalidPayload
	}
	insertionReq := dto.BaseInsertion{
		// Copy fields from req to insertionReq
		WorkspaceID: id,
		Title:       req.Title,
		Description: req.Description,
		// ... copy other fields ...
		CreatedBy:   req.CreatedBy,
		UpdatedBy:   req.CreatedBy,
	}
	insertedBase, err := s.baseService.BaseInsertion(ctx, insertionReq, schemaName)
	if err != nil {
		return tenant.Base{}, err
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

func (s baseManagementService) UpdateBase(ctx context.Context, schemaName string, id string, req dto.BaseUpdate, userId string) (tenant.Base, error) {
	if req.UpdatedBy == "" {
		req.UpdatedBy = userId
	}
	updatedBase, err := s.baseService.UpdateBase(ctx, schemaName, id, req)
	if err != nil {
		return tenant.Base{}, err
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
