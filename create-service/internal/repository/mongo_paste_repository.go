package repository

import (
	"context"
	"time"

	"github.com/ArsiHien/pastebin-ms/create-service/internal/domain/paste"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoPasteRepository struct {
	collection *mongo.Collection
}

func NewMongoPasteRepository(collection *mongo.Collection) *MongoPasteRepository {
	return &MongoPasteRepository{
		collection: collection,
	}
}

func (r *MongoPasteRepository) Save(p *paste.Paste) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.InsertOne(ctx, p)
	return err
}
