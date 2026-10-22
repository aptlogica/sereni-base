package core_test

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/aptlogica/go-postgres-rest/pkg"
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

type keyModelService struct {
	interfaces.ModelService
	mock.Mock
}

func (m *keyModelService) GetModelByID(ctx context.Context, schemaName, id string) (tenant.Model, error) {
	args := m.Called(schemaName, id)
	return args.Get(0).(tenant.Model), args.Error(1)
}

type keyColumnService struct {
	interfaces.ColumnService
	mock.Mock
}

func (m *keyColumnService) GetColumnByModelID(ctx context.Context, schemaName, modelID string) ([]tenant.Column, error) {
	args := m.Called(schemaName, modelID)
	return args.Get(0).([]tenant.Column), args.Error(1)
}

const (
	keySchema = "public"
	keyTable  = `"public"."org"`
)

var (
	keyModelID = uuid.MustParse("5b0d3a52-8f8e-4d55-9a3c-0b7d1d2f0a11")
	colID      = keyColumn("id", "number", true)
	colTitle   = keyColumn("title", "text", true)
	colCreated = keyColumn("created_time", "datetime", true)
	colCode    = keyColumn("code", "text", false)
	colDept    = keyColumn("dept", "select", false)
	colTags    = keyColumn("tags", "multiSelect", false)
	colLink    = keyColumn("emps", "links", false)
	colOpened  = keyColumn("opened", "date", false)
	colActive  = keyColumn("active", "boolean", false)
)

// System columns are stored with virtual = true, as table creation saves them
func keyColumn(name, uidt string, system bool) tenant.Column {
	return tenant.Column{ID: uuid.New(), ModelID: keyModelID.String(), ColumnName: name, Title: name, UIDT: uidt, System: system, Virtual: system}
}

type keyFixture struct {
	svc     interfaces.PrimaryKeyService
	sql     sqlmock.Sqlmock
	columns *keyColumnService
}

func newKeyFixture(t *testing.T, columns ...tenant.Column) *keyFixture {
	t.Helper()
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, sqlMock.ExpectationsWereMet())
		db.Close()
	})
	models := &keyModelService{}
	models.On("GetModelByID", keySchema, keyModelID.String()).Return(tenant.Model{ID: keyModelID, Alias: "org"}, nil)
	cols := &keyColumnService{}
	cols.On("GetColumnByModelID", keySchema, keyModelID.String()).Return(columns, nil)
	repo := &pkg.DatabaseService{DB: db, SchemaService: newSchemaService(db)}
	return &keyFixture{svc: services.NewPrimaryKeyService(repo, models, cols), sql: sqlMock, columns: cols}
}

func (f *keyFixture) expectKey(cols ...string) {
	f.sql.ExpectQuery(`contype = \$2`).WithArgs(keyTable, "p").
		WillReturnRows(sqlmock.NewRows([]string{"conname", "columns"}).AddRow("org_pkey", "{"+strings.Join(cols, ",")+"}"))
}

// expectCounts answers the validation counts: empty values per key column, records with any empty key
// field, and duplicated combinations
func (f *keyFixture) expectCounts(perColumn []int64, nulls, duplicateGroups int64) {
	names := []string{}
	values := []driver.Value{}
	for i, n := range perColumn {
		names = append(names, fmt.Sprintf("c%d", i))
		values = append(values, n)
	}
	names = append(names, "nulls", "dups")
	values = append(values, nulls, duplicateGroups)
	f.sql.ExpectQuery(`SELECT count\(\*\) FILTER`).WillReturnRows(sqlmock.NewRows(names).AddRow(values...))
}

// expectConstraints answers the primary key name and the UNIQUE constraints, with one on id alone when idUnique is set
func (f *keyFixture) expectConstraints(key, idUnique interface{}) {
	keyRows := sqlmock.NewRows([]string{"conname", "columns"})
	if key != nil {
		keyRows.AddRow(key, "{id}")
	}
	f.sql.ExpectQuery(`contype = \$2`).WithArgs(keyTable, "p").WillReturnRows(keyRows)
	uniqueRows := sqlmock.NewRows([]string{"conname", "columns"})
	if idUnique != nil {
		uniqueRows.AddRow(idUnique, "{id}")
	}
	f.sql.ExpectQuery(`contype = \$2`).WithArgs(keyTable, "u").WillReturnRows(uniqueRows)
}

func exactExec(stmt string) string { return "^" + regexp.QuoteMeta(stmt) + "$" }

