package handlers

import (
	"encoding/json"
	"net/http"

	"analytics-service/internal/domain/analytics"
	"analytics-service/internal/service/analytics"
	"analytics-service/internal/shared"
	"github.com/go-chi/chi/v5"
)

type AnalyticsHandler struct {
	service *analytics.AnalyticsService
	logger  *shared.Logger
}

func NewAnalyticsHandler(service *analytics.AnalyticsService, logger *shared.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{
		service: service,
		logger:  logger,
	}
}

func (h *AnalyticsHandler) GetHourlyAnalytics(w http.ResponseWriter, r *http.Request) {
	pasteURL := chi.URLParam(r, "pasteUrl")
	if pasteURL == "" {
		http.Error(w, "pasteUrl is required", http.StatusBadRequest)
		return
	}

	resp, err := h.service.GetAnalytics(r.Context(), pasteURL, analytics.Hourly)
	if err != nil {
		h.logger.Errorf("Failed to get hourly analytics for %s: %v", pasteURL, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AnalyticsHandler) GetWeeklyAnalytics(w http.ResponseWriter, r *http.Request) {
	pasteURL := chi.URLParam(r, "pasteUrl")
	if pasteURL == "" {
		http.Error(w, "pasteUrl is required", http.StatusBadRequest)
		return
	}

	resp, err := h.service.GetAnalytics(r.Context(), pasteURL, analytics.Weekly)
	if err != nil {
		h.logger.Errorf("Failed to get weekly analytics for %s: %v", pasteURL, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AnalyticsHandler) GetMonthlyAnalytics(w http.ResponseWriter, r *http.Request) {
	pasteURL := chi.URLParam(r, "pasteUrl")
	if pasteURL == "" {
		http.Error(w, "pasteUrl is required", http.StatusBadRequest)
		return
	}

	resp, err := h.service.GetAnalytics(r.Context(), pasteURL, analytics.Monthly)
	if err != nil {
		h.logger.Errorf("Failed to get monthly analytics for %s: %v", pasteURL, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AnalyticsHandler) GetPastesStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetPastesStats(r.Context())
	if err != nil {
		h.logger.Errorf("Failed to get pastes stats: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
