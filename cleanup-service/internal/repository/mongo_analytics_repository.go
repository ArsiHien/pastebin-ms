package repository

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AnalyticsRepository interface {
	Delete(ctx context.Context, pasteURL string) error
}

type MongoAnalyticsRepository struct {
	viewsCollection *mongo.Collection
	statsCollection *mongo.Collection
}

func NewMongoAnalyticsRepository(client *mongo.Client, dbName string) *MongoAnalyticsRepository {
	return &MongoAnalyticsRepository{
		viewsCollection: client.Database(dbName).Collection("paste_views"),
		statsCollection: client.Database(dbName).Collection("paste_stats"),
	}
}

func (r *MongoAnalyticsRepository) Delete(ctx context.Context, pasteURL string) error {
	// Delete from paste_views
	_, err := r.viewsCollection.DeleteMany(ctx, bson.M{"paste_url": pasteURL})
	if err != nil {
		return fmt.Errorf("failed to delete views for %s: %w", pasteURL, err)
	}

	// Delete from paste_stats
	result, err := r.statsCollection.DeleteOne(ctx, bson.M{"paste_url": pasteURL})
	if err != nil {
		return fmt.Errorf("failed to delete stats for %s: %w", pasteURL, err)
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("no stats found for %s", pasteURL)
	}

	return nil
}
