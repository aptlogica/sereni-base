package core_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/aptlogica/go-postgres-rest/pkg"
	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	services "github.com/aptlogica/sereni-base/internal/services/core"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// automationModelService returns the table an automation belongs to
type automationModelService struct {
	interfaces.ModelService
	mock.Mock
}

func (m *automationModelService) GetModelByID(ctx context.Context, schemaName, id string) (tenant.Model, error) {
	args := m.Called(schemaName, id)
	return args.Get(0).(tenant.Model), args.Error(1)
}

const (
	automationSchema = "tenant_1"
	automationID     = "7f1c2e0a-5b9d-4c3e-8a21-0d9f6b4e1a77"
	automationLabel  = "automation:" + automationID

	triggerQuery  = `CREATE TRIGGER log_salary AFTER INSERT OR UPDATE ON "public"."employee" FOR EACH ROW EXECUTE FUNCTION log_fn()`
	functionQuery = `CREATE OR REPLACE FUNCTION log_fn() RETURNS trigger AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql`
)

var automationsTable = tenant.Automation{}.TableName(automationSchema)

type automationFixture struct {
	svc    interfaces.AutomationService
	table  *MockTableService
	models *automationModelService
	sql    sqlmock.Sqlmock
}

func newAutomationFixture(t *testing.T) *automationFixture {
	t.Helper()
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, sqlMock.ExpectationsWereMet())
		db.Close()
	})

	table := &MockTableService{}
	models := &automationModelService{}
	models.On("GetModelByID", automationSchema, "m1").Return(tenant.Model{Alias: "employee"}, nil).Maybe()

	repo := &pkg.DatabaseService{TableService: table, DB: db}
	return &automationFixture{svc: services.NewAutomationService(repo, models), table: table, models: models, sql: sqlMock}
}

// exact matches a statement literally
func exact(stmt string) string { return "^" + regexp.QuoteMeta(stmt) + "$" }

func savedAutomation(kind, context string) map[string]interface{} {
	return map[string]interface{}{
		"id": automationID, "model_id": "m1", "title": "Saved", "type": kind, "context": context, "event": "",
	}
}

// expectGetByID makes the stored entry with this context the one loaded by id
func (f *automationFixture) expectGetByID(kind, context string) {
	f.table.On("GetTableData", automationsTable, mock.MatchedBy(func(p dbModels.QueryParams) bool {
		return len(p.Filters) == 1 && p.Filters[0].Column == "id" && p.Filters[0].Value == automationID
	})).Return([]map[string]interface{}{savedAutomation(kind, context)}, nil).Once()
}

func (f *automationFixture) expectLabelledTrigger(name string) {
	q := f.sql.ExpectQuery("FROM pg_trigger").WithArgs(`"public"."employee"`, automationLabel)
	if name == "" {
		q.WillReturnError(sql.ErrNoRows)
		return
	}
	q.WillReturnRows(sqlmock.NewRows([]string{"quote_ident"}).AddRow(name))
}

func (f *automationFixture) expectLabelledFunction(name string) {
	q := f.sql.ExpectQuery("FROM pg_proc").WithArgs(automationLabel)
	if name == "" {
		q.WillReturnError(sql.ErrNoRows)
		return
	}
	q.WillReturnRows(sqlmock.NewRows([]string{"quote_ident"}).AddRow(name))
}

// expectTitles makes the titles already used on the table the given (type, title) pairs
func (f *automationFixture) expectTitles(used ...[2]string) {
	rows := []map[string]interface{}{}
	for i, u := range used {
		row := savedAutomation(u[0], "{}")
		row["id"] = fmt.Sprintf("00000000-0000-4000-8000-%012d", i+1)
		row["title"] = u[1]
		rows = append(rows, row)
	}
	f.table.On("GetTableData", automationsTable, mock.MatchedBy(func(p dbModels.QueryParams) bool {
		return len(p.Filters) == 1 && p.Filters[0].Column == "model_id"
	})).Return(rows, nil).Maybe()
}

func ok() sql.Result { return sqlmock.NewResult(0, 0) }

