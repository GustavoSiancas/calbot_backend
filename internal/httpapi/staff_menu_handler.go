package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"calbot/internal/domain"
	"calbot/internal/menu"
)

// StaffMenuHandler exposes menu administration operations to STAFF accounts only.
type StaffMenuHandler struct {
	service *menu.Service
}

func NewStaffMenuHandler(service *menu.Service) *StaffMenuHandler {
	return &StaffMenuHandler{service: service}
}

func (h *StaffMenuHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/menu-options", h.listAll)
	mux.HandleFunc("POST /api/v1/menu-options", h.create)
	mux.HandleFunc("PATCH /api/v1/menu-options/{id}/deactivate", h.deactivate)
}

func (h *StaffMenuHandler) listAll(w http.ResponseWriter, r *http.Request) {
	options, err := h.service.ListAllForStaff(r.Context(), bearerToken(r))
	if !handleMenuAuthorizationError(w, err) {
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to retrieve menu options"})
		return
	}

	response := make([]staffMenuOptionResponse, 0, len(options))
	for _, option := range options {
		prompts := make([]staffPromptResponse, 0, len(option.Prompts))
		for _, prompt := range option.Prompts {
			prompts = append(prompts, staffPromptResponse{ID: prompt.ID, Message: prompt.Message, SortOrder: prompt.SortOrder, Weight: prompt.Weight, IsActive: prompt.IsActive, CreatedAt: prompt.CreatedAt, UpdatedAt: prompt.UpdatedAt})
		}
		groups := make([]staffResponseGroupResponse, 0, len(option.ResponseGroups))
		for _, group := range option.ResponseGroups {
			items := make([]staffResponseItemResponse, 0, len(group.ResponseItems))
			for _, item := range group.ResponseItems {
				items = append(items, staffResponseItemResponse{ID: item.ID, Type: item.Type, Text: item.Text, URL: item.URL, Caption: item.Caption, Metadata: item.Metadata, SortOrder: item.SortOrder, IsActive: item.IsActive, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt})
			}
			groups = append(groups, staffResponseGroupResponse{ID: group.ResponseGroup.ID, Name: group.ResponseGroup.Name, Description: group.ResponseGroup.Description, SortOrder: group.ResponseGroup.SortOrder, IsActive: group.ResponseGroup.IsActive, CreatedAt: group.ResponseGroup.CreatedAt, UpdatedAt: group.ResponseGroup.UpdatedAt, Items: items})
		}
		response = append(response, staffMenuOptionResponse{ID: option.MenuOption.ID, ParentOptionID: option.MenuOption.ParentOptionID, Title: option.MenuOption.Title, Description: option.MenuOption.Description, SortOrder: option.MenuOption.SortOrder, IsActive: option.MenuOption.IsActive, CreatedAt: option.MenuOption.CreatedAt, UpdatedAt: option.MenuOption.UpdatedAt, Prompts: prompts, ResponseGroups: groups})
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *StaffMenuHandler) create(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ParentOptionID *int64  `json:"parent_option_id"`
		Title          string  `json:"title"`
		Description    *string `json:"description"`
		SortOrder      int     `json:"sort_order"`
		Prompts        []struct {
			Message   string `json:"message"`
			SortOrder int    `json:"sort_order"`
			Weight    *int   `json:"weight"`
		} `json:"prompts"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	prompts := make([]menu.PromptInput, 0, len(request.Prompts))
	for _, prompt := range request.Prompts {
		weight := 1
		if prompt.Weight != nil {
			weight = *prompt.Weight
		}
		prompts = append(prompts, menu.PromptInput{Message: prompt.Message, SortOrder: prompt.SortOrder, Weight: weight})
	}

	option, createdPrompts, err := h.service.Create(r.Context(), bearerToken(r), menu.CreateInput{
		ParentOptionID: request.ParentOptionID, Title: request.Title, Description: request.Description,
		SortOrder: request.SortOrder, Prompts: prompts,
	})
	if !handleMenuAuthorizationError(w, err) {
		return
	}
	if errors.Is(err, menu.ErrInvalidInput) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title and each prompt message are required; prompt weight must be greater than zero"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unable to create menu option"})
		return
	}

	promptResponses := make([]promptResponse, 0, len(createdPrompts))
	for _, prompt := range createdPrompts {
		promptResponses = append(promptResponses, promptResponse{ID: prompt.ID, Message: prompt.Message})
	}
	writeJSON(w, http.StatusCreated, struct {
		menuOptionResponse
		Prompts []promptResponse `json:"prompts"`
	}{menuOptionResponse: newMenuOptionResponse(option), Prompts: promptResponses})
}

func (h *StaffMenuHandler) deactivate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "menu option id must be a positive integer"})
		return
	}
	err = h.service.Deactivate(r.Context(), bearerToken(r), id)
	if !handleMenuAuthorizationError(w, err) {
		return
	}
	if errors.Is(err, menu.ErrInvalidInput) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "menu option id must be a positive integer"})
		return
	}
	if errors.Is(err, menu.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "active menu option not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to deactivate menu option"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func bearerToken(r *http.Request) string {
	return strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
}

func handleMenuAuthorizationError(w http.ResponseWriter, err error) bool {
	if errors.Is(err, menu.ErrUnauthorized) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "a valid staff Bearer token is required"})
		return false
	}
	if errors.Is(err, menu.ErrForbidden) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "staff role is required"})
		return false
	}
	return true
}

type staffMenuOptionResponse struct {
	ID             int64                        `json:"id"`
	ParentOptionID *int64                       `json:"parent_option_id"`
	Title          string                       `json:"title"`
	Description    *string                      `json:"description"`
	SortOrder      int                          `json:"sort_order"`
	IsActive       bool                         `json:"is_active"`
	CreatedAt      time.Time                    `json:"created_at"`
	UpdatedAt      time.Time                    `json:"updated_at"`
	Prompts        []staffPromptResponse        `json:"prompts"`
	ResponseGroups []staffResponseGroupResponse `json:"response_groups"`
}

type staffPromptResponse struct {
	ID        int64     `json:"id"`
	Message   string    `json:"message"`
	SortOrder int       `json:"sort_order"`
	Weight    int       `json:"weight"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type staffResponseGroupResponse struct {
	ID          int64                       `json:"id"`
	Name        string                      `json:"name"`
	Description *string                     `json:"description"`
	SortOrder   int                         `json:"sort_order"`
	IsActive    bool                        `json:"is_active"`
	CreatedAt   time.Time                   `json:"created_at"`
	UpdatedAt   time.Time                   `json:"updated_at"`
	Items       []staffResponseItemResponse `json:"items"`
}

type staffResponseItemResponse struct {
	ID        int64                   `json:"id"`
	Type      domain.ResponseItemType `json:"type"`
	Text      *string                 `json:"text"`
	URL       *string                 `json:"url"`
	Caption   *string                 `json:"caption"`
	Metadata  json.RawMessage         `json:"metadata"`
	SortOrder int                     `json:"sort_order"`
	IsActive  bool                    `json:"is_active"`
	CreatedAt time.Time               `json:"created_at"`
	UpdatedAt time.Time               `json:"updated_at"`
}
