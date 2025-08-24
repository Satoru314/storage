package model

import (
	"time"

	"github.com/google/uuid"
)

type Image struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	ObjectKey     string     `gorm:"uniqueIndex;not null" json:"objectKey"`
	OriginalName  string     `gorm:"not null" json:"originalName"`
	MimeType      string     `gorm:"not null" json:"mimeType"`
	ByteSize      int64      `gorm:"not null" json:"byteSize"`
	Width         *int       `json:"width,omitempty"`
	Height        *int       `json:"height,omitempty"`
	ETag          *string    `json:"etag,omitempty"`
	StorageStatus string     `gorm:"not null;check:storage_status IN ('requested','uploaded','failed','deleted')" json:"status"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"createdAt"`
	UploadedAt    *time.Time `json:"uploadedAt,omitempty"`
}

func (Image) TableName() string {
	return "images"
}

const (
	StatusRequested = "requested"
	StatusUploaded  = "uploaded"
	StatusFailed    = "failed"
	StatusDeleted   = "deleted"
)