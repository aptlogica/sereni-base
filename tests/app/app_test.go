package tests

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"github.com/aptlogica/go-postgres-rest/pkg"
	"github.com/aptlogica/go-postgres-rest/pkg/config"
	"github.com/aptlogica/go-postgres-rest/pkg/database"
	"github.com/aptlogica/go-postgres-rest/pkg/database/interfaces"
	"github.com/aptlogica/go-postgres-rest/pkg/models"

	appPkg "github.com/aptlogica/sereni-base/internal/app"
	appConfig "github.com/aptlogica/sereni-base/internal/config"
)

// ---- fakes for go-postgres-rest ----

type fakeDB struct{}

func (f *fakeDB) Exec(query string, args ...any) (sql.Result, error) { return nil, errors.New("fake") }
func (f *fakeDB) Query(query string, args ...any) (*sql.Rows, error) { return nil, errors.New("fake") }
func (f *fakeDB) QueryRow(query string, args ...any) *sql.Row        { return &sql.Row{} }
func (f *fakeDB) Close() error                                       { return nil }
func (f *fakeDB) Ping() error                                        { return nil }
func (f *fakeDB) Begin() (*sql.Tx, error)                            { return nil, errors.New("fake") }
func (f *fakeDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, errors.New("fake")
}
func (f *fakeDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return nil, errors.New("fake")
}
func (f *fakeDB) Driver() driver.Driver { return nil }

// fakeRepo implements interfaces.DatabaseRepo with no-op errors.
type fakeRepo struct{}

func (f *fakeRepo) Ping() (bool, error) { return false, errors.New("fake") }
func (f *fakeRepo) ListCollections(schema string) ([]models.Table, error) {
	return nil, errors.New("fake")
}
func (f *fakeRepo) ExecuteQuery(name string, params models.QueryParams) (any, error) {
	return nil, errors.New("fake")
}
func (f *fakeRepo) ExecuteFunction(ctx context.Context, name string, args map[string]interface{}) (any, error) {
	return nil, errors.New("fake")
}
func (f *fakeRepo) ExecuteRawSQL(ctx context.Context, sql string) error  { return errors.New("fake") }
func (f *fakeRepo) CreateCollection(req models.CreateTableRequest) error { return errors.New("fake") }
func (f *fakeRepo) AddField(collection string, req models.AddColumnRequest) error {
	return errors.New("fake")
}
func (f *fakeRepo) AlterCollection(collection string, req models.AlterTableRequest) error {
	return errors.New("fake")
}
func (f *fakeRepo) CheckTableExists(tableName string) (bool, error) { return false, errors.New("fake") }
func (f *fakeRepo) Insert(collection string, data map[string]any) (any, error) {
	return nil, errors.New("fake")
}
func (f *fakeRepo) Update(collection string, id any, data map[string]any) (any, error) {
	return nil, errors.New("fake")
}
func (f *fakeRepo) Delete(collection string, id any) error { return errors.New("fake") }
func (f *fakeRepo) BulkInsert(tableName string, records []map[string]interface{}) ([]map[string]interface{}, error) {
	return nil, errors.New("fake")
}
func (f *fakeRepo) BulkUpdate(tableName string, updates []map[string]interface{}, whereColumn string) (int64, error) {
	return 0, errors.New("fake")
}
func (f *fakeRepo) BulkDelete(tableName string, ids []interface{}, idColumn string) (int64, error) {
	return 0, errors.New("fake")
}
func (f *fakeRepo) Upsert(tableName string, data map[string]interface{}, conflictColumns, updateColumns []string) (map[string]interface{}, error) {
	return nil, errors.New("fake")
}
func (f *fakeRepo) CreateForeignKeyConstraint(relationship *models.RelationshipDefinition) error {
	return errors.New("fake")
}
func (f *fakeRepo) DropRelationshipConstraints(relationship *models.RelationshipDefinition) error {
	return errors.New("fake")
}
func (f *fakeRepo) CreateJoinTable(relationship *models.RelationshipDefinition, joinTable models.CreateJoinTableRequest) error {
	return errors.New("fake")
}
func (f *fakeRepo) DropJoinTable(tableName string) error { return errors.New("fake") }
func (f *fakeRepo) SetOneToOneRelation(relationship *models.RelationshipDefinition, sourceID interface{}, targetID interface{}) error {
	return errors.New("fake")
}
func (f *fakeRepo) SetOneToManyRelation(relationship *models.RelationshipDefinition, sourceID interface{}, targetIDs []interface{}) error {
	return errors.New("fake")
}
func (f *fakeRepo) SetOneToManyRelations(relationship *models.RelationshipDefinition, sourceID interface{}, targetIDs []interface{}) error {
	return errors.New("fake")
}
func (f *fakeRepo) SetManyToManyRelations(relationship *models.RelationshipDefinition, sourceID interface{}, targetIDs []interface{}, data map[string]interface{}) ([]map[string]interface{}, error) {
	return nil, errors.New("fake")
}
func (f *fakeRepo) RemoveOneToManyRelations(relationship *models.RelationshipDefinition, sourceID interface{}, targetIDs []interface{}) (int, error) {
	return 0, errors.New("fake")
}
func (f *fakeRepo) RemoveManyToManyRelations(relationship *models.RelationshipDefinition, sourceID interface{}, targetIDs []interface{}) (int, error) {
	return 0, errors.New("fake")
}
func (f *fakeRepo) GetRelationshipData(ctx context.Context, relationship *models.RelationshipDefinition, sourceID string, params models.QueryParams) ([]map[string]interface{}, error) {
	return nil, errors.New("fake")
}
func (f *fakeRepo) CreateIndex(tableName, indexName, columns string) error { return errors.New("fake") }
func (f *fakeRepo) GetPerformanceMetrics() (map[string]interface{}, error) {
	return nil, errors.New("fake")
}
func (f *fakeRepo) AnalyzeQuery(query string) ([]string, error) { return nil, errors.New("fake") }
func (f *fakeRepo) GetMigrationHistory() ([]map[string]interface{}, error) {
	return nil, errors.New("fake")
}
func (f *fakeRepo) RecordMigration(name, sql, checksum string) error { return errors.New("fake") }

