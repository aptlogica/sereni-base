package asset_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"go-postgres-rest/pkg"
	app_errors "serenibase/internal/app-errors"
	"serenibase/internal/dto"
	"serenibase/internal/models/tenant"
	antivirusInterfaces "serenibase/internal/providers/antivirus/interfaces"
	storageInterfaces "serenibase/internal/providers/storage/interfaces"
	services "serenibase/internal/services/asset"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAssetService is a mock implementation of AssetService interface
type MockAssetService struct {
	mock.Mock
}

func (m *MockAssetService) AssetInsertion(ctx context.Context, assetData dto.AssetInsertion, schemaName string) (tenant.Assets, error) {
	args := m.Called(ctx, assetData, schemaName)
	return args.Get(0).(tenant.Assets), args.Error(1)
}

func (m *MockAssetService) GetBulkAssets(ctx context.Context, schemaName string, ids []string) ([]tenant.Assets, error) {
	args := m.Called(ctx, schemaName, ids)
	if args.Error(1) != nil || args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]tenant.Assets), args.Error(1)
}

func (m *MockAssetService) AssetBulkInsertion(ctx context.Context, assetData []dto.AssetInsertion, schemaName string) ([]tenant.Assets, error) {
	args := m.Called(ctx, assetData, schemaName)
	if args.Error(1) != nil || args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]tenant.Assets), args.Error(1)
}

func (m *MockAssetService) AssetUpdate(ctx context.Context, assetId string, assetData dto.AssetUpdate, schemaName string) (tenant.Assets, error) {
	args := m.Called(ctx, assetId, assetData, schemaName)
	return args.Get(0).(tenant.Assets), args.Error(1)
}

func (m *MockAssetService) GetAssetByID(ctx context.Context, id string, schemaName string) (tenant.Assets, error) {
	args := m.Called(ctx, id, schemaName)
	return args.Get(0).(tenant.Assets), args.Error(1)
}

func (m *MockAssetService) DeleteAsset(ctx context.Context, assetId string, schemaName string) error {
	args := m.Called(ctx, assetId, schemaName)
	return args.Error(0)
}

func (m *MockAssetService) GetAssetByURL(ctx context.Context, url string, schemaName string) (tenant.Assets, error) {
	args := m.Called(ctx, url, schemaName)
	return args.Get(0).(tenant.Assets), args.Error(1)
}

// MockStorageProvider is a mock implementation of StorageProvider interface
type MockStorageProvider struct {
	mock.Mock
}

func (m *MockStorageProvider) Upload(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (storageInterfaces.UploadResponse, error) {
	args := m.Called(ctx, objectName, reader, size, contentType)
	if args.Error(1) != nil {
		return storageInterfaces.UploadResponse{}, args.Error(1)
	}
	return args.Get(0).(storageInterfaces.UploadResponse), args.Error(1)
}

func (m *MockStorageProvider) Download(ctx context.Context, objectName string) (io.ReadCloser, error) {
	args := m.Called(ctx, objectName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockStorageProvider) Delete(ctx context.Context, objectName string) error {
	args := m.Called(ctx, objectName)
	return args.Error(0)
}

func (m *MockStorageProvider) Exists(ctx context.Context, objectName string) (bool, error) {
	args := m.Called(ctx, objectName)
	return args.Bool(0), args.Error(1)
}

// MockAntivirusProvider is a mock implementation of antivirus Provider interface
type MockAntivirusProvider struct {
	mock.Mock
}

func (m *MockAntivirusProvider) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockAntivirusProvider) ScanReader(ctx context.Context, filename string, reader io.Reader) (antivirusInterfaces.ScanResult, error) {
	args := m.Called(ctx, filename, reader)
	if args.Error(1) != nil {
		return antivirusInterfaces.ScanResult{}, args.Error(1)
	}
	return args.Get(0).(antivirusInterfaces.ScanResult), args.Error(1)
}

// Helper function to create a test image file
func createTestImageFile(t *testing.T, filename string, width, height int) *multipart.FileHeader {
	// Create a simple image
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with a color
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 100, 255})
		}
	}

	// Encode to buffer
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("Failed to encode image: %v", err)
	}

	// Create multipart file header
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	h.Set("Content-Type", "image/png")

	part, err := writer.CreatePart(h)
	if err != nil {
		t.Fatalf("Failed to create part: %v", err)
	}

	if _, err := io.Copy(part, &buf); err != nil {
		t.Fatalf("Failed to copy: %v", err)
	}

	writer.Close()

	// Parse the multipart data
	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(10 << 20) // 10MB
	if err != nil {
		t.Fatalf("Failed to read form: %v", err)
	}

	files := form.File["file"]
	if len(files) == 0 {
		t.Fatal("No files found")
	}

	return files[0]
}

