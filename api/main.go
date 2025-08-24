package main

import (
	"log"

	"storage-api/internal/config"
	"storage-api/internal/database"
	"storage-api/internal/handler"
	"storage-api/internal/model"
	"storage-api/internal/s3"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	cfg := config.Load()

	// Debug: 設定値を確認
	log.Printf("Debug: AWSAccessKeyID='%s', AWSSecretAccessKey='%s'", cfg.AWSAccessKeyID, cfg.AWSSecretAccessKey)

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := database.AutoMigrate(db, &model.Image{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	var s3Client *s3.Client
	if cfg.AWSAccessKeyID != "" && cfg.AWSSecretAccessKey != "" {
		s3Client, err = s3.NewClient(cfg.AWSRegion, cfg.S3Bucket, cfg.AWSAccessKeyID, cfg.AWSSecretAccessKey)
		if err != nil {
			log.Printf("Warning: Failed to initialize S3 client: %v", err)
		}
	} else {
		log.Println("Warning: AWS credentials not provided. S3 functionality will be disabled.")
	}

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{cfg.AllowedOrigin},
		AllowMethods: []string{echo.GET, echo.POST, echo.OPTIONS},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	healthHandler := handler.NewHealthHandler()
	e.GET("/healthz", healthHandler.GetHealth)

	imageHandler := handler.NewImageHandler(db, s3Client, cfg.PresignPutTTLSec, cfg.PresignGetTTLSec)
	e.POST("/images/upload-request", imageHandler.UploadRequest)
	e.POST("/images/upload-complete", imageHandler.UploadComplete)
	e.GET("/images", imageHandler.ListImages)
	e.GET("/images/:id", imageHandler.GetImage)
	e.POST("/images/view-urls", imageHandler.ViewUrls)

	log.Printf("Server starting on port %s", cfg.Port)
	if err := e.Start(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
