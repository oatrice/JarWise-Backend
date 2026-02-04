package handlers

import (
	"encoding/json"
	"jarwise-backend/internal/service"
	"net/http"
)

type MigrationHandler struct {
	service service.MigrationService
}

func NewMigrationHandler(svc service.MigrationService) *MigrationHandler {
	return &MigrationHandler{
		service: svc,
	}
}

// HandleUpload processes the file upload request
// POST /api/v1/migrations/money-manager
func (h *MigrationHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	// 1. Parse Multipart Form
	// Limit upload size to 60MB (50MB mmbak + 10MB xls approx)
	if err := r.ParseMultipartForm(60 << 20); err != nil {
		http.Error(w, "File too large or invalid format", http.StatusBadRequest)
		return
	}

	// 2. Get Files
	mmbakFile, mmbakHeader, err := r.FormFile("mmbak_file")
	if err != nil {
		http.Error(w, "Missing mmbak_file", http.StatusBadRequest)
		return
	}
	defer mmbakFile.Close()

	xlsFile, xlsHeader, err := r.FormFile("xls_file")
	if err != nil {
		http.Error(w, "Missing xls_file", http.StatusBadRequest)
		return
	}
	defer xlsFile.Close()

	// 3. Call Service
	resp, err := h.service.ProcessUpload(r.Context(), mmbakHeader, xlsHeader)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 4. Return Response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