func TestPrimaryKeyService_Get(t *testing.T) {
	f := newKeyFixture(t, colID, colTitle, colCreated, colCode, colTags)
	f.expectKey("id")

	out, err := f.svc.Get(context.Background(), keySchema, keyModelID.String())

	require.NoError(t, err)
	assert.Equal(t, []string{colID.ID.String()}, out.ColumnIDs)
	assert.Equal(t, []string{"id"}, out.ColumnNames)
	assert.False(t, out.Locked)
	assert.Empty(t, out.LinkColumns)

	byName := map[string]dto.PrimaryKeyColumn{}
	for _, c := range out.Columns {
		byName[c.ColumnName] = c
	}
	assert.True(t, byName["id"].InKey)
	assert.True(t, byName["id"].Eligible)
	assert.True(t, byName["title"].Eligible)
	assert.True(t, byName["code"].Eligible)
	assert.False(t, byName["created_time"].Eligible, "audit columns are filled in by the system")
	assert.False(t, byName["tags"].Eligible, "multi-value fields can't be key fields")
	assert.NotEmpty(t, byName["tags"].Reason)
}

func TestPrimaryKeyService_GetLockedByLinks(t *testing.T) {
	f := newKeyFixture(t, colID, colCode, colLink)
	f.expectKey("id")

	out, err := f.svc.Get(context.Background(), keySchema, keyModelID.String())

	require.NoError(t, err)
	assert.True(t, out.Locked)
	assert.Equal(t, []string{"emps"}, out.LinkColumns)
	for _, c := range out.Columns {
		if c.ColumnName == "emps" {
			assert.False(t, c.Eligible)
		}
	}
}

func TestPrimaryKeyService_UpdateRejects(t *testing.T) {
	missing := uuid.NewString()
	cases := []struct {
		name    string
		columns []string
		want    error
		message string
	}{
		{"no columns", []string{}, app_errors.PrimaryKeyRequired, ""},
		{"unknown column", []string{missing}, app_errors.PrimaryKeyColumnInvalid, "A selected field (" + missing + ") no longer exists in this table."},
		{"repeated column", []string{colCode.ID.String(), colCode.ID.String()}, app_errors.PrimaryKeyColumnInvalid, "code is selected more than once."},
		{"multi-value field", []string{colTags.ID.String()}, app_errors.PrimaryKeyColumnInvalid, "tags can't be part of the primary key: holds multiple or long values."},
		{"system audit column", []string{colCreated.ID.String()}, app_errors.PrimaryKeyColumnInvalid, "created_time can't be part of the primary key: filled in automatically by the system."},
		{"link field", []string{colLink.ID.String()}, app_errors.PrimaryKeyColumnInvalid, "emps can't be part of the primary key: computed from other records."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newKeyFixture(t, colID, colCreated, colCode, colTags, colLink)
			if len(tc.columns) > 0 {
				f.expectKey("id")
			}
			_, err := f.svc.Update(context.Background(), keySchema, keyModelID.String(), dto.UpdatePrimaryKeyRequest{ColumnIDs: tc.columns})
			assert.ErrorIs(t, err, tc.want)
			if tc.message != "" {
				assert.Contains(t, err.Error(), tc.message)
			}
		})
	}
}

// Any eligible field types can form the key; new records must bring their own key values
func TestPrimaryKeyService_DateWithATextFieldIsAllowed(t *testing.T) {
	f := newKeyFixture(t, colID, colOpened, colCode)
	f.expectKey("id")
	f.expectCounts([]int64{0, 0}, 0, 0)
	f.expectConstraints("org_pkey", nil)
	f.sql.ExpectBegin()
	f.sql.ExpectExec(`DROP CONSTRAINT "org_pkey"`).WillReturnResult(sqlmock.NewResult(0, 0))
	f.sql.ExpectExec(`ADD UNIQUE`).WillReturnResult(sqlmock.NewResult(0, 0))
	f.sql.ExpectExec(`SET NOT NULL`).WillReturnResult(sqlmock.NewResult(0, 0))
	f.sql.ExpectExec(`ADD PRIMARY KEY \("opened", "code"\)`).WillReturnResult(sqlmock.NewResult(0, 0))
	f.sql.ExpectCommit()

	_, err := f.svc.Update(context.Background(), keySchema, keyModelID.String(), dto.UpdatePrimaryKeyRequest{ColumnIDs: []string{colOpened.ID.String(), colCode.ID.String()}})

	require.NoError(t, err)
}

func TestPrimaryKeyService_UpdateUnchangedIsNoop(t *testing.T) {
	f := newKeyFixture(t, colID, colCode, colLink)
	f.expectKey("id")

	out, err := f.svc.Update(context.Background(), keySchema, keyModelID.String(), dto.UpdatePrimaryKeyRequest{ColumnIDs: []string{colID.ID.String()}})

	require.NoError(t, err, "saving the same key is allowed even while links lock it")
	assert.Equal(t, []string{"id"}, out.ColumnNames)
}

