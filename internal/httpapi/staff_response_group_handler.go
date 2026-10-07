package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"calbot/internal/domain"
	"calbot/internal/menu"
)

// StaffResponseGroupHandler replaces response-group content for STAFF accounts.
type StaffResponseGroupHandler struct {
	service *menu.Service
}

func NewStaffResponseGroupHandler(service *menu.Service) *StaffResponseGroupHandler {
	return &StaffResponseGroupHandler{service: service}
}

func (h *StaffResponseGroupHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/staff/response-groups", h.replace)
}

func (h *StaffResponseGroupHandler) replace(w http.ResponseWriter, r *http.Request) {
	var request struct {
		MenuOptionID    int64   `json:"menu_option_id"`
		ResponseGroupID *int64  `json:"response_group_id"`
		Name            string  `json:"name"`
		Description     *string `json:"description"`
		SortOrder       int     `json:"sort_order"`
		Items           []struct {
			Type      domain.ResponseItemType `json:"type"`
			Text      *string                 `json:"text"`
			URL       *string                 `json:"url"`
			Caption   *string                 `json:"caption"`
			Metadata  json.RawMessage         `json:"metadata"`
			SortOrder int                     `json:"sort_order"`
		} `json:"items"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	items := make([]menu.ResponseItemInput, 0, len(request.Items))
	for _, item := range request.Items {
		items = append(items, menu.ResponseItemInput{Type: item.Type, Text: item.Text, URL: item.URL, Caption: item.Caption, Metadata: item.Metadata, SortOrder: item.SortOrder})
	}
	group, createdItems, err := h.service.ReplaceResponseGroup(r.Context(), bearerToken(r), request.MenuOptionID, menu.ResponseGroupInput{
		ResponseGroupID: request.ResponseGroupID, Name: request.Name, Description: request.Description, SortOrder: request.SortOrder, Items: items,
	})
	if !handleMenuAuthorizationError(w, err) {
		return
	}
	if errors.Is(err, menu.ErrInvalidInput) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid response group or response item data"})
		return
	}
	if errors.Is(err, menu.ErrResponseGroupDoesNotBelongToOption) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "response group belongs to another menu option"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unable to save response group"})
		return
	}

	responseItems := make([]responseItemResponse, 0, len(createdItems))
	for _, item := range createdItems {
		responseItems = append(responseItems, responseItemResponse{ID: item.ID, Type: item.Type, Text: item.Text, URL: item.URL, Caption: item.Caption, Metadata: item.Metadata, SortOrder: item.SortOrder})
	}
	writeJSON(w, http.StatusCreated, responseGroupResponse{ID: group.ID, Name: group.Name, Description: group.Description, SortOrder: group.SortOrder, Items: responseItems})
}
