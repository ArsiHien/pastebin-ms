package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"retrieval-service/config"
	"retrieval-service/internal/cache" // Import mới
	"retrieval-service/internal/eventbus"
	"retrieval-service/internal/handlers"
	"retrieval-service/internal/repository"
	"retrieval-service/internal/service/paste"
	"retrieval-service/internal/shared"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger := shared.NewLogger()

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		logger.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	// Connect to Redis
	redisClient, err := cache.NewRedisClient(cfg.RedisURI) // Sử dụng cache.NewRedisClient
	if err != nil {
		logger.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Connect to RabbitMQ
	rabbitConn, err := shared.NewRabbitMQConn(cfg.RabbitMQURI)
	if err != nil {
		logger.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitConn.Close()

	// Initialize dependencies
	pasteRepo := repository.NewMongoPasteRepository(mongoClient, cfg.MongoDBName)
	pasteCache := cache.NewRedisPasteCache(redisClient) // Sử dụng cache.NewRedisPasteCache
	publisher, err := eventbus.NewRabbitMQPublisher(rabbitConn)
	if err != nil {
		logger.Fatalf("Failed to create RabbitMQ publisher: %v", err)
	}
	defer publisher.Close()

	// Initialize service and handler
	retrieveService := paste.NewRetrieveService(pasteRepo, pasteCache, publisher)
	handler := handlers.NewPasteHandler(retrieveService, logger)

	// Set up router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Get("/api/pastes/{url}", handler.GetPaste)

	// Start server
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		logger.Infof("Starting server on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Info("Shutting down server...")
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Errorf("Server shutdown failed: %v", err)
	}
	logger.Info("Server stopped")
}
