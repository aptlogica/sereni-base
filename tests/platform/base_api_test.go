package platform_test

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/aptlogica/go-postgres-rest/pkg"
	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"
	pgInterfaces "github.com/aptlogica/go-postgres-rest/pkg/services/interfaces"
	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	"github.com/aptlogica/sereni-base/internal/platform"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	schema      = "public"
	baseID = "740da362-0722-41b7-b8cc-3b04bdf8fcbd"
	publicID    = "bs_8f3k2m9qx7"
)

var publicIDPattern = regexp.MustCompile(`^bs_[0-9a-z]{10}$`)

// mockTableService implements the table calls the base API service makes; the embedded
// interface is nil, so any other call panics
type mockTableService struct {
	pgInterfaces.Table
	mock.Mock
}

func (m *mockTableService) GetTableData(tableName string, params dbModels.QueryParams) ([]map[string]interface{}, error) {
	args := m.Called(tableName, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]map[string]interface{}), args.Error(1)
}

func (m *mockTableService) CreateRecord(tableName string, data map[string]interface{}) (map[string]interface{}, error) {
	args := m.Called(tableName, data)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *mockTableService) UpdateByColumns(tableName string, where dbModels.ComplexFilter, data map[string]any) (map[string]interface{}, error) {
	args := m.Called(tableName, where, data)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func setup() (interfaces.BaseAPIService, *mockTableService, string) {
	m := &mockTableService{}
	return platform.NewBaseAPIService(&pkg.DatabaseService{TableService: m}), m, tenant.BaseAPI{}.TableName(schema)
}

func row(enabled bool) map[string]interface{} {
	r := map[string]interface{}{
		"base_id":       baseID,
		"public_id":          publicID,
		"enabled":            enabled,
		"enabled_at":         nil,
		"schema_version":     int64(3),
		"created_time":       time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC),
		"last_modified_time": time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC),
	}
	if enabled {
		r["enabled_at"] = time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	}
	return r
}

// pgError builds the typed error go-postgres-rest returns, wrapped the way TableService wraps it
func pgError(code, constraint string) error {
	return fmt.Errorf("failed to create record: %w", &dbModels.DBError{Code: code, Constraint: constraint, Message: "pq error"})
}

func TestNewPublicIDFormat(t *testing.T) {
	seen := make(map[string]bool, 10000)
	for i := 0; i < 10000; i++ {
		id, err := platform.NewPublicID()
		require.NoError(t, err)
		require.Regexp(t, publicIDPattern, id)
		seen[id] = true
	}
	assert.Len(t, seen, 10000, "10,000 random IDs should not repeat")
}

func TestBasePath(t *testing.T) {
	assert.Equal(t, "/api/v1/bs_8f3k2m9qx7", platform.BasePath(publicID))
}

func TestGetWithoutRowIsDisabled(t *testing.T) {
	svc, m, table := setup()
	m.On("GetTableData", table, mock.Anything).Return([]map[string]interface{}{}, nil)

	out, err := svc.Get(context.Background(), schema, baseID)

	require.NoError(t, err)
	assert.Equal(t, baseID, out.BaseID)
	assert.False(t, out.Enabled)
	assert.Empty(t, out.PublicID)
	assert.Empty(t, out.BasePath)
	assert.Nil(t, out.EnabledAt)
}

func TestGetReturnsRow(t *testing.T) {
	svc, m, table := setup()
	m.On("GetTableData", table, mock.MatchedBy(func(p dbModels.QueryParams) bool {
		return len(p.Filters) == 1 && p.Filters[0].Column == "base_id" && p.Filters[0].Value == baseID
	})).Return([]map[string]interface{}{row(true)}, nil)

	out, err := svc.Get(context.Background(), schema, baseID)

	require.NoError(t, err)
	assert.True(t, out.Enabled)
	assert.Equal(t, publicID, out.PublicID)
	assert.Equal(t, "/api/v1/"+publicID, out.BasePath)
	require.NotNil(t, out.EnabledAt)
}

func TestGetDatabaseError(t *testing.T) {
	svc, m, table := setup()
	m.On("GetTableData", table, mock.Anything).Return(nil, errors.New("connection refused"))

	_, err := svc.Get(context.Background(), schema, baseID)

	assert.ErrorIs(t, err, app_errors.DatabaseError)
}

func TestEnableCreatesRow(t *testing.T) {
	svc, m, table := setup()
	m.On("GetTableData", table, mock.Anything).Return([]map[string]interface{}{}, nil)
	m.On("CreateRecord", table, mock.MatchedBy(func(data map[string]interface{}) bool {
		id, _ := data["public_id"].(string)
		_, hasEnabledAt := data["enabled_at"].(time.Time)
		return data["base_id"] == baseID && data["enabled"] == true && hasEnabledAt && publicIDPattern.MatchString(id)
	})).Return(row(true), nil)

	out, err := svc.Enable(context.Background(), schema, baseID)

	require.NoError(t, err)
	assert.True(t, out.Enabled)
	assert.Equal(t, publicID, out.PublicID)
	assert.Equal(t, "/api/v1/"+publicID, out.BasePath)
	m.AssertNumberOfCalls(t, "CreateRecord", 1)
	m.AssertNotCalled(t, "UpdateByColumns", mock.Anything, mock.Anything, mock.Anything)
}

