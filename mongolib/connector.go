package mongolib

import (
	"context"
	"sync"
	"time"

	"github.com/thanvuc/go-core-lib/utils"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

type MongoConnector struct {
	Client   *mongo.Client
	Database *mongo.Database
	cfg      MongoConnectorConfig
}

// NewMongoConnector creates a new MongoDB connector based on the provided configuration.
// It establishes a connection to the MongoDB server and returns a MongoConnector instance.
// If the connection fails, it returns an error.
// The function also pings the MongoDB server to ensure the connection is valid.
func NewMongoConnector(ctx context.Context, cfg MongoConnectorConfig) (*MongoConnector, error) {
	withDefaults(&cfg)

	clientOptions := options.Client().ApplyURI(cfg.URI)
	clientOptions.Auth = &options.Credential{
		Username:   cfg.Username,
		Password:   cfg.Password,
		AuthSource: cfg.Database,
	}
	// Set the configuration options
	configureConnector(cfg, clientOptions)

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, err
	}

	// Ping the primary
	ctx, cancel := context.WithTimeout(ctx, *cfg.Timeout)
	defer cancel()

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	database := client.Database(cfg.Database)

	return &MongoConnector{
		Client:   client,
		Database: database,
		cfg:      cfg,
	}, nil
}

// HealthCheck pings the MongoDB server to check if the connection is alive.
func (c *MongoConnector) HealthCheck(ctx context.Context) error {
	return c.Client.Ping(ctx, nil)
}

// configureConnector sets the MongoDB client options based on the provided configuration.
// It applies default values for options that are not explicitly set in the configuration.
func configureConnector(cfg MongoConnectorConfig, clientOptions *options.ClientOptions) {
	clientOptions.ConnectTimeout = cfg.ConnectTimeout
	clientOptions.Timeout = cfg.Timeout
	clientOptions.MaxPoolSize = cfg.MaxPoolSize
	clientOptions.MinPoolSize = cfg.MinPoolSize
	clientOptions.MaxConnIdleTime = cfg.MaxConnIdleTime
	clientOptions.MaxConnecting = cfg.MaxConnecting
}

// GracefulClose disconnects the MongoDB client and cleans up resources.
// It should be called when the application is shutting down to ensure a clean disconnection.
// The function takes a context and a wait group to synchronize the shutdown process.
// It returns an error if the disconnection fails.
func (c *MongoConnector) GracefulClose(ctx context.Context, wg *sync.WaitGroup) error {
	if wg != nil {
		defer wg.Done()
	}

	if c.Client != nil {
		err := c.Client.Disconnect(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetCollection retrieves a MongoDB collection by its name.
// It returns a pointer to the mongo.Collection instance.
func (c *MongoConnector) GetCollection(name string) *mongo.Collection {
	return c.Database.Collection(name)
}

// WithTransaction executes the provided function within a MongoDB transaction.
// It handles the transaction lifecycle, including starting, committing, and aborting the transaction.
// The function takes a context and a callback function that contains the operations to be executed within the transaction.
// It returns the result of the callback function and any error that occurred during the transaction.
func (c *MongoConnector) WithTransaction(ctx context.Context, fn func(fnCtx context.Context) (any, error)) (any, error) {
	wc := c.cfg.WriteConcern
	rc := c.cfg.ReadConcern
	txnOpts := options.Transaction().SetWriteConcern(wc).SetReadConcern(rc)

	session, err := c.Client.StartSession()
	if err != nil {
		return nil, err
	}
	defer session.EndSession(ctx)

	txnCtx, cancel := context.WithTimeout(ctx, utils.Ternary(c.cfg.TransactionTimeout != nil, *c.cfg.TransactionTimeout, 15*time.Second))
	defer cancel()

	result, err := session.WithTransaction(txnCtx, fn, txnOpts)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// withDefaults fills in missing config fields with sane defaults.
// It modifies the receiver and returns it for chaining.
func withDefaults(cfg *MongoConnectorConfig) {
	if cfg.Timeout == nil {
		d := 10 * time.Second
		cfg.Timeout = &d
	}
	if cfg.ConnectTimeout == nil {
		d := 10 * time.Second
		cfg.ConnectTimeout = &d
	}
	if cfg.MaxPoolSize == nil {
		v := uint64(100)
		cfg.MaxPoolSize = &v
	}
	if cfg.MinPoolSize == nil {
		v := uint64(5)
		cfg.MinPoolSize = &v
	}
	if cfg.MaxConnIdleTime == nil {
		d := 30 * time.Second
		cfg.MaxConnIdleTime = &d
	}
	if cfg.MaxConnecting == nil {
		v := uint64(2)
		cfg.MaxConnecting = &v
	}
	if cfg.TransactionTimeout == nil {
		d := 15 * time.Second
		cfg.TransactionTimeout = &d
	}
	if cfg.WriteConcern == nil {
		cfg.WriteConcern = writeconcern.Majority()
	}
	if cfg.ReadConcern == nil {
		cfg.ReadConcern = readconcern.Snapshot()
	}
}

// IsConnected checks if the MongoDB client is connected.
func (c *MongoConnector) IsConnected() bool {
	return c.Client != nil
}
