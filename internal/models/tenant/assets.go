// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package tenant

import (
	"fmt"
	"go-postgres-rest/pkg/models"
	"time"

	"github.com/google/uuid"
)

type Assets struct {
	ID           uuid.UUID `db:"id" json:"id" mapstructure:"id"`
	Title        string    `db:"title" json:"title" mapstructure:"title"`
	Url          string    `db:"url" json:"url" mapstructure:"url"`
	ThumbnailUrl string    `db:"thumbnail_url" json:"thumbnail_url" mapstructure:"thumbnail_url"`
	BasePath     string    `db:"base_path" json:"base_path" mapstructure:"base_path"`
	MimeType     string    `db:"mime_type" json:"mime_type" mapstructure:"mime_type"`
	Size         int64     `db:"size" json:"size" mapstructure:"size"`
	Height       int       `db:"height" json:"height" mapstructure:"height" optional:"true" omitempty:"true"`
	Width        int       `db:"width" json:"width" mapstructure:"width" optional:"true" omitempty:"true"`
	CreatedAt    time.Time `db:"created_time" json:"created_time" mapstructure:"created_time"`
	UpdatedAt    time.Time `db:"last_modified_time" json:"last_modified_time" mapstructure:"last_modified_time"`
}

func (a Assets) Map() map[string]interface{} {
	return map[string]interface{}{
		"id":                 a.ID,
		"title":              a.Title,
		"url":                a.Url,
		"thumbnail_url":      a.ThumbnailUrl,
		"base_path":          a.BasePath,
		"mime_type":          a.MimeType,
		"size":               a.Size,
		"height":             a.Height,
		"width":              a.Width,
		"created_time":       a.CreatedAt,
		"last_modified_time": a.UpdatedAt,
	}
}

func (Assets) TableName(prefix string) string {
	return fmt.Sprintf("\"%s\".assets", prefix)
}

func (a Assets) TableSchema(prefix string) models.CreateTableRequest {
	null := "NULL"
	return models.CreateTableRequest{
		Name: a.TableName(prefix),
		Columns: []models.ColumnDefinition{
			{Name: "id", DataType: "uuid", NotNull: true},
			{Name: "title", DataType: "varchar(255)", NotNull: true},
			{Name: "url", DataType: "text", NotNull: true},
			{Name: "thumbnail_url", DataType: "text", NotNull: true},
			{Name: "base_path", DataType: "text", NotNull: true},
			{Name: "mime_type", DataType: "varchar(255)"},
			{Name: "size", DataType: "numeric"},
			{Name: "height", DataType: "int", DefaultValue: &null},
			{Name: "width", DataType: "int", DefaultValue: &null},
			{Name: "created_time", DataType: "timestamp", NotNull: true},
			{Name: "last_modified_time", DataType: "timestamp", NotNull: true},
		},
	}
}
