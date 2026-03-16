package workspace_test

import (
	"context"
	"errors"
	"testing"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	services "github.com/aptlogica/sereni-base/internal/services/workspace"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewWorkspaceMemberService(t *testing.T) {
	db, _ := setupMockDB()

	svc := services.NewWorkspaceMemberService(db)

	assert.NotNil(t, svc)
}

func TestGetAllWorkspaceMembersByUser(t *testing.T) {
	t.Run("fetch error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceMemberService(db)

		mockTable.On("GetTableData", tenant.WorkspaceMember{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.GetAllWorkspaceMembersByUser(context.Background(), "schema", "user")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("not found", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceMemberService(db)

		mockTable.On("GetTableData", tenant.WorkspaceMember{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{}, nil)

		_, err := svc.GetAllWorkspaceMembersByUser(context.Background(), "schema", "user")

		assert.ErrorIs(t, err, app_errors.WorkspaceMemberNotFound)
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceMemberService(db)

		mockTable.On("GetTableData", tenant.WorkspaceMember{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": make(chan int)}}, nil)

		_, err := svc.GetAllWorkspaceMembersByUser(context.Background(), "schema", "user")

		assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceMemberService(db)

		id := uuid.New().String()
		mockTable.On("GetTableData", tenant.WorkspaceMember{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id, "user_id": "u", "workspace_id": "w"}}, nil)

		rows, err := svc.GetAllWorkspaceMembersByUser(context.Background(), "schema", "user")

		assert.NoError(t, err)
		assert.Len(t, rows, 1)
	})
}

func TestGetWorkspaceMemberByUserAndWorkspace(t *testing.T) {
	t.Run("fetch error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceMemberService(db)

		mockTable.On("GetTableData", tenant.WorkspaceMember{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.GetWorkspaceMemberByUserAndWorkspace(context.Background(), "schema", "user", "ws")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("not found", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceMemberService(db)

		mockTable.On("GetTableData", tenant.WorkspaceMember{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{}, nil)

		_, err := svc.GetWorkspaceMemberByUserAndWorkspace(context.Background(), "schema", "user", "ws")

		assert.ErrorIs(t, err, app_errors.WorkspaceMemberNotFound)
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceMemberService(db)

		mockTable.On("GetTableData", tenant.WorkspaceMember{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": make(chan int)}}, nil)

		_, err := svc.GetWorkspaceMemberByUserAndWorkspace(context.Background(), "schema", "user", "ws")

		assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceMemberService(db)

		id := uuid.New().String()
		mockTable.On("GetTableData", tenant.WorkspaceMember{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id, "user_id": "u", "workspace_id": "w"}}, nil)

		member, err := svc.GetWorkspaceMemberByUserAndWorkspace(context.Background(), "schema", "user", "ws")

		assert.NoError(t, err)
		assert.NotNil(t, member)
		assert.Equal(t, id, member.ID.String())
	})
}

func TestDeleteWorkspaceMember(t *testing.T) {
	t.Run("delete error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceMemberService(db)

		mockTable.On("DeleteRecord", tenant.WorkspaceMember{}.TableName("schema"), "id").
			Return(errors.New("db error"))

		err := svc.DeleteWorkspaceMember(context.Background(), "schema", "id")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete workspace member")
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceMemberService(db)

		mockTable.On("DeleteRecord", tenant.WorkspaceMember{}.TableName("schema"), "id").
			Return(nil)

		err := svc.DeleteWorkspaceMember(context.Background(), "schema", "id")

		assert.NoError(t, err)
	})
}

func TestGetWorkspaceMemberByUser(t *testing.T) {
	t.Run("fetch error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceMemberService(db)

		mockTable.On("GetTableData", tenant.WorkspaceMember{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.GetWorkspaceMemberByUser(context.Background(), "schema", "user")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("not found", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceMemberService(db)

		mockTable.On("GetTableData", tenant.WorkspaceMember{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{}, nil)

		_, err := svc.GetWorkspaceMemberByUser(context.Background(), "schema", "user")

		assert.ErrorIs(t, err, app_errors.WorkspaceMemberNotFound)
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceMemberService(db)

		mockTable.On("GetTableData", tenant.WorkspaceMember{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": make(chan int)}}, nil)

		_, err := svc.GetWorkspaceMemberByUser(context.Background(), "schema", "user")

		assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceMemberService(db)

		id := uuid.New().String()
		mockTable.On("GetTableData", tenant.WorkspaceMember{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id, "user_id": "u", "workspace_id": "w"}}, nil)

		rows, err := svc.GetWorkspaceMemberByUser(context.Background(), "schema", "user")

		assert.NoError(t, err)
		assert.Len(t, rows, 1)
	})
}

func TestGetWorkspaceMembersByWorkspace(t *testing.T) {
	t.Run("fetch error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceMemberService(db)

		mockTable.On("GetTableData", tenant.WorkspaceMember{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.GetWorkspaceMembersByWorkspace(context.Background(), "schema", "ws")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("not found", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceMemberService(db)

		mockTable.On("GetTableData", tenant.WorkspaceMember{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{}, nil)

		_, err := svc.GetWorkspaceMembersByWorkspace(context.Background(), "schema", "ws")

		assert.ErrorIs(t, err, app_errors.WorkspaceMemberNotFound)
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceMemberService(db)

		mockTable.On("GetTableData", tenant.WorkspaceMember{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": make(chan int)}}, nil)

		_, err := svc.GetWorkspaceMembersByWorkspace(context.Background(), "schema", "ws")

		assert.ErrorIs(t, err, app_errors.ErrMapToStruct)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceMemberService(db)

		id := uuid.New().String()
		mockTable.On("GetTableData", tenant.WorkspaceMember{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id, "user_id": "u", "workspace_id": "w"}}, nil)

		rows, err := svc.GetWorkspaceMembersByWorkspace(context.Background(), "schema", "ws")

		assert.NoError(t, err)
		assert.Len(t, rows, 1)
	})
}

func TestDeleteUserMappings(t *testing.T) {
	t.Run("get members error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceMemberService(db)

		mockTable.On("GetTableData", tenant.WorkspaceMember{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		err := svc.DeleteUserMappings(context.Background(), "schema", "user")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("delete record error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceMemberService(db)

		id := uuid.New()
		mockTable.On("GetTableData", tenant.WorkspaceMember{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id}}, nil)
		mockTable.On("DeleteRecord", tenant.WorkspaceMember{}.TableName("schema"), id).
			Return(errors.New("db error"))

		err := svc.DeleteUserMappings(context.Background(), "schema", "user")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceMemberService(db)

		id1 := uuid.New()
		id2 := uuid.New()
		mockTable.On("GetTableData", tenant.WorkspaceMember{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id1}, {"id": id2}}, nil)
		mockTable.On("DeleteRecord", tenant.WorkspaceMember{}.TableName("schema"), id1).
			Return(nil).Once()
		mockTable.On("DeleteRecord", tenant.WorkspaceMember{}.TableName("schema"), id2).
			Return(nil).Once()

		err := svc.DeleteUserMappings(context.Background(), "schema", "user")

		assert.NoError(t, err)
		mockTable.AssertExpectations(t)
	})
}

func TestUpdateWorkspaceMemberBases(t *testing.T) {
	t.Run("get member error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceMemberService(db)

		mockTable.On("GetTableData", tenant.WorkspaceMember{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		err := svc.UpdateWorkspaceMemberBases(context.Background(), "schema", "ws", "user", "full_access", "")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("update error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceMemberService(db)

		id := uuid.New()
		mockTable.On("GetTableData", tenant.WorkspaceMember{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id, "user_id": "u", "workspace_id": "w"}}, nil)
		mockTable.On("UpdateRecord", tenant.WorkspaceMember{}.TableName("schema"), id.String(), mock.Anything).
			Return(nil, errors.New("db error"))

		err := svc.UpdateWorkspaceMemberBases(context.Background(), "schema", "ws", "user", "full_access", "")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("full access", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceMemberService(db)

		id := uuid.New()
		mockTable.On("GetTableData", tenant.WorkspaceMember{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id, "user_id": "u", "workspace_id": "w"}}, nil)
		mockTable.On("UpdateRecord", tenant.WorkspaceMember{}.TableName("schema"), id.String(), mock.MatchedBy(func(m map[string]interface{}) bool {
			return m["bases_ids"] == "*" && m["access_level"] == "full_access"
		})).Return(map[string]interface{}{"id": id.String()}, nil)

		err := svc.UpdateWorkspaceMemberBases(context.Background(), "schema", "ws", "user", "full_access", "ignored")

		assert.NoError(t, err)
	})

	t.Run("limited access", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewWorkspaceMemberService(db)

		id := uuid.New()
		mockTable.On("GetTableData", tenant.WorkspaceMember{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id, "user_id": "u", "workspace_id": "w"}}, nil)
		mockTable.On("UpdateRecord", tenant.WorkspaceMember{}.TableName("schema"), id.String(), mock.MatchedBy(func(m map[string]interface{}) bool {
			return m["bases_ids"] == "b1,b2" && m["access_level"] == "limited_access"
		})).Return(map[string]interface{}{"id": id.String()}, nil)

		err := svc.UpdateWorkspaceMemberBases(context.Background(), "schema", "ws", "user", "limited_access", "b1,b2")

		assert.NoError(t, err)
	})
}
