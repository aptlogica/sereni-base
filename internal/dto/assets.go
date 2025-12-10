package dto

import (
	"mime/multipart"
	"time"

	"github.com/google/uuid"
)

type UploadAssetRequest struct {
	Files []*multipart.FileHeader `json:"files" form:"files"`
}

type BulkAssetRequest struct {
	IDs []string `json:"ids" binding:"required" example:"[\"id1\",\"id2\"]" format:"array"`
}

type AssetInsertion struct {
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

func (a AssetInsertion) Map() map[string]interface{} {
	return map[string]interface{}{
		"id":            a.ID,
		"title":         a.Title,
		"url":           a.Url,
		"thumbnail_url": a.ThumbnailUrl,
		"base_path":     a.BasePath,
		"mime_type":     a.MimeType,
		"size":          a.Size,
		"height":        a.Height,
		"width":         a.Width,
		"created_time": a.CreatedAt,
		"last_modified_time": a.UpdatedAt,

	}
}

type AssetUpdate struct {
	Title     *string   `db:"title" json:"title,omitempty" mapstructure:"title"`
	UpdatedAt time.Time `db:"last_modified_time" json:"last_modified_time,omitempty" mapstructure:"last_modified_time"`

}

func (a AssetUpdate) Map() map[string]interface{} {
	m := make(map[string]interface{})
	if a.Title != nil {
		m["title"] = a.Title
	}
	if !a.UpdatedAt.IsZero() {
		m["last_modified_time"] = a.UpdatedAt
	}
	return m
}
