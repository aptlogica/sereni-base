// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/aptlogica/go-postgres-rest/pkg"
	"github.com/lib/pq"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/providers/logger"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
)

// primaryKeySampleLimit caps the offending records returned when a key change is rejected
const primaryKeySampleLimit = 10

const duplicateKeyReadFailed = "failed to read duplicate key values"

// System columns that may be key fields; the audit columns are filled in by the system
var primaryKeySystemColumns = []string{constant.PrimaryKeyIDColumn, "title"}

type primaryKeyService struct {
	repo          *pkg.DatabaseService
	modelService  interfaces.ModelService
	columnService interfaces.ColumnService
}

func NewPrimaryKeyService(repo *pkg.DatabaseService, modelService interfaces.ModelService, columnService interfaces.ColumnService) interfaces.PrimaryKeyService {
	return &primaryKeyService{repo: repo, modelService: modelService, columnService: columnService}
}

// keyTable is everything a primary key change needs to know about the table
type keyTable struct {
	model   tenant.Model
	table   string // quoted "schema"."alias"
	columns []tenant.Column
	keyCols []string // current key column names, in key order
	links   []string // titles of the link fields
}

// Get returns the primary key of a table and which fields can be key fields
func (s *primaryKeyService) Get(ctx context.Context, schemaName, modelID string) (dto.PrimaryKeyResponse, error) {
	t, err := s.load(ctx, schemaName, modelID)
	if err != nil {
		return dto.PrimaryKeyResponse{}, err
	}
	return t.response(), nil
}

// Update replaces the primary key with the given columns, in order
func (s *primaryKeyService) Update(ctx context.Context, schemaName, modelID string, req dto.UpdatePrimaryKeyRequest) (dto.PrimaryKeyResponse, error) {
	if len(req.ColumnIDs) == 0 {
		return dto.PrimaryKeyResponse{}, app_errors.PrimaryKeyRequired
	}
	t, err := s.load(ctx, schemaName, modelID)
	if err != nil {
		return dto.PrimaryKeyResponse{}, err
	}

	newCols, err := t.resolve(req.ColumnIDs)
	if err != nil {
		return dto.PrimaryKeyResponse{}, err
	}
	if slices.Equal(newCols, t.keyCols) {
		return t.response(), nil
	}
	// Link fields store this table's key values, so a different key would break them
	if len(t.links) > 0 {
		return dto.PrimaryKeyResponse{}, fmt.Errorf("%w: The primary key can't be changed while this table has link fields (%s). "+
			"Link fields store the primary key values of the records they link to; remove them first.",
			app_errors.PrimaryKeyLocked, strings.Join(t.links, ", "))
	}

	if err := s.validateData(t, newCols); err != nil {
		return dto.PrimaryKeyResponse{}, err
	}
	if err := s.replaceKey(t, newCols); err != nil {
		return dto.PrimaryKeyResponse{}, err
	}

	t.keyCols = newCols
	return t.response(), nil
}

// EnsureColumnNotInKey fails when the column is part of its table's primary key
func (s *primaryKeyService) EnsureColumnNotInKey(ctx context.Context, schemaName, modelID, columnName string) error {
	model, err := s.modelService.GetModelByID(ctx, schemaName, modelID)
	if err != nil {
		return err
	}
	keyCols, err := PrimaryKeyColumns(s.repo, quotedTable(schemaName, model.Alias))
	if err != nil {
		return err
	}
	if slices.Contains(keyCols, columnName) {
		return app_errors.ColumnInPrimaryKey
	}
	return nil
}

// PrimaryKeyColumns returns the primary key column names of a quoted "schema"."table", in key order
func PrimaryKeyColumns(repo *pkg.DatabaseService, table string) ([]string, error) {
	rows, err := repo.DB.Query(constant.PrimaryKeyColumnsQuery, table)
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to read primary key")
	}
	defer rows.Close()
	cols := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, app_errors.LogDatabaseError(err, "failed to read primary key")
		}
		cols = append(cols, name)
	}
	return cols, rows.Err()
}