func TestPrimaryKeyService_UpdateLockedByLinks(t *testing.T) {
	f := newKeyFixture(t, colID, colCode, colLink)
	f.expectKey("id")

	_, err := f.svc.Update(context.Background(), keySchema, keyModelID.String(), dto.UpdatePrimaryKeyRequest{ColumnIDs: []string{colID.ID.String(), colCode.ID.String()}})

	assert.ErrorIs(t, err, app_errors.PrimaryKeyLocked)
	assert.Contains(t, err.Error(), "while this table has link fields (emps)")
}

// The id and title system columns are stored as virtual but are real columns, so they can be key fields
func TestPrimaryKeyService_SystemIdAndTitleWithACompositeKey(t *testing.T) {
	f := newKeyFixture(t, colID, colTitle, colCode)
	require.True(t, colID.Virtual)
	f.expectKey("id")
	f.expectCounts([]int64{0, 0}, 0, 0)
	f.expectConstraints("org_pkey", nil)
	f.sql.ExpectBegin()
	f.sql.ExpectExec(exactExec(`ALTER TABLE "public"."org" DROP CONSTRAINT "org_pkey"`)).WillReturnResult(sqlmock.NewResult(0, 0))
	f.sql.ExpectExec(exactExec(`ALTER TABLE "public"."org" ALTER COLUMN "id" SET NOT NULL`)).WillReturnResult(sqlmock.NewResult(0, 0))
	f.sql.ExpectExec(exactExec(`ALTER TABLE "public"."org" ADD PRIMARY KEY ("id", "code")`)).WillReturnResult(sqlmock.NewResult(0, 0))
	f.sql.ExpectCommit()

	out, err := f.svc.Update(context.Background(), keySchema, keyModelID.String(), dto.UpdatePrimaryKeyRequest{ColumnIDs: []string{colID.ID.String(), colCode.ID.String()}})

	require.NoError(t, err)
	assert.Equal(t, []string{"id", "code"}, out.ColumnNames)
	for _, c := range out.Columns {
		if c.ColumnName == "id" || c.ColumnName == "title" {
			assert.True(t, c.Eligible, c.ColumnName)
		}
	}
}

func TestPrimaryKeyService_UpdateRejectsBadData(t *testing.T) {
	t.Run("names every empty field and duplicated combination with its records", func(t *testing.T) {
		f := newKeyFixture(t, colID, colTitle, colCode, colDept)
		f.expectKey("id")
		f.expectCounts([]int64{1, 2}, 2, 1)
		f.sql.ExpectQuery(regexp.QuoteMeta(`SELECT id::bigint, COALESCE(title::text, ''), array_remove(ARRAY[CASE WHEN "code" IS NULL THEN 'code' END, CASE WHEN "dept" IS NULL THEN 'dept' END], NULL)::text[] FROM "public"."org" WHERE "code" IS NULL OR "dept" IS NULL ORDER BY id LIMIT 10`)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "title", "missing"}).AddRow(4, "Acme", "{code,dept}").AddRow(9, "", "{dept}"))
		f.sql.ExpectQuery(regexp.QuoteMeta(`SELECT jsonb_build_object('code', "code", 'dept', "dept") AS k, count(*), (array_agg(id::bigint ORDER BY id))[1:10] FROM "public"."org" WHERE NOT ("code" IS NULL OR "dept" IS NULL) GROUP BY 1 HAVING count(*) > 1`)).
			WillReturnRows(sqlmock.NewRows([]string{"k", "count", "ids"}).AddRow([]byte(`{"code":"IN","dept":"HR"}`), 3, "{1,5,7}"))

		_, err := f.svc.Update(context.Background(), keySchema, keyModelID.String(), dto.UpdatePrimaryKeyRequest{ColumnIDs: []string{colCode.ID.String(), colDept.ID.String()}})

		assert.ErrorIs(t, err, app_errors.PrimaryKeyDataInvalid)
		want := `Can't set the primary key to code + dept. ` +
			`code is empty in 1 record (#4 Acme). ` +
			`dept is empty in 2 records (#4 Acme, #9). ` +
			`The combination code = "IN", dept = "HR" is used by 3 records (#1, #5, #7). ` +
			`Fill in or change these records, then save again.`
		assert.EqualError(t, err, want)

		var violation *app_errors.PrimaryKeyViolation
		require.True(t, errors.As(err, &violation))
		assert.Equal(t, want, violation.Message)
		details := violation.Details.(dto.PrimaryKeyViolationDetails)
		assert.Equal(t, want, details.Message)
		assert.Equal(t, []string{"code", "dept"}, details.Columns)
		assert.Equal(t, int64(2), details.NullCount)
		assert.Equal(t, []dto.PrimaryKeyEmptyField{{Column: "code", Title: "code", Count: 1}, {Column: "dept", Title: "dept", Count: 2}}, details.EmptyFields)
		assert.Equal(t, []dto.PrimaryKeyRecord{{ID: 4, Title: "Acme", Missing: []string{"code", "dept"}}, {ID: 9, Title: "", Missing: []string{"dept"}}}, details.NullRecords)
		assert.Equal(t, int64(1), details.DuplicateGroups)
		assert.Equal(t, []dto.PrimaryKeyDuplicate{{Values: map[string]interface{}{"code": "IN", "dept": "HR"}, Count: 3, RecordIDs: []int64{1, 5, 7}}}, details.Duplicates)
	})

	t.Run("a single key field with duplicates and more groups than shown", func(t *testing.T) {
		f := newKeyFixture(t, colID, colCode)
		f.expectKey("id")
		f.expectCounts([]int64{0}, 0, 12)
		rows := sqlmock.NewRows([]string{"k", "count", "ids"})
		for i := 0; i < 10; i++ {
			rows.AddRow([]byte(fmt.Sprintf(`{"code":"C%d"}`, i)), 2, "{1,2}")
		}
		f.sql.ExpectQuery(`AS k, count`).WillReturnRows(rows)

		_, err := f.svc.Update(context.Background(), keySchema, keyModelID.String(), dto.UpdatePrimaryKeyRequest{ColumnIDs: []string{colCode.ID.String()}})

		require.Error(t, err)
		assert.Contains(t, err.Error(), `The value code = "C0" is used by 2 records (#1, #2).`)
		assert.Contains(t, err.Error(), `2 more values are also duplicated.`)
	})
}

