package storage

import "time"

type Config struct {
	AccountID string
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	PublicURL string
}

type UploadOptions struct {
	KeyPrefix    string
	ResizeWidth  int
	ResizeHeight int
	MaxSizeMB    int
	Url          *string
}

type PresignOptions struct {
	KeyPrefix   string
	ContentType string
	Expiry      time.Duration
	ObjectKey   string
}

type GeneratedURLResponse struct {
	PresignedURL string `json:"presigned_url"`
	PublicURL    string `json:"public_url"`
	ObjectKey    string `json:"object_key"`
}
