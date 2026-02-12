package handlers

import (
	"encoding/json"
	"jarwise-backend/internal/models"
	"jarwise-backend/internal/service"
	"net/http"
	"strings"
	"time"
)

type ReportHandler struct {
	service service.ReportService
}

func NewReportHandler(service service.ReportService) *ReportHandler {
	return &ReportHandler{service: service}
}

func (h *ReportHandler) GetReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

	report, err := h.service.GenerateReport(r.Context(), filter)
	if err != nil {
		http.Error(w, "Failed to generate report", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func parseIDsParam(r *http.Request, keys ...string) []string {
	for _, key := range keys {
		raw := strings.TrimSpace(r.URL.Query().Get(key))
		if raw == "" {
			continue
		}
		return splitCommaSeparated(raw)
	}
	return []string{}
}

func splitCommaSeparated(value string) []string {
	parts := strings.Split(value, ",")
	results := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		results = append(results, trimmed)
	}
	return results
}

func parseDateParam(value string, defaultValue time.Time, isEnd bool) (time.Time, error) {
	if value == "" {
		return defaultValue, nil
	}

	if len(value) == len("2006-01-02") {
		parsed, err := time.ParseInLocation("2006-01-02", value, time.UTC)
		if err != nil {
			return time.Time{}, err
		}
		if isEnd {
			return parsed.AddDate(0, 0, 1).Add(-time.Nanosecond), nil
		}
		return parsed, nil
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, err
	}
	return parsed, nil
}
