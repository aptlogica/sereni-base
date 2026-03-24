package core_test

import (
	"context"
	"errors"
	"testing"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	services "github.com/aptlogica/sereni-base/internal/services/core"

	dbModels "github.com/aptlogica/go-postgres-rest/pkg/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewOrganizationService(t *testing.T) {
	db, _ := setupMockDB()

	svc := services.NewOrganizationService(db)

	assert.NotNil(t, svc)
}

func TestCreateOrganization(t *testing.T) {
	t.Run("invalid settings", func(t *testing.T) {
		db, _ := setupMockDB()
		svc := services.NewOrganizationService(db)

		bad := map[string]interface{}{"bad": func() {}}
		_, err := svc.CreateOrganization(context.Background(), "schema", dto.CreateOrganizationRequest{
			Name:     "Org",
			Email:    "org@example.com",
			Settings: bad,
		})

		assert.Error(t, err)
	})

	t.Run("invalid meta", func(t *testing.T) {
		db, _ := setupMockDB()
		svc := services.NewOrganizationService(db)

		bad := map[string]interface{}{"bad": func() {} /* intentionally empty for testing invalid metadata */}
		_, err := svc.CreateOrganization(context.Background(), "schema", dto.CreateOrganizationRequest{
			Name:  "Org",
			Email: "org@example.com",
			Meta:  bad,
		})

		assert.Error(t, err)
	})

	t.Run("create error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewOrganizationService(db)

		mockTable.On("CreateRecord", tenant.Organization{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.CreateOrganization(context.Background(), "schema", dto.CreateOrganizationRequest{
			Name:  "Org",
			Email: "org@example.com",
		})

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewOrganizationService(db)

		var captured map[string]interface{}
		mockTable.On("CreateRecord", tenant.Organization{}.TableName("schema"), mock.Anything).
			Run(func(args mock.Arguments) { captured = args.Get(1).(map[string]interface{}) }).
			Return(map[string]interface{}{"id": uuid.New().String()}, nil)

		org, err := svc.CreateOrganization(context.Background(), "schema", dto.CreateOrganizationRequest{
			Name:  "Org",
			Email: "org@example.com",
		})

		assert.NoError(t, err)
		assert.Equal(t, "active", org.Status)
		assert.Equal(t, "Org", captured["name"])
	})
}

func TestGetOrganizationByID(t *testing.T) {
	t.Run("empty id", func(t *testing.T) {
		db, _ := setupMockDB()
		svc := services.NewOrganizationService(db)

		_, err := svc.GetOrganizationByID(context.Background(), "schema", "")

		assert.Error(t, err)
	})

	t.Run("db error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewOrganizationService(db)

		mockTable.On("GetTableData", tenant.Organization{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.GetOrganizationByID(context.Background(), "schema", "id")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("not found", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewOrganizationService(db)

		mockTable.On("GetTableData", tenant.Organization{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{}, nil)

		_, err := svc.GetOrganizationByID(context.Background(), "schema", "id")

		assert.ErrorIs(t, err, app_errors.ErrRecordNotFound)
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewOrganizationService(db)

		mockTable.On("GetTableData", tenant.Organization{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": make(chan int)}}, nil)

		_, err := svc.GetOrganizationByID(context.Background(), "schema", "id")

		assert.ErrorIs(t, err, app_errors.ErrStructToStruct)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewOrganizationService(db)

		id := uuid.New()
		mockTable.On("GetTableData", tenant.Organization{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id.String(), "name": "Org", "email": "org@example.com"}}, nil)

		org, err := svc.GetOrganizationByID(context.Background(), "schema", "id")

		assert.NoError(t, err)
		assert.Equal(t, id, org.ID)
	})
}

func TestGetOrganization(t *testing.T) {
	t.Run("db error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewOrganizationService(db)

		mockTable.On("GetTableData", tenant.Organization{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.GetOrganization(context.Background(), "schema")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("not found", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewOrganizationService(db)

		mockTable.On("GetTableData", tenant.Organization{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{}, nil)

		_, err := svc.GetOrganization(context.Background(), "schema")

		assert.ErrorIs(t, err, app_errors.ErrRecordNotFound)
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewOrganizationService(db)

		mockTable.On("GetTableData", tenant.Organization{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": make(chan int)}}, nil)

		_, err := svc.GetOrganization(context.Background(), "schema")

		assert.ErrorIs(t, err, app_errors.ErrStructToStruct)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewOrganizationService(db)

		id := uuid.New()
		mockTable.On("GetTableData", tenant.Organization{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id.String(), "name": "Org", "email": "org@example.com"}}, nil)

		org, err := svc.GetOrganization(context.Background(), "schema")

		assert.NoError(t, err)
		assert.Equal(t, id, org.ID)
	})
}

func TestUpdateOrganization(t *testing.T) {
	t.Run("get org error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewOrganizationService(db)

		mockTable.On("GetTableData", tenant.Organization{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.UpdateOrganization(context.Background(), "schema", "id", dto.UpdateOrganizationRequest{})

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("invalid settings", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewOrganizationService(db)

		mockTable.On("GetTableData", tenant.Organization{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": uuid.New().String(), "name": "Org", "email": "org@example.com"}}, nil)

		bad := map[string]interface{}{"bad": func() {}}
		_, err := svc.UpdateOrganization(context.Background(), "schema", "id", dto.UpdateOrganizationRequest{Settings: bad})

		assert.Error(t, err)
	})

	t.Run("invalid meta", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewOrganizationService(db)

		mockTable.On("GetTableData", tenant.Organization{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": uuid.New().String(), "name": "Org", "email": "org@example.com"}}, nil)

		bad := map[string]interface{}{"bad": func() {}}
		_, err := svc.UpdateOrganization(context.Background(), "schema", "id", dto.UpdateOrganizationRequest{Meta: bad})

		assert.Error(t, err)
	})

	t.Run("update error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewOrganizationService(db)

		mockTable.On("GetTableData", tenant.Organization{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": uuid.New().String(), "name": "Org", "email": "org@example.com"}}, nil)
		mockTable.On("UpdateRecord", tenant.Organization{}.TableName("schema"), "id", mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.UpdateOrganization(context.Background(), "schema", "id", dto.UpdateOrganizationRequest{})

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewOrganizationService(db)

		mockTable.On("GetTableData", tenant.Organization{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": uuid.New().String(), "name": "Org", "email": "org@example.com"}}, nil)

		var captured map[string]interface{}
		mockTable.On("UpdateRecord", tenant.Organization{}.TableName("schema"), "id", mock.Anything).
			Run(func(args mock.Arguments) { captured = args.Get(2).(map[string]interface{}) }).
			Return(map[string]interface{}{}, nil)

		name := "Updated"
		desc := "Desc"
		email := "new@example.com"
		phone := "123"
		website := "site"
		logo := "logo"
		address := "addr"
		city := "city"
		state := "state"
		country := "country"
		zip := "zip"
		settings := map[string]interface{}{"k": "v"}
		meta := map[string]interface{}{"m": "v"}
		status := "inactive"
		_, err := svc.UpdateOrganization(context.Background(), "schema", "id", dto.UpdateOrganizationRequest{
			Name:        &name,
			Description: &desc,
			Email:       &email,
			Phone:       &phone,
			Website:     &website,
			Logo:        &logo,
			Address:     &address,
			City:        &city,
			State:       &state,
			Country:     &country,
			ZipCode:     &zip,
			Settings:    settings,
			Meta:        meta,
			Status:      &status,
		})

		assert.NoError(t, err)
		assert.Equal(t, "Updated", captured["name"])
		assert.Equal(t, &desc, captured["description"])
		assert.Equal(t, "new@example.com", captured["email"])
		assert.Equal(t, &phone, captured["phone"])
		assert.Equal(t, &website, captured["website"])
		assert.Equal(t, &logo, captured["logo"])
		assert.Equal(t, &address, captured["address"])
		assert.Equal(t, &city, captured["city"])
		assert.Equal(t, &state, captured["state"])
		assert.Equal(t, &country, captured["country"])
		assert.Equal(t, &zip, captured["zip_code"])
		assert.Equal(t, "inactive", captured["status"])
	})
}

func TestDeleteOrganization(t *testing.T) {
	t.Run("delete error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewOrganizationService(db)

		mockTable.On("DeleteRecord", tenant.Organization{}.TableName("schema"), "id").
			Return(errors.New("delete fail"))

		err := svc.DeleteOrganization(context.Background(), "schema", "id")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewOrganizationService(db)

		mockTable.On("DeleteRecord", tenant.Organization{}.TableName("schema"), "id").
			Return(nil)

		err := svc.DeleteOrganization(context.Background(), "schema", "id")

		assert.NoError(t, err)
	})
}

func TestGetOrganizationByEmail(t *testing.T) {
	t.Run("db error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewOrganizationService(db)

		mockTable.On("GetTableData", tenant.Organization{}.TableName("schema"), mock.Anything).
			Return(nil, errors.New("db error"))

		_, err := svc.GetOrganizationByEmail(context.Background(), "schema", "org@example.com")

		assert.ErrorIs(t, err, app_errors.DatabaseError)
	})

	t.Run("not found", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewOrganizationService(db)

		mockTable.On("GetTableData", tenant.Organization{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{}, nil)

		_, err := svc.GetOrganizationByEmail(context.Background(), "schema", "org@example.com")

		assert.ErrorIs(t, err, app_errors.ErrRecordNotFound)
	})

	t.Run("map error", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewOrganizationService(db)

		mockTable.On("GetTableData", tenant.Organization{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": make(chan int)}}, nil)

		_, err := svc.GetOrganizationByEmail(context.Background(), "schema", "org@example.com")

		assert.ErrorIs(t, err, app_errors.ErrStructToStruct)
	})

	t.Run("success", func(t *testing.T) {
		db, mockTable := setupMockDB()
		svc := services.NewOrganizationService(db)

		id := uuid.New()
		mockTable.On("GetTableData", tenant.Organization{}.TableName("schema"), mock.Anything).
			Return([]map[string]interface{}{{"id": id.String(), "name": "Org", "email": "org@example.com"}}, nil)

		org, err := svc.GetOrganizationByEmail(context.Background(), "schema", "org@example.com")

		assert.NoError(t, err)
		assert.Equal(t, id, org.ID)
	})
}

// Ensure mock type satisfies the table interface used by DatabaseService
var _ interface {
	GetTableData(tableName string, params dbModels.QueryParams) ([]map[string]interface{}, error)
	CreateRecord(tableName string, data map[string]interface{}) (map[string]interface{}, error)
	UpdateRecord(tableName string, id interface{}, data map[string]interface{}) (map[string]interface{}, error)
	DeleteRecord(tableName string, id interface{}) error
	GetTables(schema string) ([]dbModels.Table, error)
	CreateTable(req dbModels.CreateTableRequest) error
	AddColumn(tableName string, req dbModels.AddColumnRequest) error
	AlterTable(tableName string, req dbModels.AlterTableRequest) error
	BuildComplexQuery(tableName string, filters map[string]interface{}) (dbModels.QueryParams, error)
	CreateSchema(ctx context.Context, schemaName string) error
	DropTable(ctx context.Context, tableName string) error
	CreateView(ctx context.Context, viewName string, viewSQL string) error
	CreateFunction(ctx context.Context, functionName string, functionSQL string) error
	GetByFunction(ctx context.Context, functionName string, args map[string]interface{}) ([]map[string]interface{}, error)
} = &MockTableService{}
