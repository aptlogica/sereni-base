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

func TestNewResourceService(t *testing.T) {
	db, _ := setupMockDB()

	svc := services.NewResourceService(db)

	assert.NotNil(t, svc)
}

func TestCreateResource(t *testing.T) {
	t.Run("sets id and succeeds", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewResourceService(db)

		var captured map[string]interface{}
		req := dto.ResourceDTO{Code: "workspace"}

		created := map[string]interface{}{
			"id":           uuid.New().String(),
			"code":         "workspace",
			"created_time": time.Now(),
		}

		mockTable.On("CreateRecord", tenant.Resource{}.TableName("schema"), mock.Anything).
			Run(func(args mock.Arguments) {
				captured = args.Get(1).(map[string]interface{})
			}).
			Return(created, nil)

		result, err := svc.CreateResource(context.Background(), "schema", req)

		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, result.ID)
		assert.Equal(t, "workspace", result.Code)
		assert.NotNil(t, captured["id"])
		mockTable.AssertExpectations(t)
	})

	t.Run("create record error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewResourceService(db)

		mockTable.On("CreateRecord", tenant.Resource{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.CreateResource(context.Background(), "schema", dto.ResourceDTO{Code: "workspace"})

		assert.Error(t, err)
		mockTable.AssertExpectations(t)
	})

	t.Run("map to struct error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewResourceService(db)

		bad := map[string]interface{}{"id": make(chan int)}
		mockTable.On("CreateRecord", tenant.Resource{}.TableName("schema"), mock.Anything).
			Return(bad, nil)

		_, err := svc.CreateResource(context.Background(), "schema", dto.ResourceDTO{Code: "workspace"})

		assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
		mockTable.AssertExpectations(t)
	})
}

func TestGetResourceByID(t *testing.T) {
	t.Run("db error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewResourceService(db)

		mockTable.On("GetTableData", tenant.Resource{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.GetResourceByID(context.Background(), "schema", uuid.New())

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("not found", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewResourceService(db)

		mockTable.On("GetTableData", tenant.Resource{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{}, nil)

		_, err := svc.GetResourceByID(context.Background(), "schema", uuid.New())

		assert.ErrorIs(t, err, app_errors.ErrRecordNotFound)
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewResourceService(db)

		mockTable.On("GetTableData", tenant.Resource{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": make(chan int)}}, nil)

		_, err := svc.GetResourceByID(context.Background(), "schema", uuid.New())

		assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewResourceService(db)

		id := uuid.New()
		mockTable.On("GetTableData", tenant.Resource{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id.String(), "code": "workspace"}}, nil)

		result, err := svc.GetResourceByID(context.Background(), "schema", id)

		assert.NoError(t, err)
		assert.Equal(t, id, result.ID)
	})
}

func TestGetResourceByCode(t *testing.T) {
	db, mockTable := setupMockDB()
	svc := services.NewResourceService(db)

	mockTable.On("GetTableData", tenant.Resource{}.TableName("schema"), mock.Anything).
		Return([]map[string]interface{}{{"id": uuid.New().String(), "code": "workspace"}}, nil)

	result, err := svc.GetResourceByCode(context.Background(), "schema", "workspace")

	assert.NoError(t, err)
	assert.Equal(t, "workspace", result.Code)
}

func TestListResources(t *testing.T) {
	t.Run("list error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewResourceService(db)

		mockTable.On("GetTableData", tenant.Resource{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, _, err := svc.ListResources(context.Background(), "schema", 10, 0)

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("count error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewResourceService(db)

		mockTable.On("GetTableData", tenant.Resource{}.TableName("schema"), mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Aggregates) == 0
		})).Return([]map[string]interface{}{}, nil)
		mockTable.On("GetTableData", tenant.Resource{}.TableName("schema"), mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Aggregates) > 0
		})).Return(nil, errors.New("count fail"))

		_, _, err := svc.ListResources(context.Background(), "schema", 10, 0)

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewResourceService(db)

		mockTable.On("GetTableData", tenant.Resource{}.TableName("schema"), mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Aggregates) == 0
		})).Return([]map[string]interface{}{{"id": make(chan int)}}, nil)
		mockTable.On("GetTableData", tenant.Resource{}.TableName("schema"), mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Aggregates) > 0
		})).Return([]map[string]interface{}{{"total": float64(1)}}, nil)

		_, _, err := svc.ListResources(context.Background(), "schema", 10, 0)

		assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewResourceService(db)

		mockTable.On("GetTableData", tenant.Resource{}.TableName("schema"), mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Aggregates) == 0
		})).Return([]map[string]interface{}{{"id": uuid.New().String(), "code": "workspace"}}, nil)
		mockTable.On("GetTableData", tenant.Resource{}.TableName("schema"), mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Aggregates) > 0
		})).Return([]map[string]interface{}{{"total": float64(2)}}, nil)

		resources, count, err := svc.ListResources(context.Background(), "schema", 10, 0)

		assert.NoError(t, err)
		assert.Len(t, resources, 1)
		assert.Equal(t, int64(2), count)
	})
}