// Helper function to create a test non-image file
func createTestTextFile(t *testing.T, filename string, content string) *multipart.FileHeader {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	h.Set("Content-Type", "text/plain")

	part, err := writer.CreatePart(h)
	if err != nil {
		t.Fatalf("Failed to create part: %v", err)
	}

	if _, err := io.WriteString(part, content); err != nil {
		t.Fatalf("Failed to write content: %v", err)
	}

	writer.Close()

	// Parse the multipart data
	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(10 << 20)
	if err != nil {
		t.Fatalf("Failed to read form: %v", err)
	}

	files := form.File["file"]
	if len(files) == 0 {
		t.Fatal("No files found")
	}

	return files[0]
}

func TestNewAssetManagementService(t *testing.T) {
	db := &pkg.DatabaseService{}
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}
	mockAntivirus := &MockAntivirusProvider{}

	service := services.NewAssetManagementService(db, mockAsset, mockStorage, mockAntivirus)

	assert.NotNil(t, service, "NewAssetManagementService should return a non-nil service")
}

func TestUpload_Success_SingleFile(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}
	mockAntivirus := &MockAntivirusProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, mockAntivirus)

	ctx := context.Background()
	schema := "test_schema"

	fileHeader := createTestImageFile(t, "test.png", 100, 100)
	req := dto.UploadAssetRequest{
		Files: []*multipart.FileHeader{fileHeader},
	}

	// Mock antivirus scan
	mockAntivirus.On("ScanReader", ctx, "test.png", mock.Anything).Return(antivirusInterfaces.ScanResult{
		FileName: "test.png",
		Clean:    true,
	}, nil)

	// Mock storage upload for main file
	mockStorage.On("Upload", ctx, mock.MatchedBy(func(name string) bool {
		return strings.Contains(name, "test_schema") && strings.Contains(name, "test") && strings.HasSuffix(name, ".png")
	}), mock.Anything, mock.Anything, "image/png").Return(storageInterfaces.UploadResponse{
		Url: "https://storage.example.com/test.png",
	}, nil).Once()

	// Mock storage upload for thumbnail
	mockStorage.On("Upload", ctx, mock.MatchedBy(func(name string) bool {
		return strings.Contains(name, "thumb_")
	}), mock.Anything, mock.Anything, "image/jpeg").Return(storageInterfaces.UploadResponse{
		Url: "https://storage.example.com/thumb.jpg",
	}, nil).Once() // Mock bulk insertion
	mockAsset.On("AssetBulkInsertion", ctx, mock.MatchedBy(func(assets []dto.AssetInsertion) bool {
		return len(assets) == 1 && assets[0].Title == "test.png"
	}), schema).Return([]tenant.Assets{
		{
			ID:           uuid.New(),
			Title:        "test.png",
			Url:          "https://storage.example.com/test.png",
			ThumbnailUrl: "https://storage.example.com/thumb.jpg",
			MimeType:     "image/png",
			Width:        100,
			Height:       100,
		},
	}, nil)

	result, err := service.Upload(ctx, req, schema)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "test.png", result[0].Title)
	mockAntivirus.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockAsset.AssertExpectations(t)
}

