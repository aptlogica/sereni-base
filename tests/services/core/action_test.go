package core_test

import (
	"context"
	"errors"
	"testing"
	"time"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	services "github.com/aptlogica/sereni-base/internal/services/core"

	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewActionService(t *testing.T) {
	db, _ := setupMockDB()

	svc := services.NewActionService(db)

	assert.NotNil(t, svc)
}

func TestCreateAction(t *testing.T) {
	t.Run("sets id and succeeds", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewActionService(db)

		var captured map[string]interface{}
		req := dto.ActionDTO{Code: "read"}

		created := map[string]interface{}{
			"id":           uuid.New().String(),
			"code":         "read",
			"created_time": time.Now(),
		}

		mockTable.On("CreateRecord", tenant.Action{}.TableName("schema"), mock.Anything).
			Run(func(args mock.Arguments) {
				captured = args.Get(1).(map[string]interface{})
			}).
			Return(created, nil)

		result, err := svc.CreateAction(context.Background(), "schema", req)

		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, result.ID)
		assert.Equal(t, "read", result.Code)
		assert.NotNil(t, captured["id"])
		mockTable.AssertExpectations(t)
	})

	t.Run("create record error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewActionService(db)

		mockTable.On("CreateRecord", tenant.Action{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.CreateAction(context.Background(), "schema", dto.ActionDTO{Code: "read"})

		assert.Error(t, err)
		mockTable.AssertExpectations(t)
	})

	t.Run("map to struct error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewActionService(db)

		bad := map[string]interface{}{"id": make(chan int)}
		mockTable.On("CreateRecord", tenant.Action{}.TableName("schema"), mock.Anything).
			Return(bad, nil)

		_, err := svc.CreateAction(context.Background(), "schema", dto.ActionDTO{Code: "read"})

		assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
		mockTable.AssertExpectations(t)
	})
}

func TestGetActionByID(t *testing.T) {
	t.Run("db error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewActionService(db)

		mockTable.On("GetTableData", tenant.Action{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.GetActionByID(context.Background(), "schema", uuid.New())

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("not found", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewActionService(db)

		mockTable.On("GetTableData", tenant.Action{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{}, nil)

		_, err := svc.GetActionByID(context.Background(), "schema", uuid.New())

		assert.ErrorIs(t, err, app_errors.ErrRecordNotFound)
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewActionService(db)

		mockTable.On("GetTableData", tenant.Action{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": make(chan int)}}, nil)

		_, err := svc.GetActionByID(context.Background(), "schema", uuid.New())

		assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewActionService(db)

		id := uuid.New()
		mockTable.On("GetTableData", tenant.Action{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id.String(), "code": "read"}}, nil)

		result, err := svc.GetActionByID(context.Background(), "schema", id)

		assert.NoError(t, err)
		assert.Equal(t, id, result.ID)
	})
}

func TestGetActionByCode(t *testing.T) {
	db, mockTable := setupMockDB()
	svc := services.NewActionService(db)

	mockTable.On("GetTableData", tenant.Action{}.TableName("schema"), mock.Anything).
		Return([]map[string]interface{}{{"id": uuid.New().String(), "code": "read"}}, nil)

	result, err := svc.GetActionByCode(context.Background(), "schema", "read")

	assert.NoError(t, err)
	assert.Equal(t, "read", result.Code)
}

func TestListActions(t *testing.T) {
	t.Run("list error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewActionService(db)

		mockTable.On("GetTableData", tenant.Action{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, _, err := svc.ListActions(context.Background(), "schema", 10, 0)

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("count error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewActionService(db)

		mockTable.On("GetTableData", tenant.Action{}.TableName("schema"), mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Aggregates) == 0
		})).Return([]map[string]interface{}{}, nil)
		mockTable.On("GetTableData", tenant.Action{}.TableName("schema"), mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Aggregates) > 0
		})).Return(nil, errors.New("count fail"))

		_, _, err := svc.ListActions(context.Background(), "schema", 10, 0)

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewActionService(db)

		mockTable.On("GetTableData", tenant.Action{}.TableName("schema"), mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Aggregates) == 0
		})).Return([]map[string]interface{}{{"id": make(chan int)}}, nil)
		mockTable.On("GetTableData", tenant.Action{}.TableName("schema"), mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Aggregates) > 0
		})).Return([]map[string]interface{}{{"total": float64(1)}}, nil)

		_, _, err := svc.ListActions(context.Background(), "schema", 10, 0)

		assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewActionService(db)

		mockTable.On("GetTableData", tenant.Action{}.TableName("schema"), mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Aggregates) == 0
		})).Return([]map[string]interface{}{{"id": uuid.New().String(), "code": "read"}}, nil)
		mockTable.On("GetTableData", tenant.Action{}.TableName("schema"), mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Aggregates) > 0
		})).Return([]map[string]interface{}{{"total": float64(2)}}, nil)

		actions, count, err := svc.ListActions(context.Background(), "schema", 10, 0)

		assert.NoError(t, err)
		assert.Len(t, actions, 1)
		assert.Equal(t, int64(2), count)
	})
}

func TestUpdateAction(t *testing.T) {
	t.Run("update error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewActionService(db)

		mockTable.On("UpdateRecord", tenant.Action{}.TableName("schema"), mock.Anything, mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.UpdateAction(context.Background(), "schema", uuid.New(), dto.ActionDTO{Code: "read"})

		assert.Error(t, err)
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewActionService(db)

		mockTable.On("UpdateRecord", tenant.Action{}.TableName("schema"), mock.Anything, mock.Anything).
			Return(map[string]interface{}{"id": make(chan int)}, nil)

		_, err := svc.UpdateAction(context.Background(), "schema", uuid.New(), dto.ActionDTO{Code: "read"})

		assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewActionService(db)

		id := uuid.New()
		mockTable.On("UpdateRecord", tenant.Action{}.TableName("schema"), id, mock.Anything).
			Return(map[string]interface{}{"id": id.String(), "code": "read"}, nil)

		result, err := svc.UpdateAction(context.Background(), "schema", id, dto.ActionDTO{Code: "read"})

		assert.NoError(t, err)
		assert.Equal(t, id, result.ID)
	})
}

func TestDeleteAction(t *testing.T) {
	t.Run("delete error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewActionService(db)

		mockTable.On("DeleteRecord", tenant.Action{}.TableName("schema"), mock.Anything).
			Return(errors.New("delete fail"))

		err := svc.DeleteAction(context.Background(), "schema", uuid.New())

		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewActionService(db)

		mockTable.On("DeleteRecord", tenant.Action{}.TableName("schema"), mock.Anything).
			Return(nil)

		err := svc.DeleteAction(context.Background(), "schema", uuid.New())

		assert.NoError(t, err)
	})
}

func TestGetOrCreateAction(t *testing.T) {
	t.Run("existing", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewActionService(db)

		mockTable.On("GetTableData", tenant.Action{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": uuid.New().String(), "code": "read"}}, nil)

		result, err := svc.GetOrCreateAction(context.Background(), "schema", "read", nil)

		assert.NoError(t, err)
		assert.Equal(t, "read", result.Code)
	})

	t.Run("create when not found", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewActionService(db)

		mockTable.On("GetTableData", tenant.Action{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{}, nil).Once()
		mockTable.On("CreateRecord", tenant.Action{}.TableName("schema"), mock.Anything).
			Return(map[string]interface{}{"id": uuid.New().String(), "code": "read"}, nil)

		result, err := svc.GetOrCreateAction(context.Background(), "schema", "read", nil)

		assert.NoError(t, err)
		assert.Equal(t, "read", result.Code)
	})
}
