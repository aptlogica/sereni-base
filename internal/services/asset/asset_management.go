// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package services

import (
	"bytes"
	"context"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"path/filepath"

	"github.com/aptlogica/go-postgres-rest/pkg"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/models/tenant"
	antivirusProviderInterface "github.com/aptlogica/sereni-base/internal/providers/antivirus/interfaces"
	storageProviderInterface "github.com/aptlogica/sereni-base/internal/providers/storage/interfaces"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"

	"fmt"
	"strings"
	"time"

	app_errors "github.com/aptlogica/sereni-base/internal/app-errors"

	"github.com/google/uuid"
	"github.com/nfnt/resize"
)

const ContentTypeImageJPEG = "image/jpeg"

type assetManagementService struct {
	repo                   *pkg.DatabaseService
	assetsService          interfaces.AssetService
	storageProviderService storageProviderInterface.StorageProvider
	antivirusProvider      antivirusProviderInterface.Provider
}

func NewAssetManagementService(
	repo *pkg.DatabaseService,
	assetsService interfaces.AssetService,
	storageProviderService storageProviderInterface.StorageProvider,
	antivirusProvider antivirusProviderInterface.Provider,
) interfaces.AssetManagementService {
	return &assetManagementService{
		repo:                   repo,
		assetsService:          assetsService,
		storageProviderService: storageProviderService,
		antivirusProvider:      antivirusProvider,
	}
}

func (s *assetManagementService) getImageDimensions(file io.ReadSeeker) (int, int, error) {
	cfg, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

func (s *assetManagementService) generateTimestampedFilename(filename string) string {
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	timestamp := time.Now().Format("20060102_150405") // e.g., 20250804_143210
	return fmt.Sprintf("%s_%s%s", name, timestamp, ext)
}

func (s *assetManagementService) processAndUploadFile(fileHeader *multipart.FileHeader, schema string) (dto.AssetInsertion, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return dto.AssetInsertion{}, app_errors.StorageFileOpenFailed
	}
	defer file.Close()

	width, height, err := s.getImageDimensions(file)
	if err != nil {
		width, height = 0, 0
	}

	// Reset the file reader before upload
	if _, err = file.Seek(0, 0); err != nil {
		return dto.AssetInsertion{}, app_errors.StorageFileOpenFailed
	}

	contentType := fileHeader.Header.Get("Content-Type")
	fileName := s.generateTimestampedFilename(fileHeader.Filename)
	objectName := filepath.Join(schema, fileName)

	filePath, err := s.uploadMainFile(objectName, file, fileHeader.Size, contentType, schema)
	if err != nil {
		return dto.AssetInsertion{}, app_errors.StorageUploadFailed
	}

	thumbnailUrl := s.getThumbnailUrl(fileHeader, contentType, fileName, filePath, schema)

	return dto.AssetInsertion{
		ID:           uuid.New(),
		Title:        fileHeader.Filename,
		BasePath:     strings.ReplaceAll(objectName, "\\", "/"),
		Url:          filePath,
		ThumbnailUrl: thumbnailUrl,
		MimeType:     contentType,
		Size:         fileHeader.Size,
		Height:       height,
		Width:        width,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

// Sagrigated helper to upload main file
func (s *assetManagementService) uploadMainFile(objectName string, file io.Reader, size int64, contentType string, schema string) (string, error) {
	response, err := s.storageProviderService.Upload(
		context.Background(),
		objectName,
		file,
		size,
		contentType,
	)
	if err != nil {
		return "", err
	}
	return response.Url, nil
}

// getThumbnailUrl generates a thumbnail for image files.
// Thumbnails are JPEG, max 200px on the longest edge.
func (s *assetManagementService) getThumbnailUrl(
	fileHeader *multipart.FileHeader, contentType, fileName, filePath string, schema string,
) string {
	isImage := strings.HasPrefix(contentType, "image/") &&
		contentType != "image/tiff" &&
		(strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".jpg") ||
			strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".jpeg") ||
			strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".png")) &&
		!strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".tiff") &&
		!strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".tif")

	if !isImage {
		return filePath // non-image files: thumbnail is same as url
	}

	const thumbnailMaxSize = 200

	// Open the image file
	file, err := fileHeader.Open()
	if err != nil {
		return filePath
	}
	defer file.Close()

	// Decode using standard library
	srcImg, _, err := image.Decode(file)
	if err != nil {
		return filePath
	}

	// Resize so that the largest edge is thumbnailMaxSize px (preserve aspect)
	thumbImg := resize.Thumbnail(uint(thumbnailMaxSize), uint(thumbnailMaxSize), srcImg, resize.Lanczos3)

	// Encode to JPEG (quality=75)
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, thumbImg, &jpeg.Options{Quality: 75})
	if err != nil {
		return filePath
	}

	thumbBytes := buf.Bytes()
	if len(thumbBytes) == 0 {
		return filePath
	}

	thumbFile := bytes.NewReader(thumbBytes)
	thumbContentType := ContentTypeImageJPEG
	thumbObjectName := filepath.Join(schema, "thumb_"+fileName)
	thumbResp, thumbErr := s.storageProviderService.Upload(
		context.Background(),
		thumbObjectName,
		thumbFile,
		int64(len(thumbBytes)),
		thumbContentType,
	)
	if thumbErr != nil {
		return filePath
	}

	return thumbResp.Url
}

