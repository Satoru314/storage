package types

import (
	"time"

	"github.com/google/uuid"
)

type UploadRequestReq struct {
	FileName    string `json:"fileName" validate:"required"`
	ContentType string `json:"contentType" validate:"required"`
	FileSize    int64  `json:"fileSize" validate:"required,min=1"`
}

type UploadRequestRes struct {
	Image  ImageResponse  `json:"image"`
	Upload UploadResponse `json:"upload"`
}

type ImageResponse struct {
	ID           uuid.UUID `json:"id"`
	ObjectKey    string    `json:"objectKey"`
	OriginalName string    `json:"originalName"`
	MimeType     string    `json:"mimeType"`
	ByteSize     int64     `json:"byteSize"`
	Status       string    `json:"status"`
}

type UploadResponse struct {
	Method       string            `json:"method"`
	URL          string            `json:"url"`
	Headers      map[string]string `json:"headers"`
	ExpiresInSec int               `json:"expiresInSec"`
}

type UploadCompleteReq struct {
	ID        uuid.UUID `json:"id" validate:"required"`
	ObjectKey string    `json:"objectKey" validate:"required"`
}

type ImageMeta struct {
	ID           uuid.UUID  `json:"id"`
	ObjectKey    string     `json:"objectKey"`
	OriginalName string     `json:"originalName"`
	MimeType     string     `json:"mimeType"`
	ByteSize     int64      `json:"byteSize"`
	UploadedAt   *time.Time `json:"uploadedAt,omitempty"`
}

type ImageListResponse struct {
	Items      []ImageMeta `json:"items"`
	NextCursor *string     `json:"nextCursor,omitempty"`
}

type ViewUrlsReq struct {
	Requests []ViewUrlRequest `json:"requests" validate:"required,dive"`
	TTLSec   *int             `json:"ttlSec,omitempty"`
}

type ViewUrlRequest struct {
	ID uuid.UUID `json:"id" validate:"required"`
}

type ViewUrlsRes struct {
	Results []ViewUrlResult `json:"results"`
}

type ViewUrlResult struct {
	ID        uuid.UUID `json:"id"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}