package main

import (
	"context"
	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ArsiHien/pastebin-ms/create-service/internal/eventbus"
	"github.com/ArsiHien/pastebin-ms/create-service/internal/handlers"
	"github.com/ArsiHien/pastebin-ms/create-service/internal/repository"
	"github.com/ArsiHien/pastebin-ms/create-service/internal/service/paste"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Load env
	mongoURI := os.Getenv("MONGO_URI")
	rabbitURI := os.Getenv("RABBITMQ_URI")
	port := os.Getenv("PORT")

	// Connect MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("MongoDB connection failed: %v", err)
	}
	defer func(mongoClient *mongo.Client, ctx context.Context) {
		err := mongoClient.Disconnect(ctx)
		if err != nil {
			log.Fatalf("MongoDB disconnection failed: %v", err)
		} else {
			log.Println("MongoDB disconnected")
		}
	}(mongoClient, ctx)

	collection := mongoClient.Database("pastebin").Collection("pastes")
	pasteRepo := repository.NewMongoPasteRepository(collection)

	// Connect RabbitMQ
	rabbitConn, err := amqp.Dial(rabbitURI)
	if err != nil {
		log.Fatalf("RabbitMQ connection failed: %v", err)
	}
	defer func(rabbitConn *amqp.Connection) {
		err := rabbitConn.Close()
		if err != nil {
			log.Fatalf("RabbitMQ disconnection failed: %v", err)
		} else {
			log.Println("RabbitMQ disconnected")
		}
	}(rabbitConn)

	publisher, err := eventbus.NewRabbitMQPublisher(rabbitConn)
	if err != nil {
		log.Fatalf("RabbitMQ publisher init failed: %v", err)
	}
	defer func(publisher *eventbus.RabbitMQPublisher) {
		err := publisher.Close()
		if err != nil {
			log.Fatalf("RabbitMQ publisher close failed: %v", err)
		} else {
			log.Println("RabbitMQ publisher closed")
		}
	}(publisher)

	// Create use case
	createPasteUseCase := paste.NewCreatePasteUseCase(pasteRepo, publisher)

	// Handler and router
	handler := handlers.NewPasteHandler(createPasteUseCase)
	router := handlers.NewRouter(handler)

	log.Printf("Server is running on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
