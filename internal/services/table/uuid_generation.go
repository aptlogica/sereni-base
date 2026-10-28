// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"errors"
	"fmt"
	"slices"

	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"

	"github.com/lib/pq"
)

// AutoUUIDColumn is a uuid field in auto mode, ready for generation
type AutoUUIDColumn struct {
	ColumnName string
	Version    string
}

// ValidateIDGeneration checks meta.idGeneration of a field and returns the settings with defaults filled in.
// Only uuid fields may have it; an auto field gets v7 when no version is named.
func ValidateIDGeneration(uidt string, meta map[string]interface{}) (dto.IDGeneration, error) {
	_, present := meta[constant.MetaIDGenerationKey]
	cfg, ok := dto.IDGenerationFromMeta(meta)
	if !ok || (present && uidt != constant.UIDTUUID) {
		return dto.IDGeneration{}, app_errors.UUIDConfigInvalid
	}
	switch cfg.Mode {
	case constant.IDModeManual:
		cfg.Version = ""
	case constant.IDModeAuto:
		if cfg.Version == "" {
			cfg.Version = constant.UUIDVersionDefault
		}
		if !slices.Contains(constant.UUIDVersionsGeneratable, cfg.Version) {
			return dto.IDGeneration{}, app_errors.UUIDConfigInvalid
		}
		// The value is owned by the system, so a fixed default would clash with it
		if _, hasDefault := meta[metaDefaultValueKey]; hasDefault {
			return dto.IDGeneration{}, app_errors.UUIDConfigInvalid
		}
	default:
		return dto.IDGeneration{}, app_errors.UUIDConfigInvalid
	}
	return cfg, nil
}

// autoUUIDColumnsOf picks the auto uuid fields out of a table's columns
func autoUUIDColumnsOf(columns []dto.ColumnResponse) []AutoUUIDColumn {
	var cols []AutoUUIDColumn
	for _, c := range columns {
		if c.UIDT != constant.UIDTUUID {
			continue
		}
		if cfg, ok := dto.IDGenerationFromMeta(c.Meta); ok && cfg.Mode == constant.IDModeAuto {
			version := cfg.Version
			if version == "" {
				version = constant.UUIDVersionDefault
			}
			cols = append(cols, AutoUUIDColumn{ColumnName: c.ColumnName, Version: version})
		}
	}
	return cols
}

// autoUUIDColumns returns the auto uuid fields of a table
func (s tableManagementService) autoUUIDColumns(ctx context.Context, schemaName, modelID string) ([]AutoUUIDColumn, error) {
	columns, err := s.GetColumnsByModelID(ctx, schemaName, modelID)
	if err != nil {
		return nil, err
	}
	return autoUUIDColumnsOf(columns), nil
}

// autoUUIDColumnsByAlias returns the auto uuid fields of the table with this alias. A table without a model
// record has no field settings, so nothing is generated for it.
func (s tableManagementService) autoUUIDColumnsByAlias(ctx context.Context, schemaName, alias string) ([]AutoUUIDColumn, error) {
	model, err := s.modelByAlias(schemaName, alias)
	if errors.Is(err, app_errors.TableNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.autoUUIDColumns(ctx, schemaName, model.ID.String())
}

// FillGeneratedUUIDs sets a new value for every auto uuid field of a record about to be created. The record is
// keyed by column name. A value already given for an auto field is refused, since the field is read-only.
func FillGeneratedUUIDs(cols []AutoUUIDColumn, record map[string]interface{}) error {
	for _, col := range cols {
		if v, given := record[col.ColumnName]; given && v != nil && v != "" {
			return app_errors.UUIDReadOnly
		}
		u, err := NewUUID(col.Version)
		if err != nil {
			return err
		}
		record[col.ColumnName] = u.String()
	}
	return nil
}

// backfillUUIDs gives every record with an empty value in an auto uuid field a new value, in batches.
// It returns how many records were filled.
func (s tableManagementService) backfillUUIDs(ctx context.Context, schemaName, alias string, col AutoUUIDColumn) (int, error) {
	tableName := fmt.Sprintf(SchemaTableFormat, schemaName, alias)
	limit := constant.UUIDBackfillBatchSize
	filled := 0
	for {
		rows, err := s.repo.TableService.GetTableData(tableName, dbModels.QueryParams{
			Select:  []string{"id"},
			Filters: []dbModels.QueryFilter{{Column: col.ColumnName, Operator: "is_null"}},
			Limit:   &limit,
		})
		if err != nil {
			return filled, app_errors.LogDatabaseError(err, "failed to read records with an empty uuid")
		}
		if len(rows) == 0 {
			return filled, nil
		}

		ids := make([]int64, 0, len(rows))
		values := make([]string, 0, len(rows))
		for _, row := range rows {
			id, err := ExtractCreatedRowID(row)
			if err != nil {
				return filled, err
			}
			u, err := NewUUID(col.Version)
			if err != nil {
				return filled, err
			}
			ids = append(ids, int64(id))
			values = append(values, u.String())
		}
		// One statement per batch; values are parameters, the column name is quoted
		query := fmt.Sprintf(`UPDATE %[1]s AS t SET %[2]s = v.value::uuid
			FROM (SELECT unnest($1::bigint[]) AS id, unnest($2::text[]) AS value) AS v
			WHERE t.id = v.id AND t.%[2]s IS NULL`, tableName, pq.QuoteIdentifier(col.ColumnName))
		if _, err := s.repo.DB.ExecContext(ctx, query, pq.Array(ids), pq.Array(values)); err != nil {
			return filled, app_errors.LogDatabaseError(err, "failed to fill empty uuid values")
		}
		filled += len(ids)
		// A short batch was the last one; this also stops a loop if the update didn't take
		if len(rows) < limit {
			return filled, nil
		}
	}
}

// CountEmptyCells returns how many records of the column's table have an empty and a filled value in it
func (s tableManagementService) CountEmptyCells(ctx context.Context, schemaName, columnID string) (dto.EmptyCountResponse, error) {
	column, err := s.GetColumnById(ctx, schemaName, columnID)
	if err != nil {
		return dto.EmptyCountResponse{}, err
	}
	model, err := s.modelService.GetModelByID(ctx, schemaName, column.ModelID.String())
	if err != nil {
		return dto.EmptyCountResponse{}, err
	}

	query := fmt.Sprintf(`SELECT COUNT(*) FILTER (WHERE %[1]s IS NULL), COUNT(%[1]s) FROM %[2]s`,
		pq.QuoteIdentifier(column.ColumnName), fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias))
	rows, err := s.repo.DB.QueryContext(ctx, query)
	if err != nil {
		return dto.EmptyCountResponse{}, app_errors.LogDatabaseError(err, "failed to count empty cells")
	}
	defer rows.Close()

	var res dto.EmptyCountResponse
	if rows.Next() {
		if err := rows.Scan(&res.Empty, &res.Filled); err != nil {
			return dto.EmptyCountResponse{}, app_errors.LogDatabaseError(err, "failed to read empty cells count")
		}
	}
	return res, nil
}
