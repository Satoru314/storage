package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"storage-api/internal/model"
	"storage-api/internal/s3"
	"storage-api/internal/types"
	"storage-api/internal/util"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type ImageHandler struct {
	db                 *gorm.DB
	s3Client           *s3.Client
	presignPutTTLSec   int
	presignGetTTLSec   int
	maxFileSizeBytes   int64
}

func NewImageHandler(db *gorm.DB, s3Client *s3.Client, presignPutTTLSec, presignGetTTLSec int) *ImageHandler {
	return &ImageHandler{
		db:                 db,
		s3Client:           s3Client,
		presignPutTTLSec:   presignPutTTLSec,
		presignGetTTLSec:   presignGetTTLSec,
		maxFileSizeBytes:   50 * 1024 * 1024, // 50MB
	}
}

func (h *ImageHandler) UploadRequest(c echo.Context) error {
	var req types.UploadRequestReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error: types.ErrorDetail{
				Code:      "BadRequest",
				Message:   "Invalid request body",
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			},
		})
	}

	if !util.IsAllowedMimeType(req.ContentType) {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error: types.ErrorDetail{
				Code:      "BadRequest",
				Message:   "Content type not allowed",
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			},
		})
	}

	if !util.ValidateFileSize(req.FileSize, h.maxFileSizeBytes) {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error: types.ErrorDetail{
				Code:      "BadRequest",
				Message:   "File size exceeds limit",
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			},
		})
	}

	objectKey, err := util.GenerateObjectKey(req.FileName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error: types.ErrorDetail{
				Code:      "InternalError",
				Message:   "Failed to generate object key",
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			},
		})
	}

	image := model.Image{
		ID:            uuid.New(),
		ObjectKey:     objectKey,
		OriginalName:  req.FileName,
		MimeType:      req.ContentType,
		ByteSize:      req.FileSize,
		StorageStatus: model.StatusRequested,
		CreatedAt:     time.Now(),
	}

	if err := h.db.Create(&image).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error: types.ErrorDetail{
				Code:      "InternalError",
				Message:   "Failed to save image metadata",
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			},
		})
	}

	if h.s3Client == nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error: types.ErrorDetail{
				Code:      "InternalError",
				Message:   "S3 client not available",
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			},
		})
	}

	presignedURL, err := h.s3Client.PresignPutObject(
		context.Background(),
		objectKey,
		req.ContentType,
		time.Duration(h.presignPutTTLSec)*time.Second,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error: types.ErrorDetail{
				Code:      "InternalError",
				Message:   "Failed to generate presigned URL",
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			},
		})
	}

	response := types.UploadRequestRes{
		Image: types.ImageResponse{
			ID:           image.ID,
			ObjectKey:    image.ObjectKey,
			OriginalName: image.OriginalName,
			MimeType:     image.MimeType,
			ByteSize:     image.ByteSize,
			Status:       image.StorageStatus,
		},
		Upload: types.UploadResponse{
			Method: "PUT",
			URL:    presignedURL,
			Headers: map[string]string{
				"Content-Type": req.ContentType,
			},
			ExpiresInSec: h.presignPutTTLSec,
		},
	}

	return c.JSON(http.StatusOK, response)
}

