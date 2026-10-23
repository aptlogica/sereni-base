// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"fmt"
	"slices"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/lib/pq"
)

const (
	keyColumnsFunctionName     = "link_key_columns"
	installKeySyncFunctionName = "link_install_key_sync"
)

// ensureNotPrimaryKeyColumn fails when the column is part of its table's primary key; such a field
// has to be removed from the key in table settings before it is deleted or its type changes.
func (s tableManagementService) ensureNotPrimaryKeyColumn(ctx context.Context, schemaName, modelID, columnName string) error {
	model, err := s.modelService.GetModelByID(ctx, schemaName, modelID)
	if err != nil {
		return err
	}
	keyCols, err := s.primaryKeyColumns(ctx, schemaName, model.Alias)
	if err != nil {
		return err
	}
	if slices.Contains(keyCols, columnName) {
		return app_errors.ColumnInPrimaryKey
	}
	return nil
}

// installKeySync adds the triggers that keep link fields pointing at this table in step with its keys
func (s tableManagementService) installKeySync(ctx context.Context, schemaName, tableName string) error {
	_, err := s.repo.TableService.GetByFunction(ctx, fmt.Sprintf("%s.%s", constant.MasterDatabase, installKeySyncFunctionName),
		map[string]interface{}{"p_schema": schemaName, "p_table": tableName})
	if err != nil {
		return app_errors.LogDatabaseError(err, "failed to install link key triggers")
	}
	return nil
}

// textArray reads a Postgres text[] value as returned by the driver
func textArray(value interface{}) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, fmt.Sprint(item))
		}
		return out
	case []byte, string:
		var out pq.StringArray
		if err := out.Scan(v); err != nil {
			return nil
		}
		return out
	}
	return nil
}
