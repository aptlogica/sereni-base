// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
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
	SchemaTableFormat    = "\"%s\".\"%s\""
	QuotedColumnFormat   = "\"%s\""
	ErrConvertViewStruct = "Failed to convert view struct"
)

// targetColumnParams holds parameters for creating target column in relation
type targetColumnParams struct {
	ColumnData      dto.AddColumnRequest
	SourceModelData tenant.Model
	RelationWith    string
	RelationID      uuid.UUID
	RelationType    string
	Now             time.Time
}

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

// updateLinkDataParams holds parameters for updating link data
type updateLinkDataParams struct {
	SourceTableName  string
	TargetTableName  string
	SourceColumnName string
	TargetColumnName string
	SourceDataType   string
	TargetDataType   string
	Request          dto.UpdateRowDataLinksRequest
}

// updateIfExistParams holds parameters for checking and updating existing links
type updateIfExistParams struct {
	RelationType     string
	SourceTableName  string
	SourceColumnName string
	TargetTableName  string
	TargetColumnName string
	SourceDataType   string
	TargetDataType   string
	Request          dto.UpdateRowDataLinksRequest
}

// unlinkRowDataParams holds parameters for unlinking row data
type unlinkRowDataParams struct {
	Request         dto.DeleteRowDataRequest
	SourceTableName string
	TargetTableName string
	Column          tenant.Column
	TargetColumn    tenant.Column
	RowData         map[string]interface{}
	SourceDataType  string
	TargetDataType  string
}

// unlinkSingleRowParams holds parameters for unlinking a single row
type unlinkSingleRowParams struct {
	Request         dto.DeleteRowDataRequest
	SourceTableName string
	TargetTableName string
	Column          tenant.Column
	TargetColumn    tenant.Column
	SourceDataType  string
	TargetDataType  string
	TargetRowId     int64
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
	return slug + "_" + fmt.Sprintf("%d", timestamp)
}

