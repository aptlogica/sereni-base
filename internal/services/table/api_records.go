// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/lib/pq"
)

// Record writes for the workspace public API (internal/platform). They run the same guards as the app's own
// writes (auto uuid fields, primary key, unique keys and check constraints), but write every record of a
// request in one transaction, so a request is saved completely or not at all.

// CreateRecordsForAPI inserts records (values by column name) in one transaction and returns them as stored
func (s tableManagementService) CreateRecordsForAPI(ctx context.Context, schemaName string, model tenant.Model, records []map[string]interface{}) ([]map[string]interface{}, error) {
	autoCols, err := s.autoUUIDColumns(ctx, schemaName, model.ID.String())
	if err != nil {
		return nil, err
	}
	// Every record is checked before anything is written
	for i, values := range records {
		if err := FillGeneratedUUIDs(autoCols, values); err != nil {
			return nil, &app_errors.RecordWriteError{Index: i, Err: err}
		}
		if err := s.checkKeyValues(ctx, schemaName, model, 0, values); err != nil {
			return nil, &app_errors.RecordWriteError{Index: i, Err: err}
		}
	}

	table := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)
	now := time.Now().UTC()
	return s.inAPITransaction(ctx, schemaName, model, func(tx *sql.Tx) ([]map[string]interface{}, int, error) {
		out := make([]map[string]interface{}, 0, len(records))
		for i, values := range records {
			columns := sortedColumns(values)
			names := make([]string, 0, len(columns)+2)
			args := make([]interface{}, 0, len(columns)+2)
			for _, c := range columns {
				names = append(names, quoteIdentifier(c))
				args = append(args, values[c])
			}
			names = append(names, `"created_time"`, `"last_modified_time"`)
			args = append(args, now, now)
			placeholders := make([]string, len(args))
			for j := range args {
				placeholders[j] = fmt.Sprintf("$%d", j+1)
			}
			query := fmt.Sprintf(`INSERT INTO %s AS t (%s) VALUES (%s) RETURNING row_to_json(t)::text`,
				table, strings.Join(names, ", "), strings.Join(placeholders, ", "))
			row, err := scanAPIRow(tx.QueryRowContext(ctx, query, args...))
			if err != nil {
				return nil, i, err
			}
			out = append(out, row)
		}
		return out, 0, nil
	})
}

// UpdateRecordsForAPI changes the given values of each record in one transaction and returns the records as
// stored. A record that doesn't exist fails the request with app_errors.ErrRecordNotFound.
func (s tableManagementService) UpdateRecordsForAPI(ctx context.Context, schemaName string, model tenant.Model, updates []dto.APIRecordUpdate) ([]map[string]interface{}, error) {
	for i, u := range updates {
		if err := s.checkKeyValues(ctx, schemaName, model, int(u.RowID), u.Values); err != nil {
			return nil, &app_errors.RecordWriteError{Index: i, Err: err}
		}
	}

	table := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)
	now := time.Now().UTC()
	return s.inAPITransaction(ctx, schemaName, model, func(tx *sql.Tx) ([]map[string]interface{}, int, error) {
		out := make([]map[string]interface{}, 0, len(updates))
		for i, u := range updates {
			var query string
			var args []interface{}
			if len(u.Values) == 0 {
				// Nothing to change: return the record as it is
				query = fmt.Sprintf(`SELECT row_to_json(t)::text FROM %s AS t WHERE t.id = $1`, table)
				args = []interface{}{u.RowID}
			} else {
				columns := sortedColumns(u.Values)
				sets := make([]string, 0, len(columns)+1)
				for _, c := range columns {
					args = append(args, u.Values[c])
					sets = append(sets, fmt.Sprintf("%s = $%d", quoteIdentifier(c), len(args)))
				}
				args = append(args, now)
				sets = append(sets, fmt.Sprintf(`"last_modified_time" = $%d`, len(args)))
				args = append(args, u.RowID)
				query = fmt.Sprintf(`UPDATE %s AS t SET %s WHERE t.id = $%d RETURNING row_to_json(t)::text`,
					table, strings.Join(sets, ", "), len(args))
			}
			row, err := scanAPIRow(tx.QueryRowContext(ctx, query, args...))
			if errors.Is(err, sql.ErrNoRows) {
				return nil, i, app_errors.ErrRecordNotFound
			}
			if err != nil {
				return nil, i, err
			}
			out = append(out, row)
		}
		return out, 0, nil
	})
}

