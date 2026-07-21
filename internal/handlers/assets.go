// Copyright 2026-2030 Aptlogica Technologies Pvt Ltd
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package handlers

import (
	"strings"

	"github.com/aptlogica/sereni-base/internal/config"
	"github.com/aptlogica/sereni-base/internal/dto"
	"github.com/aptlogica/sereni-base/internal/services/interfaces"

	"github.com/aptlogica/sereni-base/internal/handlers/validators"
	"github.com/aptlogica/sereni-base/internal/utils/response"
	responseConst "github.com/aptlogica/sereni-base/internal/utils/response/constants"

	_ "github.com/aptlogica/sereni-base/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type AssetsHandler struct {
	assetManagementService interfaces.AssetManagementService
}

func NewAssetsHandler(assetManagementService interfaces.AssetManagementService) *AssetsHandler {
	return &AssetsHandler{assetManagementService: assetManagementService}
}

// Upload handles asset upload requests.
// @Summary      Upload multiple assets
// @Description  Accepts multipart files (images/documents) and stores them while validating size and type limits.
// @Tags         Admin Asset
// @Accept       multipart/form-data
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        files  formData  file  true  "Files to upload"
// @Success      200    {object}  models.SuccessResponse  "Stored assets list returned in success.data"
// @Failure      400    {object}  models.ErrorResponse  "Bad Request — invalid form or missing files"
// @Failure      401    {object}  models.ErrorResponse  "Unauthorized"
// @Failure      413    {object}  models.ErrorResponse  "Payload Too Large — file exceeds limit"
// @Failure      500    {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /asset/upload [post]
func (h *AssetsHandler) Upload(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		response.SendError(c, responseConst.AssetError.MultipartFormNotFound)
		return
	}

	files, filesErr := form.File["files"]
	if !filesErr || files == nil || len(files) == 0 {
		response.SendError(c, responseConst.AssetError.FilesNotFound)
		return
	}

	maxSize := int64(config.AppConfig.Asset.MaxSize) // e.g. 5242880 (5MB)
	// validate each file
	for _, file := range files {
		if file.Size > maxSize {
			response.SendError(c, responseConst.AssetError.FileTooLargeError)
			return
		}
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	req := dto.UploadAssetRequest{
		Files: files,
	}

	records, err := h.assetManagementService.Upload(c, req, schemaName)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.AssetSuccess.AssetUpload, records)
}

// UploadImage handles image upload requests (single image only).
// @Summary      Upload a single image asset
// @Description  Accepts exactly one image file, validates the MIME type, and stores it alongside metadata.
// @Tags         Admin Asset
// @Accept       multipart/form-data
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        files  formData  file  true  "Single image to upload"
// @Success      200    {object}  models.SuccessResponse  "Stored image asset returned in success.data"
// @Failure      400    {object}  models.ErrorResponse  "Bad Request — missing image or invalid format"
// @Failure      401    {object}  models.ErrorResponse  "Unauthorized"
// @Failure      413    {object}  models.ErrorResponse  "Payload Too Large — file exceeds size limit"
// @Failure      500    {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /asset/upload-image [post]
func (h *AssetsHandler) UploadImage(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		response.SendError(c, responseConst.AssetError.MultipartFormNotFound)
		return
	}

	files, filesErr := form.File["files"]
	if !filesErr || files == nil || len(files) == 0 {
		response.SendError(c, responseConst.AssetError.FilesNotFound)
		return
	}

	if len(files) > 1 {
		response.SendError(c, responseConst.AssetError.TooManyFilesError)
		return
	}

	file := files[0]
	contentType := file.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		response.SendError(c, responseConst.AssetError.InvalidFileFormat)
		return
	}

	maxSize := int64(config.AppConfig.Asset.MaxSize) // e.g. 5242880 (5MB)
	if file.Size > maxSize {
		response.SendError(c, responseConst.AssetError.FileTooLargeError)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	req := dto.UploadAssetRequest{
		Files: files,
	}

	records, err := h.assetManagementService.UploadImage(c, req, schemaName)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.AssetSuccess.AssetUpload, records)
}

