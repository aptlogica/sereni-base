package table_test

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/aptlogica/go-postgres-rest/pkg"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	services "github.com/aptlogica/sereni-base/internal/services/table"
	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// writeFixture is a table "org" keyed by (code, yr): Code is text, Year a number, Notes a plain field
type writeFixture struct {
	*relFixture
	sql     sqlmock.Sqlmock
	modelID uuid.UUID
	code    tenant.Column
	year    tenant.Column
	notes   tenant.Column
}

func newWriteFixture(t *testing.T) *writeFixture {
	t.Helper()
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, sqlMock.ExpectationsWereMet())
		db.Close()
	})
	f := &relFixture{
		table: &MockTableService{}, bulk: &MockBulkService{}, model: &MockModelService{},
		column: &MockColumnService{}, view: &MockViewService{}, rel: &MockRelationshipService{},
	}
	f.table.On("GetByFunction", mock.Anything, "public.link_key_columns", map[string]interface{}{"p_table": `"schema"."org"`}).
		Return([]map[string]interface{}{{"link_key_columns": "{code,yr}"}}, nil).Maybe()
	repo := &pkg.DatabaseService{TableService: f.table, BulkService: f.bulk, DB: db}
	f.svc = services.NewTableManagementService("postgres", repo, f.model, f.column, f.view, f.rel, &MockAssetManagementService{})

	w := &writeFixture{relFixture: f, sql: sqlMock, modelID: uuid.New()}
	w.code = plainCol(uuid.New(), w.modelID.String(), "code", "text")
	w.code.Title = "Code"
	w.year = plainCol(uuid.New(), w.modelID.String(), "yr", "number")
	w.year.Title = "Year"
	w.notes = plainCol(uuid.New(), w.modelID.String(), "notes", "text")
	w.notes.Title = "Notes"
	model := tenant.Model{ID: w.modelID, Alias: "org"}
	f.model.On("GetModelByID", mock.Anything, relSchema, w.modelID.String()).Return(model, nil).Maybe()
	f.table.On("GetTableData", `"schema".models`, mock.Anything).Return([]map[string]interface{}{{"id": w.modelID.String(), "alias": "org"}}, nil).Maybe()
	f.column.On("GetColumnByModelID", mock.Anything, relSchema, w.modelID.String()).Return([]tenant.Column{w.code, w.year, w.notes}, nil).Maybe()
	for _, c := range []tenant.Column{w.code, w.year, w.notes} {
		f.column.On("GetColumnByID", mock.Anything, relSchema, c.ID.String()).Return(c, nil).Maybe()
	}
	return w
}

func (w *writeFixture) write(column tenant.Column, value interface{}) error {
	_, err := w.svc.InsertRowData(context.Background(), relSchema, dto.InsertRowDataRequest{
		ModelID: w.modelID.String(), ColumnId: column.ID.String(), RowId: 2, Value: &value,
	})
	return err
}

func apiError(t *testing.T, err error) *app_errors.APIError {
	t.Helper()
	var apiErr *app_errors.APIError
	require.True(t, errors.As(err, &apiErr), "want APIError, got %v", err)
	return apiErr
}

var duplicateQuery = regexp.QuoteMeta(`SELECT o.id::bigint, COALESCE(o."code"::text, ''), jsonb_build_object('code', o."code", 'yr', o."yr") FROM "schema"."org" o JOIN "schema"."org" cur ON cur.id = $2 WHERE o.id <> $2 AND o."code" = $1 AND o."yr" = cur."yr" LIMIT 1`)

