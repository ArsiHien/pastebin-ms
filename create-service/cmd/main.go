package main

import (
	"github.com/ArsiHien/pastebin-ms/create-service/config"
	"log"
	"net/http"

	"github.com/ArsiHien/pastebin-ms/create-service/internal/handlers"
	pasteService "github.com/ArsiHien/pastebin-ms/create-service/internal/service/paste"
)

func main() {
	cfg := config.LoadConfig()

	app, err := config.Initialize(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}
	defer config.Cleanup(app)

	// Create use case
	createPasteUseCase := pasteService.NewCreatePasteUseCase(
		app.PasteRepo,
		app.ExpirationPolicyRepo,
		app.Publisher,
	)

	// Handler and router
	handler := handlers.NewPasteHandler(createPasteUseCase)
	router := handlers.NewRouter(handler)

	// Start server
	log.Printf("Server is running on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, router))
}