func (s *primaryKeyService) load(ctx context.Context, schemaName, modelID string) (keyTable, error) {
	model, err := s.modelService.GetModelByID(ctx, schemaName, modelID)
	if err != nil {
		return keyTable{}, err
	}
	columns, err := s.columnService.GetColumnByModelID(ctx, schemaName, modelID)
	if err != nil {
		return keyTable{}, err
	}
	t := keyTable{model: model, table: quotedTable(schemaName, model.Alias)}
	for _, col := range columns {
		if col.Deleted {
			continue
		}
		t.columns = append(t.columns, col)
		if col.UIDT == constant.UIDTLinks {
			t.links = append(t.links, col.Title)
		}
	}
	if t.keyCols, err = PrimaryKeyColumns(s.repo, t.table); err != nil {
		return keyTable{}, err
	}
	return t, nil
}

// resolve maps the requested column IDs to column names, rejecting unknown, repeated or ineligible fields
func (t keyTable) resolve(columnIDs []string) ([]string, error) {
	names := make([]string, 0, len(columnIDs))
	for _, id := range columnIDs {
		col, ok := t.columnByID(strings.TrimSpace(id))
		if !ok {
			return nil, fmt.Errorf("%w: A selected field (%s) no longer exists in this table. Refresh the page and try again.",
				app_errors.PrimaryKeyColumnInvalid, strings.TrimSpace(id))
		}
		if slices.Contains(names, col.ColumnName) {
			return nil, fmt.Errorf("%w: %s is selected more than once.", app_errors.PrimaryKeyColumnInvalid, col.Title)
		}
		if eligible, reason := keyEligibility(col); !eligible {
			return nil, fmt.Errorf("%w: %s can't be part of the primary key: %s.",
				app_errors.PrimaryKeyColumnInvalid, col.Title, strings.ToLower(reason[:1])+reason[1:])
		}
		names = append(names, col.ColumnName)
	}
	return names, nil
}

func (t keyTable) columnByID(id string) (tenant.Column, bool) {
	for _, col := range t.columns {
		if col.ID.String() == id {
			return col, true
		}
	}
	return tenant.Column{}, false
}

// title is the field title of a column name, or the name itself
func (t keyTable) title(columnName string) string {
	for _, col := range t.columns {
		if col.ColumnName == columnName && col.Title != "" {
			return col.Title
		}
	}
	return columnName
}

func (t keyTable) titles(columnNames []string) []string {
	out := make([]string, len(columnNames))
	for i, c := range columnNames {
		out[i] = t.title(c)
	}
	return out
}

func (t keyTable) response() dto.PrimaryKeyResponse {
	out := dto.PrimaryKeyResponse{
		ModelID:     t.model.ID.String(),
		ColumnIDs:   []string{},
		ColumnNames: t.keyCols,
		Locked:      len(t.links) > 0,
		LinkColumns: t.links,
		Columns:     []dto.PrimaryKeyColumn{},
	}
	if out.LinkColumns == nil {
		out.LinkColumns = []string{}
	}
	for _, name := range t.keyCols {
		for _, col := range t.columns {
			if col.ColumnName == name {
				out.ColumnIDs = append(out.ColumnIDs, col.ID.String())
			}
		}
	}
	for _, col := range t.columns {
		eligible, reason := keyEligibility(col)
		out.Columns = append(out.Columns, dto.PrimaryKeyColumn{
			ID:         col.ID.String(),
			Title:      col.Title,
			ColumnName: col.ColumnName,
			UIDT:       col.UIDT,
			System:     col.System,
			InKey:      slices.Contains(t.keyCols, col.ColumnName),
			Eligible:   eligible,
			Reason:     reason,
		})
	}
	return out
}

