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
	ObjectKey   *string
	FileName    string
}

type GeneratedURLResponse struct {
	PresignedURL string `json:"presigned_url"`
	PublicURL    string `json:"public_url"`
	ObjectKey    string `json:"object_key"`
}

type UploadFileOptions struct {
	KeyPrefix string
	MaxSizeMB int
}

type UploadFileResult struct {
	FileName  string `json:"file_name"`
	ObjectKey string `json:"object_key"`
	PublicURL string `json:"public_url"`
	Error     string `json:"error,omitempty"`
}
