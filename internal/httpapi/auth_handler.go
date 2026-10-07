package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"calbot/internal/domain"
	"calbot/internal/user"
)

type AuthHandler struct {
	service *user.Service
}

func NewAuthHandler(service *user.Service) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/bootstrap-admin", h.bootstrapAdmin)
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.HandleFunc("POST /api/v1/staff", h.registerStaff)
}

func (h *AuthHandler) bootstrapAdmin(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	account, err := h.service.CreateInitialAdmin(r.Context(), request.Username, request.Password)
	if errors.Is(err, user.ErrInitialAdminExists) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "initial admin already exists"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, accountResponse(account))
}

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	token, account, err := h.service.Login(r.Context(), request.Username, request.Password)
	if errors.Is(err, user.ErrUnauthorized) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": accountResponse(account)})
}

func (h *AuthHandler) registerStaff(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token == r.Header.Get("Authorization") {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Bearer token is required"})
		return
	}
	account, err := h.service.RegisterStaff(r.Context(), token, request.Username, request.Password)
	if errors.Is(err, user.ErrUnauthorized) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
		return
	}
	if errors.Is(err, user.ErrForbidden) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role is required"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, accountResponse(account))
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON request"})
		return false
	}
	return true
}

func accountResponse(account domain.User) map[string]any {
	return map[string]any{"id": account.ID, "username": account.Username, "role": account.Role, "created_at": account.CreatedAt}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