// keyEligibility reports whether a field may be a key field and, if not, why
func keyEligibility(col tenant.Column) (bool, string) {
	// System columns are stored with virtual = true, but id and title are real columns
	if col.System {
		if slices.Contains(primaryKeySystemColumns, col.ColumnName) {
			return true, ""
		}
		return false, "Filled in automatically by the system"
	}
	switch {
	case col.Virtual || col.UIDT == constant.UIDTLinks:
		return false, "Computed from other records"
	case !slices.Contains(constant.PrimaryKeyEligibleUIDTs, col.UIDT):
		return false, "Holds multiple or long values"
	}
	return true, ""
}

// validateData checks that every record has the key fields filled in and their combination is unique.
// On failure the error names each empty field and duplicated combination with the records concerned.
func (s *primaryKeyService) validateData(t keyTable, cols []string) error {
	quoted := make([]string, len(cols))
	nullChecks := make([]string, len(cols))
	pairs := make([]string, len(cols))
	perColumn := make([]string, len(cols))
	missing := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = quoteIdent(c)
		nullChecks[i] = quoted[i] + " IS NULL"
		literal := "'" + strings.ReplaceAll(c, "'", "''") + "'"
		pairs[i] = literal + ", " + quoted[i]
		perColumn[i] = fmt.Sprintf("count(*) FILTER (WHERE %s IS NULL)", quoted[i])
		missing[i] = fmt.Sprintf("CASE WHEN %s IS NULL THEN %s END", quoted[i], literal)
	}
	anyNull := strings.Join(nullChecks, " OR ")
	keyObject := "jsonb_build_object(" + strings.Join(pairs, ", ") + ")"

	details := dto.PrimaryKeyViolationDetails{
		Columns:      cols,
		ColumnTitles: t.titles(cols),
		EmptyFields:  []dto.PrimaryKeyEmptyField{},
		NullRecords:  []dto.PrimaryKeyRecord{},
		Duplicates:   []dto.PrimaryKeyDuplicate{},
	}

	// Per-column empty counts, records with any empty key field, and duplicated combinations
	counts := make([]int64, len(cols))
	targets := make([]interface{}, 0, len(cols)+2)
	for i := range counts {
		targets = append(targets, &counts[i])
	}
	targets = append(targets, &details.NullCount, &details.DuplicateGroups)
	if err := s.repo.DB.QueryRow(fmt.Sprintf(
		`SELECT %s, count(*) FILTER (WHERE %s), (SELECT count(*) FROM (SELECT 1 FROM %s WHERE NOT (%s) GROUP BY %s HAVING count(*) > 1) d) FROM %s`,
		strings.Join(perColumn, ", "), anyNull, t.table, anyNull, strings.Join(quoted, ", "), t.table,
	)).Scan(targets...); err != nil {
		return app_errors.LogDatabaseError(err, "failed to validate primary key data")
	}
	if details.NullCount == 0 && details.DuplicateGroups == 0 {
		return nil
	}

	for i, c := range cols {
		if counts[i] > 0 {
			details.EmptyFields = append(details.EmptyFields, dto.PrimaryKeyEmptyField{Column: c, Title: t.title(c), Count: counts[i]})
		}
	}
	if details.NullCount > 0 {
		records, err := s.nullRecords(t, anyNull, strings.Join(missing, ", "))
		if err != nil {
			return err
		}
		details.NullRecords = records
	}
	if details.DuplicateGroups > 0 {
		duplicates, err := s.duplicates(t.table, keyObject, anyNull)
		if err != nil {
			return err
		}
		details.Duplicates = duplicates
	}
	details.Message = violationMessage(details)
	return &app_errors.PrimaryKeyViolation{Message: details.Message, Details: details}
}

func (s *primaryKeyService) nullRecords(t keyTable, anyNull, missing string) ([]dto.PrimaryKeyRecord, error) {
	rows, err := s.repo.DB.Query(fmt.Sprintf(
		`SELECT id::bigint, COALESCE(title::text, ''), array_remove(ARRAY[%s], NULL)::text[] FROM %s WHERE %s ORDER BY id LIMIT %d`,
		missing, t.table, anyNull, primaryKeySampleLimit))
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, "failed to read records without key values")
	}
	defer rows.Close()
	records := []dto.PrimaryKeyRecord{}
	for rows.Next() {
		var r dto.PrimaryKeyRecord
		var cols pq.StringArray
		if err := rows.Scan(&r.ID, &r.Title, &cols); err != nil {
			return nil, app_errors.LogDatabaseError(err, "failed to read records without key values")
		}
		r.Missing = t.titles(cols)
		records = append(records, r)
	}
	return records, rows.Err()
}

