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
	mux.HandleFunc("GET /public/menu-options/{id}/sub-options", h.getSubOptions)
	mux.HandleFunc("GET /public/menu-options/{id}/responses", h.getResponses)
}

func (h *PublicMenuHandler) getResponses(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "menu option id must be a positive integer"})
		return
	}
	group, err := h.service.GetActiveResponse(r.Context(), id)
	if errors.Is(err, menu.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "active menu option not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to retrieve menu responses"})
		return
	}

	if group == nil {
		writeJSON(w, http.StatusOK, nil)
		return
	}
	items := make([]responseItemResponse, 0, len(group.ResponseItems))
	for _, item := range group.ResponseItems {
		items = append(items, responseItemResponse{ID: item.ID, Type: item.Type, Text: item.Text, URL: item.URL, Caption: item.Caption, Metadata: item.Metadata, SortOrder: item.SortOrder})
	}
	response := responseGroupResponse{ID: group.ResponseGroup.ID, Name: group.ResponseGroup.Name, Description: group.ResponseGroup.Description, Items: items}
	writeJSON(w, http.StatusOK, response)
}

func (h *PublicMenuHandler) listRootOptions(w http.ResponseWriter, r *http.Request) {
	options, err := h.service.ListRootOptions(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to retrieve menu options"})
		return
	}
	response := make([]publicParentMenuOptionResponse, 0, len(options))
	for _, option := range options {
		response = append(response, newPublicParentMenuOptionResponse(option))
	}
	writeJSON(w, http.StatusOK, response)
}

type publicParentMenuOptionResponse struct {
	ID        int64                            `json:"id"`
	Title     string                           `json:"title"`
	SortOrder int                              `json:"sort_order"`
	IsActive  bool                             `json:"is_active"`
	IsAviable publicParentAvailabilityResponse `json:"isAviable"`
}

type publicParentAvailabilityResponse struct {
	HasResponse bool `json:"has_response"`
	HasChildren bool `json:"has_children"`
}

type publicSubOptionsResponse struct {
	Prompt  *promptResponse                  `json:"prompt"`
	Options []publicParentMenuOptionResponse `json:"options"`
}

func newPublicParentMenuOptionResponse(option menu.ParentOptionDetail) publicParentMenuOptionResponse {
	return publicParentMenuOptionResponse{
		ID: option.MenuOption.ID, Title: option.MenuOption.Title, SortOrder: option.MenuOption.SortOrder,
		IsActive:  option.MenuOption.IsActive,
		IsAviable: publicParentAvailabilityResponse{HasResponse: option.HasResponse, HasChildren: option.HasChildren},
	}
}

func (h *PublicMenuHandler) getSubOptions(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "menu option id must be a positive integer"})
		return
	}
	detail, err := h.service.GetSubOptions(r.Context(), id)
	if errors.Is(err, menu.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "active menu option not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to retrieve sub options"})
		return
	}
	options := make([]publicParentMenuOptionResponse, 0, len(detail.Options))
	for _, option := range detail.Options {
		options = append(options, newPublicParentMenuOptionResponse(option))
	}
	response := publicSubOptionsResponse{Options: options}
	if detail.Prompt != nil {
		response.Prompt = &promptResponse{ID: detail.Prompt.ID, Message: detail.Prompt.Message}
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
	HasPrompts  bool    `json:"has_prompt"`
}

func newMenuOptionResponse(option domain.MenuOption) menuOptionResponse {
	return menuOptionResponse{ID: option.ID, ParentID: option.ParentOptionID, Title: option.Title, Description: option.Description, SortOrder: option.SortOrder, HasPrompts: option.HasPrompts}
}

type promptResponse struct {
	ID      int64  `json:"id"`
	Message string `json:"message"`
}

type responseGroupResponse struct {
	ID          int64                  `json:"id"`
	Name        string                 `json:"name"`
	Description *string                `json:"description"`
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
