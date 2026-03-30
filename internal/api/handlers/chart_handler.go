package handlers

import (
	"encoding/json"
	"jarwise-backend/internal/auth"
	"jarwise-backend/internal/models"
	"jarwise-backend/internal/service"
	"net/http"
	"time"
)

// ChartHandler จัดการ HTTP request สำหรับ chart data
type ChartHandler struct {
	service service.ChartService
}

// NewChartHandler สร้าง ChartHandler instance ใหม่
func NewChartHandler(service service.ChartService) *ChartHandler {
	return &ChartHandler{service: service}
}

// GetChartData handler สำหรับ GET /api/v1/charts
func (h *ChartHandler) GetChartData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Default: เดือนปัจจุบัน
	now := time.Now().UTC()
	defaultStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	defaultEnd := defaultStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

	startDate, err := parseDateParam(r.URL.Query().Get("start_date"), defaultStart, false)
	if err != nil {
		http.Error(w, "Invalid start_date format. Use YYYY-MM-DD or RFC3339.", http.StatusBadRequest)
		return
	}
	endDate, err := parseDateParam(r.URL.Query().Get("end_date"), defaultEnd, true)
	if err != nil {
		http.Error(w, "Invalid end_date format. Use YYYY-MM-DD or RFC3339.", http.StatusBadRequest)
		return
	}
	if endDate.Before(startDate) {
		http.Error(w, "end_date must be after start_date", http.StatusBadRequest)
		return
	}

	jarIDs := parseIDsParam(r, "jar_ids", "category_ids")
	walletIDs := parseIDsParam(r, "wallet_ids", "account_ids")

	filter := models.ReportFilter{
		StartDate: startDate,
		EndDate:   endDate,
		JarIDs:    jarIDs,
		WalletIDs: walletIDs,
	}

	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	chart, err := h.service.GetChartDataForUser(r.Context(), user.ID, filter)
	if err != nil {
		http.Error(w, "Failed to generate chart data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chart)
}
