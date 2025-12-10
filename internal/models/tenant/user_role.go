package tenant

import (
	"fmt"
	"godbgrest/pkg/models"

	"github.com/google/uuid"
)

type UserRole struct {
	ID     uuid.UUID `db:"id" json:"id,omitempty" mapstructure:"id"`
	UserID string    `db:"user_id" json:"user_id,omitempty" mapstructure:"user_id"`
	RoleID string    `db:"role_id" json:"role_id,omitempty" mapstructure:"role_id"`
}

func (UserRole) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".user_roles", prefix)
}

func (tbl UserRole) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "uuid", NotNull: true, Unique: true},
			{Name: "user_id", DataType: "varchar", NotNull: true},
			{Name: "role_id", DataType: "varchar", NotNull: true},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_user_roles_user_id", Columns: []string{"user_id"}},
			{Name: "idx_user_roles_role_id", Columns: []string{"role_id"}},
			{Name: "idx_user_roles_user_role", Columns: []string{"user_id", "role_id"}, Unique: true},
		},
	}
}
