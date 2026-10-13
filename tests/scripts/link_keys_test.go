package scripts_test

import (
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/aptlogica/go-postgres-rest/pkg"
	appConstant "github.com/aptlogica/sereni-base/internal/constant"
	"github.com/aptlogica/sereni-base/internal/scripts"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var linkColumnFields = []string{"id", "column_name", "alias", "target_alias", "is_array"}

func newSQLMockDB(t *testing.T) (*pkg.DatabaseService, *MockTableService, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, sqlMock.ExpectationsWereMet())
		sqlDB.Close()
	})
	table := &MockTableService{}
	return &pkg.DatabaseService{TableService: table, DB: sqlDB}, table, sqlMock
}

func ok() driver.Result { return sqlmock.NewResult(0, 0) }

// expectTriggerInstall expects the key-sync triggers to be installed and, for the given CREATE TRIGGER
// statements, the key defaults trigger on tables with a custom key
func expectTriggerInstall(sqlMock sqlmock.Sqlmock, keyDefaults ...string) {
	sqlMock.ExpectExec(`SELECT public\.link_install_key_sync\('public', m\.alias\)`).WillReturnResult(ok())
	rows := sqlmock.NewRows([]string{"format"})
	for _, stmt := range keyDefaults {
		rows.AddRow(stmt)
	}
	sqlMock.ExpectQuery(`link_key_columns\(format\('%I\.%I', 'public', m\.alias\)\) <> ARRAY\['id'\]`).WillReturnRows(rows)
	for _, stmt := range keyDefaults {
		sqlMock.ExpectExec("^" + regexp.QuoteMeta(stmt) + "$").WillReturnResult(ok())
	}
}

// TestCreateFunctionsAndMigration checks startup drops superseded signatures, installs every function with
// CREATE OR REPLACE (so changed bodies reach existing databases) and then runs the link key migration.
func TestCreateFunctionsAndMigration(t *testing.T) {
	db, table, sqlMock := newSQLMockDB(t)
	table.On("CreateSchema", mock.Anything, mock.Anything).Return(nil)
	table.On("CreateTable", mock.Anything).Return(nil)

	for _, signature := range appConstant.DroppedFunctions {
		sqlMock.ExpectExec("^" + regexp.QuoteMeta(`DROP FUNCTION IF EXISTS "public".`+signature) + "$").WillReturnResult(ok())
	}
	for i, fn := range appConstant.DefinedFunctions {
		create := sqlMock.ExpectExec("^" + regexp.QuoteMeta(`CREATE OR REPLACE FUNCTION "public".`+fn.FunctionName+"("+fn.FunctionParams+")"))
		if i == 0 {
			// A changed return type is dropped and created again
			create.WillReturnError(errors.New("cannot change return type of existing function"))
			sqlMock.ExpectExec("^" + regexp.QuoteMeta(`DROP FUNCTION IF EXISTS "public".`+fn.FunctionName+"(") + ".*").WillReturnResult(ok())
			sqlMock.ExpectExec("^" + regexp.QuoteMeta(`CREATE OR REPLACE FUNCTION "public".`+fn.FunctionName+"(")).WillReturnResult(ok())
			continue
		}
		create.WillReturnResult(ok())
	}
	sqlMock.ExpectQuery(`FROM "public"\.columns c`).WillReturnRows(sqlmock.NewRows(linkColumnFields))
	expectTriggerInstall(sqlMock)

	scripts.CreateMasterSchema(db)
}

