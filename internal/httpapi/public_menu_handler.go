package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"calbot/internal/domain"
	"calbot/internal/menu"
)

type PublicMenuHandler struct {
	service *menu.Service
}

func NewPublicMenuHandler(service *menu.Service) *PublicMenuHandler {
	return &PublicMenuHandler{service: service}
}

func (h *PublicMenuHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /public/menu-options", h.listRootOptions)
	mux.HandleFunc("GET /public/menu-options/{id}", h.getOptionDetail)
	mux.HandleFunc("GET /public/menu-options/{id}/responses", h.getResponses)
}

func (h *PublicMenuHandler) getResponses(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "menu option id must be a positive integer"})
		return
	}
	groups, err := h.service.GetActiveResponses(r.Context(), id)
	if errors.Is(err, menu.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "active menu option not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to retrieve menu responses"})
		return
	}

	response := make([]responseGroupResponse, 0, len(groups))
	for _, group := range groups {
		items := make([]responseItemResponse, 0, len(group.ResponseItems))
		for _, item := range group.ResponseItems {
			items = append(items, responseItemResponse{ID: item.ID, Type: item.Type, Text: item.Text, URL: item.URL, Caption: item.Caption, Metadata: item.Metadata, SortOrder: item.SortOrder})
		}
		response = append(response, responseGroupResponse{ID: group.ResponseGroup.ID, Name: group.ResponseGroup.Name, Description: group.ResponseGroup.Description, SortOrder: group.ResponseGroup.SortOrder, Items: items})
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *PublicMenuHandler) listRootOptions(w http.ResponseWriter, r *http.Request) {
	options, err := h.service.ListRootOptions(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to retrieve menu options"})
		return
	}
	response := make([]menuOptionResponse, 0, len(options))
	for _, option := range options {
		response = append(response, newMenuOptionResponse(option))
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *PublicMenuHandler) getOptionDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "menu option id must be a positive integer"})
		return
	}
	detail, err := h.service.GetOptionDetail(r.Context(), id)
	if errors.Is(err, menu.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "active menu option not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to retrieve menu option"})
		return
	}

	children := make([]menuOptionResponse, 0, len(detail.Children))
	for _, child := range detail.Children {
		children = append(children, newMenuOptionResponse(child))
	}
	response := struct {
		menuOptionResponse
		Prompt  *promptResponse      `json:"prompt"`
		Options []menuOptionResponse `json:"options"`
	}{menuOptionResponse: newMenuOptionResponse(detail.MenuOption), Options: children}
	if detail.Prompt != nil {
		response.Prompt = &promptResponse{ID: detail.Prompt.ID, Message: detail.Prompt.Message}
	}
	writeJSON(w, http.StatusOK, response)
}

type menuOptionResponse struct {
	ID          int64   `json:"id"`
	ParentID    *int64  `json:"parent_option_id"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
	SortOrder   int     `json:"sort_order"`
}

func newMenuOptionResponse(option domain.MenuOption) menuOptionResponse {
	return menuOptionResponse{ID: option.ID, ParentID: option.ParentOptionID, Title: option.Title, Description: option.Description, SortOrder: option.SortOrder}
}

type promptResponse struct {
	ID      int64  `json:"id"`
	Message string `json:"message"`
}

type responseGroupResponse struct {
	ID          int64                  `json:"id"`
	Name        string                 `json:"name"`
	Description *string                `json:"description"`
	SortOrder   int                    `json:"sort_order"`
	Items       []responseItemResponse `json:"items"`
}

type responseItemResponse struct {
	ID        int64                   `json:"id"`
	Type      domain.ResponseItemType `json:"type"`
	Text      *string                 `json:"text"`
	URL       *string                 `json:"url"`
	Caption   *string                 `json:"caption"`
	Metadata  json.RawMessage         `json:"metadata"`
	SortOrder int                     `json:"sort_order"`
}
