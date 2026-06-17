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
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

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

var (
	symbolCharSet = map[rune]struct{}{
		'@': {}, '#': {}, '$': {}, '%': {}, '^': {}, '&': {}, '*': {},
		'!': {}, '~': {}, '`': {}, '|': {}, '\\': {},
	}
	currencyCharSet = map[rune]struct{}{
		'₹': {}, '$': {}, '€': {}, '£': {}, '¥': {},
	}
	bracketCharSet = map[rune]struct{}{
		'(': {}, ')': {}, '[': {}, ']': {}, '{': {}, '}': {}, '<': {}, '>': {},
	}
	punctuationCharSet = map[rune]struct{}{
		'.': {}, ',': {}, ';': {}, ':': {}, '!': {}, '?': {}, '\'': {}, '"': {}, '-': {},
	}
	// Collapse only runs of whitespace that occur between two non-space characters
	multiSpaceBetweenWordsRegex = regexp.MustCompile(`(\S)\s{2,}(\S)`)
	currencySymbolRegex         = regexp.MustCompile(`[₹$€£¥]`)
	numericSeparatorRegex       = regexp.MustCompile(`,`)
	flexibleDateLayouts         = []string{
		time.RFC3339Nano, time.RFC3339,
		"2006-01-02 15:04:05", "2006-01-02 15:04:05.999999", "2006-01-02 15:04",
		"2006-01-02", "20060102", "02-01-2006", "02/01/2006", "02.01.2006",
		"02 01 2006",
		"01-02-2006", "01/02/2006", "01.02.2006",
		"2006/01/02", "2006.01.02",
		"02 Jan 2006", "02 January 2006", "Jan 02 2006", "January 02 2006",
	}
	regexEmail    = regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`)
	regexURL      = regexp.MustCompile(`(?i)\b(?:https?://|www\.)[^\s<>()]+`)
	regexHashtag  = regexp.MustCompile(`(?:^|\s)(#[A-Za-z0-9_]+)`)
	regexMention  = regexp.MustCompile(`(?:^|\s)(@[A-Za-z0-9_.-]+)`)
	regexKeywords = regexp.MustCompile(`[\p{L}\p{N}]+`)
	regexEmoji    = regexp.MustCompile(`(?:[\x{1F1E6}-\x{1F1FF}]{2}|[#*0-9]\x{FE0F}?\x{20E3}|[\x{00A9}\x{00AE}\x{203C}\x{2049}\x{2122}\x{2139}\x{2194}-\x{21AA}\x{231A}-\x{231B}\x{2328}\x{23CF}\x{23E9}-\x{23F3}\x{23F8}-\x{23FA}\x{24C2}\x{25AA}-\x{25AB}\x{25B6}\x{25C0}\x{25FB}-\x{25FE}\x{2600}-\x{27BF}\x{2934}-\x{2935}\x{2B05}-\x{2B07}\x{2B1B}-\x{2B1C}\x{2B50}\x{2B55}\x{3030}\x{303D}\x{3297}\x{3299}\x{1F000}-\x{1FAFF}](?:\x{FE0F}|\x{FE0E})?(?:\x{200D}[\x{00A9}\x{00AE}\x{203C}\x{2049}\x{2122}\x{2139}\x{2194}-\x{21AA}\x{231A}-\x{231B}\x{2328}\x{23CF}\x{23E9}-\x{23F3}\x{23F8}-\x{23FA}\x{24C2}\x{25AA}-\x{25AB}\x{25B6}\x{25C0}\x{25FB}-\x{25FE}\x{2600}-\x{27BF}\x{2934}-\x{2935}\x{2B05}-\x{2B07}\x{2B1B}-\x{2B1C}\x{2B50}\x{2B55}\x{3030}\x{303D}\x{3297}\x{3299}\x{1F000}-\x{1FAFF}](?:\x{FE0F}|\x{FE0E})?)*)`)
	regexPhone    = regexp.MustCompile(`(?:^|[^\d])(\+?\d[\d\s().-]{7,}\d)(?:$|[^\d])`)
)

type columnSplitStrategy struct {
	kind      string
	separator string
	action    string
	value     int
	pattern   string
	regex     *regexp.Regexp
}

type splitColumnRow struct {
	id    interface{}
	value string
	parts []string
}

// targetColumnParams holds parameters for creating target column in relation
type targetColumnParams struct {
	ColumnData      dto.AddColumnRequest
	SourceModelData tenant.Model
	RelationWith    string
	RelationID      uuid.UUID
	RelationType    string
	Now             time.Time
}

// --- extraction helper functions ---

func cleanExtractionMatch(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), ".,;:!?)")
}

func extractFirstEmail(s string) (string, bool) {
	m := cleanExtractionMatch(regexEmail.FindString(s))
	if m != "" {
		return m, true
	}
	return "", false
}

func extractFirstURL(s string) (string, bool) {
	m := cleanExtractionMatch(regexURL.FindString(s))
	if m != "" {
		return m, true
	}
	return "", false
}

func extractURLsFromText(s string) (string, bool) {
	matches := regexURL.FindAllString(s, -1)
	if len(matches) == 0 {
		return "", false
	}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		cleaned := cleanExtractionMatch(m)
		if cleaned == "" {
			continue
		}
		out = append(out, cleaned)
	}
	if len(out) == 0 {
		return "", false
	}
	return strings.Join(out, ", "), true
}

func extractDomainFromText(s string) (string, bool) {
	if email, ok := extractFirstEmail(s); ok {
		parts := strings.SplitN(email, "@", 2)
		if len(parts) == 2 {
			domain := strings.TrimSpace(parts[1])
			domain = strings.Trim(domain, ",;:)}]")
			domain = strings.TrimPrefix(domain, "www.")
			return domain, true
		}
	}
	if uStr, ok := extractFirstURL(s); ok {
		if u, err := url.Parse(uStr); err == nil {
			host := u.Hostname()
			host = strings.TrimPrefix(host, "www.")
			return host, true
		}
	}
	return "", false
}

func extractHashtagsFromText(s string) (string, bool) {
	matches := regexHashtag.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return "", false
	}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		raw := strings.TrimSpace(m[1])
		if raw == "" {
			continue
		}
		out = append(out, raw)
	}
	if len(out) == 0 {
		return "", false
	}
	return strings.Join(out, ", "), true
}

func extractMentionsFromText(s string) (string, bool) {
	matches := regexMention.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return "", false
	}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		raw := strings.TrimSpace(m[1])
		if raw == "" {
			continue
		}
		out = append(out, raw)
	}
	if len(out) == 0 {
		return "", false
	}
	return strings.Join(out, ", "), true
}

func extractKeywordsFromText(s string) (string, bool) {
	matches := regexKeywords.FindAllString(s, -1)
	if len(matches) == 0 {
		return "", false
	}
	stopWords := map[string]struct{}{
		"a": {}, "an": {}, "and": {}, "the": {}, "or": {}, "but": {}, "to": {}, "of": {}, "in": {}, "on": {}, "at": {}, "for": {}, "with": {}, "from": {}, "by": {},
		"is": {}, "are": {}, "was": {}, "were": {}, "be": {}, "been": {}, "it": {}, "this": {}, "that": {}, "these": {}, "those": {}, "as": {}, "into": {},
		"over": {}, "under": {}, "about": {}, "after": {}, "before": {}, "between": {}, "through": {}, "during": {}, "without": {}, "within": {},
	}

	seen := make(map[string]struct{})
	out := make([]string, 0, len(matches))
	for _, tok := range matches {
		t := strings.TrimSpace(tok)
		if t == "" {
			continue
		}
		tl := strings.ToLower(t)
		if _, ok := stopWords[tl]; ok {
			continue
		}
		if len([]rune(t)) <= 2 {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	if len(out) == 0 {
		return "", false
	}
	// limit to 20 as frontend
	if len(out) > 20 {
		out = out[:20]
	}
	return strings.Join(out, ", "), true
}

func extractEmojiFromText(s string) (string, bool) {
	out := regexEmoji.FindAllString(s, -1)
	if len(out) == 0 {
		return "", false
	}
	return strings.Join(out, ", "), true
}

func extractPhoneNumberFromText(s string) (string, bool) {
	m := regexPhone.FindStringSubmatch(s)
	if len(m) < 2 {
		return "", false
	}
	phone := strings.TrimSpace(m[1])
	if phone == "" {
		return "", false
	}
	return phone, true
}

func extractEmailPrefixFromText(s string) (string, bool) {
	matches := regexEmail.FindAllString(s, -1)
	if len(matches) == 0 {
		return "", false
	}
	prefixes := make([]string, 0, len(matches))
	for _, email := range matches {
		parts := strings.SplitN(strings.TrimSpace(email), "@", 2)
		if len(parts) < 2 {
			continue
		}
		prefix := strings.TrimSpace(parts[0])
		if prefix == "" {
			continue
		}
		prefixes = append(prefixes, prefix)
	}
	if len(prefixes) == 0 {
		return "", false
	}
	return strings.Join(prefixes, ", "), true
}

func extractBetweenCharactersFromText(s, startAfter, endBefore string) (string, bool) {
	if startAfter == "" || endBefore == "" {
		return "", false
	}
	startIdx := strings.Index(s, startAfter)
	if startIdx == -1 {
		return "", false
	}
	startPos := startIdx + len(startAfter)
	if startPos >= len(s) {
		return "", false
	}
	rest := s[startPos:]
	endIdx := strings.Index(rest, endBefore)
	if endIdx == -1 {
		return "", false
	}
	extracted := strings.TrimSpace(rest[:endIdx])
	if extracted == "" {
		return "", false
	}
	return extracted, true
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

func (s tableManagementService) TrimWhitespace(ctx context.Context, schemaName string, req dto.TrimWhitespaceRequest) (dto.TrimWhitespaceResponse, error) {
	lg := logger.Get()

	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.TrimWhitespaceResponse{}, err
	}

	columnsData, err := s.GetColumnsByModelID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.TrimWhitespaceResponse{}, err
	}

	selectedColumns, err := s.getSelectedColumnsFromRequest(columnsData, req.Columns)
	if err != nil {
		return dto.TrimWhitespaceResponse{}, err
	}

	selectColumns := make([]string, 0, len(selectedColumns)+1)
	selectColumns = append(selectColumns, "id")
	selectColumns = append(selectColumns, selectedColumns...)

	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)
	rows, err := s.fetchTableRowsForTrim(ctx, tableName, selectColumns)
	if err != nil {
		return dto.TrimWhitespaceResponse{}, err
	}

	updates, result := s.buildTrimUpdates(rows, selectedColumns, req.TrimMode)

	if len(updates) > 0 {
		if err := s.columnsService.BulkUpdateByColumns(ctx, schemaName, model.Alias, updates); err != nil {
			return dto.TrimWhitespaceResponse{}, err
		}
	}

	lg.Info().
		Str("model_id", req.ModelID).
		Int("columns_selected", len(selectedColumns)).
		Int("total_scanned", result.TotalScanned).
		Int("total_updated", result.TotalUpdated).
		Int("total_skipped", result.TotalSkipped).
		Int("total_rows", result.TotalRows).
		Int("total_rows_updated", result.TotalRowsUpdated).
		Int("total_rows_skipped", result.TotalRowsSkipped).
		Msg("Trim whitespace action completed")

	return result, nil
}

func (s tableManagementService) getSelectedColumnsFromRequest(columnsData []dto.ColumnResponse, requested []string) ([]string, error) {
	columnSet := make(map[string]string, len(columnsData))
	for _, col := range columnsData {
		columnSet[col.ID.String()] = col.ColumnName
	}

	selectedColumns := make([]string, 0, len(requested))
	seen := make(map[string]struct{}, len(requested))
	for _, colID := range requested {
		columnID := strings.TrimSpace(colID)
		if columnID == "" {
			continue
		}
		if _, ok := seen[columnID]; ok {
			continue
		}
		columnName, exists := columnSet[columnID]
		if !exists {
			return nil, app_errors.ColumnNotFound
		}
		seen[columnID] = struct{}{}
		selectedColumns = append(selectedColumns, columnName)
	}
	if len(selectedColumns) == 0 {
		return nil, app_errors.InvalidPayload
	}
	return selectedColumns, nil
}