func TestMigrateLinkKeys(t *testing.T) {
	t.Run("converts single and multi link columns to key arrays with their metadata", func(t *testing.T) {
		db, _, sqlMock := newSQLMockDB(t)
		sqlMock.ExpectQuery(`a\.atttypid IN`).WillReturnRows(sqlmock.NewRows(linkColumnFields).
			AddRow("c1", "org", "emp", "org", false).
			AddRow("c2", "emps", "org", "emp", true))

		sqlMock.ExpectBegin()
		sqlMock.ExpectExec("^" + regexp.QuoteMeta(`ALTER TABLE "public"."emp" ALTER COLUMN "org" TYPE JSONB USING public.link_ids_to_keys('"public"."org"', CASE WHEN "org" IS NULL THEN NULL ELSE ARRAY["org"::bigint] END)`) + "$").WillReturnResult(ok())
		sqlMock.ExpectExec("^" + regexp.QuoteMeta(`UPDATE "public".columns SET dt = 'JSONB' WHERE id::text = 'c1'`) + "$").WillReturnResult(ok())
		sqlMock.ExpectCommit()

		sqlMock.ExpectBegin()
		sqlMock.ExpectExec("^" + regexp.QuoteMeta(`ALTER TABLE "public"."org" ALTER COLUMN "emps" TYPE JSONB USING public.link_ids_to_keys('"public"."emp"', "emps"::bigint[])`) + "$").WillReturnResult(ok())
		sqlMock.ExpectExec(`UPDATE "public"\.columns SET dt = 'JSONB'`).WillReturnResult(ok())
		sqlMock.ExpectCommit()

		expectTriggerInstall(sqlMock)
		assert.NoError(t, scripts.MigrateLinkKeys(db, "public"))
	})

	t.Run("clears a column whose target table no longer exists", func(t *testing.T) {
		db, _, sqlMock := newSQLMockDB(t)
		sqlMock.ExpectQuery(`a\.atttypid IN`).WillReturnRows(sqlmock.NewRows(linkColumnFields).AddRow("c1", "org", "emp", "", false))
		sqlMock.ExpectBegin()
		sqlMock.ExpectExec(`TYPE JSONB USING NULL$`).WillReturnResult(ok())
		sqlMock.ExpectExec(`UPDATE "public"\.columns`).WillReturnResult(ok())
		sqlMock.ExpectCommit()
		expectTriggerInstall(sqlMock)
		assert.NoError(t, scripts.MigrateLinkKeys(db, "public"))
	})

	t.Run("a failed column is rolled back and the others still run", func(t *testing.T) {
		db, _, sqlMock := newSQLMockDB(t)
		sqlMock.ExpectQuery(`a\.atttypid IN`).WillReturnRows(sqlmock.NewRows(linkColumnFields).AddRow("c1", "org", "emp", "org", false))
		sqlMock.ExpectBegin()
		sqlMock.ExpectExec(`ALTER TABLE`).WillReturnError(errors.New("boom"))
		sqlMock.ExpectRollback()
		expectTriggerInstall(sqlMock)
		assert.NoError(t, scripts.MigrateLinkKeys(db, "public"))
	})

	t.Run("installs the key defaults trigger on tables with a custom key", func(t *testing.T) {
		db, _, sqlMock := newSQLMockDB(t)
		sqlMock.ExpectQuery(`a\.atttypid IN`).WillReturnRows(sqlmock.NewRows(linkColumnFields))
		expectTriggerInstall(sqlMock, `CREATE TRIGGER sb_key_defaults BEFORE INSERT ON public.dept FOR EACH ROW EXECUTE FUNCTION public.key_fill_defaults()`)
		assert.NoError(t, scripts.MigrateLinkKeys(db, "public"))
	})

	t.Run("reports listing and trigger failures", func(t *testing.T) {
		db, _, sqlMock := newSQLMockDB(t)
		sqlMock.ExpectQuery(`a\.atttypid IN`).WillReturnError(errors.New("no access"))
		assert.ErrorContains(t, scripts.MigrateLinkKeys(db, "public"), "list link columns")

		db2, _, sqlMock2 := newSQLMockDB(t)
		sqlMock2.ExpectQuery(`a\.atttypid IN`).WillReturnRows(sqlmock.NewRows(linkColumnFields))
		sqlMock2.ExpectExec(`link_install_key_sync`).WillReturnError(errors.New("no access"))
		assert.ErrorContains(t, scripts.MigrateLinkKeys(db2, "public"), "install link key triggers")
	})
}