func TestUpload_Success_MultipleFiles(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}
	mockAntivirus := &MockAntivirusProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, mockAntivirus)

	ctx := context.Background()
	schema := "test_schema"

	file1 := createTestImageFile(t, "test1.png", 100, 100)
	file2 := createTestImageFile(t, "test2.png", 200, 200)

	req := dto.UploadAssetRequest{
		Files: []*multipart.FileHeader{file1, file2},
	}

	// Mock antivirus scans
	mockAntivirus.On("ScanReader", ctx, "test1.png", mock.Anything).Return(antivirusInterfaces.ScanResult{
		FileName: "test1.png",
		Clean:    true,
	}, nil)
	mockAntivirus.On("ScanReader", ctx, "test2.png", mock.Anything).Return(antivirusInterfaces.ScanResult{
		FileName: "test2.png",
		Clean:    true,
	}, nil)

	// Mock storage uploads
	mockStorage.On("Upload", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(storageInterfaces.UploadResponse{
		Url: "https://storage.example.com/file.png",
	}, nil)

	// Mock bulk insertion
	mockAsset.On("AssetBulkInsertion", ctx, mock.MatchedBy(func(assets []dto.AssetInsertion) bool {
		return len(assets) == 2
	}), schema).Return([]tenant.Assets{
		{ID: uuid.New(), Title: "test1.png"},
		{ID: uuid.New(), Title: "test2.png"},
	}, nil)

	result, err := service.Upload(ctx, req, schema)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	mockAntivirus.AssertExpectations(t)
	mockAsset.AssertExpectations(t)
}

func TestUpload_AntivirusDetectsThreat(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}
	mockAntivirus := &MockAntivirusProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, mockAntivirus)

	ctx := context.Background()
	schema := "test_schema"

	fileHeader := createTestImageFile(t, "malicious.png", 100, 100)
	req := dto.UploadAssetRequest{
		Files: []*multipart.FileHeader{fileHeader},
	}

	// Mock antivirus detecting a threat
	mockAntivirus.On("ScanReader", ctx, "malicious.png", mock.Anything).Return(
		antivirusInterfaces.ScanResult{
			FileName: "malicious.png",
			Threat:   "Trojan.Generic",
			Clean:    false,
		},
		fmt.Errorf("file 'malicious.png' is infected"),
	)

	result, err := service.Upload(ctx, req, schema)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "infected")
	mockAntivirus.AssertExpectations(t)
}

func TestUpload_NoAntivirusProvider(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}

	// No antivirus provider
	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, nil)

	ctx := context.Background()
	schema := "test_schema"

	fileHeader := createTestImageFile(t, "test.png", 100, 100)
	req := dto.UploadAssetRequest{
		Files: []*multipart.FileHeader{fileHeader},
	}

	// Mock storage uploads
	mockStorage.On("Upload", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(storageInterfaces.UploadResponse{
		Url: "https://storage.example.com/file.png",
	}, nil)

	// Mock bulk insertion
	mockAsset.On("AssetBulkInsertion", ctx, mock.Anything, schema).Return([]tenant.Assets{
		{ID: uuid.New(), Title: "test.png"},
	}, nil)

	result, err := service.Upload(ctx, req, schema)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	mockAsset.AssertExpectations(t)
}

func TestUpload_StorageUploadFails(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}
	mockAntivirus := &MockAntivirusProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, mockAntivirus)

	ctx := context.Background()
	schema := "test_schema"

	fileHeader := createTestImageFile(t, "test.png", 100, 100)
	req := dto.UploadAssetRequest{
		Files: []*multipart.FileHeader{fileHeader},
	}

	// Mock antivirus scan
	mockAntivirus.On("ScanReader", ctx, "test.png", mock.Anything).Return(antivirusInterfaces.ScanResult{
		FileName: "test.png",
		Clean:    true,
	}, nil)

	// Mock storage upload failure
	mockStorage.On("Upload", ctx, mock.Anything, mock.Anything, mock.Anything, "image/png").Return(
		nil, errors.New("storage service unavailable"),
	)

	result, err := service.Upload(ctx, req, schema)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, app_errors.StorageUploadFailed, err)
	mockAntivirus.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestUpload_BulkInsertionFails(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}
	mockAntivirus := &MockAntivirusProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, mockAntivirus)

	ctx := context.Background()
	schema := "test_schema"

	fileHeader := createTestImageFile(t, "test.png", 100, 100)
	req := dto.UploadAssetRequest{
		Files: []*multipart.FileHeader{fileHeader},
	}

	// Mock antivirus scan
	mockAntivirus.On("ScanReader", ctx, "test.png", mock.Anything).Return(antivirusInterfaces.ScanResult{
		FileName: "test.png",
		Clean:    true,
	}, nil)

	// Mock storage uploads
	mockStorage.On("Upload", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(storageInterfaces.UploadResponse{
		Url: "https://storage.example.com/file.png",
	}, nil)

	// Mock bulk insertion failure
	mockAsset.On("AssetBulkInsertion", ctx, mock.Anything, schema).Return(storageInterfaces.UploadResponse{}, errors.New("database error"))

	result, err := service.Upload(ctx, req, schema)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockAntivirus.AssertExpectations(t)
	mockAsset.AssertExpectations(t)
}