// Create a new function to scan files in req.Files using antivirusProvider (if present)
func (s *assetManagementService) scanFilesWithAntivirus(ctx context.Context, req dto.UploadAssetRequest) error {
	for _, fileHeader := range req.Files {
		if s.antivirusProvider != nil {
			file, err := fileHeader.Open()
			if err != nil {
				continue
			}
			scanResult, scanErr := s.antivirusProvider.ScanReader(ctx, fileHeader.Filename, file)
			file.Close()
			if scanErr != nil {
				fmt.Printf("Antivirus: %s is infected or unreadable: %s\n", scanResult.FileName, scanResult.Threat)
				return fmt.Errorf("file '%s' is infected", scanResult.FileName)
			}
		}
	}
	return nil
}

func (s *assetManagementService) Upload(ctx context.Context, req dto.UploadAssetRequest, schema string) ([]tenant.Assets, error) {
	var records []dto.AssetInsertion

	err := s.scanFilesWithAntivirus(ctx, req)
	if err != nil {
		return nil, err
	}

	for _, fileHeader := range req.Files {
		asset, err := s.processAndUploadFile(fileHeader, schema)
		if err != nil {
			return nil, err
		}
		records = append(records, asset)
	}
	insertedAssets, err := s.assetsService.AssetBulkInsertion(ctx, records, schema)
	if err != nil {
		return nil, err
	}
	return insertedAssets, nil
}

func (s *assetManagementService) UploadImage(ctx context.Context, req dto.UploadAssetRequest, schema string) ([]tenant.Assets, error) {
	// Validate that there is exactly one file
	if len(req.Files) != 1 {
		return nil, fmt.Errorf("exactly one image file is required")
	}

	fileHeader := req.Files[0]

	// Validate that the file is an image
	contentType := fileHeader.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return nil, fmt.Errorf("file '%s' is not an image", fileHeader.Filename)
	}

	err := s.scanFilesWithAntivirus(ctx, req)
	if err != nil {
		return nil, err
	}

	asset, err := s.processAndUploadFile(fileHeader, schema)
	if err != nil {
		return nil, err
	}

	insertedAsset, err := s.assetsService.AssetInsertion(ctx, asset, schema)
	if err != nil {
		return nil, err
	}
	return []tenant.Assets{insertedAsset}, nil
}

func (s *assetManagementService) GetBulkAssets(ctx context.Context, schemaName string, ids []string) ([]tenant.Assets, error) {
	return s.assetsService.GetBulkAssets(ctx, schemaName, ids)
}

func (s *assetManagementService) UpdateAsset(ctx context.Context, assetId string, assetData dto.AssetUpdate, schemaName string) (tenant.Assets, error) {
	return s.assetsService.AssetUpdate(ctx, assetId, assetData, schemaName)
}

func (s *assetManagementService) DeleteAsset(ctx context.Context, assetId string, schemaName string) error {

	asset, err := s.assetsService.GetAssetByID(ctx, assetId, schemaName)
	if err != nil {
		return err
	}

	// Delete thumbnail if it exists and is different from the main file
	if asset.ThumbnailUrl != "" && asset.ThumbnailUrl != asset.Url {
		// Extract thumbnail path from URL if needed
		// The thumbnail BasePath would be like "schema/thumb_filename.jpg"
		thumbnailBasePath := strings.Replace(asset.BasePath, filepath.Base(asset.BasePath), "thumb_"+filepath.Base(asset.BasePath), 1)

		// Try to delete thumbnail, but don't fail if it doesn't exist
		thumbnailErr := s.storageProviderService.Delete(ctx, thumbnailBasePath)
		if thumbnailErr != nil {
			fmt.Printf("Warning: failed to delete thumbnail at %s: %v\n", thumbnailBasePath, thumbnailErr)
			// Continue anyway - thumbnail deletion failure shouldn't block asset deletion
		}
	}

	// Delete the main file
	err = s.storageProviderService.Delete(ctx, asset.BasePath)
	if err != nil {
		// If file not found, continue anyway - file may have been deleted manually
		if !strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("failed to delete asset file: %w", err)
		}
		fmt.Printf("Warning: asset file not found at %s, continuing with deletion\n", asset.BasePath)
	}

	// Delete from database
	err = s.assetsService.DeleteAsset(ctx, assetId, schemaName)
	if err != nil {
		return fmt.Errorf("failed to delete asset from database: %w", err)
	}

	return nil
}

func (s *assetManagementService) GetAssetByURL(ctx context.Context, schemaName string, url string) (tenant.Assets, error) {
	return s.assetsService.GetAssetByURL(ctx, url, schemaName)
}
