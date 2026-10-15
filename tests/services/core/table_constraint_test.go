package core_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/aptlogica/go-postgres-rest/pkg"
	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	services "github.com/aptlogica/sereni-base/internal/services/core"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// constraintColumnService returns the columns of a table
type constraintColumnService struct {
	interfaces.ColumnService
	mock.Mock
}

func (m *constraintColumnService) GetColumnByModelID(ctx context.Context, schemaName, modelID string) ([]tenant.Column, error) {
	args := m.Called(schemaName, modelID)
	return args.Get(0).([]tenant.Column), args.Error(1)
}

const (
	constraintSchema = "tenant_1"
	constraintID     = "0b6f3a52-1c4d-4e8f-9a7b-2d5e6f708192"
	constraintPGName = "uq_employee_0b6f3a52"
	employeeTable    = `"tenant_1"."employee"`
)

var (
	constraintsTable = tenant.TableConstraint{}.TableName(constraintSchema)

	emailCol = tenant.Column{ID: uuid.MustParse("11111111-0000-0000-0000-000000000001"), ColumnName: "email", Title: "Email", UIDT: "email"}
	codeCol  = tenant.Column{ID: uuid.MustParse("11111111-0000-0000-0000-000000000002"), ColumnName: "emp_code", Title: "Emp Code", UIDT: "text"}
	titleCol = tenant.Column{ID: uuid.MustParse("11111111-0000-0000-0000-000000000003"), ColumnName: "title", Title: "Title", UIDT: "text", System: true}
	idCol    = tenant.Column{ID: uuid.MustParse("11111111-0000-0000-0000-000000000004"), ColumnName: "id", Title: "Id", UIDT: "number", System: true}
	linkCol  = tenant.Column{ID: uuid.MustParse("11111111-0000-0000-0000-000000000005"), ColumnName: "dept", Title: "Dept", UIDT: "links", Virtual: true}
	goneCol  = tenant.Column{ID: uuid.MustParse("11111111-0000-0000-0000-000000000006"), ColumnName: "old", Title: "Old", UIDT: "text", Deleted: true}
)

type constraintFixture struct {
	svc     interfaces.TableConstraintService
	table   *MockTableService
	columns *constraintColumnService
	sql     sqlmock.Sqlmock
}

func newConstraintFixture(t *testing.T) *constraintFixture {
	t.Helper()
	return newConstraintFixtureFor(t, "employee")
}

// newConstraintFixtureFor builds the service for table m1 with the given alias
func newConstraintFixtureFor(t *testing.T, alias string) *constraintFixture {
	t.Helper()
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, sqlMock.ExpectationsWereMet())
		db.Close()
	})

	table := &MockTableService{}
	models := &automationModelService{}
	models.On("GetModelByID", constraintSchema, "m1").Return(tenant.Model{Alias: alias}, nil).Maybe()
	columns := &constraintColumnService{}
	columns.On("GetColumnByModelID", constraintSchema, "m1").
		Return([]tenant.Column{idCol, titleCol, emailCol, codeCol, linkCol, goneCol}, nil).Maybe()

	repo := &pkg.DatabaseService{TableService: table, DB: db}
	return &constraintFixture{svc: services.NewTableConstraintService(repo, models, columns), table: table, columns: columns, sql: sqlMock}
}

// savedKey is a stored unique key as the database layer returns it (jsonb already parsed)
func savedKey(id string, columnIDs []string) map[string]interface{} {
	ids := make([]interface{}, len(columnIDs))
	for i, c := range columnIDs {
		ids[i] = c
	}
	return map[string]interface{}{
		"id": id, "model_id": "m1", "type": "unique", "title": "Unique email", "constraint_name": constraintPGName,
		"column_ids": ids,
	}
}

func byModel(p dbModels.QueryParams) bool {
	return len(p.Filters) == 1 && p.Filters[0].Column == "model_id"
}

func byID(p dbModels.QueryParams) bool {
	return len(p.Filters) == 1 && p.Filters[0].Column == "id" && p.Filters[0].Value == constraintID
}

func (f *constraintFixture) expectNoOtherKeys() {
	f.table.On("GetTableData", constraintsTable, mock.MatchedBy(byModel)).Return([]map[string]interface{}{}, nil).Once()
}