func TestAutomationService_Create(t *testing.T) {
	t.Run("installs a trigger, labels it and stores the entry", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.expectTitles()
		f.sql.ExpectBegin()
		f.sql.ExpectExec(exact(triggerQuery)).WillReturnResult(ok())
		f.sql.ExpectExec(`^COMMENT ON TRIGGER log_salary ON "public"\."employee" IS 'automation:[0-9a-f-]{36}'$`).WillReturnResult(ok())
		f.sql.ExpectCommit()

		var stored map[string]interface{}
		f.table.On("CreateRecord", automationsTable, mock.Anything).
			Run(func(args mock.Arguments) { stored = args.Get(1).(map[string]interface{}) }).
			Return(savedAutomation("trigger", triggerQuery), nil)

		out, err := f.svc.Create(context.Background(), automationSchema, dto.CreateAutomationRequest{
			ModelID: "m1", Title: "Log salary", Type: "trigger", Context: "  " + triggerQuery + ";  ",
			Event: "after update, after insert", CreatedBy: "user-1",
		})

		require.NoError(t, err)
		assert.Equal(t, automationID, out.ID.String())
		assert.Equal(t, triggerQuery, stored["context"], "trailing ; and spaces are trimmed")
		assert.Equal(t, "AFTER INSERT, AFTER UPDATE", stored["event"], "events are normalised")
		assert.Equal(t, "user-1", stored["created_by"])
		assert.Equal(t, "user-1", stored["last_modified_by"])
	})

	t.Run("installs a function", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.expectTitles()
		f.sql.ExpectBegin()
		f.sql.ExpectExec(exact(functionQuery)).WillReturnResult(ok())
		f.sql.ExpectExec(`^COMMENT ON FUNCTION log_fn\(\) IS 'automation:[0-9a-f-]{36}'$`).WillReturnResult(ok())
		f.sql.ExpectCommit()
		f.table.On("CreateRecord", automationsTable, mock.MatchedBy(func(d map[string]interface{}) bool {
			return d["type"] == "function" && d["event"] == ""
		})).Return(savedAutomation("function", functionQuery), nil)

		_, err := f.svc.Create(context.Background(), automationSchema, dto.CreateAutomationRequest{
			ModelID: "m1", Title: "Log fn", Type: "function", Context: functionQuery + ";",
		})
		require.NoError(t, err)
	})

	t.Run("stores a webhook without running any SQL", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.expectTitles()
		// An empty transaction: there is no query to run for a webhook
		f.sql.ExpectBegin()
		f.sql.ExpectCommit()
		f.table.On("CreateRecord", automationsTable, mock.Anything).Return(savedAutomation("webhook", `{"url":"x"}`), nil)

		_, err := f.svc.Create(context.Background(), automationSchema, dto.CreateAutomationRequest{
			ModelID: "m1", Title: "Notify", Type: "webhook", Context: `{"url":"x"}`, Event: "ignored",
		})
		require.NoError(t, err)
	})

	invalid := []struct {
		name string
		req  dto.CreateAutomationRequest
		want error
	}{
		{"trigger on another table", dto.CreateAutomationRequest{Type: "trigger", Event: "AFTER INSERT",
			Context: `CREATE TRIGGER t AFTER INSERT ON "public"."salary" FOR EACH ROW EXECUTE FUNCTION f()`}, app_errors.InvalidTriggerQuery},
		{"more than one statement", dto.CreateAutomationRequest{Type: "trigger", Event: "AFTER INSERT",
			Context: triggerQuery + "; DROP TABLE employee"}, app_errors.InvalidTriggerQuery},
		{"not a trigger", dto.CreateAutomationRequest{Type: "trigger", Event: "AFTER INSERT", Context: "SELECT 1"}, app_errors.InvalidTriggerQuery},
		{"trigger with its own function", dto.CreateAutomationRequest{Type: "trigger", Event: "AFTER INSERT",
			Context: functionQuery + "; " + triggerQuery}, app_errors.TriggerHasFunction},
		{"function that is not a trigger function", dto.CreateAutomationRequest{Type: "function",
			Context: "CREATE FUNCTION add(a int) RETURNS int AS $$ SELECT a $$ LANGUAGE sql"}, app_errors.InvalidFunctionQuery},
		{"function followed by more SQL", dto.CreateAutomationRequest{Type: "function",
			Context: functionQuery + "; " + triggerQuery}, app_errors.InvalidFunctionQuery},
		{"event without timing", dto.CreateAutomationRequest{Type: "trigger", Event: "INSERT", Context: triggerQuery}, app_errors.InvalidTriggerEvent},
		{"two timings", dto.CreateAutomationRequest{Type: "trigger", Event: "BEFORE INSERT, AFTER UPDATE", Context: triggerQuery}, app_errors.InvalidTriggerEvent},
		{"unknown operation", dto.CreateAutomationRequest{Type: "trigger", Event: "AFTER SELECT", Context: triggerQuery}, app_errors.InvalidTriggerEvent},
	}
	for _, tc := range invalid {
		t.Run("rejects "+tc.name, func(t *testing.T) {
			f := newAutomationFixture(t)
			f.expectTitles()
			tc.req.ModelID, tc.req.Title = "m1", "x"
			_, err := f.svc.Create(context.Background(), automationSchema, tc.req)
			assert.ErrorIs(t, err, tc.want)
			f.table.AssertNotCalled(t, "CreateRecord", mock.Anything, mock.Anything)
		})
	}

	t.Run("returns the Postgres error and stores nothing when the query fails", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.expectTitles()
		f.sql.ExpectBegin()
		f.sql.ExpectExec(exact(triggerQuery)).WillReturnError(errors.New(`function log_fn() does not exist`))
		f.sql.ExpectRollback()

		_, err := f.svc.Create(context.Background(), automationSchema, dto.CreateAutomationRequest{
			ModelID: "m1", Title: "x", Type: "trigger", Context: triggerQuery, Event: "AFTER INSERT",
		})
		assert.ErrorIs(t, err, app_errors.TriggerFailed)
		assert.Contains(t, err.Error(), "function log_fn() does not exist")
		f.table.AssertNotCalled(t, "CreateRecord", mock.Anything, mock.Anything)
	})

	t.Run("drops the installed trigger when storing the entry fails", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.expectTitles()
		f.sql.ExpectBegin()
		f.sql.ExpectExec(exact(triggerQuery)).WillReturnResult(ok())
		f.sql.ExpectExec("^COMMENT ON TRIGGER").WillReturnResult(ok())
		f.sql.ExpectCommit()
		f.sql.ExpectExec(exact(`DROP TRIGGER IF EXISTS log_salary ON "public"."employee"`)).WillReturnResult(ok())
		f.table.On("CreateRecord", automationsTable, mock.Anything).Return(nil, errors.New("db down"))

		_, err := f.svc.Create(context.Background(), automationSchema, dto.CreateAutomationRequest{
			ModelID: "m1", Title: "x", Type: "trigger", Context: triggerQuery, Event: "AFTER INSERT",
		})
		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("drops the installed function when storing the entry fails", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.expectTitles()
		f.sql.ExpectBegin()
		f.sql.ExpectExec(exact(functionQuery)).WillReturnResult(ok())
		f.sql.ExpectExec("^COMMENT ON FUNCTION").WillReturnResult(ok())
		f.sql.ExpectCommit()
		f.sql.ExpectExec(exact(`DROP FUNCTION IF EXISTS log_fn()`)).WillReturnResult(ok())
		f.table.On("CreateRecord", automationsTable, mock.Anything).Return(nil, errors.New("db down"))

		_, err := f.svc.Create(context.Background(), automationSchema, dto.CreateAutomationRequest{
			ModelID: "m1", Title: "x", Type: "function", Context: functionQuery,
		})
		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("fails when the table does not exist", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.models.On("GetModelByID", automationSchema, "missing").Return(tenant.Model{}, app_errors.ErrRecordNotFound)

		_, err := f.svc.Create(context.Background(), automationSchema, dto.CreateAutomationRequest{
			ModelID: "missing", Title: "x", Type: "webhook", Context: "{}",
		})
		assert.ErrorIs(t, err, app_errors.ErrRecordNotFound)
	})
}