func TestInsertRowData_KeyFieldChecks(t *testing.T) {
	t.Run("an empty key field is named", func(t *testing.T) {
		w := newWriteFixture(t)
		for _, value := range []interface{}{nil, "", "   "} {
			err := w.write(w.code, value)

			apiErr := apiError(t, err)
			assert.Equal(t, "PKY_1007", apiErr.Code)
			assert.Equal(t, 400, apiErr.StatusCode)
			assert.Equal(t, "Code can't be empty: it is part of the primary key (Code + Year).", apiErr.Message)
		}
		w.table.AssertNotCalled(t, "UpdateRecord", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("a used key names the values and the record holding them", func(t *testing.T) {
		w := newWriteFixture(t)
		w.sql.ExpectQuery(duplicateQuery).WithArgs("HR", 2).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "key"}).AddRow(4, "Acme", []byte(`{"code":"HR","yr":2026}`)))

		err := w.write(w.code, "HR")

		apiErr := apiError(t, err)
		assert.Equal(t, "PKY_1008", apiErr.Code)
		assert.Equal(t, 409, apiErr.StatusCode)
		assert.Equal(t, `The combination Code = "HR", Year = 2026 is already used by record #4 (Acme). Each record needs a unique primary key (Code + Year).`, apiErr.Message)
		assert.JSONEq(t, `{"fields":"Code + Year","values":{"code":"HR","yr":2026},"record_id":4}`, apiErr.Details.(string))
		w.table.AssertNotCalled(t, "UpdateRecord", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("a free key value is saved", func(t *testing.T) {
		w := newWriteFixture(t)
		w.sql.ExpectQuery(duplicateQuery).WithArgs("OPS", 2).WillReturnRows(sqlmock.NewRows([]string{"id", "title", "key"}))
		w.table.On("UpdateRecord", `"schema"."org"`, 2, mock.Anything).Return(map[string]interface{}{"id": 2, "code": "OPS"}, nil)

		require.NoError(t, w.write(w.code, "OPS"))
	})

	t.Run("other fields are not checked", func(t *testing.T) {
		w := newWriteFixture(t)
		w.table.On("UpdateRecord", `"schema"."org"`, 2, mock.Anything).Return(map[string]interface{}{"id": 2}, nil)

		require.NoError(t, w.write(w.notes, ""))
	})

	t.Run("a clash found by Postgres is explained the same way", func(t *testing.T) {
		w := newWriteFixture(t)
		w.sql.ExpectQuery(duplicateQuery).WillReturnRows(sqlmock.NewRows([]string{"id", "title", "key"}))
		// Another record took the value between the check and the write
		pgErr := &pq.Error{Code: "23505", Message: `duplicate key value violates unique constraint "org_pkey"`, Detail: "Key (code, yr)=(HR, 2026) already exists."}
		w.table.On("UpdateRecord", `"schema"."org"`, 2, mock.Anything).Return(nil, fmt.Errorf("failed to update record in table: %w", fmt.Errorf("failed to update record: %w", pgErr)))

		apiErr := apiError(t, w.write(w.code, "HR"))
		assert.Equal(t, "PKY_1008", apiErr.Code)
		assert.Equal(t, `The combination Code = "HR", Year = 2026 is already used by another record. Each record needs a unique primary key (Code + Year).`, apiErr.Message)
	})

	t.Run("a value repeated in a unique key field names the unique key, not the primary key", func(t *testing.T) {
		w := newWriteFixture(t)
		pgErr := &pq.Error{Code: "23505", Constraint: "uq_tbl_1a2b3c4d", Detail: "Key (notes)=(dup) already exists."}
		w.table.On("UpdateRecord", `"schema"."org"`, 2, mock.Anything).Return(nil, fmt.Errorf("failed to update record: %w", pgErr))
		w.table.On("GetTableData", guardConstraintsTable, mock.Anything).
			Return([]map[string]interface{}{uniqueKeyRow(w.modelID.String(), []string{w.notes.ID.String()})}, nil)

		err := w.write(w.notes, "dup")

		assert.ErrorIs(t, err, app_errors.UniqueKeyViolation)
		assert.Equal(t, `Duplicate value not allowed. The unique key "Unique email" does not allow two records with the same Notes.`, detailedMessage(t, err))
	})

	t.Run("a unique key added outside the app is told apart by its columns", func(t *testing.T) {
		w := newWriteFixture(t)
		pgErr := &pq.Error{Code: "23505", Constraint: "org_notes_key", Detail: "Key (notes)=(dup) already exists."}
		w.table.On("UpdateRecord", `"schema"."org"`, 2, mock.Anything).Return(nil, pgErr)
		w.table.On("GetTableData", guardConstraintsTable, mock.Anything).Return([]map[string]interface{}{}, nil)

		err := w.write(w.notes, "dup")

		assert.ErrorIs(t, err, app_errors.UniqueKeyViolation)
		assert.Equal(t, "Duplicate value not allowed. Two records can't have the same notes.", detailedMessage(t, err))
	})

	t.Run("a named primary key constraint is still a primary key error", func(t *testing.T) {
		w := newWriteFixture(t)
		w.sql.ExpectQuery(duplicateQuery).WillReturnRows(sqlmock.NewRows([]string{"id", "title", "key"}))
		pgErr := &pq.Error{Code: "23505", Constraint: "org_pkey", Detail: "Key (code, yr)=(HR, 2026) already exists."}
		w.table.On("UpdateRecord", `"schema"."org"`, 2, mock.Anything).Return(nil, pgErr)
		w.table.On("GetTableData", guardConstraintsTable, mock.Anything).Return([]map[string]interface{}{}, nil)

		assert.Equal(t, "PKY_1008", apiError(t, w.write(w.code, "HR")).Code)
	})

	t.Run("other database errors stay generic", func(t *testing.T) {
		w := newWriteFixture(t)
		w.table.On("UpdateRecord", `"schema"."org"`, 2, mock.Anything).Return(nil, errors.New("connection reset"))

		assert.ErrorIs(t, w.write(w.notes, "x"), app_errors.DatabaseError)
	})
}

type rowValuesService interface {
	UpsertRowValues(ctx context.Context, schemaName string, modelID string, rowID int, values map[string]interface{}, user string) (dto.RecordResponse, error)
	UpdateRowValues(ctx context.Context, schemaName string, modelID string, rowID int, values map[string]interface{}, updatedBy string) (dto.RecordResponse, error)
	CreateRowsWithValues(ctx context.Context, schemaName string, modelID string, rowsInput []map[string]interface{}, createdBy string, updatedBy string) ([]dto.RecordResponse, error)
}

func TestUpdateRowValues_SavesTheWholeRecordAtOnce(t *testing.T) {
	t.Run("both key fields are checked together and saved in one statement", func(t *testing.T) {
		w := newWriteFixture(t)
		// The new Code and Year together, not Code with the old Year
		w.sql.ExpectQuery(regexp.QuoteMeta(`WHERE o.id <> $3 AND o."code" = $1 AND o."yr" = $2 LIMIT 1`)).WithArgs("HR", 2027.0, 2).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "key"}))
		var saved map[string]interface{}
		w.table.On("UpdateRecord", `"schema"."org"`, 2, mock.Anything).
			Run(func(args mock.Arguments) { saved = args.Get(2).(map[string]interface{}) }).
			Return(map[string]interface{}{"id": 2}, nil).Once()

		_, err := w.svc.(rowValuesService).UpdateRowValues(context.Background(), relSchema, w.modelID.String(), 2,
			map[string]interface{}{w.code.ID.String(): "HR", w.year.ID.String(): 2027.0, w.notes.ID.String(): "n"}, "u1")

		require.NoError(t, err)
		assert.Equal(t, "HR", saved[`"code"`])
		assert.Equal(t, 2027.0, saved[`"yr"`])
		assert.Equal(t, "n", saved[`"notes"`])
		assert.Equal(t, "u1", saved["last_modified_by"])
	})

	t.Run("nothing is saved when one field is rejected", func(t *testing.T) {
		w := newWriteFixture(t)

		_, err := w.svc.(rowValuesService).UpdateRowValues(context.Background(), relSchema, w.modelID.String(), 2,
			map[string]interface{}{w.notes.ID.String(): "n", w.year.ID.String(): nil}, "u1")

		assert.Equal(t, "Year can't be empty: it is part of the primary key (Code + Year).", apiError(t, err).Message)
		w.table.AssertNotCalled(t, "UpdateRecord", mock.Anything, mock.Anything, mock.Anything)
	})
}

