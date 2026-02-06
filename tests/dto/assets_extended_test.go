package tests

import (
	"testing"
	"time"

	"serenibase/internal/dto"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAssetInsertion_Map(t *testing.T) {
	assetID := uuid.New()
	now := time.Now().UTC()

	insertion := dto.AssetInsertion{
		ID:           assetID,
		Title:        "test-image.png",
		Url:          "https://storage.example.com/test-image.png",
		ThumbnailUrl: "https://storage.example.com/thumb-test-image.png",
		BasePath:     "/uploads",
		MimeType:     "image/png",
		Size:         1024000,
		Height:       1080,
		Width:        1920,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	mapped := insertion.Map()

	assert.Equal(t, assetID, mapped["id"])
	assert.Equal(t, "test-image.png", mapped["title"])
	assert.Equal(t, "https://storage.example.com/test-image.png", mapped["url"])
	assert.Equal(t, "image/png", mapped["mime_type"])
	assert.Equal(t, int64(1024000), mapped["size"])
	assert.Equal(t, 1920, mapped["width"])
	assert.Equal(t, 1080, mapped["height"])
}

func TestAssetUpdate_Map(t *testing.T) {
	title := "updated-image.png"

	update := dto.AssetUpdate{
		Title: &title,
	}

	mapped := update.Map()

	assert.Equal(t, "updated-image.png", *mapped["title"].(*string))
	assert.Len(t, mapped, 1)
}

func TestBulkAssetRequest(t *testing.T) {
	assetIDs := []string{
		uuid.New().String(),
		uuid.New().String(),
		uuid.New().String(),
	}

	req := dto.BulkAssetRequest{
		IDs: assetIDs,
	}

	assert.Len(t, req.IDs, 3)
	assert.NotEmpty(t, req.IDs[0])
	assert.NotEmpty(t, req.IDs[1])
	assert.NotEmpty(t, req.IDs[2])
}

func TestAssetInsertion_AllFields(t *testing.T) {
	assetID := uuid.New()

	insertion := dto.AssetInsertion{
		ID:       assetID,
		Title:    "image.png",
		Url:      "https://example.com/image.png",
		MimeType: "image/png",
		Size:     5000,
	}

	assert.NotEqual(t, uuid.Nil, insertion.ID)
	assert.NotEmpty(t, insertion.Title)
	assert.NotEmpty(t, insertion.Url)
	assert.NotEmpty(t, insertion.MimeType)
	assert.Greater(t, insertion.Size, int64(0))
}

func TestAssetUpdate_PartialUpdate(t *testing.T) {
	title := "new-title.png"

	update := dto.AssetUpdate{
		Title: &title,
	}

	assert.NotNil(t, update.Title)
	assert.Equal(t, "new-title.png", *update.Title)
}

func TestAssetInsertion_WithDimensions(t *testing.T) {
	insertion := dto.AssetInsertion{
		ID:       uuid.New(),
		Title:    "image.jpg",
		Url:      "https://example.com/image.jpg",
		MimeType: "image/jpeg",
		Size:     100000,
		Width:    1920,
		Height:   1080,
	}

	mapped := insertion.Map()

	assert.Equal(t, 1920, mapped["width"])
	assert.Equal(t, 1080, mapped["height"])
}

func TestAssetInsertion_WithoutDimensions(t *testing.T) {
	insertion := dto.AssetInsertion{
		ID:       uuid.New(),
		Title:    "document.pdf",
		Url:      "https://example.com/document.pdf",
		MimeType: "application/pdf",
		Size:     50000,
	}

	mapped := insertion.Map()

	assert.Equal(t, 0, mapped["width"])
	assert.Equal(t, 0, mapped["height"])
}