func TestAutomationService_DuplicateTitle(t *testing.T) {
	t.Run("create rejects a title used by the same type, ignoring case and spaces", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.expectTitles([2]string{"trigger", "Log Salary"})

		_, err := f.svc.Create(context.Background(), automationSchema, dto.CreateAutomationRequest{
			ModelID: "m1", Title: "  log salary ", Type: "trigger", Context: triggerQuery, Event: "AFTER INSERT",
		})
		assert.ErrorIs(t, err, app_errors.DuplicateAutomationTitle)
		f.table.AssertNotCalled(t, "CreateRecord", mock.Anything, mock.Anything)
	})

	t.Run("create allows a title used by another type", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.expectTitles([2]string{"function", "Log Salary"})
		f.sql.ExpectBegin()
		f.sql.ExpectExec(exact(triggerQuery)).WillReturnResult(ok())
		f.sql.ExpectExec(`^COMMENT ON TRIGGER`).WillReturnResult(ok())
		f.sql.ExpectCommit()
		f.table.On("CreateRecord", automationsTable, mock.Anything).Return(savedAutomation("trigger", triggerQuery), nil)

		_, err := f.svc.Create(context.Background(), automationSchema, dto.CreateAutomationRequest{
			ModelID: "m1", Title: "Log salary", Type: "trigger", Context: triggerQuery, Event: "AFTER INSERT",
		})
		require.NoError(t, err)
	})

	t.Run("create fails when the titles can't be read", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.table.On("GetTableData", automationsTable, mock.Anything).Return(nil, errors.New("db down"))

		_, err := f.svc.Create(context.Background(), automationSchema, dto.CreateAutomationRequest{
			ModelID: "m1", Title: "x", Type: "webhook", Context: "{}",
		})
		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("update rejects a title used by another entry", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.expectGetByID("trigger", triggerQuery)
		f.expectTitles([2]string{"trigger", "Audit"})

		_, err := f.svc.Update(context.Background(), automationSchema, automationID, dto.UpdateAutomationRequest{
			Title: "audit", Context: triggerQuery, Event: "AFTER INSERT",
		})
		assert.ErrorIs(t, err, app_errors.DuplicateAutomationTitle)
		f.table.AssertNotCalled(t, "UpdateRecord", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("update keeps the entry's own title", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.expectGetByID("webhook", "{}")
		// The listed entry has the id being updated, so it is not a duplicate
		f.table.On("GetTableData", automationsTable, mock.MatchedBy(func(p dbModels.QueryParams) bool {
			return len(p.Filters) == 1 && p.Filters[0].Column == "model_id"
		})).Return([]map[string]interface{}{savedAutomation("webhook", "{}")}, nil)
		f.table.On("UpdateRecord", automationsTable, automationID, mock.Anything).Return(savedAutomation("webhook", "{}"), nil)

		_, err := f.svc.Update(context.Background(), automationSchema, automationID, dto.UpdateAutomationRequest{Title: "Saved", Context: "{}"})
		require.NoError(t, err)
	})
}

func TestAutomationService_Update(t *testing.T) {
	t.Run("saves title, query and event without applying the query", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.expectTitles()
		f.expectGetByID("trigger", triggerQuery)
		// Already labelled, so nothing is written to Postgres
		f.expectLabelledTrigger("log_salary")

		newQuery := `CREATE TRIGGER log_salary BEFORE DELETE ON "public"."employee" FOR EACH ROW EXECUTE FUNCTION log_fn()`
		var saved map[string]interface{}
		f.table.On("UpdateRecord", automationsTable, automationID, mock.Anything).
			Run(func(args mock.Arguments) { saved = args.Get(2).(map[string]interface{}) }).
			Return(savedAutomation("trigger", newQuery), nil)

		_, err := f.svc.Update(context.Background(), automationSchema, automationID, dto.UpdateAutomationRequest{
			Title: "Audit", Context: newQuery + ";", Event: "before delete", UpdatedBy: "user-2",
		})

		require.NoError(t, err)
		assert.Equal(t, "Audit", saved["title"])
		assert.Equal(t, newQuery, saved["context"])
		assert.Equal(t, "BEFORE DELETE", saved["event"])
		assert.Equal(t, "user-2", saved["last_modified_by"])
		assert.NotNil(t, saved["last_modified_time"])
	})

	t.Run("labels a trigger installed before labels existed", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.expectTitles()
		f.expectGetByID("trigger", triggerQuery)
		f.expectLabelledTrigger("")
		f.sql.ExpectExec(exact(`COMMENT ON TRIGGER log_salary ON "public"."employee" IS '` + automationLabel + `'`)).WillReturnResult(ok())
		f.table.On("UpdateRecord", automationsTable, automationID, mock.Anything).Return(savedAutomation("trigger", triggerQuery), nil)

		_, err := f.svc.Update(context.Background(), automationSchema, automationID, dto.UpdateAutomationRequest{
			Title: "x", Context: triggerQuery, Event: "AFTER INSERT",
		})
		require.NoError(t, err)
	})

	t.Run("labels a function installed before labels existed", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.expectTitles()
		f.expectGetByID("function", functionQuery)
		f.expectLabelledFunction("")
		f.sql.ExpectExec(exact(`COMMENT ON FUNCTION log_fn() IS '` + automationLabel + `'`)).WillReturnResult(ok())
		f.table.On("UpdateRecord", automationsTable, automationID, mock.Anything).Return(savedAutomation("function", functionQuery), nil)

		_, err := f.svc.Update(context.Background(), automationSchema, automationID, dto.UpdateAutomationRequest{Title: "x", Context: functionQuery})
		require.NoError(t, err)
	})

	t.Run("checks the query format", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.expectTitles()
		f.expectGetByID("trigger", triggerQuery)

		_, err := f.svc.Update(context.Background(), automationSchema, automationID, dto.UpdateAutomationRequest{
			Title: "x", Context: "DROP TABLE employee", Event: "AFTER INSERT",
		})
		assert.ErrorIs(t, err, app_errors.InvalidTriggerQuery)
		f.table.AssertNotCalled(t, "UpdateRecord", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("checks the event", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.expectTitles()
		f.expectGetByID("trigger", triggerQuery)

		_, err := f.svc.Update(context.Background(), automationSchema, automationID, dto.UpdateAutomationRequest{Title: "x", Context: triggerQuery})
		assert.ErrorIs(t, err, app_errors.InvalidTriggerEvent)
	})

	t.Run("returns not found for an unknown id", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.table.On("GetTableData", automationsTable, mock.Anything).Return([]map[string]interface{}{}, nil)

		_, err := f.svc.Update(context.Background(), automationSchema, automationID, dto.UpdateAutomationRequest{Title: "x", Context: "{}"})
		assert.ErrorIs(t, err, app_errors.AutomationNotFound)
	})

	t.Run("fails when saving fails", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.expectTitles()
		f.expectGetByID("webhook", "{}")
		f.table.On("UpdateRecord", automationsTable, automationID, mock.Anything).Return(nil, errors.New("db down"))

		_, err := f.svc.Update(context.Background(), automationSchema, automationID, dto.UpdateAutomationRequest{Title: "x", Context: "{}"})
		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})
}

