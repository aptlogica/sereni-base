// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"sort"

	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	storageinterfaces "github.com/aptlogica/sereni-base/internal/providers/storage/interfaces"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
)

const (
	uidtLinks  = constant.UIDTLinks
	uidtLookup = "lookup"

	entityRoleSource = "source"
	entityRoleTarget = "target"

	linkAction = "link"

	// rowBatchSize is how many rows one insert call carries
	rowBatchSize = 500
)

// rowCreator is implemented by the table service; it validates values per column and sets the audit fields.
type rowCreator interface {
	CreateRowsWithValues(ctx context.Context, schemaName string, modelID string, rowsInput []map[string]interface{}, createdBy string, updatedBy string) ([]dto.RecordResponse, error)
}

type baseTransferService struct {
	baseService       interfaces.BaseManagementService
	tableService      interfaces.TableManagementService
	primaryKeyService interfaces.PrimaryKeyService
	// storage and assets move attachment files; both may be nil, then attachments are not transferred
	storage storageinterfaces.StorageProvider
	assets  interfaces.AssetManagementService
	// constraints reads and creates unique and check constraints; may be nil, then they are not transferred
	constraints interfaces.TableConstraintService
	// automations reads and creates trigger functions and triggers; may be nil, then they are not transferred
	automations interfaces.AutomationService
}

// NewBaseTransferService builds the base export/import service. primaryKeyService may be nil; link cells are then
// resolved by the id column. storage and assets may be nil; files are then not transferred. constraints may be nil;
// constraints are then not transferred. automations may be nil; triggers and functions are then not transferred.
func NewBaseTransferService(
	baseService interfaces.BaseManagementService,
	tableService interfaces.TableManagementService,
	primaryKeyService interfaces.PrimaryKeyService,
	storage storageinterfaces.StorageProvider,
	assets interfaces.AssetManagementService,
	constraints interfaces.TableConstraintService,
	automations interfaces.AutomationService,
) interfaces.BaseTransferService {
	return &baseTransferService{
		baseService:       baseService,
		tableService:      tableService,
		primaryKeyService: primaryKeyService,
		storage:           storage,
		assets:            assets,
		constraints:       constraints,
		automations:       automations,
	}
}

// isSupportedUIDT reports whether the field type can be created again by import.
func isSupportedUIDT(uidt string) bool {
	if uidt == uidtLinks || uidt == uidtLookup {
		return true
	}
	_, ok := constant.UITypeMappings[uidt]
	return ok
}

func orderOf(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

func boolOf(v *bool) bool {
	return v != nil && *v
}

func sortedFields(fields []dto.BaseExportField) []dto.BaseExportField {
	out := append([]dto.BaseExportField(nil), fields...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].OrderIndex < out[j].OrderIndex })
	return out
}
