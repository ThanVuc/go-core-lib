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
	KeyPrefix    string
	ResizeWidth  int
	ResizeHeight int
	MaxSizeMB    int
	Url          *string
}