func (s tableManagementService) fetchTableRowsForTrim(ctx context.Context, tableName string, selectColumns []string) ([]map[string]interface{}, error) {
	rows, err := s.repo.TableService.GetTableData(tableName, dbModels.QueryParams{Select: selectColumns})
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to fetch rows for trim whitespace")
	}
	return rows, nil
}

func (s tableManagementService) buildTrimUpdates(rows []map[string]interface{}, selectedColumns []string, trimMode string) ([]dto.UpdateColumnValueRequest, dto.TrimWhitespaceResponse) {
	result := dto.TrimWhitespaceResponse{
		TotalScanned: len(rows) * len(selectedColumns),
		TotalRows:    len(rows),
	}

	if len(rows) == 0 {
		return nil, result
	}

	updates := make([]dto.UpdateColumnValueRequest, 0)

	for _, row := range rows {
		rowID, hasRowID := row["id"]
		if !hasRowID {
			result.TotalSkipped += len(selectedColumns)
			result.TotalRowsSkipped++
			continue
		}
		// process columns for this row in a helper to reduce cognitive complexity
		rowUpdates, skipped, rowUpdated := s.buildTrimUpdatesForRow(rowID, row, selectedColumns, trimMode)

		if skipped > 0 {
			result.TotalSkipped += skipped
		}
		if len(rowUpdates) > 0 {
			updates = append(updates, rowUpdates...)
			result.TotalUpdated += len(rowUpdates)
		}
		if rowUpdated {
			result.TotalRowsUpdated++
		} else {
			result.TotalRowsSkipped++
		}
	}

	return updates, result
}

// buildTrimUpdatesForRow processes a single row and returns the required updates,
// the number of skipped cells for that row, and whether any cell in the row was
// updated. Keeping this as a helper reduces nesting and cognitive load in the
// parent function without changing behavior.
func (s tableManagementService) buildTrimUpdatesForRow(rowID interface{}, row map[string]interface{}, selectedColumns []string, trimMode string) ([]dto.UpdateColumnValueRequest, int, bool) {
	updates := make([]dto.UpdateColumnValueRequest, 0)
	skipped := 0
	rowUpdated := false

	for _, columnName := range selectedColumns {
		value, exists := row[columnName]
		if !exists || value == nil {
			skipped++
			continue
		}

		strValue, ok := value.(string)
		if !ok {
			skipped++
			continue
		}

		cleaned := cleanWhitespaceValue(strValue, trimMode)
		if cleaned == strValue {
			skipped++
			continue
		}

		updates = append(updates, dto.UpdateColumnValueRequest{
			Id:     rowID,
			Column: columnName,
			Value:  cleaned,
		})
		rowUpdated = true
	}

	return updates, skipped, rowUpdated
}

func cleanWhitespaceValue(value, trimMode string) string {
	cleaned := value
	switch trimMode {
	case "trim_both":
		cleaned = strings.TrimSpace(cleaned)
	case "trim_leading":
		cleaned = strings.TrimLeftFunc(cleaned, unicode.IsSpace)
	case "trim_trailing":
		cleaned = strings.TrimRightFunc(cleaned, unicode.IsSpace)
	case "collapse_spaces":
		cleaned = collapseInternalSpaces(cleaned)
	default:
		cleaned = strings.TrimSpace(cleaned)
	}

	return cleaned
}

// collapseInternalSpaces collapses runs of 2+ whitespace that occur between
// non-space characters into a single ASCII space, preserving leading/trailing whitespace.
func collapseInternalSpaces(s string) string {
	return multiSpaceBetweenWordsRegex.ReplaceAllString(s, "$1 $2")
}

func (s tableManagementService) CaseNormalization(ctx context.Context, schemaName string, req dto.CaseNormalizationRequest) (dto.CaseNormalizationResponse, error) {
	lg := logger.Get()

	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.CaseNormalizationResponse{}, err
	}

	columnsData, err := s.GetColumnsByModelID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.CaseNormalizationResponse{}, err
	}

	selectedColumns, err := s.getSelectedColumnsFromRequest(columnsData, req.Columns)
	if err != nil {
		return dto.CaseNormalizationResponse{}, err
	}

	selectColumns := make([]string, 0, len(selectedColumns)+1)
	selectColumns = append(selectColumns, "id")
	selectColumns = append(selectColumns, selectedColumns...)

	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)
	rows, err := s.fetchTableRowsForTrim(ctx, tableName, selectColumns)
	if err != nil {
		return dto.CaseNormalizationResponse{}, err
	}

	updates, result := s.buildCaseNormalizationUpdates(rows, selectedColumns, req.CaseFormat)

	if len(updates) > 0 {
		if err := s.columnsService.BulkUpdateByColumns(ctx, schemaName, model.Alias, updates); err != nil {
			return dto.CaseNormalizationResponse{}, err
		}
	}

	lg.Info().
		Str("model_id", req.ModelID).
		Int("columns_selected", len(selectedColumns)).
		Int("total_scanned", result.TotalScanned).
		Int("total_updated", result.TotalUpdated).
		Int("total_skipped", result.TotalSkipped).
		Int("total_rows", result.TotalRows).
		Int("total_rows_updated", result.TotalRowsUpdated).
		Int("total_rows_skipped", result.TotalRowsSkipped).
		Str("case_format", req.CaseFormat).
		Msg("Case normalization action completed")

	return result, nil
}

func (s tableManagementService) buildCaseNormalizationUpdates(rows []map[string]interface{}, selectedColumns []string, caseFormat string) ([]dto.UpdateColumnValueRequest, dto.CaseNormalizationResponse) {
	result := dto.CaseNormalizationResponse{
		TotalScanned: len(rows) * len(selectedColumns),
		TotalRows:    len(rows),
	}

	if len(rows) == 0 {
		return nil, result
	}

	updates := make([]dto.UpdateColumnValueRequest, 0)

	for _, row := range rows {
		rowID, hasRowID := row["id"]
		if !hasRowID {
			result.TotalSkipped += len(selectedColumns)
			result.TotalRowsSkipped++
			continue
		}
		// Delegate per-row processing to helper to reduce nesting and complexity
		rowUpdates, skipped, rowUpdated := s.buildCaseNormalizationUpdatesForRow(rowID, row, selectedColumns, caseFormat)

		if skipped > 0 {
			result.TotalSkipped += skipped
		}
		if len(rowUpdates) > 0 {
			updates = append(updates, rowUpdates...)
			result.TotalUpdated += len(rowUpdates)
		}
		if rowUpdated {
			result.TotalRowsUpdated++
		} else {
			result.TotalRowsSkipped++
		}
	}

	return updates, result
}

// buildCaseNormalizationUpdatesForRow processes a single row for case normalization
// and returns the updates required for that row, the number of skipped cells,
// and whether any cell in the row was updated. Behavior is identical to the
// original inline logic but extracted to reduce cognitive complexity.
func (s tableManagementService) buildCaseNormalizationUpdatesForRow(rowID interface{}, row map[string]interface{}, selectedColumns []string, caseFormat string) ([]dto.UpdateColumnValueRequest, int, bool) {
	updates := make([]dto.UpdateColumnValueRequest, 0)
	skipped := 0
	rowUpdated := false

	for _, columnName := range selectedColumns {
		value, exists := row[columnName]
		if !exists || value == nil {
			skipped++
			continue
		}

		strValue, ok := value.(string)
		if !ok {
			skipped++
			continue
		}

		normalized := normalizeValue(strValue, caseFormat)
		if normalized == strValue {
			skipped++
			continue
		}

		updates = append(updates, dto.UpdateColumnValueRequest{
			Id:     rowID,
			Column: columnName,
			Value:  normalized,
		})
		rowUpdated = true
	}

	return updates, skipped, rowUpdated
}

func normalizeValue(value, caseFormat string) string {
	switch caseFormat {
	case "lowercase":
		return strings.ToLower(value)
	case "uppercase":
		return strings.ToUpper(value)
	case "title_case":
		return toTitleCase(value)
	case "sentence_case":
		return toSentenceCase(value)
	default:
		return value
	}
}

// toTitleCase capitalizes the first letter of each word and lowercases the rest.
func toTitleCase(s string) string {
	var b strings.Builder
	prevIsLetter := false
	for _, r := range s {
		if !prevIsLetter && unicode.IsLetter(r) {
			b.WriteRune(unicode.ToUpper(r))
			prevIsLetter = true
			continue
		}
		if unicode.IsLetter(r) {
			b.WriteRune(unicode.ToLower(r))
			prevIsLetter = true
			continue
		}
		// non-letter resets word boundary but preserved as-is
		b.WriteRune(r)
		prevIsLetter = false
	}
	return b.String()
}

// toSentenceCase capitalizes the first letter of each sentence and lowercases the rest.
func toSentenceCase(s string) string {
	var b strings.Builder
	startOfSentence := true
	for _, r := range s {
		if startOfSentence && unicode.IsLetter(r) {
			b.WriteRune(unicode.ToUpper(r))
			startOfSentence = false
			continue
		}
		if unicode.IsLetter(r) {
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		b.WriteRune(r)
		// mark start of sentence when we hit terminal punctuation
		if r == '.' || r == '!' || r == '?' {
			startOfSentence = true
		}
	}
	return b.String()
}

// FindReplace searches for `findValue` in selected columns and replaces with `replaceValue`.
// It processes rows in batches to limit memory usage and uses the existing
// BulkUpdateByColumns repository function to perform efficient column-level updates.
func (s tableManagementService) FindReplace(ctx context.Context, schemaName string, req dto.FindReplaceRequest) (dto.FindReplaceResponse, error) {
	lg := logger.Get()

	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.FindReplaceResponse{}, err
	}

	columnsData, err := s.GetColumnsByModelID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.FindReplaceResponse{}, err
	}

	selectedColumns, err := s.getSelectedColumnsFromRequest(columnsData, req.Columns)
	if err != nil {
		return dto.FindReplaceResponse{}, err
	}

	selectColumns := make([]string, 0, len(selectedColumns)+1)
	selectColumns = append(selectColumns, "id")
	selectColumns = append(selectColumns, selectedColumns...)

	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)

	batchSize := 1000
	limit := batchSize
	offset := 0

	totalResult := dto.FindReplaceResponse{}

	// Pre-compile regex for ignore_case for performance
	var ignoreRe *regexp.Regexp
	if req.MatchType == "ignore_case" {
		ignoreRe = regexp.MustCompile("(?i)" + regexp.QuoteMeta(req.FindValue))
	}

	for {
		params := dbModels.QueryParams{
			Select: selectColumns,
			Limit:  &limit,
			Offset: &offset,
		}
		rows, err := s.repo.TableService.GetTableData(tableName, params)
		if err != nil {
			return dto.FindReplaceResponse{}, app_errors.LogDatabaseError(err, "failed to fetch rows for find and replace")
		}

		updates, result := s.buildFindReplaceUpdates(rows, selectedColumns, req.FindValue, req.ReplaceValue, req.MatchType, ignoreRe)

		if len(updates) > 0 {
			if err := s.columnsService.BulkUpdateByColumns(ctx, schemaName, model.Alias, updates); err != nil {
				return dto.FindReplaceResponse{}, err
			}
		}

		totalResult.TotalScanned += result.TotalScanned
		totalResult.TotalMatched += result.TotalMatched
		totalResult.TotalUpdated += result.TotalUpdated
		totalResult.TotalSkipped += result.TotalSkipped
		totalResult.TotalRows += result.TotalRows
		totalResult.TotalRowsUpdated += result.TotalRowsUpdated
		totalResult.TotalRowsSkipped += result.TotalRowsSkipped

		lg.Info().
			Str("model_id", req.ModelID).
			Int("batch_offset", offset).
			Int("batch_rows", len(rows)).
			Int("batch_updates", len(updates)).
			Msg("FindReplace batch processed")

		if len(rows) < limit {
			break
		}
		offset += limit
	}

	lg.Info().
		Str("model_id", req.ModelID).
		Int("columns_selected", len(selectedColumns)).
		Int("total_scanned", totalResult.TotalScanned).
		Int("total_matched", totalResult.TotalMatched).
		Int("total_updated", totalResult.TotalUpdated).
		Int("total_skipped", totalResult.TotalSkipped).
		Int("total_rows", totalResult.TotalRows).
		Int("total_rows_updated", totalResult.TotalRowsUpdated).
		Int("total_rows_skipped", totalResult.TotalRowsSkipped).
		Str("match_type", req.MatchType).
		Str("find_value", req.FindValue).
		Msg("Find & Replace action completed")

	return totalResult, nil
}

