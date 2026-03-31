package handlers

import (
	"encoding/json"
	"jarwise-backend/internal/auth"
	"jarwise-backend/internal/repository"
	"net/http"
	"strings"
)

type WalletHandler struct {
	repo repository.WalletRepository
}

func NewWalletHandler(repo repository.WalletRepository) *WalletHandler {
	return &WalletHandler{repo: repo}
}

func (h *WalletHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Simple path parsing /api/v1/wallets/:id
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		http.Error(w, "Invalid wallet ID", http.StatusBadRequest)
		return
	}
	id := pathParts[4]
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	cascade := r.URL.Query().Get("cascade") == "true"
	replacementID := r.URL.Query().Get("replacement_id")

	if cascade {
		// Cascade delete
		err := h.repo.DeleteCascadeForUser(user.ID, id)
		if err != nil {
			http.Error(w, "Failed to cascade delete wallet: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else if replacementID == "" {
		// Attempt direct delete
		err := h.repo.DeleteForUser(user.ID, id)
		if err != nil {
			http.Error(w, "Failed to delete wallet (it might have transactions): "+err.Error(), http.StatusConflict)
			return
		}
	} else {
		// Delete with replacement
		err := h.repo.DeleteWithReplacementForUser(user.ID, id, replacementID)
		if err != nil {
			http.Error(w, "Failed to delete with replacement: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *WalletHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	wallets, err := h.repo.ListAllForUser(user.ID)
	if err != nil {
		http.Error(w, "Failed to load wallets", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wallets)
}
