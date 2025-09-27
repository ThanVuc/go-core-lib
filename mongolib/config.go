package mongolib

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

type MongoConnectorConfig struct {
	URI                string
	Database           string
	Username           string
	Password           string
	Timeout            *time.Duration
	ConnectTimeout     *time.Duration
	MaxPoolSize        *uint64
	MinPoolSize        *uint64
	MaxConnIdleTime    *time.Duration
	MaxConnecting      *uint64
	TransactionTimeout *time.Duration
	WriteConcern       *writeconcern.WriteConcern
	ReadConcern        *readconcern.ReadConcern
}
