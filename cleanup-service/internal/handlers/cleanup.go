package handlers

import (
	"cleanup-service/shared"
	"encoding/json"
	"net/http"

	"cleanup-service/internal/service/cleanup"
)

type CleanupHandler struct {
	service *cleanup.Service
	logger  *shared.Logger
}

func NewCleanupHandler(service *cleanup.Service, logger *shared.Logger) *CleanupHandler {
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
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		return
	}
}

func (h *CleanupHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status := h.service.GetStatus()
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(status)
	if err != nil {
		return
	}
}