// DeleteRecordsForAPI deletes records in one statement. When some of the ids don't exist, nothing is deleted
// and those ids are returned.
func (s tableManagementService) DeleteRecordsForAPI(ctx context.Context, schemaName string, model tenant.Model, ids []int64) ([]int64, error) {
	if s.repo.DB == nil {
		return nil, app_errors.DatabaseError
	}
	tx, err := s.repo.DB.Begin()
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to start transaction")
	}
	defer func() { _ = tx.Rollback() }()

	table := fmt.Sprintf(SchemaTableFormat, schemaName, model.Alias)
	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id = ANY($1) RETURNING id`, table), pq.Array(ids))
	if err != nil {
		if stopped, ok := writeStoppedError(err); ok {
			return nil, stopped
		}
		return nil, app_errors.LogDatabaseError(err, "failed to delete records")
	}
	deleted := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, app_errors.LogDatabaseError(err, "failed to read deleted records")
		}
		deleted[id] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		// An AFTER DELETE trigger runs while the rows are read
		if stopped, ok := writeStoppedError(err); ok {
			return nil, stopped
		}
		return nil, app_errors.LogDatabaseError(err, "failed to delete records")
	}

	var missing []int64
	for _, id := range ids {
		if !deleted[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		// Rolled back by the deferred call
		return missing, nil
	}
	if err := tx.Commit(); err != nil {
		// Deferred constraint triggers run at commit
		if stopped, ok := writeStoppedError(err); ok {
			return nil, stopped
		}
		return nil, app_errors.LogDatabaseError(err, "failed to commit deleted records")
	}
	return nil, nil
}

// inAPITransaction runs write in a transaction and commits it when write succeeds. A failed write is rolled
// back and its error turned into the message the app shows (key or constraint violation), with the position
// of the record that failed.
func (s tableManagementService) inAPITransaction(ctx context.Context, schemaName string, model tenant.Model,
	write func(tx *sql.Tx) ([]map[string]interface{}, int, error)) ([]map[string]interface{}, error) {
	if s.repo.DB == nil {
		return nil, app_errors.DatabaseError
	}
	tx, err := s.repo.DB.Begin()
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to start transaction")
	}
	out, index, err := write(tx)
	if err != nil {
		_ = tx.Rollback()
		if errors.Is(err, app_errors.ErrRecordNotFound) {
			return nil, &app_errors.RecordWriteError{Index: index, Err: err}
		}
		// Rolled back first: the messages below read the table's keys and constraints
		return nil, &app_errors.RecordWriteError{Index: index, Err: s.keyWriteError(ctx, schemaName, model.Alias, err, "failed to write records")}
	}
	if err := tx.Commit(); err != nil {
		if stopped, ok := writeStoppedError(err); ok {
			// Deferred triggers run at commit, so no single record can be named
			return nil, stopped
		}
		return nil, app_errors.LogDatabaseError(err, "failed to commit records")
	}
	return out, nil
}

// scanAPIRow reads a row returned as row_to_json text, keeping numbers exact
func scanAPIRow(row *sql.Row) (map[string]interface{}, error) {
	var raw string
	if err := row.Scan(&raw); err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader([]byte(raw)))
	dec.UseNumber()
	var out map[string]interface{}
	if err := dec.Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to decode record: %w", err)
	}
	return out, nil
}

// sortedColumns returns the keys of values in a fixed order, so statements don't depend on map order
func sortedColumns(values map[string]interface{}) []string {
	columns := make([]string, 0, len(values))
	for c := range values {
		columns = append(columns, c)
	}
	slices.Sort(columns)
	return columns
}
