package core_test

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	checkPGName    = "ck_employee_0b6f3a52"
	checkCondition = "emp_code <> ''"
)

// savedCheck is a stored check constraint as the database layer returns it
func savedCheck(title string, columnIDs ...string) map[string]interface{} {
	ids := make([]interface{}, len(columnIDs))
	for i, c := range columnIDs {
		ids[i] = c
	}
	return map[string]interface{}{
		"id": constraintID, "model_id": "m1", "type": "check", "title": title, "constraint_name": checkPGName,
		"column_ids": ids,
	}
}

func checkLabel(category, condition string) string {
	b, _ := json.Marshal(dto.CheckInput{Category: category, Condition: condition})
	return string(b)
}

// expectTest answers the rolled-back Test: limits, the NOT VALID constraint, then the count
func (f *constraintFixture) expectTest(condition string, count int) {
	f.sql.ExpectBegin()
	f.sql.ExpectExec(`^SET LOCAL statement_timeout`).WillReturnResult(sqlmock.NewResult(0, 0))
	f.sql.ExpectExec(`^SET LOCAL lock_timeout`).WillReturnResult(sqlmock.NewResult(0, 0))
	f.sql.ExpectExec(`^ALTER TABLE "tenant_1"\."employee" ADD CONSTRAINT "ck_sereni_test" CHECK \(` + regexp.QuoteMeta(condition) + `\) NOT VALID$`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	f.sql.ExpectQuery(`^SELECT count\(\*\) FROM "tenant_1"\."employee" WHERE \(` + regexp.QuoteMeta(condition) + `\) IS FALSE$`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(count))
	f.sql.ExpectRollback()
}

// expectApply answers adding the constraint: limits, [drop,] add, label, then its columns
func (f *constraintFixture) expectApply(replace bool, condition string, columns ...string) {
	f.sql.ExpectBegin()
	f.sql.ExpectExec(`^SET LOCAL statement_timeout`).WillReturnResult(sqlmock.NewResult(0, 0))
	f.sql.ExpectExec(`^SET LOCAL lock_timeout`).WillReturnResult(sqlmock.NewResult(0, 0))
	if replace {
		f.sql.ExpectExec(`^ALTER TABLE "tenant_1"\."employee" DROP CONSTRAINT IF EXISTS "` + checkPGName + `"$`).WillReturnResult(sqlmock.NewResult(0, 0))
	}
	f.sql.ExpectExec(`^ALTER TABLE "tenant_1"\."employee" ADD CONSTRAINT "ck_employee_[0-9a-f]{8}" CHECK \(` + regexp.QuoteMeta(condition) + `\)$`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	f.sql.ExpectExec(`^COMMENT ON CONSTRAINT "ck_employee_[0-9a-f]{8}" ON "tenant_1"\."employee" IS `).WillReturnResult(sqlmock.NewResult(0, 0))
	rows := sqlmock.NewRows([]string{"attname"})
	for _, c := range columns {
		rows.AddRow(c)
	}
	f.sql.ExpectQuery(`^SELECT a\.attname FROM pg_constraint`).WithArgs(employeeTable, sqlmock.AnyArg()).WillReturnRows(rows)
	f.sql.ExpectCommit()
}

// expectInstalledChecks answers the CHECK constraints Postgres has: name, definition, label
func (f *constraintFixture) expectInstalledChecks(rows ...[3]string) {
	result := sqlmock.NewRows([]string{"conname", "def", "comment"})
	for _, r := range rows {
		result.AddRow(r[0], r[1], r[2])
	}
	f.sql.ExpectQuery(`^SELECT conname, pg_get_constraintdef`).WithArgs(employeeTable).WillReturnRows(result)
}

func TestTableConstraintService_CheckCondition(t *testing.T) {
	check := func(condition string) dto.CheckTableConstraintRequest {
		return dto.CheckTableConstraintRequest{ModelID: "m1", Type: "check", CheckInput: dto.CheckInput{Category: "number_date", Condition: condition}}
	}

	t.Run("valid when Postgres accepts it and no record breaks it", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.expectTest(checkCondition, 0)

		result, err := f.svc.Check(context.Background(), constraintSchema, check("  "+checkCondition+" "))

		require.NoError(t, err)
		assert.True(t, result.Valid)
	})

	t.Run("counts the records that break it", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.expectTest(checkCondition, 3)

		result, err := f.svc.Check(context.Background(), constraintSchema, check(checkCondition))

		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Equal(t, 3, result.Violations)
		assert.Equal(t, "3 records break this condition. Fix them or change the condition.", result.Message)
	})

	t.Run("returns Postgres' reason when it rejects the condition", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.sql.ExpectBegin()
		f.sql.ExpectExec(`^SET LOCAL statement_timeout`).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectExec(`^SET LOCAL lock_timeout`).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectExec(`ADD CONSTRAINT "ck_sereni_test"`).WillReturnError(&pq.Error{Code: "42703", Message: `column "agee" does not exist`})
		f.sql.ExpectRollback()

		_, err := f.svc.Check(context.Background(), constraintSchema, check("agee > 0"))

		assert.ErrorIs(t, err, app_errors.ConstraintFailed)
		assert.Equal(t, `PostgreSQL: column "agee" does not exist`, detailedMessage(err))
	})

	t.Run("says record, not records, for one", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.expectTest(checkCondition, 1)

		result, err := f.svc.Check(context.Background(), constraintSchema, check(checkCondition))

		require.NoError(t, err)
		assert.Equal(t, 1, result.Violations)
		assert.Equal(t, "1 record breaks this condition. Fix it or change the condition.", result.Message)
	})

	t.Run("treats a doubled quote as part of the text", func(t *testing.T) {
		f := newConstraintFixture(t)
		condition := `name <> 'it''s;'`
		f.expectTest(condition, 0)

		result, err := f.svc.Check(context.Background(), constraintSchema, check(condition))

		require.NoError(t, err)
		assert.True(t, result.Valid)
	})

	t.Run("fails when the transaction can't start", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.sql.ExpectBegin().WillReturnError(errors.New("connection refused"))

		_, err := f.svc.Check(context.Background(), constraintSchema, check(checkCondition))

		assert.ErrorIs(t, err, app_errors.ConstraintFailed)
	})

	t.Run("fails when a limit can't be set", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.sql.ExpectBegin()
		f.sql.ExpectExec(`^SET LOCAL statement_timeout`).WillReturnError(errors.New("boom"))
		f.sql.ExpectRollback()

		_, err := f.svc.Check(context.Background(), constraintSchema, check(checkCondition))

		assert.ErrorIs(t, err, app_errors.ConstraintFailed)
	})

	t.Run("fails when the records can't be counted", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.sql.ExpectBegin()
		f.sql.ExpectExec(`^SET LOCAL statement_timeout`).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectExec(`^SET LOCAL lock_timeout`).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectExec(`ADD CONSTRAINT "ck_sereni_test"`).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectQuery(`^SELECT count`).WillReturnError(errors.New("boom"))
		f.sql.ExpectRollback()

		_, err := f.svc.Check(context.Background(), constraintSchema, check(checkCondition))

		assert.ErrorIs(t, err, app_errors.ConstraintFailed)
	})

	t.Run("allows ; -- and now() inside quoted text", func(t *testing.T) {
		f := newConstraintFixture(t)
		condition := `status IN ('a;b', 'c--d', 'now()') AND "Order" > 0`
		f.expectTest(condition, 0)

		result, err := f.svc.Check(context.Background(), constraintSchema, check(condition))

		require.NoError(t, err)
		assert.True(t, result.Valid)
	})

	rejected := map[string]string{
		"empty":              ``,
		"second statement":   `age > 0; DROP TABLE x`,
		"line comment":       `age > 0 --`,
		"block comment":      `age > 0 /* x */`,
		"dollar quoting":     `name = $$x$$`,
		"closes CHECK early": `age > 0), DROP COLUMN email, ADD CHECK (true`,
		"open parenthesis":   `(age > 0`,
		"open quote":         `name = 'abc`,
		"escape string":      `name = E'\'' ) --'`,
		"backslash quote":    `name = 'a\'' OR true`,
		"current date":       `joining_date <= CURRENT_DATE`,
		"now":                `created_at < now ()`,
		"current timestamp":  `x < current_timestamp`,
		"too long":           string(make([]byte, 2001)),
	}
	for name, condition := range rejected {
		t.Run("rejects "+name, func(t *testing.T) {
			f := newConstraintFixture(t)

			_, err := f.svc.Check(context.Background(), constraintSchema, check(condition))

			assert.ErrorIs(t, err, app_errors.CheckConditionInvalid)
		})
	}

	t.Run("rejects an unknown category", func(t *testing.T) {
		f := newConstraintFixture(t)
		req := check(checkCondition)
		req.Category = "number_dates"

		_, err := f.svc.Check(context.Background(), constraintSchema, req)

		assert.ErrorIs(t, err, app_errors.CheckCategoryUnknown)
	})
}

