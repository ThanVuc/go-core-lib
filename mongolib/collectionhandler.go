package mongolib

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// CreateCollection creates a new collection with the specified name, JSON schema validator, and indexes.
func (c *MongoConnector) CreateCollection(ctx context.Context, name string, validator bson.M, indexes []mongo.IndexModel) error {
	err := c.Database.CreateCollection(ctx, name)
	if err != nil {
		// Ignore "collection already exists"
		if !isNamespaceExistsError(err) {
			return err
		}
	}

	if validator != nil {
		if err := c.updateValidator(ctx, name, validator); err != nil {
			return err
		}
	}

	if len(indexes) > 0 {
		if err := c.ensureIndexes(ctx, name, indexes); err != nil {
			return err
		}
	}

	return nil
}

// updateValidator updates the JSON schema validator for the specified collection.
func (c *MongoConnector) updateValidator(ctx context.Context, name string, validator bson.M) error {
	cmd := bson.D{
		{Key: "collMod", Value: name},
		{Key: "validator", Value: bson.M{"$jsonSchema": validator}},
	}
	return c.Database.RunCommand(ctx, cmd).Err()
}

// ensureIndexes creates the specified indexes on the collection if they do not already exist.
func (c *MongoConnector) ensureIndexes(ctx context.Context, name string, indexes []mongo.IndexModel) error {
	collection := c.Database.Collection(name)
	for _, idx := range indexes {
		_, err := collection.Indexes().CreateOne(ctx, idx)
		if err != nil && !mongo.IsDuplicateKeyError(err) {
			return err
		}
	}
	return nil
}

// isNamespaceExistsError checks if the error indicates that the collection already exists.
func isNamespaceExistsError(err error) bool {
	// Mongo returns code 48 for "NamespaceExists"
	if we, ok := err.(mongo.CommandError); ok && we.Code == 48 {
		return true
	}
	return false
}
