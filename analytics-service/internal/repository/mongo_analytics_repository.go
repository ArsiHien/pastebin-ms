package repository

import (
	"context"
	"errors"
	"log"
	"time"

	"analytics-service/internal/domain/analytics"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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

func (r *MongoAnalyticsRepository) SaveView(ctx context.Context, view *analytics.View) error {
	_, err := r.viewsCollection.InsertOne(ctx, view)
	return err
}

func (r *MongoAnalyticsRepository) IncrementViewCount(ctx context.Context, pasteURL string) error {
	filter := bson.M{"paste_url": pasteURL}
	update := bson.M{
		"$inc":         bson.M{"view_count": 1},
		"$setOnInsert": bson.M{"paste_url": pasteURL},
	}
	opts := options.Update().SetUpsert(true)
	_, err := r.statsCollection.UpdateOne(ctx, filter, update, opts)
	return err
}

func (r *MongoAnalyticsRepository) GetViewCount(ctx context.Context, pasteURL string) (int, error) {
	filter := bson.M{"paste_url": pasteURL}
	var stats analytics.Stats

	err := r.statsCollection.FindOne(ctx, filter).Decode(&stats)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return stats.ViewCount, nil
}

func (r *MongoAnalyticsRepository) GetAnalytics(ctx context.Context, pasteURL string, period string) ([]analytics.View, error) {
	// Calculate time range based on period
	now := time.Now()
	var startTime time.Time

	switch period {
	case "hourly":
		startTime = now.Add(-1 * time.Hour)
	case "weekly":
		startTime = now.AddDate(0, 0, -7) // 7 days ago
	case "monthly":
		startTime = now.AddDate(-1, 0, 0) // 12 months ago
	default:
		return nil, analytics.ErrInvalidPeriod
	}

	log.Println("Start time:", startTime)

	// Find all views in the time range
	filter := bson.M{
		"paste_url": pasteURL,
		"viewed_at": bson.M{
			"$gte": startTime,
			"$lte": now,
		},
	}

	cursor, err := r.viewsCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			return
		}
	}(cursor, ctx)

	// Get all views
	var views []analytics.View
	if err = cursor.All(ctx, &views); err != nil {
		return nil, err
	}

	return views, nil
}

func (r *MongoAnalyticsRepository) GetPastesStats(ctx context.Context) (map[string]int, error) {
	cursor, err := r.statsCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			return
		}
	}(cursor, ctx)

	stats := make(map[string]int)
	for cursor.Next(ctx) {
		var stat analytics.Stats
		if err := cursor.Decode(&stat); err != nil {
			return nil, err
		}
		stats[stat.PasteURL] = stat.ViewCount
	}

	return stats, cursor.Err()
}