func TestUpload_NonImageFile_NoThumbnail(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}
	mockAntivirus := &MockAntivirusProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, mockAntivirus)

	ctx := context.Background()
	schema := "test_schema"

	fileHeader := createTestTextFile(t, "document.txt", "test content")
	req := dto.UploadAssetRequest{
		Files: []*multipart.FileHeader{fileHeader},
	}

	// Mock antivirus scan
	mockAntivirus.On("ScanReader", ctx, "document.txt", mock.Anything).Return(antivirusInterfaces.ScanResult{
		FileName: "document.txt",
		Clean:    true,
	}, nil)

	// Mock storage upload for main file only (no thumbnail for non-images)
	mockStorage.On("Upload", ctx, mock.MatchedBy(func(name string) bool {
		return strings.Contains(name, "document") && strings.HasSuffix(name, ".txt")
	}), mock.Anything, mock.Anything, "text/plain").Return(storageInterfaces.UploadResponse{
		Url: "https://storage.example.com/document.txt",
	}, nil).Once()

	// Mock bulk insertion
	mockAsset.On("AssetBulkInsertion", ctx, mock.MatchedBy(func(assets []dto.AssetInsertion) bool {
		// Verify thumbnail URL is same as main URL for non-images
		return len(assets) == 1 && assets[0].ThumbnailUrl == assets[0].Url
	}), schema).Return([]tenant.Assets{
		{
			ID:       uuid.New(),
			Title:    "document.txt",
			Url:      "https://storage.example.com/document.txt",
			MimeType: "text/plain",
		},
	}, nil)

	result, err := service.Upload(ctx, req, schema)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	mockAntivirus.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockAsset.AssertExpectations(t)
}

func TestUploadImage_Success(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}
	mockAntivirus := &MockAntivirusProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, mockAntivirus)

	ctx := context.Background()
	schema := "test_schema"

	fileHeader := createTestImageFile(t, "profile.jpg", 100, 100)
	req := dto.UploadAssetRequest{
		Files: []*multipart.FileHeader{fileHeader},
	}

	// Mock antivirus scan
	mockAntivirus.On("ScanReader", ctx, "profile.jpg", mock.Anything).Return(antivirusInterfaces.ScanResult{
		FileName: "profile.jpg",
		Clean:    true,
	}, nil)

	// Mock storage uploads
	mockStorage.On("Upload", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(storageInterfaces.UploadResponse{
		Url: "https://storage.example.com/profile.jpg",
	}, nil)

	// Mock single insertion
	mockAsset.On("AssetInsertion", ctx, mock.MatchedBy(func(asset dto.AssetInsertion) bool {
		return asset.Title == "profile.jpg"
	}), schema).Return(tenant.Assets{
		ID:       uuid.New(),
		Title:    "profile.jpg",
		Url:      "https://storage.example.com/profile.jpg",
		MimeType: "image/png",
	}, nil)

	result, err := service.UploadImage(ctx, req, schema)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "profile.jpg", result[0].Title)
	mockAntivirus.AssertExpectations(t)
	mockAsset.AssertExpectations(t)
}

func TestUploadImage_NoFiles(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}
	mockAntivirus := &MockAntivirusProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, mockAntivirus)

	ctx := context.Background()
	schema := "test_schema"

	req := dto.UploadAssetRequest{
		Files: []*multipart.FileHeader{},
	}

	result, err := service.UploadImage(ctx, req, schema)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "exactly one")
}

func TestUploadImage_MultipleFiles(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}
	mockAntivirus := &MockAntivirusProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, mockAntivirus)

	ctx := context.Background()
	schema := "test_schema"

	file1 := createTestImageFile(t, "image1.png", 100, 100)
	file2 := createTestImageFile(t, "image2.png", 100, 100)

	req := dto.UploadAssetRequest{
		Files: []*multipart.FileHeader{file1, file2},
	}

	result, err := service.UploadImage(ctx, req, schema)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "exactly one")
}