func TestAutomationService_Run(t *testing.T) {
	t.Run("replaces the installed trigger in one transaction", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.expectGetByID("trigger", triggerQuery)
		// Installed under an older name; found by its label
		f.expectLabelledTrigger("old_name")
		f.sql.ExpectBegin()
		f.sql.ExpectExec(exact(`DROP TRIGGER IF EXISTS old_name ON "public"."employee"`)).WillReturnResult(ok())
		f.sql.ExpectExec(exact(triggerQuery)).WillReturnResult(ok())
		f.sql.ExpectExec(exact(`COMMENT ON TRIGGER log_salary ON "public"."employee" IS '` + automationLabel + `'`)).WillReturnResult(ok())
		f.sql.ExpectCommit()

		out, err := f.svc.Run(context.Background(), automationSchema, automationID)
		require.NoError(t, err)
		assert.Equal(t, automationID, out.ID.String())
	})

	t.Run("keeps the old trigger when the new query fails", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.expectGetByID("trigger", triggerQuery)
		f.expectLabelledTrigger("log_salary")
		f.sql.ExpectBegin()
		f.sql.ExpectExec("^DROP TRIGGER").WillReturnResult(ok())
		f.sql.ExpectExec(exact(triggerQuery)).WillReturnError(errors.New("function log_fn() does not exist"))
		f.sql.ExpectRollback()

		_, err := f.svc.Run(context.Background(), automationSchema, automationID)
		assert.ErrorIs(t, err, app_errors.TriggerFailed)
	})

	t.Run("replaces the body of a function with the same name", func(t *testing.T) {
		f := newAutomationFixture(t)
		plain := `CREATE FUNCTION log_fn() RETURNS trigger AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql`
		f.expectGetByID("function", plain)
		f.expectLabelledFunction("log_fn")
		f.sql.ExpectBegin()
		f.sql.ExpectExec(exact(functionQuery)).WillReturnResult(ok())
		f.sql.ExpectExec(exact(`COMMENT ON FUNCTION log_fn() IS '` + automationLabel + `'`)).WillReturnResult(ok())
		f.sql.ExpectCommit()

		_, err := f.svc.Run(context.Background(), automationSchema, automationID)
		require.NoError(t, err)
	})

	t.Run("creates a renamed function and drops the old one", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.expectGetByID("function", functionQuery)
		f.expectLabelledFunction("old_fn")
		f.sql.ExpectBegin()
		f.sql.ExpectExec(exact(functionQuery)).WillReturnResult(ok())
		f.sql.ExpectExec(exact(`DROP FUNCTION IF EXISTS old_fn()`)).WillReturnResult(ok())
		f.sql.ExpectExec("^COMMENT ON FUNCTION log_fn").WillReturnResult(ok())
		f.sql.ExpectCommit()

		_, err := f.svc.Run(context.Background(), automationSchema, automationID)
		require.NoError(t, err)
	})

	t.Run("does nothing in Postgres for a webhook", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.expectGetByID("webhook", "{}")

		_, err := f.svc.Run(context.Background(), automationSchema, automationID)
		require.NoError(t, err)
	})

	t.Run("rejects a saved query that is no longer valid", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.expectGetByID("trigger", "SELECT 1")

		_, err := f.svc.Run(context.Background(), automationSchema, automationID)
		assert.ErrorIs(t, err, app_errors.InvalidTriggerQuery)
	})
}

