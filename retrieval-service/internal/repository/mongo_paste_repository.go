package repository

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/mongo"
	"retrieval-service/internal/domain/paste"
	"time"
)

type MongoPasteRepository struct {
	collection *mongo.Collection
}

func NewMongoPasteRepository(db *mongo.Database) *MongoPasteRepository {
	collection := db.Collection("pastes")
	return &MongoPasteRepository{collection: collection}
}

func (r *MongoPasteRepository) FindByURL(url string) (*paste.Paste, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var p paste.Paste
	err := r.collection.FindOne(ctx, map[string]interface{}{"url": url}).Decode(&p)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *MongoPasteRepository) MarkAsRead(url string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := r.collection.UpdateOne(ctx, map[string]interface{}{"url": url}, map[string]interface{}{
		"$set": map[string]interface{}{
			"expiration_policy.is_read": true,
		},
	})
	if err != nil {
		return err
	}
	return nil
}