// expectDuplicates answers the duplicate check
func (f *constraintFixture) expectDuplicates(found bool) {
	f.sql.ExpectQuery(`^SELECT EXISTS \(SELECT 1 FROM "tenant_1"\.`).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(found))
}

func (f *constraintFixture) expectNoDuplicates() { f.expectDuplicates(false) }

func TestTableConstraintService_Check(t *testing.T) {
	t.Run("names the column whose values repeat", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.sql.ExpectQuery(`^SELECT EXISTS \(SELECT 1 FROM "tenant_1"\."employee" WHERE "email" IS NOT NULL GROUP BY "email" HAVING count\(\*\) > 1\)$`).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		result, err := f.svc.Check(context.Background(), constraintSchema, dto.CheckTableConstraintRequest{ModelID: "m1", ColumnIDs: []string{emailCol.ID.String()}})

		require.NoError(t, err)
		assert.Equal(t, dto.ConstraintCheckResult{Valid: false, Message: `Duplicate data in column "Email". Remove the duplicates and try again.`}, result)
	})

	t.Run("names all columns of a combined key", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.sql.ExpectQuery(`WHERE "email" IS NOT NULL AND "emp_code" IS NOT NULL GROUP BY "email", "emp_code"`).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		result, err := f.svc.Check(context.Background(), constraintSchema, dto.CheckTableConstraintRequest{
			ModelID: "m1", ColumnIDs: []string{emailCol.ID.String(), codeCol.ID.String()},
		})

		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Equal(t, `Duplicate data in columns "Email" + "Emp Code": some records have the same combination of values. Remove the duplicates and try again.`, result.Message)
	})

	t.Run("is valid without a message when nothing repeats", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.expectNoDuplicates()

		result, err := f.svc.Check(context.Background(), constraintSchema, dto.CheckTableConstraintRequest{ModelID: "m1", ColumnIDs: []string{emailCol.ID.String()}})
		require.NoError(t, err)
		assert.Equal(t, dto.ConstraintCheckResult{Valid: true}, result)
	})

	t.Run("returns the Postgres message when the check fails", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.sql.ExpectQuery("FROM").WillReturnError(&pq.Error{Message: `relation "employee" does not exist`})

		_, err := f.svc.Check(context.Background(), constraintSchema, dto.CheckTableConstraintRequest{ModelID: "m1", ColumnIDs: []string{emailCol.ID.String()}})
		assert.ErrorIs(t, err, app_errors.ConstraintFailed)
		var detailed *app_errors.DetailedError
		require.ErrorAs(t, err, &detailed)
		assert.Equal(t, `relation "employee" does not exist`, detailed.Message)
	})

	columnErrors := []struct {
		name    string
		ids     []string
		want    error
		message string
	}{
		{"unknown column", []string{uuid.NewString()}, app_errors.ConstraintColumnNotFound, ""},
		{"deleted column", []string{goneCol.ID.String()}, app_errors.ConstraintColumnNotFound, ""},
		{"repeated column", []string{emailCol.ID.String(), emailCol.ID.String()}, app_errors.ConstraintRepeatedColumn, ""},
		{"link column", []string{linkCol.ID.String()}, app_errors.ConstraintColumnNotAllowed, "Dept can't be part of a key: link fields can't be part of a key"},
		{"id column", []string{idCol.ID.String()}, app_errors.ConstraintColumnNotAllowed, "Id can't be part of a key: it is already unique"},
	}
	for _, tc := range columnErrors {
		t.Run("rejects "+tc.name, func(t *testing.T) {
			f := newConstraintFixture(t)
			_, err := f.svc.Check(context.Background(), constraintSchema, dto.CheckTableConstraintRequest{ModelID: "m1", ColumnIDs: tc.ids})
			assert.ErrorIs(t, err, tc.want)
			if tc.message != "" {
				var detailed *app_errors.DetailedError
				require.ErrorAs(t, err, &detailed)
				assert.Equal(t, tc.message, detailed.Message)
			}
		})
	}

	t.Run("allows the title column", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.expectNoDuplicates()
		_, err := f.svc.Check(context.Background(), constraintSchema, dto.CheckTableConstraintRequest{ModelID: "m1", ColumnIDs: []string{titleCol.ID.String()}})
		assert.NoError(t, err)
	})
}

