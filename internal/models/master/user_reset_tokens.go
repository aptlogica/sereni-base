package master

import (
	"fmt"
	"godbgrest/pkg/models"
	"time"

	"github.com/google/uuid"
)

type UserResetToken struct {
	ID     uuid.UUID `db:"id" json:"id" mapstructure:"id"`
	Token  string    `db:"token" json:"token" mapstructure:"token"`
	UserID uuid.UUID `db:"user_id" json:"user_id" mapstructure:"user_id"`
	Expiry time.Time `db:"expiry" json:"expiry" mapstructure:"expiry"`
}

func (UserResetToken) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".user_reset_tokens", prefix)
}

func (tbl UserResetToken) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "uuid", NotNull: true, Unique: true},
			{Name: "token", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "user_id", DataType: "uuid", NotNull: true},
			{Name: "expiry", DataType: "timestamp", NotNull: true},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_user_reset_tokens_user_id", Columns: []string{"user_id"}},
		},
	}
}
