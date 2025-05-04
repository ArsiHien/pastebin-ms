package main

import (
	"analytics-service/config"
	"analytics-service/internal/eventbus"
	"analytics-service/internal/handlers"
	"analytics-service/internal/metrics"
	"analytics-service/internal/repository"
	"analytics-service/internal/service/analytics"
	"analytics-service/shared"
	"context"
	"errors"
	"github.com/rabbitmq/amqp091-go"
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
	defer func(mongoClient *mongo.Client, ctx context.Context) {
		err := mongoClient.Disconnect(ctx)
		if err != nil {
			logger.Fatalf("Failed to disconnect from MongoDB: %v", err)
		}
	}(mongoClient, ctx)

	// Connect to RabbitMQ
	rabbitConn, err := shared.NewRabbitMQConn(cfg.RabbitMQURI)
	if err != nil {
		logger.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer func(rabbitConn *amqp091.Connection) {
		err := rabbitConn.Close()
		if err != nil {
			logger.Fatalf("Failed to close RabbitMQ connection: %v", err)
		}
	}(rabbitConn)

	metrics.ExposeMetrics()

	// Initialize dependencies
	viewRepo := repository.NewMongoAnalyticsRepository(mongoClient, cfg.MongoDBName)
	consumer, err := eventbus.NewRabbitMQConsumer(rabbitConn, cfg.RabbitMQQueue)
	if err != nil {
		logger.Fatalf("Failed to create RabbitMQ consumer: %v", err)
	}
	analyticsService := analytics.NewAnalyticsService(viewRepo, consumer, logger)
	handler := handlers.NewAnalyticsHandler(analyticsService, logger)

	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	defer consumerCancel()
	// Start event consumer
	go func() {
		if err := analyticsService.StartConsumer(consumerCtx); err != nil {
			logger.Fatalf("Event consumer failed: %v", err)
		}
	}()

	// Set up router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	// In cmd/main.go, update the router setup
	r.Get("/api/analytics/hourly/{pasteUrl}", handler.GetHourlyAnalytics)
	r.Get("/api/analytics/weekly/{pasteUrl}", handler.GetWeeklyAnalytics)
	r.Get("/api/analytics/monthly/{pasteUrl}", handler.GetMonthlyAnalytics)
	r.Get("/api/pastes/{url}/stats", handler.GetPasteStats) // Add this line

	// Start server
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		logger.Infof("Starting server on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatalf("Server failed: %v", err)
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
		logger.Errorf("Server shutdown failed: %v", err)
	}
	logger.Infof("Server stopped")
}