func (s *primaryKeyService) duplicates(table, keyObject, anyNull string) ([]dto.PrimaryKeyDuplicate, error) {
	rows, err := s.repo.DB.Query(fmt.Sprintf(
		`SELECT %s AS k, count(*), (array_agg(id::bigint ORDER BY id))[1:%d] FROM %s WHERE NOT (%s) GROUP BY 1 HAVING count(*) > 1 ORDER BY 2 DESC, 1::text LIMIT %d`,
		keyObject, primaryKeySampleLimit, table, anyNull, primaryKeySampleLimit))
	if err != nil {
		return nil, app_errors.LogDatabaseError(err, duplicateKeyReadFailed)
	}
	defer rows.Close()
	duplicates := []dto.PrimaryKeyDuplicate{}
	for rows.Next() {
		var raw []byte
		var ids pq.Int64Array
		var d dto.PrimaryKeyDuplicate
		if err := rows.Scan(&raw, &d.Count, &ids); err != nil {
			return nil, app_errors.LogDatabaseError(err, duplicateKeyReadFailed)
		}
		if err := json.Unmarshal(raw, &d.Values); err != nil {
			return nil, app_errors.LogDatabaseError(err, duplicateKeyReadFailed)
		}
		d.RecordIDs = ids
		duplicates = append(duplicates, d)
	}
	return duplicates, rows.Err()
}

// violationMessage spells out every problem found, e.g. Can't set the primary key to Code + Region.
// Code is empty in 1 record (#3 Fin). The combination Code = "HR", Region = "IN" is used by 2 records (#1, #4).
func violationMessage(d dto.PrimaryKeyViolationDetails) string {
	parts := []string{fmt.Sprintf("Can't set the primary key to %s.", strings.Join(d.ColumnTitles, " + "))}

	for _, field := range d.EmptyFields {
		var examples []string
		for _, r := range d.NullRecords {
			if slices.Contains(r.Missing, field.Title) {
				examples = append(examples, recordLabel(r))
			}
		}
		parts = append(parts, fmt.Sprintf("%s is empty in %s%s.", field.Title, plural(field.Count, "record", "records"), recordList(examples, field.Count)))
	}

	for _, dup := range d.Duplicates {
		values := make([]string, len(d.Columns))
		for i, c := range d.Columns {
			values[i] = fmt.Sprintf("%s = %s", d.ColumnTitles[i], formatKeyValue(dup.Values[c]))
		}
		ids := make([]string, len(dup.RecordIDs))
		for i, id := range dup.RecordIDs {
			ids[i] = fmt.Sprintf("#%d", id)
		}
		subject := "The value"
		if len(d.Columns) > 1 {
			subject = "The combination"
		}
		parts = append(parts, fmt.Sprintf("%s %s is used by %s%s.", subject, strings.Join(values, ", "), plural(dup.Count, "record", "records"), recordList(ids, dup.Count)))
	}
	if more := d.DuplicateGroups - int64(len(d.Duplicates)); more > 0 {
		parts = append(parts, fmt.Sprintf("%s also duplicated.", plural(more, "more value is", "more values are")))
	}

	parts = append(parts, "Fill in or change these records, then save again.")
	return strings.Join(parts, " ")
}

// recordList renders " (#3 Fin, #9 and 4 more)" for the sampled records out of total
func recordList(items []string, total int64) string {
	if len(items) == 0 {
		return ""
	}
	shown := items
	if len(shown) > 5 {
		shown = shown[:5]
	}
	list := strings.Join(shown, ", ")
	if rest := total - int64(len(shown)); rest > 0 {
		list += fmt.Sprintf(" and %d more", rest)
	}
	return " (" + list + ")"
}

