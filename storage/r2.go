package storage

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net/url"
	"strings"
	"time"

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

func (c *R2Client) GeneratePresignedUploadURL(ctx context.Context, keyPrefix string, contentType string, expiry time.Duration) (*GeneratedURLResponse, error) {
	exts, err := mime.ExtensionsByType(contentType)
	if err != nil || len(exts) == 0 {
		return nil, fmt.Errorf("invalid or unknown content type: %s", contentType)
	}
	ext := exts[0]
	key := fmt.Sprintf("%s/%s%s", strings.TrimSuffix(keyPrefix, "/"), uuid.NewString(), ext)
	reqParams := make(url.Values)
	reqParams.Set("Content-Type", contentType)

	presignedURL, err := c.mc.Presign(ctx, "PUT", c.cfg.Bucket, key, expiry, reqParams)
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

func (c *R2Client) GetPublicURL(key string) (string, error) {
	if c.cfg.PublicURL == "" {
		return "", errors.New("public URL is not configured")
	}
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(c.cfg.PublicURL, "/"), key), nil
}

func (c *R2Client) ParseURLToKey(url string) (string, error) {
	if c.cfg.PublicURL == "" {
		return "", errors.New("public URL is not configured, cannot parse")
	}
	base := strings.TrimSuffix(c.cfg.PublicURL, "/") + "/"
	if !strings.HasPrefix(url, base) {
		return "", fmt.Errorf("invalid URL prefix: %s", url)
	}
	uuidString := strings.TrimPrefix(url, base)
	if uuid, err := uuid.Parse(uuidString); err != nil || uuid.String() != uuidString {
		return "", fmt.Errorf("invalid URL format: %s", url)
	}
	return uuidString, nil
}

func (c *R2Client) Delete(ctx context.Context, key string) error {
	return c.mc.RemoveObject(ctx, c.cfg.Bucket, key, minio.RemoveObjectOptions{})
}

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
