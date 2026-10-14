package table_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/aptlogica/sereni-base/internal/utils/helpers"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var guardConstraintsTable = tenant.TableConstraint{}.TableName("schema")

func uniqueKeyRow(modelID string, columnIDs []string) map[string]interface{} {
	ids := make([]interface{}, len(columnIDs))
	for i, id := range columnIDs {
		ids[i] = id
	}
	return map[string]interface{}{
		"id": uuid.NewString(), "model_id": modelID, "type": "unique", "title": "Unique email",
		"constraint_name": "uq_tbl_1a2b3c4d", "column_ids": ids,
	}
}

func detailedMessage(t *testing.T, err error) string {
	t.Helper()
	var detailed *app_errors.DetailedError
	require.ErrorAs(t, err, &detailed)
	return detailed.Message
}

// setupColumnDelete mocks a successful column delete of "email" on table "tbl"
func setupColumnDelete(t *testing.T) (tenant.Column, *MockTableService, *MockColumnService, interfaces.TableManagementService) {
	t.Helper()
	_, mockTable, _, mockModel, mockColumn, _, _, _, svc := setupTableManagementService()

	modelID := uuid.New().String()
	order := 1.0
	col := tenant.Column{ID: uuid.New(), ModelID: modelID, BaseID: uuid.NewString(), ColumnName: "email", Title: "Email", UIDT: "email", DT: helpers.StringPtr("TEXT"), OrderIndex: &order}
	mockColumn.On("GetColumnByID", mock.Anything, "schema", "cid").Return(col, nil)
	mockColumn.On("DeleteColumn", mock.Anything, "schema", "cid").Return(nil)
	mockColumn.On("GetColumnByModelID", mock.Anything, "schema", modelID).Return([]tenant.Column{}, nil)
	mockModel.On("GetModelByID", mock.Anything, "schema", modelID).Return(tenant.Model{Alias: "tbl", ID: uuid.MustParse(modelID)}, nil)
	mockTable.On("GetByFunction", mock.Anything, "public.reorder_columns_after_delete", mock.Anything).Return([]map[string]interface{}{}, nil)
	mockTable.On("AlterTable", mock.Anything, mock.Anything).Return(nil)
	return col, mockTable, mockColumn, svc
}

func TestDeleteColumn_RemovesItsUniqueKeys(t *testing.T) {
	col, mockTable, mockColumn, svc := setupColumnDelete(t)

	usesColumn := uniqueKeyRow(col.ModelID, []string{uuid.NewString(), col.ID.String()})
	alsoUsesColumn := uniqueKeyRow(col.ModelID, []string{col.ID.String()})
	other := uniqueKeyRow(col.ModelID, []string{uuid.NewString()})
	mockTable.On("GetTableData", guardConstraintsTable, mock.MatchedBy(func(p dbModels.QueryParams) bool {
		return p.Filters[0].Column == "model_id" && p.Filters[0].Value == col.ModelID
	})).Return([]map[string]interface{}{usesColumn, other, alsoUsesColumn}, nil)
	mockTable.On("DeleteRecord", guardConstraintsTable, mock.Anything).Return(nil)

	err := svc.DeleteColumn(context.Background(), "schema", "cid")

	require.NoError(t, err)
	// The column is deleted, and dropped with CASCADE so Postgres drops the keys with it
	mockColumn.AssertCalled(t, "DeleteColumn", mock.Anything, "schema", "cid")
	mockTable.AssertCalled(t, "AlterTable", mock.Anything, mock.MatchedBy(func(req dbModels.AlterTableRequest) bool {
		drop, ok := req.Data.(dbModels.DropColumnRequest)
		return req.Action == "drop_column" && ok && drop.Cascade
	}))
	// Only the entries of keys that used the column are removed
	mockTable.AssertCalled(t, "DeleteRecord", guardConstraintsTable, usesColumn["id"])
	mockTable.AssertCalled(t, "DeleteRecord", guardConstraintsTable, alsoUsesColumn["id"])
	mockTable.AssertNotCalled(t, "DeleteRecord", guardConstraintsTable, other["id"])
	mockTable.AssertNumberOfCalls(t, "DeleteRecord", 2)
}

