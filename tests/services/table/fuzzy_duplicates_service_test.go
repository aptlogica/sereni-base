package table_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fuzzyExecCapture struct {
	mu      sync.Mutex
	queries []string
	args    [][]any
}

func (c *fuzzyExecCapture) record(query string, args []any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.queries = append(c.queries, query)
	c.args = append(c.args, args)
}

// captureFuzzyExec makes every DELETE/UPDATE report 1 affected row per bound id.
func captureFuzzyExec(f *enhancementTestFixture) *fuzzyExecCapture {
	capture := &fuzzyExecCapture{}
	f.dbHooks.execFn = func(_ context.Context, query string, args ...any) (int64, error) {
		capture.record(query, args)
		return int64(len(args)), nil
	}
	return capture
}

func fuzzyRequest(f *enhancementTestFixture) dto.FuzzyDuplicatesRequest {
	return dto.FuzzyDuplicatesRequest{
		ModelID:   f.modelID.String(),
		Columns:   []string{f.columns[0].ID.String()},
		Threshold: "medium",
		Duplicate: "remove_row",
		KeepRule:  "keep_first",
	}
}

func fuzzyRows() []map[string]interface{} {
	return []map[string]interface{}{
		{"id": 1, "name": "John Smith"},
		{"id": 2, "name": "Jon Smith"},
		{"id": 3, "name": "Mary Jones"},
		{"id": 4, "name": "mary jones"},
		{"id": 5, "name": "Unique Person"},
	}
}

func TestFuzzyDuplicates_RemoveRow(t *testing.T) {
	f := setupEnhancementFixture(t, nil)
	f.mockTable.On("GetTableData", f.tableName(), mock.Anything).Return(fuzzyRows(), nil)
	capture := captureFuzzyExec(f)

	res, err := f.svc.FuzzyDuplicates(context.Background(), f.schema, fuzzyRequest(f))

	require.NoError(t, err)
	assert.Equal(t, int64(2), res.TotalDuplicateRows)
	assert.Equal(t, int64(2), res.TotalRowsAffected)
	require.Len(t, capture.queries, 1)
	assert.True(t, strings.HasPrefix(capture.queries[0], "DELETE FROM "+f.tableName()))
	assert.ElementsMatch(t, []any{int64(2), int64(4)}, capture.args[0])
}

func TestFuzzyDuplicates_RemoveDuplicatesClearsColumns(t *testing.T) {
	f := setupEnhancementFixture(t, nil)
	f.mockTable.On("GetTableData", f.tableName(), mock.Anything).Return(fuzzyRows(), nil)
	capture := captureFuzzyExec(f)

	req := fuzzyRequest(f)
	req.Duplicate = "remove_duplicates"
	req.Threshold = "high"
	res, err := f.svc.FuzzyDuplicates(context.Background(), f.schema, req)

	require.NoError(t, err)
	assert.Equal(t, int64(2), res.TotalDuplicateRows)
	require.Len(t, capture.queries, 1)
	assert.Contains(t, capture.queries[0], "UPDATE "+f.tableName()+" SET ")
	assert.Contains(t, capture.queries[0], `"name" = NULL`)
}

func TestFuzzyDuplicates_ThresholdLevels(t *testing.T) {
	rows := []map[string]interface{}{
		{"id": 1, "name": "abcdefghij"},
		{"id": 2, "name": "abcdefghxx"}, // 80% similar
		{"id": 3, "name": "abcdefgxxx"}, // 70% of #1
	}
	counts := map[string]int64{}
	for _, threshold := range []string{"low", "medium", "high", "unknown"} {
		f := setupEnhancementFixture(t, nil)
		f.mockTable.On("GetTableData", f.tableName(), mock.Anything).Return(rows, nil)
		captureFuzzyExec(f)
		req := fuzzyRequest(f)
		req.Threshold = threshold
		res, err := f.svc.FuzzyDuplicates(context.Background(), f.schema, req)
		require.NoError(t, err, threshold)
		counts[threshold] = res.TotalDuplicateRows
	}
	// a looser threshold never finds fewer duplicates, and unknown values behave like medium
	assert.LessOrEqual(t, counts["low"], counts["medium"])
	assert.LessOrEqual(t, counts["medium"], counts["high"])
	assert.Equal(t, counts["medium"], counts["unknown"])
	assert.Greater(t, counts["high"], counts["low"])
}

func TestFuzzyDuplicates_ManualMode(t *testing.T) {
	for _, mode := range []string{"manual", ""} {
		t.Run("mode "+mode, func(t *testing.T) {
			f := setupEnhancementFixture(t, nil)
			f.mockTable.On("GetTableData", f.tableName(), mock.Anything).Return(fuzzyRows(), nil)
			capture := captureFuzzyExec(f)

			req := fuzzyRequest(f)
			req.DeduplicationMode = mode
			req.RowActions = map[string]string{"1": "delete", "2": "clear", "3": "keep"}
			res, err := f.svc.FuzzyDuplicates(context.Background(), f.schema, req)

			require.NoError(t, err)
			assert.Equal(t, int64(2), res.TotalDuplicateRows)
			assert.Equal(t, int64(2), res.TotalRowsAffected)
			require.Len(t, capture.queries, 2)
		})
	}
}