var _ interfaces.DB = (*fakeDB)(nil)
var _ interfaces.DatabaseRepo = (*fakeRepo)(nil)

// fake connection factory for database.NewDatabaseConnectorFactory

type fakeConnFactory struct{}

func (f *fakeConnFactory) CreateConnection(cfg *config.DatabaseConfig) (interfaces.DB, error) {
	return &fakeDB{}, nil
}

func withFakeDatabase(t *testing.T) func() {
	originalFactory := pkg.CreateConnectorFactory
	originalRepo := pkg.CreateRepository

	pkg.CreateConnectorFactory = func() *database.DatabaseConnectorFactory {
		factory := database.NewDatabaseConnectorFactory()
		factory.RegisterConnector("postgres", &fakeConnFactory{})
		return factory
	}
	pkg.CreateRepository = func(dbType string, db interfaces.DB) (interfaces.DatabaseRepo, error) {
		return &fakeRepo{}, nil
	}

	return func() {
		pkg.CreateConnectorFactory = originalFactory
		pkg.CreateRepository = originalRepo
	}
}

func validConfig() *appConfig.Config {
	return &appConfig.Config{
		Database: appConfig.DatabaseConfig{
			Host:         "localhost",
			Port:         5432,
			Username:     "user",
			Password:     "pass",
			DatabaseName: "db",
			Driver:       "postgres",
			SSLMode:      "disable",
			MaxOpenConns: 1,
			MaxIdleConns: 1,
		},
		Server: appConfig.ServerConfig{
			Host:         "127.0.0.1",
			Port:         "8080",
			ReadTimeout:  1,
			WriteTimeout: 1,
			Scheme:       "http",
			Env:          "test",
		},
		Auth: appConfig.AuthConfig{
			JWT: appConfig.JWTConfig{
				Secret:             "test-secret-key-minimum-32-chars",
				AccessTokenExpiry:  3600,
				RefreshTokenExpiry: 7200,
				Issuer:             "serenibase-test",
			},
		},
		Email: appConfig.EmailConfig{URL: "http://localhost:8082"},
		Storage: appConfig.StorageConfig{
			URL: "http://localhost:8083",
		},
		Antivirus: appConfig.AntivirusConfig{
			URL: "http://localhost:8084",
		},
		TemporaryAddedUserPassword: appConfig.TemporaryAddedUserPasswordConfig{Value: "temp"},
		OwnerRegistration:          appConfig.OwnerRegistrationConfig{},
	}
}

func TestAppNew_Success(t *testing.T) {
	cleanup := withFakeDatabase(t)
	defer cleanup()

	cfg := validConfig()
	appInstance, err := appPkg.New(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if appInstance == nil {
		t.Fatal("expected app instance, got nil")
	}
	if appInstance.Router() == nil {
		t.Fatal("expected router to be initialized")
	}
}

func TestAppNew_DatabaseInitError(t *testing.T) {
	originalFactory := pkg.CreateConnectorFactory
	defer func() { pkg.CreateConnectorFactory = originalFactory }()

	pkg.CreateConnectorFactory = func() *database.DatabaseConnectorFactory {
		factory := database.NewDatabaseConnectorFactory()
		factory.RegisterConnector("postgres", &failingConnFactory{})
		return factory
	}

	cfg := validConfig()
	_, err := appPkg.New(cfg)
	if err == nil {
		t.Fatal("expected error from database init, got nil")
	}
}

type failingConnFactory struct{}

func (f *failingConnFactory) CreateConnection(cfg *config.DatabaseConfig) (interfaces.DB, error) {
	return nil, errors.New("connect fail")
}

func TestAppNew_StorageError(t *testing.T) {
	cleanup := withFakeDatabase(t)
	defer cleanup()

	cfg := validConfig()
	cfg.Storage.URL = ""

	_, err := appPkg.New(cfg)
	if err == nil {
		t.Fatal("expected storage error, got nil")
	}
}

func TestAppNew_AntivirusError(t *testing.T) {
	cleanup := withFakeDatabase(t)
	defer cleanup()

	cfg := validConfig()
	cfg.Antivirus.URL = ""

	_, err := appPkg.New(cfg)
	if err == nil {
		t.Fatal("expected antivirus error, got nil")
	}
}

func TestAppRun_NilDatabaseService(t *testing.T) {
	var a appPkg.App
	err := a.Run()
	if err == nil {
		t.Fatal("expected error for nil database service")
	}
}

func TestAppRun_ExecutesRunBeforeServer(t *testing.T) {
	cleanup := withFakeDatabase(t)
	defer cleanup()

	cfg := validConfig()
	cfg.Server.Port = "bad"          // force ListenAndServe to fail fast
	cfg.OwnerRegistration.Email = "" // skip owner registration in scripts

	appInstance, err := appPkg.New(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Run should execute runBeforeServer and then fail to listen
	err = appInstance.Run()
	if err == nil {
		t.Fatal("expected ListenAndServe error, got nil")
	}

	// small delay to ensure any goroutines from providers have time to start/stop
	time.Sleep(10 * time.Millisecond)
}