func TestAutomationService_Validate(t *testing.T) {
	t.Run("runs a new trigger and rolls back", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.sql.ExpectBegin()
		f.sql.ExpectExec(exact(triggerQuery)).WillReturnResult(ok())
		f.sql.ExpectExec(`^COMMENT ON TRIGGER log_salary ON "public"."employee" IS 'automation:`).WillReturnResult(ok())
		f.sql.ExpectRollback()

		out, err := f.svc.Validate(context.Background(), automationSchema, dto.ValidateAutomationRequest{
			ModelID: "m1", Type: "trigger", Context: triggerQuery + ";", Event: "AFTER INSERT, AFTER UPDATE",
		})
		require.NoError(t, err)
		assert.True(t, out.Valid)
		assert.Empty(t, out.Message)
	})

	t.Run("returns Postgres' message and rolls back", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.sql.ExpectBegin()
		f.sql.ExpectExec(exact(triggerQuery)).WillReturnError(errors.New("pq: function log_fn() does not exist"))
		f.sql.ExpectRollback()

		out, err := f.svc.Validate(context.Background(), automationSchema, dto.ValidateAutomationRequest{
			ModelID: "m1", Type: "trigger", Context: triggerQuery, Event: "AFTER INSERT",
		})
		require.NoError(t, err)
		assert.False(t, out.Valid)
		assert.Equal(t, "function log_fn() does not exist", out.Message)
	})

	t.Run("checks an edited trigger as Run replaces the installed one", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.expectGetByID("trigger", triggerQuery)
		f.expectLabelledTrigger("old_name")
		edited := `CREATE TRIGGER log_salary BEFORE DELETE ON "public"."employee" FOR EACH ROW EXECUTE FUNCTION log_fn()`
		f.sql.ExpectBegin()
		f.sql.ExpectExec(exact(`DROP TRIGGER IF EXISTS old_name ON "public"."employee"`)).WillReturnResult(ok())
		f.sql.ExpectExec(exact(edited)).WillReturnResult(ok())
		f.sql.ExpectExec(exact(`COMMENT ON TRIGGER log_salary ON "public"."employee" IS '` + automationLabel + `'`)).WillReturnResult(ok())
		f.sql.ExpectRollback()

		out, err := f.svc.Validate(context.Background(), automationSchema, dto.ValidateAutomationRequest{
			ModelID: "m1", ID: automationID, Type: "trigger", Context: edited, Event: "BEFORE DELETE",
		})
		require.NoError(t, err)
		assert.True(t, out.Valid)
	})

	t.Run("checks an edited function as a replacement of the same function", func(t *testing.T) {
		f := newAutomationFixture(t)
		plain := `CREATE FUNCTION log_fn() RETURNS trigger AS $$ BEGIN RETURN NEW; END; $$ LANGUAGE plpgsql`
		f.expectGetByID("function", plain)
		f.expectLabelledFunction("log_fn")
		f.sql.ExpectBegin()
		f.sql.ExpectExec(exact(functionQuery)).WillReturnResult(ok())
		f.sql.ExpectExec(exact(`COMMENT ON FUNCTION log_fn() IS '` + automationLabel + `'`)).WillReturnResult(ok())
		f.sql.ExpectRollback()

		out, err := f.svc.Validate(context.Background(), automationSchema, dto.ValidateAutomationRequest{
			ModelID: "m1", ID: automationID, Type: "function", Context: plain,
		})
		require.NoError(t, err)
		assert.True(t, out.Valid)
	})

	t.Run("rejects a query in the wrong format without touching Postgres", func(t *testing.T) {
		f := newAutomationFixture(t)

		_, err := f.svc.Validate(context.Background(), automationSchema, dto.ValidateAutomationRequest{
			ModelID: "m1", Type: "trigger", Context: "SELECT 1", Event: "AFTER INSERT",
		})
		assert.ErrorIs(t, err, app_errors.InvalidTriggerQuery)
	})

	t.Run("rejects invalid trigger events", func(t *testing.T) {
		f := newAutomationFixture(t)

		_, err := f.svc.Validate(context.Background(), automationSchema, dto.ValidateAutomationRequest{
			ModelID: "m1", Type: "trigger", Context: triggerQuery, Event: "SOMETIMES",
		})
		assert.ErrorIs(t, err, app_errors.InvalidTriggerEvent)
	})

	t.Run("returns not found for an unknown entry", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.table.On("GetTableData", automationsTable, mock.Anything).Return([]map[string]interface{}{}, nil).Once()

		_, err := f.svc.Validate(context.Background(), automationSchema, dto.ValidateAutomationRequest{
			ModelID: "m1", ID: automationID, Type: "trigger", Context: triggerQuery,
		})
		assert.ErrorIs(t, err, app_errors.AutomationNotFound)
	})

	t.Run("fails when the transaction can't start", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.sql.ExpectBegin().WillReturnError(errors.New("connection refused"))

		_, err := f.svc.Validate(context.Background(), automationSchema, dto.ValidateAutomationRequest{
			ModelID: "m1", Type: "function", Context: functionQuery,
		})
		assert.ErrorIs(t, err, app_errors.TriggerFailed)
	})
}

