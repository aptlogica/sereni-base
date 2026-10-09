// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"errors"
	"fmt"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
	"github.com/google/uuid"
)

const (
	uidtLinks  = "links"
	uidtLookup = "lookup"
	uidtRollup = "rollup"

	entityRoleSource = "source"
	entityRoleTarget = "target"

	relationOneToOne = "one-to-one"
	relationHasMany  = "has-many"

	linkDataTypeInt      = "INT"
	linkDataTypeIntArray = "INT[]"

	linkActionLink   = "link"
	linkActionUnlink = "unlink"

	lookupDataFunctionName           = "get_table_data_with_lookups"
	linkRowsFunctionName             = "link_rows"
	removeBackReferencesFunctionName = "remove_link_back_references"

	// Statuses returned by link_rows.
	linkStatusOK             = "ok"
	linkStatusSourceNotFound = "source_not_found"
	linkStatusTargetNotFound = "target_not_found"
	linkStatusCycle          = "cycle"
)

// linkColumnInfo is the parsed meta of a "links" column.
type linkColumnInfo struct {
	RelationID   string
	EntityRole   string
	RelationType string
	With         string // model ID of the other side
}

// linkSide describes one side of a relation: the table and link column that hold IDs of the other side.
type linkSide struct {
	TableName  string
	ColumnName string
	DataType   string
}

// metaString reads a string value from column meta. Values written before a DB round-trip may still be uuid.UUID.
func metaString(meta map[string]interface{}, key string) string {
	if meta == nil {
		return ""
	}
	switch v := meta[key].(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return ""
	}
}

func parseLinkMeta(meta map[string]interface{}) (linkColumnInfo, bool) {
	info := linkColumnInfo{
		RelationID: metaString(meta, "relation_id"),
		EntityRole: metaString(meta, "entity_role"),
	}
	if relation, ok := meta["relation"].(map[string]interface{}); ok {
		info.RelationType, _ = relation["type"].(string)
		info.With, _ = relation["with"].(string)
	}
	ok := info.RelationID != "" && (info.EntityRole == entityRoleSource || info.EntityRole == entityRoleTarget)
	return info, ok
}

func oppositeEntityRole(role string) string {
	if role == entityRoleSource {
		return entityRoleTarget
	}
	return entityRoleSource
}

func isSelfRelation(relation tenant.Relation) bool {
	return relation.SourceModelID == relation.TargetModelID
}

// isSingleColumnRelation reports a self-link stored in one one-way column (source and target are the
// same column). Self-links created before this existed have two columns and use the two-sided path.
func isSingleColumnRelation(relation tenant.Relation) bool {
	return relation.SourceColumnID == relation.TargetColumnID
}

// linkDataType returns the physical type (INT or INT[]) of a link column for the given role and relation type.
func (s tableManagementService) linkDataType(entityRole, relationType string) (string, error) {
	return s.getDataBaseType(fmt.Sprintf("%s_%s_%s", uidtLinks, entityRole, relationType))
}

// ---------------------------------------------------------------------------
// Lookup → link column resolution
// ---------------------------------------------------------------------------

// pickLegacyLinkColumn finds the link column for a lookup created before meta.link_column_id existed.
// On a self-link both columns share the relation; the source side wins.
func pickLegacyLinkColumn(columns []dto.ColumnResponse, relationID string) (dto.ColumnResponse, bool) {
	var found dto.ColumnResponse
	matched := false
	for _, col := range columns {
		if col.UIDT != uidtLinks {
			continue
		}
		info, ok := parseLinkMeta(col.Meta)
		if !ok || info.RelationID != relationID {
			continue
		}
		if !matched || info.EntityRole == entityRoleSource {
			found = col
			matched = true
		}
	}
	return found, matched
}

