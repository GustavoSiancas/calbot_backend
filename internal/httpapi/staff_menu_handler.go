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
	mux.HandleFunc("GET /api/v1/staff/menu-options", h.listAll)
	mux.HandleFunc("GET /api/v1/staff/menu-options/{id}", h.getByID)
	mux.HandleFunc("DELETE /api/v1/staff/menu-options/{id}", h.delete)
	mux.HandleFunc("GET /api/v1/staff/menu-options/tree", h.listByParentQuery)
	mux.HandleFunc("POST /api/v1/staff/menu-options", h.create)
	mux.HandleFunc("PATCH /api/v1/staff/menu-options/{id}/deactivate", h.deactivate)
}

func (h *StaffMenuHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parsePositivePathID(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "menu option id must be a positive integer"})
		return
	}
	err = h.service.Delete(r.Context(), bearerToken(r), id)
	if !handleMenuAuthorizationError(w, err) {
		return
	}
	if errors.Is(err, menu.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "menu option not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to delete menu option"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *StaffMenuHandler) listByParentQuery(w http.ResponseWriter, r *http.Request) {
	var parentID *int64
	if value := r.URL.Query().Get("parent_id"); value != "" {
		parsedID, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsedID < 1 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "parent_id must be a positive integer"})
			return
		}
		parentID = &parsedID
	}
	h.listByParent(w, r, parentID)
}

func (h *StaffMenuHandler) listByParent(w http.ResponseWriter, r *http.Request, parentID *int64) {
	options, err := h.service.ListByParentForStaff(r.Context(), bearerToken(r), parentID)
	if !handleMenuAuthorizationError(w, err) {
		return
	}
	if errors.Is(err, menu.ErrInvalidInput) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "parent id must be a positive integer"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to retrieve menu options"})
		return
	}
	response := make([]staffMenuOptionTreeResponse, 0, len(options))
	for _, option := range options {
		response = append(response, staffMenuOptionTreeResponse{staffMenuOptionRowResponse: newStaffMenuOptionRowResponse(option.MenuOption), HasResponse: option.HasResponse, HasChildren: option.HasChildren})
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *StaffMenuHandler) getByID(w http.ResponseWriter, r *http.Request) {
	id, err := parsePositivePathID(r, "id")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "menu option id must be a positive integer"})
		return
	}
	option, err := h.service.GetByIDForStaff(r.Context(), bearerToken(r), id)
	if !handleMenuAuthorizationError(w, err) {
		return
	}
	if errors.Is(err, menu.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "menu option not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unable to retrieve menu option"})
		return
	}
	writeJSON(w, http.StatusOK, newStaffMenuOptionRowResponse(option))
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
			prompts = append(prompts, staffPromptResponse{ID: prompt.ID, Message: prompt.Message, Weight: prompt.Weight, IsActive: prompt.IsActive, CreatedAt: prompt.CreatedAt, UpdatedAt: prompt.UpdatedAt})
		}
		groups := make([]staffResponseGroupResponse, 0, len(option.ResponseGroups))
		for _, group := range option.ResponseGroups {
			items := make([]staffResponseItemResponse, 0, len(group.ResponseItems))
			for _, item := range group.ResponseItems {
				items = append(items, staffResponseItemResponse{ID: item.ID, Type: item.Type, Text: item.Text, URL: item.URL, Caption: item.Caption, Metadata: item.Metadata, SortOrder: item.SortOrder, IsActive: item.IsActive, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt})
			}
			groups = append(groups, staffResponseGroupResponse{ID: group.ResponseGroup.ID, MenuOptionID: group.ResponseGroup.MenuOptionID, Name: group.ResponseGroup.Name, Description: group.ResponseGroup.Description, Weight: group.ResponseGroup.Weight, IsActive: group.ResponseGroup.IsActive, CreatedAt: group.ResponseGroup.CreatedAt, UpdatedAt: group.ResponseGroup.UpdatedAt, Items: items})
		}
		response = append(response, staffMenuOptionResponse{ID: option.MenuOption.ID, ParentOptionID: option.MenuOption.ParentOptionID, Title: option.MenuOption.Title, Description: option.MenuOption.Description, SortOrder: option.MenuOption.SortOrder, IsActive: option.MenuOption.IsActive, CreatedAt: option.MenuOption.CreatedAt, UpdatedAt: option.MenuOption.UpdatedAt, Prompts: prompts, ResponseGroups: groups})
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *StaffMenuHandler) create(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ParentOptionID *int64 `json:"parent_option_id"`
		Title          string `json:"title"`
		SortOrder      int    `json:"sort_order"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	option, err := h.service.Create(r.Context(), bearerToken(r), menu.CreateInput{
		ParentOptionID: request.ParentOptionID, Title: request.Title, SortOrder: request.SortOrder,
	})
	if !handleMenuAuthorizationError(w, err) {
		return
	}
	if errors.Is(err, menu.ErrInvalidInput) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
		return
	}
	if errors.Is(err, menu.ErrMenuOptionHasResponse) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "cannot create a child option because its parent already has response items"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unable to create menu option"})
		return
	}

	writeJSON(w, http.StatusCreated, newMenuOptionResponse(option))
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

type staffMenuOptionRowResponse struct {
	ID             int64     `json:"id"`
	ParentOptionID *int64    `json:"parent_option_id"`
	Title          string    `json:"title"`
	Description    *string   `json:"description"`
	SortOrder      int       `json:"sort_order"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type staffMenuOptionTreeResponse struct {
	staffMenuOptionRowResponse
	HasResponse bool `json:"has_response"`
	HasChildren bool `json:"has_children"`
}

func newStaffMenuOptionRowResponse(option domain.MenuOption) staffMenuOptionRowResponse {
	return staffMenuOptionRowResponse{ID: option.ID, ParentOptionID: option.ParentOptionID, Title: option.Title, Description: option.Description, SortOrder: option.SortOrder, IsActive: option.IsActive, CreatedAt: option.CreatedAt, UpdatedAt: option.UpdatedAt}
}

type staffPromptResponse struct {
	ID        int64     `json:"id"`
	Message   string    `json:"message"`
	Weight    int       `json:"weight"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type staffResponseGroupResponse struct {
	ID           int64                       `json:"id"`
	MenuOptionID int64                       `json:"menu_option_id"`
	Name         string                      `json:"name"`
	Description  *string                     `json:"description"`
	Weight       int                         `json:"weight"`
	IsActive     bool                        `json:"is_active"`
	CreatedAt    time.Time                   `json:"created_at"`
	UpdatedAt    time.Time                   `json:"updated_at"`
	Items        []staffResponseItemResponse `json:"items"`
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
