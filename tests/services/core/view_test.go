package core_test

import (
	"context"
	"errors"
	"testing"
	"time"

	app_errors "serenibase/internal/app-errors"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	services "serenibase/internal/services/core"

	dbModels "go-postgres-rest/pkg/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewViewService(t *testing.T) {
	db, _ := setupMockDB()

	svc := services.NewViewService(db)

	assert.NotNil(t, svc)
}

func TestCreateView(t *testing.T) {
	t.Run("create error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewViewService(db)

		mockTable.On("AddColumn", tenant.View{}.TableName("schema"), mock.Anything).
			Return(errors.New("already exists")).Twice()
		mockTable.On("CreateRecord", tenant.View{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.Create(context.Background(), dto.ViewInsertion{ID: uuid.New()}, "schema")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewViewService(db)

		mockTable.On("AddColumn", tenant.View{}.TableName("schema"), mock.Anything).
			Return(nil).Twice()
		mockTable.On("CreateRecord", tenant.View{}.TableName("schema"), mock.Anything).
			Return(map[string]interface{}{"id": make(chan int)}, nil)

		_, err := svc.Create(context.Background(), dto.ViewInsertion{ID: uuid.New()}, "schema")

		assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewViewService(db)

		mockTable.On("AddColumn", tenant.View{}.TableName("schema"), mock.Anything).
			Return(nil).Twice()
		mockTable.On("CreateRecord", tenant.View{}.TableName("schema"), mock.Anything).
			Return(map[string]interface{}{"id": uuid.New().String(), "title": "View"}, nil)

		result, err := svc.Create(context.Background(), dto.ViewInsertion{ID: uuid.New(), Title: "View"}, "schema")

		assert.NoError(t, err)
		assert.Equal(t, "View", result.Title)
	})
}

func TestGetViewByID(t *testing.T) {
	t.Run("db error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewViewService(db)

		mockTable.On("GetTableData", tenant.View{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.GetViewByID(context.Background(), "schema", "id")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("not found", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewViewService(db)

		mockTable.On("GetTableData", tenant.View{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{}, nil)

		_, err := svc.GetViewByID(context.Background(), "schema", "id")

		assert.ErrorIs(t, err, app_errors.ViewNotFound)
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewViewService(db)

		mockTable.On("GetTableData", tenant.View{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": make(chan int)}}, nil)

		_, err := svc.GetViewByID(context.Background(), "schema", "id")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewViewService(db)

		id := uuid.New()
		mockTable.On("GetTableData", tenant.View{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id.String(), "title": "View"}}, nil)

		result, err := svc.GetViewByID(context.Background(), "schema", id.String())

		assert.NoError(t, err)
		assert.Equal(t, id, result.ID)
	})
}

func TestGetAllViews(t *testing.T) {
	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewViewService(db)

		mockTable.On("GetTableData", tenant.View{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": make(chan int)}}, nil)

		_, err := svc.GetAllViews(context.Background(), "schema")

		assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewViewService(db)

		mockTable.On("GetTableData", tenant.View{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": uuid.New().String(), "title": "View"}}, nil)

		views, err := svc.GetAllViews(context.Background(), "schema")

		assert.NoError(t, err)
		assert.Len(t, views, 1)
	})
}

func TestUpdateView(t *testing.T) {
	t.Run("get view error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewViewService(db)

		mockTable.On("AddColumn", tenant.View{}.TableName("schema"), mock.Anything).
			Return(errors.New("already exists")).Twice()
		mockTable.On("GetTableData", tenant.View{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{}, nil)

		_, err := svc.UpdateView(context.Background(), "schema", "id", dto.ViewUpdate{})

		assert.ErrorIs(t, err, app_errors.ViewNotFound)
	})

	t.Run("update record error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewViewService(db)

		id := uuid.New().String()
		mockTable.On("AddColumn", tenant.View{}.TableName("schema"), mock.Anything).
			Return(errors.New("boom")).Twice()
		mockTable.On("GetTableData", tenant.View{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id, "title": "View"}}, nil).Once()
		mockTable.On("UpdateRecord", tenant.View{}.TableName("schema"), id, mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.UpdateView(context.Background(), "schema", id, dto.ViewUpdate{UpdatedAt: time.Now()})

		assert.ErrorIs(t, err, app_errors.ViewUploadFailed)
	})

	t.Run("empty update result", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewViewService(db)

		id := uuid.New().String()
		mockTable.On("AddColumn", tenant.View{}.TableName("schema"), mock.Anything).
			Return(nil).Twice()
		mockTable.On("GetTableData", tenant.View{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id, "title": "View"}}, nil).Once()
		mockTable.On("UpdateRecord", tenant.View{}.TableName("schema"), id, mock.Anything).
			Return(map[string]interface{}{}, nil)

		_, err := svc.UpdateView(context.Background(), "schema", id, dto.ViewUpdate{UpdatedAt: time.Now()})

		assert.ErrorIs(t, err, app_errors.InvalidPayload)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewViewService(db)

		id := uuid.New().String()
		mockTable.On("AddColumn", tenant.View{}.TableName("schema"), mock.Anything).
			Return(nil).Twice()
		mockTable.On("GetTableData", tenant.View{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id, "title": "View"}}, nil).Once()
		mockTable.On("UpdateRecord", tenant.View{}.TableName("schema"), id, mock.Anything).
			Return(map[string]interface{}{"id": id}, nil)
		mockTable.On("GetTableData", tenant.View{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id, "title": "Updated"}}, nil).Once()

		result, err := svc.UpdateView(context.Background(), "schema", id, dto.ViewUpdate{UpdatedAt: time.Now()})

		assert.NoError(t, err)
		assert.Equal(t, "Updated", result.Title)
	})
}

func TestDeleteView(t *testing.T) {
	t.Run("get view error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewViewService(db)

		mockTable.On("GetTableData", tenant.View{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{}, nil)

		err := svc.DeleteView(context.Background(), "schema", "id")

		assert.ErrorIs(t, err, app_errors.ViewNotFound)
	})

	t.Run("delete error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewViewService(db)

		id := uuid.New()
		mockTable.On("GetTableData", tenant.View{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id.String(), "title": "View"}}, nil)
		mockTable.On("DeleteRecord", tenant.View{}.TableName("schema"), id.String()).
			Return(errors.New("delete fail"))

		err := svc.DeleteView(context.Background(), "schema", id.String())

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewViewService(db)

		id := uuid.New()
		mockTable.On("GetTableData", tenant.View{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id.String(), "title": "View"}}, nil)
		mockTable.On("DeleteRecord", tenant.View{}.TableName("schema"), id.String()).
			Return(nil)

		err := svc.DeleteView(context.Background(), "schema", id.String())

		assert.NoError(t, err)
	})
}

func TestGetViewsByModelID(t *testing.T) {
	t.Run("error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewViewService(db)

		mockTable.On("GetTableData", tenant.View{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.GetViewsByModelID(context.Background(), "schema", "model")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewViewService(db)

		mockTable.On("GetTableData", tenant.View{}.TableName("schema"), mock.MatchedBy(func(q dbModels.QueryParams) bool {
			return len(q.Filters) == 1 && q.Filters[0].Column == "model_id"
		})).Return([]map[string]interface{}{{"id": uuid.New().String(), "title": "View"}}, nil)

		views, err := svc.GetViewsByModelID(context.Background(), "schema", "model")

		assert.NoError(t, err)
		assert.Len(t, views, 1)
	})
}