func TestDeleteColumn_UniqueKeyCleanupFailureDoesNotFailTheDelete(t *testing.T) {
	t.Run("reading the keys fails", func(t *testing.T) {
		col, mockTable, mockColumn, svc := setupColumnDelete(t)
		mockTable.On("GetTableData", guardConstraintsTable, mock.Anything).Return(nil, errors.New("db down"))

		require.NoError(t, svc.DeleteColumn(context.Background(), "schema", "cid"))
		mockColumn.AssertCalled(t, "DeleteColumn", mock.Anything, "schema", "cid")
		assert.NotEmpty(t, col.ID)
	})

	t.Run("removing an entry fails", func(t *testing.T) {
		col, mockTable, _, svc := setupColumnDelete(t)
		mockTable.On("GetTableData", guardConstraintsTable, mock.Anything).
			Return([]map[string]interface{}{uniqueKeyRow(col.ModelID, []string{col.ID.String()})}, nil)
		mockTable.On("DeleteRecord", guardConstraintsTable, mock.Anything).Return(errors.New("db down"))

		require.NoError(t, svc.DeleteColumn(context.Background(), "schema", "cid"))
	})
}

func TestDeleteColumn_KeepsKeysWhenTheDeleteFails(t *testing.T) {
	_, mockTable, _, mockModel, mockColumn, _, _, _, svc := setupTableManagementService()
	col := tenant.Column{ID: uuid.New(), ModelID: uuid.New().String(), BaseID: uuid.NewString(), ColumnName: "email", UIDT: "email", DT: helpers.StringPtr("TEXT")}
	mockModel.On("GetModelByID", mock.Anything, "schema", col.ModelID).Return(tenant.Model{Alias: "tbl", ID: uuid.MustParse(col.ModelID)}, nil)
	mockColumn.On("GetColumnByID", mock.Anything, "schema", "cid").Return(col, nil)
	mockColumn.On("GetColumnByModelID", mock.Anything, "schema", col.ModelID).Return([]tenant.Column{}, nil)
	mockColumn.On("DeleteColumn", mock.Anything, "schema", "cid").Return(errors.New("db down"))

	assert.Error(t, svc.DeleteColumn(context.Background(), "schema", "cid"))
	mockTable.AssertNotCalled(t, "GetTableData", guardConstraintsTable, mock.Anything)
	mockTable.AssertNotCalled(t, "DeleteRecord", mock.Anything, mock.Anything)
}

