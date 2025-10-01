package storage

import (
	"context"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client struct {
	mc  *minio.Client
	cfg Config
}

func NewClient(cfg Config) (*Client, error) {
	endpoint := strings.TrimPrefix(cfg.Endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")

	mc, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: "",
	})
	if err != nil {
		return nil, err
	}
	return &Client{mc: mc, cfg: cfg}, nil
}

type UploadOptions struct {
	KeyPrefix    string
	Filename     string
	ResizeWidth  int
	ResizeHeight int
	Quality      int
}

type UploadResult struct {
	Key  string
	URL  string
	Size int64
}

func (c *Client) UploadImage(ctx context.Context, src io.Reader, opts UploadOptions) (*UploadResult, error) {
	rs, size, contentType, err := processImage(src, opts)
	if err != nil {
		return nil, err
	}

	ext := filepath.Ext(opts.Filename)
	if ext == "" {
		if exts, _ := mime.ExtensionsByType(contentType); len(exts) > 0 {
			ext = exts[0]
		} else {
			ext = ".jpg"
		}
	}

	var key string
	if opts.KeyPrefix != "" {
		key = fmt.Sprintf("%s/%s%s", strings.TrimSuffix(opts.KeyPrefix, "/"), uuid.NewString(), ext)
	} else {
		key = fmt.Sprintf("%s%s", uuid.NewString(), ext)
	}

	info, err := c.mc.PutObject(ctx, c.cfg.Bucket, key, rs, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return nil, err
	}

	return &UploadResult{
		Key:  key,
		URL:  c.PublicURL(key),
		Size: info.Size,
	}, nil
}

func (c *Client) Delete(ctx context.Context, key string) error {
	return c.mc.RemoveObject(ctx, c.cfg.Bucket, key, minio.RemoveObjectOptions{})
}

func (c *Client) PublicURL(key string) string {
	return fmt.Sprintf("https://%s.r2.cloudflarestorage.com/%s/%s", c.cfg.AccountID, c.cfg.Bucket, key)
}

func (c *Client) DeleteMany(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	objCh := make(chan minio.ObjectInfo, len(keys))
	for _, k := range keys {
		objCh <- minio.ObjectInfo{Key: k}
	}
	close(objCh)

	errs := c.mc.RemoveObjects(ctx, c.cfg.Bucket, objCh, minio.RemoveObjectsOptions{})

	// gom lỗi trả về
	var failed []string
	for e := range errs {
		failed = append(failed, fmt.Sprintf("%s: %v", e.ObjectName, e.Err))
	}
	if len(failed) > 0 {
		return fmt.Errorf("delete many failed: %v", strings.Join(failed, "; "))
	}
	return nil
}
