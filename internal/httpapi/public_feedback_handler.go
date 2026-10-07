package httpapi

import (
	"errors"
	"net/http"

	"calbot/internal/domain"
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
		MenuOptionID     *int64                         `json:"menu_option_id"`
		ResponseGroupID  *int64                         `json:"response_group_id"`
		FeedbackType     domain.ContentFeedbackType     `json:"feedback_type"`
		Category         domain.ContentFeedbackCategory `json:"category"`
		Subject          *string                        `json:"subject"`
		Message          string                         `json:"message"`
		SuggestedContent *string                        `json:"suggested_content"`
		SourceURL        *string                        `json:"source_url"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	created, err := h.service.Create(r.Context(), feedback.CreateInput{
		MenuOptionID: request.MenuOptionID, ResponseGroupID: request.ResponseGroupID,
		FeedbackType: request.FeedbackType, Category: request.Category, Subject: request.Subject,
		Message: request.Message, SuggestedContent: request.SuggestedContent, SourceURL: request.SourceURL,
	})
	if errors.Is(err, feedback.ErrInvalidFeedback) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "feedback_type, category, and a non-empty message are required"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unable to create content feedback"})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id": created.ID, "status": created.Status, "created_at": created.CreatedAt,
	})
}
