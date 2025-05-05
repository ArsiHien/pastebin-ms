package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/ArsiHien/pastebin-ms/create-service/internal/metrics"
	"github.com/ArsiHien/pastebin-ms/create-service/internal/service/paste"
	"github.com/ArsiHien/pastebin-ms/create-service/internal/shared"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type PasteHandler struct {
	UseCase *paste.CreatePasteUseCase
	Logger  *zap.Logger
}

func NewPasteHandler(useCase *paste.CreatePasteUseCase, logger *zap.Logger) *PasteHandler {
	return &PasteHandler{UseCase: useCase, Logger: logger}
}

func (h *PasteHandler) CreatePaste(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	requestID := uuid.New().String()
	ctx := context.WithValue(r.Context(), "requestID", requestID)
	logger := h.Logger.With(zap.String("requestID", requestID))

	// Giai đoạn 1: Nhận yêu cầu
	logger.Info("Received create paste request")
	metrics.CreateRequestDuration.WithLabelValues("receive_request").Observe(time.Since(startTime).Seconds())

	// Giai đoạn 2: Xử lý JSON
	phaseStart := time.Now()
	var req paste.CreatePasteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Failed to decode request body", zap.Error(err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	logger.Info("Decoded request body", zap.Any("request", req))
	metrics.CreateRequestDuration.WithLabelValues("decode_json").Observe(time.Since(phaseStart).Seconds())

	// Giai đoạn 3-6: Thực thi use case (sẽ đo chi tiết trong Execute)
	resp, err := h.UseCase.Execute(ctx, req)
	if err != nil {
		var httpErr shared.HTTPError
		if errors.As(err, &httpErr) {
			logger.Error("Use case error", zap.Error(err), zap.Int("code", httpErr.Code))
			w.WriteHeader(httpErr.Code)
			_ = json.NewEncoder(w).Encode(httpErr)
			return
		}
		logger.Error("Internal error", zap.Error(err))
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Giai đoạn 7: Trả về phản hồi
	phaseStart = time.Now()
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Error("Failed to encode response", zap.Error(err))
	}
	logger.Info("Sent response", zap.String("url", resp.URL))
	metrics.CreateRequestDuration.WithLabelValues("send_response").Observe(time.Since(phaseStart).Seconds())

	totalDuration := time.Since(startTime).Seconds()
	logger.Info("Request completed", zap.Float64("totalDurationSeconds", totalDuration))
}

func NewRouter(handler *PasteHandler) http.Handler {
	r := chi.NewRouter()
	r.Post("/api/pastes", handler.CreatePaste)
	r.Get("/metrics", promhttp.Handler().ServeHTTP)
	return r
}
