package handlers

import (
	"analytics-service/shared"
	"encoding/json"
	"net/http"

	"analytics-service/internal/service/analytics"
	"github.com/go-chi/chi/v5"
)

type AnalyticsHandler struct {
	service *analytics.Service
	logger  *shared.Logger
}

func NewAnalyticsHandler(service *analytics.Service,
	logger *shared.Logger) *AnalyticsHandler {
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

func (h *AnalyticsHandler) GetPasteStats(w http.ResponseWriter, r *http.Request) {
	pasteURL := chi.URLParam(r, "url")
	if pasteURL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}

	count, err := h.service.GetPasteStats(r.Context(), pasteURL)
	if err != nil {
		h.logger.Errorf("Failed to get stats for paste %s: %v", pasteURL, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]int{"viewCount": count})
	if err != nil {
		return
	}
}
