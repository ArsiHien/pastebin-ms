package repository

import (
	"context"
	"time"

	"analytics-service/internal/domain/analytics"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AnalyticsRepository interface {
	SaveView(ctx context.Context, view *analytics.View) error
	IncrementViewCount(ctx context.Context, pasteURL string) error
	GetAnalytics(ctx context.Context, pasteURL string, period string) ([]analytics.TimeSeriesPoint, int, error)
	GetPastesStats(ctx context.Context) (map[string]int, error)
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

func (r *MongoAnalyticsRepository) GetAnalytics(ctx context.Context, pasteURL string, period string) ([]analytics.TimeSeriesPoint, int, error) {
	var granularity string
	var truncateFormat string
	var timeRange time.Duration
	switch period {
	case analytics.Hourly:
		granularity = "hour"
		truncateFormat = "2006-01-02T15:00:00Z"
		timeRange = 7 * 24 * time.Hour // Last 7 days for hourly
	case analytics.Weekly:
		granularity = "day"
		truncateFormat = "2006-01-02T00:00:00Z"
		timeRange = 30 * 24 * time.Hour // Last 30 days for weekly
	case analytics.Monthly:
		granularity = "day"
		truncateFormat = "2006-01-02T00:00:00Z"
		timeRange = 90 * 24 * time.Hour // Last 90 days for monthly
	default:
		return nil, 0, analytics.ErrInvalidPeriod
	}

	pipeline := mongo.Pipeline{
		{
			{"$match", bson.M{
				"paste_url": pasteURL,
				"viewed_at": bson.M{
					"$gte": time.Now().Add(-timeRange),
				},
			}},
		},
		{
			{"$group", bson.M{
				"_id": bson.M{
					"timestamp": bson.M{
						"$dateTrunc": bson.M{
							"date": "$viewed_at",
							"unit": granularity,
						},
					},
				},
				"viewCount": bson.M{"$sum": 1},
			}},
		},
		{
			{"$sort", bson.M{
				"_id.timestamp": 1,
			}},
		},
	}

	cursor, err := r.viewsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var points []analytics.TimeSeriesPoint
	totalViews := 0
	for cursor.Next(ctx) {
		var result struct {
			ID        struct{ Timestamp time.Time } `bson:"_id"`
			ViewCount int                           `bson:"viewCount"`
		}
		if err := cursor.Decode(&result); err != nil {
			return nil, 0, err
		}
		points = append(points, analytics.TimeSeriesPoint{
			Timestamp: result.ID.Timestamp,
			ViewCount: result.ViewCount,
		})
		totalViews += result.ViewCount
	}

	return points, totalViews, cursor.Err()
}

func (r *MongoAnalyticsRepository) GetPastesStats(ctx context.Context) (map[string]int, error) {
	cursor, err := r.statsCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

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
