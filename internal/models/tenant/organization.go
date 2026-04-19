// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package tenant

import (
	"fmt"
	"time"

	"github.com/aptlogica/go-postgres-rest/pkg/models"

	"github.com/google/uuid"
)

type Organization struct {
	ID          uuid.UUID              `db:"id" json:"id" mapstructure:"id"`
	Name        string                 `db:"name" json:"name" mapstructure:"name"`
	Description *string                `db:"description" json:"description" mapstructure:"description"`
	Email       string                 `db:"email" json:"email" mapstructure:"email"`
	Phone       *string                `db:"phone" json:"phone" mapstructure:"phone"`
	Website     *string                `db:"website" json:"website" mapstructure:"website"`
	Logo        *string                `db:"logo" json:"logo" mapstructure:"logo"`
	Address     *string                `db:"address" json:"address" mapstructure:"address"`
	City        *string                `db:"city" json:"city" mapstructure:"city"`
	State       *string                `db:"state" json:"state" mapstructure:"state"`
	Country     *string                `db:"country" json:"country" mapstructure:"country"`
	ZipCode     *string                `db:"zip_code" json:"zip_code" mapstructure:"zip_code"`
	Settings    map[string]interface{} `db:"settings" json:"settings" mapstructure:"settings"`
	Meta        map[string]interface{} `db:"meta" json:"meta" mapstructure:"meta"`
	Status      string                 `db:"status" json:"status" mapstructure:"status"`

	CreatedBy string `db:"created_by" json:"created_by" mapstructure:"created_by"`
	UpdatedBy string `db:"last_modified_by" json:"last_modified_by" mapstructure:"last_modified_by"`

	CreatedAt time.Time `db:"created_time" json:"created_time" mapstructure:"created_time"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time" mapstructure:"last_modified_time"`
}

func (Organization) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".organizations", prefix)
}

func (tbl Organization) TableSchema(prefix string) models.CreateTableRequest {
	return models.CreateTableRequest{
		Name: tbl.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "varchar", NotNull: true, Unique: true},
			{Name: "name", DataType: "varchar", NotNull: true},
			{Name: "description", DataType: "text"},
			{Name: "email", DataType: "varchar", NotNull: true},
			{Name: "phone", DataType: "varchar"},
			{Name: "website", DataType: "varchar"},
			{Name: "logo", DataType: "varchar"},
			{Name: "address", DataType: "text"},
			{Name: "city", DataType: "varchar"},
			{Name: "state", DataType: "varchar"},
			{Name: "country", DataType: "varchar"},
			{Name: "zip_code", DataType: "varchar"},
			{Name: "settings", DataType: "jsonb"},
			{Name: "meta", DataType: "jsonb"},
			{Name: "status", DataType: "varchar", DefaultValue: StrPtr("'active'")},
			{Name: "created_by", DataType: "varchar"},
			{Name: "last_modified_by", DataType: "varchar"},
			{Name: "created_time", DataType: "timestamp", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
			{Name: "last_modified_time", DataType: "timestamp", NotNull: true, DefaultValue: StrPtr("CURRENT_TIMESTAMP")},
		},
		Indexes: []models.IndexDefinition{
			{Name: "idx_organizations_email", Columns: []string{"email"}},
			{Name: "idx_organizations_status", Columns: []string{"status"}},
			{Name: "idx_organizations_created_by", Columns: []string{"created_by"}},
		},
	}
}