func TestEnableAlreadyEnabledWritesNothing(t *testing.T) {
	svc, m, table := setup()
	m.On("GetTableData", table, mock.Anything).Return([]map[string]interface{}{row(true)}, nil)

	out, err := svc.Enable(context.Background(), schema, baseID)

	require.NoError(t, err)
	assert.True(t, out.Enabled)
	assert.Equal(t, publicID, out.PublicID)
	m.AssertNotCalled(t, "CreateRecord", mock.Anything, mock.Anything)
	m.AssertNotCalled(t, "UpdateByColumns", mock.Anything, mock.Anything, mock.Anything)
}

func TestEnableTurnsDisabledRowBackOn(t *testing.T) {
	svc, m, table := setup()
	m.On("GetTableData", table, mock.Anything).Return([]map[string]interface{}{row(false)}, nil)
	m.On("UpdateByColumns", table,
		mock.MatchedBy(func(where dbModels.ComplexFilter) bool {
			return len(where.Filters) == 1 && where.Filters[0].Column == "base_id" && where.Filters[0].Value == baseID
		}),
		mock.MatchedBy(func(data map[string]any) bool {
			_, hasEnabledAt := data["enabled_at"].(time.Time)
			_, changesID := data["public_id"]
			return data["enabled"] == true && hasEnabledAt && !changesID
		}),
	).Return(row(true), nil)

	out, err := svc.Enable(context.Background(), schema, baseID)

	require.NoError(t, err)
	assert.True(t, out.Enabled)
	assert.Equal(t, publicID, out.PublicID, "turning the API back on keeps the same public ID")
	m.AssertNotCalled(t, "CreateRecord", mock.Anything, mock.Anything)
}

func TestEnableRetriesWhenPublicIDTaken(t *testing.T) {
	svc, m, table := setup()
	m.On("GetTableData", table, mock.Anything).Return([]map[string]interface{}{}, nil)
	m.On("CreateRecord", table, mock.Anything).Return(nil, pgError("23505", "base_api_public_id_key")).Once()
	m.On("CreateRecord", table, mock.Anything).Return(row(true), nil).Once()

	out, err := svc.Enable(context.Background(), schema, baseID)

	require.NoError(t, err)
	assert.Equal(t, publicID, out.PublicID)
	m.AssertNumberOfCalls(t, "CreateRecord", 2)
}

func TestEnableGivesUpAfterFiveTakenPublicIDs(t *testing.T) {
	svc, m, table := setup()
	m.On("GetTableData", table, mock.Anything).Return([]map[string]interface{}{}, nil)
	m.On("CreateRecord", table, mock.Anything).Return(nil, pgError("23505", "base_api_public_id_key"))

	_, err := svc.Enable(context.Background(), schema, baseID)

	assert.ErrorIs(t, err, app_errors.BaseAPIPublicIDFailed)
	m.AssertNumberOfCalls(t, "CreateRecord", 5)
}

func TestEnableConcurrentRequestCreatedRowFirst(t *testing.T) {
	svc, m, table := setup()
	m.On("GetTableData", table, mock.Anything).Return([]map[string]interface{}{}, nil).Once()
	m.On("CreateRecord", table, mock.Anything).Return(nil, pgError("23505", "base_api_pkey")).Once()
	m.On("GetTableData", table, mock.Anything).Return([]map[string]interface{}{row(true)}, nil).Once()

	out, err := svc.Enable(context.Background(), schema, baseID)

	require.NoError(t, err)
	assert.Equal(t, publicID, out.PublicID, "the row the other request created is returned")
	m.AssertNumberOfCalls(t, "CreateRecord", 1)
}

func TestEnableMissingBase(t *testing.T) {
	svc, m, table := setup()
	m.On("GetTableData", table, mock.Anything).Return([]map[string]interface{}{}, nil)
	m.On("CreateRecord", table, mock.Anything).Return(nil, pgError("23503", "base_api_base_id_fkey"))

	_, err := svc.Enable(context.Background(), schema, baseID)

	assert.ErrorIs(t, err, app_errors.BaseNotFound)
}

func TestEnableOtherDatabaseError(t *testing.T) {
	svc, m, table := setup()
	m.On("GetTableData", table, mock.Anything).Return([]map[string]interface{}{}, nil)
	m.On("CreateRecord", table, mock.Anything).Return(nil, errors.New("connection reset"))

	_, err := svc.Enable(context.Background(), schema, baseID)

	assert.ErrorIs(t, err, app_errors.DatabaseError)
}