func TestTableConstraintService_Create(t *testing.T) {
	request := dto.CreateTableConstraintRequest{
		ModelID: "m1", Type: "unique", Title: "Dept code", ColumnIDs: []string{codeCol.ID.String(), emailCol.ID.String()},
		CreatedBy: "user-1",
	}

	t.Run("adds the unique key in Postgres and stores it", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.expectNoOtherKeys()
		f.expectNoDuplicates()
		f.sql.ExpectBegin()
		f.sql.ExpectExec(`^ALTER TABLE "tenant_1"\."employee" ADD CONSTRAINT "uq_employee_[0-9a-f]{8}" UNIQUE \("emp_code", "email"\)$`).
			WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectCommit()

		var stored map[string]interface{}
		f.table.On("CreateRecord", constraintsTable, mock.Anything).
			Run(func(args mock.Arguments) { stored = args.Get(1).(map[string]interface{}) }).
			Return(savedKey(constraintID, request.ColumnIDs), nil)

		out, err := f.svc.Create(context.Background(), constraintSchema, request)

		require.NoError(t, err)
		assert.Regexp(t, `^uq_employee_[0-9a-f]{8}$`, stored["constraint_name"])
		assert.Equal(t, `["`+codeCol.ID.String()+`","`+emailCol.ID.String()+`"]`, stored["column_ids"])
		assert.NotContains(t, stored, "options")
		assert.NotContains(t, stored, "error_message")
		assert.Equal(t, "user-1", stored["created_by"])

		assert.Equal(t, constraintID, out.ID)
		assert.Equal(t, "ok", out.Status)
		require.Len(t, out.Columns, 2)
		assert.Equal(t, "Emp Code", out.Columns[0].Title)
		assert.Equal(t, dto.ConstraintColumn{ID: emailCol.ID.String(), Title: "Email"}, out.Columns[1])
	})

	t.Run("rejects a key with the same columns in another order", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.table.On("GetTableData", constraintsTable, mock.MatchedBy(byModel)).
			Return([]map[string]interface{}{savedKey(uuid.NewString(), []string{emailCol.ID.String(), codeCol.ID.String()})}, nil)

		_, err := f.svc.Create(context.Background(), constraintSchema, request)
		assert.ErrorIs(t, err, app_errors.ConstraintAlreadyExists)
		f.table.AssertNotCalled(t, "CreateRecord", mock.Anything, mock.Anything)
	})

	t.Run("refuses while records repeat the values", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.expectNoOtherKeys()
		f.expectDuplicates(true)

		_, err := f.svc.Create(context.Background(), constraintSchema, request)
		assert.ErrorIs(t, err, app_errors.ConstraintDuplicateValues)
		assert.Contains(t, err.Error(), `Duplicate data in columns "Emp Code" + "Email"`)
	})

	t.Run("reports duplicates added since the check", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.expectNoOtherKeys()
		f.expectNoDuplicates()
		f.sql.ExpectBegin()
		f.sql.ExpectExec("ADD CONSTRAINT").WillReturnError(&pq.Error{Code: "23505", Message: "could not create unique index"})
		f.sql.ExpectRollback()

		_, err := f.svc.Create(context.Background(), constraintSchema, request)
		assert.ErrorIs(t, err, app_errors.ConstraintDuplicateValues)
	})

	t.Run("returns other Postgres errors", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.expectNoOtherKeys()
		f.expectNoDuplicates()
		f.sql.ExpectBegin()
		f.sql.ExpectExec("ADD CONSTRAINT").WillReturnError(&pq.Error{Code: "54000", Message: "index row size exceeds maximum"})
		f.sql.ExpectRollback()

		_, err := f.svc.Create(context.Background(), constraintSchema, request)
		assert.ErrorIs(t, err, app_errors.ConstraintFailed)
		assert.Contains(t, err.Error(), "index row size exceeds maximum")
	})

	t.Run("passes on Postgres' own column limit", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.expectNoOtherKeys()
		f.expectNoDuplicates()
		f.sql.ExpectBegin()
		f.sql.ExpectExec("ADD CONSTRAINT").WillReturnError(&pq.Error{Code: "54011", Message: "cannot use more than 32 columns in an index"})
		f.sql.ExpectRollback()

		_, err := f.svc.Create(context.Background(), constraintSchema, request)
		assert.ErrorIs(t, err, app_errors.ConstraintFailed)
		var detailed *app_errors.DetailedError
		require.ErrorAs(t, err, &detailed)
		assert.Equal(t, "cannot use more than 32 columns in an index", detailed.Message)
	})

	t.Run("drops the key again when storing it fails", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.expectNoOtherKeys()
		f.expectNoDuplicates()
		f.sql.ExpectBegin()
		f.sql.ExpectExec("ADD CONSTRAINT").WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectCommit()
		f.sql.ExpectExec(`^ALTER TABLE "tenant_1"\."employee" DROP CONSTRAINT IF EXISTS "uq_employee_[0-9a-f]{8}"$`).WillReturnResult(sqlmock.NewResult(0, 0))
		f.table.On("CreateRecord", constraintsTable, mock.Anything).Return(nil, errors.New("db down"))

		_, err := f.svc.Create(context.Background(), constraintSchema, request)
		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("shortens long table names to fit Postgres' 63 characters", func(t *testing.T) {
		long := "a_very_long_table_alias_that_goes_on_and_on_and_on_and_on"
		f := newConstraintFixtureFor(t, long)
		f.expectNoOtherKeys()
		f.expectNoDuplicates()
		f.sql.ExpectBegin()
		f.sql.ExpectExec("ADD CONSTRAINT").WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectCommit()

		name := ""
		f.table.On("CreateRecord", constraintsTable, mock.Anything).
			Run(func(args mock.Arguments) { name = args.Get(1).(map[string]interface{})["constraint_name"].(string) }).
			Return(savedKey(constraintID, []string{emailCol.ID.String()}), nil)

		_, err := f.svc.Create(context.Background(), constraintSchema, dto.CreateTableConstraintRequest{
			ModelID: "m1", Type: "unique", Title: "x", ColumnIDs: []string{emailCol.ID.String()},
		})
		require.NoError(t, err)
		assert.LessOrEqual(t, len(name), 63)
		assert.Regexp(t, `^uq_a_very_long_table_alias_that_goes_on_and_on_and_on_[0-9a-f]{8}$`, name)
	})
}

