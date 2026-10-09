package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"calbot/internal/domain"
	"calbot/internal/menu"
)

// StaffResponseGroupHandler creates and manages response-group variants for STAFF accounts.
type StaffResponseGroupHandler struct {
	service *menu.Service
}

func NewStaffResponseGroupHandler(service *menu.Service) *StaffResponseGroupHandler {
	return &StaffResponseGroupHandler{service: service}
}

func (h *StaffResponseGroupHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/staff/response-groups/by-menu-option/{menuOptionID}", h.listByMenuOption)
	mux.HandleFunc("POST /api/v1/staff/response-groups", h.create)
	mux.HandleFunc("DELETE /api/v1/staff/response-groups/{id}", h.delete)
}

func (h *StaffResponseGroupHandler) listByMenuOption(w http.ResponseWriter, r *http.Request) {
	menuOptionID, err := parsePositivePathID(r, "menuOptionID")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "menu option id must be a positive integer"})
		return
	}
	groups, err := h.service.ListResponseGroupsForStaff(r.Context(), bearerToken(r), menuOptionID)
	if !handleMenuAuthorizationError(w, err) {
		return
	}
	if errors.Is(err, menu.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "menu option not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to retrieve response groups"})
		return
	}

	response := make([]staffResponseGroupResponse, 0, len(groups))
	for _, group := range groups {
		items := make([]staffResponseItemResponse, 0, len(group.ResponseItems))
		for _, item := range group.ResponseItems {
			items = append(items, staffResponseItemResponse{ID: item.ID, Type: item.Type, Text: item.Text, URL: item.URL, Caption: item.Caption, Metadata: item.Metadata, SortOrder: item.SortOrder, IsActive: item.IsActive, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt})
		}
		response = append(response, staffResponseGroupResponse{ID: group.ResponseGroup.ID, MenuOptionID: group.ResponseGroup.MenuOptionID, Name: group.ResponseGroup.Name, Description: group.ResponseGroup.Description, Weight: group.ResponseGroup.Weight, IsActive: group.ResponseGroup.IsActive, CreatedAt: group.ResponseGroup.CreatedAt, UpdatedAt: group.ResponseGroup.UpdatedAt, Items: items})
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *StaffResponseGroupHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parsePositivePathID(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "response group id must be a positive integer"})
		return
	}
	err = h.service.DeleteResponseGroup(r.Context(), bearerToken(r), id)
	if !handleMenuAuthorizationError(w, err) {
		return
	}
	if errors.Is(err, menu.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "response group not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to delete response group"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *StaffResponseGroupHandler) create(w http.ResponseWriter, r *http.Request) {
	var request struct {
		MenuOptionID int64   `json:"menu_option_id"`
		Name         string  `json:"name"`
		Description  *string `json:"description"`
		Weight       *int    `json:"weight"`
		Items        []struct {
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
	weight := 1
	if request.Weight != nil {
		weight = *request.Weight
	}
	items := make([]menu.ResponseItemInput, 0, len(request.Items))
	for _, item := range request.Items {
		items = append(items, menu.ResponseItemInput{Type: item.Type, Text: item.Text, URL: item.URL, Caption: item.Caption, Metadata: item.Metadata, SortOrder: item.SortOrder})
	}
	group, createdItems, err := h.service.CreateResponseGroup(r.Context(), bearerToken(r), request.MenuOptionID, menu.ResponseGroupInput{
		Name: request.Name, Description: request.Description, Weight: weight, Items: items,
	})
	if !handleMenuAuthorizationError(w, err) {
		return
	}
	if errors.Is(err, menu.ErrInvalidInput) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid response group or response item data"})
		return
	}
	if errors.Is(err, menu.ErrMenuOptionHasChildren) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "cannot add response items because this menu option already has child options"})
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
	writeJSON(w, http.StatusCreated, responseGroupResponse{ID: group.ID, Name: group.Name, Description: group.Description, Items: responseItems})
}