func TestFuzzyDuplicates_ManualModeWithoutActionsDoesNothing(t *testing.T) {
	f := setupEnhancementFixture(t, nil)
	f.mockTable.On("GetTableData", f.tableName(), mock.Anything).Return(fuzzyRows(), nil)
	capture := captureFuzzyExec(f)

	req := fuzzyRequest(f)
	req.DeduplicationMode = "manual"
	res, err := f.svc.FuzzyDuplicates(context.Background(), f.schema, req)

	require.NoError(t, err)
	assert.Equal(t, int64(0), res.TotalDuplicateRows)
	assert.Empty(t, capture.queries)
}

func TestFuzzyDuplicates_AutomaticModeIgnoresRowActions(t *testing.T) {
	f := setupEnhancementFixture(t, nil)
	f.mockTable.On("GetTableData", f.tableName(), mock.Anything).Return(fuzzyRows(), nil)
	captureFuzzyExec(f)

	req := fuzzyRequest(f)
	req.DeduplicationMode = "automatic"
	req.RowActions = map[string]string{"5": "delete"}
	res, err := f.svc.FuzzyDuplicates(context.Background(), f.schema, req)

	require.NoError(t, err)
	assert.Equal(t, int64(2), res.TotalDuplicateRows)
}

func TestFuzzyDuplicates_UnknownDuplicateModeFindsNothingToApply(t *testing.T) {
	f := setupEnhancementFixture(t, nil)
	f.mockTable.On("GetTableData", f.tableName(), mock.Anything).Return(fuzzyRows(), nil)
	capture := captureFuzzyExec(f)

	req := fuzzyRequest(f)
	req.Duplicate = "something_else"
	res, err := f.svc.FuzzyDuplicates(context.Background(), f.schema, req)

	require.NoError(t, err)
	assert.Equal(t, int64(0), res.TotalDuplicateRows)
	assert.Empty(t, capture.queries)
}

func TestFuzzyDuplicates_ChunksLargeIDLists(t *testing.T) {
	f := setupEnhancementFixture(t, nil)
	f.mockTable.On("GetTableData", f.tableName(), mock.Anything).Return(fuzzyRows(), nil)
	capture := captureFuzzyExec(f)

	actions := make(map[string]string, 1001)
	for i := 0; i < 1001; i++ {
		actions[uuid.New().String()] = "delete"
	}
	req := fuzzyRequest(f)
	req.DeduplicationMode = "manual"
	req.RowActions = actions
	res, err := f.svc.FuzzyDuplicates(context.Background(), f.schema, req)

	require.NoError(t, err)
	assert.Equal(t, int64(1001), res.TotalRowsAffected)
	require.Len(t, capture.queries, 3) // 500 + 500 + 1
	sizes := []int{len(capture.args[0]), len(capture.args[1]), len(capture.args[2])}
	assert.ElementsMatch(t, []int{500, 500, 1}, sizes)
	assert.Contains(t, capture.queries[0], "$500")
	assert.NotContains(t, capture.queries[0], "$501")
}

func TestFuzzyDuplicates_KeepLatestUpdated(t *testing.T) {
	f := setupEnhancementFixture(t, nil)
	f.mockTable.On("GetTableData", "information_schema.columns", mock.Anything).Return([]map[string]interface{}{
		{"column_name": "last_modified_time"},
	}, nil)
	f.mockTable.On("GetTableData", f.tableName(), mock.Anything).Return([]map[string]interface{}{
		{"id": 1, "name": "same", "last_modified_time": "2024-01-01"},
		{"id": 2, "name": "same", "last_modified_time": "2024-06-01"},
	}, nil)
	capture := captureFuzzyExec(f)

	req := fuzzyRequest(f)
	req.KeepRule = "keep_latest_updated"
	res, err := f.svc.FuzzyDuplicates(context.Background(), f.schema, req)

	require.NoError(t, err)
	assert.Equal(t, int64(1), res.TotalDuplicateRows)
	require.Len(t, capture.queries, 1)
	assert.Equal(t, []any{int64(1)}, capture.args[0])
}

func TestFuzzyDuplicates_KeepLatestUpdatedRequiresTimestampColumn(t *testing.T) {
	f := setupEnhancementFixture(t, nil)
	f.mockTable.On("GetTableData", "information_schema.columns", mock.Anything).Return([]map[string]interface{}{
		{"column_name": "id"},
	}, nil)

	req := fuzzyRequest(f)
	req.KeepRule = "keep_latest_updated"
	_, err := f.svc.FuzzyDuplicates(context.Background(), f.schema, req)

	assert.ErrorIs(t, err, app_errors.InvalidPayload)
	assert.Contains(t, err.Error(), "last_modified_time")
}