// lookupLinkColumn resolves the link column a lookup goes through, using already-loaded columns of the lookup's table.
func lookupLinkColumn(lookup dto.ColumnResponse, relationID string, columns []dto.ColumnResponse) (dto.ColumnResponse, bool) {
	linkColumnID := metaString(lookup.Meta, "link_column_id")
	if linkColumnID == "" {
		return pickLegacyLinkColumn(columns, relationID)
	}
	for _, col := range columns {
		if col.ID.String() != linkColumnID || col.UIDT != uidtLinks {
			continue
		}
		info, ok := parseLinkMeta(col.Meta)
		if ok && info.RelationID == relationID {
			return col, true
		}
	}
	return dto.ColumnResponse{}, false
}

// resolveLookupLinkColumn validates and returns the link column on modelID that a lookup goes through.
func (s tableManagementService) resolveLookupLinkColumn(
	ctx context.Context,
	schemaName string,
	modelID string,
	relationID string,
	linkColumnID string,
) (dto.ColumnResponse, error) {
	columns, err := s.GetColumnsByModelID(ctx, schemaName, modelID)
	if err != nil {
		return dto.ColumnResponse{}, err
	}
	lookup := dto.ColumnResponse{Meta: map[string]interface{}{"link_column_id": linkColumnID}}
	linkCol, ok := lookupLinkColumn(lookup, relationID, columns)
	if !ok {
		return dto.ColumnResponse{}, app_errors.InvalidLookupLinkColumn
	}
	return linkCol, nil
}

// validateLookupTarget checks that the looked-up column lives on the other side of the link and is a plain field.
func validateLookupTarget(linkCol dto.ColumnResponse, lookupColumn tenant.Column) error {
	info, _ := parseLinkMeta(linkCol.Meta)
	if info.With == "" || lookupColumn.ModelID != info.With {
		return app_errors.InvalidLookupLinkColumn
	}
	switch lookupColumn.UIDT {
	case uidtLinks, uidtLookup, uidtRollup:
		return app_errors.InvalidLookupTargetColumn
	}
	return nil
}

// lookupColumnName builds a unique physical-looking name for a lookup column. It is used as the
// output key of the read query, so two lookups of the same field through different links never collide.
func lookupColumnName(lookupColumn tenant.Column) string {
	return fmt.Sprintf("lk_%s_%s", uuid.NewString()[:8], lookupColumn.ColumnName)
}

// ---------------------------------------------------------------------------
// Read path: build relation data from lookup columns
// ---------------------------------------------------------------------------

// buildLookupRelationData groups the table's lookup columns by the link column they go through.
// Each entry tells get_table_data_with_lookups which foreign columns to pull and the output key for each.
func (s tableManagementService) buildLookupRelationData(
	ctx context.Context,
	schemaName string,
	columnsData []dto.ColumnResponse,
) []map[string]interface{} {
	entries := map[string]map[string]interface{}{}
	var order []string
	modelAliases := map[string]string{}

	for _, col := range columnsData {
		linkCol, foreign, ok := s.resolveLookupSource(ctx, schemaName, col, columnsData)
		if !ok {
			continue
		}

		key := linkCol.ID.String()
		entry, exists := entries[key]
		if !exists {
			entry, ok = s.newLookupRelationEntry(ctx, schemaName, linkCol, modelAliases)
			if !ok {
				continue
			}
			entries[key] = entry
			order = append(order, key)
		}
		entry["targets"] = append(entry["targets"].([]map[string]interface{}), map[string]interface{}{
			"column": foreign.ColumnName,
			"alias":  col.ColumnName,
		})
	}

	relationData := make([]map[string]interface{}, 0, len(order))
	for _, key := range order {
		relationData = append(relationData, entries[key])
	}
	return relationData
}