func TestTableConstraintService_CreateCheck(t *testing.T) {
	request := dto.CreateTableConstraintRequest{
		ID: constraintID, ModelID: "m1", Type: "check", Title: "Code not empty", CreatedBy: "user-1",
		CheckInput: dto.CheckInput{Condition: checkCondition},
	}

	t.Run("adds the constraint with the popup's id, labels it and stores its columns", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.expectApply(false, checkCondition, "emp_code", "created_time")

		var stored map[string]interface{}
		f.table.On("CreateRecord", constraintsTable, mock.Anything).
			Run(func(args mock.Arguments) { stored = args.Get(1).(map[string]interface{}) }).
			Return(savedCheck("Code not empty", codeCol.ID.String()), nil)

		out, err := f.svc.Create(context.Background(), constraintSchema, request)

		require.NoError(t, err)
		assert.Equal(t, checkPGName, stored["constraint_name"])
		assert.Equal(t, "check", stored["type"])
		// System columns have no field, so only Emp Code is kept
		assert.Equal(t, `["`+codeCol.ID.String()+`"]`, stored["column_ids"])
		assert.Equal(t, "custom", out.Category)
		assert.Equal(t, checkCondition, out.Condition)
		assert.Equal(t, employeeTable, out.TableName)
		require.Len(t, out.Columns, 1)
		assert.Equal(t, "Emp Code", out.Columns[0].Title)
	})

	t.Run("says how many records break it and adds nothing", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.sql.ExpectBegin()
		f.sql.ExpectExec(`^SET LOCAL statement_timeout`).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectExec(`^SET LOCAL lock_timeout`).WillReturnResult(sqlmock.NewResult(0, 0))
		f.sql.ExpectExec(`ADD CONSTRAINT "ck_employee_0b6f3a52"`).
			WillReturnError(&pq.Error{Code: "23514", Message: `check constraint "ck_employee_0b6f3a52" is violated by some row`})
		f.sql.ExpectRollback()
		f.expectTest(checkCondition, 2)

		_, err := f.svc.Create(context.Background(), constraintSchema, request)

		assert.ErrorIs(t, err, app_errors.CheckExistingRows)
		assert.Equal(t, "2 records break this condition. Fix them or change the condition.", detailedMessage(err))
		f.table.AssertNotCalled(t, "CreateRecord", mock.Anything, mock.Anything)
	})

	t.Run("drops the constraint again when it can't be stored", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.expectApply(false, checkCondition, "emp_code")
		f.table.On("CreateRecord", constraintsTable, mock.Anything).Return(nil, errors.New("insert failed"))
		f.sql.ExpectExec(`^ALTER TABLE "tenant_1"\."employee" DROP CONSTRAINT IF EXISTS "` + checkPGName + `"$`).WillReturnResult(sqlmock.NewResult(0, 0))

		_, err := f.svc.Create(context.Background(), constraintSchema, request)

		assert.Error(t, err)
	})
}

