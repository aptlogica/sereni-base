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

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewRelationshipService(t *testing.T) {
	db, _ := setupMockDB()

	svc := services.NewRelationshipService(db)

	assert.NotNil(t, svc)
}

func TestCreateRelation(t *testing.T) {
	t.Run("create error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewRelationshipService(db)

		mockTable.On("CreateRecord", tenant.Relation{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.Create(context.Background(), dto.RelationInsertion{ID: uuid.New()}, "schema")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewRelationshipService(db)

		mockTable.On("CreateRecord", tenant.Relation{}.TableName("schema"), mock.Anything).
			Return(map[string]interface{}{"id": make(chan int)}, nil)

		_, err := svc.Create(context.Background(), dto.RelationInsertion{ID: uuid.New()}, "schema")

		assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewRelationshipService(db)

		id := uuid.New()
		mockTable.On("CreateRecord", tenant.Relation{}.TableName("schema"), mock.Anything).
			Return(map[string]interface{}{"id": id.String(), "relation_type": "one"}, nil)

		result, err := svc.Create(context.Background(), dto.RelationInsertion{ID: id, RelationType: "one"}, "schema")

		assert.NoError(t, err)
		assert.Equal(t, id, result.ID)
	})
}

func TestGetRelationByID(t *testing.T) {
	t.Run("db error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewRelationshipService(db)

		mockTable.On("GetTableData", tenant.Relation{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.GetRelationByID(context.Background(), "id", "schema")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("empty data", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewRelationshipService(db)

		mockTable.On("GetTableData", tenant.Relation{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{}, nil)

		_, err := svc.GetRelationByID(context.Background(), "id", "schema")

		assert.ErrorIs(t, err, app_errors.InvalidPayload)
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewRelationshipService(db)

		mockTable.On("GetTableData", tenant.Relation{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": make(chan int)}}, nil)

		_, err := svc.GetRelationByID(context.Background(), "id", "schema")

		assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewRelationshipService(db)

		id := uuid.New()
		mockTable.On("GetTableData", tenant.Relation{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id.String(), "relation_type": "one"}}, nil)

		result, err := svc.GetRelationByID(context.Background(), "id", "schema")

		assert.NoError(t, err)
		assert.Equal(t, id, result.ID)
	})
}

func TestDeleteRelation(t *testing.T) {
	t.Run("delete error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewRelationshipService(db)

		mockTable.On("DeleteRecord", tenant.Relation{}.TableName("schema"), "rel").
			Return(errors.New("delete fail"))

		err := svc.DeleteRelation(context.Background(), "rel", "schema")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewRelationshipService(db)

		mockTable.On("DeleteRecord", tenant.Relation{}.TableName("schema"), "rel").
			Return(nil)

		err := svc.DeleteRelation(context.Background(), "rel", "schema")

		assert.NoError(t, err)
	})
}

func TestUpdateRelation(t *testing.T) {
	t.Run("update error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewRelationshipService(db)

		mockTable.On("UpdateRecord", tenant.Relation{}.TableName("schema"), "rel", mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.UpdateRelation(context.Background(), "rel", dto.RelationUpdate{UpdatedAt: time.Now()}, "schema")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewRelationshipService(db)

		mockTable.On("UpdateRecord", tenant.Relation{}.TableName("schema"), "rel", mock.Anything).
			Return(map[string]interface{}{"id": make(chan int)}, nil)

		_, err := svc.UpdateRelation(context.Background(), "rel", dto.RelationUpdate{UpdatedAt: time.Now()}, "schema")

		assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewRelationshipService(db)

		id := uuid.New()
		mockTable.On("UpdateRecord", tenant.Relation{}.TableName("schema"), "rel", mock.Anything).
			Return(map[string]interface{}{"id": id.String(), "relation_type": "one"}, nil)

		result, err := svc.UpdateRelation(context.Background(), "rel", dto.RelationUpdate{UpdatedAt: time.Now()}, "schema")

		assert.NoError(t, err)
		assert.Equal(t, id, result.ID)
	})
}
