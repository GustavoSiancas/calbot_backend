package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"calbot/internal/menu"
)

// StaffPromptHandler manages prompt variants for STAFF accounts.
type StaffPromptHandler struct {
	service *menu.Service
}

func NewStaffPromptHandler(service *menu.Service) *StaffPromptHandler {
	return &StaffPromptHandler{service: service}
}

func (h *StaffPromptHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/staff/prompts", h.list)
	mux.HandleFunc("POST /api/v1/staff/prompts", h.create)
	mux.HandleFunc("DELETE /api/v1/staff/prompts/{id}", h.delete)
	mux.HandleFunc("PATCH /api/v1/staff/prompts/{id}/active", h.setActive)
}

func (h *StaffPromptHandler) list(w http.ResponseWriter, r *http.Request) {
	menuOptionID, err := strconv.ParseInt(r.URL.Query().Get("menu_option_id"), 10, 64)
	if err != nil || menuOptionID < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "menu option id must be a positive integer"})
		return
	}
	prompts, err := h.service.ListPromptsForStaff(r.Context(), bearerToken(r), menuOptionID)
	if !handleMenuAuthorizationError(w, err) {
		return
	}
	if errors.Is(err, menu.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "menu option not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to retrieve menu option prompts"})
		return
	}
	response := make([]staffPromptResponse, 0, len(prompts))
	for _, prompt := range prompts {
		response = append(response, staffPromptResponse{ID: prompt.ID, Message: prompt.Message, SortOrder: prompt.SortOrder, Weight: prompt.Weight, IsActive: prompt.IsActive, CreatedAt: prompt.CreatedAt, UpdatedAt: prompt.UpdatedAt})
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *StaffPromptHandler) create(w http.ResponseWriter, r *http.Request) {
	var request struct {
		MenuOptionID int64  `json:"menu_option_id"`
		Message      string `json:"message"`
		SortOrder    int    `json:"sort_order"`
		Weight       *int   `json:"weight"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	weight := 1
	if request.Weight != nil {
		weight = *request.Weight
	}
	prompt, err := h.service.CreatePrompt(r.Context(), bearerToken(r), request.MenuOptionID, menu.PromptInput{Message: request.Message, SortOrder: request.SortOrder, Weight: weight})
	if !handleMenuAuthorizationError(w, err) {
		return
	}
	if errors.Is(err, menu.ErrInvalidInput) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message is required and weight must be greater than zero"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unable to create menu option prompt"})
		return
	}
	writeJSON(w, http.StatusCreated, promptResponse{ID: prompt.ID, Message: prompt.Message})
}

func (h *StaffPromptHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parsePositivePathID(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "prompt id must be a positive integer"})
		return
	}
	err = h.service.DeletePrompt(r.Context(), bearerToken(r), id)
	if !handleMenuAuthorizationError(w, err) {
		return
	}
	if errors.Is(err, menu.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "menu option prompt not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to delete menu option prompt"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *StaffPromptHandler) setActive(w http.ResponseWriter, r *http.Request) {
	id, err := parsePositivePathID(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "prompt id must be a positive integer"})
		return
	}
	var request struct {
		IsActive *bool `json:"is_active"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if request.IsActive == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "is_active is required"})
		return
	}
	prompt, err := h.service.SetPromptActive(r.Context(), bearerToken(r), id, *request.IsActive)
	if !handleMenuAuthorizationError(w, err) {
		return
	}
	if errors.Is(err, menu.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "menu option prompt not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to update menu option prompt"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": prompt.ID, "is_active": prompt.IsActive})
}

func parsePositivePathID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}
