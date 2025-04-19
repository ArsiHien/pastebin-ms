package repository

import (
	"context"
	"fmt"

	"cleanup-service/internal/domain/paste"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type RetrievalRepository interface {
	FindAll(ctx context.Context) ([]paste.Paste, error)
	Delete(ctx context.Context, url string) error
}

type MongoRetrievalRepository struct {
	collection *mongo.Collection
}

func NewMongoRetrievalRepository(client *mongo.Client, dbName string) *MongoRetrievalRepository {
	return &MongoRetrievalRepository{
		collection: client.Database(dbName).Collection("pastes"),
	}
}

func (r *MongoRetrievalRepository) FindAll(ctx context.Context) ([]paste.Paste, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to find pastes: %w", err)
	}
	defer cursor.Close(ctx)

	var pastes []paste.Paste
	for cursor.Next(ctx) {
		var p paste.Paste
		if err := cursor.Decode(&p); err != nil {
			return nil, fmt.Errorf("failed to decode paste: %w", err)
		}
		pastes = append(pastes, p)
	}
	return pastes, cursor.Err()
}

func (r *MongoRetrievalRepository) Delete(ctx context.Context, url string) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"url": url})
	if err != nil {
		return fmt.Errorf("failed to delete paste %s: %w", url, err)
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("no paste found with url %s", url)
	}
	return nil
}
