package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type CleanupRepository interface {
	AddTask(ctx context.Context, url string, expireAt time.Time, isBurnAfterRead bool) error
	MarkRead(ctx context.Context, url string) error
	FindExpired(ctx context.Context, now time.Time) ([]string, error)
	DeleteTask(ctx context.Context, url string) error
}

type MongoCleanupRepository struct {
	collection *mongo.Collection
}

func NewMongoCleanupRepository(client *mongo.Client, dbName string) *MongoCleanupRepository {
	return &MongoCleanupRepository{
		collection: client.Database(dbName).Collection("cleanup_tasks"),
	}
}

func (r *MongoCleanupRepository) AddTask(ctx context.Context, url string, expireAt time.Time, isBurnAfterRead bool) error {
	task := bson.M{
		"url":                url,
		"expire_at":          expireAt,
		"is_burn_after_read": isBurnAfterRead,
		"is_read":            false,
	}
	_, err := r.collection.InsertOne(ctx, task)
	return err
}

func (r *MongoCleanupRepository) MarkRead(ctx context.Context, url string) error {
	filter := bson.M{"url": url}
	update := bson.M{
		"$set": bson.M{
			"is_read":   true,
			"expire_at": time.Now(),
		},
	}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *MongoCleanupRepository) FindExpired(ctx context.Context, now time.Time) ([]string, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"expire_at": bson.M{"$lte": now}},
			{"is_burn_after_read": true, "is_read": true},
		},
	}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var urls []string
	for cursor.Next(ctx) {
		var task struct {
			URL string `bson:"url"`
		}
		if err := cursor.Decode(&task); err != nil {
			return nil, err
		}
		urls = append(urls, task.URL)
	}
	return urls, cursor.Err()
}

func (r *MongoCleanupRepository) DeleteTask(ctx context.Context, url string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"url": url})
	return err
}