func (s tableManagementService) buildFindReplaceUpdates(rows []map[string]interface{}, selectedColumns []string, findValue, replaceValue, matchType string, ignoreRe *regexp.Regexp) ([]dto.UpdateColumnValueRequest, dto.FindReplaceResponse) {
	result := dto.FindReplaceResponse{
		TotalScanned: len(rows) * len(selectedColumns),
		TotalRows:    len(rows),
	}

	if len(rows) == 0 {
		return nil, result
	}

	updates := make([]dto.UpdateColumnValueRequest, 0)

	for _, row := range rows {
		rowID, hasRowID := row["id"]
		if !hasRowID {
			result.TotalSkipped += len(selectedColumns)
			result.TotalRowsSkipped++
			continue
		}

		rowUpdates, skipped, rowUpdated, matchedCount, updatedCount := s.buildFindReplaceUpdatesForRow(rowID, row, selectedColumns, findValue, replaceValue, matchType, ignoreRe)

		if skipped > 0 {
			result.TotalSkipped += skipped
		}
		if matchedCount > 0 {
			result.TotalMatched += matchedCount
		}
		if len(rowUpdates) > 0 {
			updates = append(updates, rowUpdates...)
			result.TotalUpdated += updatedCount
		}
		if rowUpdated {
			result.TotalRowsUpdated++
		} else {
			result.TotalRowsSkipped++
		}
	}

	return updates, result
}

// computeFindReplace determines whether a string value matches the find criteria
// according to matchType and returns whether it matched and the replacement
// value (which may be equal to the original when no effective change occurs).
func (s tableManagementService) computeFindReplace(strValue, findValue, replaceValue, matchType string, ignoreRe *regexp.Regexp) (bool, string) {
	switch matchType {
	case "match_case":
		if strings.Contains(strValue, findValue) {
			return true, strings.ReplaceAll(strValue, findValue, replaceValue)
		}
	case "ignore_case":
		if ignoreRe != nil && ignoreRe.MatchString(strValue) {
			return true, ignoreRe.ReplaceAllString(strValue, replaceValue)
		}
	case "match_entire_value":
		if strValue == findValue {
			return true, replaceValue
		}
	default:
		if strings.Contains(strValue, findValue) {
			return true, strings.ReplaceAll(strValue, findValue, replaceValue)
		}
	}
	return false, ""
}

// buildFindReplaceUpdatesForRow processes a single row and returns the required updates,
// the number of skipped cells for that row, whether any cell in the row was
// updated, how many cells matched the find criteria, and how many were updated.
func (s tableManagementService) buildFindReplaceUpdatesForRow(rowID interface{}, row map[string]interface{}, selectedColumns []string, findValue, replaceValue, matchType string, ignoreRe *regexp.Regexp) ([]dto.UpdateColumnValueRequest, int, bool, int, int) {
	updates := make([]dto.UpdateColumnValueRequest, 0)
	skipped := 0
	rowUpdated := false
	matched := 0
	updated := 0

	for _, columnName := range selectedColumns {
		value, exists := row[columnName]
		if !exists || value == nil {
			skipped++
			continue
		}

		strValue, ok := value.(string)
		if !ok {
			skipped++
			continue
		}

		// Delegate match and replacement computation to helper to reduce
		// cognitive complexity in this function.
		isMatch, newVal := s.computeFindReplace(strValue, findValue, replaceValue, matchType, ignoreRe)

		if !isMatch {
			skipped++
			continue
		}

		matched++

		if newVal == strValue {
			// matched but no effective change
			continue
		}

		updates = append(updates, dto.UpdateColumnValueRequest{
			Id:     rowID,
			Column: columnName,
			Value:  newVal,
		})
		updated++
		rowUpdated = true
	}

	return updates, skipped, rowUpdated, matched, updated
}

func removeCharSetForType(removeType string) map[rune]struct{} {
	switch removeType {
	case "symbols":
		return symbolCharSet
	case "currency_symbols":
		return currencyCharSet
	case "brackets":
		return bracketCharSet
	case "punctuation":
		return punctuationCharSet
	default:
		return nil
	}
}

func computeRemoveSpecialCharacters(
	strValue, removeType string,
	customChars []string,
) (bool, string) {

	if removeType == "custom" {
		newVal := strValue
		matched := false

		for _, ch := range customChars {
			if strings.Contains(newVal, ch) {
				matched = true
				newVal = strings.ReplaceAll(newVal, ch, "")
			}
		}

		if !matched {
			return false, ""
		}

		return true, newVal
	}

	removeSet := removeCharSetForType(removeType)
	if removeSet == nil {
		return false, ""
	}

	var b strings.Builder
	b.Grow(len(strValue))
	matched := false

	for _, r := range strValue {
		if _, ok := removeSet[r]; ok {
			matched = true
			continue
		}
		b.WriteRune(r)
	}

	if !matched {
		return false, ""
	}

	return true, b.String()
}

func stripFormattingByType(value string, formatting string, customPatterns []string) (bool, string, bool) {
	switch formatting {
	case "currency":
		return removeCurrencyFormatting(value)
	case "percentage":
		return removePercentageFormatting(value)
	case "separator":
		return removeSeparatorFormatting(value)
	case "phone":
		return removePhoneFormatting(value)
	case "date":
		return normalizeDateFormatting(value)
	case "custom":
		return removeCustomFormatting(value, customPatterns)
	default:
		return false, "", false
	}
}

func removeCurrencyFormatting(value string) (bool, string, bool) {
	replaced := currencySymbolRegex.ReplaceAllString(value, "")
	replaced = numericSeparatorRegex.ReplaceAllString(replaced, "")
	return true, replaced, false
}

func removePercentageFormatting(value string) (bool, string, bool) {
	replaced := strings.ReplaceAll(value, "%", "")
	return replaced != value, replaced, false
}

func removeSeparatorFormatting(value string) (bool, string, bool) {
	replaced := numericSeparatorRegex.ReplaceAllString(value, "")
	return replaced != value, replaced, false
}

func removePhoneFormatting(value string) (bool, string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false, "", false
	}

	if _, ok := parseFlexibleDate(trimmed); ok {
		return false, "", false
	}

	var b strings.Builder
	b.Grow(len(trimmed))
	changed := false
	plusAllowed := true

	for _, r := range trimmed {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			plusAllowed = false
		case r == '+' && plusAllowed:
			b.WriteRune(r)
			plusAllowed = false
		default:
			changed = true
		}
	}

	replaced := b.String()
	if replaced == trimmed {
		return false, "", false
	}

	return changed || replaced != value, replaced, false
}

func removeCustomFormatting(value string, customPatterns []string) (bool, string, bool) {
	if len(customPatterns) == 0 {
		return false, "", false
	}

	replaced := value
	for _, p := range customPatterns {
		if p == "" {
			continue
		}
		replaced = strings.ReplaceAll(replaced, p, "")
	}

	return replaced != value, replaced, false
}

func normalizeDateFormatting(value string) (bool, string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false, "", false
	}

	if parsed, ok := parseFlexibleDate(trimmed); ok {
		return true, parsed.Format(dateOutputLayout), false
	}
	return false, "", false
}

func parseFlexibleDate(value string) (time.Time, bool) {
	for _, layout := range flexibleDateLayouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, true
		}
		if parsed, err := time.ParseInLocation(layout, value, time.UTC); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

// processRemoveFormattingCell handles the logic for a single cell when
// removing formatting. It returns a pointer to an UpdateColumnValueRequest
// when an update should be applied, and a boolean indicating whether an
// update was produced.
func (s tableManagementService) processRemoveFormattingCell(
	rowID interface{},
	columnName string,
	value interface{},
	formatting string,
	customPatterns []string,
) (*dto.UpdateColumnValueRequest, bool) {
	formatted := toStringValue(value)

	changed, newValue, failed := stripFormattingByType(formatted, formatting, customPatterns)
	if failed {
		return nil, false
	}
	if !changed {
		return nil, false
	}

	// Date formatting: parse and return YYYY-MM-DD string.
	if strings.EqualFold(strings.TrimSpace(formatting), "date") {
		if parsed, ok := parseFlexibleDate(formatted); ok {
			return &dto.UpdateColumnValueRequest{
				Id:     rowID,
				Column: columnName,
				Value:  parsed.Format(dateOutputLayout),
			}, true
		}
		return nil, false
	}

	// Non-date formatting: infer type (int/float/bool/string)
	updatedValue, ok := inferFormattedCellValue(newValue)
	if !ok {
		return nil, false
	}

	return &dto.UpdateColumnValueRequest{
		Id:     rowID,
		Column: columnName,
		Value:  updatedValue,
	}, true
}

func (s tableManagementService) RemoveFormatting(ctx context.Context, schemaName string, req dto.RemoveFormattingRequest) (dto.RemoveFormattingResponse, error) {
	lg := logger.Get()

	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.RemoveFormattingResponse{}, err
	}

	// Resolve selected columns from the request and only fetch those columns
	columnsData, err := s.GetColumnsByModelID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.RemoveFormattingResponse{}, err
	}

	selectedColumns, err := s.getSelectedColumnsFromRequest(columnsData, req.Columns)
	if err != nil {
		return dto.RemoveFormattingResponse{}, err
	}

	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)
	limit := columnActionBatchSize
	offset := 0
	totalResult := dto.RemoveFormattingResponse{}

	// Build select list (always include id)
	selectColumns := make([]string, 0, len(selectedColumns)+1)
	selectColumns = append(selectColumns, "id")
	selectColumns = append(selectColumns, selectedColumns...)

	for {
		params := dbModels.QueryParams{
			Select: selectColumns,
			Limit:  &limit,
			Offset: &offset,
		}
		rows, err := s.repo.TableService.GetTableData(tableName, params)
		if err != nil {
			return dto.RemoveFormattingResponse{}, app_errors.LogDatabaseError(err, "failed to fetch rows for remove formatting")
		}

		updates, result := s.buildRemoveFormattingUpdates(rows, req.Formatting, req.CustomPattern, selectedColumns)

		if len(updates) > 0 {
			if err := s.applyRemoveFormattingUpdates(ctx, fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias), updates); err != nil {
				return dto.RemoveFormattingResponse{}, err
			}
		}

		totalResult.ScannedRecords += result.ScannedRecords
		totalResult.UpdatedRecords += result.UpdatedRecords
		totalResult.SkippedRecords += result.SkippedRecords
		totalResult.FailedRecords += result.FailedRecords

		lg.Info().
			Str("model_id", req.ModelID).
			Int("batch_offset", offset).
			Int("batch_rows", len(rows)).
			Int("batch_updates", len(updates)).
			Str("formatting", req.Formatting).
			Msg("RemoveFormatting batch processed")

		if len(rows) < limit {
			break
		}
		offset += limit
	}

	lg.Info().
		Str("model_id", req.ModelID).
		Int("columns_selected", 0).
		Int("scanned_records", totalResult.ScannedRecords).
		Int("updated_records", totalResult.UpdatedRecords).
		Int("skipped_records", totalResult.SkippedRecords).
		Int("failed_records", totalResult.FailedRecords).
		Str("formatting", req.Formatting).
		Msg("Remove formatting action completed")

	return totalResult, nil
}

func (s tableManagementService) buildRemoveFormattingUpdates(rows []map[string]interface{}, formatting string, customPatterns []string, selectedColumns []string) ([]dto.UpdateColumnValueRequest, dto.RemoveFormattingResponse) {
	result := dto.RemoveFormattingResponse{ScannedRecords: len(rows) * len(selectedColumns)}
	if len(rows) == 0 {
		return nil, result
	}

	updates := make([]dto.UpdateColumnValueRequest, 0)
	for _, row := range rows {
		rowUpdates, rowStatus := s.buildRemoveFormattingUpdatesForRow(row, formatting, customPatterns, selectedColumns)
		if len(rowUpdates) > 0 {
			updates = append(updates, rowUpdates...)
		}

		switch {
		case rowStatus == removeFormattingRowUpdated:
			result.UpdatedRecords++
		case rowStatus == removeFormattingRowSkipped:
			result.SkippedRecords++
		}
	}

	return updates, result
}

