package core_test

import (
	"database/sql"

	"github.com/aptlogica/go-postgres-rest/pkg/database/postgres"
	libServices "github.com/aptlogica/go-postgres-rest/pkg/services"
	servicesInterface "github.com/aptlogica/go-postgres-rest/pkg/services/interfaces"
)

// newSchemaService builds the library's schema service on a sqlmock connection, as the app gets it from DatabaseService
func newSchemaService(db *sql.DB) servicesInterface.Schema {
	return libServices.NewSchemaService(postgres.NewSchemaRepo(postgres.NewPostgresDbServiceInstance(db)))
}
