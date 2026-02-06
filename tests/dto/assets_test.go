package tests

import (
	"serenibase/internal/dto"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAssetInsertionMap(t *testing.T) {
	id := uuid.New()
	now := time.Now()

	asset := dto.AssetInsertion{
		ID:           id,
		Title:        "test-image.png",
		Url:          "https://example.com/test-image.png",
		ThumbnailUrl: "https://example.com/thumb.png",
		BasePath:     "/uploads",
		MimeType:     "image/png",
		Size:         1024000,
		Height:       800,
		Width:        1200,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	m := asset.Map()

	if m["id"] != id {
		t.Errorf("Map() id = %v, want %v", m["id"], id)
	}
	if m["title"] != "test-image.png" {
		t.Errorf("Map() title = %v, want %v", m["title"], "test-image.png")
	}
	if m["url"] != "https://example.com/test-image.png" {
		t.Errorf("Map() url = %v, want %v", m["url"], "https://example.com/test-image.png")
	}
	if m["thumbnail_url"] != "https://example.com/thumb.png" {
		t.Errorf("Map() thumbnail_url = %v, want %v", m["thumbnail_url"], "https://example.com/thumb.png")
	}
	if m["mime_type"] != "image/png" {
		t.Errorf("Map() mime_type = %v, want %v", m["mime_type"], "image/png")
	}
	if m["size"] != int64(1024000) {
		t.Errorf("Map() size = %v, want %v", m["size"], int64(1024000))
	}
	if m["height"] != 800 {
		t.Errorf("Map() height = %v, want %v", m["height"], 800)
	}
	if m["width"] != 1200 {
		t.Errorf("Map() width = %v, want %v", m["width"], 1200)
	}
}

func TestAssetUpdateMap(t *testing.T) {
	title := "updated-title.png"
	now := time.Now()

	update := dto.AssetUpdate{
		Title:     &title,
		UpdatedAt: now,
	}

	m := update.Map()

	if m["title"] != &title {
		t.Errorf("Map() title = %v, want %v", m["title"], &title)
	}
	if m["last_modified_time"] != now {
		t.Errorf("Map() last_modified_time = %v, want %v", m["last_modified_time"], now)
	}
}

func TestAssetUpdateMapEmpty(t *testing.T) {
	update := dto.AssetUpdate{}

	m := update.Map()

	// Empty map since all fields are nil/zero
	if len(m) != 0 {
		t.Errorf("Map() length = %v, want %v", len(m), 0)
	}
}

func TestAssetUpdateMapOnlyTitle(t *testing.T) {
	title := "new-title.jpg"

	update := dto.AssetUpdate{
		Title: &title,
	}

	m := update.Map()

	if m["title"] != &title {
		t.Errorf("Map() title = %v, want %v", m["title"], &title)
	}
	if _, ok := m["last_modified_time"]; ok {
		t.Error("Map() should not contain 'last_modified_time' key when it's zero")
	}
}

func TestBulkAssetRequestFields(t *testing.T) {
	req := dto.BulkAssetRequest{
		IDs: []string{"id1", "id2", "id3"},
	}

	if len(req.IDs) != 3 {
		t.Errorf("len(IDs) = %v, want %v", len(req.IDs), 3)
	}
	if req.IDs[0] != "id1" {
		t.Errorf("IDs[0] = %v, want %v", req.IDs[0], "id1")
	}
}
