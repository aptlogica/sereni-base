// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"database/sql"
	"fmt"
	"mime/multipart"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aptlogica/go-postgres-rest/pkg"
	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"

	"github.com/google/uuid"
)

type tableManagementService struct {
	driver                 string
	repo                   *pkg.DatabaseService
	modelService           interfaces.ModelService
	columnsService         interfaces.ColumnService
	viewService            interfaces.ViewService
	relationshipService    interfaces.RelationshipService
	assetManagementService interfaces.AssetManagementService
}

const (
	SchemaTableFormat     = "\"%s\".\"%s\""
	QuotedColumnFormat    = "\"%s\""
	ErrConvertViewStruct  = "Failed to convert view struct"
	columnActionBatchSize = 1000
	dateOutputLayout      = "2006-01-02"
)

// relationRecordParams holds parameters for creating relation record
type relationRecordParams struct {
	BaseID          uuid.UUID
	RelationID      uuid.UUID
	SourceModelData tenant.Model
	SourceColumn    tenant.Column
	TargetModelData tenant.Model
	TargetColumn    tenant.Column
	RelationType    string
	Now             time.Time
}

// targetColumnParams holds parameters for creating target column in relation
type targetColumnParams struct {
	ColumnData      dto.AddColumnRequest
	SourceModelData tenant.Model
	RelationWith    string
	RelationID      uuid.UUID
	RelationType    string
	InverseTitle    string
	Now             time.Time
}

func NewTableManagementService(
	driver string,
	repo *pkg.DatabaseService,
	modelService interfaces.ModelService,
	columnsService interfaces.ColumnService,
	viewService interfaces.ViewService,
	relationshipService interfaces.RelationshipService,
	assetManagementService interfaces.AssetManagementService,
) interfaces.TableManagementService {
	return &tableManagementService{
		driver:                 driver,
		repo:                   repo,
		modelService:           modelService,
		columnsService:         columnsService,
		viewService:            viewService,
		relationshipService:    relationshipService,
		assetManagementService: assetManagementService,
	}
}

func (s tableManagementService) createTableWithDefaultsInDB(schemaName string, tableName string) ([]dto.AddColumnRequest, error) {
	columnsData := constant.SystemColumns
	var columnsDefinitionParams []dbModels.ColumnDefinition
	for _, col := range columnsData {
		columnsDefinitionParams = append(columnsDefinitionParams, dbModels.ColumnDefinition{
			Name:     helpers.ToSnakeCase(col.Title),
			DataType: col.DT,
		})
	}

	creationReq := dbModels.CreateTableRequest{
		Name:       fmt.Sprintf(SchemaTableFormat, schemaName, tableName),
		Columns:    columnsDefinitionParams,
		PrimaryKey: []string{"id"},
	}

	err := s.repo.TableService.CreateTable(creationReq)
	if err != nil {
		return []dto.AddColumnRequest{}, app_errors.LogDatabaseError(err, "failed to create table in DB")
	}

	return columnsData, nil
}

func (s tableManagementService) createDefaultView(ctx context.Context, schemaName string, tableData tenant.Model) (dto.ViewResponse, error) {

	viewData := dto.CreateViewRequest{
		ModelID:     tableData.ID,
		BaseID:      tableData.BaseID,
		Title:       "Default Grid View",
		Description: "",
		Type:        "grid",
		OrderIndex:  helpers.Float64Ptr(0),
		Meta:        &map[string]interface{}{},
		CreatedBy:   tableData.CreatedBy,
	}

	return s.CreateView(ctx, schemaName, viewData)
}

func (s tableManagementService) insertSystemColumns(schemaName string, tableData tenant.Model, columnsData []dto.AddColumnRequest) ([]dto.ColumnResponse, error) {
	var colDataList []dto.ColumnInsertion
	now := time.Now().UTC()
	for index, column := range columnsData {
		// Use the System value from the column definition, default to true if not specified
		systemValue := true
		if column.System != nil {
			systemValue = *column.System
		}

		colData := dto.ColumnInsertion{
			ID:          uuid.New(),
			ModelID:     tableData.ID,
			BaseID:      tableData.BaseID,
			ColumnName:  helpers.ToSnakeCase(column.Title),
			Title:       column.Title,
			UIDT:        column.UIDT,
			DT:          &column.DT,
			Description: helpers.StringPtr(column.Description),
			Meta:        map[string]interface{}{},
			Virtual:     true,
			System:      systemValue,
			Deleted:     false,
			OrderIndex:  helpers.Float64Ptr(float64(index)),
			CreatedAt:   now,
			UpdatedAt:   now,
			CreatedBy:   tableData.CreatedBy,
			UpdatedBy:   tableData.CreatedBy,
		}
		colDataList = append(colDataList, colData)
	}

	insertedColumns, err := s.columnsService.BulkInsert(colDataList, schemaName)
	if err != nil {
		return []dto.ColumnResponse{}, err
	}

	var columnResponses []dto.ColumnResponse
	for _, col := range insertedColumns {
		var colResp dto.ColumnResponse
		if err := helpers.StructToStruct(col, &colResp); err != nil {
			return []dto.ColumnResponse{}, err
		}
		columnResponses = append(columnResponses, colResp)
	}
	return columnResponses, nil

}

func (s tableManagementService) CreateTableWithDefaultsImport(ctx context.Context, tableData dto.CreateTableRequest, schemaName string) (dto.TableResponse, error) {
	insertedModel, err := s.createModel(ctx, tableData, schemaName)
	if err != nil {
		return dto.TableResponse{}, err
	}

	columnsResponse, err := s.setupSystemColumns(ctx, schemaName, insertedModel)
	if err != nil {
		return dto.TableResponse{}, err
	}

	viewResponse, err := s.createDefaultView(ctx, schemaName, insertedModel)
	if err != nil {
		return dto.TableResponse{}, err
	}

	recordsData, err := s.GetAllRecords(ctx, schemaName, insertedModel.ID.String())
	if err != nil {
		return dto.TableResponse{}, err
	}

	modelResponse := s.convertModelToResponse(insertedModel)

	// Add import metadata and log
	importMeta := map[string]interface{}{
		"imported_at":   time.Now().UTC(),
		"import_source": "import_service",
	}
	fmt.Println("Table imported with metadata:", importMeta)

	tableResponse := dto.TableResponse{
		Model:   modelResponse,
		Columns: columnsResponse,
		Views: []dto.ViewResponse{
			viewResponse,
		},
		Records: recordsData.Records,
	}

	return tableResponse, nil
}

func (s tableManagementService) CreateTableWithDefaults(ctx context.Context, tableData dto.CreateTableRequest, schemaName string) (dto.TableResponse, error) {
	insertedModel, err := s.createModel(ctx, tableData, schemaName)
	if err != nil {
		return dto.TableResponse{}, err
	}

	columnsResponse, err := s.setupSystemColumns(ctx, schemaName, insertedModel)
	if err != nil {
		return dto.TableResponse{}, err
	}

	viewResponse, err := s.createDefaultView(ctx, schemaName, insertedModel)
	if err != nil {
		return dto.TableResponse{}, err
	}

	recordsData, err := s.GetAllRecords(ctx, schemaName, insertedModel.ID.String())
	if err != nil {
		return dto.TableResponse{}, err
	}

	modelResponse := s.convertModelToResponse(insertedModel)

	tableResponse := dto.TableResponse{
		Model:   modelResponse,
		Columns: columnsResponse,
		Views: []dto.ViewResponse{
			viewResponse,
		},
		Records: recordsData.Records,
	}

	return tableResponse, nil
}

func (s tableManagementService) createModel(ctx context.Context, tableData dto.CreateTableRequest, schemaName string) (tenant.Model, error) {
	modelInsertionData := dto.ModelInsertion{
		ID:               uuid.New().String(),
		BaseID:           tableData.BaseID,
		WorkspaceID:      tableData.WorkspaceID,
		Title:            tableData.Title,
		Description:      tableData.Description,
		Alias:            s.slugify(tableData.Title),
		Type:             "table",
		Meta:             map[string]interface{}{},
		Schema:           schemaName,
		Tags:             "",
		OrderIndex:       tableData.OrderIndex,
		CreatedBy:        tableData.CreatedBy,
		UpdatedBy:        tableData.CreatedBy,
		CreatedTime:      time.Now().UTC(),
		LastModifiedTime: time.Now().UTC(),
	}

	insertedModel, err := s.modelService.Create(ctx, modelInsertionData, schemaName)
	if err != nil {
		return tenant.Model{}, err
	}

	return insertedModel, nil
}

func (s tableManagementService) setupSystemColumns(ctx context.Context, schemaName string, model tenant.Model) ([]dto.ColumnResponse, error) {
	systemColumns, err := s.createTableWithDefaultsInDB(schemaName, model.Alias)
	if err != nil {
		return []dto.ColumnResponse{}, err
	}

	columnsResponse, err := s.insertSystemColumns(schemaName, model, systemColumns)
	if err != nil {
		return []dto.ColumnResponse{}, err
	}

	return columnsResponse, nil
}

func (s tableManagementService) convertModelToResponse(model tenant.Model) dto.ModelResponse {
	var modelResponse dto.ModelResponse
	helpers.StructToStruct(model, &modelResponse)
	return modelResponse
}

func (s tableManagementService) UpdateTable(ctx context.Context, id string, tableData dto.UpdateTableRequest, schemaName string) (dto.TableResponse, error) {

	var modelData dto.UpdateModelRequest
	if err := helpers.StructToStruct(tableData, &modelData); err != nil {
		return dto.TableResponse{}, app_errors.ErrStructToStruct
	}

	if tableData.UpdatedBy != "" {
		modelData.UpdatedBy = tableData.UpdatedBy
	}

	updatedModel, err := s.modelService.Update(ctx, schemaName, id, modelData)
	if err != nil {
		return dto.TableResponse{}, err
	}

	var modelResponse dto.ModelResponse
	if err := helpers.StructToStruct(updatedModel, &modelResponse); err != nil {
		return dto.TableResponse{}, app_errors.ErrStructToStruct
	}

	tableResponse := dto.TableResponse{
		Model: modelResponse,
	}

	return tableResponse, nil
}

func (s tableManagementService) GetTableByID(ctx context.Context, id string, schemaName string) (dto.TableResponse, error) {
	model, err := s.modelService.GetModelByID(ctx, schemaName, id)
	if err != nil {
		return dto.TableResponse{}, err
	}

	var modelResponse dto.ModelResponse
	if err := helpers.StructToStruct(model, &modelResponse); err != nil {
		return dto.TableResponse{}, app_errors.ErrStructToStruct
	}

	columnsData, err := s.GetColumnsByModelID(ctx, schemaName, id)
	if err != nil {
		return dto.TableResponse{}, err
	}

	viewsData, err := s.GetViewsByModelID(ctx, schemaName, id)
	if err != nil {
		return dto.TableResponse{}, err
	}

	recordsData, err := s.GetRecordsWithLookups(ctx, schemaName, model.Alias, columnsData)
	if err != nil {
		return dto.TableResponse{}, err
	}

	tableResponse := dto.TableResponse{
		Model:   modelResponse,
		Columns: columnsData,
		Views:   viewsData,
		Records: recordsData.Records,
	}

	return tableResponse, nil
}