func TestUploadImage_NonImageFile(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}
	mockAntivirus := &MockAntivirusProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, mockAntivirus)

	ctx := context.Background()
	schema := "test_schema"

	fileHeader := createTestTextFile(t, "document.txt", "test content")
	req := dto.UploadAssetRequest{
		Files: []*multipart.FileHeader{fileHeader},
	}

	result, err := service.UploadImage(ctx, req, schema)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not an image")
}

func TestUploadImage_AntivirusDetectsThreat(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}
	mockAntivirus := &MockAntivirusProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, mockAntivirus)

	ctx := context.Background()
	schema := "test_schema"

	fileHeader := createTestImageFile(t, "malicious.png", 100, 100)
	req := dto.UploadAssetRequest{
		Files: []*multipart.FileHeader{fileHeader},
	}

	// Mock antivirus detecting a threat
	mockAntivirus.On("ScanReader", ctx, "malicious.png", mock.Anything).Return(
		antivirusInterfaces.ScanResult{
			FileName: "malicious.png",
			Threat:   "Trojan.Generic",
			Clean:    false,
		},
		fmt.Errorf("file 'malicious.png' is infected"),
	)

	result, err := service.UploadImage(ctx, req, schema)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "infected")
	mockAntivirus.AssertExpectations(t)
}

func TestUploadImage_StorageUploadFails(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}
	mockAntivirus := &MockAntivirusProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, mockAntivirus)

	ctx := context.Background()
	schema := "test_schema"

	fileHeader := createTestImageFile(t, "test.png", 100, 100)
	req := dto.UploadAssetRequest{
		Files: []*multipart.FileHeader{fileHeader},
	}

	// Mock antivirus scan
	mockAntivirus.On("ScanReader", ctx, "test.png", mock.Anything).Return(antivirusInterfaces.ScanResult{
		FileName: "test.png",
		Clean:    true,
	}, nil)

	// Mock storage upload failure
	mockStorage.On("Upload", ctx, mock.Anything, mock.Anything, mock.Anything, "image/png").Return(
		nil, errors.New("storage service unavailable"),
	)

	result, err := service.UploadImage(ctx, req, schema)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, app_errors.StorageUploadFailed, err)
	mockAntivirus.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestUploadImage_InsertionFails(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}
	mockAntivirus := &MockAntivirusProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, mockAntivirus)

	ctx := context.Background()
	schema := "test_schema"

	fileHeader := createTestImageFile(t, "test.png", 100, 100)
	req := dto.UploadAssetRequest{
		Files: []*multipart.FileHeader{fileHeader},
	}

	// Mock antivirus scan
	mockAntivirus.On("ScanReader", ctx, "test.png", mock.Anything).Return(antivirusInterfaces.ScanResult{
		FileName: "test.png",
		Clean:    true,
	}, nil)

	// Mock storage uploads
	mockStorage.On("Upload", ctx, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(storageInterfaces.UploadResponse{
		Url: "https://storage.example.com/test.png",
	}, nil)

	// Mock insertion failure
	mockAsset.On("AssetInsertion", ctx, mock.Anything, schema).Return(tenant.Assets{}, errors.New("database error"))

	result, err := service.UploadImage(ctx, req, schema)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockAntivirus.AssertExpectations(t)
	mockAsset.AssertExpectations(t)
}

func TestAssetManagement_GetBulkAssets(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, nil)

	ctx := context.Background()
	schemaName := "test_schema"
	ids := []string{"id1", "id2"}

	expectedAssets := []tenant.Assets{
		{ID: uuid.New(), Title: "asset1.png"},
		{ID: uuid.New(), Title: "asset2.png"},
	}

	mockAsset.On("GetBulkAssets", ctx, schemaName, ids).Return(expectedAssets, nil)

	result, err := service.GetBulkAssets(ctx, schemaName, ids)

	assert.NoError(t, err)
	assert.Equal(t, expectedAssets, result)
	mockAsset.AssertExpectations(t)
}

func TestAssetManagement_GetBulkAssets_Error(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, nil)

	ctx := context.Background()
	schemaName := "test_schema"
	ids := []string{"id1"}

	mockAsset.On("GetBulkAssets", ctx, schemaName, ids).Return(storageInterfaces.UploadResponse{}, errors.New("database error"))

	result, err := service.GetBulkAssets(ctx, schemaName, ids)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockAsset.AssertExpectations(t)
}