var insertDuplicateQuery = regexp.QuoteMeta(`SELECT o.id::bigint, COALESCE(o."code"::text, ''), jsonb_build_object('code', o."code", 'yr', o."yr") FROM "schema"."org" o WHERE o."code" = $1 AND o."yr" = $2 LIMIT 1`)

func (w *writeFixture) insert(rows ...map[string]interface{}) ([]dto.RecordResponse, error) {
	return w.svc.(rowValuesService).CreateRowsWithValues(context.Background(), relSchema, w.modelID.String(), rows, "u1", "u1")
}

func TestCreateRowsWithValues_NeedsEveryKeyField(t *testing.T) {
	t.Run("missing key fields are named and nothing is created", func(t *testing.T) {
		w := newWriteFixture(t)

		rows, err := w.insert(map[string]interface{}{w.notes.ID.String(): "n"})

		assert.Nil(t, rows)
		apiErr := apiError(t, err)
		assert.Equal(t, "PKY_1007", apiErr.Code)
		assert.Equal(t, "Code and Year can't be empty: they are part of the primary key (Code + Year).", apiErr.Message)
		w.table.AssertNotCalled(t, "CreateRecord", mock.Anything, mock.Anything)
	})

	t.Run("one missing key field", func(t *testing.T) {
		w := newWriteFixture(t)

		_, err := w.insert(map[string]interface{}{w.code.ID.String(): "HR", w.year.ID.String(): ""})

		assert.Equal(t, "Year can't be empty: it is part of the primary key (Code + Year).", apiError(t, err).Message)
	})

	t.Run("a used key is rejected before inserting", func(t *testing.T) {
		w := newWriteFixture(t)
		w.sql.ExpectQuery(insertDuplicateQuery).WithArgs("HR", 2026.0).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "key"}).AddRow(4, "Acme", []byte(`{"code":"HR","yr":2026}`)))

		_, err := w.insert(map[string]interface{}{w.code.ID.String(): "HR", w.year.ID.String(): 2026.0})

		apiErr := apiError(t, err)
		assert.Equal(t, "PKY_1008", apiErr.Code)
		assert.Equal(t, `The combination Code = "HR", Year = 2026 is already used by record #4 (Acme). Each record needs a unique primary key (Code + Year).`, apiErr.Message)
		w.table.AssertNotCalled(t, "CreateRecord", mock.Anything, mock.Anything)
	})

	t.Run("a complete record is inserted with its values in one statement", func(t *testing.T) {
		w := newWriteFixture(t)
		w.sql.ExpectQuery(insertDuplicateQuery).WithArgs("HR", 2026.0).WillReturnRows(sqlmock.NewRows([]string{"id", "title", "key"}))
		var saved map[string]interface{}
		w.table.On("CreateRecord", `"schema"."org"`, mock.Anything).
			Run(func(args mock.Arguments) { saved = args.Get(1).(map[string]interface{}) }).
			Return(map[string]interface{}{"id": int64(9)}, nil).Once()

		rows, err := w.insert(map[string]interface{}{w.code.ID.String(): "HR", w.year.ID.String(): 2026.0, w.notes.ID.String(): "n"})

		require.NoError(t, err)
		assert.Len(t, rows, 1)
		assert.Equal(t, "HR", saved[`"code"`])
		assert.Equal(t, 2026.0, saved[`"yr"`])
		assert.Equal(t, "n", saved[`"notes"`])
		assert.Equal(t, "u1", saved["created_by"])
		w.table.AssertNotCalled(t, "UpdateRecord", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("when a later record fails the earlier ones are removed", func(t *testing.T) {
		w := newWriteFixture(t)
		w.sql.ExpectQuery(insertDuplicateQuery).WithArgs("HR", 2026.0).WillReturnRows(sqlmock.NewRows([]string{"id", "title", "key"}))
		w.table.On("CreateRecord", `"schema"."org"`, mock.Anything).Return(map[string]interface{}{"id": int64(9)}, nil).Once()
		w.table.On("DeleteRecord", `"schema"."org"`, 9).Return(nil).Once()

		rows, err := w.insert(
			map[string]interface{}{w.code.ID.String(): "HR", w.year.ID.String(): 2026.0},
			map[string]interface{}{w.code.ID.String(): "FIN"},
		)

		assert.Nil(t, rows)
		assert.Equal(t, "Year can't be empty: it is part of the primary key (Code + Year).", apiError(t, err).Message)
		w.table.AssertCalled(t, "DeleteRecord", `"schema"."org"`, 9)
	})
}

// With id in the key (Id + Code) a new record only needs Code, and can't clash since its id is new
func TestCreateRowsWithValues_IdInTheKey(t *testing.T) {
	w := newWriteFixture(t)
	w.table.ExpectedCalls = nil
	w.table.On("GetByFunction", mock.Anything, "public.link_key_columns", mock.Anything).
		Return([]map[string]interface{}{{"link_key_columns": "{id,code}"}}, nil)
	w.table.On("GetTableData", mock.Anything, mock.Anything).Return([]map[string]interface{}{}, nil).Maybe()

	_, err := w.insert(map[string]interface{}{w.notes.ID.String(): "n"})
	// The fixture has no Id column, so its title falls back to the column name
	assert.Equal(t, "Code can't be empty: it is part of the primary key (id + Code).", apiError(t, err).Message)

	w.table.On("CreateRecord", `"schema"."org"`, mock.Anything).Return(map[string]interface{}{"id": int64(9)}, nil).Once()
	rows, err := w.insert(map[string]interface{}{w.code.ID.String(): "HR"})
	require.NoError(t, err)
	assert.Len(t, rows, 1)
}

func TestUpsertRowValues(t *testing.T) {
	t.Run("without a row id the record is created", func(t *testing.T) {
		w := newWriteFixture(t)
		w.sql.ExpectQuery(insertDuplicateQuery).WithArgs("HR", 2026.0).WillReturnRows(sqlmock.NewRows([]string{"id", "title", "key"}))
		w.table.On("CreateRecord", `"schema"."org"`, mock.Anything).Return(map[string]interface{}{"id": int64(9)}, nil).Once()

		out, err := w.svc.(rowValuesService).UpsertRowValues(context.Background(), relSchema, w.modelID.String(), 0,
			map[string]interface{}{w.code.ID.String(): "HR", w.year.ID.String(): 2026.0}, "u1")

		require.NoError(t, err)
		assert.Equal(t, int64(9), out.Record["id"])
	})

	t.Run("without a row id the key fields are required", func(t *testing.T) {
		w := newWriteFixture(t)

		_, err := w.svc.(rowValuesService).UpsertRowValues(context.Background(), relSchema, w.modelID.String(), 0,
			map[string]interface{}{w.code.ID.String(): "HR"}, "u1")

		assert.Equal(t, "Year can't be empty: it is part of the primary key (Code + Year).", apiError(t, err).Message)
	})

	t.Run("with a row id the record is updated", func(t *testing.T) {
		w := newWriteFixture(t)
		w.table.On("UpdateRecord", `"schema"."org"`, 4, mock.Anything).Return(map[string]interface{}{"id": 4}, nil).Once()

		_, err := w.svc.(rowValuesService).UpsertRowValues(context.Background(), relSchema, w.modelID.String(), 4,
			map[string]interface{}{w.notes.ID.String(): "n"}, "u1")

		require.NoError(t, err)
		w.table.AssertNotCalled(t, "CreateRecord", mock.Anything, mock.Anything)
	})
}

func TestCreateRow_EmptyRecordNeedsTheKeyFields(t *testing.T) {
	w := newWriteFixture(t)

	_, err := w.svc.CreateRow(context.Background(), relSchema, dto.CreateRowRequest{ModelID: w.modelID.String(), CreatedBy: "u1"})

	assert.Equal(t, "Code and Year can't be empty: they are part of the primary key (Code + Year).", apiError(t, err).Message)
	w.table.AssertNotCalled(t, "CreateRecord", mock.Anything, mock.Anything)
}

func TestBulkWrites_KeyViolationsAreExplained(t *testing.T) {
	t.Run("bulk update that empties a key field", func(t *testing.T) {
		w := newWriteFixture(t)
		violation := app_errors.DatabaseOrKeyError(&pq.Error{Code: "23502", Column: "yr", Message: `null value in column "yr"`}, "bulk")
		w.column.On("BulkUpdate", mock.Anything, relSchema, "org", "yr", mock.Anything).Return(violation)

		err := w.svc.BulkUpdateColumns(context.Background(), relSchema, w.modelID.String(), w.year.ID.String(), []dto.UpdateColumnsRequest{{}})

		apiErr := apiError(t, err)
		assert.Equal(t, "PKY_1007", apiErr.Code)
		assert.Equal(t, "Year can't be empty: it is part of the primary key (Code + Year).", apiErr.Message)
	})

	t.Run("reset of a key field", func(t *testing.T) {
		w := newWriteFixture(t)
		violation := app_errors.DatabaseOrKeyError(&pq.Error{Code: "23502", Column: "code"}, "reset")
		w.column.On("ResetColumn", mock.Anything, relSchema, "org", "code").Return(violation)

		apiErr := apiError(t, w.svc.ResetColumnValues(context.Background(), relSchema, w.modelID.String(), w.code.ID.String()))
		assert.Equal(t, "Code can't be empty: it is part of the primary key (Code + Year).", apiErr.Message)
	})

	t.Run("cleaning tool that makes two records equal", func(t *testing.T) {
		w := newWriteFixture(t)
		violation := app_errors.DatabaseOrKeyError(&pq.Error{Code: "23505", Detail: "Key (code, yr)=(HR, 2026) already exists."}, "bulk")
		w.column.On("BulkUpdate", mock.Anything, relSchema, "org", "code", mock.Anything).Return(violation)

		apiErr := apiError(t, w.svc.BulkUpdateColumns(context.Background(), relSchema, w.modelID.String(), w.code.ID.String(), []dto.UpdateColumnsRequest{{}}))
		assert.Equal(t, "PKY_1008", apiErr.Code)
		assert.Equal(t, `The combination Code = "HR", Year = 2026 is already used by another record. Each record needs a unique primary key (Code + Year).`, apiErr.Message)
	})

	t.Run("values containing commas fall back to the key fields", func(t *testing.T) {
		w := newWriteFixture(t)
		violation := app_errors.DatabaseOrKeyError(&pq.Error{Code: "23505", Detail: "Key (code, yr)=(A, B, 2026) already exists."}, "bulk")
		w.column.On("BulkUpdate", mock.Anything, relSchema, "org", "code", mock.Anything).Return(violation)

		apiErr := apiError(t, w.svc.BulkUpdateColumns(context.Background(), relSchema, w.modelID.String(), w.code.ID.String(), []dto.UpdateColumnsRequest{{}}))
		assert.Equal(t, "This primary key value is already used by another record. Each record needs a unique primary key (Code + Year).", apiErr.Message)
	})
}

func TestDatabaseOrKeyError(t *testing.T) {
	wrapped := fmt.Errorf("call: %w", &pq.Error{Code: "23505"})
	err := app_errors.DatabaseOrKeyError(wrapped, "x")
	var violation *app_errors.KeyViolation
	assert.True(t, errors.As(err, &violation))
	assert.ErrorIs(t, err, app_errors.DatabaseError, "still a database error for existing checks")

	assert.Equal(t, app_errors.DatabaseError, app_errors.DatabaseOrKeyError(&pq.Error{Code: "42P01"}, "x"))
	assert.Equal(t, app_errors.DatabaseError, app_errors.DatabaseOrKeyError(errors.New("boom"), "x"))
}
