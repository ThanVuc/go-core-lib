package storage

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"net/url"
	"path/filepath"
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

func (c *R2Client) GeneratePresignedURL(ctx context.Context, otps PresignOptions) (*GeneratedURLResponse, error) {
	var key string
	if otps.ObjectKey != nil {
		key = *otps.ObjectKey
	} else {
		fileName := c.sanitizeFileName(otps.FileName)
		ext := filepath.Ext(fileName)

		if ext == "" {
			ext = c.getExtension(otps.ContentType)
		}

		name := strings.TrimSuffix(fileName, ext)

		timestamp := time.Now().UnixMilli()

		newFileName := fmt.Sprintf("%s-%d%s", name, timestamp, ext)

		key = fmt.Sprintf(
			"%s/%s",
			strings.TrimSuffix(otps.KeyPrefix, "/"),
			newFileName,
		)
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

func (c *R2Client) getExtension(contentType string) string {
	exts, err := mime.ExtensionsByType(contentType)
	if err == nil && len(exts) > 0 {
		return exts[0]
	}

	switch contentType {
	case "text/markdown", "text/x-markdown":
		return ".md"
	case "text/plain":
		return ".txt"
	case "text/csv":
		return ".csv"
	case "application/json":
		return ".json"
	case "image/webp":
		return ".webp"
	default:
		return ".bin"
	}
}

func (c *R2Client) sanitizeFileName(name string) string {
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "..", "")
	name = strings.ReplaceAll(name, ";", "_")
	return name
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

func (c *R2Client) UploadFiles(
	ctx context.Context,
	files []*multipart.FileHeader,
	opts UploadFileOptions,
) ([]UploadFileResult, error) {

	if len(files) == 0 {
		return nil, nil
	}

	results := make([]UploadFileResult, len(files))

	// limit concurrent (tránh spam goroutine)
	sem := make(chan struct{}, 5) // max 5 concurrent
	done := make(chan struct{})

	for i, file := range files {
		i := i
		file := file

		go func() {
			sem <- struct{}{}
			defer func() {
				<-sem
				done <- struct{}{}
			}()

			result := UploadFileResult{
				FileName: file.Filename,
			}

			// check size
			if opts.MaxSizeMB > 0 && file.Size > int64(opts.MaxSizeMB)*1024*1024 {
				result.Error = fmt.Sprintf("file too large (max %dMB)", opts.MaxSizeMB)
				results[i] = result
				return
			}

			src, err := file.Open()
			if err != nil {
				result.Error = err.Error()
				results[i] = result
				return
			}
			defer src.Close()

			// detect content type
			contentType := file.Header.Get("Content-Type")
			if contentType == "" {
				contentType = "application/octet-stream"
			}

			// lấy extension
			ext := filepath.Ext(file.Filename)
			if ext == "" {
				exts, _ := mime.ExtensionsByType(contentType)
				if len(exts) > 0 {
					ext = exts[0]
				}
			}

			// generate key
			key := fmt.Sprintf("%s/%s%s",
				strings.TrimSuffix(opts.KeyPrefix, "/"),
				uuid.NewString(),
				ext,
			)

			// upload trực tiếp (stream)
			_, err = c.mc.PutObject(ctx, c.cfg.Bucket, key, src, file.Size, minio.PutObjectOptions{
				ContentType: contentType,
			})
			if err != nil {
				result.Error = err.Error()
				results[i] = result
				return
			}

			publicURL, err := c.GetPublicURL(key)
			if err != nil {
				result.Error = err.Error()
				results[i] = result
				return
			}

			result.ObjectKey = key
			result.PublicURL = publicURL
			results[i] = result
		}()
	}

	// wait all
	for range files {
		<-done
	}

	return results, nil
}

func (c *R2Client) ReplaceFile(ctx context.Context, oldURL string, file *multipart.FileHeader, prefix string) (string, error) {
	// new upload
	results, err := c.UploadFiles(ctx, []*multipart.FileHeader{file}, UploadFileOptions{
		KeyPrefix: prefix,
		MaxSizeMB: 10, // optional limit
	})
	if err != nil {
		return "", err
	}

	newURL := results[0].PublicURL

	// delete old file if oldURL provided
	if oldURL != "" {
		_ = c.Delete(ctx, oldURL) // ignore error optional
	}

	return newURL, nil
}

// GenerateMultiplePresignedURLs
func (c *R2Client) GenerateMultiplePresignedURLs(ctx context.Context, otps []PresignOptions) ([]GeneratedURLResponse, error) {
	responses := make([]GeneratedURLResponse, len(otps))
	for i, otp := range otps {
		resp, err := c.GeneratePresignedURL(ctx, otp)
		if err != nil {
			return nil, fmt.Errorf("failed to generate presigned URL for index %d: %w", i, err)
		}
		responses[i] = *resp
	}
	return responses, nil
}