func TestTableConstraintService_Update(t *testing.T) {
	t.Run("replaces the key in Postgres when the columns change", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.table.On("GetTableData", constraintsTable, mock.MatchedBy(byID)).Return([]map[string]interface{}{savedKey(constraintID, []string{emailCol.ID.String()})}, nil)
		f.table.On("GetTableData", constraintsTable, mock.MatchedBy(byModel)).Return([]map[string]interface{}{savedKey(constraintID, []string{emailCol.ID.String()})}, nil)
		f.expectNoDuplicates()
		f.sql.ExpectBegin()
		f.sql.ExpectExec(`^ALTER TABLE "tenant_1"\."employee" DROP CONSTRAINT IF EXISTS "` + constraintPGName + `"$`).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectExec(`^ALTER TABLE "tenant_1"\."employee" ADD CONSTRAINT "` + constraintPGName + `" UNIQUE \("email", "emp_code"\)$`).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectCommit()

		var saved map[string]interface{}
		f.table.On("UpdateRecord", constraintsTable, constraintID, mock.Anything).
			Run(func(args mock.Arguments) { saved = args.Get(2).(map[string]interface{}) }).
			Return(savedKey(constraintID, []string{emailCol.ID.String(), codeCol.ID.String()}), nil)

		out, err := f.svc.Update(context.Background(), constraintSchema, constraintID, dto.UpdateTableConstraintRequest{
			Title: "Email + code", ColumnIDs: []string{emailCol.ID.String(), codeCol.ID.String()}, UpdatedBy: "user-2",
		})

		require.NoError(t, err)
		assert.Equal(t, "Email + code", saved["title"])
		assert.Equal(t, "user-2", saved["last_modified_by"])
		assert.Len(t, out.Columns, 2)
	})

	t.Run("only saves the name and message when the key is the same", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.table.On("GetTableData", constraintsTable, mock.MatchedBy(byID)).Return([]map[string]interface{}{savedKey(constraintID, []string{emailCol.ID.String()})}, nil)
		f.table.On("UpdateRecord", constraintsTable, constraintID, mock.Anything).Return(savedKey(constraintID, []string{emailCol.ID.String()}), nil)

		_, err := f.svc.Update(context.Background(), constraintSchema, constraintID, dto.UpdateTableConstraintRequest{
			Title: "Renamed", ColumnIDs: []string{emailCol.ID.String()},
		})
		require.NoError(t, err)
	})

	t.Run("returns not found for an unknown id", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.table.On("GetTableData", constraintsTable, mock.Anything).Return([]map[string]interface{}{}, nil)

		_, err := f.svc.Update(context.Background(), constraintSchema, constraintID, dto.UpdateTableConstraintRequest{Title: "x", ColumnIDs: []string{emailCol.ID.String()}})
		assert.ErrorIs(t, err, app_errors.ConstraintNotFound)
	})
}

