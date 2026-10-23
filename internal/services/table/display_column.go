// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"strings"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
)

const uidtText = "text"

const (
	// metaDisplayColumnKey holds, in the meta of a link column, the id of the column of the linked
	// table that the link shows as the label of a linked record.
	metaDisplayColumnKey = "display_column_id"
	// relationInverseDisplayKey is sent inside meta.relation when a link is created: it is the display
	// column of the partner link column. It is read once and never stored on the source column.
	relationInverseDisplayKey = "inverse_display_column_id"
)

// displayColumnID reads the display column id from the meta of a link column.
func displayColumnID(meta map[string]interface{}) string {
	return metaString(meta, metaDisplayColumnKey)
}

// popInverseDisplayColumn removes meta.relation.inverse_display_column_id from the request meta and returns it.
func popInverseDisplayColumn(meta map[string]interface{}) string {
	relation, ok := meta["relation"].(map[string]interface{})
	if !ok {
		return ""
	}
	value, _ := relation[relationInverseDisplayKey].(string)
	delete(relation, relationInverseDisplayKey)
	return strings.TrimSpace(value)
}

// validateDisplayColumn checks that the column can label records of the given table.
func (s tableManagementService) validateDisplayColumn(ctx context.Context, schemaName, modelID, columnID string) error {
	col, err := s.columnsService.GetColumnByID(ctx, schemaName, columnID)
	if err != nil {
		return app_errors.InvalidDisplayColumn
	}
	if col.ModelID != modelID || !col.IsDisplayColumn() {
		return app_errors.InvalidDisplayColumn
	}
	return nil
}

// defaultDisplayColumnID returns the id of the table's system Id column, which a new link shows unless another
// column is chosen. A table without one falls back to its first text column, or "" when it has none.
func (s tableManagementService) defaultDisplayColumnID(ctx context.Context, schemaName, modelID string) string {
	columns, err := s.columnsService.GetColumnByModelID(ctx, schemaName, modelID)
	if err != nil {
		logger.Get().Warn().Err(err).Str("modelID", modelID).Msg("Could not load columns to pick a display column")
		return ""
	}
	if col, ok := tenant.IDColumn(columns); ok {
		return col.ID.String()
	}
	if col, ok := tenant.FirstTextColumn(columns); ok {
		return col.ID.String()
	}
	return ""
}

// resolveDisplayColumnID validates the requested display column for the table, or picks the default
// (the first text column) when none is requested. It returns "" when there is nothing to show.
func (s tableManagementService) resolveDisplayColumnID(ctx context.Context, schemaName, modelID, requested string) (string, error) {
	if requested == "" {
		return s.defaultDisplayColumnID(ctx, schemaName, modelID), nil
	}
	if err := s.validateDisplayColumn(ctx, schemaName, modelID, requested); err != nil {
		return "", err
	}
	return requested, nil
}

// setDisplayColumnMeta stores or clears the display column in a link column's meta.
func setDisplayColumnMeta(meta map[string]interface{}, columnID string) {
	if columnID == "" {
		delete(meta, metaDisplayColumnKey)
		return
	}
	meta[metaDisplayColumnKey] = columnID
}

// linksUsingDisplayColumn returns the link columns, on any table, whose display column is the given one.
func (s tableManagementService) linksUsingDisplayColumn(ctx context.Context, schemaName, columnID string) ([]tenant.Column, error) {
	all, err := s.columnsService.GetAllColumns(ctx, schemaName)
	if err != nil {
		return nil, err
	}
	var links []tenant.Column
	for _, col := range all {
		if strings.ToLower(col.UIDT) == uidtLinks && displayColumnID(col.Meta) == columnID {
			links = append(links, col)
		}
	}
	return links, nil
}

// ensureTypeFitsDisplayColumn rejects changing a column to a type that can't label a record while
// a link shows it.
func (s tableManagementService) ensureTypeFitsDisplayColumn(ctx context.Context, schemaName string, columnData dto.ColumnResponse, req dto.ColumnUpdate) error {
	if req.UIDT == nil || *req.UIDT == "" || *req.UIDT == columnData.UIDT || tenant.IsDisplayColumnType(*req.UIDT) {
		return nil
	}
	links, err := s.linksUsingDisplayColumn(ctx, schemaName, columnData.ID.String())
	if err != nil {
		return err
	}
	if len(links) > 0 {
		return app_errors.DisplayColumnInUse
	}
	return nil
}

// clearDisplayColumnReferences removes a deleted column from the links that showed it, so they fall
// back to the first text column. It is best effort: a failure is logged and does not undo the delete.
func (s tableManagementService) clearDisplayColumnReferences(ctx context.Context, schemaName string, columnData dto.ColumnResponse) {
	lg := logger.Get()
	links, err := s.linksUsingDisplayColumn(ctx, schemaName, columnData.ID.String())
	if err != nil {
		lg.Warn().Err(err).Str("columnID", columnData.ID.String()).Msg("Could not look up links using the deleted display column")
		return
	}
	for _, link := range links {
		meta := make(map[string]interface{}, len(link.Meta))
		for k, v := range link.Meta {
			meta[k] = v
		}
		delete(meta, metaDisplayColumnKey)
		if _, err := s.columnsService.UpdateColumn(ctx, schemaName, link.ID.String(), dto.ColumnUpdate{Meta: &meta}); err != nil {
			lg.Warn().Err(err).Str("linkID", link.ID.String()).Msg("Could not clear the display column of a link")
		}
	}
}
