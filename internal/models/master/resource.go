package master

import (
	"godbgrest/pkg/models"
	"time"

	"github.com/google/uuid"
)

// Resource represents a system resource that can be accessed
// Examples: workspace, base, records, members, settings, etc.
type Resource struct {
	ID          uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	Code        string    `db:"code" json:"code,omitempty" mapstructure:"code"` // workspace, base, records, members, etc.
	Description *string   `db:"description" json:"description,omitempty" mapstructure:"description"`
	CreatedAt   time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
}

func (Resource) TableName() string {
	return "\"master\".resources"
}

func (tbl Resource) TableSchema() models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "uuid", NotNull: true, Unique: true},
			{Name: "code", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "description", DataType: "text"},
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: strPtr("CURRENT_TIMESTAMP")},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_resources_code", Columns: []string{"code"}},
		},
	}
}
