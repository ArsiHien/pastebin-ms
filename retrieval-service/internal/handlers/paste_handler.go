package handlers

import (
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"net/http"
	pasteService "retrieval-service/internal/service/paste"
	"retrieval-service/shared"
)

type PasteHandler struct {
	service *pasteService.RetrieveService
	logger  *shared.Logger
}

func NewPasteHandler(service *pasteService.RetrieveService,
	logger *shared.Logger) *PasteHandler {
	return &PasteHandler{
		service: service,
		logger:  logger,
	}
}

func (h *PasteHandler) GetPasteContent(w http.ResponseWriter, r *http.Request) {
	url := chi.URLParam(r, "url")
	if url == "" {
		h.writeError(w, http.StatusBadRequest, "URL parameter is required")
		return
	}

	resp, err := h.service.GetPasteContent(url)
	if err != nil {
		switch {
		case errors.Is(err, shared.ErrPasteNotFound), errors.Is(err, shared.ErrPasteExpired):
			h.writeError(w, http.StatusNotFound, err.Error())
		default:
			h.logger.Errorf("Internal error for URL %s: %v", url, err)
			h.writeError(w, http.StatusInternalServerError, "Internal server error")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Errorf("Failed to encode response for URL %s: %v", url, err)
	}
}

func (h *PasteHandler) GetPastePolicy(w http.ResponseWriter, r *http.Request) {
	url := chi.URLParam(r, "url")
	if url == "" {
		h.writeError(w, http.StatusBadRequest, "URL parameter is required")
		return
	}

	resp, err := h.service.GetPastePolicy(url)
	if err != nil {
		switch {
		case errors.Is(err, shared.ErrPasteNotFound), errors.Is(err, shared.ErrPasteExpired):
			h.writeError(w, http.StatusNotFound, err.Error())
		default:
			h.logger.Errorf("Internal error for URL %s: %v", url, err)
			h.writeError(w, http.StatusInternalServerError, "Internal server error")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Errorf("Failed to encode response for URL %s: %v", url, err)
	}
}

func (h *PasteHandler) writeError(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(map[string]string{"message": message})
	if err != nil {
		return
	}
}
