package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port                string
	DatabaseURL         string
	AWSRegion          string
	S3Bucket           string
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	PresignPutTTLSec   int
	PresignGetTTLSec   int
	AllowedOrigin      string
}

func Load() *Config {
	return &Config{
		Port:                getEnv("PORT", "8080"),
		DatabaseURL:         getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/images?sslmode=disable"),
		AWSRegion:          getEnv("AWS_REGION", "ap-northeast-1"),
		S3Bucket:           getEnv("S3_BUCKET", "app-images"),
		AWSAccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
		PresignPutTTLSec:   getEnvInt("PRESIGN_PUT_TTL_SEC", 300),
		PresignGetTTLSec:   getEnvInt("PRESIGN_GET_TTL_SEC", 300),
		AllowedOrigin:      getEnv("ALLOWED_ORIGIN", "http://localhost:5173"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}