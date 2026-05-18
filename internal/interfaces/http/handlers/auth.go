package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/lidar-platform/backend/internal/application/usecases"
)

type AuthHandler struct {
	auth *usecases.AuthUseCase
}

func NewAuthHandler(auth *usecases.AuthUseCase) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" {
		http.Error(w, "email and password required", http.StatusBadRequest)
		return
	}

	user, err := h.auth.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, usecases.ErrEmailTaken) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"id":    user.ID,
		"email": user.Email,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" {
		http.Error(w, "email and password required", http.StatusBadRequest)
		return
	}

	token, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, usecases.ErrInvalidCredentials) {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})
}