func TestAutomationService_List(t *testing.T) {
	t.Run("returns the automations of a table, newest first", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.table.On("GetTableData", automationsTable, mock.MatchedBy(func(p dbModels.QueryParams) bool {
			return len(p.Filters) == 1 && p.Filters[0].Column == "model_id" && p.Filters[0].Value == "m1" &&
				len(p.OrderBy) == 1 && p.OrderBy[0] == "created_time DESC"
		})).Return([]map[string]interface{}{savedAutomation("trigger", triggerQuery), savedAutomation("webhook", "{}")}, nil)

		out, err := f.svc.List(context.Background(), automationSchema, "m1")
		require.NoError(t, err)
		assert.Len(t, out, 2)
		assert.Equal(t, "webhook", out[1].Type)
	})

	t.Run("fails when the read fails", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.table.On("GetTableData", automationsTable, mock.Anything).Return(nil, errors.New("db down"))

		_, err := f.svc.List(context.Background(), automationSchema, "m1")
		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("fails on rows that do not fit an automation", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.table.On("GetTableData", automationsTable, mock.Anything).Return([]map[string]interface{}{{"id": "not-a-uuid"}}, nil)

		_, err := f.svc.List(context.Background(), automationSchema, "m1")
		assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
	})
}

func TestAutomationService_Delete(t *testing.T) {
	// Trigger entries: Delete drops the trigger from Postgres (removeTriggerFromPostgres)
	t.Run("trigger entry", func(t *testing.T) {
		t.Run("drops the installed trigger and deletes the entry", func(t *testing.T) {
			f := newAutomationFixture(t)
			f.expectGetByID("trigger", triggerQuery)
			f.expectLabelledTrigger("log_salary")
			f.sql.ExpectExec(exact(`DROP TRIGGER IF EXISTS log_salary ON "public"."employee"`)).WillReturnResult(ok())
			f.table.On("DeleteRecord", automationsTable, automationID).Return(nil)

			require.NoError(t, f.svc.Delete(context.Background(), automationSchema, automationID))
		})

		t.Run("also drops the function of an older trigger saved with one", func(t *testing.T) {
			f := newAutomationFixture(t)
			f.expectGetByID("trigger", functionQuery+"; "+triggerQuery)
			f.expectLabelledTrigger("")
			f.sql.ExpectExec(exact(`DROP TRIGGER IF EXISTS log_salary ON "public"."employee"`)).WillReturnResult(ok())
			f.sql.ExpectExec(exact(`DROP FUNCTION IF EXISTS log_fn()`)).WillReturnResult(ok())
			f.table.On("DeleteRecord", automationsTable, automationID).Return(nil)

			require.NoError(t, f.svc.Delete(context.Background(), automationSchema, automationID))
		})

		t.Run("returns the Postgres error when the trigger can't be dropped", func(t *testing.T) {
			f := newAutomationFixture(t)
			f.expectGetByID("trigger", triggerQuery)
			f.expectLabelledTrigger("log_salary")
			f.sql.ExpectExec("^DROP TRIGGER").WillReturnError(errors.New("permission denied"))

			err := f.svc.Delete(context.Background(), automationSchema, automationID)
			assert.ErrorIs(t, err, app_errors.TriggerFailed)
			f.table.AssertNotCalled(t, "DeleteRecord", mock.Anything, mock.Anything)
		})

		t.Run("still deletes the entry when the older trigger's function can't be dropped", func(t *testing.T) {
			f := newAutomationFixture(t)
			f.expectGetByID("trigger", functionQuery+"; "+triggerQuery)
			f.expectLabelledTrigger("log_salary")
			f.sql.ExpectExec(exact(`DROP TRIGGER IF EXISTS log_salary ON "public"."employee"`)).WillReturnResult(ok())
			f.sql.ExpectExec(exact(`DROP FUNCTION IF EXISTS log_fn()`)).WillReturnError(errors.New("other objects depend on it"))
			f.table.On("DeleteRecord", automationsTable, automationID).Return(nil)

			require.NoError(t, f.svc.Delete(context.Background(), automationSchema, automationID))
		})

		t.Run("fails when the table of a trigger can't be loaded", func(t *testing.T) {
			f := newAutomationFixture(t)
			saved := savedAutomation("trigger", triggerQuery)
			saved["model_id"] = "m2"
			f.table.On("GetTableData", automationsTable, mock.Anything).Return([]map[string]interface{}{saved}, nil).Once()
			f.models.On("GetModelByID", automationSchema, "m2").Return(tenant.Model{}, errors.New("model not found"))

			assert.EqualError(t, f.svc.Delete(context.Background(), automationSchema, automationID), "model not found")
			f.table.AssertNotCalled(t, "DeleteRecord", mock.Anything, mock.Anything)
		})

		t.Run("fails when the saved trigger query is not valid", func(t *testing.T) {
			f := newAutomationFixture(t)
			f.expectGetByID("trigger", "SELECT 1")

			assert.Error(t, f.svc.Delete(context.Background(), automationSchema, automationID))
			f.table.AssertNotCalled(t, "DeleteRecord", mock.Anything, mock.Anything)
		})
	})

	// Function entries: Delete drops the function from Postgres (removeFunctionFromPostgres)
	t.Run("function entry", func(t *testing.T) {
		functionsQuery := mock.MatchedBy(func(p dbModels.QueryParams) bool {
			return len(p.Filters) == 1 && p.Filters[0].Column == "type" && p.Filters[0].Value == tenant.AutomationTypeFunction
		})

		t.Run("drops a function no other entry defines", func(t *testing.T) {
			f := newAutomationFixture(t)
			f.expectGetByID("function", functionQuery)
			f.table.On("GetTableData", automationsTable, functionsQuery).Return([]map[string]interface{}{savedAutomation("function", functionQuery)}, nil)
			f.expectLabelledFunction("log_fn")
			f.sql.ExpectExec(exact(`DROP FUNCTION IF EXISTS log_fn()`)).WillReturnResult(ok())
			f.table.On("DeleteRecord", automationsTable, automationID).Return(nil)

			require.NoError(t, f.svc.Delete(context.Background(), automationSchema, automationID))
		})

		t.Run("keeps a function another entry also defines", func(t *testing.T) {
			f := newAutomationFixture(t)
			f.expectGetByID("function", functionQuery)
			other := savedAutomation("function", `CREATE FUNCTION "LOG_FN"() RETURNS trigger AS $$ $$`)
			other["id"] = "11111111-2222-3333-4444-555555555555"
			f.table.On("GetTableData", automationsTable, functionsQuery).Return([]map[string]interface{}{savedAutomation("function", functionQuery), other}, nil)
			f.table.On("DeleteRecord", automationsTable, automationID).Return(nil)

			require.NoError(t, f.svc.Delete(context.Background(), automationSchema, automationID))
		})

		t.Run("refuses while a trigger still uses the function", func(t *testing.T) {
			f := newAutomationFixture(t)
			f.expectGetByID("function", functionQuery)
			f.table.On("GetTableData", automationsTable, functionsQuery).Return([]map[string]interface{}{}, nil)
			f.expectLabelledFunction("")
			f.sql.ExpectExec(exact(`DROP FUNCTION IF EXISTS log_fn()`)).WillReturnError(errors.New("cannot drop function log_fn() because other objects depend on it"))

			err := f.svc.Delete(context.Background(), automationSchema, automationID)
			assert.ErrorIs(t, err, app_errors.TriggerFailed)
			assert.Contains(t, err.Error(), "other objects depend on it")
			f.table.AssertNotCalled(t, "DeleteRecord", mock.Anything, mock.Anything)
		})

		t.Run("fails when the other function entries can't be read", func(t *testing.T) {
			f := newAutomationFixture(t)
			f.expectGetByID("function", functionQuery)
			f.table.On("GetTableData", automationsTable, functionsQuery).Return(nil, errors.New("db down"))

			assert.Error(t, f.svc.Delete(context.Background(), automationSchema, automationID))
			f.table.AssertNotCalled(t, "DeleteRecord", mock.Anything, mock.Anything)
		})

		t.Run("fails when the saved function query names no function", func(t *testing.T) {
			f := newAutomationFixture(t)
			f.expectGetByID("function", "SELECT 1")
			f.table.On("GetTableData", automationsTable, functionsQuery).Return([]map[string]interface{}{}, nil)
			f.expectLabelledFunction("")

			assert.ErrorIs(t, f.svc.Delete(context.Background(), automationSchema, automationID), app_errors.InvalidFunctionQuery)
			f.table.AssertNotCalled(t, "DeleteRecord", mock.Anything, mock.Anything)
		})
	})

	t.Run("deletes a webhook entry only", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.expectGetByID("webhook", "{}")
		f.table.On("DeleteRecord", automationsTable, automationID).Return(nil)

		require.NoError(t, f.svc.Delete(context.Background(), automationSchema, automationID))
	})

	t.Run("fails when deleting the entry fails", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.expectGetByID("webhook", "{}")
		f.table.On("DeleteRecord", automationsTable, automationID).Return(errors.New("db down"))

		assert.ErrorIs(t, f.svc.Delete(context.Background(), automationSchema, automationID), app_errors.DatabaseError)
	})

	t.Run("returns not found for an unknown id", func(t *testing.T) {
		f := newAutomationFixture(t)
		f.table.On("GetTableData", automationsTable, mock.Anything).Return([]map[string]interface{}{}, nil)

		assert.ErrorIs(t, f.svc.Delete(context.Background(), automationSchema, automationID), app_errors.AutomationNotFound)
	})
}