func TestUpdateResource(t *testing.T) {
	t.Run("update error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewResourceService(db)

		mockTable.On("UpdateRecord", tenant.Resource{}.TableName("schema"), mock.Anything, mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.UpdateResource(context.Background(), "schema", uuid.New(), dto.ResourceDTO{Code: "workspace"})

		assert.Error(t, err)
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewResourceService(db)

		mockTable.On("UpdateRecord", tenant.Resource{}.TableName("schema"), mock.Anything, mock.Anything).
			Return(map[string]interface{}{"id": make(chan int)}, nil)

		_, err := svc.UpdateResource(context.Background(), "schema", uuid.New(), dto.ResourceDTO{Code: "workspace"})

		assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewResourceService(db)

		id := uuid.New()
		mockTable.On("UpdateRecord", tenant.Resource{}.TableName("schema"), id, mock.Anything).
			Return(map[string]interface{}{"id": id.String(), "code": "workspace"}, nil)

		result, err := svc.UpdateResource(context.Background(), "schema", id, dto.ResourceDTO{Code: "workspace"})

		assert.NoError(t, err)
		assert.Equal(t, id, result.ID)
	})
}

func TestDeleteResource(t *testing.T) {
	t.Run("delete error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewResourceService(db)

		mockTable.On("DeleteRecord", tenant.Resource{}.TableName("schema"), mock.Anything).
			Return(errors.New("delete fail"))

		err := svc.DeleteResource(context.Background(), "schema", uuid.New())

		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewResourceService(db)

		mockTable.On("DeleteRecord", tenant.Resource{}.TableName("schema"), mock.Anything).
			Return(nil)

		err := svc.DeleteResource(context.Background(), "schema", uuid.New())

		assert.NoError(t, err)
	})
}

func TestGetOrCreateResource(t *testing.T) {
	t.Run("existing", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewResourceService(db)

		mockTable.On("GetTableData", tenant.Resource{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": uuid.New().String(), "code": "workspace"}}, nil)

		result, err := svc.GetOrCreateResource(context.Background(), "schema", "workspace", nil)

		assert.NoError(t, err)
		assert.Equal(t, "workspace", result.Code)
	})

	t.Run("create when not found", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewResourceService(db)

		mockTable.On("GetTableData", tenant.Resource{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{}, nil).Once()
		mockTable.On("CreateRecord", tenant.Resource{}.TableName("schema"), mock.Anything).
			Return(map[string]interface{}{"id": uuid.New().String(), "code": "workspace"}, nil)

		result, err := svc.GetOrCreateResource(context.Background(), "schema", "workspace", nil)

		assert.NoError(t, err)
		assert.Equal(t, "workspace", result.Code)
	})
}