func TestTableConstraintService_Delete(t *testing.T) {
	t.Run("drops the key in Postgres and removes it", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.table.On("GetTableData", constraintsTable, mock.MatchedBy(byID)).Return([]map[string]interface{}{savedKey(constraintID, []string{emailCol.ID.String()})}, nil)
		f.sql.ExpectBegin()
		f.sql.ExpectExec(`DROP CONSTRAINT IF EXISTS "` + constraintPGName + `"`).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectCommit()
		f.table.On("DeleteRecord", constraintsTable, constraintID).Return(nil)

		require.NoError(t, f.svc.Delete(context.Background(), constraintSchema, constraintID))
	})

	t.Run("keeps the entry when Postgres can't drop the key", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.table.On("GetTableData", constraintsTable, mock.MatchedBy(byID)).Return([]map[string]interface{}{savedKey(constraintID, []string{emailCol.ID.String()})}, nil)
		f.sql.ExpectBegin()
		f.sql.ExpectExec("DROP CONSTRAINT").WillReturnError(&pq.Error{Message: "constraint is referenced by a foreign key"})
		f.sql.ExpectRollback()

		err := f.svc.Delete(context.Background(), constraintSchema, constraintID)
		assert.ErrorIs(t, err, app_errors.ConstraintFailed)
		f.table.AssertNotCalled(t, "DeleteRecord", mock.Anything, mock.Anything)
	})

	t.Run("returns not found for an unknown id", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.table.On("GetTableData", constraintsTable, mock.Anything).Return([]map[string]interface{}{}, nil)
		assert.ErrorIs(t, f.svc.Delete(context.Background(), constraintSchema, constraintID), app_errors.ConstraintNotFound)
	})
}

func TestTableConstraintService_List(t *testing.T) {
	t.Run("returns keys with their columns and whether Postgres still has them", func(t *testing.T) {
		f := newConstraintFixture(t)
		missing := savedKey(uuid.NewString(), []string{codeCol.ID.String(), goneCol.ID.String()})
		missing["constraint_name"] = "uq_employee_deadbeef"
		f.table.On("GetTableData", constraintsTable, mock.MatchedBy(func(p dbModels.QueryParams) bool {
			return len(p.Filters) == 2 && p.Filters[1].Column == "type" && p.Filters[1].Value == "unique" && p.OrderBy[0] == "created_time DESC"
		})).Return([]map[string]interface{}{savedKey(constraintID, []string{emailCol.ID.String()}), missing}, nil)
		f.sql.ExpectQuery(`FROM pg_constraint WHERE conrelid = \$1::regclass`).WithArgs(employeeTable).
			WillReturnRows(sqlmock.NewRows([]string{"conname"}).AddRow(constraintPGName).AddRow("employee_pkey"))

		out, err := f.svc.List(context.Background(), constraintSchema, "m1", "unique")

		require.NoError(t, err)
		require.Len(t, out, 2)
		assert.Equal(t, "ok", out[0].Status)
		assert.Equal(t, "Email", out[0].Columns[0].Title)
		assert.Equal(t, "missing", out[1].Status)
		assert.Equal(t, "Deleted column", out[1].Columns[1].Title, "deleted columns are named as such")
	})

	t.Run("returns an empty list without asking Postgres", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.table.On("GetTableData", constraintsTable, mock.Anything).Return([]map[string]interface{}{}, nil)

		out, err := f.svc.List(context.Background(), constraintSchema, "m1", "unique")
		require.NoError(t, err)
		assert.Empty(t, out)
		assert.NotNil(t, out)
	})

	t.Run("fails when reading fails", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.table.On("GetTableData", constraintsTable, mock.Anything).Return(nil, errors.New("db down"))

		_, err := f.svc.List(context.Background(), constraintSchema, "m1", "unique")
		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})
}

