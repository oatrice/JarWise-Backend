package handlers

import (
	"encoding/json"
	"errors"
	"jarwise-backend/internal/auth"
	"jarwise-backend/internal/service"
	"net/http"
	"strings"
)

type MigrationHandler struct {
	service service.MigrationService
}

func NewMigrationHandler(svc service.MigrationService) *MigrationHandler {
	return &MigrationHandler{service: svc}
}

func (h *MigrationHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	if err := r.ParseMultipartForm(60 << 20); err != nil {
		http.Error(w, "File too large or invalid format", http.StatusBadRequest)
		return
	}

	mmbakFile, _, err := r.FormFile("mmbak_file")
	if err != nil {
		http.Error(w, "Missing mmbak_file", http.StatusBadRequest)
		return
	}
	mmbakFile.Close()

	xlsFile, _, err := r.FormFile("xls_file")
	if err != nil {
		http.Error(w, "Missing xls_file", http.StatusBadRequest)
		return
	}
	xlsFile.Close()

	mmbakHeader := r.MultipartForm.File["mmbak_file"][0]
	xlsHeader := r.MultipartForm.File["xls_file"][0]

	resp, err := h.service.CreateJob(r.Context(), user.ID, mmbakHeader, xlsHeader)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(resp)
}

func (h *MigrationHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	jobID, err := migrationJobIDFromPath(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := h.service.GetJob(r.Context(), user.ID, jobID)
	if err != nil {
		if errors.Is(err, service.ErrMigrationJobNotFound) {
			http.Error(w, "Migration job not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to load migration job", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *MigrationHandler) ConfirmJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	jobID, err := migrationJobIDFromPath(strings.TrimSuffix(r.URL.Path, "/confirm"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := h.service.ConfirmJob(r.Context(), user.ID, jobID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMigrationJobNotFound):
			http.Error(w, "Migration job not found", http.StatusNotFound)
		case errors.Is(err, service.ErrMigrationJobConflict):
			http.Error(w, "Migration job cannot be confirmed in its current state", http.StatusConflict)
		default:
			http.Error(w, "Failed to confirm migration job", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func migrationJobIDFromPath(path string) (string, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 6 {
		return "", errors.New("invalid migration job path")
	}
	return parts[5], nil
}