// resolveLookupSource returns the link column a lookup reads through and the foreign column it shows.
// ok is false for non-lookup columns and for lookups that can't be resolved (those are skipped).
func (s tableManagementService) resolveLookupSource(
	ctx context.Context,
	schemaName string,
	col dto.ColumnResponse,
	columnsData []dto.ColumnResponse,
) (dto.ColumnResponse, tenant.Column, bool) {
	if col.UIDT != uidtLookup {
		return dto.ColumnResponse{}, tenant.Column{}, false
	}
	lookupColumnID, relationID, ok := s.validateMetaForLookup(col.Meta)
	if !ok {
		return dto.ColumnResponse{}, tenant.Column{}, false
	}
	lg := logger.Get()
	linkCol, ok := lookupLinkColumn(col, relationID, columnsData)
	if !ok {
		lg.Warn().Str("lookupColumnID", col.ID.String()).Msg("Lookup has no matching link column, skipping")
		return dto.ColumnResponse{}, tenant.Column{}, false
	}
	info, _ := parseLinkMeta(linkCol.Meta)

	foreign, err := s.columnsService.GetColumnByID(ctx, schemaName, lookupColumnID)
	if err != nil || foreign.ModelID != info.With {
		lg.Warn().Str("lookupColumnID", col.ID.String()).Msg("Looked-up column is missing or on another table, skipping")
		return dto.ColumnResponse{}, tenant.Column{}, false
	}
	return linkCol, foreign, true
}

// newLookupRelationEntry builds the relation_data entry for one link column. Target table aliases are
// cached in modelAliases; ok is false if the linked model can't be loaded.
func (s tableManagementService) newLookupRelationEntry(
	ctx context.Context,
	schemaName string,
	linkCol dto.ColumnResponse,
	modelAliases map[string]string,
) (map[string]interface{}, bool) {
	info, _ := parseLinkMeta(linkCol.Meta)
	alias, cached := modelAliases[info.With]
	if !cached {
		targetModel, err := s.modelService.GetModelByID(ctx, schemaName, info.With)
		if err != nil {
			return nil, false
		}
		alias = targetModel.Alias
		modelAliases[info.With] = alias
	}
	return map[string]interface{}{
		"source_column_name": linkCol.ColumnName,
		"relation":           info.RelationType,
		"is_array":           linkCol.DT == linkDataTypeIntArray,
		"target_table_name":  alias,
		"target_column_name": "id",
		"targets":            []map[string]interface{}{},
	}, true
}

// ---------------------------------------------------------------------------
// Link / unlink and row-delete cleanup (Postgres functions, see constant.DefinedFunctions)
// ---------------------------------------------------------------------------

// callRelationFunction runs one of the link SQL functions. Each call is a single statement, so
// everything it writes is applied together or not at all.
func (s tableManagementService) callRelationFunction(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	rows, err := s.repo.TableService.GetByFunction(ctx, fmt.Sprintf("%s.%s", constant.MasterDatabase, name), args)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to run "+name)
	}
	if len(rows) == 0 {
		return nil, app_errors.LogDatabaseError(fmt.Errorf("%s returned no result", name), "failed to run "+name)
	}
	return rows[0][name], nil
}

// linkRows links or unlinks two rows through link_rows: locks both rows, checks for loops,
// releases existing partners (one-to-one / has-many) and writes both sides.
func (s tableManagementService) linkRows(
	ctx context.Context,
	relation tenant.Relation,
	source, target linkSide,
	req dto.UpdateRowDataLinksRequest,
) error {
	action := linkActionUnlink
	if req.Action == linkActionLink {
		action = linkActionLink
	}
	result, err := s.callRelationFunction(ctx, linkRowsFunctionName, map[string]interface{}{
		"p_relation_type":   relation.RelationType,
		"p_single_column":   isSingleColumnRelation(relation),
		"p_is_self":         isSelfRelation(relation),
		"p_action":          action,
		"p_source_table":    source.TableName,
		"p_source_column":   source.ColumnName,
		"p_source_is_array": source.DataType == linkDataTypeIntArray,
		"p_target_table":    target.TableName,
		"p_target_column":   target.ColumnName,
		"p_target_is_array": target.DataType == linkDataTypeIntArray,
		"p_source_row":      int64(req.SourceRowId),
		"p_target_row":      int64(req.TargetRowId),
		"p_updated_by":      req.UpdatedBy,
	})
	if err != nil {
		return err
	}
	return linkStatusError(result)
}

