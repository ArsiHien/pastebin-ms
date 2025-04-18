package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ArsiHien/pastebin-ms/create-service/internal/service/paste"
	"github.com/ArsiHien/pastebin-ms/create-service/internal/shared"
	"github.com/go-chi/chi/v5"
)

type PasteHandler struct {
	UseCase *paste.CreatePasteUseCase
}

func NewPasteHandler(useCase *paste.CreatePasteUseCase) *PasteHandler {
	return &PasteHandler{UseCase: useCase}
}

func (h *PasteHandler) CreatePaste(w http.ResponseWriter, r *http.Request) {
	var req paste.CreatePasteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.UseCase.Execute(req)
	if err != nil {
		var httpErr shared.HTTPError
		if errors.As(err, &httpErr) {
			w.WriteHeader(httpErr.Code)
			_ = json.NewEncoder(w).Encode(httpErr)
			return
		}
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func NewRouter(handler *PasteHandler) http.Handler {
	r := chi.NewRouter()
	r.Post("/api/pastes", handler.CreatePaste)
	return r
}
