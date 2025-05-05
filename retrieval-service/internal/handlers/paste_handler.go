package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"net/http"
	"retrieval-service/internal/metrics"
	"retrieval-service/internal/service/paste"
	"retrieval-service/shared"
	"time"
)

type PasteHandler struct {
	service *paste.RetrieveService
	logger  *shared.Logger
}

func NewPasteHandler(service *paste.RetrieveService, logger *shared.Logger) *PasteHandler {
	return &PasteHandler{
		service: service,
		logger:  logger,
	}
}

func (h *PasteHandler) GetPasteContent(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	requestID := uuid.New().String()
	ctx := context.WithValue(r.Context(), "requestID", requestID)
	url := chi.URLParam(r, "url")
	logger := h.logger.With("requestID", requestID, "url", url)

	// Giai đoạn 1: Nhận yêu cầu
	logger.Infof("Received get paste content request")
	metrics.RetrievalRequestDuration.WithLabelValues("receive_request").Observe(time.Since(startTime).Seconds())

	if url == "" {
		logger.Errorf("URL parameter is required")
		h.writeError(w, http.StatusBadRequest, "URL parameter is required")
		return
	}

	// Giai đoạn 2-5: Thực thi service
	resp, err := h.service.GetPasteContent(ctx, url)
	if err != nil {
		switch {
		case errors.Is(err, shared.ErrPasteNotFound), errors.Is(err, shared.ErrPasteExpired):
			logger.Errorf("Paste not found or expired", "error", err.Error())
			h.writeError(w, http.StatusNotFound, err.Error())
		default:
			logger.Errorf("Internal error", "error", err.Error())
			h.writeError(w, http.StatusInternalServerError, "Internal server error")
		}
		return
	}

	// Giai đoạn 6: Trả về phản hồi
	phaseStart := time.Now()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Errorf("Failed to encode response", "error", err.Error())
	}
	logger.Infof("Sent response")
	metrics.RetrievalRequestDuration.WithLabelValues("send_response").Observe(time.Since(phaseStart).Seconds())

	totalDuration := time.Since(startTime).Seconds()
	logger.Infof("Request completed", "totalDurationSeconds", totalDuration)
}

func (h *PasteHandler) GetPastePolicy(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	requestID := uuid.New().String()
	ctx := context.WithValue(r.Context(), "requestID", requestID)
	url := chi.URLParam(r, "url")
	logger := h.logger.With("requestID", requestID, "url", url)

	// Giai đoạn 1: Nhận yêu cầu
	logger.Infof("Received get paste policy request")
	metrics.RetrievalRequestDuration.WithLabelValues("receive_request").Observe(time.Since(startTime).Seconds())

	if url == "" {
		logger.Errorf("URL parameter is required")
		h.writeError(w, http.StatusBadRequest, "URL parameter is required")
		return
	}

	// Giai đoạn 2-5: Thực thi service
	resp, err := h.service.GetPastePolicy(ctx, url)
	if err != nil {
		switch {
		case errors.Is(err, shared.ErrPasteNotFound), errors.Is(err, shared.ErrPasteExpired):
			logger.Errorf("Paste not found or expired", "error", err.Error())
			h.writeError(w, http.StatusNotFound, err.Error())
		default:
			logger.Errorf("Internal error", "error", err.Error())
			h.writeError(w, http.StatusInternalServerError, "Internal server error")
		}
		return
	}

	// Giai đoạn 6: Trả về phản hồi
	phaseStart := time.Now()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"policy": resp}); err != nil {
		logger.Errorf("Failed to encode response", "error", err.Error())
	}
	logger.Infof("Sent response")
	metrics.RetrievalRequestDuration.WithLabelValues("send_response").Observe(time.Since(phaseStart).Seconds())

	totalDuration := time.Since(startTime).Seconds()
	logger.Infof("Request completed", "totalDurationSeconds", totalDuration)
}

func (h *PasteHandler) writeError(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": message}); err != nil {
		h.logger.Errorf("Failed to encode error response", "error", err.Error())
	}
}