type removeFormattingRowStatus int

const (
	removeFormattingRowSkipped removeFormattingRowStatus = iota
	removeFormattingRowUpdated
)

func (s tableManagementService) buildRemoveFormattingUpdatesForRow(
	row map[string]interface{},
	formatting string, customPatterns []string,
	selectedColumns []string,
) ([]dto.UpdateColumnValueRequest, removeFormattingRowStatus) {
	rowID, hasRowID := row["id"]
	if !hasRowID {
		return nil, removeFormattingRowSkipped
	}

	logger.Get().
		Info().
		Interface("row_id", rowID).
		Str("formatting", formatting).
		Int("selected_columns", len(selectedColumns)).
		Msg("Processing row for remove formatting")

	updates := make([]dto.UpdateColumnValueRequest, 0)
	rowUpdated := false

	for _, columnName := range selectedColumns {
		if strings.EqualFold(strings.TrimSpace(columnName), "id") {
			continue
		}
		value, exists := lookupRowValue(row, columnName)
		if !exists || value == nil {
			continue
		}

		if upd, ok := s.processRemoveFormattingCell(rowID, columnName, value, formatting, customPatterns); ok {
			updates = append(updates, *upd)
			rowUpdated = true
		}
	}

	if rowUpdated {
		return updates, removeFormattingRowUpdated
	}
	return updates, removeFormattingRowSkipped
}

func toStringValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case time.Time:
		return v.Format(time.RFC3339Nano)
	default:
		return fmt.Sprint(v)
	}
}

func inferFormattedCellValue(value string) (interface{}, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return value, true
	}

	if parsed, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return parsed, true
	}

	if parsed, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return parsed, true
	}

	if parsed, err := strconv.ParseBool(trimmed); err == nil {
		return parsed, true
	}

	return trimmed, true
}

func (s tableManagementService) applyRemoveFormattingUpdates(ctx context.Context, tableName string, updates []dto.UpdateColumnValueRequest) error {
	for _, update := range updates {
		rowID := update.Id
		updateData := map[string]interface{}{
			update.Column: update.Value,
		}
		logger.Get().
			Info().
			Interface("row_id", rowID).
			Str("table", tableName).
			Str("column", update.Column).
			Interface("value", update.Value).
			Msg("Applying remove formatting update")
		if _, err := s.repo.TableService.UpdateRecord(tableName, rowID, updateData); err != nil {
			logger.Get().
				Error().
				Interface("row_id", rowID).
				Str("table", tableName).
				Str("column", update.Column).
				Interface("value", update.Value).
				Err(err).
				Msg("Remove formatting update failed")
			return app_errors.LogDatabaseError(err, "failed to apply remove formatting updates")
		}
	}

	return nil
}

func lookupRowValue(row map[string]interface{}, columnName string) (interface{}, bool) {
	if value, exists := row[columnName]; exists {
		return value, true
	}

	normalizedTarget := strings.ToLower(strings.TrimSpace(columnName))
	for key, value := range row {
		if strings.ToLower(strings.TrimSpace(key)) == normalizedTarget {
			return value, true
		}
	}

	return nil, false
}

// RemoveSpecialCharacters fetches target rows from the database, removes special
// characters from selected columns, and persists only changed cell values via
// bulk_update_by_columns. Rows are processed in batches to limit memory usage.
func (s tableManagementService) RemoveSpecialCharacters(ctx context.Context, schemaName string, req dto.RemoveSpecialCharactersRequest) (dto.RemoveSpecialCharactersResponse, error) {
	lg := logger.Get()

	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.RemoveSpecialCharactersResponse{}, err
	}
	fmt.Println("model alias: ", model.Alias)

	columnsData, err := s.GetColumnsByModelID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.RemoveSpecialCharactersResponse{}, err
	}

	selectedColumns, err := s.getSelectedColumnsFromRequest(columnsData, req.Columns)
	if err != nil {
		return dto.RemoveSpecialCharactersResponse{}, err
	}

	selectColumns := make([]string, 0, len(selectedColumns)+1)
	selectColumns = append(selectColumns, "id")
	selectColumns = append(selectColumns, selectedColumns...)

	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)

	limit := columnActionBatchSize
	offset := 0
	totalResult := dto.RemoveSpecialCharactersResponse{}

	for {
		params := dbModels.QueryParams{
			Select: selectColumns,
			Limit:  &limit,
			Offset: &offset,
		}
		rows, err := s.repo.TableService.GetTableData(tableName, params)
		if err != nil {
			return dto.RemoveSpecialCharactersResponse{}, app_errors.LogDatabaseError(err, "failed to fetch rows for remove special characters")
		}

		updates, result := s.buildRemoveSpecialCharactersUpdates(rows, selectedColumns, req.SpecialCharactersType, req.CustomCharacter)

		if len(updates) > 0 {
			if err := s.columnsService.BulkUpdateByColumns(ctx, schemaName, model.Alias, updates); err != nil {
				return dto.RemoveSpecialCharactersResponse{}, err
			}
		}

		totalResult.TotalScanned += result.TotalScanned
		totalResult.TotalMatched += result.TotalMatched
		totalResult.TotalUpdated += result.TotalUpdated
		totalResult.TotalSkipped += result.TotalSkipped
		totalResult.TotalRows += result.TotalRows
		totalResult.TotalRowsUpdated += result.TotalRowsUpdated
		totalResult.TotalRowsSkipped += result.TotalRowsSkipped

		lg.Info().
			Str("model_id", req.ModelID).
			Int("batch_offset", offset).
			Int("batch_rows", len(rows)).
			Int("batch_updates", len(updates)).
			Msg("RemoveSpecialCharacters batch processed")

		if len(rows) < limit {
			break
		}
		offset += limit
	}

	lg.Info().
		Str("model_id", req.ModelID).
		Int("columns_selected", len(selectedColumns)).
		Int("total_scanned", totalResult.TotalScanned).
		Int("total_matched", totalResult.TotalMatched).
		Int("total_updated", totalResult.TotalUpdated).
		Int("total_skipped", totalResult.TotalSkipped).
		Int("total_rows", totalResult.TotalRows).
		Int("total_rows_updated", totalResult.TotalRowsUpdated).
		Int("total_rows_skipped", totalResult.TotalRowsSkipped).
		Str("special_characters_type", req.SpecialCharactersType).
		Msg("Remove special characters action completed")

	return totalResult, nil
}

func (s tableManagementService) RemoveDuplicates(ctx context.Context, schemaName string, req dto.RemoveDuplicatesRequest) (dto.RemoveDuplicatesResponse, error) {
	lg := logger.Get()

	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.RemoveDuplicatesResponse{}, err
	}

	columnsData, err := s.GetColumnsByModelID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.RemoveDuplicatesResponse{}, err
	}

	selectedColumns, err := s.getSelectedColumnsFromRequest(columnsData, req.Columns)
	if err != nil {
		return dto.RemoveDuplicatesResponse{}, err
	}

	if req.KeepRule == "keep_latest_updated" {
		supported, err := s.hasUpdateTimestampColumn(ctx, schemaName, model.Alias)
		if err != nil {
			return dto.RemoveDuplicatesResponse{}, err
		}
		if !supported {
			return dto.RemoveDuplicatesResponse{}, fmt.Errorf("%w: keep_latest_updated requires last_modified_time column on table %s", app_errors.InvalidPayload, model.Alias)
		}
	}

	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)
	affectedRows, totalDuplicateRows, err := s.executeRemoveDuplicates(ctx, tableName, req.Duplicate, req.KeepRule, selectedColumns)
	if err != nil {
		return dto.RemoveDuplicatesResponse{}, err
	}

	lg.Info().
		Str("model_id", req.ModelID).
		Int("columns_selected", len(selectedColumns)).
		Int64("total_rows_affected", affectedRows).
		Int64("total_duplicate_rows", totalDuplicateRows).
		Str("duplicate", req.Duplicate).
		Str("keep_rule", req.KeepRule).
		Msg("Remove duplicates action completed")

	return dto.RemoveDuplicatesResponse{
		TotalRowsAffected:  affectedRows,
		TotalDuplicateRows: totalDuplicateRows,
	}, nil
}