func TestFuzzyDuplicates_TimestampInspectionError(t *testing.T) {
	f := setupEnhancementFixture(t, nil)
	f.mockTable.On("GetTableData", "information_schema.columns", mock.Anything).Return(nil, errors.New("boom"))

	req := fuzzyRequest(f)
	req.KeepRule = "keep_latest_updated"
	_, err := f.svc.FuzzyDuplicates(context.Background(), f.schema, req)

	assert.Error(t, err)
}

func TestFuzzyDuplicates_NoRows(t *testing.T) {
	f := setupEnhancementFixture(t, nil)
	f.mockTable.On("GetTableData", f.tableName(), mock.Anything).Return([]map[string]interface{}{}, nil)
	capture := captureFuzzyExec(f)

	res, err := f.svc.FuzzyDuplicates(context.Background(), f.schema, fuzzyRequest(f))

	require.NoError(t, err)
	assert.Equal(t, int64(0), res.TotalRowsAffected)
	assert.Equal(t, int64(0), res.TotalDuplicateRows)
	assert.Empty(t, capture.queries)
}

func TestFuzzyDuplicates_NoDuplicatesFound(t *testing.T) {
	f := setupEnhancementFixture(t, nil)
	f.mockTable.On("GetTableData", f.tableName(), mock.Anything).Return([]map[string]interface{}{
		{"id": 1, "name": "apple"}, {"id": 2, "name": "zucchini"},
	}, nil)
	capture := captureFuzzyExec(f)

	res, err := f.svc.FuzzyDuplicates(context.Background(), f.schema, fuzzyRequest(f))

	require.NoError(t, err)
	assert.Equal(t, int64(0), res.TotalDuplicateRows)
	assert.Empty(t, capture.queries)
}

func TestFuzzyDuplicates_LoadErrors(t *testing.T) {
	t.Run("model lookup fails", func(t *testing.T) {
		f := setupEnhancementFixture(t, nil)
		req := fuzzyRequest(f)
		f.mockModel.ExpectedCalls = nil
		f.mockModel.On("GetModelByID", mock.Anything, f.schema, f.modelID.String()).Return(tenant.Model{}, app_errors.TableNotFound)

		_, err := f.svc.FuzzyDuplicates(context.Background(), f.schema, req)
		assert.ErrorIs(t, err, app_errors.TableNotFound)
	})

	t.Run("columns lookup fails", func(t *testing.T) {
		f := setupEnhancementFixture(t, nil)
		req := fuzzyRequest(f)
		f.mockColumn.ExpectedCalls = nil
		f.mockColumn.On("GetColumnByModelID", mock.Anything, f.schema, f.modelID.String()).Return([]tenant.Column(nil), errors.New("columns down"))

		_, err := f.svc.FuzzyDuplicates(context.Background(), f.schema, req)
		assert.Error(t, err)
	})

	t.Run("selected column does not exist", func(t *testing.T) {
		f := setupEnhancementFixture(t, nil)
		req := fuzzyRequest(f)
		req.Columns = []string{uuid.New().String()}

		_, err := f.svc.FuzzyDuplicates(context.Background(), f.schema, req)
		assert.ErrorIs(t, err, app_errors.ColumnNotFound)
	})

	t.Run("row fetch fails", func(t *testing.T) {
		f := setupEnhancementFixture(t, nil)
		f.mockTable.On("GetTableData", f.tableName(), mock.Anything).Return(nil, errors.New("select failed"))

		_, err := f.svc.FuzzyDuplicates(context.Background(), f.schema, fuzzyRequest(f))
		assert.Error(t, err)
	})
}

func TestFuzzyDuplicates_ExecFailures(t *testing.T) {
	t.Run("delete fails", func(t *testing.T) {
		f := setupEnhancementFixture(t, nil)
		f.mockTable.On("GetTableData", f.tableName(), mock.Anything).Return(fuzzyRows(), nil)
		f.dbHooks.execFn = func(context.Context, string, ...any) (int64, error) {
			return 0, errors.New("delete failed")
		}

		_, err := f.svc.FuzzyDuplicates(context.Background(), f.schema, fuzzyRequest(f))
		assert.Error(t, err)
	})

	t.Run("clear fails after delete succeeds", func(t *testing.T) {
		f := setupEnhancementFixture(t, nil)
		f.mockTable.On("GetTableData", f.tableName(), mock.Anything).Return(fuzzyRows(), nil)
		f.dbHooks.execFn = func(_ context.Context, query string, args ...any) (int64, error) {
			if strings.HasPrefix(query, "UPDATE") {
				return 0, errors.New("update failed")
			}
			return int64(len(args)), nil
		}

		req := fuzzyRequest(f)
		req.DeduplicationMode = "manual"
		req.RowActions = map[string]string{"1": "delete", "2": "clear"}
		_, err := f.svc.FuzzyDuplicates(context.Background(), f.schema, req)
		assert.Error(t, err)
	})
}