// linkStatusError maps the status returned by link_rows to the service error.
func linkStatusError(result interface{}) error {
	switch functionResultString(result) {
	case linkStatusOK:
		return nil
	case linkStatusSourceNotFound:
		return app_errors.RowNotFound
	case linkStatusTargetNotFound:
		return app_errors.LinkTargetRowNotFound
	case linkStatusCycle:
		return app_errors.LinkCycleDetected
	default:
		return app_errors.LogDatabaseError(fmt.Errorf("unexpected status %v", result), "failed to update link data")
	}
}

func functionResultString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return ""
	}
}

// backReferencePartners returns, for every link column on the model, the partner column that can
// hold IDs of this model's rows.
func (s tableManagementService) backReferencePartners(
	ctx context.Context,
	schemaName string,
	sourceModel tenant.Model,
) ([]map[string]interface{}, error) {
	columns, err := s.columnsService.GetColumnByModelID(ctx, schemaName, sourceModel.ID.String())
	if err != nil {
		return nil, err
	}
	var partners []map[string]interface{}
	for _, column := range columns {
		if column.UIDT != uidtLinks {
			continue
		}
		partner, err := s.partnerSide(ctx, schemaName, column)
		if err != nil {
			return nil, err
		}
		partners = append(partners, map[string]interface{}{
			"table":    partner.TableName,
			"column":   partner.ColumnName,
			"is_array": partner.DataType == linkDataTypeIntArray,
		})
	}
	return partners, nil
}

// partnerSide returns the other side (table, column, type) of a link column.
func (s tableManagementService) partnerSide(ctx context.Context, schemaName string, column tenant.Column) (linkSide, error) {
	info, ok := parseLinkMeta(column.Meta)
	if !ok {
		return linkSide{}, app_errors.InvalidColumnMetaForLinkType
	}
	relation, err := s.relationshipService.GetRelationByID(ctx, info.RelationID, schemaName)
	if err != nil {
		return linkSide{}, err
	}
	if isSingleColumnRelation(relation) {
		// One-way self-link: other rows of the same column hold references to this table's rows.
		model, err := s.modelService.GetModelByID(ctx, schemaName, column.ModelID)
		if err != nil {
			return linkSide{}, err
		}
		dataType, err := s.linkDataType(info.EntityRole, relation.RelationType)
		if err != nil {
			return linkSide{}, err
		}
		return linkSide{
			TableName:  fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias),
			ColumnName: column.ColumnName,
			DataType:   dataType,
		}, nil
	}
	partnerModelID, partnerColumnID := relation.TargetModelID, relation.TargetColumnID
	if info.EntityRole == entityRoleTarget {
		partnerModelID, partnerColumnID = relation.SourceModelID, relation.SourceColumnID
	}
	partnerModel, err := s.modelService.GetModelByID(ctx, schemaName, partnerModelID)
	if err != nil {
		return linkSide{}, err
	}
	partnerColumn, err := s.columnsService.GetColumnByID(ctx, schemaName, partnerColumnID)
	if err != nil {
		return linkSide{}, err
	}
	dataType, err := s.linkDataType(oppositeEntityRole(info.EntityRole), relation.RelationType)
	if err != nil {
		return linkSide{}, err
	}
	return linkSide{
		TableName:  fmt.Sprintf(SchemaTableFormat, schemaName, partnerModel.Alias),
		ColumnName: partnerColumn.ColumnName,
		DataType:   dataType,
	}, nil
}

func (s tableManagementService) columnExists(ctx context.Context, schemaName string, columnID string) (bool, error) {
	_, err := s.columnsService.GetColumnByID(ctx, schemaName, columnID)
	if errors.Is(err, app_errors.ColumnNotFound) {
		return false, nil
	}
	return err == nil, err
}
