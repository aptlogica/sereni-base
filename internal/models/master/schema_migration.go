package master

import (
	"fmt"
	"godbgrest/pkg/models"
	"time"

	"github.com/google/uuid"
)

type SchemaMigration struct {
	ID               uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	TenantID         uuid.UUID `db:"tenant_id" json:"tenant_id,omitempty" mapstructure:"tenant_id"`
	MigrationName    string    `db:"migration_name" json:"migration_name,omitempty" mapstructure:"migration_name"`
	MigrationVersion string    `db:"migration_version" json:"migration_version,omitempty" mapstructure:"migration_version"`
	Status           string    `db:"status" json:"status,omitempty" mapstructure:"status"` // pending, running, completed, failed

	StartedAt    *time.Time `db:"started_at" json:"started_at,omitempty" mapstructure:"started_at"`
	CompletedAt  *time.Time `db:"completed_at" json:"completed_at,omitempty" mapstructure:"completed_at"`
	ErrorMessage *string    `db:"error_message" json:"error_message,omitempty" mapstructure:"error_message"`
}

func (SchemaMigration) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".schema_migrations", prefix)
}

func (tbl SchemaMigration) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "uuid", NotNull: true, Unique: true},
			{Name: "tenant_id", DataType: "uuid"},
			{Name: "migration_name", DataType: "varchar", NotNull: true},
			{Name: "migration_version", DataType: "varchar", NotNull: true},
			{Name: "status", DataType: "varchar", DefaultValue: strPtr("'pending'")},
			{Name: "started_at", DataType: "timestamp"},
			{Name: "completed_at", DataType: "timestamp"},
			{Name: "error_message", DataType: "text"},
		},
		ForeignKeys: []models.ForeignKeyDef{
			{
				Columns:           []string{"tenant_id"},
				ReferencedTable:   fmt.Sprintf("\"%s\".tenants", prefix),
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
			},
		},
	}
}
