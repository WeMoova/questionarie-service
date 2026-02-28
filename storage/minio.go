package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOStorage handles file storage operations with MinIO
type MinIOStorage struct {
	client     *minio.Client
	bucketName string
	publicURL  string
}

// NewMinIOStorage creates a new MinIO storage client from environment variables
func NewMinIOStorage() (*MinIOStorage, error) {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	bucket := os.Getenv("MINIO_BUCKET")
	publicURL := os.Getenv("MINIO_PUBLIC_URL")

	if endpoint == "" || accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("MinIO configuration incomplete: MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY required")
	}
	if bucket == "" {
		bucket = "questionnaire-images"
	}
	if publicURL == "" {
		publicURL = fmt.Sprintf("http://%s", endpoint)
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false, // internal cluster, no TLS
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	return &MinIOStorage{
		client:     client,
		bucketName: bucket,
		publicURL:  publicURL,
	}, nil
}

// AllowedImageTypes for validation
var AllowedImageTypes = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true,
}

// MaxImageSize is 5MB
const MaxImageSize = 5 * 1024 * 1024

// UploadImage uploads an image and returns the relative object path
func (s *MinIOStorage) UploadImage(ctx context.Context, reader io.Reader, size int64, filename string, folder string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if !AllowedImageTypes[ext] {
		return "", fmt.Errorf("unsupported image type: %s (allowed: jpg, jpeg, png, webp, gif)", ext)
	}
	if size > MaxImageSize {
		return "", fmt.Errorf("image too large: max %d MB", MaxImageSize/(1024*1024))
	}

	objectName := fmt.Sprintf("%s/%s-%s%s", folder, time.Now().Format("20060102"), uuid.New().String()[:8], ext)

	contentType := "image/" + strings.TrimPrefix(ext, ".")
	if ext == ".jpg" {
		contentType = "image/jpeg"
	}

	_, err := s.client.PutObject(ctx, s.bucketName, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload image: %w", err)
	}

	return objectName, nil
}

// GetObject retrieves an object from MinIO for streaming
func (s *MinIOStorage) GetObject(ctx context.Context, objectName string) (*minio.Object, error) {
	return s.client.GetObject(ctx, s.bucketName, objectName, minio.GetObjectOptions{})
}

// DeleteImage deletes an image by its object path
func (s *MinIOStorage) DeleteImage(ctx context.Context, objectName string) error {
	return s.client.RemoveObject(ctx, s.bucketName, objectName, minio.RemoveObjectOptions{})
}

// HealthCheck checks if MinIO is accessible
func (s *MinIOStorage) HealthCheck(ctx context.Context) error {
	_, err := s.client.BucketExists(ctx, s.bucketName)
	return err
}