func (s tableManagementService) ColumnSplit(ctx context.Context, schemaName string, req dto.ColumnSplitRequest) (dto.ColumnSplitResponse, error) {
	lg := logger.Get()

	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID.String())
	if err != nil {
		return dto.ColumnSplitResponse{}, err
	}

	columnsData, err := s.GetColumnsByModelID(ctx, schemaName, req.ModelID.String())
	if err != nil {
		return dto.ColumnSplitResponse{}, err
	}

	selectedColumn, selectedIndex, err := s.getColumnByIDFromList(columnsData, req.ColumnID.String())
	if err != nil {
		return dto.ColumnSplitResponse{}, err
	}

	selectedOrder := 0.0
	if selectedColumn.OrderIndex != nil {
		selectedOrder = *selectedColumn.OrderIndex
	}

	strategy, err := s.parseColumnSplitStrategy(req.SplitBy)
	if err != nil {
		return dto.ColumnSplitResponse{}, err
	}

	// Validate fixedLength bounds directly in database (ignoring NULL/empty strings)
	if strategy.kind == "fixedLength" {
		query := fmt.Sprintf(
			`SELECT EXISTS (SELECT 1 FROM %s WHERE %s IS NOT NULL AND %s <> '' AND char_length(%s) < $1)`,
			fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias),
			fmt.Sprintf(QuotedColumnFormat, selectedColumn.ColumnName),
			fmt.Sprintf(QuotedColumnFormat, selectedColumn.ColumnName),
			fmt.Sprintf(QuotedColumnFormat, selectedColumn.ColumnName),
		)
		var exceeds bool
		rows, queryErr := s.repo.DB.QueryContext(ctx, query, strategy.value)
		if queryErr != nil {
			return dto.ColumnSplitResponse{}, app_errors.LogDatabaseError(queryErr, "failed to validate fixed length bounds")
		}
		defer rows.Close()
		if rows.Next() {
			if scanErr := rows.Scan(&exceeds); scanErr != nil {
				return dto.ColumnSplitResponse{}, app_errors.LogDatabaseError(scanErr, "failed to scan fixed length bounds")
			}
		}
		if exceeds {
			return dto.ColumnSplitResponse{}, fmt.Errorf("%w: fixedLength value cannot exceed string length", app_errors.InvalidPayload)
		}
	}

	// Calculate maxParts using database-side array split
	arrayExpr, params := s.getSplitSQLArrayExpr(selectedColumn.ColumnName, strategy)
	maxPartsQuery := fmt.Sprintf(
		`SELECT COALESCE(MAX(array_length(%s, 1)), 0) FROM %s`,
		arrayExpr,
		fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias),
	)
	var maxParts int
	rows, queryErr := s.repo.DB.QueryContext(ctx, maxPartsQuery, params...)
	if queryErr != nil {
		return dto.ColumnSplitResponse{}, app_errors.LogDatabaseError(queryErr, "failed to calculate maximum split parts")
	}
	defer rows.Close()
	if rows.Next() {
		if scanErr := rows.Scan(&maxParts); scanErr != nil {
			return dto.ColumnSplitResponse{}, app_errors.LogDatabaseError(scanErr, "failed to scan maximum split parts")
		}
	}

	if maxParts <= 0 {
		return dto.ColumnSplitResponse{}, fmt.Errorf("%w: no values available to split in column %s", app_errors.InvalidPayload, selectedColumn.ColumnName)
	}

	fmt.Println("maxParts: ", maxParts)

	if err = ensureSplitIsPossible(maxParts, selectedColumn.ColumnName); err != nil {
		return dto.ColumnSplitResponse{}, err
	}

	columnCount, err := resolveSplitColumnCount(maxParts, req.Limit)
	if err != nil {
		return dto.ColumnSplitResponse{}, err
	}

	orderIndexes, err := s.computeSplitOrderIndexes(columnsData, selectedIndex, req.Where, columnCount)
	if err != nil {
		return dto.ColumnSplitResponse{}, err
	}

	// Shift columns if inserting next to the selected column to avoid conflicts
	if req.Where == "next" {
		// collect affected columns and sort descending to avoid update collisions
		affected := make([]dto.ColumnResponse, 0)
		for _, col := range columnsData {
			if col.OrderIndex != nil && *col.OrderIndex > selectedOrder {
				affected = append(affected, col)
			}
		}
		sort.SliceStable(affected, func(i, j int) bool {
			return *affected[i].OrderIndex > *affected[j].OrderIndex
		})

		for _, col := range affected {
			newOrderIndex := *col.OrderIndex + float64(columnCount)
			orderIndexUpdate := dto.ColumnUpdate{
				OrderIndex: &newOrderIndex,
			}
			_, err = s.UpdateColumn(ctx, schemaName, col.ID.String(), orderIndexUpdate)
			if err != nil {
				return dto.ColumnSplitResponse{}, err
			}
		}
	}

	newColumnNames := make([]string, 0, columnCount)
	createdColumns := make([]tenant.Column, 0, columnCount)

	// Generate deterministic physical column names based on selected column name
	generatedNames, err := s.buildSplitColumnNames(columnsData, selectedColumn.ColumnName, columnCount)
	if err != nil {
		return dto.ColumnSplitResponse{}, err
	}

	now := time.Now().UTC()

	for idx, orderIndex := range orderIndexes {
		title := s.buildSplitColumnTitle(selectedColumn.Title, selectedColumn.ColumnName, idx+1)

		// Create column metadata with explicit column_name to ensure deterministic naming
		colInsert := dto.ColumnInsertion{
			ID:          uuid.New(),
			ModelID:     model.ID,
			BaseID:      model.BaseID,
			Title:       title,
			ColumnName:  generatedNames[idx],
			Description: &selectedColumn.Description,
			Meta:        map[string]interface{}{},
			UIDT:        "longText",
			DT:          helpers.StringPtr("TEXT"),
			Virtual:     false,
			System:      false,
			Deleted:     false,
			OrderIndex:  &orderIndex,
			CreatedBy:   "",
			UpdatedBy:   "",
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		createdCol, createErr := s.columnsService.Create(ctx, colInsert, schemaName)
		if createErr != nil {
			// attempt cleanup of any previously created columns
			for _, c := range createdColumns {
				var dtoCol dto.ColumnResponse
				_ = helpers.StructToStruct(c, &dtoCol)
				_ = s.DeleteColumnAndCleanUp(ctx, schemaName, c.ID.String(), dtoCol)
			}
			return dto.ColumnSplitResponse{}, createErr
		}

		// add physical column in table
		if addErr := s.addColumnInTableDb(schemaName, model.Alias, createdCol); addErr != nil {
			// attempt cleanup of this and previous columns
			var dtoCol dto.ColumnResponse
			_ = helpers.StructToStruct(createdCol, &dtoCol)
			_ = s.DeleteColumnAndCleanUp(ctx, schemaName, createdCol.ID.String(), dtoCol)
			for _, c := range createdColumns {
				var dc dto.ColumnResponse
				_ = helpers.StructToStruct(c, &dc)
				_ = s.DeleteColumnAndCleanUp(ctx, schemaName, c.ID.String(), dc)
			}
			return dto.ColumnSplitResponse{}, addErr
		}

		createdColumns = append(createdColumns, createdCol)
		newColumnNames = append(newColumnNames, createdCol.ColumnName)
	}

	if err = s.performBulkSplitUpdate(ctx, schemaName, model.Alias, selectedColumn.ColumnName, newColumnNames, strategy, columnCount); err != nil {
		// cleanup created columns on failure
		for _, c := range createdColumns {
			var dtoCol dto.ColumnResponse
			_ = helpers.StructToStruct(c, &dtoCol)
			_ = s.DeleteColumnAndCleanUp(ctx, schemaName, c.ID.String(), dtoCol)
		}
		return dto.ColumnSplitResponse{}, err
	}

	if !req.KeepOriginal {
		// Use a transaction only for dropping the column to avoid long-lived exclusive table locks during split updates
		tx, startErr := s.repo.DB.Begin()
		if startErr != nil {
			return dto.ColumnSplitResponse{}, app_errors.LogDatabaseError(startErr, "failed to start transaction for column drop")
		}
		defer func() {
			if err != nil {
				_ = tx.Rollback()
			}
		}()

		if err = s.deleteSplitOriginalColumn(tx, schemaName, model.Alias, selectedColumn); err != nil {
			return dto.ColumnSplitResponse{}, err
		}

		if err = tx.Commit(); err != nil {
			return dto.ColumnSplitResponse{}, app_errors.LogDatabaseError(err, "failed to commit column split transaction")
		}
	}

	lg.Info().
		Str("model_id", req.ModelID.String()).
		Str("column_id", req.ColumnID.String()).
		Str("column_name", selectedColumn.ColumnName).
		Str("split_type", strategy.kind).
		Int("created_columns", len(newColumnNames)).
		Bool("keep_original", req.KeepOriginal).
		Str("where", req.Where).
		Msg("Column split completed")

	return dto.ColumnSplitResponse{
		Message:        "Column split completed successfully",
		CreatedColumns: newColumnNames,
	}, nil
}

func (s tableManagementService) executeRemoveDuplicates(
	txCtx context.Context,
	tableName string,
	Duplicate string,
	keepRule string,
	selectedColumns []string,
) (int64, int64, error) {
	matchCase := s.determineMatchCase(Duplicate)

	partitionBy, notAllSelectedColsEmpty := s.buildDuplicateKeyExpressions(selectedColumns, matchCase)
	orderBy := s.buildDuplicateKeepOrderBy(keepRule)

	totalDuplicateRows, err := s.countDuplicateRows(txCtx, tableName, partitionBy, notAllSelectedColsEmpty)
	if err != nil {
		return 0, 0, err
	}

	switch Duplicate {
	case "remove_row":
		q := s.buildDeleteDuplicatesQuery(tableName, partitionBy, orderBy, notAllSelectedColsEmpty)
		affected, err := s.execQueryAndRowsAffected(txCtx, q, "failed to remove duplicate rows")
		if err != nil {
			return 0, 0, err
		}
		return affected, totalDuplicateRows, nil

	case "remove_duplicates", "remove_duplicates_matchCase":
		setClauses := make([]string, 0, len(selectedColumns))
		for _, columnName := range selectedColumns {
			setClauses = append(setClauses, fmt.Sprintf("%s = NULL", fmt.Sprintf(QuotedColumnFormat, columnName)))
		}

		q := s.buildUpdateDuplicatesQuery(tableName, partitionBy, orderBy, notAllSelectedColsEmpty, strings.Join(setClauses, ", "))
		affected, err := s.execQueryAndRowsAffected(txCtx, q, "failed to clear duplicate values")
		if err != nil {
			return 0, 0, err
		}
		return affected, totalDuplicateRows, nil

	default:
		return 0, 0, fmt.Errorf("%w: unsupported duplicate handling mode %s", app_errors.InvalidPayload, Duplicate)
	}
}

// Helper to determine whether values should be matched case-sensitively
func (s tableManagementService) determineMatchCase(mode string) bool {
	switch mode {
	case "remove_duplicates_matchCase":
		return true
	case "remove_row", "remove_duplicates":
		return false
	default:
		return true
	}
}

// Helper to count total rows that belong to duplicate groups
func (s tableManagementService) countDuplicateRows(ctx context.Context, tableName, partitionBy, condition string) (int64, error) {
	query := fmt.Sprintf(
		`SELECT COALESCE(SUM(cnt), 0) FROM (SELECT COUNT(*) AS cnt FROM %s WHERE %s GROUP BY %s HAVING COUNT(*) > 1) t;`,
		tableName,
		condition,
		partitionBy,
	)

	var total int64
	rows, err := s.repo.DB.QueryContext(ctx, query)
	if err != nil {
		return 0, app_errors.LogDatabaseError(err, "failed to count duplicate rows")
	}
	defer rows.Close()
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return 0, app_errors.LogDatabaseError(err, "failed to read duplicate rows count")
		}
	}
	return total, nil
}

// Helper to execute a query and return rows affected with consistent error wrapping
func (s tableManagementService) execQueryAndRowsAffected(ctx context.Context, query, errMsg string) (int64, error) {
	result, err := s.repo.DB.ExecContext(ctx, query)
	if err != nil {
		return 0, app_errors.LogDatabaseError(err, errMsg)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, app_errors.LogDatabaseError(err, "failed to get rows affected")
	}
	return affected, nil
}

func (s tableManagementService) buildDeleteDuplicatesQuery(tableName, partitionBy, orderBy, condition string) string {
	return fmt.Sprintf(
		`WITH duplicates AS (
			SELECT id,
				   ROW_NUMBER() OVER (PARTITION BY %s ORDER BY %s) AS row_number
			FROM %s
			WHERE %s
		)
		DELETE FROM %s
		WHERE id IN (
			SELECT id FROM duplicates WHERE row_number > 1
		);`,
		partitionBy,
		orderBy,
		tableName,
		condition,
		tableName,
	)
}

func (s tableManagementService) buildUpdateDuplicatesQuery(tableName, partitionBy, orderBy, condition, setClause string) string {
	return fmt.Sprintf(
		`WITH duplicates AS (
			SELECT id,
				   ROW_NUMBER() OVER (PARTITION BY %s ORDER BY %s) AS row_number
			FROM %s
			WHERE %s
		)
		UPDATE %s target
		SET %s
		FROM duplicates d
		WHERE target.id = d.id
		  AND d.row_number > 1;`,
		partitionBy,
		orderBy,
		tableName,
		condition,
		tableName,
		setClause,
	)
}

func (s tableManagementService) buildDuplicateKeyExpressions(selectedColumns []string, matchCase bool) (string, string) {
	expressions := make([]string, 0, len(selectedColumns))
	nullChecks := make([]string, 0, len(selectedColumns))

	for _, columnName := range selectedColumns {
		baseExpr := fmt.Sprintf("NULLIF(TRIM(CAST(%s AS TEXT)), '')", fmt.Sprintf(QuotedColumnFormat, columnName))
		expr := baseExpr
		if !matchCase {
			expr = fmt.Sprintf("LOWER(%s)", baseExpr)
		}
		expressions = append(expressions, expr)
		nullChecks = append(nullChecks, fmt.Sprintf("%s IS NULL", expr))
	}

	return strings.Join(expressions, ", "), fmt.Sprintf("NOT (%s)", strings.Join(nullChecks, " AND "))
}

func (s tableManagementService) buildDuplicateKeepOrderBy(keepRule string) string {
	switch keepRule {
	case "keep_last":
		return fmt.Sprintf("%s DESC", fmt.Sprintf(QuotedColumnFormat, "id"))
	case "keep_latest_updated":
		return fmt.Sprintf("%s DESC, %s DESC", fmt.Sprintf(QuotedColumnFormat, "last_modified_time"), fmt.Sprintf(QuotedColumnFormat, "id"))
	default:
		return fmt.Sprintf("%s ASC", fmt.Sprintf(QuotedColumnFormat, "id"))
	}
}

func (s tableManagementService) hasUpdateTimestampColumn(ctx context.Context, schemaName, tableName string) (bool, error) {
	filters := []dbModels.QueryFilter{
		{Column: "table_schema", Operator: "=", Value: schemaName},
		{Column: "table_name", Operator: "=", Value: tableName},
	}
	params := dbModels.QueryParams{Select: []string{"column_name"}, Filters: filters}

	rows, err := s.repo.TableService.GetTableData("information_schema.columns", params)
	if err != nil {
		return false, app_errors.LogDatabaseError(err, "failed to inspect table columns")
	}

	for _, row := range rows {
		if colName, ok := row["column_name"].(string); ok && colName == "last_modified_time" {
			return true, nil
		}
	}

	return false, nil
}

