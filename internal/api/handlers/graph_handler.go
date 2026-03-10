package handlers

import (
	"encoding/json"
	"jarwise-backend/internal/models"
	"jarwise-backend/internal/service"
	"net/http"
)

type GraphHandler struct {
	service service.GraphService
}

func NewGraphHandler(service service.GraphService) *GraphHandler {
	return &GraphHandler{service: service}
}

type GraphResponse struct {
	Data []models.GraphDataPoint `json:"data"`
}

func (h *GraphHandler) GetExpenseGraphData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	jarID := r.URL.Query().Get("id")
	period := r.URL.Query().Get("period")

	if jarID == "" || period == "" {
		http.Error(w, "Missing id or period parameter", http.StatusBadRequest)
		return
	}

	data, err := h.service.GetExpenseGraphData(jarID, period)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Ensure non-nil array for JSON
	if data == nil {
		data = []models.GraphDataPoint{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GraphResponse{Data: data})
}