// withDefaults gives the fixture's table columns a default value
func (f *constraintFixture) withDefaults(cols ...tenant.Column) {
	f.columns.ExpectedCalls = nil
	f.columns.On("GetColumnByModelID", constraintSchema, "m1").
		Return(append([]tenant.Column{idCol, titleCol, linkCol, goneCol}, cols...), nil).Maybe()
}

func withDefault(c tenant.Column, value interface{}) tenant.Column {
	c.Meta = map[string]interface{}{"defaultValue": value}
	return c
}

func assertDefaultConflict(t *testing.T, err error) {
	t.Helper()
	assert.ErrorIs(t, err, app_errors.UniqueKeyDefaultConflict)
	var detailed *app_errors.DetailedError
	require.ErrorAs(t, err, &detailed)
	assert.Equal(t, tenant.UniqueDefaultConflictMessage, detailed.Message)
}

func TestTableConstraintService_DefaultValueConflict(t *testing.T) {
	t.Run("create rejects a key whose only column has a default", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.withDefaults(withDefault(emailCol, "a@b.c"), codeCol)

		_, err := f.svc.Create(context.Background(), constraintSchema, dto.CreateTableConstraintRequest{
			ModelID: "m1", Type: "unique", Title: "x", ColumnIDs: []string{emailCol.ID.String()},
		})
		assertDefaultConflict(t, err)
	})

	t.Run("create rejects a combined key when every column has a default", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.withDefaults(withDefault(emailCol, "a@b.c"), withDefault(codeCol, 5.0))

		_, err := f.svc.Create(context.Background(), constraintSchema, dto.CreateTableConstraintRequest{
			ModelID: "m1", Type: "unique", Title: "x", ColumnIDs: []string{emailCol.ID.String(), codeCol.ID.String()},
		})
		assertDefaultConflict(t, err)
	})

	t.Run("create allows a combined key when one column has no default", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.withDefaults(withDefault(emailCol, "a@b.c"), codeCol)
		f.expectNoOtherKeys()
		f.expectNoDuplicates()
		f.sql.ExpectBegin()
		f.sql.ExpectExec(`ADD CONSTRAINT`).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectCommit()
		f.table.On("CreateRecord", constraintsTable, mock.Anything).
			Return(savedKey(constraintID, []string{emailCol.ID.String(), codeCol.ID.String()}), nil)

		_, err := f.svc.Create(context.Background(), constraintSchema, dto.CreateTableConstraintRequest{
			ModelID: "m1", Type: "unique", Title: "x", ColumnIDs: []string{emailCol.ID.String(), codeCol.ID.String()},
		})
		require.NoError(t, err)
	})

	t.Run("create ignores an empty default", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.withDefaults(withDefault(emailCol, "  "), codeCol)
		f.expectNoOtherKeys()
		f.expectNoDuplicates()
		f.sql.ExpectBegin()
		f.sql.ExpectExec(`ADD CONSTRAINT`).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectCommit()
		f.table.On("CreateRecord", constraintsTable, mock.Anything).
			Return(savedKey(constraintID, []string{emailCol.ID.String()}), nil)

		_, err := f.svc.Create(context.Background(), constraintSchema, dto.CreateTableConstraintRequest{
			ModelID: "m1", Type: "unique", Title: "x", ColumnIDs: []string{emailCol.ID.String()},
		})
		require.NoError(t, err)
	})

	t.Run("update rejects changing the key to a column with a default", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.withDefaults(emailCol, withDefault(codeCol, "X"))
		f.table.On("GetTableData", constraintsTable, mock.MatchedBy(byID)).
			Return([]map[string]interface{}{savedKey(constraintID, []string{emailCol.ID.String()})}, nil)

		_, err := f.svc.Update(context.Background(), constraintSchema, constraintID, dto.UpdateTableConstraintRequest{
			Title: "Code", ColumnIDs: []string{codeCol.ID.String()}, UpdatedBy: "user-2",
		})
		assertDefaultConflict(t, err)
	})
}