func (h *ImageHandler) UploadComplete(c echo.Context) error {
	var req types.UploadCompleteReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error: types.ErrorDetail{
				Code:      "BadRequest",
				Message:   "Invalid request body",
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			},
		})
	}

	var image model.Image
	if err := h.db.Where("id = ? AND object_key = ?", req.ID, req.ObjectKey).First(&image).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, types.ErrorResponse{
				Error: types.ErrorDetail{
					Code:      "NotFound",
					Message:   "Image not found",
					RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				},
			})
		}
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error: types.ErrorDetail{
				Code:      "InternalError",
				Message:   "Database error",
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			},
		})
	}

	if image.StorageStatus != model.StatusRequested {
		return c.JSON(http.StatusConflict, types.ErrorResponse{
			Error: types.ErrorDetail{
				Code:      "Conflict",
				Message:   "Image already processed",
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			},
		})
	}

	if h.s3Client == nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error: types.ErrorDetail{
				Code:      "InternalError",
				Message:   "S3 client not available",
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			},
		})
	}

	headOutput, err := h.s3Client.HeadObject(context.Background(), image.ObjectKey)
	if err != nil {
		h.db.Model(&image).Update("storage_status", model.StatusFailed)
		return c.JSON(http.StatusNotFound, types.ErrorResponse{
			Error: types.ErrorDetail{
				Code:      "NotFound",
				Message:   "Object not found in S3",
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			},
		})
	}

	now := time.Now()
	updateData := map[string]interface{}{
		"storage_status": model.StatusUploaded,
		"uploaded_at":    &now,
	}
	
	if headOutput.ETag != nil {
		etag := *headOutput.ETag
		updateData["e_tag"] = &etag
	}

	if err := h.db.Model(&image).Updates(updateData).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error: types.ErrorDetail{
				Code:      "InternalError",
				Message:   "Failed to update image status",
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			},
		})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (h *ImageHandler) ListImages(c echo.Context) error {
	limit := 20
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	var images []model.Image
	query := h.db.Where("storage_status = ?", model.StatusUploaded).
		Order("created_at DESC").
		Limit(limit + 1) // +1 to check if there are more items

	if cursor := c.QueryParam("cursor"); cursor != "" {
		// Simple cursor implementation using created_at timestamp
		if decodedTime, err := time.Parse(time.RFC3339, cursor); err == nil {
			query = query.Where("created_at < ?", decodedTime)
		}
	}

	if err := query.Find(&images).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error: types.ErrorDetail{
				Code:      "InternalError",
				Message:   "Failed to fetch images",
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			},
		})
	}

	hasMore := len(images) > limit
	if hasMore {
		images = images[:limit] // Remove the extra item
	}

	items := make([]types.ImageMeta, len(images))
	for i, img := range images {
		items[i] = types.ImageMeta{
			ID:           img.ID,
			ObjectKey:    img.ObjectKey,
			OriginalName: img.OriginalName,
			MimeType:     img.MimeType,
			ByteSize:     img.ByteSize,
			UploadedAt:   img.UploadedAt,
		}
	}

	response := types.ImageListResponse{
		Items: items,
	}

	if hasMore && len(images) > 0 {
		nextCursor := images[len(images)-1].CreatedAt.Format(time.RFC3339)
		response.NextCursor = &nextCursor
	}

	return c.JSON(http.StatusOK, response)
}

func (h *ImageHandler) GetImage(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error: types.ErrorDetail{
				Code:      "BadRequest",
				Message:   "Invalid image ID",
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			},
		})
	}

	var image model.Image
	if err := h.db.Where("id = ? AND storage_status = ?", id, model.StatusUploaded).First(&image).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, types.ErrorResponse{
				Error: types.ErrorDetail{
					Code:      "NotFound",
					Message:   "Image not found",
					RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				},
			})
		}
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error: types.ErrorDetail{
				Code:      "InternalError",
				Message:   "Database error",
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			},
		})
	}

	response := types.ImageMeta{
		ID:           image.ID,
		ObjectKey:    image.ObjectKey,
		OriginalName: image.OriginalName,
		MimeType:     image.MimeType,
		ByteSize:     image.ByteSize,
		UploadedAt:   image.UploadedAt,
	}

	return c.JSON(http.StatusOK, response)
}

func (h *ImageHandler) ViewUrls(c echo.Context) error {
	var req types.ViewUrlsReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error: types.ErrorDetail{
				Code:      "BadRequest",
				Message:   "Invalid request body",
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			},
		})
	}

	if len(req.Requests) == 0 {
		return c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error: types.ErrorDetail{
				Code:      "BadRequest",
				Message:   "At least one request is required",
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			},
		})
	}

	if h.s3Client == nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error: types.ErrorDetail{
				Code:      "InternalError",
				Message:   "S3 client not available",
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			},
		})
	}

	ttlSec := h.presignGetTTLSec
	if req.TTLSec != nil && *req.TTLSec > 0 && *req.TTLSec <= h.presignGetTTLSec {
		ttlSec = *req.TTLSec
	}
	ttl := time.Duration(ttlSec) * time.Second

	ids := make([]uuid.UUID, len(req.Requests))
	for i, r := range req.Requests {
		ids[i] = r.ID
	}

	var images []model.Image
	if err := h.db.Where("id IN ? AND storage_status = ?", ids, model.StatusUploaded).Find(&images).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error: types.ErrorDetail{
				Code:      "InternalError",
				Message:   "Database error",
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			},
		})
	}

	imageMap := make(map[uuid.UUID]model.Image)
	for _, img := range images {
		imageMap[img.ID] = img
	}

	results := make([]types.ViewUrlResult, 0, len(req.Requests))
	expiresAt := time.Now().Add(ttl)

	for _, r := range req.Requests {
		if img, exists := imageMap[r.ID]; exists {
			presignedURL, err := h.s3Client.PresignGetObject(context.Background(), img.ObjectKey, ttl)
			if err != nil {
				continue // Skip this image if we can't generate URL
			}

			results = append(results, types.ViewUrlResult{
				ID:        r.ID,
				URL:       presignedURL,
				ExpiresAt: expiresAt,
			})
		}
	}

	return c.JSON(http.StatusOK, types.ViewUrlsRes{Results: results})
}