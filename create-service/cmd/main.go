package main

import (
	"github.com/ArsiHien/pastebin-ms/create-service/config"
	"github.com/ArsiHien/pastebin-ms/create-service/internal/handlers"
	pasteService "github.com/ArsiHien/pastebin-ms/create-service/internal/service/paste"
	"go.uber.org/zap"
	"log"
	"net/http"
)

func main() {
	// Khởi tạo logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Tải cấu hình
	cfg := config.LoadConfig()

	// Khởi tạo ứng dụng
	app, err := config.Initialize(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize application", zap.Error(err))
	}
	defer config.Cleanup(app)

	// Create use case
	createPasteUseCase := pasteService.NewCreatePasteUseCase(
		app.PasteRepo,
		app.ExpirationPolicyRepo,
		app.Publisher,
	)

	// Handler và router
	handler := handlers.NewPasteHandler(createPasteUseCase, logger)
	router := handlers.NewRouter(handler)

	// Khởi động server
	logger.Info("Server is running", zap.String("port", cfg.Port))
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		logger.Fatal("Server failed", zap.Error(err))
	}
}