func TestPrimaryKeyService_UpdateReplacesKey(t *testing.T) {
	t.Run("composite key without id keeps id unique", func(t *testing.T) {
		f := newKeyFixture(t, colID, colCode, colDept)
		f.expectKey("id")
		f.expectCounts([]int64{0, 0}, 0, 0)
		f.expectConstraints("org_pkey", nil)
		f.sql.ExpectBegin()
		f.sql.ExpectExec(exactExec(`ALTER TABLE "public"."org" DROP CONSTRAINT "org_pkey"`)).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectExec(exactExec(`ALTER TABLE "public"."org" ADD UNIQUE ("id")`)).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectExec(exactExec(`ALTER TABLE "public"."org" ALTER COLUMN "id" SET NOT NULL`)).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectExec(exactExec(`ALTER TABLE "public"."org" ADD PRIMARY KEY ("code", "dept")`)).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectCommit()

		out, err := f.svc.Update(context.Background(), keySchema, keyModelID.String(), dto.UpdatePrimaryKeyRequest{ColumnIDs: []string{colCode.ID.String(), colDept.ID.String()}})

		require.NoError(t, err)
		assert.Equal(t, []string{"code", "dept"}, out.ColumnNames)
		assert.Equal(t, []string{colCode.ID.String(), colDept.ID.String()}, out.ColumnIDs)
	})

	t.Run("removing a key field revalidates the rest and lets it be blank again", func(t *testing.T) {
		f := newKeyFixture(t, colID, colCode, colDept)
		f.expectKey("code", "dept")
		f.expectCounts([]int64{0}, 0, 0)
		f.expectConstraints("org_pkey", "org_id_key")
		f.sql.ExpectBegin()
		f.sql.ExpectExec(exactExec(`ALTER TABLE "public"."org" DROP CONSTRAINT "org_pkey"`)).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectExec(exactExec(`ALTER TABLE "public"."org" ALTER COLUMN "id" SET NOT NULL`)).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectExec(exactExec(`ALTER TABLE "public"."org" ALTER COLUMN "dept" DROP NOT NULL`)).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectExec(exactExec(`ALTER TABLE "public"."org" ADD PRIMARY KEY ("code")`)).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectCommit()

		_, err := f.svc.Update(context.Background(), keySchema, keyModelID.String(), dto.UpdatePrimaryKeyRequest{ColumnIDs: []string{colCode.ID.String()}})

		require.NoError(t, err)
	})

	t.Run("back to id drops the extra unique constraint", func(t *testing.T) {
		f := newKeyFixture(t, colID, colCode)
		f.expectKey("code")
		f.expectCounts([]int64{0}, 0, 0)
		f.expectConstraints("org_pkey", "org_id_key")
		f.sql.ExpectBegin()
		f.sql.ExpectExec(exactExec(`ALTER TABLE "public"."org" DROP CONSTRAINT "org_pkey"`)).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectExec(exactExec(`ALTER TABLE "public"."org" DROP CONSTRAINT "org_id_key"`)).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectExec(exactExec(`ALTER TABLE "public"."org" ALTER COLUMN "id" SET NOT NULL`)).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectExec(exactExec(`ALTER TABLE "public"."org" ALTER COLUMN "code" DROP NOT NULL`)).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectExec(exactExec(`ALTER TABLE "public"."org" ADD PRIMARY KEY ("id")`)).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectCommit()

		_, err := f.svc.Update(context.Background(), keySchema, keyModelID.String(), dto.UpdatePrimaryKeyRequest{ColumnIDs: []string{colID.ID.String()}})

		require.NoError(t, err)
	})

	t.Run("Postgres errors keep their detail", func(t *testing.T) {
		f := newKeyFixture(t, colID, colCode)
		f.expectKey("id")
		f.expectCounts([]int64{0}, 0, 0)
		f.expectConstraints("org_pkey", nil)
		f.sql.ExpectBegin()
		f.sql.ExpectExec(`DROP CONSTRAINT`).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectExec(`ADD UNIQUE`).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectExec(`SET NOT NULL`).WillReturnResult(sqlmock.NewResult(0, 0))
		// A record added after validation duplicates the key
		f.sql.ExpectExec(`ADD PRIMARY KEY`).WillReturnError(&pq.Error{
			Code: "23505", Message: `could not create unique index "org_pkey"`, Detail: `Key (code)=(HR) is duplicated.`,
		})
		f.sql.ExpectRollback()

		_, err := f.svc.Update(context.Background(), keySchema, keyModelID.String(), dto.UpdatePrimaryKeyRequest{ColumnIDs: []string{colCode.ID.String()}})

		assert.ErrorIs(t, err, app_errors.PrimaryKeyUpdateFailed)
		assert.Contains(t, err.Error(), "Duplicate key values: Key (code)=(HR) is duplicated.")
	})

	t.Run("an empty value found by Postgres names the field", func(t *testing.T) {
		f := newKeyFixture(t, colID, colCode)
		f.expectKey("id")
		f.expectCounts([]int64{0}, 0, 0)
		f.expectConstraints("org_pkey", nil)
		f.sql.ExpectBegin()
		f.sql.ExpectExec(`DROP CONSTRAINT`).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectExec(`ADD UNIQUE`).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectExec(`SET NOT NULL`).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectExec(`ADD PRIMARY KEY`).WillReturnError(&pq.Error{
			Code: "23502", Message: `column "code" of relation "org" contains null values`, Column: "code",
		})
		f.sql.ExpectRollback()

		_, err := f.svc.Update(context.Background(), keySchema, keyModelID.String(), dto.UpdatePrimaryKeyRequest{ColumnIDs: []string{colCode.ID.String()}})

		assert.ErrorIs(t, err, app_errors.PrimaryKeyUpdateFailed)
		assert.Contains(t, err.Error(), "code is empty in some records")
	})

	t.Run("a Postgres failure rolls everything back", func(t *testing.T) {
		f := newKeyFixture(t, colID, colCode)
		f.expectKey("id")
		f.expectCounts([]int64{0}, 0, 0)
		f.expectConstraints("org_pkey", nil)
		f.sql.ExpectBegin()
		f.sql.ExpectExec(`DROP CONSTRAINT`).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectExec(`ADD UNIQUE`).WillReturnError(errors.New(`could not create unique index "org_id_key"`))
		f.sql.ExpectRollback()

		_, err := f.svc.Update(context.Background(), keySchema, keyModelID.String(), dto.UpdatePrimaryKeyRequest{ColumnIDs: []string{colCode.ID.String()}})

		assert.ErrorIs(t, err, app_errors.PrimaryKeyUpdateFailed)
		assert.Contains(t, err.Error(), "could not create unique index")
	})
}

func TestPrimaryKeyService_EnsureColumnNotInKey(t *testing.T) {
	f := newKeyFixture(t)
	f.expectKey("code", "dept")
	assert.ErrorIs(t, f.svc.EnsureColumnNotInKey(context.Background(), keySchema, keyModelID.String(), "dept"), app_errors.ColumnInPrimaryKey)

	f.expectKey("code", "dept")
	assert.NoError(t, f.svc.EnsureColumnNotInKey(context.Background(), keySchema, keyModelID.String(), "title"))
}