func (s tableManagementService) addColumnInTableDb(schemaName string, tableName string, columnData tenant.Column) error {
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

// 	s.addColumnInTableDb(schemaName, trgTable.Alias)
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

	err = s.addColumnInTableDb(schemaName, model.Alias, column)
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

	sourcColumn, sourceModelData, err := s.createSourceColumnForRelation(ctx, schemaName, columnData, sourceMeta, relationId, relationType, now)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	targetColumn, targetModelData, err := s.createTargetColumnForRelation(ctx, schemaName, targetColumnParams{
		ColumnData:      columnData,
		SourceModelData: sourceModelData,
		RelationWith:    relationWith,
		RelationID:      relationId,
		RelationType:    relationType,
		Now:             now,
	})
	if err != nil {
		return dto.ColumnResponse{}, err
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

func (s tableManagementService) createSourceColumnForRelation(
	ctx context.Context,
	schemaName string,
	columnData dto.AddColumnRequest,
	sourceMeta map[string]interface{},
	relationId uuid.UUID,
	relationType string,
	now time.Time,
) (tenant.Column, tenant.Model, error) {
	sourceMeta["entity_role"] = "source"
	sourceMeta["relation_id"] = relationId

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

	if err := s.addColumnInTableDb(schemaName, sourceModelData.Alias, sourcColumn); err != nil {
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
		"entity_role": "target",
		"relation_id": params.RelationID,
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
		Title:       params.SourceModelData.Title,
		ColumnName:  s.slugify(params.SourceModelData.Title),
		Description: helpers.StringPtr(""),
		Meta:        targetMeta,
		UIDT:        params.ColumnData.UIDT,
		DT:          helpers.StringPtr(targetDataType),
		Virtual:     params.ColumnData.Virtual != nil && *params.ColumnData.Virtual,
		System:      params.ColumnData.System != nil && *params.ColumnData.System,
		Deleted:     false,
		OrderIndex:  helpers.Float64Ptr(targetCurrentOrderIndex + 1),
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

	if err := s.addColumnInTableDb(schemaName, targetModelData.Alias, targetColumn); err != nil {
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

func (s tableManagementService) addLookupColumnInRelation(
	ctx context.Context,
	schemaName string,
	modelId string,
	relationID string,
	lookupColumnName string,
) error {
	relationData, err := s.relationshipService.GetRelationByID(ctx, relationID, schemaName)
	if err != nil {
		lg := logger.Get()
		lg.Debug().Str("relationID", relationID).Str("schemaName", schemaName).Msg("Fetching source lookup columns for relation")
		lg.Error().Stack().Err(err).Msg("Failed to get relation by ID")
		return err
	}

	relationUpdation := dto.RelationUpdate{
		UpdatedAt: time.Now().UTC(),
	}

	if relationData.SourceModelID == modelId {
		if relationData.SourceLookupColumns == nil {
			relationUpdation.SourceLookupColumns = []string{lookupColumnName}
		} else {
			newArr := append(relationData.SourceLookupColumns, lookupColumnName)
			relationUpdation.SourceLookupColumns = newArr
		}
	}
	if relationData.TargetModelID == modelId {
		if relationData.TargetLookupColumns == nil {
			relationUpdation.TargetLookupColumns = []string{lookupColumnName}
		} else {
			newArr := append(relationData.TargetLookupColumns, lookupColumnName)
			relationUpdation.TargetLookupColumns = newArr
		}
	}

	_, err = s.relationshipService.UpdateRelation(ctx, relationID, relationUpdation, schemaName)
	if err != nil {
		lg := logger.Get()
		lg.Error().Stack().Err(err).Msg("Failed to update relation with source lookup columns")
		return err
	}
	return nil
}

func (s tableManagementService) removeLookupColumnInRelation(
	ctx context.Context,
	schemaName string,
	modelId string,
	relationID string,
	lookupColumnName string,
) error {
	relationData, err := s.relationshipService.GetRelationByID(ctx, relationID, schemaName)
	if err != nil {
		lg := logger.Get()
		lg.Debug().Str("relationID", relationID).Str("schemaName", schemaName).Msg("Fetching target lookup columns for relation")
		lg.Error().Stack().Err(err).Msg("Failed to get relation by ID for removal")
		return err
	}

	relationUpdation := dto.RelationUpdate{
		UpdatedAt: time.Now().UTC(),
	}

	if relationData.SourceModelID == modelId {
		relationUpdation.SourceLookupColumns = s.removeLookupColumnFromList(relationData.SourceLookupColumns, lookupColumnName, "SourceLookupColumns")
	}
	if relationData.TargetModelID == modelId {
		relationUpdation.TargetLookupColumns = s.removeLookupColumnFromList(relationData.TargetLookupColumns, lookupColumnName, "TargetLookupColumns")
	}
	_, err = s.relationshipService.UpdateRelation(ctx, relationID, relationUpdation, schemaName)
	if err != nil {
		lg := logger.Get()
		lg.Error().Stack().Err(err).Msg("Failed to update relation with target lookup columns")
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

func (s tableManagementService) addColumnWithLookup(
	ctx context.Context,
	schemaName string,
	columnData dto.AddColumnRequest,
) (dto.ColumnResponse, error) {
	now := time.Now().UTC()

	lookupColumnID, relationID, ok := s.validateMetaForLookup(columnData.Meta)
	if !ok {
		return dto.ColumnResponse{}, app_errors.InvalidColumnMetaForLookupType
	}

	lookupColumnData, err := s.columnsService.GetColumnByID(ctx, schemaName, lookupColumnID)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	lookupModelData, err := s.modelService.GetModelByID(ctx, schemaName, lookupColumnData.ModelID)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	srcColumnCreatedata := dto.ColumnInsertion{
		ID:          uuid.New(),
		ModelID:     columnData.ModelID,
		BaseID:      columnData.BaseID,
		Title:       columnData.Title,
		ColumnName:  fmt.Sprintf("%s_%s", lookupModelData.Alias, lookupColumnData.ColumnName),
		Description: &columnData.Description,
		Meta:        columnData.Meta,
		UIDT:        columnData.UIDT,
		DT:          helpers.StringPtr(columnData.UIDT),
		Virtual:     columnData.Virtual != nil && *columnData.Virtual,
		System:      columnData.System != nil && *columnData.System,
		Deleted:     false,
		OrderIndex:  columnData.OrderIndex,
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

	if err := s.addLookupColumnInRelation(ctx, schemaName, columnData.ModelID.String(), relationID, lookupColumnData.ColumnName); err != nil {
		lg := logger.Get()
		lg.Error().Stack().Err(err).Msg("Failed to add lookup column in relationship")
		return dto.ColumnResponse{}, app_errors.ErrStructToStruct
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
	// For link columns, only update title, description, last_modified_time and last_modified_by
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

	var updatedColumnResponse dto.ColumnResponse
	if err := helpers.StructToStruct(updatedColumn, &updatedColumnResponse); err != nil {
		return dto.ColumnResponse{}, app_errors.ErrStructToStruct
	}

	return updatedColumnResponse, nil
}

func (s tableManagementService) updateColumnForLookup(
	ctx context.Context,
	schemaName string,
	columnData dto.ColumnResponse,
	req dto.ColumnUpdate,
) (dto.ColumnResponse, error) {
	lookupColumnID, relationID, ok := s.validateMetaForLookup(columnData.Meta)
	if ok {
		lookupColumn, err := s.columnsService.GetColumnByID(ctx, schemaName, lookupColumnID)
		if err != nil {
			return dto.ColumnResponse{}, err
		}

		err = s.removeLookupColumnInRelation(ctx, schemaName, columnData.ModelID.String(), relationID, lookupColumn.ColumnName)
		if err != nil {
			return dto.ColumnResponse{}, err
		}
	}

	var updatedColumn tenant.Column
	updatedLookupColumnID, _, ok := s.validateMetaForLookup(*req.Meta)
	if ok {
		updatedLookupColumn, err := s.columnsService.GetColumnByID(ctx, schemaName, updatedLookupColumnID)
		if err != nil {
			return dto.ColumnResponse{}, err
		}

		err = s.addLookupColumnInRelation(ctx, schemaName, columnData.ModelID.String(), relationID, updatedLookupColumn.ColumnName)
		if err != nil {
			return dto.ColumnResponse{}, err
		}

		lookupModelData, err := s.modelService.GetModelByID(ctx, schemaName, updatedLookupColumn.ModelID)
		if err != nil {
			return dto.ColumnResponse{}, err
		}

		req.ColumnName = helpers.StringPtr(fmt.Sprintf("%s_%s", lookupModelData.Alias, updatedLookupColumn.ColumnName))
		updatedColumn, err = s.columnsService.UpdateColumn(ctx, schemaName, columnData.ID.String(), req)
		if err != nil {
			return dto.ColumnResponse{}, err
		}
	}

	var updatedColumnResponse dto.ColumnResponse
	if err := helpers.StructToStruct(updatedColumn, &updatedColumnResponse); err != nil {
		return dto.ColumnResponse{}, app_errors.ErrStructToStruct
	}

	return updatedColumnResponse, nil
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

	if req.UIDT != nil && *req.UIDT != "" {
		dt, _ := s.getDataBaseType(*req.UIDT)
		req.DT = helpers.StringPtr(dt)
	}

	if columnData.UIDT == "link" {
		return s.updateColumnForLink(ctx, schemaName, columnData, req)
	}

	if columnData.UIDT == "lookup" {
		return s.updateColumnForLookup(ctx, schemaName, columnData, req)
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
		if col.UIDT == "lookup" {
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
			s.handleLinkedColumnDeletion(ctx, schemaName, col, columnData)
		}
	}
	return nil
}

func (s tableManagementService) handleLinkedColumnDeletion(ctx context.Context, schemaName string, col dto.ColumnResponse, columnData dto.ColumnResponse) {
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
				s.deleteLookupColumnAndReorder(ctx, schemaName, linkedCol)
			}
		}
	}
}

func (s tableManagementService) deleteLookupColumnAndReorder(ctx context.Context, schemaName string, linkedCol dto.ColumnResponse) {
	_ = s.DeleteUsedLookupColumnForRelation(ctx, schemaName, linkedCol)
	_ = s.reorderColumnsAfterDelete(ctx, schemaName, linkedCol.ModelID.String(), linkedCol)
}

func (s tableManagementService) DeleteUsedLookupColumnForRelation(ctx context.Context, schemaName string, columnData dto.ColumnResponse) error {
	lookupColumnID, relationID, _ := s.validateMetaForLookup(columnData.Meta)

	lookupColumn, err := s.columnsService.GetColumnByID(ctx, schemaName, lookupColumnID)
	if err != nil {
		return err
	}

	err = s.removeLookupColumnInRelation(ctx, schemaName, columnData.ModelID.String(), relationID, lookupColumn.ColumnName)
	if err != nil {
		return err
	}

	err = s.columnsService.DeleteColumn(ctx, schemaName, columnData.ID.String())
	if err != nil {
		return err
	}
	return nil

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
	if columnData.UIDT == "links" {
		return s.handleDeleteColumnForLink(ctx, schemaName, columnData, id)
	}

	if columnData.UIDT == "lookup" {
		return s.DeleteUsedLookupColumnForRelation(ctx, schemaName, columnData)
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

func (s tableManagementService) checkLookuup(columnsData []dto.ColumnResponse) []string {
	relationIdsSet := make(map[string]struct{})
	for _, col := range columnsData {
		if col.UIDT == "lookup" {
			relationId, _ := col.Meta["relation_id"].(string)
			if relationId != "" {
				relationIdsSet[relationId] = struct{}{}
			}
		}
	}
	relationIds := make([]string, 0, len(relationIdsSet))
	for id := range relationIdsSet {
		relationIds = append(relationIds, id)
	}
	return relationIds
}

func (s tableManagementService) GetRecordsWithLookups(ctx context.Context, schemaName string, tableName string, columnsData []dto.ColumnResponse) (dto.RecordsResponse, error) {
	lg := logger.Get()
	functionName := "get_table_data_with_relation"
	schemaFunctionName := fmt.Sprintf("%s.%s", constant.MasterDatabase, functionName)

	relationData := s.buildRelationData(ctx, schemaName, columnsData)

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

func (s tableManagementService) buildRelationData(ctx context.Context, schemaName string, columnsData []dto.ColumnResponse) []map[string]interface{} {
	relationIds := s.checkLookuup(columnsData)
	if len(relationIds) == 0 {
		return nil
	}

	var relationData []map[string]interface{}
	for _, col := range columnsData {
		if col.UIDT != "links" {
			continue
		}

		rData := s.buildRelationDataForColumn(ctx, schemaName, col, relationIds)
		if rData != nil {
			relationData = append(relationData, rData)
		}
	}
	return relationData
}

func (s tableManagementService) buildRelationDataForColumn(
	ctx context.Context,
	schemaName string,
	col dto.ColumnResponse,
	relationIds []string,
) map[string]interface{} {
	rData := map[string]interface{}{
		"source_column_name": col.ColumnName,
	}

	relationId, _ := col.Meta["relation_id"].(string)
	if !s.isRelationIdInList(relationId, relationIds) {
		return nil
	}

	entityRole, _ := col.Meta["entity_role"].(string)

	relation, err := s.relationshipService.GetRelationByID(ctx, relationId, schemaName)
	if err != nil {
		return nil
	}

	rData["relation"] = relation.RelationType

	if err := s.addTargetInfoToRelationData(ctx, schemaName, rData, relation, entityRole); err != nil {
		return nil
	}

	return rData
}

func (s tableManagementService) isRelationIdInList(relationId string, relationIds []string) bool {
	for _, relID := range relationIds {
		if relationId == relID {
			return true
		}
	}
	return false
}

func (s tableManagementService) addTargetInfoToRelationData(
	ctx context.Context,
	schemaName string,
	rData map[string]interface{},
	relation tenant.Relation,
	entityRole string,
) error {
	if entityRole == "source" {
		return s.addSourceTargetInfo(ctx, schemaName, rData, relation)
	}
	return s.addTargetSourceInfo(ctx, schemaName, rData, relation)
}

func (s tableManagementService) addSourceTargetInfo(
	ctx context.Context,
	schemaName string,
	rData map[string]interface{},
	relation tenant.Relation,
) error {
	if len(relation.SourceLookupColumns) == 0 {
		return fmt.Errorf("no source lookup columns")
	}

	targetModel, err := s.modelService.GetModelByID(ctx, schemaName, relation.TargetModelID)
	if err != nil {
		return err
	}

	rData["target_table_name"] = targetModel.Alias
	rData["target_column_name"] = "id"
	rData["target_columns"] = relation.SourceLookupColumns
	return nil
}

func (s tableManagementService) addTargetSourceInfo(
	ctx context.Context,
	schemaName string,
	rData map[string]interface{},
	relation tenant.Relation,
) error {
	if len(relation.TargetLookupColumns) == 0 {
		return fmt.Errorf("no target lookup columns")
	}

	targetModel, err := s.modelService.GetModelByID(ctx, schemaName, relation.SourceModelID)
	if err != nil {
		return err
	}

	rData["target_table_name"] = targetModel.Alias
	rData["target_column_name"] = "id"
	rData["target_columns"] = relation.TargetLookupColumns
	return nil
}

func (s tableManagementService) normalizeRecords(records []map[string]interface{}) []map[string]interface{} {
	var normalizedRecord []map[string]interface{}

	getPaginated, ok := records[0]["get_table_data_with_relation"]
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

func (s tableManagementService) getRowByRelationColumn(ctx context.Context, tableName string, columnName string, linkedId interface{}) (map[string]interface{}, error) {
	limit := 1
	params := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   columnName,
				Operator: "eq",
				Value:    linkedId,
			},
		},
		Limit: &limit,
	}

	records, err := s.repo.TableService.GetTableData(tableName, params)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to get row by relation column")
	}
	if len(records) == 0 {
		return nil, app_errors.RowNotFound
	}
	return records[0], nil
}

func (s tableManagementService) getRowByRelationColumnHasMany(ctx context.Context, tableName string, columnName string, linkedId interface{}) (map[string]interface{}, error) {
	limit := 1
	params := dbModels.QueryParams{
		Filters: []dbModels.QueryFilter{
			{
				Column:   columnName,
				Operator: "any",
				Value:    linkedId,
			},
		},
		Limit: &limit,
	}

	records, err := s.repo.TableService.GetTableData(tableName, params)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to get row by relation column (has many)")
	}
	if len(records) == 0 {
		return nil, app_errors.RowNotFound
	}
	return records[0], nil
}

func (s tableManagementService) linkRecord(
	ctx context.Context,
	datatype string,
	tableName string,
	rowId int,
	columnName string,
	value int,
	updatedBy string,
) (map[string]interface{}, error) {
	rowData, err := s.getRowByID(ctx, tableName, rowId)
	if err != nil {
		return nil, err
	}

	switch datatype {
	case "INT[]":
		return s.linkIntArray(tableName, rowId, columnName, value, updatedBy, rowData)
	case "INT":
		return s.linkInt(tableName, rowId, columnName, value, updatedBy)
	default:
		return nil, fmt.Errorf("unsupported datatype: %s", datatype)
	}
}

func (s tableManagementService) linkIntArray(
	tableName string,
	rowId int,
	columnName string,
	value int,
	updatedBy string,
	rowData map[string]interface{},
) (map[string]interface{}, error) {
	updatedArr := s.buildUpdatedArrayForLink(rowData[columnName], value)

	data := map[string]interface{}{
		columnName:           updatedArr,
		"last_modified_time": time.Now().UTC(),
	}
	if updatedBy != "" {
		data["last_modified_by"] = updatedBy
	}
	return s.repo.TableService.UpdateRecord(tableName, rowId, data)
}

func (s tableManagementService) buildUpdatedArrayForLink(existingValue interface{}, value int) []int64 {
	switch v := existingValue.(type) {
	case nil:
		return []int64{int64(value)}
	case []int64:
		return s.appendIfNotExists(v, value)
	case []string:
		return s.appendToConvertedStringArray(v, value)
	case int64:
		return s.buildArrayFromInt64(v, value)
	case int:
		return s.buildArrayFromInt(v, value)
	default:
		return []int64{int64(value)}
	}
}

func (s tableManagementService) appendIfNotExists(arr []int64, value int) []int64 {
	for _, item := range arr {
		if item == int64(value) {
			return arr
		}
	}
	return append(arr, int64(value))
}

func (s tableManagementService) appendToConvertedStringArray(strArr []string, value int) []int64 {
	var arr []int64
	for _, s := range strArr {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			arr = append(arr, n)
		}
	}

	for _, item := range arr {
		if item == int64(value) {
			return arr
		}
	}
	return append(arr, int64(value))
}

func (s tableManagementService) buildArrayFromInt64(existing int64, value int) []int64 {
	if existing == int64(value) {
		return []int64{existing}
	}
	return []int64{existing, int64(value)}
}

func (s tableManagementService) buildArrayFromInt(existing int, value int) []int64 {
	if existing == value {
		return []int64{int64(existing)}
	}
	return []int64{int64(existing), int64(value)}
}

func (s tableManagementService) linkInt(
	tableName string,
	rowId int,
	columnName string,
	value int,
	updatedBy string,
) (map[string]interface{}, error) {
	data := map[string]interface{}{
		columnName:           value,
		"last_modified_time": time.Now().UTC(),
	}
	if updatedBy != "" {
		data["last_modified_by"] = updatedBy
	}
	return s.repo.TableService.UpdateRecord(tableName, rowId, data)
}

func (s tableManagementService) unlinkRecord(
	ctx context.Context,
	datatype string,
	tableName string,
	rowId int,
	columnName string,
	value int,
	updatedBy string,
) (map[string]interface{}, error) {
	rowData, err := s.getRowByID(ctx, tableName, rowId)
	if err != nil {
		return nil, err
	}

	switch datatype {
	case "INT[]":
		return s.unlinkIntArray(tableName, rowId, columnName, value, updatedBy, rowData)
	case "INT":
		return s.unlinkInt(tableName, rowId, columnName, value, updatedBy, rowData)
	default:
		return rowData, nil
	}
}

func (s tableManagementService) unlinkIntArray(
	tableName string,
	rowId int,
	columnName string,
	value int,
	updatedBy string,
	rowData map[string]interface{},
) (map[string]interface{}, error) {
	arrInt64 := s.convertToInt64Array(rowData[columnName])
	if arrInt64 == nil {
		return rowData, nil
	}

	newArr := make([]int64, 0, len(arrInt64))
	for _, v := range arrInt64 {
		if v != int64(value) {
			newArr = append(newArr, v)
		}
	}

	data := map[string]interface{}{
		columnName:           newArr,
		"last_modified_time": time.Now().UTC(),
	}
	if updatedBy != "" {
		data["last_modified_by"] = updatedBy
	}
	return s.repo.TableService.UpdateRecord(tableName, rowId, data)
}

func (s tableManagementService) convertToInt64Array(value interface{}) []int64 {
	switch arr := value.(type) {
	case []int64:
		return arr
	case []int:
		var arrInt64 []int64
		for _, v := range arr {
			arrInt64 = append(arrInt64, int64(v))
		}
		return arrInt64
	case []string:
		var arrInt64 []int64
		for _, s := range arr {
			if n, err := strconv.ParseInt(s, 10, 64); err == nil {
				arrInt64 = append(arrInt64, n)
			}
		}
		return arrInt64
	case int64:
		return []int64{arr}
	case int:
		return []int64{int64(arr)}
	case nil:
		return nil
	default:
		return nil
	}
}

func (s tableManagementService) unlinkInt(
	tableName string,
	rowId int,
	columnName string,
	value int,
	updatedBy string,
	rowData map[string]interface{},
) (map[string]interface{}, error) {
	val, ok := rowData[columnName].(int64)
	if !ok {
		if v, ok2 := rowData[columnName].(int); ok2 {
			val = int64(v)
			ok = true
		}
	}
	if ok && int(val) == value {
		data := map[string]interface{}{
			columnName:           nil,
			"last_modified_time": time.Now().UTC(),
		}
		if updatedBy != "" {
			data["last_modified_by"] = updatedBy
		}
		return s.repo.TableService.UpdateRecord(tableName, rowId, data)
	}
	return rowData, nil
}

func (s tableManagementService) updateLinkData(
	ctx context.Context,
	params updateLinkDataParams,
) (dto.RecordResponse, error) {
	var (
		sourceInsertedRecord map[string]interface{}
		err                  error
	)
	switch params.Request.Action {
	case "link":
		sourceInsertedRecord, err = s.linkRecord(ctx, params.SourceDataType, params.SourceTableName, params.Request.SourceRowId, params.SourceColumnName, params.Request.TargetRowId, params.Request.UpdatedBy)
	default:
		sourceInsertedRecord, err = s.unlinkRecord(ctx, params.SourceDataType, params.SourceTableName, params.Request.SourceRowId, params.SourceColumnName, params.Request.TargetRowId, params.Request.UpdatedBy)
	}
	if err != nil {
		return dto.RecordResponse{}, app_errors.LogDatabaseError(err, "failed to update link data (source side)")
	}

	switch params.Request.Action {
	case "link":
		_, err = s.linkRecord(ctx, params.TargetDataType, params.TargetTableName, params.Request.TargetRowId, params.TargetColumnName, params.Request.SourceRowId, params.Request.UpdatedBy)
	default:
		_, err = s.unlinkRecord(ctx, params.TargetDataType, params.TargetTableName, params.Request.TargetRowId, params.TargetColumnName, params.Request.SourceRowId, params.Request.UpdatedBy)
	}
	if err != nil {
		return dto.RecordResponse{}, app_errors.LogDatabaseError(err, "failed to update link data (target side)")
	}

	return dto.RecordResponse{
		Record: sourceInsertedRecord,
	}, nil
}

func (s tableManagementService) updateIfExist(
	ctx context.Context,
	params updateIfExistParams,
) error {

	type check struct {
		srcTable    string
		srcColumn   string
		srcDatatype string
		trgTable    string
		trgColumn   string
		trgDataType string
		id          int
	}
	checks := []check{
		{params.SourceTableName, params.SourceColumnName, params.SourceDataType, params.TargetTableName, params.TargetColumnName, params.TargetDataType, params.Request.TargetRowId},
		{params.TargetTableName, params.TargetColumnName, params.TargetDataType, params.SourceTableName, params.SourceColumnName, params.SourceDataType, params.Request.SourceRowId},
	}

	for _, c := range checks {
		switch {
		case params.RelationType == "one-to-one":
			if err := s.handleOneToOneRelation(ctx, c, params.Request); err != nil {
				return err
			}
		case params.RelationType == "has-many" && c.srcDatatype == "INT[]":
			if err := s.handleHasManyIntArrayRelation(ctx, c, params.Request); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s tableManagementService) handleOneToOneRelation(
	ctx context.Context,
	c struct {
		srcTable, srcColumn, srcDatatype, trgTable, trgColumn, trgDataType string
		id                                                                 int
	},
	req dto.UpdateRowDataLinksRequest,
) error {
	data, err := s.getRowByRelationColumn(ctx, c.srcTable, c.srcColumn, c.id)
	if err != nil && err != app_errors.RowNotFound {
		return err
	}
	if err == nil {
		srcID, _ := data["id"].(int64)
		tgtID := c.id
		req.SourceRowId = int(srcID)
		req.TargetRowId = int(tgtID)
		req.Action = "unlink"
		_, err = s.updateLinkData(ctx, updateLinkDataParams{
			SourceTableName:  c.srcTable,
			TargetTableName:  c.trgTable,
			SourceColumnName: c.srcColumn,
			TargetColumnName: c.trgColumn,
			SourceDataType:   c.srcDatatype,
			TargetDataType:   c.trgDataType,
			Request:          req,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s tableManagementService) handleHasManyIntArrayRelation(
	ctx context.Context,
	c struct {
		srcTable, srcColumn, srcDatatype, trgTable, trgColumn, trgDataType string
		id                                                                 int
	},
	req dto.UpdateRowDataLinksRequest,
) error {
	lg := logger.Get()
	lg.Debug().Str("srcTable", c.srcTable).Str("srcColumn", c.srcColumn).Int("id", c.id).Msg("Handling has-many int array relation")
	data, err := s.getRowByRelationColumnHasMany(ctx, c.srcTable, c.srcColumn, c.id)
	if err != nil && err != app_errors.RowNotFound {
		return err
	}
	if data != nil {
		srcID, _ := data["id"].(int64)
		tgtID := c.id
		req.SourceRowId = int(srcID)
		req.TargetRowId = int(tgtID)
		req.Action = "unlink"
		_, err = s.updateLinkData(ctx, updateLinkDataParams{
			SourceTableName:  c.srcTable,
			TargetTableName:  c.trgTable,
			SourceColumnName: c.srcColumn,
			TargetColumnName: c.trgColumn,
			SourceDataType:   c.srcDatatype,
			TargetDataType:   c.trgDataType,
			Request:          req,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s tableManagementService) UpdateRawDataForLinks(
	ctx context.Context,
	schemaName string,
	req dto.UpdateRowDataLinksRequest,
) (dto.RecordResponse, error) {

	sourceColumnData, err := s.GetColumnById(ctx, schemaName, req.ColumnId)
	if err != nil {
		return dto.RecordResponse{}, err
	}

	sourceModel, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.RecordResponse{}, err
	}

	sourceTableName := fmt.Sprintf(SchemaTableFormat, schemaName, sourceModel.Alias)

	relationId, ok := sourceColumnData.Meta["relation_id"].(string)
	if !ok {
		return dto.RecordResponse{}, app_errors.ErrInternal
	}

	relationData, err := s.relationshipService.GetRelationByID(ctx, relationId, schemaName)
	if err != nil {
		return dto.RecordResponse{}, app_errors.LogDatabaseError(err, "failed to fetch relation by id")
	}

	srcEntityRole := sourceColumnData.Meta["entity_role"]
	trgModelId := relationData.TargetModelID
	trgColumnId := relationData.TargetColumnID
	if srcEntityRole == "target" {
		trgModelId = relationData.SourceModelID
		trgColumnId = relationData.SourceColumnID
	}

	targetModel, err := s.modelService.GetModelByID(ctx, schemaName, trgModelId)
	if err != nil {
		return dto.RecordResponse{}, err
	}

	targetTableName := fmt.Sprintf(SchemaTableFormat, schemaName, targetModel.Alias)

	targetColumnData, err := s.columnsService.GetColumnByID(ctx, schemaName, trgColumnId)
	if err != nil {
		return dto.RecordResponse{}, err
	}

	relationType, _, _ := s.validateMetaForLink(sourceColumnData.Meta)

	trgEntityRole := "source"
	if srcEntityRole == "source" {
		trgEntityRole = "target"
	}
	srcUidt := fmt.Sprintf("links_%v_%v", srcEntityRole, relationType)
	sourceDataType, err := s.getDataBaseType(srcUidt)
	if err != nil {
		return dto.RecordResponse{}, err
	}
	trgUidt := fmt.Sprintf("links_%v_%v", trgEntityRole, relationType)
	targetDataType, err := s.getDataBaseType(trgUidt)
	if err != nil {
		return dto.RecordResponse{}, err
	}

	if req.Action == "link" {
		err = s.updateIfExist(ctx, updateIfExistParams{
			RelationType:     relationType,
			SourceTableName:  sourceTableName,
			SourceColumnName: sourceColumnData.ColumnName,
			TargetTableName:  targetTableName,
			TargetColumnName: targetColumnData.ColumnName,
			SourceDataType:   sourceDataType,
			TargetDataType:   targetDataType,
			Request:          req,
		})
		if err != nil {
			return dto.RecordResponse{}, err
		}
	}

	return s.updateLinkData(ctx, updateLinkDataParams{
		SourceTableName:  sourceTableName,
		TargetTableName:  targetTableName,
		SourceColumnName: sourceColumnData.ColumnName,
		TargetColumnName: targetColumnData.ColumnName,
		SourceDataType:   sourceDataType,
		TargetDataType:   targetDataType,
		Request:          req,
	})
}

func (s tableManagementService) InsertRowData(ctx context.Context, schemaName string, req dto.InsertRowDataRequest) (dto.RecordResponse, error) {
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

func (s tableManagementService) handleDeleteRowForLinks(ctx context.Context, sourceModel tenant.Model, rowData map[string]interface{}, schemaName string, req dto.DeleteRowDataRequest) error {
	columns, err := s.columnsService.GetColumnByModelID(ctx, schemaName, sourceModel.ID.String())
	if err != nil {
		return err
	}

	for _, column := range columns {
		if column.UIDT != "links" {
			continue
		}

		val, ok := rowData[column.ColumnName]
		if !ok || val == nil {
			continue
		}
		// Check if it's an empty array (slice)
		if arr, isSlice := val.([]interface{}); isSlice && len(arr) == 0 {
			continue
		}

		if err := s.handleLinkColumn(ctx, schemaName, req, sourceModel, rowData, column); err != nil {
			return err
		}
	}

	return nil
}

func (s tableManagementService) handleLinkColumn(
	ctx context.Context,
	schemaName string,
	req dto.DeleteRowDataRequest,
	sourceModel tenant.Model,
	rowData map[string]interface{},
	column tenant.Column,
) error {
	relationId := column.Meta["relation_id"].(string)
	entityRole := column.Meta["entity_role"].(string)

	relationData, err := s.relationshipService.GetRelationByID(ctx, relationId, schemaName)
	if err != nil {
		return err
	}

	targetModelId := relationData.SourceModelID
	targetColumnID := relationData.SourceColumnID
	if entityRole == "source" {
		targetModelId = relationData.TargetModelID
		targetColumnID = relationData.TargetColumnID
	}

	targetModel, err := s.modelService.GetModelByID(ctx, schemaName, targetModelId)
	if err != nil {
		return err
	}

	targetColumn, err := s.columnsService.GetColumnByID(ctx, schemaName, targetColumnID)
	if err != nil {
		return err
	}

	sourceDataType, targetDataType, err := s.resolveDataTypes(column)
	if err != nil {
		return err
	}

	sourceTableName := fmt.Sprintf(SchemaTableFormat, schemaName, sourceModel.Alias)
	targetTableName := fmt.Sprintf(SchemaTableFormat, schemaName, targetModel.Alias)
	return s.unlinkRowData(ctx, unlinkRowDataParams{
		Request:         req,
		SourceTableName: sourceTableName,
		TargetTableName: targetTableName,
		Column:          column,
		TargetColumn:    targetColumn,
		RowData:         rowData,
		SourceDataType:  sourceDataType,
		TargetDataType:  targetDataType,
	})
}

// Resolve source/target datatype from relation metadata
func (s tableManagementService) resolveDataTypes(column tenant.Column) (string, string, error) {
	relation := column.Meta["relation"].(map[string]interface{})
	relationType := relation["type"]
	entityRole := column.Meta["entity_role"]

	// source role
	tempUidt := fmt.Sprintf("%s_%v_%v", column.UIDT, entityRole, relationType)
	sourceDataType, err := s.getDataBaseType(tempUidt)
	if err != nil {
		return "", "", err
	}

	// target role
	targteEntityRole := "source"
	if entityRole == "source" {
		targteEntityRole = "target"
	}
	trgTempUidt := fmt.Sprintf("%s_%v_%v", column.UIDT, targteEntityRole, relationType)
	targetDataType, err := s.getDataBaseType(trgTempUidt)
	if err != nil {
		return "", "", err
	}

	return sourceDataType, targetDataType, nil
}

// Unlink row(s) depending on datatype (INT or INT[])
func (s tableManagementService) unlinkRowData(
	ctx context.Context,
	params unlinkRowDataParams,
) error {
	if params.SourceDataType == "INT" {
		targetRowId := params.RowData[params.Column.ColumnName].(int64)
		return s.unlinkSingleRow(ctx, unlinkSingleRowParams{
			Request:         params.Request,
			SourceTableName: params.SourceTableName,
			TargetTableName: params.TargetTableName,
			Column:          params.Column,
			TargetColumn:    params.TargetColumn,
			SourceDataType:  params.SourceDataType,
			TargetDataType:  params.TargetDataType,
			TargetRowId:     targetRowId,
		})
	}

	// handle multiple (INT[])
	targetRowIds := params.RowData[params.Column.ColumnName].([]int64)
	for _, targetRowId := range targetRowIds {
		if err := s.unlinkSingleRow(ctx, unlinkSingleRowParams{
			Request:         params.Request,
			SourceTableName: params.SourceTableName,
			TargetTableName: params.TargetTableName,
			Column:          params.Column,
			TargetColumn:    params.TargetColumn,
			SourceDataType:  params.SourceDataType,
			TargetDataType:  params.TargetDataType,
			TargetRowId:     targetRowId,
		}); err != nil {
			return err
		}
	}
	return nil
}

// Build unlink request and call updateLinkData
func (s tableManagementService) unlinkSingleRow(
	ctx context.Context,
	params unlinkSingleRowParams,
) error {
	updateLinkReq := dto.UpdateRowDataLinksRequest{
		ModelID:     params.Request.ModelID,
		ColumnId:    params.Column.ID.String(),
		SourceRowId: params.Request.RowId,
		TargetRowId: int(params.TargetRowId),
		Action:      "unlink",
	}

	_, err := s.updateLinkData(
		ctx,
		updateLinkDataParams{
			SourceTableName:  params.SourceTableName,
			TargetTableName:  params.TargetTableName,
			SourceColumnName: params.Column.ColumnName,
			TargetColumnName: params.TargetColumn.ColumnName,
			SourceDataType:   params.SourceDataType,
			TargetDataType:   params.TargetDataType,
			Request:          updateLinkReq,
		},
	)
	return err
}

func (s tableManagementService) DeleteRow(ctx context.Context, schemaName string, req dto.DeleteRowDataRequest) error {
	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return err
	}

	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)
	rowData, err := s.getRowByID(ctx, tableName, req.RowId)
	if err != nil {
		return err
	}

	if err := s.handleDeleteRowForLinks(ctx, model, rowData, schemaName, req); err != nil {
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
	// Process each row for link cleanup before bulk delete
	for _, rowId := range req.RowIds {
		rowData, err := s.getRowByID(ctx, tableName, rowId)
		if err != nil {
			lg.Warn().Err(err).Int("rowId", rowId).Msg("Row not found, skipping")
			continue
		}
		// Handle link cleanup for this row
		deleteReq := dto.DeleteRowDataRequest{
			ModelID: req.ModelID,
			RowId:   rowId,
		}
		if err := s.handleDeleteRowForLinks(ctx, model, rowData, schemaName, deleteReq); err != nil {
			lg.Error().Stack().Err(err).Int("rowId", rowId).Msg("Failed to handle links for row")
			return deletedCount, err
		}
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
