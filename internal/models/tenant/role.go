package tenant

import (
	"fmt"
	"godbgrest/pkg/models"
	"time"

	"github.com/google/uuid"
)

type Role struct {
	ID          uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	Name        string    `db:"name" json:"name,omitempty" mapstructure:"name"`
	Description *string   `db:"description" json:"description,omitempty" mapstructure:"description"`
	IsDefault   bool      `db:"is_default" json:"is_default,omitempty" mapstructure:"is_default"`
	CreatedAt   time.Time `db:"created_time" json:"created_time,omitempty" mapstructure:"created_time"`
	UpdatedAt   time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`
}

func (Role) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".roles", prefix)
}

func (tbl Role) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "uuid", NotNull: true, Unique: true},
			{Name: "name", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "description", DataType: "text"},
			{Name: "is_default", DataType: "boolean", NotNull: true, DefaultValue: strPtr("false")},
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: strPtr("CURRENT_TIMESTAMP")},
			{Name: "last_modified_time", DataType: "timestamp", NotNull: true, DefaultValue: strPtr("CURRENT_TIMESTAMP")},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_roles_name", Columns: []string{"name"}},
		},
	}
}