func TestAssetManagement_UpdateAsset(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, nil)

	ctx := context.Background()
	schemaName := "test_schema"
	assetID := uuid.New().String()
	newTitle := "updated.png"

	assetData := dto.AssetUpdate{
		Title:     &newTitle,
		UpdatedAt: time.Now(),
	}

	expectedAsset := tenant.Assets{
		ID:    uuid.MustParse(assetID),
		Title: "updated.png",
	}

	mockAsset.On("AssetUpdate", ctx, assetID, assetData, schemaName).Return(expectedAsset, nil)

	result, err := service.UpdateAsset(ctx, assetID, assetData, schemaName)

	assert.NoError(t, err)
	assert.Equal(t, expectedAsset, result)
	mockAsset.AssertExpectations(t)
}

func TestAssetManagement_UpdateAsset_Error(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, nil)

	ctx := context.Background()
	schemaName := "test_schema"
	assetID := uuid.New().String()
	newTitle := "updated.png"

	assetData := dto.AssetUpdate{
		Title: &newTitle,
	}

	mockAsset.On("AssetUpdate", ctx, assetID, assetData, schemaName).Return(tenant.Assets{}, errors.New("database error"))

	result, err := service.UpdateAsset(ctx, assetID, assetData, schemaName)

	assert.Error(t, err)
	assert.Equal(t, tenant.Assets{}, result)
	mockAsset.AssertExpectations(t)
}

func TestAssetManagement_DeleteAsset_Success(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, nil)

	ctx := context.Background()
	schemaName := "test_schema"
	assetID := uuid.New().String()

	asset := tenant.Assets{
		ID:           uuid.MustParse(assetID),
		Title:        "test.png",
		Url:          "https://storage.example.com/test.png",
		ThumbnailUrl: "https://storage.example.com/thumb.jpg",
		BasePath:     "test_schema/test.png",
	}

	mockAsset.On("GetAssetByID", ctx, assetID, schemaName).Return(asset, nil)
	mockStorage.On("Delete", ctx, "test_schema/thumb_test.png").Return(nil)
	mockStorage.On("Delete", ctx, "test_schema/test.png").Return(nil)
	mockAsset.On("DeleteAsset", ctx, assetID, schemaName).Return(nil)

	err := service.DeleteAsset(ctx, assetID, schemaName)

	assert.NoError(t, err)
	mockAsset.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestAssetManagement_DeleteAsset_GetAssetFails(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, nil)

	ctx := context.Background()
	schemaName := "test_schema"
	assetID := uuid.New().String()

	mockAsset.On("GetAssetByID", ctx, assetID, schemaName).Return(tenant.Assets{}, errors.New("not found"))

	err := service.DeleteAsset(ctx, assetID, schemaName)

	assert.Error(t, err)
	mockAsset.AssertExpectations(t)
}

