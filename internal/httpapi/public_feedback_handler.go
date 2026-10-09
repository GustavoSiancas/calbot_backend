package httpapi

import (
	"errors"
	"net/http"

	"calbot/internal/feedback"
)

type PublicFeedbackHandler struct {
	service *feedback.Service
}

func NewPublicFeedbackHandler(service *feedback.Service) *PublicFeedbackHandler {
	return &PublicFeedbackHandler{service: service}
}

func (h *PublicFeedbackHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /public/content-feedback", h.create)
}

func (h *PublicFeedbackHandler) create(w http.ResponseWriter, r *http.Request) {
	var request struct {
		FullName string `json:"full_name"`
		IP       string `json:"ip"`
		Message  string `json:"message"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	created, err := h.service.Create(r.Context(), feedback.CreateInput{
		FullName: request.FullName, IP: request.IP, Message: request.Message,
	})
	if errors.Is(err, feedback.ErrInvalidFeedback) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "full_name, a valid IP address, and a non-empty message are required"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unable to create content feedback"})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id": created.ID, "created_at": created.CreatedAt,
	})
}
