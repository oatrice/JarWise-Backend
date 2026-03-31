package handlers

import (
	"encoding/json"
	"fmt"
	"jarwise-backend/internal/auth"
	"jarwise-backend/internal/models"
	"jarwise-backend/internal/service"
	"log"
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

	filter, err := h.parseFilter(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	report, err := h.service.GenerateReportForUser(r.Context(), user.ID, filter)
	if err != nil {
		http.Error(w, "Failed to generate report", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func (h *ReportHandler) ExportReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filter, err := h.parseFilter(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	csvData, err := h.service.ExportTransactionsToCSVForUser(r.Context(), user.ID, filter)
	if err != nil {
		log.Printf("Export error: %v", err)
		http.Error(w, "Failed to export report: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Exporting CSV for user=%s: %d bytes (Filter: %v - %v)", user.ID, len(csvData), filter.StartDate, filter.EndDate)

	filename := "jarwise-report-" + time.Now().Format("2006-01-02-150405") + ".csv"
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Write(csvData)
}

func (h *ReportHandler) parseFilter(r *http.Request) (models.ReportFilter, error) {
	now := time.Now().UTC()
	defaultStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	defaultEnd := defaultStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

	startDate, err := parseDateParam(r.URL.Query().Get("start_date"), defaultStart, false)
	if err != nil {
		return models.ReportFilter{}, fmt.Errorf("invalid start_date format. Use YYYY-MM-DD or RFC3339")
	}
	endDate, err := parseDateParam(r.URL.Query().Get("end_date"), defaultEnd, true)
	if err != nil {
		return models.ReportFilter{}, fmt.Errorf("invalid end_date format. Use YYYY-MM-DD or RFC3339")
	}
	if endDate.Before(startDate) {
		return models.ReportFilter{}, fmt.Errorf("end_date must be after start_date")
	}

	jarIDs := parseIDsParam(r, "jar_ids", "category_ids")
	walletIDs := parseIDsParam(r, "wallet_ids", "account_ids")

	return models.ReportFilter{
		StartDate: startDate,
		EndDate:   endDate,
		JarIDs:    jarIDs,
		WalletIDs: walletIDs,
	}, nil
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
		// Fallback for fractional seconds if RFC3339 is strict (Go 1.x behavior varies)
		parsed, err = time.Parse(time.RFC3339Nano, value)
		if err != nil {
			return time.Time{}, err
		}
	}
	return parsed, nil
}
