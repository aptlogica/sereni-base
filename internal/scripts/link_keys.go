// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package scripts

import (
	"fmt"
	"strings"

	"github.com/aptlogica/go-postgres-rest/pkg"
	appConstant "github.com/aptlogica/sereni-base/internal/constant"
)

// linkColumnToMigrate is a link column still holding row ids (INT / INT[])
type linkColumnToMigrate struct {
	ID          string
	ColumnName  string
	Table       string
	TargetTable string
	IsArray     bool
}

// MigrateLinkKeys converts link columns that still store row ids into JSONB arrays of the linked rows'
// primary-key values, then installs the key-sync triggers on every table. Safe to run on every start:
// converted columns and tables that already have the triggers are skipped.
func MigrateLinkKeys(repo *pkg.DatabaseService, schemaName string) error {
	columns, err := linkColumnsToMigrate(repo, schemaName)
	if err != nil {
		return err
	}
	for _, col := range columns {
		if err := migrateLinkColumn(repo, schemaName, col); err != nil {
			// One broken column must not keep the others from being migrated
			fmt.Printf("error migrating link column %s.%s: %v\n", col.Table, col.ColumnName, err)
		}
	}

	if _, err := repo.DB.Exec(fmt.Sprintf(appConstant.InstallKeySyncTriggersQuery, quoteIdent(schemaName), schemaName)); err != nil {
		return fmt.Errorf("install link key triggers: %w", err)
	}
	return nil
}

func linkColumnsToMigrate(repo *pkg.DatabaseService, schemaName string) ([]linkColumnToMigrate, error) {
	rows, err := repo.DB.Query(fmt.Sprintf(appConstant.LinkColumnsToMigrateQuery, quoteIdent(schemaName), schemaName))
	if err != nil {
		return nil, fmt.Errorf("list link columns: %w", err)
	}
	defer rows.Close()
	var columns []linkColumnToMigrate
	for rows.Next() {
		var col linkColumnToMigrate
		if err := rows.Scan(&col.ID, &col.ColumnName, &col.Table, &col.TargetTable, &col.IsArray); err != nil {
			return nil, fmt.Errorf("list link columns: %w", err)
		}
		columns = append(columns, col)
	}
	return columns, rows.Err()
}

// migrateLinkColumn rewrites one column and its metadata together
func migrateLinkColumn(repo *pkg.DatabaseService, schemaName string, col linkColumnToMigrate) error {
	column := quoteIdent(col.ColumnName)
	ids := fmt.Sprintf("CASE WHEN %[1]s IS NULL THEN NULL ELSE ARRAY[%[1]s::bigint] END", column)
	if col.IsArray {
		ids = column + "::bigint[]"
	}
	using := "NULL"
	if col.TargetTable != "" {
		target := quoteIdent(schemaName) + "." + quoteIdent(col.TargetTable)
		using = fmt.Sprintf("public.link_ids_to_keys('%s', %s)", escapeLiteral(target), ids)
	}

	tx, err := repo.DB.Begin()
	if err != nil {
		return err
	}
	statements := []string{
		fmt.Sprintf("ALTER TABLE %s.%s ALTER COLUMN %s TYPE JSONB USING %s", quoteIdent(schemaName), quoteIdent(col.Table), column, using),
		fmt.Sprintf("UPDATE %s.columns SET dt = 'JSONB' WHERE id::text = '%s'", quoteIdent(schemaName), escapeLiteral(col.ID)),
	}
	for _, stmt := range statements {
		if _, err := tx.Exec(stmt); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func escapeLiteral(value string) string {
	return strings.ReplaceAll(value, `'`, `''`)
}