func TestTableConstraintService_UpdateCheck(t *testing.T) {
	label := checkLabel("custom", checkCondition)
	installed := [3]string{checkPGName, "CHECK ((emp_code <> ''::text))", label}

	t.Run("replaces the constraint when the condition changes", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.table.On("GetTableData", constraintsTable, mock.MatchedBy(byID)).Return([]map[string]interface{}{savedCheck("Code not empty", codeCol.ID.String())}, nil)
		f.expectInstalledChecks(installed)
		f.expectApply(true, "email <> ''", "email")
		var updated map[string]interface{}
		f.table.On("UpdateRecord", constraintsTable, constraintID, mock.Anything).
			Run(func(args mock.Arguments) { updated = args.Get(2).(map[string]interface{}) }).
			Return(savedCheck("Email not empty", emailCol.ID.String()), nil)

		out, err := f.svc.Update(context.Background(), constraintSchema, constraintID, dto.UpdateTableConstraintRequest{
			Title: "Email not empty", CheckInput: dto.CheckInput{Category: "custom", Condition: "email <> ''"},
		})

		require.NoError(t, err)
		assert.Equal(t, `["`+emailCol.ID.String()+`"]`, updated["column_ids"])
		assert.Equal(t, "email <> ''", out.Condition)
	})

	t.Run("only saves the title when the condition is the same", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.table.On("GetTableData", constraintsTable, mock.MatchedBy(byID)).Return([]map[string]interface{}{savedCheck("Code not empty", codeCol.ID.String())}, nil)
		f.expectInstalledChecks(installed)
		f.table.On("UpdateRecord", constraintsTable, constraintID, mock.Anything).Return(savedCheck("Code required", codeCol.ID.String()), nil)

		_, err := f.svc.Update(context.Background(), constraintSchema, constraintID, dto.UpdateTableConstraintRequest{
			Title: "Code required", CheckInput: dto.CheckInput{Condition: checkCondition},
		})

		require.NoError(t, err)
	})

	t.Run("only relabels when the category changes", func(t *testing.T) {
		f := newConstraintFixture(t)
		f.table.On("GetTableData", constraintsTable, mock.MatchedBy(byID)).Return([]map[string]interface{}{savedCheck("Code not empty", codeCol.ID.String())}, nil)
		f.expectInstalledChecks(installed)
		f.sql.ExpectExec(`^COMMENT ON CONSTRAINT "` + checkPGName + `"`).WillReturnResult(sqlmock.NewResult(0, 0))
		f.table.On("UpdateRecord", constraintsTable, constraintID, mock.Anything).Return(savedCheck("Code not empty", codeCol.ID.String()), nil)

		out, err := f.svc.Update(context.Background(), constraintSchema, constraintID, dto.UpdateTableConstraintRequest{
			Title: "Code not empty", CheckInput: dto.CheckInput{Category: "text", Condition: checkCondition},
		})

		require.NoError(t, err)
		assert.Equal(t, "text", out.Category)
	})
}

