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

	// Connect to RabbitMQ
	rabbitConn, err := shared.NewRabbitMQConn(cfg.RabbitMQURI)
	if err != nil {
		logger.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitConn.Close()

	// Initialize dependencies
	viewRepo := repository.NewMongoAnalyticsRepository(mongoClient, cfg.MongoDBName)
	consumer, err := eventbus.NewRabbitMQConsumer(rabbitConn, cfg.RabbitMQQueue)
	if err != nil {
		logger.Fatalf("Failed to create RabbitMQ consumer: %v", err)
	}
	analyticsService := analytics.NewAnalyticsService(viewRepo, consumer, logger)
	handler := handlers.NewAnalyticsHandler(analyticsService, logger)

	// Start event consumer
	go func() {
		if err := analyticsService.StartConsumer(ctx); err != nil {
			logger.Fatalf("Event consumer failed: %v", err)
		}
	}()

	// Set up router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Get("/api/analytics/hourly/{pasteUrl}", handler.GetHourlyAnalytics)
	r.Get("/api/analytics/weekly/{pasteUrl}", handler.GetWeeklyAnalytics)
	r.Get("/api/analytics/monthly/{pasteUrl}", handler.GetMonthlyAnalytics)
	r.Get("/api/pastes/stats", handler.GetPastesStats)

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
