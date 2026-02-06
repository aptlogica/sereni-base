package tenant

import (
	"fmt"
	"go-postgres-rest/pkg/models"
	"time"

	"github.com/google/uuid"
)

type Base struct {
	ID          uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	WorkspaceID string    `db:"workspace_id" json:"workspace_id,omitempty" mapstructure:"workspace_id"`
	Title       string    `db:"title" json:"title,omitempty" mapstructure:"title"`
	Description *string   `db:"description" json:"description,omitempty" mapstructure:"description"`
	Image       string    `db:"image" json:"image,omitempty" mapstructure:"image"`

	// Database connection (for external sources)
	Type   string                 `db:"type" json:"type,omitempty" mapstructure:"type"`
	Config map[string]interface{} `db:"config" json:"config,omitempty" mapstructure:"config"`

	// Settings and metadata
	Settings map[string]interface{} `db:"settings" json:"settings,omitempty" mapstructure:"settings"`
	Meta     map[string]interface{} `db:"meta" json:"meta,omitempty" mapstructure:"meta"`

	// Status and visibility
	Status     string `db:"status" json:"status,omitempty" mapstructure:"status"`
	Visibility string `db:"visibility" json:"visibility,omitempty" mapstructure:"visibility"`

	// Resource tracking
	TableCount       int   `db:"table_count" json:"table_count,omitempty" mapstructure:"table_count"`
	RowCount         int64 `db:"row_count" json:"row_count,omitempty" mapstructure:"row_count"`
	StorageUsedBytes int64 `db:"storage_used_bytes" json:"storage_used_bytes,omitempty" mapstructure:"storage_used_bytes"`

	CreatedBy string `db:"created_by" json:"created_by" mapstructure:"created_by"`
	UpdatedBy string `db:"last_modified_by" json:"last_modified_by" mapstructure:"last_modified_by"`

	CreatedAt time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
}

func (Base) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".bases", prefix)
}

func (tbl Base) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "workspace_id", DataType: "varchar", NotNull: true},
			{Name: "title", DataType: "varchar", NotNull: true},
			{Name: "description", DataType: "text"},
			{Name: "image", DataType: "varchar"},
			{Name: "type", DataType: "varchar", DefaultValue: StrPtr("'internal'")},
			{Name: "config", DataType: "jsonb"},
			{Name: "settings", DataType: "jsonb"},
			{Name: "meta", DataType: "jsonb"},
			{Name: "status", DataType: "varchar", DefaultValue: StrPtr("'active'")},
			{Name: "visibility", DataType: "varchar", DefaultValue: StrPtr("'private'")},
			CreateIntegerColumn("table_count"),
			CreateIntegerColumn("row_count"),
			CreateIntegerColumn("storage_used_bytes"),
			{Name: "created_by", DataType: "varchar"},
			{Name: "last_modified_by", DataType: "varchar"},
			CreateTimestampColumn("created_time", true, false),
			CreateTimestampColumn("last_modified_time", true, false),
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_bases_workspace_id", Columns: []string{"workspace_id"}},
			{Name: "idx_bases_status", Columns: []string{"status"}},
			{Name: "idx_bases_visibility", Columns: []string{"visibility"}},
		},
		ForeignKeys: []models.ForeignKeyDef{
			{
				Name:              "fk_bases_workspace_id",
				Columns:           []string{"workspace_id"},
				ReferencedTable:   fmt.Sprintf("\"%s\".workspaces", prefix),
				ReferencedColumns: []string{"id"},
				OnDelete:          "CASCADE",
			},
		},
	}
}
