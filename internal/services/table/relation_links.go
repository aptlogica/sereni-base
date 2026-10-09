// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
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

	linkActionLink = "link"

	lookupDataFunctionName = "get_table_data_with_lookups"
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
	lg := logger.Get()
	entries := map[string]map[string]interface{}{}
	var order []string
	modelAliases := map[string]string{}

	for _, col := range columnsData {
		if col.UIDT != uidtLookup {
			continue
		}
		lookupColumnID, relationID, ok := s.validateMetaForLookup(col.Meta)
		if !ok {
			continue
		}
		linkCol, ok := lookupLinkColumn(col, relationID, columnsData)
		if !ok {
			lg.Warn().Str("lookupColumnID", col.ID.String()).Msg("Lookup has no matching link column, skipping")
			continue
		}
		info, _ := parseLinkMeta(linkCol.Meta)

		foreign, err := s.columnsService.GetColumnByID(ctx, schemaName, lookupColumnID)
		if err != nil || foreign.ModelID != info.With {
			lg.Warn().Str("lookupColumnID", col.ID.String()).Msg("Looked-up column is missing or on another table, skipping")
			continue
		}

		key := linkCol.ID.String()
		entry, exists := entries[key]
		if !exists {
			alias, cached := modelAliases[info.With]
			if !cached {
				targetModel, err := s.modelService.GetModelByID(ctx, schemaName, info.With)
				if err != nil {
					continue
				}
				alias = targetModel.Alias
				modelAliases[info.With] = alias
			}
			entry = map[string]interface{}{
				"source_column_name": linkCol.ColumnName,
				"relation":           info.RelationType,
				"is_array":           linkCol.DT == linkDataTypeIntArray,
				"target_table_name":  alias,
				"target_column_name": "id",
				"targets":            []map[string]interface{}{},
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

// ---------------------------------------------------------------------------
// Transactional link / unlink
// ---------------------------------------------------------------------------

func (s tableManagementService) withTx(ctx context.Context, fn func(tx *sql.Tx) error) (err error) {
	tx, err := s.repo.DB.Begin()
	if err != nil {
		return app_errors.LogDatabaseError(err, "failed to start transaction")
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()
	if err = fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err = tx.Commit(); err != nil {
		return app_errors.LogDatabaseError(err, "failed to commit transaction")
	}
	return nil
}

// lockRow locks a row for the rest of the transaction and reports whether it exists.
func lockRow(ctx context.Context, tx *sql.Tx, tableName string, rowID int64) (bool, error) {
	var id int64
	err := tx.QueryRowContext(ctx, fmt.Sprintf(`SELECT id FROM %s WHERE id = $1 FOR UPDATE`, tableName), rowID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, app_errors.LogDatabaseError(err, "failed to lock row")
	}
	return true, nil
}

// applyLinkChange adds or removes value in one row's link column, atomically in SQL.
func applyLinkChange(ctx context.Context, tx *sql.Tx, side linkSide, rowID int64, value int64, link bool, updatedBy string) error {
	column := fmt.Sprintf(QuotedColumnFormat, side.ColumnName)
	var setExpr, where string
	switch {
	case side.DataType == linkDataTypeIntArray && link:
		setExpr = fmt.Sprintf(`%[1]s = CASE WHEN $1::int = ANY(COALESCE(%[1]s, '{}'::int[])) THEN %[1]s ELSE array_append(COALESCE(%[1]s, '{}'::int[]), $1::int) END`, column)
		where = "id = $2"
	case side.DataType == linkDataTypeIntArray:
		setExpr = fmt.Sprintf(`%[1]s = array_remove(%[1]s, $1::int)`, column)
		where = "id = $2"
	case link:
		setExpr = fmt.Sprintf(`%s = $1::int`, column)
		where = "id = $2"
	default:
		setExpr = fmt.Sprintf(`%s = NULL`, column)
		where = fmt.Sprintf(`id = $2 AND %s = $1::int`, column)
	}

	args := []interface{}{value, rowID, time.Now().UTC()}
	audit := `, last_modified_time = $3`
	if updatedBy != "" {
		audit += `, last_modified_by = $4`
		args = append(args, updatedBy)
	}

	query := fmt.Sprintf(`UPDATE %s SET %s%s WHERE %s`, side.TableName, setExpr, audit, where)
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return app_errors.LogDatabaseError(err, "failed to update link column")
	}
	return nil
}

// applyLinkPair writes both sides of one link: sourceRow.sourceColumn ↔ targetRow.targetColumn.
func applyLinkPair(ctx context.Context, tx *sql.Tx, source, target linkSide, sourceRowID, targetRowID int64, link bool, updatedBy string) error {
	if err := applyLinkChange(ctx, tx, source, sourceRowID, targetRowID, link, updatedBy); err != nil {
		return err
	}
	return applyLinkChange(ctx, tx, target, targetRowID, sourceRowID, link, updatedBy)
}

// rowsHoldingID returns rows of side.TableName whose link column contains value.
func rowsHoldingID(ctx context.Context, tx *sql.Tx, side linkSide, value int64) ([]int64, error) {
	column := fmt.Sprintf(QuotedColumnFormat, side.ColumnName)
	condition := fmt.Sprintf(`%s = $1::int`, column)
	if side.DataType == linkDataTypeIntArray {
		condition = fmt.Sprintf(`$1::int = ANY(%s)`, column)
	}
	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`SELECT id FROM %s WHERE %s FOR UPDATE`, side.TableName, condition), value)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to find linked rows")
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, app_errors.LogDatabaseError(err, "failed to scan linked row")
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// releaseExistingLinks enforces cardinality before a link: in one-to-one either row loses its current
// partner, and in has-many a child loses its current parent.
func releaseExistingLinks(ctx context.Context, tx *sql.Tx, relationType string, source, target linkSide, sourceRowID, targetRowID int64, updatedBy string) error {
	checks := []struct {
		holder, other linkSide
		id            int64
	}{
		{source, target, targetRowID},
		{target, source, sourceRowID},
	}
	for _, c := range checks {
		enforce := relationType == relationOneToOne ||
			(relationType == relationHasMany && c.holder.DataType == linkDataTypeIntArray)
		if !enforce {
			continue
		}
		holders, err := rowsHoldingID(ctx, tx, c.holder, c.id)
		if err != nil {
			return err
		}
		for _, holderID := range holders {
			if err := applyLinkPair(ctx, tx, c.holder, c.other, holderID, c.id, false, updatedBy); err != nil {
				return err
			}
		}
	}
	return nil
}

// wouldCreateCycle reports whether making parentID the parent of childID creates a loop.
// parentSide is the INT column that points from a child to its parent.
func wouldCreateCycle(ctx context.Context, tx *sql.Tx, parentSide linkSide, parentID, childID int64) (bool, error) {
	if parentID == childID {
		return true, nil
	}
	column := fmt.Sprintf(QuotedColumnFormat, parentSide.ColumnName)
	query := fmt.Sprintf(`
		WITH RECURSIVE ancestors(id) AS (
			SELECT %[1]s::bigint FROM %[2]s WHERE id = $1
			UNION
			SELECT t.%[1]s::bigint FROM %[2]s t JOIN ancestors a ON t.id = a.id
		)
		SELECT EXISTS (SELECT 1 FROM ancestors WHERE id = $2)`, column, parentSide.TableName)
	var exists bool
	if err := tx.QueryRowContext(ctx, query, parentID, childID).Scan(&exists); err != nil {
		return false, app_errors.LogDatabaseError(err, "failed to check link cycle")
	}
	return exists, nil
}

// applySingleColumnLink links or unlinks on a one-way self-link column: only rowID's cell changes.
// one-to-one and has-many still allow a record to be chosen by only one row, so other holders release it.
func applySingleColumnLink(ctx context.Context, tx *sql.Tx, side linkSide, relationType string, rowID, targetRowID int64, link bool, updatedBy string) error {
	if !link {
		return applyLinkChange(ctx, tx, side, rowID, targetRowID, false, updatedBy)
	}

	if relationType == relationHasMany {
		cycle, err := wouldCreateCycleInChildren(ctx, tx, side, rowID, targetRowID)
		if err != nil {
			return err
		}
		if cycle {
			return app_errors.LinkCycleDetected
		}
	}

	if relationType == relationOneToOne || relationType == relationHasMany {
		holders, err := rowsHoldingID(ctx, tx, side, targetRowID)
		if err != nil {
			return err
		}
		for _, holderID := range holders {
			if holderID == rowID {
				continue
			}
			if err := applyLinkChange(ctx, tx, side, holderID, targetRowID, false, updatedBy); err != nil {
				return err
			}
		}
	}

	return applyLinkChange(ctx, tx, side, rowID, targetRowID, true, updatedBy)
}

// wouldCreateCycleInChildren reports whether adding childID to parentID's children (an INT[] column)
// creates a loop, i.e. childID is already parentID or one of its ancestors.
func wouldCreateCycleInChildren(ctx context.Context, tx *sql.Tx, childrenSide linkSide, parentID, childID int64) (bool, error) {
	if parentID == childID {
		return true, nil
	}
	column := fmt.Sprintf(QuotedColumnFormat, childrenSide.ColumnName)
	query := fmt.Sprintf(`
		WITH RECURSIVE ancestors(id) AS (
			SELECT id::bigint FROM %[2]s WHERE $1::int = ANY(%[1]s)
			UNION
			SELECT t.id::bigint FROM %[2]s t JOIN ancestors a ON a.id::int = ANY(t.%[1]s)
		)
		SELECT EXISTS (SELECT 1 FROM ancestors WHERE id = $2)`, column, childrenSide.TableName)
	var exists bool
	if err := tx.QueryRowContext(ctx, query, parentID, childID).Scan(&exists); err != nil {
		return false, app_errors.LogDatabaseError(err, "failed to check link cycle")
	}
	return exists, nil
}

// ---------------------------------------------------------------------------
// Row delete: remove back-references
// ---------------------------------------------------------------------------

// removeBackReferences strips rowID from every partner link column that points at this table.
// It does not rely on the deleted row's own values, so one-sided or stale links are cleaned too.
func (s tableManagementService) removeBackReferences(
	ctx context.Context,
	tx *sql.Tx,
	schemaName string,
	sourceModel tenant.Model,
	rowIDs []int64,
	updatedBy string,
) error {
	columns, err := s.columnsService.GetColumnByModelID(ctx, schemaName, sourceModel.ID.String())
	if err != nil {
		return err
	}
	for _, column := range columns {
		if column.UIDT != uidtLinks {
			continue
		}
		partner, err := s.partnerSide(ctx, schemaName, column)
		if err != nil {
			return err
		}
		for _, rowID := range rowIDs {
			holders, err := rowsHoldingID(ctx, tx, partner, rowID)
			if err != nil {
				return err
			}
			for _, holderID := range holders {
				if err := applyLinkChange(ctx, tx, partner, holderID, rowID, false, updatedBy); err != nil {
					return err
				}
			}
		}
	}
	return nil
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
