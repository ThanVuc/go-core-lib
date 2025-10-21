package storage

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
	KeyPrefix string
	Url       *string
}

type GeneratedURLResponse struct {
	PresignedURL string `json:"presigned_url"`
	PublicURL    string `json:"public_url"`
	ObjectKey    string `json:"object_key"`
}