func TestAssetManagement_DeleteAsset_ThumbnailSameAsMain(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, nil)

	ctx := context.Background()
	schemaName := "test_schema"
	assetID := uuid.New().String()

	asset := tenant.Assets{
		ID:           uuid.MustParse(assetID),
		Title:        "document.pdf",
		Url:          "https://storage.example.com/doc.pdf",
		ThumbnailUrl: "https://storage.example.com/doc.pdf", // Same as main URL
		BasePath:     "test_schema/doc.pdf",
	}

	mockAsset.On("GetAssetByID", ctx, assetID, schemaName).Return(asset, nil)
	mockStorage.On("Delete", ctx, "test_schema/doc.pdf").Return(nil)
	mockAsset.On("DeleteAsset", ctx, assetID, schemaName).Return(nil)

	err := service.DeleteAsset(ctx, assetID, schemaName)

	assert.NoError(t, err)
	mockAsset.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestAssetManagement_DeleteAsset_ThumbnailDeleteFails_ContinuesAnyway(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, nil)

	ctx := context.Background()
	schemaName := "test_schema"
	assetID := uuid.New().String()

	asset := tenant.Assets{
		ID:           uuid.MustParse(assetID),
		Title:        "test.png",
		Url:          "https://storage.example.com/test.png",
		ThumbnailUrl: "https://storage.example.com/thumb.jpg",
		BasePath:     "test_schema/test.png",
	}

	mockAsset.On("GetAssetByID", ctx, assetID, schemaName).Return(asset, nil)
	mockStorage.On("Delete", ctx, "test_schema/thumb_test.png").Return(errors.New("thumbnail not found"))
	mockStorage.On("Delete", ctx, "test_schema/test.png").Return(nil)
	mockAsset.On("DeleteAsset", ctx, assetID, schemaName).Return(nil)

	err := service.DeleteAsset(ctx, assetID, schemaName)

	assert.NoError(t, err)
	mockAsset.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestAssetManagement_DeleteAsset_MainFileNotFound_ContinuesAnyway(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, nil)

	ctx := context.Background()
	schemaName := "test_schema"
	assetID := uuid.New().String()

	asset := tenant.Assets{
		ID:           uuid.MustParse(assetID),
		Title:        "test.png",
		Url:          "https://storage.example.com/test.png",
		ThumbnailUrl: "https://storage.example.com/test.png",
		BasePath:     "test_schema/test.png",
	}

	mockAsset.On("GetAssetByID", ctx, assetID, schemaName).Return(asset, nil)
	mockStorage.On("Delete", ctx, "test_schema/test.png").Return(errors.New("file not found"))
	mockAsset.On("DeleteAsset", ctx, assetID, schemaName).Return(nil)

	err := service.DeleteAsset(ctx, assetID, schemaName)

	assert.NoError(t, err)
	mockAsset.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestAssetManagement_DeleteAsset_MainFileDeleteFails(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, nil)

	ctx := context.Background()
	schemaName := "test_schema"
	assetID := uuid.New().String()

	asset := tenant.Assets{
		ID:           uuid.MustParse(assetID),
		Title:        "test.png",
		Url:          "https://storage.example.com/test.png",
		ThumbnailUrl: "https://storage.example.com/test.png",
		BasePath:     "test_schema/test.png",
	}

	mockAsset.On("GetAssetByID", ctx, assetID, schemaName).Return(asset, nil)
	mockStorage.On("Delete", ctx, "test_schema/test.png").Return(errors.New("permission denied"))

	err := service.DeleteAsset(ctx, assetID, schemaName)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete asset file")
	mockAsset.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestAssetManagement_DeleteAsset_DatabaseDeleteFails(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, nil)

	ctx := context.Background()
	schemaName := "test_schema"
	assetID := uuid.New().String()

	asset := tenant.Assets{
		ID:           uuid.MustParse(assetID),
		Title:        "test.png",
		Url:          "https://storage.example.com/test.png",
		ThumbnailUrl: "https://storage.example.com/test.png",
		BasePath:     "test_schema/test.png",
	}

	mockAsset.On("GetAssetByID", ctx, assetID, schemaName).Return(asset, nil)
	mockStorage.On("Delete", ctx, "test_schema/test.png").Return(nil)
	mockAsset.On("DeleteAsset", ctx, assetID, schemaName).Return(errors.New("database error"))

	err := service.DeleteAsset(ctx, assetID, schemaName)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete asset from database")
	mockAsset.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestAssetManagement_GetAssetByURL(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, nil)

	ctx := context.Background()
	schemaName := "test_schema"
	url := "https://storage.example.com/test.png"

	expectedAsset := tenant.Assets{
		ID:    uuid.New(),
		Title: "test.png",
		Url:   url,
	}

	mockAsset.On("GetAssetByURL", ctx, url, schemaName).Return(expectedAsset, nil)

	result, err := service.GetAssetByURL(ctx, schemaName, url)

	assert.NoError(t, err)
	assert.Equal(t, expectedAsset, result)
	mockAsset.AssertExpectations(t)
}

func TestAssetManagement_GetAssetByURL_Error(t *testing.T) {
	mockAsset := &MockAssetService{}
	mockStorage := &MockStorageProvider{}

	service := services.NewAssetManagementService(nil, mockAsset, mockStorage, nil)

	ctx := context.Background()
	schemaName := "test_schema"
	url := "https://storage.example.com/notfound.png"

	mockAsset.On("GetAssetByURL", ctx, url, schemaName).Return(tenant.Assets{}, errors.New("not found"))

	result, err := service.GetAssetByURL(ctx, schemaName, url)

	assert.Error(t, err)
	assert.Equal(t, tenant.Assets{}, result)
	mockAsset.AssertExpectations(t)
}
