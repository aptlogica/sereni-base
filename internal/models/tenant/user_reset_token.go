// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package tenant

import (
	"fmt"
	"github.com/aptlogica/go-postgres-rest/pkg/models"

	"github.com/google/uuid"
)

type UserResetToken struct {
	ID       uuid.UUID `db:"id" json:"id" mapstructure:"id"`
	Token    string    `db:"token" json:"token" mapstructure:"token"`
	UserID   uuid.UUID `db:"user_id" json:"user_id" mapstructure:"user_id"`
	IssuedAt string    `db:"issued_at" json:"issued_at" mapstructure:"issued_at"`
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
			{Name: "issued_at", DataType: "varchar", NotNull: true},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_user_reset_tokens_user_id", Columns: []string{"user_id"}},
		},
	}
}
