package mongo

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// HealthCheck verifies the cluster is reachable. Mirrors
// Foxel.Mongo.Healths.MongoHealthCheck.
//
// Wire it into your health endpoint:
//
//	if err := mongo.HealthCheck(ctx, client); err != nil { ... }
func HealthCheck(ctx context.Context, client *Client) error {
	if client == nil {
		return errors.New("mongo: nil client")
	}
	if err := client.Ping(ctx); err != nil {
		return err
	}
	db, err := client.Database(ctx)
	if err != nil {
		return err
	}
	_, err = db.ListCollectionNames(ctx, bson.M{})
	return err
}
