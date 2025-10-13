package storage

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type R2Client struct {
	mc  *minio.Client
	cfg Config
}

func (c *R2Client) Config() {
	panic("unimplemented")
}

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

type UploadOptions struct {
	KeyPrefix    string
	ResizeWidth  int
	ResizeHeight int
	Quality      int
	MaxSizeMB    int
}

type UploadResult struct {
	Key  string
	URL  string
	Size int64
}

func (c *R2Client) UploadImage(ctx context.Context, file *multipart.FileHeader, opts UploadOptions) (*UploadResult, error) {
	if opts.MaxSizeMB >0 && file.Size > int64(opts.MaxSizeMB)*1024*1024 {
		return nil, fmt.Errorf("file vượt quá giới hạn %dMB", opts.MaxSizeMB)
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

	ext := ".webp"
	key := fmt.Sprintf("%s/%s%s", strings.TrimSuffix(opts.KeyPrefix, "/"), uuid.NewString(), ext)

	info, err := c.mc.PutObject(ctx, c.cfg.Bucket, key, rs, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return nil, err
	}

	url, err := c.GetPublicURL(key)
	if err != nil {
		return nil, err
	}

	return &UploadResult{
		Key:  key,
		URL:  url,
		Size: info.Size,
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
	return strings.TrimPrefix(url, base), nil
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