// @Summary      Retrieve a batch of assets
// @Description  Returns the metadata for the requested asset IDs so the UI can render thumbnails.
// @Tags         Admin Asset
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        request  body      dto.BulkAssetRequest  true  "List of asset IDs"
// @Success      200      {object}  models.SuccessResponse     "Bulk assets returned inside data"
// @Failure      400      {object}  models.ErrorResponse  "Bad Request — empty ID list"
// @Failure      401      {object}  models.ErrorResponse  "Unauthorized"
// @Failure      404      {object}  models.ErrorResponse  "Not Found — some assets missing"
// @Failure      500      {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /asset/bulk [post]
func (h *AssetsHandler) GetBulkAssets(c *gin.Context) {
	var req dto.BulkAssetRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			response.SendError(c, validators.BulkInsertValidationError(ve[0]))
			return
		}
		response.CheckAndSendError(c, err)
		return
	}

	if len(req.IDs) == 0 {
		response.SendError(c, responseConst.AssetError.InvalidRequest)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	assets, err := h.assetManagementService.GetBulkAssets(c, schemaName, req.IDs)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.AssetSuccess.AssetFetch, assets)
}

// @Summary      Update asset metadata
// @Description  Updates the title or other metadata for an existing asset record.
// @Tags         Admin Asset
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id       path      string         true  "Asset ID"
// @Param        request  body      dto.AssetUpdate  true  "Fields to update"
// @Success      200      {object}  models.SuccessResponse  "Updated asset returned inside success.data"
// @Failure      400      {object}  models.ErrorResponse  "Bad Request — invalid asset ID"
// @Failure      401      {object}  models.ErrorResponse  "Unauthorized"
// @Failure      404      {object}  models.ErrorResponse  "Not Found — asset missing"
// @Failure      422      {object}  models.ErrorResponse  "Unprocessable Entity — invalid title"
// @Failure      500      {object}  models.ErrorResponse  "Internal Server Error"
// @Security     BearerAuth
// @Router       /asset/{id} [patch]
func (h *AssetsHandler) UpdateAssetByID(c *gin.Context) {
	assetId := c.Param("id")
	if assetId == "" {
		response.SendError(c, responseConst.AssetError.InvalidRequest)
		return
	}

	if _, err := uuid.Parse(assetId); err != nil {
		response.SendError(c, responseConst.AssetError.InvalidRequest)
		return
	}

	var req dto.AssetUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	updatedAsset, err := h.assetManagementService.UpdateAsset(c, assetId, req, schemaName)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.AssetSuccess.AssetUpdated, updatedAsset)
}

// @Summary      Delete an asset
// @Description  Removes the asset record and underlying storage references for the given ID.
// @Tags         Admin Asset
// @Accept       json
// @Produce      json
// @Param        X-Request-ID  header  string  false  "Optional client-generated request trace ID"
// @Param        id   path      string  true  "Asset ID"
// @Success      200  {object}  models.SuccessResponse  "Asset deleted successfully"
// @Failure      400  {object}  models.ErrorResponse    "Bad Request — invalid id"
// @Failure      401  {object}  models.ErrorResponse    "Unauthorized"
// @Failure      403  {object}  models.ErrorResponse    "Forbidden — insufficient permissions"
// @Failure      404  {object}  models.ErrorResponse    "Not Found — asset not found"
// @Failure      500  {object}  models.ErrorResponse    "Internal Server Error"
// @Security     BearerAuth
// @Router       /asset/{id} [delete]
func (h *AssetsHandler) DeleteAssetByID(c *gin.Context) {
	assetId := c.Param("id")
	if assetId == "" {
		response.SendError(c, responseConst.AssetError.InvalidRequest)
		return
	}

	if _, err := uuid.Parse(assetId); err != nil {
		response.SendError(c, responseConst.AssetError.InvalidRequest)
		return
	}

	schemaNameVal, _ := c.Get("schema")
	schemaName, _ := schemaNameVal.(string)

	err := h.assetManagementService.DeleteAsset(c, assetId, schemaName)
	if err != nil {
		response.CheckAndSendError(c, err)
		return
	}

	response.SendSuccess(c, responseConst.AssetSuccess.AssetDeleted, nil)
}