func (s tableManagementService) buildRemoveSpecialCharactersUpdates(
	rows []map[string]interface{},
	selectedColumns []string,
	removeType string, customChars []string,
) ([]dto.UpdateColumnValueRequest, dto.RemoveSpecialCharactersResponse) {
	result := dto.RemoveSpecialCharactersResponse{
		TotalScanned: len(rows) * len(selectedColumns),
		TotalRows:    len(rows),
	}

	if len(rows) == 0 {
		return nil, result
	}

	updates := make([]dto.UpdateColumnValueRequest, 0)

	for _, row := range rows {
		rowID, hasRowID := row["id"]
		if !hasRowID {
			result.TotalSkipped += len(selectedColumns)
			result.TotalRowsSkipped++
			continue
		}

		rowUpdates, skipped, rowUpdated, matchedCount, updatedCount := s.buildRemoveSpecialCharactersUpdatesForRow(
			rowID, row, selectedColumns, removeType, customChars,
		)

		if skipped > 0 {
			result.TotalSkipped += skipped
		}
		if matchedCount > 0 {
			result.TotalMatched += matchedCount
		}
		if len(rowUpdates) > 0 {
			updates = append(updates, rowUpdates...)
			result.TotalUpdated += updatedCount
		}
		if rowUpdated {
			result.TotalRowsUpdated++
		} else {
			result.TotalRowsSkipped++
		}
	}

	return updates, result
}

func (s tableManagementService) buildRemoveSpecialCharactersUpdatesForRow(
	rowID interface{},
	row map[string]interface{},
	selectedColumns []string,
	removeType string, customChars []string,
) ([]dto.UpdateColumnValueRequest, int, bool, int, int) {
	updates := make([]dto.UpdateColumnValueRequest, 0)
	skipped := 0
	rowUpdated := false
	matched := 0
	updated := 0

	for _, columnName := range selectedColumns {
		value, exists := row[columnName]
		if !exists || value == nil {
			skipped++
			continue
		}

		strValue, ok := value.(string)
		if !ok {
			skipped++
			continue
		}

		isMatch, newVal := computeRemoveSpecialCharacters(strValue, removeType, customChars)
		if !isMatch {
			skipped++
			continue
		}

		matched++

		if newVal == strValue {
			continue
		}

		updates = append(updates, dto.UpdateColumnValueRequest{
			Id:     rowID,
			Column: columnName,
			Value:  newVal,
		})
		updated++
		rowUpdated = true
	}

	return updates, skipped, rowUpdated, matched, updated
}

func (s tableManagementService) parseColumnSplitStrategy(splitBy dto.SplitByRequest) (columnSplitStrategy, error) {
	kind := strings.TrimSpace(strings.ToLower(splitBy.Type))
	switch kind {
	case "separator":
		separator, _ := splitBy.Config["separator"].(string)
		if separator == "" {
			return columnSplitStrategy{}, fmt.Errorf("%w: separator cannot be empty", app_errors.InvalidPayload)
		}
		return columnSplitStrategy{
			kind:      kind,
			separator: separator,
		}, nil
	case "fixedlength":
		action, _ := splitBy.Config["action"].(string)
		action = strings.TrimSpace(strings.ToLower(action))
		if action != "after" && action != "before" {
			return columnSplitStrategy{}, fmt.Errorf("%w: fixedLength action must be after or before", app_errors.InvalidPayload)
		}

		value, ok := splitBy.Config["value"]
		if !ok {
			return columnSplitStrategy{}, fmt.Errorf("%w: fixedLength value is required", app_errors.InvalidPayload)
		}

		valueInt, err := parsePositiveSplitInt(value)
		if err != nil {
			return columnSplitStrategy{}, err
		}

		return columnSplitStrategy{
			kind:   "fixedLength",
			action: action,
			value:  valueInt,
		}, nil
	case "pattern":
		pattern, _ := splitBy.Config["pattern"].(string)
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			return columnSplitStrategy{}, fmt.Errorf("%w: pattern cannot be empty", app_errors.InvalidPayload)
		}

		// Allowed pattern whitelist (aligns with spec supported patterns)
		allowed := map[string]struct{}{
			"\\d+":         {},
			"[A-Z]+":       {},
			"[a-z]+":       {},
			"[A-Za-z]+":    {},
			"\\s+":         {},
			"[^a-zA-Z0-9]": {},
			"@(.+)":        {},
			"\\.":          {},
		}
		if _, ok := allowed[pattern]; !ok {
			return columnSplitStrategy{}, fmt.Errorf("%w: unsupported regex pattern", app_errors.InvalidPayload)
		}

		re, err := regexp.Compile(pattern)
		if err != nil {
			return columnSplitStrategy{}, fmt.Errorf("%w: invalid regex pattern: %v", app_errors.InvalidPayload, err)
		}
		return columnSplitStrategy{
			kind:    kind,
			pattern: pattern,
			regex:   re,
		}, nil
	default:
		return columnSplitStrategy{}, fmt.Errorf("%w: unsupported split type %s", app_errors.InvalidPayload, splitBy.Type)
	}
}