func (s tableManagementService) GetAllTables(ctx context.Context, schemaName string) ([]dto.TableResponse, error) {
	models, err := s.modelService.GetAllModels(ctx, schemaName)
	if err != nil {
		return nil, err
	}

	var tableResponses []dto.TableResponse
	for _, model := range models {
		var modelResponse dto.ModelResponse
		if err := helpers.StructToStruct(model, &modelResponse); err != nil {
			return nil, app_errors.ErrStructToStruct
		}
		tableResponses = append(tableResponses, dto.TableResponse{
			Model: modelResponse,
		})
	}

	return tableResponses, nil
}

func (s tableManagementService) GetModelByBaseID(ctx context.Context, schemaName string, baseID string) ([]dto.TableResponse, error) {
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

func (s tableManagementService) GetModelByWorkspaceID(ctx context.Context, schemaName string, workspaceID string) ([]dto.TableResponse, error) {
	models, err := s.modelService.GetModelByWorkspaceID(ctx, schemaName, workspaceID)
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

func (s tableManagementService) deleteTableInDB(ctx context.Context, schemaName string, tableName string) error {
	err := s.repo.TableService.DropTable(ctx, fmt.Sprintf(SchemaTableFormat, schemaName, tableName))
	if err != nil {
		return app_errors.LogDatabaseError(err, "failed to drop table")
	}
	return nil
}

func (s tableManagementService) DeleteTable(
	ctx context.Context,
	schemaName string,
	modelID string,
) error {
	model, err := s.modelService.GetModelByID(ctx, schemaName, modelID)
	if err != nil {
		return app_errors.TableNotFound
	}

	if err := s.deleteColumnsForModel(ctx, schemaName, modelID); err != nil {
		return err
	}

	if err := s.deleteViewsForModel(ctx, schemaName, modelID); err != nil {
		return err
	}

	if err := s.modelService.DeleteModel(ctx, schemaName, modelID); err != nil {
		return err
	}

	lg := logger.Get()
	lg.Debug().Str("schemaName", schemaName).Str("tableAlias", model.Alias).Msg("Deleting table from database")

	if err := s.deleteTableInDB(ctx, schemaName, model.Alias); err != nil {
		lg.Error().Stack().Err(err).Str("schemaName", schemaName).Str("tableAlias", model.Alias).Msg("Failed to delete table from database")
		return err
	}

	return nil
}

func (s tableManagementService) deleteColumnsForModel(ctx context.Context, schemaName string, modelID string) error {
	columns, err := s.columnsService.GetColumnByModelID(ctx, schemaName, modelID)
	if err != nil {
		return err
	}
	for _, col := range columns {
		if col.ModelID == modelID {
			// Deleting a link also removes its partner column and its lookups, which may be later in this
			// list (always, for a self-link), so skip columns that are already gone.
			exists, err := s.columnExists(ctx, schemaName, col.ID.String())
			if err != nil {
				return err
			}
			if !exists {
				continue
			}
			if err := s.DeleteColumnForTable(ctx, schemaName, col); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s tableManagementService) deleteViewsForModel(ctx context.Context, schemaName string, modelID string) error {
	views, err := s.viewService.GetViewsByModelID(ctx, schemaName, modelID)
	if err != nil {
		return err
	}
	for _, view := range views {
		if view.ModelID == modelID {
			if err := s.viewService.DeleteView(ctx, schemaName, view.ID.String()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s tableManagementService) slugify(input string) string {
	// Replace spaces with underscores
	slug := strings.ReplaceAll(input, " ", "_")
	// Remove special characters, keeping only letters, numbers, and underscores
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]+`)
	slug = reg.ReplaceAllString(slug, "")
	// Ensure it starts with a letter or underscore
	if slug == "" || (slug[0] >= '0' && slug[0] <= '9') {
		slug = "table_" + slug
	}
	slug = strings.ToLower(slug)
	timestamp := time.Now().Unix()
	// Random suffix keeps names unique when two columns with the same title are
	// created in the same second (e.g. both sides of a self-link).
	return slug + "_" + fmt.Sprintf("%d", timestamp) + "_" + uuid.NewString()[:4]
}

func (s tableManagementService) AddColumnInTableDb(schemaName string, tableName string, columnData tenant.Column) error {
	schematableName := fmt.Sprintf(SchemaTableFormat, schemaName, tableName)

	addColumnReq := dbModels.AddColumnRequest{
		Column: dbModels.ColumnDefinition{
			Name:     fmt.Sprintf(QuotedColumnFormat, columnData.ColumnName),
			DataType: *columnData.DT,
		},
	}

	err := s.repo.TableService.AddColumn(schematableName, addColumnReq)
	if err != nil {
		return app_errors.LogDatabaseError(err, "failed to add column in DB")
	}
	return nil
}

func (s tableManagementService) getDataBaseType(uidt string) (string, error) {
	lg := logger.Get()
	lg.Debug().Str("uidt", uidt).Msg("Getting database type for UIDT")

	mapping, exists := constant.UITypeMappings[uidt]
	if !exists {
		lg.Warn().Str("uidt", uidt).Msg("UIDT not found in mappings")
		return "", app_errors.InvalidUIDT
	}

	switch s.driver {
	case "postgres":
		return mapping.Postgres, nil
	case "sqlite":
		return mapping.SQLite, nil
	default:
		return "", app_errors.InvalidDriver
	}
}

// implement it using struct
func (s tableManagementService) validateMetaForLink(meta map[string]interface{}) (string, string, bool) {
	if meta == nil {
		return "", "", false
	}
	relation, ok := meta["relation"].(map[string]interface{})
	if !ok {
		return "", "", false
	}
	withStr, ok := relation["with"].(string)
	if !ok {
		return "", "", false
	}
	if uuid.Validate(withStr) != nil {
		return "", "", false
	}
	rType, ok := relation["type"].(string)
	if !ok {
		return "", "", false
	}
	switch rType {
	case "many-to-many", "has-many", "one-to-one":
		// valid
	default:
		return "", "", false
	}

	return rType, withStr, true
}

func (s tableManagementService) validateMetaForLookup(meta map[string]interface{}) (string, string, bool) {
	if meta == nil {
		return "", "", false
	}
	lookupColumnID, ok := meta["lookup_column_id"].(string)
	if !ok || uuid.Validate(lookupColumnID) != nil {
		return "", "", false
	}
	relationID, ok := meta["relation_id"].(string)
	if !ok || uuid.Validate(relationID) != nil {
		return "", "", false
	}
	return lookupColumnID, relationID, true
}

// 	s.AddColumnInTableDb(schemaName, trgTable.Alias)
// 	// create column in target table (alter table)
// 	// entry in columns table
// 	// entry in relationship table
// }

func (s tableManagementService) AddColumn(
	ctx context.Context,
	schemaName string,
	columnData dto.AddColumnRequest,
) (dto.ColumnResponse, error) {
	var meta map[string]interface{}
	if columnData.Meta != nil {
		meta = columnData.Meta
	} else {
		meta = make(map[string]interface{})
	}

	if columnData.UIDT == "links" {
		return s.addColumnWithRelation(ctx, schemaName, columnData, meta)
	}
	if columnData.UIDT == "lookup" {
		return s.addColumnWithLookup(ctx, schemaName, columnData)
	}

	dt, err := s.getDataBaseType(columnData.UIDT)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	now := time.Now().UTC()
	ColumnCreatedata := dto.ColumnInsertion{
		ID:          uuid.New(),
		ModelID:     columnData.ModelID,
		BaseID:      columnData.BaseID,
		Title:       columnData.Title,
		ColumnName:  s.slugify(columnData.Title),
		Description: &columnData.Description,
		Meta:        meta,
		UIDT:        columnData.UIDT,
		DT:          helpers.StringPtr(dt),
		Virtual:     columnData.Virtual != nil && *columnData.Virtual,
		System:      columnData.System != nil && *columnData.System,
		Deleted:     false,
		OrderIndex:  columnData.OrderIndex,
		CreatedBy:   columnData.CreatedBy,
		UpdatedBy:   columnData.CreatedBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	column, err := s.columnsService.Create(ctx, ColumnCreatedata, schemaName)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	model, err := s.modelService.GetModelByID(ctx, schemaName, column.ModelID)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	err = s.AddColumnInTableDb(schemaName, model.Alias, column)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	var columnResponse dto.ColumnResponse
	if err := helpers.StructToStruct(column, &columnResponse); err != nil {
		lg := logger.Get()
		lg.Error().Stack().Err(err).Msg("Failed to convert column struct to response")
		return dto.ColumnResponse{}, app_errors.ErrStructToStruct
	}

	return columnResponse, nil
}

func (s tableManagementService) addColumnWithRelation(
	ctx context.Context,
	schemaName string,
	columnData dto.AddColumnRequest,
	sourceMeta map[string]interface{},
) (dto.ColumnResponse, error) {
	relationType, relationWith, ok := s.validateMetaForLink(columnData.Meta)
	if !ok {
		return dto.ColumnResponse{}, app_errors.InvalidColumnMetaForLinkType
	}

	relationId := uuid.New()
	now := time.Now().UTC()
	isSelf := relationWith == columnData.ModelID.String()
	inverseTitle := s.extractInverseTitle(sourceMeta)

	sourcColumn, sourceModelData, err := s.createSourceColumnForRelation(ctx, schemaName, columnData, sourceMeta, relationId, relationType, now)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	// A self-link is a single one-way column: the relation points at the same column on both sides.
	targetColumn, targetModelData := sourcColumn, sourceModelData
	if !isSelf {
		targetColumn, targetModelData, err = s.createTargetColumnForRelation(ctx, schemaName, targetColumnParams{
			ColumnData:      columnData,
			SourceModelData: sourceModelData,
			RelationWith:    relationWith,
			RelationID:      relationId,
			RelationType:    relationType,
			InverseTitle:    inverseTitle,
			Now:             now,
		})
		if err != nil {
			s.rollbackLinkColumn(ctx, schemaName, sourcColumn, sourceModelData.Alias)
			return dto.ColumnResponse{}, err
		}
	}

	if err := s.createRelationRecord(ctx, schemaName, relationRecordParams{
		BaseID:          columnData.BaseID,
		RelationID:      relationId,
		SourceModelData: sourceModelData,
		SourceColumn:    sourcColumn,
		TargetModelData: targetModelData,
		TargetColumn:    targetColumn,
		RelationType:    relationType,
		Now:             now,
	}); err != nil {
		if !isSelf {
			s.rollbackLinkColumn(ctx, schemaName, targetColumn, targetModelData.Alias)
		}
		s.rollbackLinkColumn(ctx, schemaName, sourcColumn, sourceModelData.Alias)
		return dto.ColumnResponse{}, err
	}

	var columnResponse dto.ColumnResponse
	if err := helpers.StructToStruct(sourcColumn, &columnResponse); err != nil {
		lg := logger.Get()
		lg.Error().Stack().Err(err).Msg("Failed to convert source column struct to response")
		return dto.ColumnResponse{}, app_errors.ErrStructToStruct
	}
	return columnResponse, nil
}

// extractInverseTitle pops relation.inverse_title from the request meta; it names the partner column
// and is not stored on the source column.
func (s tableManagementService) extractInverseTitle(meta map[string]interface{}) string {
	relation, ok := meta["relation"].(map[string]interface{})
	if !ok {
		return ""
	}
	title, _ := relation["inverse_title"].(string)
	delete(relation, "inverse_title")
	return strings.TrimSpace(title)
}

// rollbackLinkColumn removes a link column created earlier in a failed link creation.
func (s tableManagementService) rollbackLinkColumn(ctx context.Context, schemaName string, column tenant.Column, tableAlias string) {
	lg := logger.Get()
	if err := s.columnsService.DeleteColumn(ctx, schemaName, column.ID.String()); err != nil {
		lg.Error().Err(err).Str("columnID", column.ID.String()).Msg("Failed to roll back link column metadata")
	}
	if err := s.removeColumnInTableDb(schemaName, tableAlias, column.ColumnName); err != nil {
		lg.Error().Err(err).Str("column", column.ColumnName).Msg("Failed to roll back link column")
	}
}

func (s tableManagementService) createSourceColumnForRelation(
	ctx context.Context,
	schemaName string,
	columnData dto.AddColumnRequest,
	sourceMeta map[string]interface{},
	relationId uuid.UUID,
	relationType string,
	now time.Time,
) (tenant.Column, tenant.Model, error) {
	sourceMeta["entity_role"] = entityRoleSource
	sourceMeta["relation_id"] = relationId
	if relation, ok := sourceMeta["relation"].(map[string]interface{}); ok {
		sourceMeta["is_self_link"] = relation["with"] == columnData.ModelID.String()
	}

	sourceTempUidt := fmt.Sprintf("%s_source_%v", columnData.UIDT, relationType)
	sourceDataType, err := s.getDataBaseType(sourceTempUidt)
	if err != nil {
		return tenant.Column{}, tenant.Model{}, err
	}

	srcColumnCreatedata := dto.ColumnInsertion{
		ID:          uuid.New(),
		ModelID:     columnData.ModelID,
		BaseID:      columnData.BaseID,
		Title:       columnData.Title,
		ColumnName:  s.slugify(columnData.Title),
		Description: &columnData.Description,
		Meta:        sourceMeta,
		UIDT:        columnData.UIDT,
		DT:          helpers.StringPtr(sourceDataType),
		Virtual:     columnData.Virtual != nil && *columnData.Virtual,
		System:      columnData.System != nil && *columnData.System,
		Deleted:     false,
		OrderIndex:  columnData.OrderIndex,
		CreatedBy:   columnData.CreatedBy,
		UpdatedBy:   columnData.CreatedBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	sourcColumn, err := s.columnsService.Create(ctx, srcColumnCreatedata, schemaName)
	if err != nil {
		return tenant.Column{}, tenant.Model{}, err
	}

	sourceModelData, err := s.modelService.GetModelByID(ctx, schemaName, columnData.ModelID.String())
	if err != nil {
		return tenant.Column{}, tenant.Model{}, err
	}

	if err := s.AddColumnInTableDb(schemaName, sourceModelData.Alias, sourcColumn); err != nil {
		return tenant.Column{}, tenant.Model{}, err
	}

	return sourcColumn, sourceModelData, nil
}

func (s tableManagementService) createTargetColumnForRelation(
	ctx context.Context,
	schemaName string,
	params targetColumnParams,
) (tenant.Column, tenant.Model, error) {
	targetMeta := map[string]interface{}{
		"relation": map[string]interface{}{
			"with": params.ColumnData.ModelID.String(),
			"type": params.RelationType,
		},
		"entity_role": entityRoleTarget,
		"relation_id": params.RelationID,
	}

	targetTitle := params.InverseTitle
	if targetTitle == "" {
		targetTitle = params.SourceModelData.Title
	}

	targetTempUidt := fmt.Sprintf("%s_target_%v", params.ColumnData.UIDT, params.RelationType)
	targetDataType, err := s.getDataBaseType(targetTempUidt)
	if err != nil {
		return tenant.Column{}, tenant.Model{}, err
	}

	targetCurrentOrderIndex, err := s.columnsService.GetMaxOrderIndexOfColumn(ctx, schemaName, params.RelationWith)
	if err != nil {
		return tenant.Column{}, tenant.Model{}, err
	}

	targetColumnCreatedata := dto.ColumnInsertion{
		ID:          uuid.New(),
		ModelID:     uuid.MustParse(params.RelationWith),
		BaseID:      params.ColumnData.BaseID,
		Title:       targetTitle,
		ColumnName:  s.slugify(targetTitle),
		Description: helpers.StringPtr(""),
		Meta:        targetMeta,
		UIDT:        params.ColumnData.UIDT,
		DT:          helpers.StringPtr(targetDataType),
		Virtual:     params.ColumnData.Virtual != nil && *params.ColumnData.Virtual,
		System:      params.ColumnData.System != nil && *params.ColumnData.System,
		Deleted:     false,
		OrderIndex:  helpers.Float64Ptr(targetCurrentOrderIndex + 1),
		CreatedBy:   params.ColumnData.CreatedBy,
		UpdatedBy:   params.ColumnData.CreatedBy,
		CreatedAt:   params.Now,
		UpdatedAt:   params.Now,
	}

	targetColumn, err := s.columnsService.Create(ctx, targetColumnCreatedata, schemaName)
	if err != nil {
		return tenant.Column{}, tenant.Model{}, err
	}

	targetModelData, err := s.modelService.GetModelByID(ctx, schemaName, targetColumn.ModelID)
	if err != nil {
		return tenant.Column{}, tenant.Model{}, err
	}

	if err := s.AddColumnInTableDb(schemaName, targetModelData.Alias, targetColumn); err != nil {
		return tenant.Column{}, tenant.Model{}, err
	}

	return targetColumn, targetModelData, nil
}

func (s tableManagementService) createRelationRecord(
	ctx context.Context,
	schemaName string,
	params relationRecordParams,
) error {
	relationInsertionData := dto.RelationInsertion{
		ID:             params.RelationID,
		BaseID:         params.BaseID.String(),
		SourceModelID:  params.SourceModelData.ID.String(),
		SourceColumnID: params.SourceColumn.ID.String(),
		TargetModelID:  params.TargetModelData.ID.String(),
		TargetColumnID: params.TargetColumn.ID.String(),
		RelationType:   params.RelationType,
		CreatedAt:      params.Now,
		UpdatedAt:      params.Now,
	}

	_, err := s.relationshipService.Create(ctx, relationInsertionData, schemaName)
	return err
}

// addLookupColumnInRelation records a lookup's foreign column on the relation side the lookup reads from.
// The side comes from the link column's entity_role, so a self-link (where both sides are the same model)
// updates exactly one array.
func (s tableManagementService) addLookupColumnInRelation(
	ctx context.Context,
	schemaName string,
	entityRole string,
	relationID string,
	lookupColumnName string,
) error {
	relationData, err := s.relationshipService.GetRelationByID(ctx, relationID, schemaName)
	if err != nil {
		lg := logger.Get()
		lg.Debug().Str("relationID", relationID).Str("schemaName", schemaName).Msg("Fetching lookup columns for relation")
		lg.Error().Stack().Err(err).Msg("Failed to get relation by ID")
		return err
	}

	relationUpdation := dto.RelationUpdate{
		UpdatedAt: time.Now().UTC(),
	}

	if entityRole == entityRoleSource {
		relationUpdation.SourceLookupColumns = append(append([]string{}, relationData.SourceLookupColumns...), lookupColumnName)
	} else {
		relationUpdation.TargetLookupColumns = append(append([]string{}, relationData.TargetLookupColumns...), lookupColumnName)
	}

	_, err = s.relationshipService.UpdateRelation(ctx, relationID, relationUpdation, schemaName)
	if err != nil {
		lg := logger.Get()
		lg.Error().Stack().Err(err).Msg("Failed to update relation with lookup columns")
		return err
	}
	return nil
}

func (s tableManagementService) removeLookupColumnInRelation(
	ctx context.Context,
	schemaName string,
	entityRole string,
	relationID string,
	lookupColumnName string,
) error {
	relationData, err := s.relationshipService.GetRelationByID(ctx, relationID, schemaName)
	if err != nil {
		lg := logger.Get()
		lg.Debug().Str("relationID", relationID).Str("schemaName", schemaName).Msg("Fetching lookup columns for relation")
		lg.Error().Stack().Err(err).Msg("Failed to get relation by ID for removal")
		return err
	}

	relationUpdation := dto.RelationUpdate{
		UpdatedAt: time.Now().UTC(),
	}

	if entityRole == entityRoleSource {
		relationUpdation.SourceLookupColumns = s.removeLookupColumnFromList(relationData.SourceLookupColumns, lookupColumnName, "SourceLookupColumns")
	} else {
		relationUpdation.TargetLookupColumns = s.removeLookupColumnFromList(relationData.TargetLookupColumns, lookupColumnName, "TargetLookupColumns")
	}
	_, err = s.relationshipService.UpdateRelation(ctx, relationID, relationUpdation, schemaName)
	if err != nil {
		lg := logger.Get()
		lg.Error().Stack().Err(err).Msg("Failed to update relation with lookup columns")
		return err
	}
	return nil
}

func (s tableManagementService) removeLookupColumnFromList(
	columns []string,
	columnToRemove string,
	columnType string,
) []string {

	lg := logger.Get()
	lg.Debug().
		Str("type", fmt.Sprintf("%T", columns)).
		Msg(fmt.Sprintf("Type of %s", columnType))

	if columns == nil {
		return []string{}
	}

	newArr := make([]string, 0, len(columns))
	removed := false

	for _, col := range columns {
		if col == columnToRemove && !removed {
			removed = true // skip only the first match
			continue
		}
		newArr = append(newArr, col)
	}

	return newArr
}

// lookupResolution is a validated lookup definition: the link column it goes through and the foreign column it shows.
type lookupResolution struct {
	LinkColumn   dto.ColumnResponse
	LookupColumn tenant.Column
	RelationID   string
}

// resolveLookupMeta validates lookup meta for a lookup on modelID and returns the resolved link and foreign columns.
func (s tableManagementService) resolveLookupMeta(
	ctx context.Context,
	schemaName string,
	modelID string,
	meta map[string]interface{},
) (lookupResolution, error) {
	lookupColumnID, relationID, ok := s.validateMetaForLookup(meta)
	if !ok {
		return lookupResolution{}, app_errors.InvalidColumnMetaForLookupType
	}

	linkCol, err := s.resolveLookupLinkColumn(ctx, schemaName, modelID, relationID, metaString(meta, "link_column_id"))
	if err != nil {
		return lookupResolution{}, err
	}

	lookupColumn, err := s.columnsService.GetColumnByID(ctx, schemaName, lookupColumnID)
	if err != nil {
		return lookupResolution{}, err
	}
	if err := validateLookupTarget(linkCol, lookupColumn); err != nil {
		return lookupResolution{}, err
	}

	return lookupResolution{LinkColumn: linkCol, LookupColumn: lookupColumn, RelationID: relationID}, nil
}

// lookupEntityRole returns the side (source/target) an existing lookup column reads through.
func (s tableManagementService) lookupEntityRole(ctx context.Context, schemaName string, lookup dto.ColumnResponse, relationID string) string {
	linkCol, err := s.resolveLookupLinkColumn(ctx, schemaName, lookup.ModelID.String(), relationID, metaString(lookup.Meta, "link_column_id"))
	if err != nil {
		return entityRoleSource
	}
	info, _ := parseLinkMeta(linkCol.Meta)
	return info.EntityRole
}

func (s tableManagementService) addColumnWithLookup(
	ctx context.Context,
	schemaName string,
	columnData dto.AddColumnRequest,
) (dto.ColumnResponse, error) {
	now := time.Now().UTC()

	resolved, err := s.resolveLookupMeta(ctx, schemaName, columnData.ModelID.String(), columnData.Meta)
	if err != nil {
		return dto.ColumnResponse{}, err
	}
	linkInfo, _ := parseLinkMeta(resolved.LinkColumn.Meta)

	meta := columnData.Meta
	meta["link_column_id"] = resolved.LinkColumn.ID.String()

	srcColumnCreatedata := dto.ColumnInsertion{
		ID:          uuid.New(),
		ModelID:     columnData.ModelID,
		BaseID:      columnData.BaseID,
		Title:       columnData.Title,
		ColumnName:  lookupColumnName(resolved.LookupColumn),
		Description: &columnData.Description,
		Meta:        meta,
		UIDT:        columnData.UIDT,
		DT:          helpers.StringPtr(columnData.UIDT),
		Virtual:     columnData.Virtual != nil && *columnData.Virtual,
		System:      columnData.System != nil && *columnData.System,
		Deleted:     false,
		OrderIndex:  columnData.OrderIndex,
		CreatedBy:   columnData.CreatedBy,
		UpdatedBy:   columnData.CreatedBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	insertedColumn, err := s.columnsService.Create(ctx, srcColumnCreatedata, schemaName)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	var columnResponse dto.ColumnResponse
	if err := helpers.StructToStruct(insertedColumn, &columnResponse); err != nil {
		lg := logger.Get()
		lg.Error().Stack().Err(err).Msg("Failed to convert relationship column struct to response")
		return dto.ColumnResponse{}, app_errors.ErrStructToStruct
	}

	if err := s.addLookupColumnInRelation(ctx, schemaName, linkInfo.EntityRole, resolved.RelationID, resolved.LookupColumn.ColumnName); err != nil {
		lg := logger.Get()
		lg.Error().Stack().Err(err).Msg("Failed to add lookup column in relationship")
		return dto.ColumnResponse{}, err
	}
	return columnResponse, nil
}

func (s tableManagementService) GetColumnById(
	ctx context.Context,
	schemaName string,
	id string,
) (dto.ColumnResponse, error) {
	lg := logger.Get()
	column, err := s.columnsService.GetColumnByID(ctx, schemaName, id)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	var columnResponse dto.ColumnResponse
	if err := helpers.StructToStruct(column, &columnResponse); err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to convert struct")
		return dto.ColumnResponse{}, app_errors.ErrStructToStruct
	}

	return columnResponse, nil
}

func (s tableManagementService) GetAllColumns(
	ctx context.Context,
	schemaName string,
) ([]dto.ColumnResponse, error) {
	lg := logger.Get()
	columns, err := s.columnsService.GetAllColumns(ctx, schemaName)
	if err != nil {
		return nil, err
	}

	var columnResponses []dto.ColumnResponse
	for _, column := range columns {
		var columnResponse dto.ColumnResponse
		if err := helpers.StructToStruct(column, &columnResponse); err != nil {
			lg.Error().Stack().Err(err).Msg("Failed to convert column struct")
			return nil, app_errors.ErrStructToStruct
		}
		columnResponses = append(columnResponses, columnResponse)
	}

	return columnResponses, nil
}

func (s tableManagementService) GetColumnsByModelID(
	ctx context.Context,
	schemaName string,
	modelID string,
) ([]dto.ColumnResponse, error) {
	lg := logger.Get()
	columns, err := s.columnsService.GetColumnByModelID(ctx, schemaName, modelID)
	if err != nil {
		return nil, err
	}

	var columnResponses []dto.ColumnResponse
	for _, column := range columns {
		var columnResponse dto.ColumnResponse
		if err := helpers.StructToStruct(column, &columnResponse); err != nil {
			lg.Error().Stack().Err(err).Msg("Failed to convert column struct")
			return nil, app_errors.ErrStructToStruct
		}
		columnResponses = append(columnResponses, columnResponse)
	}
	return columnResponses, nil
}

func (s tableManagementService) CreateView(
	ctx context.Context,
	schemaName string,
	viewData dto.CreateViewRequest,
) (dto.ViewResponse, error) {
	lg := logger.Get()

	viewInserionData := dto.ViewInsertion{
		ID:          uuid.New(),
		ModelID:     viewData.ModelID,
		BaseID:      viewData.BaseID,
		Title:       viewData.Title,
		Description: &viewData.Description,
		Alias:       helpers.StringPtr(s.slugify(viewData.Title)),
		Type:        viewData.Type,
		IsDefault:   false,
		LockType:    helpers.StringPtr(""),
		Password:    helpers.StringPtr(""),
		Public:      false,
		UUID:        helpers.StringPtr(uuid.New().String()),
		Meta:        *viewData.Meta,
		OrderIndex:  viewData.OrderIndex,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		CreatedBy:   viewData.CreatedBy,
		UpdatedBy:   viewData.CreatedBy,
	}

	view, err := s.viewService.Create(ctx, viewInserionData, schemaName)
	if err != nil {
		return dto.ViewResponse{}, err
	}

	var viewResponse dto.ViewResponse
	if err := helpers.StructToStruct(view, &viewResponse); err != nil {
		lg.Error().Stack().Err(err).Msg(ErrConvertViewStruct)
		return dto.ViewResponse{}, app_errors.ErrStructToStruct
	}

	return viewResponse, nil
}

func (s tableManagementService) GetViewByID(
	ctx context.Context,
	schemaName string,
	id string,
) (dto.ViewResponse, error) {
	lg := logger.Get()
	view, err := s.viewService.GetViewByID(ctx, schemaName, id)
	if err != nil {
		return dto.ViewResponse{}, err
	}

	var viewResponse dto.ViewResponse
	if err := helpers.StructToStruct(view, &viewResponse); err != nil {
		lg.Error().Stack().Err(err).Msg(ErrConvertViewStruct)
		return dto.ViewResponse{}, app_errors.ErrStructToStruct
	}

	return viewResponse, nil
}

func (s tableManagementService) GetAllViews(
	ctx context.Context,
	schemaName string,
) ([]dto.ViewResponse, error) {
	lg := logger.Get()
	views, err := s.viewService.GetAllViews(ctx, schemaName)
	if err != nil {
		return nil, err
	}

	viewResponses := make([]dto.ViewResponse, 0, len(views))
	for _, view := range views {
		var viewResponse dto.ViewResponse
		if err := helpers.StructToStruct(view, &viewResponse); err != nil {
			lg.Error().Stack().Err(err).Msg(ErrConvertViewStruct)
			return nil, app_errors.ErrStructToStruct
		}
		viewResponses = append(viewResponses, viewResponse)
	}

	return viewResponses, nil
}

func (s tableManagementService) GetViewsByModelID(
	ctx context.Context,
	schemaName string,
	modelID string,
) ([]dto.ViewResponse, error) {
	lg := logger.Get()
	views, err := s.viewService.GetViewsByModelID(ctx, schemaName, modelID)
	if err != nil {
		return nil, err
	}

	viewResponses := make([]dto.ViewResponse, 0, len(views))
	for _, view := range views {
		var viewResponse dto.ViewResponse
		if err := helpers.StructToStruct(view, &viewResponse); err != nil {
			lg.Error().Stack().Err(err).Msg(ErrConvertViewStruct)
			return nil, app_errors.ErrStructToStruct
		}
		viewResponses = append(viewResponses, viewResponse)
	}

	return viewResponses, nil
}

func (s tableManagementService) UpdateView(
	ctx context.Context,
	schemaName string,
	id string,
	req dto.ViewUpdate,
) (dto.ViewResponse, error) {
	lg := logger.Get()

	if req.UpdatedAt.IsZero() {
		req.UpdatedAt = time.Now().UTC()
	}

	_, err := s.viewService.GetViewByID(ctx, schemaName, id)
	if err != nil {
		return dto.ViewResponse{}, err
	}

	view, err := s.viewService.UpdateView(ctx, schemaName, id, req)
	if err != nil {
		return dto.ViewResponse{}, err
	}

	var viewResponse dto.ViewResponse
	if err := helpers.StructToStruct(view, &viewResponse); err != nil {
		lg.Error().Stack().Err(err).Msg(ErrConvertViewStruct)
		return dto.ViewResponse{}, app_errors.ErrStructToStruct
	}

	return viewResponse, nil
}

func (s tableManagementService) DeleteView(
	ctx context.Context,
	schemaName string,
	id string,
) error {
	_, err := s.viewService.GetViewByID(ctx, schemaName, id)
	if err != nil {
		return err
	}
	return s.viewService.DeleteView(ctx, schemaName, id)
}

func (s tableManagementService) allowUpdate(columnData dto.ColumnResponse) bool {
	if *columnData.System {
		if columnData.ColumnName == "title" {
			return true
		}
		return false
	}
	return true
}

func (s tableManagementService) allowDelete(columnData dto.ColumnResponse) bool {
	if *columnData.System {
		return false
	}
	return true
}
func (s tableManagementService) updateColumnDatatypeInDb(ctx context.Context, schemaName string, tableName string, columnName string, newDataType string, emptyBefore bool) error {
	lg := logger.Get()
	functionName := "convert_column_type"
	schemaFunctionName := fmt.Sprintf("%s.%s", constant.MasterDatabase, functionName)

	args := map[string]interface{}{
		"schema_name":  schemaName,
		"table_name":   tableName,
		"column_name":  columnName,
		"target_type":  newDataType,
		"empty_before": emptyBefore,
	}

	lg.Debug().Interface("args", args).Msg("Converting column datatype")

	_, err := s.repo.TableService.GetByFunction(
		ctx,
		schemaFunctionName,
		args,
	)
	if err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to convert column datatype")
		return err
	}

	return nil
}

func (s tableManagementService) updateColumnForLink(
	ctx context.Context,
	schemaName string,
	columnData dto.ColumnResponse,
	req dto.ColumnUpdate,
) (dto.ColumnResponse, error) {
	// For link columns, only update title, description, last_modified_time and last_modified_by.
	// Type, target and relation type can't change; a new link must be created instead.
	linkUpdateReq := dto.ColumnUpdate{
		Title:       req.Title,
		Description: req.Description,
		UpdatedBy:   req.UpdatedBy,
		UpdatedAt:   req.UpdatedAt,
	}

	updatedColumn, err := s.columnsService.UpdateColumn(ctx, schemaName, columnData.ID.String(), linkUpdateReq)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	if req.Meta != nil {
		if inverseTitle := s.extractInverseTitle(*req.Meta); inverseTitle != "" {
			if err := s.renamePartnerLinkColumn(ctx, schemaName, columnData, inverseTitle, req); err != nil {
				return dto.ColumnResponse{}, err
			}
		}
	}

	var updatedColumnResponse dto.ColumnResponse
	if err := helpers.StructToStruct(updatedColumn, &updatedColumnResponse); err != nil {
		return dto.ColumnResponse{}, app_errors.ErrStructToStruct
	}

	return updatedColumnResponse, nil
}

// renamePartnerLinkColumn sets the title of the other column of a link (the inverse field).
func (s tableManagementService) renamePartnerLinkColumn(
	ctx context.Context,
	schemaName string,
	columnData dto.ColumnResponse,
	title string,
	req dto.ColumnUpdate,
) error {
	info, ok := parseLinkMeta(columnData.Meta)
	if !ok {
		return app_errors.InvalidColumnMetaForLinkType
	}
	relation, err := s.relationshipService.GetRelationByID(ctx, info.RelationID, schemaName)
	if err != nil {
		return err
	}
	if isSingleColumnRelation(relation) {
		return nil // one-way self-link: no inverse column to rename
	}
	partnerColumnID := relation.TargetColumnID
	if info.EntityRole == entityRoleTarget {
		partnerColumnID = relation.SourceColumnID
	}
	_, err = s.columnsService.UpdateColumn(ctx, schemaName, partnerColumnID, dto.ColumnUpdate{
		Title:     helpers.StringPtr(title),
		UpdatedBy: req.UpdatedBy,
		UpdatedAt: req.UpdatedAt,
	})
	return err
}

func (s tableManagementService) updateColumnForLookup(
	ctx context.Context,
	schemaName string,
	columnData dto.ColumnResponse,
	req dto.ColumnUpdate,
) (dto.ColumnResponse, error) {
	lookupUpdateReq := dto.ColumnUpdate{
		Title:       req.Title,
		Description: req.Description,
		UpdatedBy:   req.UpdatedBy,
		UpdatedAt:   req.UpdatedAt,
	}

	// Title/description-only edit: the lookup definition stays as it is.
	if req.Meta != nil {
		newMeta, columnName, err := s.repointLookup(ctx, schemaName, columnData, *req.Meta)
		if err != nil {
			return dto.ColumnResponse{}, err
		}
		lookupUpdateReq.Meta = &newMeta
		lookupUpdateReq.ColumnName = columnName
	}

	updatedColumn, err := s.columnsService.UpdateColumn(ctx, schemaName, columnData.ID.String(), lookupUpdateReq)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	var updatedColumnResponse dto.ColumnResponse
	if err := helpers.StructToStruct(updatedColumn, &updatedColumnResponse); err != nil {
		return dto.ColumnResponse{}, app_errors.ErrStructToStruct
	}

	return updatedColumnResponse, nil
}

// repointLookup validates new lookup meta and, when the link or looked-up column changed, moves the
// lookup's entry between relation arrays. It returns the meta to store and a new column_name if one is needed.
func (s tableManagementService) repointLookup(
	ctx context.Context,
	schemaName string,
	columnData dto.ColumnResponse,
	meta map[string]interface{},
) (map[string]interface{}, *string, error) {
	// Validate everything before touching the relation, so a bad request changes nothing.
	resolved, err := s.resolveLookupMeta(ctx, schemaName, columnData.ModelID.String(), meta)
	if err != nil {
		return nil, nil, err
	}
	meta["link_column_id"] = resolved.LinkColumn.ID.String()

	oldLookupColumnID, oldRelationID, hadOld := s.validateMetaForLookup(columnData.Meta)
	sameLookupColumn := hadOld && oldLookupColumnID == resolved.LookupColumn.ID.String()
	sameLink := sameLookupColumn &&
		oldRelationID == resolved.RelationID &&
		metaString(columnData.Meta, "link_column_id") == resolved.LinkColumn.ID.String()
	if sameLink {
		return meta, nil, nil
	}

	lg := logger.Get()
	if hadOld {
		oldRole := s.lookupEntityRole(ctx, schemaName, columnData, oldRelationID)
		if oldColumn, err := s.columnsService.GetColumnByID(ctx, schemaName, oldLookupColumnID); err == nil {
			if err := s.removeLookupColumnInRelation(ctx, schemaName, oldRole, oldRelationID, oldColumn.ColumnName); err != nil {
				lg.Warn().Err(err).Str("relationID", oldRelationID).Msg("Failed to remove old lookup entry from relation")
			}
		}
	}

	linkInfo, _ := parseLinkMeta(resolved.LinkColumn.Meta)
	if err := s.addLookupColumnInRelation(ctx, schemaName, linkInfo.EntityRole, resolved.RelationID, resolved.LookupColumn.ColumnName); err != nil {
		return nil, nil, err
	}

	if sameLookupColumn {
		return meta, nil, nil
	}
	return meta, helpers.StringPtr(lookupColumnName(resolved.LookupColumn)), nil
}

func (s tableManagementService) UpdateColumn(
	ctx context.Context,
	schemaName string,
	id string,
	req dto.ColumnUpdate,
) (dto.ColumnResponse, error) {
	lg := logger.Get()
	if req.UpdatedAt.IsZero() {
		req.UpdatedAt = time.Now().UTC()
	}

	columnData, err := s.GetColumnById(ctx, schemaName, id)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	req, err = s.sanitizeUpdateRequest(columnData, req)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	if columnData.UIDT == uidtLinks {
		return s.updateColumnForLink(ctx, schemaName, columnData, req)
	}

	if columnData.UIDT == uidtLookup {
		return s.updateColumnForLookup(ctx, schemaName, columnData, req)
	}

	// Link and lookup columns need their relation set up, so a plain column can't be converted into one.
	if req.UIDT != nil && (*req.UIDT == uidtLinks || *req.UIDT == uidtLookup) {
		return dto.ColumnResponse{}, app_errors.UpdateNotAllowed
	}

	if req.UIDT != nil && *req.UIDT != "" {
		dt, _ := s.getDataBaseType(*req.UIDT)
		req.DT = helpers.StringPtr(dt)
	}

	column, err := s.columnsService.UpdateColumn(ctx, schemaName, id, req)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	if err := s.handleDatatypeChangeIfNeeded(ctx, schemaName, id, columnData, column, req); err != nil {
		return dto.ColumnResponse{}, err
	}

	var columnResponse dto.ColumnResponse
	if err := helpers.StructToStruct(column, &columnResponse); err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to convert updated column struct")
		return dto.ColumnResponse{}, app_errors.ErrStructToStruct
	}

	return columnResponse, nil
}

func (s tableManagementService) sanitizeUpdateRequest(columnData dto.ColumnResponse, req dto.ColumnUpdate) (dto.ColumnUpdate, error) {
	if !s.allowUpdate(columnData) {
		if req.Title == nil || strings.Contains(columnData.ColumnName, *req.Title) {
			return dto.ColumnUpdate{}, app_errors.UpdateNotAllowed
		}
		return dto.ColumnUpdate{
			Title:     req.Title,
			UpdatedAt: req.UpdatedAt,
		}, nil
	}
	return req, nil
}

func (s tableManagementService) handleDatatypeChangeIfNeeded(
	ctx context.Context,
	schemaName string,
	id string,
	columnData dto.ColumnResponse,
	column tenant.Column,
	req dto.ColumnUpdate,
) error {
	if !s.shouldUpdateDatatype(req, columnData) {
		return nil
	}

	model, err := s.modelService.GetModelByID(ctx, schemaName, column.ModelID)
	if err != nil {
		return err
	}

	allowed := s.isConversionAllowed(columnData.UIDT, *req.UIDT)

	if err := s.updateColumnDatatypeInDb(ctx, schemaName, model.Alias, column.ColumnName, *req.DT, !allowed); err != nil {
		s.revertColumnMetadata(ctx, schemaName, id, columnData)
		return err
	}

	return nil
}

func (s tableManagementService) shouldUpdateDatatype(req dto.ColumnUpdate, columnData dto.ColumnResponse) bool {
	return (req.UIDT != nil && *req.UIDT != "") && (columnData.DT != *req.DT)
}

func (s tableManagementService) isConversionAllowed(fromUIdt string, toUIdt string) bool {
	if fromUIdt == toUIdt {
		return true
	}

	conversions, ok := constant.AllowedConversions[fromUIdt]
	if !ok {
		return false
	}

	for _, conv := range conversions {
		if conv == toUIdt {
			return true
		}
	}
	return false
}

func (s tableManagementService) revertColumnMetadata(ctx context.Context, schemaName string, id string, columnData dto.ColumnResponse) {
	revertReq := dto.ColumnUpdate{
		DT:   helpers.StringPtr(columnData.DT),
		UIDT: helpers.StringPtr(columnData.UIDT),
	}
	_, _ = s.columnsService.UpdateColumn(ctx, schemaName, id, revertReq)
}

func (s tableManagementService) removeColumnInTableDb(schemaName string, tableName string, columnName string) error {
	schematableName := fmt.Sprintf(SchemaTableFormat, schemaName, tableName)

	addColumnReq := dbModels.AlterTableRequest{
		Action: "drop_column",
		Data: dbModels.DropColumnRequest{
			ColumnName: fmt.Sprintf(QuotedColumnFormat, columnName),
			Cascade:    true,
		},
	}

	err := s.repo.TableService.AlterTable(schematableName, addColumnReq)
	if err != nil {
		return app_errors.LogDatabaseError(err, "failed to drop column in DB")
	}
	return nil
}

func (s tableManagementService) deleteLookups(ctx context.Context, relationId string, modelId string, schemaName string) error {
	columns, err := s.columnsService.GetColumnByModelID(ctx, schemaName, modelId)
	if err != nil {
		return err
	}

	for _, col := range columns {
		// Only lookups that read through this relation; lookups of other links on the table stay.
		if col.UIDT == uidtLookup && metaString(col.Meta, "relation_id") == relationId {
			var columnData dto.ColumnResponse
			if err := helpers.StructToStruct(col, &columnData); err != nil {
				return app_errors.ErrStructToStruct
			}

			err := s.columnsService.DeleteColumn(ctx, schemaName, col.ID.String())
			if err != nil {
				return err
			}

			err = s.reorderColumnsAfterDelete(ctx, schemaName, modelId, columnData)
			if err != nil {
				return err
			}

		}
	}

	return nil
}

func (s tableManagementService) handleDeleteColumnForLink(ctx context.Context, schemaName string, srcColumnData dto.ColumnResponse, id string) error {
	lg := logger.Get()
	srcColumnMeta := srcColumnData.Meta
	relationId, ok := srcColumnMeta["relation_id"].(string)
	if !ok {
		return app_errors.InvalidColumnMetaForLinkType
	}
	entityRole, ok := srcColumnMeta["entity_role"].(string)
	if !ok {
		return app_errors.InvalidColumnMetaForLinkType
	}

	relation, err := s.relationshipService.GetRelationByID(ctx, relationId, schemaName)
	if err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to get relationship")
		return err
	}

	// source column
	err = s.columnsService.DeleteColumn(ctx, schemaName, srcColumnData.ID.String())
	if err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to delete source column")
		return err
	}

	model, err := s.modelService.GetModelByID(ctx, schemaName, srcColumnData.ModelID.String())
	if err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to get source model by ID")
		return err
	}

	err = s.removeColumnInTableDb(schemaName, model.Alias, srcColumnData.ColumnName)
	if err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to remove column from database")
		return err
	}

	err = s.deleteLookups(ctx, relationId, srcColumnData.ModelID.String(), schemaName)
	if err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to delete lookups")
		return err
	}

	// A one-way self-link has no partner column; everything is already removed.
	if isSingleColumnRelation(relation) {
		return nil
	}

	// target column
	columnIdForDeletion := relation.TargetColumnID
	if entityRole == "target" {
		columnIdForDeletion = relation.SourceColumnID
	}

	targetColumnData, err := s.columnsService.GetColumnByID(ctx, schemaName, columnIdForDeletion)
	if err != nil {
		return err
	}

	err = s.columnsService.DeleteColumn(ctx, schemaName, columnIdForDeletion)
	if err != nil {
		return err
	}

	model, err = s.modelService.GetModelByID(ctx, schemaName, targetColumnData.ModelID)
	if err != nil {
		return err
	}

	err = s.removeColumnInTableDb(schemaName, model.Alias, targetColumnData.ColumnName)
	if err != nil {
		return err
	}

	err = s.deleteLookups(ctx, relationId, targetColumnData.ModelID, schemaName)
	if err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to delete lookups")
		return err
	}

	return nil
}

func (s tableManagementService) DeleteColumnForTable(
	ctx context.Context,
	schemaName string,
	columnData tenant.Column,
) error {
	var columnResponse dto.ColumnResponse
	if err := helpers.StructToStruct(columnData, &columnResponse); err != nil {
		return app_errors.ErrStructToStruct
	}

	if columnData.UIDT == "links" {
		return s.handleDeleteColumnForLink(ctx, schemaName, columnResponse, columnData.ID.String())
	}

	return s.columnsService.DeleteColumn(ctx, schemaName, columnData.ID.String())
}

func (s tableManagementService) ReorderColumn(
	ctx context.Context,
	schemaName string,
	req dto.ReorderColumnRequest,
) ([]dto.ColumnResponse, error) {
	lg := logger.Get()

	sourceColumnData, err := s.columnsService.GetColumnByID(ctx, schemaName, req.SourceColumnID.String())
	if err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to get source column data for reordering")
		return []dto.ColumnResponse{}, err
	}

	targetColumnData, err := s.columnsService.GetColumnByID(ctx, schemaName, req.TargetColumnID.String())
	if err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to get target column data for reordering")
		return []dto.ColumnResponse{}, err
	}

	var updateSourceColumn dto.ColumnResponse
	sourceUpdateReq := dto.ColumnUpdate{
		OrderIndex: targetColumnData.OrderIndex,
	}
	updatedSource, err := s.columnsService.UpdateColumn(ctx, schemaName, req.SourceColumnID.String(), sourceUpdateReq)
	if err != nil {
		return []dto.ColumnResponse{}, err
	}
	if err := helpers.StructToStruct(updatedSource, &updateSourceColumn); err != nil {
		return []dto.ColumnResponse{}, app_errors.ErrStructToStruct
	}

	var updateTargetColumn dto.ColumnResponse
	targetUpdateReq := dto.ColumnUpdate{
		OrderIndex: sourceColumnData.OrderIndex,
	}

	updatedTarget, err := s.columnsService.UpdateColumn(ctx, schemaName, req.TargetColumnID.String(), targetUpdateReq)
	if err != nil {
		return []dto.ColumnResponse{}, err
	}
	if err := helpers.StructToStruct(updatedTarget, &updateTargetColumn); err != nil {
		return []dto.ColumnResponse{}, app_errors.ErrStructToStruct
	}

	return []dto.ColumnResponse{
		updateSourceColumn,
		updateTargetColumn,
	}, nil
}

func (s tableManagementService) reorderColumnsAfterDelete(ctx context.Context, schemaName string, modelID string, deletedColumn dto.ColumnResponse) error {
	functionName := "reorder_columns_after_delete"
	schemaFunctionName := fmt.Sprintf("%s.%s", constant.MasterDatabase, functionName)

	args := map[string]interface{}{
		"p_schema_name": schemaName,
		"p_model_id":    modelID,
		"p_order_index": *deletedColumn.OrderIndex,
	}

	_, err := s.repo.TableService.GetByFunction(
		ctx,
		schemaFunctionName,
		args,
	)
	if err != nil {
		return app_errors.LogDatabaseError(err, "failed to reorder columns after delete")
	}

	return nil
}

func (s tableManagementService) DeleteColumnAndCleanUp(
	ctx context.Context,
	schemaName string,
	id string,
	columnData dto.ColumnResponse,
) error {
	err := s.columnsService.DeleteColumn(ctx, schemaName, id)
	if err != nil {
		return err
	}

	model, err := s.modelService.GetModelByID(ctx, schemaName, columnData.ModelID.String())
	if err != nil {
		return err
	}

	err = s.reorderColumnsAfterDelete(ctx, schemaName, model.ID.String(), columnData)
	if err != nil {
		return err
	}

	err = s.removeColumnInTableDb(schemaName, model.Alias, columnData.ColumnName)
	if err != nil {
		return err
	}
	return nil
}

// DeleteUsedLookupColumn checks for linked columns in all models, then for each linked model,
// checks for lookup columns referencing the deleted column, and deletes them using DeleteColumnAndCleanUp.
func (s tableManagementService) DeleteUsedLookupColumn(ctx context.Context, schemaName string, columnData dto.ColumnResponse) error {
	columns, err := s.GetColumnsByModelID(ctx, schemaName, columnData.ModelID.String())
	if err != nil {
		return err
	}
	for _, col := range columns {
		if col.UIDT == "links" {
			s.HandleLinkedColumnDeletion(ctx, schemaName, col, columnData)
		}
	}
	return nil
}

func (s tableManagementService) HandleLinkedColumnDeletion(ctx context.Context, schemaName string, col dto.ColumnResponse, columnData dto.ColumnResponse) {
	relation, ok := col.Meta["relation"].(map[string]interface{})
	if !ok {
		return
	}
	linkedModelID, ok := relation["with"].(string)
	if !ok || linkedModelID == "" {
		return
	}
	linkedColumns, err := s.GetColumnsByModelID(ctx, schemaName, linkedModelID)
	if err != nil {
		return
	}
	for _, linkedCol := range linkedColumns {
		if linkedCol.UIDT == "lookup" {
			lookupColumnID, ok := linkedCol.Meta["lookup_column_id"].(string)
			if ok && lookupColumnID == columnData.ID.String() {
				s.DeleteLookupColumnAndReorder(ctx, schemaName, linkedCol)
			}
		}
	}
}

func (s tableManagementService) DeleteLookupColumnAndReorder(ctx context.Context, schemaName string, linkedCol dto.ColumnResponse) {
	_ = s.DeleteUsedLookupColumnForRelation(ctx, schemaName, linkedCol)
	_ = s.reorderColumnsAfterDelete(ctx, schemaName, linkedCol.ModelID.String(), linkedCol)
}

func (s tableManagementService) DeleteUsedLookupColumnForRelation(ctx context.Context, schemaName string, columnData dto.ColumnResponse) error {
	lookupColumnID, relationID, ok := s.validateMetaForLookup(columnData.Meta)

	// The relation entry is bookkeeping only (reads are built from lookup meta), so a missing
	// looked-up column or relation must not keep the lookup itself from being deleted.
	if ok {
		if lookupColumn, err := s.columnsService.GetColumnByID(ctx, schemaName, lookupColumnID); err == nil {
			role := s.lookupEntityRole(ctx, schemaName, columnData, relationID)
			if err := s.removeLookupColumnInRelation(ctx, schemaName, role, relationID, lookupColumn.ColumnName); err != nil {
				logger.Get().Warn().Err(err).Str("relationID", relationID).Msg("Failed to remove lookup entry from relation")
			}
		}
	}

	return s.columnsService.DeleteColumn(ctx, schemaName, columnData.ID.String())
}

func (s tableManagementService) DeleteColumn(
	ctx context.Context,
	schemaName string,
	id string,
) error {
	// Check if the column exists
	columnData, err := s.GetColumnById(ctx, schemaName, id)
	if err != nil {
		return err
	}
	if columnData.UIDT == uidtLinks {
		return s.handleDeleteColumnForLink(ctx, schemaName, columnData, id)
	}

	if columnData.UIDT == uidtLookup {
		if err := s.DeleteUsedLookupColumnForRelation(ctx, schemaName, columnData); err != nil {
			return err
		}
		if columnData.OrderIndex == nil {
			return nil
		}
		return s.reorderColumnsAfterDelete(ctx, schemaName, columnData.ModelID.String(), columnData)
	}

	ok := s.allowDelete(columnData)
	if !ok {
		return app_errors.DeleteNotAllowed
	}

	// go func() {
	err = s.DeleteUsedLookupColumn(ctx, schemaName, columnData)
	if err != nil {
		logger.Get().Error().Err(err).Msg("Failed to delete used lookup column in background")
	}
	// }()

	return s.DeleteColumnAndCleanUp(ctx, schemaName, id, columnData)
}

func (s tableManagementService) CreateRow(ctx context.Context, schemaName string, req dto.CreateRowRequest) (dto.RecordResponse, error) {
	lg := logger.Get()

	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.RecordResponse{}, err
	}

	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)

	data := map[string]interface{}{
		"created_by":         req.CreatedBy,
		"last_modified_by":   req.CreatedBy,
		"created_time":       time.Now().UTC(),
		"last_modified_time": time.Now().UTC(),
	}

	createdRecord, err := s.repo.TableService.CreateRecord(tableName, data)
	if err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to create row record")
		return dto.RecordResponse{}, app_errors.LogDatabaseError(err, "failed to create row record")
	}

	return dto.RecordResponse{
		Record: createdRecord,
	}, nil
}

func (s tableManagementService) GetAllRecords(ctx context.Context, schemaName string, modelID string) (dto.RecordsResponse, error) {
	model, err := s.modelService.GetModelByID(ctx, schemaName, modelID)
	if err != nil {
		return dto.RecordsResponse{}, err
	}

	columnsData, err := s.GetColumnsByModelID(ctx, schemaName, modelID)
	if err != nil {
		return dto.RecordsResponse{}, err
	}

	recordsData, err := s.GetRecordsWithLookups(ctx, schemaName, model.Alias, columnsData)
	if err != nil {
		return dto.RecordsResponse{}, err
	}

	return dto.RecordsResponse{
		Records: recordsData.Records,
	}, nil
}

func (s tableManagementService) GetRecordsWithLookups(ctx context.Context, schemaName string, tableName string, columnsData []dto.ColumnResponse) (dto.RecordsResponse, error) {
	lg := logger.Get()
	schemaFunctionName := fmt.Sprintf("%s.%s", constant.MasterDatabase, lookupDataFunctionName)

	relationData := s.buildLookupRelationData(ctx, schemaName, columnsData)

	args := map[string]interface{}{
		"schema_name":       schemaName,
		"source_table_name": tableName,
		"relation_data":     relationData,
	}

	lg.Debug().Interface("args", args).Msg("Executing pagination function with args")

	records, err := s.repo.TableService.GetByFunction(ctx, schemaFunctionName, args)
	if err != nil {
		return dto.RecordsResponse{}, err
	}

	if len(records) == 0 {
		return dto.RecordsResponse{Records: nil}, nil
	}

	normalizedRecord := s.normalizeRecords(records)
	return dto.RecordsResponse{Records: normalizedRecord}, nil
}

func (s tableManagementService) normalizeRecords(records []map[string]interface{}) []map[string]interface{} {
	var normalizedRecord []map[string]interface{}

	getPaginated, ok := records[0][lookupDataFunctionName]
	if !ok {
		return normalizedRecord
	}

	switch val := getPaginated.(type) {
	case []map[string]interface{}:
		normalizedRecord = val
	case []interface{}:
		for _, v := range val {
			if rec, ok := v.(map[string]interface{}); ok {
				normalizedRecord = append(normalizedRecord, rec)
			}
		}
	}
	return normalizedRecord
}

func (s tableManagementService) allowInsert(columnData dto.ColumnResponse) bool {
	if *columnData.System {
		if strings.Contains(strings.ToLower(columnData.ColumnName), "title") {
			return true
		}
		return false
	}
	return true
}

func (s tableManagementService) getRowByID(ctx context.Context, tableName string, rowID interface{}) (map[string]interface{}, error) {
	limit := 1
	params := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   "id",
				Operator: "eq",
				Value:    rowID,
			},
		},
		Limit: &limit,
	}

	records, err := s.repo.TableService.GetTableData(tableName, params)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to get row by id")
	}
	if len(records) == 0 {
		return nil, app_errors.RowNotFound
	}
	return records[0], nil
}

// resolveLinkSides loads the relation behind a link column and returns both sides as seen from that column.
func (s tableManagementService) resolveLinkSides(
	ctx context.Context,
	schemaName string,
	req dto.UpdateRowDataLinksRequest,
) (source, target linkSide, relation tenant.Relation, err error) {
	sourceColumnData, err := s.GetColumnById(ctx, schemaName, req.ColumnId)
	if err != nil {
		return source, target, relation, err
	}
	info, ok := parseLinkMeta(sourceColumnData.Meta)
	if sourceColumnData.UIDT != uidtLinks || !ok || sourceColumnData.ModelID.String() != req.ModelID {
		return source, target, relation, app_errors.InvalidColumnMetaForLinkType
	}

	sourceModel, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return source, target, relation, err
	}

	relation, err = s.relationshipService.GetRelationByID(ctx, info.RelationID, schemaName)
	if err != nil {
		return source, target, relation, app_errors.LogDatabaseError(err, "failed to fetch relation by id")
	}

	if isSingleColumnRelation(relation) {
		// One-way self-link: both "sides" are the same column.
		dataType, err := s.linkDataType(info.EntityRole, relation.RelationType)
		if err != nil {
			return source, target, relation, err
		}
		source = linkSide{
			TableName:  fmt.Sprintf(SchemaTableFormat, schemaName, sourceModel.Alias),
			ColumnName: sourceColumnData.ColumnName,
			DataType:   dataType,
		}
		return source, source, relation, nil
	}

	trgModelId, trgColumnId := relation.TargetModelID, relation.TargetColumnID
	if info.EntityRole == entityRoleTarget {
		trgModelId, trgColumnId = relation.SourceModelID, relation.SourceColumnID
	}

	targetModel, err := s.modelService.GetModelByID(ctx, schemaName, trgModelId)
	if err != nil {
		return source, target, relation, err
	}
	targetColumnData, err := s.columnsService.GetColumnByID(ctx, schemaName, trgColumnId)
	if err != nil {
		return source, target, relation, err
	}

	sourceDataType, err := s.linkDataType(info.EntityRole, relation.RelationType)
	if err != nil {
		return source, target, relation, err
	}
	targetDataType, err := s.linkDataType(oppositeEntityRole(info.EntityRole), relation.RelationType)
	if err != nil {
		return source, target, relation, err
	}

	source = linkSide{
		TableName:  fmt.Sprintf(SchemaTableFormat, schemaName, sourceModel.Alias),
		ColumnName: sourceColumnData.ColumnName,
		DataType:   sourceDataType,
	}
	target = linkSide{
		TableName:  fmt.Sprintf(SchemaTableFormat, schemaName, targetModel.Alias),
		ColumnName: targetColumnData.ColumnName,
		DataType:   targetDataType,
	}
	return source, target, relation, nil
}

// UpdateRawDataForLinks links or unlinks two rows. Both sides, and any partner released to keep
// one-to-one / has-many cardinality, are written in one transaction.
func (s tableManagementService) UpdateRawDataForLinks(
	ctx context.Context,
	schemaName string,
	req dto.UpdateRowDataLinksRequest,
) (dto.RecordResponse, error) {
	source, target, relation, err := s.resolveLinkSides(ctx, schemaName, req)
	if err != nil {
		return dto.RecordResponse{}, err
	}

	isLink := req.Action == linkActionLink
	isSelf := isSelfRelation(relation)
	sourceRowID, targetRowID := int64(req.SourceRowId), int64(req.TargetRowId)

	if isLink && isSelf && sourceRowID == targetRowID {
		return dto.RecordResponse{}, app_errors.SelfReferenceNotAllowed
	}

	err = s.withTx(ctx, func(tx *sql.Tx) error {
		if err := lockLinkRows(ctx, tx, source.TableName, sourceRowID, target.TableName, targetRowID); err != nil {
			return err
		}
		if isSingleColumnRelation(relation) {
			return applySingleColumnLink(ctx, tx, source, relation.RelationType, sourceRowID, targetRowID, isLink, req.UpdatedBy)
		}
		if !isLink {
			return applyLinkPair(ctx, tx, source, target, sourceRowID, targetRowID, false, req.UpdatedBy)
		}
		if isSelf && relation.RelationType == relationHasMany {
			// The INT side of a has-many points from a child to its parent.
			parentSide, parentID, childID := target, sourceRowID, targetRowID
			if source.DataType == linkDataTypeInt {
				parentSide, parentID, childID = source, targetRowID, sourceRowID
			}
			cycle, err := wouldCreateCycle(ctx, tx, parentSide, parentID, childID)
			if err != nil {
				return err
			}
			if cycle {
				return app_errors.LinkCycleDetected
			}
		}
		if err := releaseExistingLinks(ctx, tx, relation.RelationType, source, target, sourceRowID, targetRowID, req.UpdatedBy); err != nil {
			return err
		}
		return applyLinkPair(ctx, tx, source, target, sourceRowID, targetRowID, true, req.UpdatedBy)
	})
	if err != nil {
		return dto.RecordResponse{}, err
	}

	sourceRecord, err := s.getRowByID(ctx, source.TableName, req.SourceRowId)
	if err != nil {
		return dto.RecordResponse{}, err
	}
	// The other row changed too; return it so the client can refresh both without refetching.
	targetRecord, err := s.getRowByID(ctx, target.TableName, req.TargetRowId)
	if err != nil {
		logger.Get().Warn().Err(err).Int("targetRowId", req.TargetRowId).Msg("Failed to load linked row after update")
	}

	return dto.RecordResponse{
		Record:        sourceRecord,
		RelatedRecord: targetRecord,
	}, nil
}

// lockLinkRows locks both rows in a fixed order (avoids deadlocks between opposite link calls)
// and checks that they exist.
func lockLinkRows(ctx context.Context, tx *sql.Tx, sourceTable string, sourceRowID int64, targetTable string, targetRowID int64) error {
	type rowRef struct {
		table   string
		id      int64
		missing error
	}
	refs := []rowRef{
		{sourceTable, sourceRowID, app_errors.RowNotFound},
		{targetTable, targetRowID, app_errors.LinkTargetRowNotFound},
	}
	if refs[1].table < refs[0].table || (refs[1].table == refs[0].table && refs[1].id < refs[0].id) {
		refs[0], refs[1] = refs[1], refs[0]
	}
	for _, ref := range refs {
		exists, err := lockRow(ctx, tx, ref.table, ref.id)
		if err != nil {
			return err
		}
		if !exists {
			return ref.missing
		}
	}
	return nil
}

func (s tableManagementService) InsertRowData(ctx context.Context, schemaName string, req dto.InsertRowDataRequest) (dto.RecordResponse, error) {
	columnData, err := s.GetColumnById(ctx, schemaName, req.ColumnId)
	if err != nil {
		return dto.RecordResponse{}, err
	}

	// Link values must go through the link endpoint so both sides stay in sync; lookups are read-only.
	if columnData.UIDT == uidtLinks || columnData.UIDT == uidtLookup {
		return dto.RecordResponse{}, app_errors.LinkColumnNotWritable
	}

	ok := s.allowInsert(columnData)
	if !ok {
		return dto.RecordResponse{}, app_errors.UpdateNotAllowed
	}

	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.RecordResponse{}, err
	}

	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)

	var value interface{}
	if req.Value != nil {
		value = *req.Value
		// If the column is an array type, ensure the value is a slice
		if columnData.DT != "" && strings.HasSuffix(columnData.DT, "[]") {
			switch value.(type) {
			case []interface{}:
				// already a slice
			default:
				value = []interface{}{value}
			}
		}
	} else {
		value = nil
	}

	data := map[string]interface{}{
		fmt.Sprintf(QuotedColumnFormat, columnData.ColumnName): value,
		"last_modified_by":   req.UpdatedBy,
		"last_modified_time": time.Now().UTC(),
	}

	insertedRecord, err := s.repo.TableService.UpdateRecord(tableName, req.RowId, data)
	if err != nil {
		return dto.RecordResponse{}, app_errors.LogDatabaseError(err, "failed to update record for column")
	}

	return dto.RecordResponse{
		Record: insertedRecord,
	}, nil
}

func (s tableManagementService) CreateRowWithRecords(ctx context.Context, schemaName string, modelAlias string, record map[string]interface{}) (dto.RecordResponse, error) {
	lg := logger.Get()
	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, modelAlias)

	createdRecord, err := s.repo.TableService.CreateRecord(tableName, record)
	if err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to create row with records")
		return dto.RecordResponse{}, app_errors.LogDatabaseError(err, "failed to create row with records")
	}

	return dto.RecordResponse{
		Record: createdRecord,
	}, nil
}

func (s tableManagementService) CreateRowsWithRecordsBulk(ctx context.Context, schemaName string, modelAlias string, records []map[string]interface{}) ([]dto.RecordResponse, error) {
	lg := logger.Get()
	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, modelAlias)

	createdRecords, err := s.repo.BulkService.BulkInsert(tableName, records)
	if err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to bulk insert rows")
		return nil, app_errors.LogDatabaseError(err, "failed to bulk insert rows")
	}

	var response []dto.RecordResponse
	for _, rec := range createdRecords {
		response = append(response, dto.RecordResponse{
			Record: rec,
		})
	}
	return response, nil
}

func (s tableManagementService) CreateRowsWithValues(
	ctx context.Context,
	schemaName string,
	modelID string,
	rowsInput []map[string]interface{},
	createdBy string,
	updatedBy string,
) ([]dto.RecordResponse, error) {
	rows := make([]dto.RecordResponse, 0, len(rowsInput))
	for _, row := range rowsInput {
		createdRow, err := s.CreateRow(ctx, schemaName, dto.CreateRowRequest{
			ModelID:   modelID,
			CreatedBy: createdBy,
		})
		if err != nil {
			return nil, err
		}

		rowID, err := ExtractCreatedRowID(createdRow.Record)
		if err != nil {
			return nil, err
		}

		updatedRow := createdRow
		for columnID, rawValue := range row {
			var valuePtr *interface{}
			if rawValue != nil {
				value := rawValue
				valuePtr = &value
			}

			updatedRow, err = s.InsertRowData(ctx, schemaName, dto.InsertRowDataRequest{
				ModelID:   modelID,
				ColumnId:  columnID,
				RowId:     rowID,
				Value:     valuePtr,
				UpdatedBy: updatedBy,
			})
			if err != nil {
				return nil, err
			}
		}

		rows = append(rows, updatedRow)
	}

	return rows, nil
}

func ExtractCreatedRowID(record map[string]interface{}) (int, error) {
	id, ok := record["id"]
	if !ok {
		return 0, fmt.Errorf("created row id is missing")
	}

	switch v := id.(type) {
	case int:
		return v, nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case float32:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		rowID, err := strconv.Atoi(v)
		if err != nil {
			return 0, err
		}
		return rowID, nil
	default:
		return 0, fmt.Errorf("created row id has unsupported type: %T", id)
	}
}

// cleanUpLinksForRows removes every reference to the given rows from partner link columns, in one transaction.
func (s tableManagementService) cleanUpLinksForRows(ctx context.Context, schemaName string, model tenant.Model, rowIDs []int64) error {
	if len(rowIDs) == 0 {
		return nil
	}
	return s.withTx(ctx, func(tx *sql.Tx) error {
		return s.removeBackReferences(ctx, tx, schemaName, model, rowIDs, "")
	})
}

func (s tableManagementService) DeleteRow(ctx context.Context, schemaName string, req dto.DeleteRowDataRequest) error {
	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return err
	}

	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)
	if _, err := s.getRowByID(ctx, tableName, req.RowId); err != nil {
		return err
	}

	if err := s.cleanUpLinksForRows(ctx, schemaName, model, []int64{int64(req.RowId)}); err != nil {
		return err
	}

	if err := s.repo.TableService.DeleteRecord(tableName, req.RowId); err != nil {
		return app_errors.LogDatabaseError(err, "failed to delete record")
	}

	return nil
}

func (s tableManagementService) checkAttachmentType(attachmentValue interface{}) []map[string]interface{} {
	var result []map[string]interface{}

	switch v := attachmentValue.(type) {
	case []map[string]interface{}:
		result = v
	case []interface{}:
		for _, item := range v {
			switch iv := item.(type) {
			case map[string]interface{}:
				result = append(result, iv)
			default:
				// skip unknown types
			}
		}
	case map[string]interface{}:
		result = []map[string]interface{}{v}
	default:
		result = nil
	}

	return result
}

func (s tableManagementService) assetsToMaps(assets []tenant.Assets) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(assets))
	for _, asset := range assets {
		result = append(result, asset.Map())
	}
	return result
}

// AddAttachment now supports uploading all file types, not just images.
func (s tableManagementService) AddAttachment(
	ctx context.Context,
	schemaName string,
	req dto.AddAttachmentRequest,
	files []*multipart.FileHeader,
) (dto.RecordResponse, error) {
	lg := logger.Get()
	// uploadAssets now supports all file types
	assets, err := s.uploadAssets(ctx, schemaName, files)
	if err != nil {
		return dto.RecordResponse{}, err
	}

	columnName, tableName, err := s.getColumnNameAndTableName(ctx, schemaName, req.ColumnId, req.ModelID)
	if err != nil {
		return dto.RecordResponse{}, err
	}

	rowData, err := s.getRowByID(ctx, tableName, req.RowId)
	if err != nil {
		return dto.RecordResponse{}, err
	}

	attachmentValue := s.mergeAttachmentValues(rowData[columnName], s.assetsToMaps(assets))

	data := map[string]interface{}{
		columnName:           attachmentValue,
		"last_modified_time": time.Now().UTC(),
	}

	insertedRecord, err := s.repo.TableService.UpdateRecord(tableName, req.RowId, data)
	if err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to add attachment to record")
		return dto.RecordResponse{}, app_errors.LogDatabaseError(err, "failed to add attachment to record")
	}

	return dto.RecordResponse{
		Record: insertedRecord,
	}, nil
}

func (s tableManagementService) updateSpecificAttachment(attachments []tenant.Assets, updatedAttachment tenant.Assets) []tenant.Assets {
	for i, asset := range attachments {
		if asset.ID == updatedAttachment.ID {
			attachments[i] = updatedAttachment
			break
		}
	}
	return attachments
}

func (s tableManagementService) attachmentValuesToAssets(attachmentValue interface{}) []tenant.Assets {
	attachmentMaps := s.checkAttachmentType(attachmentValue)
	assets := make([]tenant.Assets, 0, len(attachmentMaps))

	for _, attachmentMap := range attachmentMaps {
		var asset tenant.Assets
		if err := helpers.MapToStruct(attachmentMap, &asset); err != nil {
			continue
		}
		assets = append(assets, asset)
	}

	return assets
}

func (s tableManagementService) UpdateAttachment(
	ctx context.Context,
	schemaName string,
	req dto.UpdateAttachmentRequest,
) (dto.RecordResponse, error) {
	columnName, tableName, err := s.getColumnNameAndTableName(ctx, schemaName, req.ColumnId, req.ModelID)
	if err != nil {
		return dto.RecordResponse{}, err
	}

	rowData, err := s.getRowByID(ctx, tableName, req.RowId)
	if err != nil {
		return dto.RecordResponse{}, err
	}

	attachments := s.attachmentValuesToAssets(rowData[columnName])

	updatedAsset, err := s.assetManagementService.UpdateAsset(ctx, req.AssetId, req.Content, schemaName)
	if err != nil {
		return dto.RecordResponse{}, err
	}

	updatedAttachments := s.updateSpecificAttachment(attachments, updatedAsset)

	attachmentValue := s.assetsToMaps(updatedAttachments)

	data := map[string]interface{}{
		columnName:           attachmentValue,
		"last_modified_time": time.Now().UTC(),
	}

	insertedRecord, err := s.repo.TableService.UpdateRecord(tableName, req.RowId, data)
	if err != nil {
		return dto.RecordResponse{}, app_errors.LogDatabaseError(err, "failed to add attachment to record")
	}
	return dto.RecordResponse{
		Record: insertedRecord,
	}, nil
}

func (s tableManagementService) BulkDeleteRows(ctx context.Context, schemaName string, req dto.BulkDeleteRowsRequest) (int, error) {
	lg := logger.Get()
	// Get the model to retrieve table name
	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		lg.Error().Stack().Err(err).Str("modelID", req.ModelID).Msg("Failed to get model for bulk delete")
		return 0, err
	}
	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)
	deletedCount := 0
	// Remove links pointing at these rows before bulk delete
	rowIDs := make([]int64, len(req.RowIds))
	for i, rowId := range req.RowIds {
		rowIDs[i] = int64(rowId)
	}
	if err := s.cleanUpLinksForRows(ctx, schemaName, model, rowIDs); err != nil {
		lg.Error().Stack().Err(err).Msg("Failed to handle links for rows")
		return deletedCount, err
	}
	// Convert row IDs to interface{} slice for BulkDelete
	ids := make([]interface{}, len(req.RowIds))
	for i, id := range req.RowIds {
		ids[i] = id
	}
	// Use BulkService to delete all rows at once
	count, err := s.repo.BulkService.BulkDelete(tableName, ids, "id")
	if err != nil {
		lg.Error().Stack().Err(err).Str("tableName", tableName).Msg("Failed to bulk delete rows")
		return deletedCount, app_errors.LogDatabaseError(err, "failed to bulk delete rows")
	}
	deletedCount = int(count)
	lg.Info().Int("deletedCount", deletedCount).Str("tableName", tableName).Msg("Successfully bulk deleted rows")
	return deletedCount, nil
}

func (s tableManagementService) RemoveAttachments(
	ctx context.Context,
	schemaName string,
	req dto.RemoveAttachmentsRequest,
) (dto.RecordResponse, error) {
	// Get column name and table name
	columnData, err := s.GetColumnById(ctx, schemaName, req.ColumnId)
	if err != nil {
		return dto.RecordResponse{}, err
	}

	ok := s.allowInsert(columnData)
	if !ok {
		return dto.RecordResponse{}, app_errors.UpdateNotAllowed
	}

	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.RecordResponse{}, err
	}

	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)

	// Get the row data
	rowData, err := s.getRowByID(ctx, tableName, req.RowId)
	if err != nil {
		return dto.RecordResponse{}, err
	}

	// Remove the specified attachments from the column value
	existingAttachments := s.checkAttachmentType(rowData[columnData.ColumnName])
	attachmentsToRemove := make(map[string]struct{}, len(req.Attachments))
	for _, id := range req.Attachments {
		attachmentsToRemove[id] = struct{}{}
	}

	var updatedAttachments []map[string]interface{}
	for _, asset := range existingAttachments {
		assetID, _ := asset["id"]
		assetIDStr, ok := assetID.(string)
		if !ok || assetIDStr == "" {
			// Skip if we can't get the asset ID as string
			updatedAttachments = append(updatedAttachments, asset)
			continue
		}
		if _, shouldRemove := attachmentsToRemove[assetIDStr]; !shouldRemove {
			updatedAttachments = append(updatedAttachments, asset)
		}
	}

	data := map[string]interface{}{
		fmt.Sprintf(QuotedColumnFormat, columnData.ColumnName): updatedAttachments,
		"last_modified_time": time.Now().UTC(),
	}

	updatedRecord, err := s.repo.TableService.UpdateRecord(tableName, req.RowId, data)
	if err != nil {
		return dto.RecordResponse{}, app_errors.LogDatabaseError(err, "failed to remove attachments from record")
	}

	return dto.RecordResponse{
		Record: updatedRecord,
	}, nil
}

// uploadAssets now supports all file types, not just images.
func (s tableManagementService) uploadAssets(ctx context.Context, schemaName string, files []*multipart.FileHeader) ([]tenant.Assets, error) {
	uploadReq := dto.UploadAssetRequest{
		Files: files, // Accepts all file types
	}
	assets, err := s.assetManagementService.Upload(ctx, uploadReq, schemaName)
	if err != nil {
		return nil, err
	}
	return assets, nil
}

func (s tableManagementService) getColumnNameAndTableName(
	ctx context.Context,
	schemaName string,
	columnId string,
	modelId string,
) (string, string, error) {
	columnData, err := s.GetColumnById(ctx, schemaName, columnId)
	if err != nil {
		return "", "", err
	}

	ok := s.allowInsert(columnData)
	if !ok {
		return "", "", app_errors.UpdateNotAllowed
	}

	model, err := s.modelService.GetModelByID(ctx, schemaName, modelId)
	if err != nil {
		return "", "", err
	}

	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)
	return columnData.ColumnName, tableName, nil
}

func (s tableManagementService) mergeAttachmentValues(existing interface{}, assets []map[string]interface{}) []map[string]interface{} {
	attachmentValue := s.checkAttachmentType(existing)
	for _, asset := range assets {
		attachmentValue = append(attachmentValue, asset)
	}
	return attachmentValue
}

func (s tableManagementService) BulkUpdateColumns(ctx context.Context, schemaName string, modelID string, columnID string, updates []dto.UpdateColumnsRequest) error {
	if len(updates) == 0 {
		return nil
	}

	fetchedModel, err := s.modelService.GetModelByID(ctx, schemaName, modelID)
	if err != nil {
		return err
	}

	// Get column details to extract the actual column name from the database
	fetchedColumn, err := s.columnsService.GetColumnByID(ctx, schemaName, columnID) // Validate column exists before reset
	if err != nil {
		return err
	}

	return s.columnsService.BulkUpdate(ctx, schemaName, fetchedModel.Alias, fetchedColumn.ColumnName, updates)
}

func (s tableManagementService) ResetColumnValues(ctx context.Context, schemaName string, modelID string, columnID string) error {
	fetchedModel, err := s.modelService.GetModelByID(ctx, schemaName, modelID)
	if err != nil {
		return err
	}

	fetchedColumn, err := s.columnsService.GetColumnByID(ctx, schemaName, columnID) // Validate column exists before reset
	if err != nil {
		return err
	}

	return s.columnsService.ResetColumn(ctx, schemaName, fetchedModel.Alias, fetchedColumn.ColumnName)
}
