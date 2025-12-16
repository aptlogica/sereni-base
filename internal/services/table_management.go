package services

import (
	"context"
	"fmt"
	"godbgrest/pkg"
	dbModels "godbgrest/pkg/models"
	"mime/multipart"
	app_errors "serenibase/internal/app-errors"
	"serenibase/internal/constant"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	"serenibase/internal/services/interfaces"
	"serenibase/internal/utils/helpers"
	"strconv"
	"strings"
	"time"

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

	var columnsDefinationParams []dbModels.ColumnDefinition

	for _, col := range columnsData {
		columnsDefinationParams = append(columnsDefinationParams, dbModels.ColumnDefinition{
			Name:     helpers.ToSnakeCase(col.Title),
			DataType: col.DT,
		})
	}

	creationReq := dbModels.CreateTableRequest{
		Name:       fmt.Sprintf("\"%s\".\"%s\"", schemaName, tableName),
		Columns:    columnsDefinationParams,
		PrimaryKey: []string{"id"},
	}

	err := s.repo.TableService.CreateTable(creationReq)
	if err != nil {
		return []dto.AddColumnRequest{}, app_errors.DatabaseError
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
			System:      true,
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

func (s tableManagementService) CreateTableWithDefaults(ctx context.Context, tableData dto.CreateTableRequest, schemaName string) (dto.TableResponse, error) {
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
		fmt.Println("modelService.Create: ", err)
		return dto.TableResponse{}, err
	}

	fmt.Println("insertedModel: ", insertedModel)

	systemColumns, err := s.createTableWithDefaultsInDB(schemaName, insertedModel.Alias)
	if err != nil {
		fmt.Println("createTableWithDefaultsInDB: ", err)
		return dto.TableResponse{}, err
	}

	columnsResponse, err := s.insertSystemColumns(schemaName, insertedModel, systemColumns)
	if err != nil {
		fmt.Println("s.insertSystemColumns: ", err)
		return dto.TableResponse{}, err
	}

	viewResponse, err := s.createDefaultView(ctx, schemaName, insertedModel)
	if err != nil {
		fmt.Println("s.createDefaultView: ", err)
		return dto.TableResponse{}, err
	}

	// Check type of insertedModel.UpdatedBy
	fmt.Printf("Type of insertedModel.UpdatedAt: %T\n", insertedModel.UpdatedAt)

	var modelResponse dto.ModelResponse
	if err := helpers.StructToStruct(insertedModel, &modelResponse); err != nil {
		return dto.TableResponse{}, app_errors.ErrStructToStruct
	}
	fmt.Println(modelResponse)

	recordsData, err := s.GetAllRecords(ctx, schemaName, insertedModel.ID.String())
	if err != nil {
		return dto.TableResponse{}, err
	}

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

func (s tableManagementService) GetTableByID(ctx context.Context, id string, schemaName string, pageSize int, pageNumber int) (dto.TableResponse, error) {
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

	var recordsData dto.RecordsResponse
	if pageSize == 0 || pageNumber == 0 {
		recordsData, err = s.GetAllRecords(ctx, schemaName, id)
	} else {
		recordsData, err = s.GetRecordsWithPagination(ctx, schemaName, model.Alias, columnsData, pageSize, pageNumber)
	}
	// recordsData, err := s.getAllRecordsIncludingAssets(ctx, schemaName, model.Alias, columnsData)
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

func (s tableManagementService) GetTableDataPagination(ctx context.Context, req dto.PaginationRequest, schemaName string) (dto.TablePageResponse, error) {
	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.TablePageResponse{}, err
	}

	columnsData, err := s.GetColumnsByModelID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.TablePageResponse{}, err
	}

	recordsData, err := s.GetRecordsWithPagination(ctx, schemaName, model.Alias, columnsData, req.PageSize, req.PageNumber)
	if err != nil {
		return dto.TablePageResponse{}, err
	}

	tableResponse := dto.TablePageResponse{
		Columns: columnsData,
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
	err := s.repo.TableService.DropTable(ctx, fmt.Sprintf("\"%s\".\"%s\"", schemaName, tableName))
	if err != nil {
		return app_errors.DatabaseError
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

	if err := s.modelService.DeleteModel(ctx, schemaName, modelID); err != nil {
		return err
	}

	fmt.Println("schemaName, model.Alias------------", schemaName, model.Alias)

	if err := s.deleteTableInDB(ctx, schemaName, model.Alias); err != nil {
		fmt.Println("err")
		return err
	}

	return nil
}

func (s tableManagementService) slugify(input string) string {
	slug := strings.ToLower(input)
	slug = strings.ReplaceAll(slug, " ", "_")
	timestamp := time.Now().Unix()
	return slug + "_" + fmt.Sprintf("%d", timestamp)
}

func (s tableManagementService) addColumnInTableDb(schemaName string, tableName string, columnData tenant.Column) error {
	schematableName := fmt.Sprintf("\"%s\".\"%s\"", schemaName, tableName)

	addColumnReq := dbModels.AddColumnRequest{
		Column: dbModels.ColumnDefinition{
			Name:     fmt.Sprintf("\"%s\"", columnData.ColumnName),
			DataType: *columnData.DT,
		},
	}

	err := s.repo.TableService.AddColumn(schematableName, addColumnReq)
	if err != nil {
		return app_errors.DatabaseError
	}
	return nil
}

func (s tableManagementService) getDataBaseType(uidt string) (string, error) {
	fmt.Println("uidt: ", uidt)
	mapping, exists := constant.UITypeMappings[uidt]
	if !exists {
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
	// Accept only "many-to-many", "has-many", or "one-to-one"
	switch rType {
	case "many-to-many", "has-many", "one-to-one":
		// valid
	default:
		return "", "", false
	}
	if !ok {
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

// func (s tableManagementService) linkWithTargetTable(ctx context.Context, schemaName string, srcColumnData tenant.Column, metaMap map[string]interface{}) error {

// 	trgModelId, _ := metaMap["with"].(string)
// 	relatioinType, _ := metaMap["type"].(string)

// 	trgTable, err := s.modelService.GetModelByID(ctx, schemaName, trgModelId)
// 	if err != nil {
// 		return err
// 	}

// 	now := time.Now()
// 	entity_role := "target"
// 	trgMetaMap := map[string]interface{}{
// 		"relation": map[string]interface{}{
// 			"with":        srcColumnData.ModelID,
// 			"type":        relatioinType,
// 			"entity_role": entity_role,
// 		},
// 	}

// 	metaBytes, err := helpers.MarshalJSON(trgMetaMap)
// 	if err != nil {
// 		return err
// 	}
// 	meta := string(metaBytes)
// 	var tempUidt string
// 	if srcColumnData.DT != nil {
// 		tempUidt = *srcColumnData.DT + fmt.Sprintf("%v", metaMap["type"]) + entity_role
// 	}
// 	dt, err := s.getDataBaseType(tempUidt)
// 	if err != nil {
// 		return err
// 	}

// 	trgColumnCreateData := dto.ColumnInsertion{
// 		ID:          uuid.New(),
// 		ModelID:     trgTable.ID,
// 		BaseID:      trgTable.BaseID,
// 		Title:       trgTable.Alias,
// 		ColumnName:  s.slugify(trgTable.Alias),
// 		Description: nil,
// 		Meta:        helpers.StringPtr(meta),
// 		UIDT:        srcColumnData.UIDT,
// 		DT:          helpers.StringPtr(dt),
// 		Virtual:     srcColumnData.Virtual,
// 		System:      srcColumnData.System,
// 		Deleted:     false,
// 		OrderIndex:  ,
// 		CreatedAt:   now,
// 		UpdatedAt:   now,
// 	}

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
	columnCreateData := dto.ColumnInsertion{
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

	column, err := s.columnsService.Create(ctx, columnCreateData, schemaName)
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
		fmt.Println(err)
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
	tempUidt := columnData.UIDT
	relationType, relationWith, ok := s.validateMetaForLink(columnData.Meta)

	relationId := uuid.New()

	if !ok {
		return dto.ColumnResponse{}, app_errors.InvalidColumnMetaForLinkType
	}
	sourceEntityRole := "source"
	sourceMeta["entity_role"] = sourceEntityRole
	sourceMeta["relation_id"] = relationId
	sourceTempUidt := fmt.Sprintf("%s_source_%v", tempUidt, relationType)
	sourceDataType, err := s.getDataBaseType(sourceTempUidt)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	now := time.Now().UTC()
	srcColumnCreateData := dto.ColumnInsertion{
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

	sourcColumn, err := s.columnsService.Create(ctx, srcColumnCreateData, schemaName)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	// Name of source model
	sourceModelData, err := s.modelService.GetModelByID(ctx, schemaName, columnData.ModelID.String())
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	err = s.addColumnInTableDb(schemaName, sourceModelData.Alias, sourcColumn)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	targetEntityRole := "target"
	targetMeta := map[string]interface{}{
		"relation": map[string]interface{}{
			"with": columnData.ModelID.String(),
			"type": relationType,
		},
		"entity_role": targetEntityRole,
		"relation_id": relationId,
	}
	targetTempUidt := fmt.Sprintf("%s_target_%v", tempUidt, relationType)
	targetDataType, err := s.getDataBaseType(targetTempUidt)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	targetCurrentOrderIndex, err := s.columnsService.GetMaxOrderIndexOfColumn(ctx, schemaName, relationWith)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	targetColumnCreateData := dto.ColumnInsertion{
		ID:          uuid.New(),
		ModelID:     uuid.MustParse(relationWith),
		BaseID:      columnData.BaseID,
		Title:       sourceModelData.Title,
		ColumnName:  s.slugify(sourceModelData.Title),
		Description: helpers.StringPtr(""),
		Meta:        targetMeta,
		UIDT:        columnData.UIDT,
		DT:          helpers.StringPtr(targetDataType),
		Virtual:     columnData.Virtual != nil && *columnData.Virtual,
		System:      columnData.System != nil && *columnData.System,
		Deleted:     false,
		OrderIndex:  helpers.Float64Ptr(targetCurrentOrderIndex + 1),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	targetColumn, err := s.columnsService.Create(ctx, targetColumnCreateData, schemaName)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	// Name of target model
	targetModelData, err := s.modelService.GetModelByID(ctx, schemaName, targetColumn.ModelID)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	err = s.addColumnInTableDb(schemaName, targetModelData.Alias, targetColumn)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	relationInsertionData := dto.RelationInsertion{
		ID:             relationId,
		BaseID:         columnData.BaseID.String(),
		SourceModelID:  sourceModelData.ID.String(),
		SourceColumnID: sourcColumn.ID.String(),
		TargetModelID:  targetModelData.ID.String(),
		TargetColumnID: targetColumn.ID.String(),
		// SourceLookupColumns: []string{},
		// TargetLookupColumns: []string{},
		RelationType: relationType,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	_, err = s.relationshipService.Create(ctx, relationInsertionData, schemaName)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	var columnResponse dto.ColumnResponse
	if err := helpers.StructToStruct(sourcColumn, &columnResponse); err != nil {
		fmt.Println(err)
		return dto.ColumnResponse{}, app_errors.ErrStructToStruct
	}
	return columnResponse, nil
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
		fmt.Println("relationID, schemaName;;;;;; ", relationID, schemaName)
		fmt.Println("err: GetRelationByID ==== ", err)
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
		fmt.Println("err: UpdateRelation ==== ", err)
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
		fmt.Println("relationID, schemaName;;;;;; ", relationID, schemaName)
		fmt.Println("err: GetRelationByID ==== ", err)
		return err
	}

	relationUpdation := dto.RelationUpdate{
		UpdatedAt: time.Now().UTC(),
	}

	if relationData.SourceModelID == modelId {
		fmt.Printf("Type of relationData.SourceLookupColumns: %T\n", relationData.SourceLookupColumns)
		if relationData.SourceLookupColumns != nil {
			newArr := make([]string, 0)
			for _, col := range relationData.SourceLookupColumns {
				if col != lookupColumnName {
					newArr = append(newArr, col)
				}
			}
			relationUpdation.SourceLookupColumns = newArr
		} else {
			relationUpdation.SourceLookupColumns = []string{}
		}
	}
	if relationData.TargetModelID == modelId {
		fmt.Printf("Type of relationData.TargetLookupColumns: %T\n", relationData.TargetLookupColumns)
		if relationData.TargetLookupColumns != nil {
			newArr := make([]string, 0)
			for _, col := range relationData.TargetLookupColumns {
				if col != lookupColumnName {
					newArr = append(newArr, col)
				}
			}
			relationUpdation.TargetLookupColumns = newArr
		} else {
			relationUpdation.TargetLookupColumns = []string{}
		}
	}

	_, err = s.relationshipService.UpdateRelation(ctx, relationID, relationUpdation, schemaName)
	if err != nil {
		fmt.Println("err: UpdateRelation ==== ", err)
		return err
	}
	return nil
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

	srcColumnCreateData := dto.ColumnInsertion{
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

	insertedColumn, err := s.columnsService.Create(ctx, srcColumnCreateData, schemaName)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	var columnResponse dto.ColumnResponse
	if err := helpers.StructToStruct(insertedColumn, &columnResponse); err != nil {
		fmt.Println("StructToStruct ::::: ", err)
		return dto.ColumnResponse{}, app_errors.ErrStructToStruct
	}

	if err := s.addLookupColumnInRelation(ctx, schemaName, columnData.ModelID.String(), relationID, lookupColumnData.ColumnName); err != nil {
		fmt.Println("addLookupColumnInRelation ::::: ", err)
		return dto.ColumnResponse{}, app_errors.ErrStructToStruct
	}
	return columnResponse, nil
}

func (s tableManagementService) GetColumnById(
	ctx context.Context,
	schemaName string,
	id string,
) (dto.ColumnResponse, error) {
	column, err := s.columnsService.GetColumnByID(ctx, schemaName, id)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	var columnResponse dto.ColumnResponse
	if err := helpers.StructToStruct(column, &columnResponse); err != nil {
		fmt.Println(err)
		return dto.ColumnResponse{}, app_errors.ErrStructToStruct
	}

	return columnResponse, nil
}

func (s tableManagementService) GetAllColumns(
	ctx context.Context,
	schemaName string,
) ([]dto.ColumnResponse, error) {
	columns, err := s.columnsService.GetAllColumns(ctx, schemaName)
	if err != nil {
		return nil, err
	}

	var columnResponses []dto.ColumnResponse
	for _, column := range columns {
		var columnResponse dto.ColumnResponse
		if err := helpers.StructToStruct(column, &columnResponse); err != nil {
			fmt.Println(err)
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
	columns, err := s.columnsService.GetColumnByModelID(ctx, schemaName, modelID)
	if err != nil {
		return nil, err
	}

	var columnResponses []dto.ColumnResponse
	for _, column := range columns {
		var columnResponse dto.ColumnResponse
		if err := helpers.StructToStruct(column, &columnResponse); err != nil {
			fmt.Println(err)
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
		fmt.Println(err)
		return dto.ViewResponse{}, app_errors.ErrStructToStruct
	}

	return viewResponse, nil
}

func (s tableManagementService) GetViewByID(
	ctx context.Context,
	schemaName string,
	id string,
) (dto.ViewResponse, error) {
	view, err := s.viewService.GetViewByID(ctx, schemaName, id)
	if err != nil {
		return dto.ViewResponse{}, err
	}

	var viewResponse dto.ViewResponse
	if err := helpers.StructToStruct(view, &viewResponse); err != nil {
		fmt.Println(err)
		return dto.ViewResponse{}, app_errors.ErrStructToStruct
	}

	return viewResponse, nil
}

func (s tableManagementService) GetAllViews(
	ctx context.Context,
	schemaName string,
) ([]dto.ViewResponse, error) {
	views, err := s.viewService.GetAllViews(ctx, schemaName)
	if err != nil {
		return nil, err
	}

	viewResponses := make([]dto.ViewResponse, 0, len(views))
	for _, view := range views {
		var viewResponse dto.ViewResponse
		if err := helpers.StructToStruct(view, &viewResponse); err != nil {
			fmt.Println(err)
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
	views, err := s.viewService.GetViewsByModelID(ctx, schemaName, modelID)
	if err != nil {
		return nil, err
	}

	viewResponses := make([]dto.ViewResponse, 0, len(views))
	for _, view := range views {
		var viewResponse dto.ViewResponse
		if err := helpers.StructToStruct(view, &viewResponse); err != nil {
			fmt.Println(err)
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
		fmt.Println(err)
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
		return false
	}
	return true
}

func (s tableManagementService) updateColumnDatatypeInDb(ctx context.Context, schemaName string, tableName string, columnName string, newDataType string, empty_before bool) error {
	functionName := "convert_column_type"
	schemaFunctionName := fmt.Sprintf("%s.%s", constant.MasterDatabase, functionName)

	args := map[string]interface{}{
		"schema_name":  schemaName,
		"table_name":   tableName,
		"column_name":  columnName,
		"target_type":  newDataType,
		"empty_before": empty_before,
	}

	fmt.Println(args)

	_, err := s.repo.TableService.GetByFunction(
		ctx,
		schemaFunctionName,
		args,
	)
	if err != nil {
		fmt.Println("err---->", err)
		return err
	}

	return nil
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

	updatedColumn, err := s.columnsService.UpdateColumn(ctx, schemaName, columnData.ID.String(), req)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	updatedLookupColumnID, updatedRelationID, ok := s.validateMetaForLookup(updatedColumn.Meta)
	if ok {
		udpatedLookupColumn, err := s.columnsService.GetColumnByID(ctx, schemaName, updatedLookupColumnID)
		if err != nil {
			return dto.ColumnResponse{}, err
		}

		err = s.addLookupColumnInRelation(ctx, schemaName, columnData.ModelID.String(), relationID, udpatedLookupColumn.ColumnName)
		if err != nil {
			return dto.ColumnResponse{}, err
		}
	}

	err = s.addLookupColumnInRelation(ctx, schemaName, columnData.ModelID.String(), updatedRelationID, columnData.ColumnName)
	if err != nil {
		return dto.ColumnResponse{}, err
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
	if req.UpdatedAt.IsZero() {
		req.UpdatedAt = time.Now().UTC()
	}

	columnData, err := s.GetColumnById(ctx, schemaName, id)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	if !s.allowUpdate(columnData) {
		if req.Title == nil || strings.Contains(columnData.ColumnName, *req.Title) {
			return dto.ColumnResponse{}, app_errors.UpdateNotAllowed
		}
		req = dto.ColumnUpdate{
			Title: req.Title,
		}
	}

	if req.UIDT != nil && *req.UIDT != "" {
		dt, _ := s.getDataBaseType(*req.UIDT)
		req.DT = helpers.StringPtr(dt)
	}

	if columnData.UIDT == "lookup" {
		return s.updateColumnForLookup(ctx, schemaName, columnData, req)
	}

	column, err := s.columnsService.UpdateColumn(ctx, schemaName, id, req)
	if err != nil {
		return dto.ColumnResponse{}, err
	}

	if (req.UIDT != nil && *req.UIDT != "") && (columnData.DT != *req.DT) {
		model, err := s.modelService.GetModelByID(ctx, schemaName, column.ModelID)
		if err != nil {
			return dto.ColumnResponse{}, err
		}

		// Check if type conversion is allowed from the old UIDT to the requested UIDT
		allowed := false
		conversions, ok := constant.AllowedConversions[columnData.UIDT]
		if ok {
			for _, conv := range conversions {
				if conv == *req.UIDT {
					allowed = true
					break
				}
			}
		}
		// Always allow conversion to the same type
		if !allowed && columnData.UIDT == *req.UIDT {
			allowed = true
		}

		if err := s.updateColumnDatatypeInDb(ctx, schemaName, model.Alias, column.ColumnName, *req.DT, !allowed); err != nil {
			// Revert metadata if DB update fails
			fmt.Printf("DEBUG: Reverting column metadata for %s due to DB error: %v\n", column.ColumnName, err)

			revertReq := dto.ColumnUpdate{
				DT:   helpers.StringPtr(columnData.DT),
				UIDT: helpers.StringPtr(columnData.UIDT),
			}
			// We ignore the error from revert because we want to return the original error
			_, _ = s.columnsService.UpdateColumn(ctx, schemaName, id, revertReq)
			return dto.ColumnResponse{}, err
		}
	}

	var columnResponse dto.ColumnResponse
	if err := helpers.StructToStruct(column, &columnResponse); err != nil {
		fmt.Println(err)
		return dto.ColumnResponse{}, app_errors.ErrStructToStruct
	}

	return columnResponse, nil
}

func (s tableManagementService) removeColumnInTableDb(schemaName string, tableName string, columnName string) error {
	schematableName := fmt.Sprintf("\"%s\".\"%s\"", schemaName, tableName)

	addColumnReq := dbModels.AlterTableRequest{
		Action: "drop_column",
		Data: dbModels.DropColumnRequest{
			ColumnName: fmt.Sprintf("\"%s\"", columnName),
			Cascade:    true,
		},
	}

	err := s.repo.TableService.AlterTable(schematableName, addColumnReq)
	if err != nil {
		return app_errors.DatabaseError
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
			fmt.Printf("Found lookup column: %s\n", col.ColumnName)
			colRelationId, _ := col.Meta["relation_id"].(string)

			if colRelationId == relationId {
				err = s.DeleteColumn(ctx, schemaName, col.ID.String())
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (s tableManagementService) handleDeleteColumnForLink(ctx context.Context, schemaName string, srcColumnData dto.ColumnResponse) error {
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
		fmt.Println("relation err--->", err)
		return err
	}

	// source column
	err = s.columnsService.DeleteColumn(ctx, schemaName, srcColumnData.ID.String())
	if err != nil {
		fmt.Println("source DeleteColumn err--->", err)
		return err
	}

	model, err := s.modelService.GetModelByID(ctx, schemaName, srcColumnData.ModelID.String())
	if err != nil {
		fmt.Println("source GetModelByID err--->", err)
		return err
	}

	err = s.removeColumnInTableDb(schemaName, model.Alias, srcColumnData.ColumnName)
	if err != nil {
		fmt.Println("source removeColumnInTableDb err--->", err)
		return err
	}

	err = s.deleteLookups(ctx, relationId, srcColumnData.ModelID.String(), schemaName)
	if err != nil {
		fmt.Println("deleteLookups err--->", err)
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
		return s.handleDeleteColumnForLink(ctx, schemaName, columnResponse)
	}

	err := s.columnsService.DeleteColumn(ctx, schemaName, columnData.ID.String())
	if err != nil {
		return err
	}

	return nil
}

func (s tableManagementService) ReorderColumn(
	ctx context.Context,
	schemaName string,
	req dto.ReorderColumnRequest,
) ([]dto.ColumnResponse, error) {

	sourceColumnData, err := s.columnsService.GetColumnByID(ctx, schemaName, req.SourceColumnID.String())
	if err != nil {
		fmt.Println("sourceColumnData------")
		return []dto.ColumnResponse{}, err
	}

	targetColumnData, err := s.columnsService.GetColumnByID(ctx, schemaName, req.TargetColumnID.String())
	if err != nil {
		fmt.Println("targetColumnData------")
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
		return app_errors.DatabaseError
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
		return s.handleDeleteColumnForLink(ctx, schemaName, columnData)
	}

	ok := s.allowUpdate(columnData)
	if !ok {
		return app_errors.DeleteNotAllowed
	}

	// Proceed to delete the column
	err = s.columnsService.DeleteColumn(ctx, schemaName, id)
	if err != nil {
		return err
	}

	model, err := s.modelService.GetModelByID(ctx, schemaName, columnData.ModelID.String())
	if err != nil {
		return err
	}

	if columnData.UIDT == "lookup" {
		fmt.Println("DeleteColumn...... ")
		lookupColumnID, relationID, ok := s.validateMetaForLookup(columnData.Meta)

		lookupColumn, err := s.columnsService.GetColumnByID(ctx, schemaName, lookupColumnID)
		if err != nil {
			return err
		}

		if ok {
			_ = s.removeLookupColumnInRelation(ctx, schemaName, columnData.ModelID.String(), relationID, lookupColumn.ColumnName)
		}
		return nil
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

func (s tableManagementService) CreateRow(ctx context.Context, schemaName string, req dto.CreateRowRequest) (dto.RecordResponse, error) {

	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.RecordResponse{}, err
	}

	tableName := fmt.Sprintf("\"%s\".\"%s\"", schemaName, model.Alias)

	data := map[string]interface{}{
		"created_by":         req.CreatedBy,
		"last_modified_by":   req.CreatedBy,
		"created_time":       time.Now().UTC(),
		"last_modified_time": time.Now().UTC(),
	}

	createdRecord, err := s.repo.TableService.CreateRecord(ctx, tableName, data)
	if err != nil {
		fmt.Println("err::::;", err)
		return dto.RecordResponse{}, app_errors.DatabaseError
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

	tableName := fmt.Sprintf("\"%s\".\"%s\"", schemaName, model.Alias)
	params := dbModels.QueryParams{
		OrderBy: []string{"id"},
	}
	records, err := s.repo.TableService.GetTableData(ctx, tableName, params)
	if err != nil {
		return dto.RecordsResponse{}, app_errors.DatabaseError
	}

	return dto.RecordsResponse{
		Records: records,
	}, nil
}

// func (s tableManagementService) getAllRecordsIncludingAssets(ctx context.Context, schemaName string, tableName string, columnsData []dto.ColumnResponse) (dto.RecordsResponse, error) {
// 	functionName := "dynamic_array_join_assets_jsonb"
// 	schemaFunctionName := fmt.Sprintf("%s.%s", constant.MasterDatabase, functionName)
// 	// Prepare the parameters for the function: schema_name TEXT, source_table TEXT, source_columns TEXT[], target_table TEXT
// 	sourceColumns := []string{}
// 	for _, col := range columnsData {
// 		if col.UIDT == "attachment" {
// 			sourceColumns = append(sourceColumns, col.ColumnName)
// 		}
// 	}

// 	args := map[string]interface{}{
// 		"schema_name":    schemaName,
// 		"source_table":   tableName,
// 		"source_columns": sourceColumns,
// 		"target_table":   "assets",
// 	}

// 	records, err := s.repo.TableService.GetByFunction(
// 		ctx,
// 		schemaFunctionName,
// 		args,
// 	)

// 	var normalizedRecord []map[string]interface{}
// 	for _, record := range records {
// 		if rec, ok := record[functionName].(map[string]interface{}); ok {
// 			normalizedRecord = append(normalizedRecord, rec)
// 		}
// 	}

// 	if err != nil {
// 		return dto.RecordsResponse{}, err
// 	}

// 	return dto.RecordsResponse{
// 		Records: normalizedRecord,
// 	}, nil
// }

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

func (s tableManagementService) GetRecordsWithPagination(ctx context.Context, schemaName string, tableName string, columnsData []dto.ColumnResponse, page_size int, page_number int) (dto.RecordsResponse, error) {
	functionName := "get_paginated_relations"
	schemaFunctionName := fmt.Sprintf("%s.%s", constant.MasterDatabase, functionName)
	fmt.Println(" page_size int, page_number int", page_size, page_number)

	// check if lookup available
	relationIds := s.checkLookuup(columnsData)
	var relation_data []map[string]interface{}
	if len(relationIds) > 0 {
		relation_data = []map[string]interface{}{}
		for _, col := range columnsData {
			if col.UIDT == "links" {
				r_data := map[string]interface{}{
					"source_column_name": col.ColumnName,
				}

				relationId, _ := col.Meta["relation_id"].(string)
				// check relationId in relationIds if not exists continue loop
				found := false
				for _, relID := range relationIds {
					if relationId == relID {
						found = true
						break
					}
				}
				if !found {
					continue
				}

				entityRole, _ := col.Meta["entity_role"].(string)

				relation, err := s.relationshipService.GetRelationByID(ctx, relationId, schemaName)
				if err != nil {
					return dto.RecordsResponse{}, err
				}

				r_data["relation"] = relation.RelationType
				if entityRole == "source" {
					if len(relation.SourceLookupColumns) == 0 {
						continue
					}
					targetModel, err := s.modelService.GetModelByID(ctx, schemaName, relation.TargetModelID)
					if err != nil {
						return dto.RecordsResponse{}, err
					}
					r_data["target_table_name"] = targetModel.Alias
					r_data["target_column_name"] = "id"
					r_data["target_columns"] = relation.SourceLookupColumns
				} else {
					if len(relation.TargetLookupColumns) == 0 {
						continue
					}
					targetModel, err := s.modelService.GetModelByID(ctx, schemaName, relation.SourceModelID)
					if err != nil {
						return dto.RecordsResponse{}, err
					}
					r_data["target_table_name"] = targetModel.Alias
					r_data["target_column_name"] = "id"
					r_data["target_columns"] = relation.TargetLookupColumns
				}

				relation_data = append(relation_data, r_data)
			}
		}
	} else {
		relation_data = nil
	}

	args := map[string]interface{}{
		"schema_name":       schemaName,
		"source_table_name": tableName,
		"relation_data":     relation_data,
		"page_size":         page_size,
		"page_number":       page_number,
	}

	fmt.Println("args==============", args)

	records, err := s.repo.TableService.GetByFunction(
		ctx,
		schemaFunctionName,
		args,
	)
	if err != nil {
		return dto.RecordsResponse{}, err
	}

	if len(records) == 0 {
		return dto.RecordsResponse{Records: nil}, nil
	}

	var normalizedRecord []map[string]interface{}
	getPaginated, ok := records[0]["get_paginated_relations"]
	if ok {
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
	}
	return dto.RecordsResponse{Records: normalizedRecord}, nil
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

	records, err := s.repo.TableService.GetTableData(ctx, tableName, params)
	if err != nil {
		return nil, app_errors.DatabaseError
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

	records, err := s.repo.TableService.GetTableData(ctx, tableName, params)
	if err != nil {
		return nil, app_errors.DatabaseError
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

	records, err := s.repo.TableService.GetTableData(ctx, tableName, params)
	if err != nil {
		return nil, app_errors.DatabaseError
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
		var updatedArr []int64

		switch v := rowData[columnName].(type) {
		case nil:
			// No value yet, just add the new value
			updatedArr = []int64{int64(value)}
		case []int64:
			// Check for existence
			exists := false
			for _, item := range v {
				if item == int64(value) {
					exists = true
					break
				}
			}
			if !exists {
				updatedArr = append(v, int64(value))
			} else {
				updatedArr = v
			}
		case []string:
			// Convert []string to []int64
			for _, s := range v {
				if n, err := strconv.ParseInt(s, 10, 64); err == nil {
					updatedArr = append(updatedArr, n)
				}
			}
			// Check for existence
			exists := false
			for _, item := range updatedArr {
				if item == int64(value) {
					exists = true
					break
				}
			}
			if !exists {
				updatedArr = append(updatedArr, int64(value))
			}
		case int64:
			// Single int64 value, convert to array
			if v == int64(value) {
				updatedArr = []int64{v}
			} else {
				updatedArr = []int64{v, int64(value)}
			}
		case int:
			// Single int value, convert to array
			if v == value {
				updatedArr = []int64{int64(v)}
			} else {
				updatedArr = []int64{int64(v), int64(value)}
			}
		default:
			// Fallback: just set as array with the new value
			updatedArr = []int64{int64(value)}
		}

		data := map[string]interface{}{
			columnName:           updatedArr,
			"last_modified_time": time.Now().UTC(),
		}
		if updatedBy != "" {
			data["last_modified_by"] = updatedBy
		}
		return s.repo.TableService.UpdateRecord(ctx, tableName, rowId, data)

	case "INT":
		data := map[string]interface{}{
			columnName:           value,
			"last_modified_time": time.Now().UTC(),
		}
		if updatedBy != "" {
			data["last_modified_by"] = updatedBy
		}
		return s.repo.TableService.UpdateRecord(ctx, tableName, rowId, data)

	default:
		return nil, fmt.Errorf("unsupported datatype: %s", datatype)
	}
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
		var arrInt64 []int64

		switch arr := rowData[columnName].(type) {
		case []int64:
			arrInt64 = arr
		case []int:
			for _, v := range arr {
				arrInt64 = append(arrInt64, int64(v))
			}
		case []string:
			for _, s := range arr {
				if n, err := strconv.ParseInt(s, 10, 64); err == nil {
					arrInt64 = append(arrInt64, n)
				}
			}
		case int64:
			arrInt64 = []int64{arr}
		case int:
			arrInt64 = []int64{int64(arr)}
		case nil:
			// nothing to unlink
			return rowData, nil
		default:
			// Not an array or convertible, just return as is
			return rowData, nil
		}

		// Remove the value from the array
		newArr := make([]int64, 0, len(arrInt64))
		for _, v := range arrInt64 {
			if int(v) != value {
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
		return s.repo.TableService.UpdateRecord(ctx, tableName, rowId, data)

	case "INT":
		val, ok := rowData[columnName].(int64)
		if !ok {
			// Try to handle int type as well
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
			return s.repo.TableService.UpdateRecord(ctx, tableName, rowId, data)
		}
		return rowData, nil

	default:
		// For other datatypes, just return the rowData as is
		return rowData, nil
	}
}

func (s tableManagementService) updateLinkData(
	ctx context.Context,
	sourceTableName string,
	targetTableName string,
	sourceColumnName string,
	targetColumnName string,
	sourceDataType string,
	targetDataType string,
	req dto.UpdateRowDataLinksRequest,
) (dto.RecordResponse, error) {
	var (
		sourceInsertedRecord map[string]interface{}
		err                  error
	)
	switch req.Action {
	case "link":
		sourceInsertedRecord, err = s.linkRecord(ctx, sourceDataType, sourceTableName, req.SourceRowId, sourceColumnName, req.TargetRowId, req.UpdatedBy)
	default:
		sourceInsertedRecord, err = s.unlinkRecord(ctx, sourceDataType, sourceTableName, req.SourceRowId, sourceColumnName, req.TargetRowId, req.UpdatedBy)
	}
	if err != nil {
		return dto.RecordResponse{}, app_errors.DatabaseError
	}

	switch req.Action {
	case "link":
		_, err = s.linkRecord(ctx, targetDataType, targetTableName, req.TargetRowId, targetColumnName, req.SourceRowId, req.UpdatedBy)
	default:
		_, err = s.unlinkRecord(ctx, targetDataType, targetTableName, req.TargetRowId, targetColumnName, req.SourceRowId, req.UpdatedBy)
	}
	if err != nil {
		return dto.RecordResponse{}, app_errors.DatabaseError
	}

	return dto.RecordResponse{
		Record: sourceInsertedRecord,
	}, nil
}

func (s tableManagementService) updateIfExist(
	ctx context.Context, relationType string,
	sourceTableName, sourceColumnName, targetTableName, targetColumnName string,
	sourceDataType string, targetDataType string,
	req dto.UpdateRowDataLinksRequest,
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
		{sourceTableName, sourceColumnName, sourceDataType, targetTableName, targetColumnName, targetDataType, req.TargetRowId},
		{targetTableName, targetColumnName, targetDataType, sourceTableName, sourceColumnName, sourceDataType, req.SourceRowId},
	}

	for _, c := range checks {
		switch {
		case relationType == "one-to-one":
			if err := s.handleOneToOneRelation(ctx, c, req); err != nil {
				return err
			}
		case relationType == "has-many" && c.srcDatatype == "INT[]":
			if err := s.handleHasManyIntArrayRelation(ctx, c, req); err != nil {
				return err
			}
			// case relationType == "has-many" && c.srcDatatype == "INT":
			// 	if err := s.handleHasManyIntRelation(ctx, c, req); err != nil {
			// 		return err
			// 	}
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
		_, err = s.updateLinkData(ctx, c.srcTable, c.trgTable, c.srcColumn, c.trgColumn, c.srcDatatype, c.trgDataType, req)
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
	fmt.Println("has-many", c.srcTable, c.srcColumn, c.id)
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
		_, err = s.updateLinkData(ctx, c.srcTable, c.trgTable, c.srcColumn, c.trgColumn, c.srcDatatype, c.trgDataType, req)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s tableManagementService) handleHasManyIntRelation(
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
	if data != nil {
		srcID, _ := data["id"].(int64)
		tgtID := c.id
		req.SourceRowId = int(srcID)
		req.TargetRowId = int(tgtID)
		req.Action = "unlink"
		_, err = s.updateLinkData(ctx, c.srcTable, c.trgTable, c.srcColumn, c.trgColumn, c.srcDatatype, c.trgDataType, req)
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

	sourceTableName := fmt.Sprintf("\"%s\".\"%s\"", schemaName, sourceModel.Alias)

	relationId, ok := sourceColumnData.Meta["relation_id"].(string)
	if !ok {
		return dto.RecordResponse{}, app_errors.ErrInternal
	}

	relationData, err := s.relationshipService.GetRelationByID(ctx, relationId, schemaName)
	if err != nil {
		return dto.RecordResponse{}, app_errors.DatabaseError
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

	targetTableName := fmt.Sprintf("\"%s\".\"%s\"", schemaName, targetModel.Alias)

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
		err = s.updateIfExist(ctx, relationType, sourceTableName, sourceColumnData.ColumnName, targetTableName, targetColumnData.ColumnName, sourceDataType, targetDataType, req)
		if err != nil {
			return dto.RecordResponse{}, err
		}
	}

	return s.updateLinkData(ctx, sourceTableName, targetTableName, sourceColumnData.ColumnName, targetColumnData.ColumnName, sourceDataType, targetDataType, req)
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

	tableName := fmt.Sprintf("\"%s\".\"%s\"", schemaName, model.Alias)

	fmt.Printf("DEBUG: InsertRowData - Column: %s, DT: %s, Value: %v\n", columnData.ColumnName, columnData.DT, req.Value)

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
		fmt.Sprintf("\"%s\"", columnData.ColumnName): value,
		"last_modified_by":                           req.UpdatedBy,
		"last_modified_time":                         time.Now().UTC(),
	}

	insertedRecord, err := s.repo.TableService.UpdateRecord(ctx, tableName, req.RowId, data)
	if err != nil {
		return dto.RecordResponse{}, app_errors.DatabaseError
	}

	return dto.RecordResponse{
		Record: insertedRecord,
	}, nil
}

func (s tableManagementService) CreateRowWithRecords(ctx context.Context, schemaName string, modelAlias string, record map[string]interface{}) (dto.RecordResponse, error) {
	tableName := fmt.Sprintf("\"%s\".\"%s\"", schemaName, modelAlias)

	createdRecord, err := s.repo.TableService.CreateRecord(ctx, tableName, record)
	if err != nil {
		fmt.Println("err::::;", err)
		return dto.RecordResponse{}, app_errors.DatabaseError
	}

	return dto.RecordResponse{
		Record: createdRecord,
	}, nil
}

func (s tableManagementService) CreateRowsWithRecordsBulk(ctx context.Context, schemaName string, modelAlias string, records []map[string]interface{}) ([]dto.RecordResponse, error) {
	tableName := fmt.Sprintf("\"%s\".\"%s\"", schemaName, modelAlias)

	createdRecords, err := s.repo.BulkService.BulkInsert(tableName, records)
	if err != nil {
		fmt.Println("bulk insert error:", err)
		return nil, app_errors.DatabaseError
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

	sourceTableName := fmt.Sprintf("\"%s\".\"%s\"", schemaName, sourceModel.Alias)
	targetTableName := fmt.Sprintf("\"%s\".\"%s\"", schemaName, targetModel.Alias)
	return s.unlinkRowData(ctx, req, sourceTableName, targetTableName, column, targetColumn, rowData, sourceDataType, targetDataType)
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
	req dto.DeleteRowDataRequest,
	sourceTableName, targetTableName string,
	column tenant.Column,
	targetColumn tenant.Column,
	rowData map[string]interface{},
	sourceDataType, targetDataType string,
) error {
	if sourceDataType == "INT" {
		fmt.Println("Unlink single one")
		targetRowId := rowData[column.ColumnName].(int64)
		return s.unlinkSingleRow(ctx, req, sourceTableName, targetTableName, column, targetColumn, sourceDataType, targetDataType, targetRowId)
	}

	// handle multiple (INT[])
	targetRowIds := rowData[targetColumn.ColumnName].([]int64)
	for _, targetRowId := range targetRowIds {
		if err := s.unlinkSingleRow(ctx, req, sourceTableName, targetTableName, column, targetColumn, sourceDataType, targetDataType, targetRowId); err != nil {
			return err
		}
	}
	return nil
}

// Build unlink request and call updateLinkData
func (s tableManagementService) unlinkSingleRow(
	ctx context.Context,
	req dto.DeleteRowDataRequest,
	sourceTableName, targetTableName string,
	column tenant.Column,
	targetColumn tenant.Column,
	sourceDataType, targetDataType string,
	targetRowId int64,
) error {
	updateLinkReq := dto.UpdateRowDataLinksRequest{
		ModelID:     req.ModelID,
		ColumnId:    column.ID.String(),
		SourceRowId: req.RowId,
		TargetRowId: int(targetRowId),
		Action:      "unlink",
	}

	_, err := s.updateLinkData(
		ctx,
		sourceTableName,
		targetTableName,
		column.ColumnName,
		targetColumn.ColumnName,
		sourceDataType,
		targetDataType,
		updateLinkReq,
	)
	return err
}

func (s tableManagementService) DeleteRow(ctx context.Context, schemaName string, req dto.DeleteRowDataRequest) error {
	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return err
	}

	tableName := fmt.Sprintf("\"%s\".\"%s\"", schemaName, model.Alias)
	rowData, err := s.getRowByID(ctx, tableName, req.RowId)
	if err != nil {
		return err
	}

	if err := s.handleDeleteRowForLinks(ctx, model, rowData, schemaName, req); err != nil {
		return err
	}

	if err := s.repo.TableService.DeleteRecord(ctx, tableName, req.RowId); err != nil {
		return app_errors.DatabaseError
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

func (s tableManagementService) AddAttachment(
	ctx context.Context,
	schemaName string,
	req dto.AddAttachmentRequest,
	files []*multipart.FileHeader,
) (dto.RecordResponse, error) {
	assets, err := s.uploadAssets(ctx, schemaName, files)
	if err != nil {
		return dto.RecordResponse{}, err
	}

	columnName, tableName, err := s.getColumnNameAndTableName(ctx, schemaName, req)
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

	insertedRecord, err := s.repo.TableService.UpdateRecord(ctx, tableName, req.RowId, data)
	if err != nil {
		fmt.Println("insertedRecord: ", err)
		return dto.RecordResponse{}, app_errors.DatabaseError
	}

	return dto.RecordResponse{
		Record: insertedRecord,
	}, nil
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

	tableName := fmt.Sprintf("\"%s\".\"%s\"", schemaName, model.Alias)

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
		fmt.Sprintf("\"%s\"", columnData.ColumnName): updatedAttachments,
		"last_modified_time":                         time.Now().UTC(),
	}

	updatedRecord, err := s.repo.TableService.UpdateRecord(ctx, tableName, req.RowId, data)
	if err != nil {
		return dto.RecordResponse{}, app_errors.DatabaseError
	}

	return dto.RecordResponse{
		Record: updatedRecord,
	}, nil
}

func (s tableManagementService) uploadAssets(ctx context.Context, schemaName string, files []*multipart.FileHeader) ([]tenant.Assets, error) {
	uploadReq := dto.UploadAssetRequest{
		Files: files,
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
	req dto.AddAttachmentRequest,
) (string, string, error) {
	columnData, err := s.GetColumnById(ctx, schemaName, req.ColumnId)
	if err != nil {
		return "", "", err
	}

	ok := s.allowInsert(columnData)
	if !ok {
		return "", "", app_errors.UpdateNotAllowed
	}

	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return "", "", err
	}

	tableName := fmt.Sprintf("\"%s\".\"%s\"", schemaName, model.Alias)
	return columnData.ColumnName, tableName, nil
}

func (s tableManagementService) mergeAttachmentValues(existing interface{}, assets []map[string]interface{}) []map[string]interface{} {
	attachmentValue := s.checkAttachmentType(existing)
	for _, asset := range assets {
		attachmentValue = append(attachmentValue, asset)
	}
	return attachmentValue
}