func parsePositiveSplitInt(value interface{}) (int, error) {
	switch v := value.(type) {
	case int:
		if v <= 0 {
			return 0, fmt.Errorf("%w: fixedLength value must be greater than zero", app_errors.InvalidPayload)
		}
		return v, nil
	case int32:
		if v <= 0 {
			return 0, fmt.Errorf("%w: fixedLength value must be greater than zero", app_errors.InvalidPayload)
		}
		return int(v), nil
	case int64:
		if v <= 0 {
			return 0, fmt.Errorf("%w: fixedLength value must be greater than zero", app_errors.InvalidPayload)
		}
		return int(v), nil
	case float32:
		if v <= 0 {
			return 0, fmt.Errorf("%w: fixedLength value must be greater than zero", app_errors.InvalidPayload)
		}
		return int(v), nil
	case float64:
		if v <= 0 {
			return 0, fmt.Errorf("%w: fixedLength value must be greater than zero", app_errors.InvalidPayload)
		}
		return int(v), nil
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil || parsed <= 0 {
			return 0, fmt.Errorf("%w: fixedLength value must be greater than zero", app_errors.InvalidPayload)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("%w: fixedLength value is invalid", app_errors.InvalidPayload)
	}
}

func (s tableManagementService) getColumnByIDFromList(columns []dto.ColumnResponse, id string) (dto.ColumnResponse, int, error) {
	for idx, col := range columns {
		if col.ID.String() == id {
			return col, idx, nil
		}
	}
	return dto.ColumnResponse{}, -1, app_errors.ColumnNotFound
}

func (s tableManagementService) buildSplitColumnNames(columns []dto.ColumnResponse, baseName string, count int) ([]string, error) {
	if count <= 0 {
		return nil, fmt.Errorf("%w: split count must be greater than zero", app_errors.InvalidPayload)
	}

	existing := make(map[string]struct{}, len(columns))
	for _, col := range columns {
		existing[col.ColumnName] = struct{}{}
	}

	names := make([]string, 0, count)
	nextSuffix := 1
	for len(names) < count {
		name := fmt.Sprintf("%s_%d", baseName, nextSuffix)
		for {
			if _, ok := existing[name]; !ok {
				break
			}
			nextSuffix++
			name = fmt.Sprintf("%s_%d", baseName, nextSuffix)
		}
		existing[name] = struct{}{}
		names = append(names, name)
		nextSuffix++
	}

	return names, nil
}

func (s tableManagementService) buildSplitColumnTitle(baseTitle, baseName string, index int) string {
	seed := strings.TrimSpace(baseTitle)
	if seed == "" {
		seed = baseName
	}
	return fmt.Sprintf("%s_%d", seed, index)
}

func (s tableManagementService) computeSplitOrderIndexes(
	columns []dto.ColumnResponse,
	selectedIndex int,
	where string,
	count int,
) ([]float64, error) {

	if count <= 0 {
		return nil, fmt.Errorf("%w: split count must be greater than zero", app_errors.InvalidPayload)
	}

	selectedOrder := 0.0
	if columns[selectedIndex].OrderIndex != nil {
		selectedOrder = *columns[selectedIndex].OrderIndex
	}

	orderIndexes := make([]float64, 0, count)

	switch where {
	case "next":
		start := selectedOrder + 1

		for i := 0; i < count; i++ {
			orderIndexes = append(orderIndexes, start+float64(i))
		}

	case "end":
		maxOrder := selectedOrder

		for _, col := range columns {
			if col.OrderIndex != nil && *col.OrderIndex > maxOrder {
				maxOrder = *col.OrderIndex
			}
		}

		start := maxOrder + 1

		for i := 0; i < count; i++ {
			orderIndexes = append(orderIndexes, start+float64(i))
		}

	default:
		return nil, fmt.Errorf(
			"%w: unsupported column placement %s",
			app_errors.InvalidPayload,
			where,
		)
	}

	return orderIndexes, nil
}

func ensureSplitIsPossible(maxParts int, columnName string) error {
	if maxParts <= 1 {
		return app_errors.SplitNotPossible
	}
	return nil
}

func resolveSplitColumnCount(maxParts int, limit *int) (int, error) {
	columnCount := maxParts
	if limit != nil {
		if *limit < maxParts {
			columnCount = *limit
		}
	}
	if columnCount <= 1 {
		return 0, app_errors.SplitNotPossible
	}
	return columnCount, nil
}

func splitJoinSeparator(strategy columnSplitStrategy) string {
	if strategy.kind == "separator" {
		return strategy.separator
	}
	return ""
}

func applySplitColumnLimit(parts []string, columnCount int, joinSeparator string) []string {
	if columnCount <= 0 || len(parts) <= columnCount {
		return parts
	}

	result := make([]string, 0, columnCount)
	result = append(result, parts[:columnCount-1]...)
	result = append(result, strings.Join(parts[columnCount-1:], joinSeparator))
	return result
}

func (s tableManagementService) insertSplitColumnMetadata(
	tx *sql.Tx,
	schemaName string,
	model tenant.Model,
	selectedColumn dto.ColumnResponse,
	columnName string,
	title string,
	orderIndex float64,
	createdBy string,
	now time.Time,
) error {
	query := fmt.Sprintf(
		`INSERT INTO %s (id, model_id, base_id, column_name, title, uidt, dt, description, meta, virtual, system, deleted, order_index, created_by, last_modified_by, created_time, last_modified_time)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10,$11,$12,$13,$14,$15,$16,$17)`,
		fmt.Sprintf(`"%s".columns`, schemaName),
	)

	description := selectedColumn.Description
	if strings.TrimSpace(description) == "" {
		description = ""
	}

	_, err := tx.ExecContext(context.Background(), query,
		uuid.New().String(),
		model.ID.String(),
		selectedColumn.BaseID.String(),
		columnName,
		title,
		"longText",
		"TEXT",
		description,
		helpers.InterfaceToJSONString(map[string]interface{}{}),
		false,
		false,
		false,
		orderIndex,
		createdBy,
		createdBy,
		now,
		now,
	)
	if err != nil {
		return app_errors.LogDatabaseError(err, "failed to insert split column metadata")
	}
	return nil
}

func (s tableManagementService) addSplitPhysicalColumn(tx *sql.Tx, schemaName, tableName, columnName string) error {
	query := fmt.Sprintf(
		`ALTER TABLE %s ADD COLUMN %s TEXT`,
		fmt.Sprintf(SchemaTableFormat, schemaName, tableName),
		fmt.Sprintf(QuotedColumnFormat, columnName),
	)
	if _, err := tx.ExecContext(context.Background(), query); err != nil {
		return app_errors.LogDatabaseError(err, "failed to add split column to table")
	}
	return nil
}

func (s tableManagementService) deleteSplitOriginalColumn(tx *sql.Tx, schemaName, tableName string, column dto.ColumnResponse) error {
	alterQuery := fmt.Sprintf(
		`ALTER TABLE %s DROP COLUMN %s`,
		fmt.Sprintf(SchemaTableFormat, schemaName, tableName),
		fmt.Sprintf(QuotedColumnFormat, column.ColumnName),
	)
	if _, err := tx.ExecContext(context.Background(), alterQuery); err != nil {
		return app_errors.LogDatabaseError(err, "failed to drop original split column")
	}

	deleteQuery := fmt.Sprintf(
		`DELETE FROM %s WHERE id = $1`,
		fmt.Sprintf(`"%s".columns`, schemaName),
	)
	if _, err := tx.ExecContext(context.Background(), deleteQuery, column.ID.String()); err != nil {
		return app_errors.LogDatabaseError(err, "failed to remove original split column metadata")
	}

	return nil
}

func (s tableManagementService) getSplitSQLArrayExpr(columnName string, strategy columnSplitStrategy) (string, []interface{}) {
	quotedCol := fmt.Sprintf(QuotedColumnFormat, columnName)
	switch strategy.kind {
	case "separator":
		pattern := fmt.Sprintf("(?:%s)+", regexp.QuoteMeta(strategy.separator))
		return fmt.Sprintf("ARRAY(SELECT x FROM unnest(regexp_split_to_array(COALESCE(%s, ''), $1)) x WHERE TRIM(x) <> '')", quotedCol), []interface{}{pattern}
	case "fixedLength":
		var partsExpr string
		if strategy.action == "after" {
			partsExpr = fmt.Sprintf("ARRAY[substring(COALESCE(%s, '') from 1 for %d), substring(COALESCE(%s, '') from %d)]", quotedCol, strategy.value, quotedCol, strategy.value+1)
		} else { // before
			partsExpr = fmt.Sprintf(
				`CASE 
					WHEN char_length(COALESCE(%s, '')) < %d THEN ARRAY[]::text[]
					ELSE ARRAY[
						substring(COALESCE(%s, '') from 1 for char_length(COALESCE(%s, '')) - %d), 
						substring(COALESCE(%s, '') from char_length(COALESCE(%s, '')) - %d + 1)
					]
				 END`,
				quotedCol, strategy.value,
				quotedCol, quotedCol, strategy.value,
				quotedCol, quotedCol, strategy.value,
			)
		}
		return fmt.Sprintf("ARRAY(SELECT x FROM unnest(%s) x WHERE TRIM(x) <> '')", partsExpr), nil
	case "pattern":
		return fmt.Sprintf("ARRAY(SELECT x FROM unnest(regexp_split_to_array(COALESCE(%s, ''), $1)) x WHERE TRIM(x) <> '')", quotedCol), []interface{}{strategy.pattern}
	default:
		return "ARRAY[]::text[]", nil
	}
}

func (s tableManagementService) performBulkSplitUpdate(
	ctx context.Context,
	schemaName string,
	tableName string,
	columnName string,
	newColumnNames []string,
	strategy columnSplitStrategy,
	columnCount int,
) error {
	fullTableName := fmt.Sprintf(SchemaTableFormat, schemaName, tableName)

	// Fetch all rows
	params := dbModels.QueryParams{
		Select: []string{"id", columnName},
	}
	rows, err := s.repo.TableService.GetTableData(fullTableName, params)
	if err != nil {
		return app_errors.LogDatabaseError(err, "failed to fetch rows for column split")
	}

	batchSize := 2000
	updates := make([]dto.UpdateColumnValueRequest, 0, batchSize*columnCount)

	for idx, row := range rows {
		rowID, hasRowID := row["id"]
		if !hasRowID {
			continue
		}

		var valStr string
		if valRaw, ok := row[columnName]; ok && valRaw != nil {
			if val, ok := valRaw.(string); ok {
				valStr = val
			}
		}

		// Split the string in Go
		rawParts := splitStringInGo(valStr, strategy)

		// Apply the column limit
		parts := applySplitColumnLimit(rawParts, columnCount, splitJoinSeparator(strategy))

		// Build updates for each new column
		for colIdx := 0; colIdx < columnCount; colIdx++ {
			var valToSet interface{}
			if colIdx < len(parts) {
				valToSet = parts[colIdx]
			} else {
				valToSet = nil
			}

			updates = append(updates, dto.UpdateColumnValueRequest{
				Id:     rowID,
				Column: newColumnNames[colIdx],
				Value:  valToSet,
			})
		}

		// Update in batches of 2000 rows
		if (idx+1)%batchSize == 0 {
			if err := s.columnsService.BulkUpdateByColumns(ctx, schemaName, tableName, updates); err != nil {
				return err
			}
			updates = updates[:0] // Clear the slice while retaining capacity
		}
	}

	// Flush any remaining updates
	if len(updates) > 0 {
		if err := s.columnsService.BulkUpdateByColumns(ctx, schemaName, tableName, updates); err != nil {
			return err
		}
	}

	return nil
}

func splitStringInGo(val string, strategy columnSplitStrategy) []string {
	var parts []string
	switch strategy.kind {
	case "separator":
		sepPattern := fmt.Sprintf("(?:%s)+", regexp.QuoteMeta(strategy.separator))
		re := regexp.MustCompile(sepPattern)
		rawParts := re.Split(val, -1)
		for _, part := range rawParts {
			if strings.TrimSpace(part) != "" {
				parts = append(parts, part)
			}
		}
	case "fixedLength":
		runes := []rune(val)
		if len(runes) < strategy.value {
			if len(runes) > 0 {
				parts = []string{string(runes)}
			}
		} else {
			if strategy.action == "after" {
				parts = []string{
					string(runes[:strategy.value]),
					string(runes[strategy.value:]),
				}
			} else { // before
				splitIdx := len(runes) - strategy.value
				parts = []string{
					string(runes[:splitIdx]),
					string(runes[splitIdx:]),
				}
			}
		}
		var filteredParts []string
		for _, part := range parts {
			if strings.TrimSpace(part) != "" {
				filteredParts = append(filteredParts, part)
			}
		}
		parts = filteredParts
	case "pattern":
		if strategy.regex != nil {
			rawParts := strategy.regex.Split(val, -1)
			for _, part := range rawParts {
				if strings.TrimSpace(part) != "" {
					parts = append(parts, part)
				}
			}
		}
	}
	return parts
}

func (s tableManagementService) MergeColumns(ctx context.Context, schemaName string, req dto.MergeColumnsRequest) (dto.MergeColumnsResponse, error) {
	lg := logger.Get()

	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.MergeColumnsResponse{}, err
	}

	columnsData, err := s.GetColumnsByModelID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.MergeColumnsResponse{}, err
	}

	selectedColumns, err := s.getSelectedColumnsFromRequest(columnsData, req.Columns)
	if err != nil {
		return dto.MergeColumnsResponse{}, err
	}

	sep := determineMergeSeparator(req.MergeFormat, req.CustomSeparator)

	baseTitle := strings.TrimSpace(req.NewColumnTitle)
	if baseTitle == "" {
		baseTitle = combineColumnTitles(selectedColumns, columnsData)
	}
	colTitle := uniqueTitleFromBase(baseTitle, columnsData)

	// Generate a slugified name (title + timestamp) and ensure uniqueness.
	baseName := s.slugify(colTitle)
	uniqueName := uniqueNameFromBase(baseName, columnsData)

	desiredOrderIndex, err := s.determineDesiredOrderIndex(ctx, schemaName, req, columnsData)
	if err != nil {
		return dto.MergeColumnsResponse{}, err
	}

	// create column and add to DB (pass explicit title)
	newCol, err := s.createNewColumnForMerge(ctx, schemaName, model, req, uniqueName, colTitle, desiredOrderIndex)
	if err != nil {
		return dto.MergeColumnsResponse{}, err
	}

	// fetch rows for merging
	selectColumns := make([]string, 0, len(selectedColumns)+1)
	selectColumns = append(selectColumns, "id")
	selectColumns = append(selectColumns, selectedColumns...)

	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)
	rows, err := s.fetchTableRowsForTrim(ctx, tableName, selectColumns)
	if err != nil {
		return dto.MergeColumnsResponse{}, err
	}

	updates, result := s.buildMergeUpdates(rows, selectedColumns, sep, newCol.ColumnName)

	if err := s.applyBulkUpdates(ctx, schemaName, model.Alias, updates); err != nil {
		return dto.MergeColumnsResponse{}, err
	}

	// optionally delete original columns
	if !req.KeepOriginalColumn {
		if err := s.deleteOriginalColumnsIfNeeded(ctx, schemaName, req, columnsData); err != nil {
			return dto.MergeColumnsResponse{}, err
		}
	}

	lg.Info().
		Str("model_id", req.ModelID).
		Int("columns_selected", len(selectedColumns)).
		Int("total_scanned", result.TotalScanned).
		Int("total_updated", result.TotalUpdated).
		Int("total_skipped", result.TotalSkipped).
		Int("total_rows", result.TotalRows).
		Int("total_rows_updated", result.TotalRowsUpdated).
		Int("total_rows_skipped", result.TotalRowsSkipped).
		Str("generated_column", newCol.ColumnName).
		Msg("Merge columns action completed")

	result.GeneratedColumn = newCol.ColumnName
	return result, nil
}

func (s tableManagementService) ExtractSubstring(ctx context.Context, schemaName string, req dto.ExtractSubstringRequest) (dto.ExtractSubstringResponse, error) {
	lg := logger.Get()

	model, err := s.modelService.GetModelByID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.ExtractSubstringResponse{}, err
	}

	columnsData, err := s.GetColumnsByModelID(ctx, schemaName, req.ModelID)
	if err != nil {
		return dto.ExtractSubstringResponse{}, err
	}

	selectedCol, found := findColumnByID(columnsData, req.ColumnId)
	if !found {
		return dto.ExtractSubstringResponse{}, app_errors.ColumnNotFound
	}

	effectiveType, err := validateExtractSubstringRequest(req)
	if err != nil {
		return dto.ExtractSubstringResponse{}, err
	}

	sourceColumnName := selectedCol.ColumnName

	// target column base name: <source>_<effective_type>
	baseName := fmt.Sprintf("%s_%s", sourceColumnName, effectiveType)
	uniqueName := uniqueNameFromBase(baseName, columnsData)

	// Title
	baseTitle := strings.TrimSpace(selectedCol.Title + " " + effectiveType)
	if baseTitle == "" {
		baseTitle = uniqueName
	}
	colTitle := uniqueTitleFromBase(baseTitle, columnsData)

	// determine order index and create column
	mergeReq := dto.MergeColumnsRequest{ModelID: req.ModelID, Columns: []string{req.ColumnId}, AddAtEnd: req.AddAtEnd}
	desiredOrderIndex, err := s.determineDesiredOrderIndex(ctx, schemaName, mergeReq, columnsData)
	if err != nil {
		return dto.ExtractSubstringResponse{}, err
	}

	newCol, err := s.createNewColumnForMerge(ctx, schemaName, model, mergeReq, uniqueName, colTitle, desiredOrderIndex)
	if err != nil {
		return dto.ExtractSubstringResponse{}, err
	}

	// Fetch only id and source column
	selectColumns := []string{"id", sourceColumnName}
	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)
	rows, err := s.fetchTableRowsForTrim(ctx, tableName, selectColumns)
	if err != nil {
		return dto.ExtractSubstringResponse{}, err
	}

	result := dto.ExtractSubstringResponse{
		Column:          sourceColumnName,
		GeneratedColumn: newCol.ColumnName,
		ExtractionType:  effectiveType,
		ScannedRecords:  len(rows),
	}

	updates, updatedCount, skippedCount := s.buildExtractSubstringUpdates(rows, sourceColumnName, newCol.ColumnName, effectiveType, req)
	result.UpdatedRecords = updatedCount
	result.SkippedRecords = skippedCount

	if len(updates) > 0 {
		if err := s.applyBulkUpdates(ctx, schemaName, model.Alias, updates); err != nil {
			return dto.ExtractSubstringResponse{}, err
		}
	}

	if !req.KeepOriginalColumn {
		if err := s.DeleteColumnAndCleanUp(ctx, schemaName, selectedCol.ID.String(), selectedCol); err != nil {
			return dto.ExtractSubstringResponse{}, err
		}
	}

	lg.Info().
		Str("model_id", req.ModelID).
		Str("source_column", sourceColumnName).
		Str("generated_column", newCol.ColumnName).
		Int("scanned_records", result.ScannedRecords).
		Int("updated_records", result.UpdatedRecords).
		Int("skipped_records", result.SkippedRecords).
		Msg("Extract substring action completed")

	return result, nil
}

func findColumnByID(columnsData []dto.ColumnResponse, columnID string) (dto.ColumnResponse, bool) {
	targetID := strings.TrimSpace(columnID)
	for _, c := range columnsData {
		if c.ID.String() == targetID {
			return c, true
		}
	}
	return dto.ColumnResponse{}, false
}

