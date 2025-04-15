package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cleanup-service/config"
	"cleanup-service/internal/eventbus"
	"cleanup-service/internal/handlers"
	"cleanup-service/internal/repository"
	"cleanup-service/internal/scheduler"
	"cleanup-service/internal/service/cleanup"
	"cleanup-service/shared"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/go-sql-driver/mysql"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger := shared.NewLogger()

	// Connect to MySQL
	mysqlDB, err := sql.Open("mysql", cfg.MySQLDSN)
	if err != nil {
		logger.Fatalf("Failed to connect to MySQL: %v", err)
	}
	defer mysqlDB.Close()

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		logger.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	// Connect to RabbitMQ
	rabbitConn, err := shared.NewRabbitMQConn(cfg.RabbitMQURI)
	if err != nil {
		logger.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitConn.Close()

	// Initialize dependencies
	mysqlRepo := repository.NewMySQLPasteRepository(mysqlDB)
	retrievalRepo := repository.NewMongoRetrievalRepository(mongoClient, cfg.MongoDBName)
	analyticsRepo := repository.NewMongoAnalyticsRepository(mongoClient, cfg.MongoDBName)
	cleanupRepo := repository.NewMongoCleanupRepository(mongoClient, cfg.MongoDBName)
	publisher, err := eventbus.NewRabbitMQPublisher(rabbitConn)
	if err != nil {
		logger.Fatalf("Failed to create RabbitMQ publisher: %v", err)
	}
	defer publisher.Close()
	consumer, err := eventbus.NewRabbitMQConsumer(rabbitConn)
	if err != nil {
		logger.Fatalf("Failed to create RabbitMQ consumer: %v", err)
	}
	defer consumer.Close()

	cleanupService := cleanup.NewCleanupService(mysqlRepo, retrievalRepo, analyticsRepo, cleanupRepo, publisher, consumer, logger)
	scheduler := scheduler.NewCleanupScheduler(cleanupService, logger)
	handler := handlers.NewCleanupHandler(cleanupService, logger)

	// Start event consumer
	go func() {
		if err := cleanupService.StartEventConsumer(ctx); err != nil {
			logger.Fatalf("Event consumer failed: %v", err)
		}
	}()

	// Start scheduler
	go scheduler.Start(ctx)

	// Set up router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Post("/api/cleanup/run", handler.RunCleanup)
	r.Get("/api/cleanup/status", handler.GetStatus)

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
