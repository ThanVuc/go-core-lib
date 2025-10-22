package storage

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type R2Client struct {
	mc  *minio.Client
	cfg Config
}

// Create a configuration struct for R2Client
func NewClient(cfg Config) (*R2Client, error) {
	endpoint := strings.TrimPrefix(cfg.Endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")

	mc, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, err
	}
	return &R2Client{mc: mc, cfg: cfg}, nil
}

// Generate a presigned URL for uploading or updating a new object.
func (c *R2Client) GeneratePresignedURL(ctx context.Context, otps PresignOptions) (*GeneratedURLResponse, error) {
	exts, err := mime.ExtensionsByType(otps.ContentType)
	if err != nil || len(exts) == 0 {
		return nil, fmt.Errorf("invalid or unknown content type: %s", otps.ContentType)
	}
	ext := exts[0]

	var key string
	if otps.ObjectKey != nil {
		key = *otps.ObjectKey
	} else {
		key = fmt.Sprintf("%s/%s%s", strings.TrimSuffix(otps.KeyPrefix, "/"), uuid.NewString(), ext)
	}

	reqParams := make(url.Values)
	reqParams.Set("Content-Type", otps.ContentType)

	presignedURL, err := c.mc.Presign(ctx, "PUT", c.cfg.Bucket, key, otps.Expiry, reqParams)
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	publicURL, err := c.GetPublicURL(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get public URL: %w", err)
	}

	return &GeneratedURLResponse{
		PresignedURL: presignedURL.String(),
		PublicURL:    publicURL,
		ObjectKey:    key,
	}, nil
}

// If uploading the image with the existing key, it may overwrite the existing file.(Update)
func (c *R2Client) UploadImage(ctx context.Context, file *multipart.FileHeader, opts UploadOptions) (*string, error) {
	if opts.MaxSizeMB > 0 && file.Size > int64(opts.MaxSizeMB)*1024*1024 {
		return nil, fmt.Errorf("limited: %dMB", opts.MaxSizeMB)
	}

	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	rs, size, contentType, err := processImage(src, opts)
	if err != nil {
		return nil, err
	}

	var key string
	if opts.Url != nil {
		key, err = c.ParseURLToKey(*opts.Url)
		if err != nil {
			return nil, err
		}
	} else {
		ext := ".webp"
		key = fmt.Sprintf("%s/%s%s", strings.TrimSuffix(opts.KeyPrefix, "/"), uuid.NewString(), ext)
	}

	_, err = c.mc.PutObject(ctx, c.cfg.Bucket, key, rs, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return nil, err
	}

	url, err := c.GetPublicURL(key)
	if err != nil {
		return nil, err
	}

	return &url, nil
}

// GetPublicURL constructs the public URL for a given object key.
func (c *R2Client) GetPublicURL(key string) (string, error) {
	if c.cfg.PublicURL == "" {
		return "", errors.New("public URL is not configured")
	}
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(c.cfg.PublicURL, "/"), key), nil
}

// ParseURLToKey extracts the object key from a given public URL.
func (c *R2Client) ParseURLToKey(url string) (string, error) {
	if c.cfg.PublicURL == "" {
		return "", errors.New("public URL is not configured, cannot parse")
	}

	base := strings.TrimSuffix(c.cfg.PublicURL, "/") + "/"
	if !strings.HasPrefix(url, base) {
		return "", fmt.Errorf("invalid URL prefix: %s", url)
	}
	key := strings.TrimPrefix(url, base)
	if key == "" {
		return "", fmt.Errorf("empty key parsed from URL: %s", url)
	}
	return key, nil
}

// Delete an object by its key.
func (c *R2Client) Delete(ctx context.Context, key string) error {
	return c.mc.RemoveObject(ctx, c.cfg.Bucket, key, minio.RemoveObjectOptions{})
}

// DeleteMany deletes multiple objects by their keys.
func (c *R2Client) DeleteMany(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	objCh := make(chan minio.ObjectInfo, len(keys))
	for _, k := range keys {
		objCh <- minio.ObjectInfo{Key: k}
	}
	close(objCh)

	errs := c.mc.RemoveObjects(ctx, c.cfg.Bucket, objCh, minio.RemoveObjectsOptions{})
	var failed []string
	for e := range errs {
		failed = append(failed, fmt.Sprintf("%s: %v", e.ObjectName, e.Err))
	}
	if len(failed) > 0 {
		return fmt.Errorf("delete many failed: %v", strings.Join(failed, "; "))
	}
	return nil
}