func validateExtractSubstringRequest(req dto.ExtractSubstringRequest) (string, error) {
	method := strings.ToLower(strings.TrimSpace(req.ExtractionMethod))
	switch method {
	case "extraction_type":
		if strings.TrimSpace(req.ExtractionType) == "" {
			return "", app_errors.InvalidPayload
		}
		allowed := map[string]struct{}{
			"email": {}, "keywords": {}, "mentions": {}, "tags": {}, "url": {}, "domain": {}, "emoji": {}, "phone": {}, "prefix": {},
		}
		effectiveType := strings.ToLower(strings.TrimSpace(req.ExtractionType))
		if _, ok := allowed[effectiveType]; !ok {
			return "", app_errors.InvalidPayload
		}
		return effectiveType, nil
	case "between_characters":
		if strings.TrimSpace(req.StartAfter) == "" || strings.TrimSpace(req.EndBefore) == "" {
			return "", app_errors.InvalidPayload
		}
		return "between_characters", nil
	default:
		return "", app_errors.InvalidPayload
	}
}

func extractSubstringByType(strVal, extractionType, startAfter, endBefore string) (string, bool) {
	switch extractionType {
	case "between_characters":
		return extractBetweenCharactersFromText(strVal, startAfter, endBefore)
	case "email":
		return extractFirstEmail(strVal)
	case "url":
		return extractURLsFromText(strVal)
	case "domain":
		return extractDomainFromText(strVal)
	case "tags":
		return extractHashtagsFromText(strVal)
	case "mentions":
		return extractMentionsFromText(strVal)
	case "keywords":
		return extractKeywordsFromText(strVal)
	case "emoji":
		return extractEmojiFromText(strVal)
	case "phone":
		return extractPhoneNumberFromText(strVal)
	case "prefix":
		return extractEmailPrefixFromText(strVal)
	default:
		return "", false
	}
}

func (s tableManagementService) buildExtractSubstringUpdates(
	rows []map[string]interface{},
	sourceColumnName string,
	generatedColumnName string,
	effectiveType string,
	req dto.ExtractSubstringRequest,
) ([]dto.UpdateColumnValueRequest, int, int) {
	updates := make([]dto.UpdateColumnValueRequest, 0)
	updated := 0
	skipped := 0

	for _, row := range rows {
		rowID, hasRowID := row["id"]
		if !hasRowID {
			skipped++
			continue
		}

		value, ok := row[sourceColumnName]
		if !ok || value == nil {
			skipped++
			continue
		}

		strVal, ok := value.(string)
		if !ok {
			skipped++
			continue
		}

		extracted, ok := extractSubstringByType(strVal, effectiveType, req.StartAfter, req.EndBefore)
		if !ok || strings.TrimSpace(extracted) == "" {
			skipped++
			continue
		}

		updates = append(updates, dto.UpdateColumnValueRequest{
			Id:     rowID,
			Column: generatedColumnName,
			Value:  extracted,
		})
		updated++
	}

	return updates, updated, skipped
}

// determineMergeSeparator returns the separator string for merge format
func determineMergeSeparator(format, custom string) string {
	switch format {
	case "space":
		return " "
	case "comma":
		return ", "
	case "dash":
		return "-"
	case "custom":
		return custom
	default:
		return " "
	}
}

// uniqueNameFromBase appends a numeric suffix when necessary to make the base name unique against existing columns.
func uniqueNameFromBase(baseName string, columnsData []dto.ColumnResponse) string {
	uniqueName := baseName
	suffix := 1
	for {
		exists := false
		for _, c := range columnsData {
			if c.ColumnName == uniqueName {
				exists = true
				break
			}
		}
		if !exists {
			break
		}
		uniqueName = fmt.Sprintf("%s_%d", baseName, suffix)
		suffix++
	}
	return uniqueName
}

func combineColumnTitles(selectedColumns []string, columnsData []dto.ColumnResponse) string {
	// map column_name -> title
	titleMap := make(map[string]string, len(columnsData))
	for _, c := range columnsData {
		titleMap[c.ColumnName] = c.Title
	}

	parts := make([]string, 0, len(selectedColumns))
	for _, col := range selectedColumns {
		t := strings.TrimSpace(titleMap[col])
		if t == "" {
			t = col
		}
		parts = append(parts, t)
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func uniqueTitleFromBase(base string, columnsData []dto.ColumnResponse) string {
	const maxTitleLen = 50

	baseTrim := strings.TrimSpace(base)
	if baseTrim == "" {
		return baseTrim
	}

	// truncateRunes trims the string to at most n runes.
	truncateRunes := func(s string, n int) string {
		r := []rune(strings.TrimSpace(s))
		if len(r) <= n {
			return string(r)
		}
		return string(r[:n])
	}

	titleExists := func(candidate string) bool {
		for _, c := range columnsData {
			if strings.EqualFold(strings.TrimSpace(c.Title), candidate) {
				return true
			}
		}
		return false
	}

	// First try using the base title truncated to the max length.
	candidate := truncateRunes(baseTrim, maxTitleLen)
	if !titleExists(candidate) {
		return candidate
	}

	// Append numeric suffixes (" 2", " 3", ...) ensuring total length <= maxTitleLen.
	suffix := 2
	for {
		suffixStr := fmt.Sprintf(" %d", suffix)
		// reserve space for the suffix (counted in runes)
		maxBase := maxTitleLen - len([]rune(suffixStr))
		if maxBase < 0 {
			maxBase = 0
		}
		basePart := truncateRunes(baseTrim, maxBase)
		candidate = basePart + suffixStr
		if !titleExists(candidate) {
			return candidate
		}
		suffix++
	}
}

// determineDesiredOrderIndex resolves the order index where the new column should be inserted and shifts existing columns when required.
func (s tableManagementService) determineDesiredOrderIndex(ctx context.Context, schemaName string, req dto.MergeColumnsRequest, columnsData []dto.ColumnResponse) (float64, error) {
	// If adding at end, simply return maxOrder + 1
	if req.AddAtEnd {
		maxOrder, err := s.columnsService.GetMaxOrderIndexOfColumn(ctx, schemaName, req.ModelID)
		if err != nil {
			return 0, err
		}
		return maxOrder + 1, nil
	}

	// Attempt to find the order index of the last selected column
	lastSel := strings.TrimSpace(req.Columns[len(req.Columns)-1])
	if idx, ok := findLastSelectedOrderIndex(columnsData, lastSel); ok {
		desiredOrderIndex := idx + 1
		// Shift affected columns upward in a single helper
		if err := s.shiftColumnsStartingFrom(ctx, schemaName, desiredOrderIndex, columnsData); err != nil {
			return 0, err
		}
		return desiredOrderIndex, nil
	}

	// Fallback to appending at end when last selected not found
	maxOrder, err := s.columnsService.GetMaxOrderIndexOfColumn(ctx, schemaName, req.ModelID)
	if err != nil {
		return 0, err
	}
	return maxOrder + 1, nil
}

// findLastSelectedOrderIndex returns the order index of the column with the given ID string, and whether it was found.
func findLastSelectedOrderIndex(columnsData []dto.ColumnResponse, lastSel string) (float64, bool) {
	for _, c := range columnsData {
		if c.ID.String() == lastSel {
			if c.OrderIndex != nil {
				return *c.OrderIndex, true
			}
			return 0, true
		}
	}
	return 0, false
}

// shiftColumnsStartingFrom increments OrderIndex for columns with index >= start
func (s tableManagementService) shiftColumnsStartingFrom(ctx context.Context, schemaName string, start float64, columnsData []dto.ColumnResponse) error {
	for _, c := range columnsData {
		if c.OrderIndex != nil && *c.OrderIndex >= start {
			upd := dto.ColumnUpdate{
				OrderIndex: helpers.Float64Ptr(*c.OrderIndex + 1),
				UpdatedAt:  time.Now().UTC(),
			}
			if _, err := s.columnsService.UpdateColumn(ctx, schemaName, c.ID.String(), upd); err != nil {
				return err
			}
		}
	}
	return nil
}

// createNewColumnForMerge encapsulates creation of the new column metadata and the actual DB column addition.
func (s tableManagementService) createNewColumnForMerge(ctx context.Context, schemaName string, model tenant.Model, req dto.MergeColumnsRequest, uniqueName, title string, desiredOrderIndex float64) (tenant.Column, error) {
	// Use `longText` UIDT so merged values are stored as long text to avoid truncation.
	dt, err := s.getDataBaseType("longText")
	if err != nil {
		return tenant.Column{}, err
	}
	columnInsert := dto.ColumnInsertion{
		ID:          uuid.New(),
		ModelID:     uuid.MustParse(req.ModelID),
		BaseID:      model.BaseID,
		Title:       title,
		ColumnName:  uniqueName,
		Description: helpers.StringPtr(""),
		Meta:        map[string]interface{}{},
		UIDT:        "longText",
		DT:          helpers.StringPtr(dt),
		Virtual:     false,
		System:      false,
		Deleted:     false,
		OrderIndex:  helpers.Float64Ptr(desiredOrderIndex),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	newCol, err := s.columnsService.Create(ctx, columnInsert, schemaName)
	if err != nil {
		return tenant.Column{}, err
	}

	if err := s.addColumnInTableDb(schemaName, model.Alias, newCol); err != nil {
		return tenant.Column{}, err
	}

	return newCol, nil
}

// buildMergeUpdates processes rows and returns the list of updates and aggregated result counters.
func (s tableManagementService) buildMergeUpdates(rows []map[string]interface{}, selectedColumns []string, sep, newColumnName string) ([]dto.UpdateColumnValueRequest, dto.MergeColumnsResponse) {
	result := dto.MergeColumnsResponse{
		TotalScanned: len(rows) * len(selectedColumns),
		TotalRows:    len(rows),
	}

	if len(rows) == 0 {
		return nil, result
	}

	updates := make([]dto.UpdateColumnValueRequest, 0)

	for _, row := range rows {
		rowID, hasRowID := row["id"]
		if !hasRowID {
			result.TotalRowsSkipped++
			result.TotalSkipped += len(selectedColumns)
			continue
		}

		tokens, skipped := collectTokensFromRow(row, selectedColumns)
		result.TotalSkipped += skipped

		if len(tokens) == 0 {
			result.TotalRowsSkipped++
			continue
		}

		merged := strings.TrimSpace(strings.Join(tokens, sep))
		if merged == "" {
			result.TotalRowsSkipped++
			continue
		}

		updates = append(updates, dto.UpdateColumnValueRequest{
			Id:     rowID,
			Column: newColumnName,
			Value:  merged,
		})
		result.TotalUpdated++
		result.TotalRowsUpdated++
	}

	return updates, result
}

// collectTokensFromRow extracts trimmed string tokens from the selected columns in a row and returns them along with how many cells were skipped.
func collectTokensFromRow(row map[string]interface{}, selectedColumns []string) ([]string, int) {
	tokens := make([]string, 0, len(selectedColumns))
	skipped := 0
	for _, colName := range selectedColumns {
		value, exists := row[colName]
		if !exists || value == nil {
			skipped++
			continue
		}
		var strVal string
		switch v := value.(type) {
		case string:
			strVal = strings.TrimSpace(v)
		default:
			strVal = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
		if strVal == "" {
			skipped++
			continue
		}
		tokens = append(tokens, strVal)
	}
	return tokens, skipped
}

// applyBulkUpdates performs batched BulkUpdateByColumns calls.
func (s tableManagementService) applyBulkUpdates(ctx context.Context, schemaName, modelAlias string, updates []dto.UpdateColumnValueRequest) error {
	if len(updates) == 0 {
		return nil
	}
	batchSize := columnActionBatchSize
	for start := 0; start < len(updates); start += batchSize {
		end := start + batchSize
		if end > len(updates) {
			end = len(updates)
		}
		if err := s.columnsService.BulkUpdateByColumns(ctx, schemaName, modelAlias, updates[start:end]); err != nil {
			return err
		}
	}
	return nil
}

// deleteOriginalColumnsIfNeeded removes original columns when KeepOriginalColumn is false.
func (s tableManagementService) deleteOriginalColumnsIfNeeded(ctx context.Context, schemaName string, req dto.MergeColumnsRequest, columnsData []dto.ColumnResponse) error {
	for _, colID := range req.Columns {
		colID = strings.TrimSpace(colID)
		for _, c := range columnsData {
			if c.ID.String() == colID {
				if err := s.DeleteColumnAndCleanUp(ctx, schemaName, c.ID.String(), c); err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}