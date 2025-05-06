package main

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"net/http"
	"os"
	"os/signal"
	"retrieval-service/shared"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"retrieval-service/config"
	"retrieval-service/internal/cache"
	"retrieval-service/internal/eventbus"
	"retrieval-service/internal/handlers"
	"retrieval-service/internal/repository"
	"retrieval-service/internal/service/paste"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger := shared.NewLogger()
	defer logger.Sync()

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	log.Println(cfg.MongoURI)
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		logger.Fatalf("Failed to connect to MongoDB", "error", err)
	}
	defer func(mongoClient *mongo.Client, ctx context.Context) {
		if err := mongoClient.Disconnect(ctx); err != nil {
			logger.Errorf("Failed to disconnect from MongoDB", "error", err)
		} else {
			logger.Infof("Disconnected from MongoDB")
		}
	}(mongoClient, ctx)

	// Connect to Redis
	redisClient, err := cache.NewRedisClient(cfg.RedisURI)
	if err != nil {
		logger.Fatalf("Failed to connect to Redis", "error", err)
	}
	defer func(redisClient *redis.Client) {
		if err := redisClient.Close(); err != nil {
			logger.Errorf("Failed to close Redis connection", "error", err)
		} else {
			logger.Infof("Disconnected from Redis")
		}
	}(redisClient)

	// Connect to RabbitMQ
	rabbitConn, err := eventbus.NewRabbitMQConn(cfg.RabbitMQURI)
	if err != nil {
		logger.Fatalf("Failed to connect to RabbitMQ", "error", err)
	}
	defer func(rabbitConn *amqp.Connection) {
		if err := rabbitConn.Close(); err != nil {
			logger.Errorf("Failed to close RabbitMQ connection", "error", err)
		} else {
			logger.Infof("Disconnected from RabbitMQ")
		}
	}(rabbitConn)
	// Initialize dependencies
	pasteRepo := repository.NewMongoPasteRepository(mongoClient.Database(cfg.MongoDBName))
	pasteCache := cache.NewRedisPasteCache(redisClient)
	publisher, err := eventbus.NewRabbitMQPublisher(rabbitConn)
	if err != nil {
		logger.Fatalf("Failed to create RabbitMQ publisher", "error", err)
	}
	defer func(publisher *eventbus.RabbitMQPublisher) {
		if err := publisher.Close(); err != nil {
			logger.Errorf("Failed to close RabbitMQ publisher", "error", err)
		} else {
			logger.Infof("Disconnected from RabbitMQ")
		}
	}(publisher)

	pasteConsumer, err := eventbus.NewRabbitMQConsumer(rabbitConn,
		mongoClient.Database(cfg.MongoDBName), pasteCache, logger)
	if err != nil {
		logger.Fatalf("Failed to create RabbitMQ consumer", "error", err)
	}

	// Start consuming messages
	if err := pasteConsumer.Start(); err != nil {
		logger.Fatalf("Failed to start RabbitMQ consumer", "error", err)
	}
	defer func(pasteConsumer *eventbus.RabbitMQConsumer) {
		if err := pasteConsumer.Stop(); err != nil {
			logger.Errorf("Failed to stop RabbitMQ consumer", "error", err)
		}
	}(pasteConsumer)

	// Initialize service and handler
	retrieveService := paste.NewRetrieveService(pasteRepo, pasteCache, publisher, logger)
	handler := handlers.NewPasteHandler(retrieveService, logger)

	// Set up router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Get("/api/pastes/{url}/content", handler.GetPasteContent)
	r.Get("/api/pastes/{url}/policy", handler.GetPastePolicy)
	r.Handle("/metrics", promhttp.Handler())

	// Start server
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		logger.Infof("Starting server on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatalf("Server failed", "error", err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Infof("Shutting down server...")
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Errorf("Server shutdown failed", "error", err)
	}
	logger.Infof("Server stopped")
}
