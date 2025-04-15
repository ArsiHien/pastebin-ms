package handlers

import (
	"encoding/json"
	"net/http"

	"cleanup-service/internal/service/cleanup"
	"cleanup-service/internal/shared"
	"github.com/go-chi/chi/v5"
)

type CleanupHandler struct {
	service *cleanup.CleanupService
	logger  *shared.Logger
}

func NewCleanupHandler(service *cleanup.CleanupService, logger *shared.Logger) *CleanupHandler {
	return &CleanupHandler{
		service: service,
		logger:  logger,
	}
}

func (h *CleanupHandler) RunCleanup(w http.ResponseWriter, r *http.Request) {
	count, err := h.service.RunCleanup(r.Context())
	if err != nil {
		h.logger.Errorf("Failed to run cleanup: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"pastes_deleted": count,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *CleanupHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status := h.service.GetStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