func TestTableConstraintService_ListChecks(t *testing.T) {
	f := newConstraintFixture(t)
	outside := savedCheck("Outside", codeCol.ID.String())
	outside["id"] = "22222222-0000-0000-0000-000000000001"
	outside["constraint_name"] = "chk_manual"
	missing := savedCheck("Gone", emailCol.ID.String())
	missing["id"] = "22222222-0000-0000-0000-000000000002"
	missing["constraint_name"] = "ck_employee_gone"
	f.table.On("GetTableData", constraintsTable, mock.Anything).
		Return([]map[string]interface{}{savedCheck("Code not empty", codeCol.ID.String()), outside, missing}, nil)
	f.expectInstalledChecks(
		[3]string{checkPGName, "CHECK ((emp_code <> ''::text))", checkLabel("text", checkCondition)},
		[3]string{"chk_manual", "CHECK ((emp_code <> 'x'::text)) NOT VALID", ""},
	)

	out, err := f.svc.List(context.Background(), constraintSchema, "m1", "check")

	require.NoError(t, err)
	require.Len(t, out, 3)
	assert.Equal(t, "ok", out[0].Status)
	assert.Equal(t, "text", out[0].Category)
	assert.Equal(t, checkCondition, out[0].Condition)
	assert.Equal(t, "CHECK ((emp_code <> ''::text))", out[0].Definition)
	assert.Equal(t, employeeTable, out[0].TableName)
	// No label: Postgres' own condition
	assert.Equal(t, "custom", out[1].Category)
	assert.Equal(t, "(emp_code <> 'x'::text)", out[1].Condition)
	assert.Equal(t, "missing", out[2].Status)
}

func TestTableConstraintService_UniqueKeyIgnoresChecks(t *testing.T) {
	f := newConstraintFixture(t)
	f.table.On("GetTableData", constraintsTable, mock.MatchedBy(byModel)).
		Return([]map[string]interface{}{savedCheck("Email not empty", emailCol.ID.String())}, nil).Once()
	f.expectNoDuplicates()
	f.sql.ExpectExec(`UNIQUE \("email"\)`).WillReturnResult(sqlmock.NewResult(0, 0))
	f.table.On("CreateRecord", constraintsTable, mock.Anything).Return(savedKey(constraintID, []string{emailCol.ID.String()}), nil)

	_, err := f.svc.Create(context.Background(), constraintSchema, dto.CreateTableConstraintRequest{
		ModelID: "m1", Type: "unique", Title: "Unique email", ColumnIDs: []string{emailCol.ID.String()},
	})

	require.NoError(t, err)
}

func detailedMessage(err error) string {
	var detailed *app_errors.DetailedError
	if errors.As(err, &detailed) {
		return detailed.Message
	}
	return ""
}
