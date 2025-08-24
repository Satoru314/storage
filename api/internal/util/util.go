package util

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	allowedMimeTypes = map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
		"image/heic": true,
	}
	
	invalidCharsRegex = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)
)

func IsAllowedMimeType(mimeType string) bool {
	return allowedMimeTypes[mimeType]
}

func GenerateObjectKey(fileName string) (string, error) {
	ext := filepath.Ext(fileName)
	baseName := strings.TrimSuffix(fileName, ext)
	
	safeBaseName := invalidCharsRegex.ReplaceAllString(baseName, "_")
	safeBaseName = strings.ReplaceAll(safeBaseName, " ", "_")
	
	if len(safeBaseName) > 100 {
		safeBaseName = safeBaseName[:100]
	}
	
	id := uuid.New()
	now := time.Now()
	
	objectKey := fmt.Sprintf("y=%d/m=%02d/d=%02d/%s_%s%s",
		now.Year(), now.Month(), now.Day(),
		id.String(), safeBaseName, ext)
	
	return objectKey, nil
}

func ValidateFileSize(size int64, maxSize int64) bool {
	return size > 0 && size <= maxSize
}