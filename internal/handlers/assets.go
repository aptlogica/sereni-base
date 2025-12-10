package handlers

import (
	"serenibase/internal/config"
	"serenibase/internal/dto"
	"serenibase/internal/services/interfaces"
	"strings"

	"serenibase/internal/handlers/validators"
	"serenibase/internal/utils/response"
	responseConst "serenibase/internal/utils/response/constants"

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