func TestUpdateColumn_TypeChangeBlockedByUniqueKey(t *testing.T) {
	_, mockTable, _, _, mockColumn, _, _, _, svc := setupTableManagementService()

	modelID := uuid.New().String()
	col := tenant.Column{ID: uuid.New(), ModelID: modelID, BaseID: uuid.NewString(), ColumnName: "email", Title: "Email", UIDT: "email", DT: helpers.StringPtr("TEXT")}
	mockColumn.On("GetColumnByID", mock.Anything, "schema", "cid").Return(col, nil)
	mockTable.On("GetTableData", guardConstraintsTable, mock.Anything).
		Return([]map[string]interface{}{uniqueKeyRow(modelID, []string{col.ID.String()})}, nil)

	attachment := "attachment"
	_, err := svc.UpdateColumn(context.Background(), "schema", "cid", dto.ColumnUpdate{UIDT: &attachment})

	assert.ErrorIs(t, err, app_errors.ColumnUsedInConstraint)
	assert.Contains(t, detailedMessage(t, err), "before changing its type")
	mockColumn.AssertNotCalled(t, "UpdateColumn", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// pqDuplicate is how Postgres reports a record that repeats a unique key, wrapped as the db layer does
func pqDuplicate(constraint, detail string) error {
	return fmt.Errorf("failed to insert record into table x: %w", &pq.Error{
		Code: "23505", Constraint: constraint, Detail: detail,
		Message: fmt.Sprintf("duplicate key value violates unique constraint %q", constraint),
	})
}

func TestRecordWrites_DuplicateUniqueKey(t *testing.T) {
	modelID := uuid.New().String()
	emailCol := tenant.Column{ID: uuid.New(), ModelID: modelID, ColumnName: "email", Title: "Email"}
	codeCol := tenant.Column{ID: uuid.New(), ModelID: modelID, ColumnName: "emp_code", Title: "Emp Code"}
	byName := mock.MatchedBy(func(p dbModels.QueryParams) bool {
		return p.Filters[0].Column == "constraint_name" && p.Filters[0].Value == "uq_tbl_1a2b3c4d"
	})

	t.Run("names the key and its columns by title, in key order", func(t *testing.T) {
		_, mockTable, _, _, mockColumn, _, _, _, svc := setupTableManagementService()
		mockTable.On("CreateRecord", mock.Anything, mock.Anything).
			Return(nil, pqDuplicate("uq_tbl_1a2b3c4d", "Key (emp_code, email)=(E01, a@x.com) already exists."))
		mockTable.On("GetTableData", guardConstraintsTable, byName).
			Return([]map[string]interface{}{uniqueKeyRow(modelID, []string{codeCol.ID.String(), emailCol.ID.String()})}, nil)
		mockColumn.On("GetColumnByModelID", mock.Anything, "schema", modelID).Return([]tenant.Column{emailCol, codeCol}, nil)

		_, err := svc.CreateRowWithRecords(context.Background(), "schema", "tbl", map[string]interface{}{"email": "a@x.com"})

		assert.ErrorIs(t, err, app_errors.UniqueKeyViolation)
		assert.Equal(t, `Duplicate value not allowed. The unique key "Unique email" does not allow two records with the same Emp Code and Email.`, detailedMessage(t, err))
	})

	t.Run("names the key and column when Postgres gives no detail", func(t *testing.T) {
		_, mockTable, _, _, mockColumn, _, _, _, svc := setupTableManagementService()
		mockTable.On("CreateRecord", mock.Anything, mock.Anything).
			Return(nil, errors.New(`pq: duplicate key value violates unique constraint "uq_tbl_1a2b3c4d"`))
		mockTable.On("GetTableData", guardConstraintsTable, byName).
			Return([]map[string]interface{}{uniqueKeyRow(modelID, []string{emailCol.ID.String()})}, nil)
		mockColumn.On("GetColumnByModelID", mock.Anything, "schema", modelID).Return([]tenant.Column{emailCol}, nil)

		_, err := svc.CreateRowWithRecords(context.Background(), "schema", "tbl", map[string]interface{}{})
		assert.Equal(t, `Duplicate value not allowed. The unique key "Unique email" does not allow two records with the same Email.`, detailedMessage(t, err))
	})

	t.Run("uses column names for keys added outside the app", func(t *testing.T) {
		_, mockTable, _, _, _, _, _, _, svc := setupTableManagementService()
		mockTable.On("CreateRecord", mock.Anything, mock.Anything).
			Return(nil, pqDuplicate("employee_badge_key", `Key (badge)=(B7) already exists.`))
		mockTable.On("GetTableData", guardConstraintsTable, mock.Anything).Return([]map[string]interface{}{}, nil)

		_, err := svc.CreateRowWithRecords(context.Background(), "schema", "tbl", map[string]interface{}{})
		assert.Equal(t, "Duplicate value not allowed. Two records can't have the same badge.", detailedMessage(t, err))
	})

	t.Run("reports duplicates from a bulk insert", func(t *testing.T) {
		_, mockTable, mockBulk, _, _, _, _, _, svc := setupTableManagementService()
		mockBulk.On("BulkInsert", mock.Anything, mock.Anything).
			Return(nil, pqDuplicate("uq_tbl_1a2b3c4d", ""))
		mockTable.On("GetTableData", guardConstraintsTable, mock.Anything).Return([]map[string]interface{}{}, nil)

		_, err := svc.CreateRowsWithRecordsBulk(context.Background(), "schema", "tbl", []map[string]interface{}{{}, {}})
		assert.ErrorIs(t, err, app_errors.UniqueKeyViolation)
		assert.Equal(t, "Duplicate value not allowed. This record repeats the values of a unique key.", detailedMessage(t, err))
	})

	t.Run("keeps other database errors generic", func(t *testing.T) {
		_, mockTable, _, _, _, _, _, _, svc := setupTableManagementService()
		mockTable.On("CreateRecord", mock.Anything, mock.Anything).
			Return(nil, &pq.Error{Code: "23502", Column: "notes", Message: "null value in column violates not-null constraint"})
		// An empty field outside the primary key is not a key error
		mockTable.On("GetByFunction", mock.Anything, "public.link_key_columns", mock.Anything).
			Return([]map[string]interface{}{{"link_key_columns": "{id}"}}, nil)

		_, err := svc.CreateRowWithRecords(context.Background(), "schema", "tbl", map[string]interface{}{})
		assert.ErrorIs(t, err, app_errors.DatabaseError)
		mockTable.AssertNotCalled(t, "GetTableData", mock.Anything, mock.Anything)
	})
}
