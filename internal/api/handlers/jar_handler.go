package handlers

import (
	"encoding/json"
	"jarwise-backend/internal/auth"
	"jarwise-backend/internal/repository"
	"net/http"
)

type JarHandler struct {
	repo repository.JarRepository
}

func NewJarHandler(repo repository.JarRepository) *JarHandler {
	return &JarHandler{repo: repo}
}

func (h *JarHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	jars, err := h.repo.ListAllForUser(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Failed to load jars", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jars)
}