func recordLabel(r dto.PrimaryKeyRecord) string {
	if r.Title == "" {
		return fmt.Sprintf("#%d", r.ID)
	}
	return fmt.Sprintf("#%d %s", r.ID, r.Title)
}

func plural(n int64, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

func formatKeyValue(v interface{}) string {
	switch val := v.(type) {
	case nil:
		return "(empty)"
	case string:
		return fmt.Sprintf("%q", val)
	default:
		return fmt.Sprint(val)
	}
}

// replaceKey swaps the primary key in one transaction. id stays NOT NULL and UNIQUE when it leaves the key,
// because rows are still addressed by id.
func (s *primaryKeyService) replaceKey(t keyTable, newCols []string) error {
	keyName, idUnique, err := s.constraints(t.table)
	if err != nil {
		return err
	}

	statements := []string{}
	if keyName != "" {
		statements = append(statements, fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s", t.table, quoteIdent(keyName)))
	}
	idInKey := slices.Contains(newCols, constant.PrimaryKeyIDColumn)
	switch {
	case !idInKey && idUnique == "":
		statements = append(statements, fmt.Sprintf("ALTER TABLE %s ADD UNIQUE (%s)", t.table, quoteIdent(constant.PrimaryKeyIDColumn)))
	case idInKey && idUnique != "" && len(newCols) == 1:
		// A single-column key on id already makes it unique
		statements = append(statements, fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s", t.table, quoteIdent(idUnique)))
	}
	statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL", t.table, quoteIdent(constant.PrimaryKeyIDColumn)))
	// Fields leaving the key may be blank again
	for _, c := range t.keyCols {
		if c != constant.PrimaryKeyIDColumn && !slices.Contains(newCols, c) {
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL", t.table, quoteIdent(c)))
		}
	}
	quoted := make([]string, len(newCols))
	for i, c := range newCols {
		quoted[i] = quoteIdent(c)
	}
	statements = append(statements, fmt.Sprintf("ALTER TABLE %s ADD PRIMARY KEY (%s)", t.table, strings.Join(quoted, ", ")))

	tx, err := s.repo.DB.Begin()
	if err != nil {
		return fmt.Errorf("%w: %v", app_errors.PrimaryKeyUpdateFailed, err)
	}
	for _, stmt := range statements {
		if _, err := tx.Exec(stmt); err != nil {
			_ = tx.Rollback()
			logger.Get().Error().Err(err).Str("statement", stmt).Msg("failed to update primary key")
			return fmt.Errorf("%w: %s", app_errors.PrimaryKeyUpdateFailed, postgresMessage(t, err))
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: %v", app_errors.PrimaryKeyUpdateFailed, err)
	}
	return nil
}

// postgresMessage turns a Postgres error into an exact message, e.g. for records changed since validation:
// Duplicate key values: Key (code, region)=(HR, IN) is duplicated.
func postgresMessage(t keyTable, err error) string {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return err.Error()
	}
	message := pqErr.Message
	switch pqErr.Code {
	case "23505": // unique_violation
		message = "Duplicate key values"
	case "23502": // not_null_violation
		if pqErr.Column != "" {
			message = fmt.Sprintf("%s is empty in some records", t.title(pqErr.Column))
		}
	}
	if pqErr.Detail != "" {
		message += ": " + pqErr.Detail
	}
	return message
}

// constraints returns the primary key constraint name and the name of a UNIQUE constraint on id alone
func (s *primaryKeyService) constraints(table string) (keyName, idUnique string, err error) {
	var key, unique sql.NullString
	if err := s.repo.DB.QueryRow(constant.PrimaryKeyConstraintsQuery, table).Scan(&key, &unique); err != nil {
		return "", "", app_errors.LogDatabaseError(err, "failed to read table constraints")
	}
	return key.String, unique.String, nil
}

func quotedTable(schemaName, tableName string) string {
	return quoteIdent(schemaName) + "." + quoteIdent(tableName)
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
