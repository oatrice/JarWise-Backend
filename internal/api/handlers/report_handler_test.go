package handlers

import (
	"context"
	"jarwise-backend/internal/auth"
	"jarwise-backend/internal/models"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockReportService struct{}

func (m *mockReportService) GenerateReport(ctx context.Context, filter models.ReportFilter) (*models.Report, error) {
	return nil, nil
}

func (m *mockReportService) GenerateReportForUser(ctx context.Context, _ string, filter models.ReportFilter) (*models.Report, error) {
	return m.GenerateReport(ctx, filter)
}

func (m *mockReportService) ExportTransactionsToCSV(ctx context.Context, filter models.ReportFilter) ([]byte, error) {
	return []byte("Date,Description,Amount,Type,Wallet,Jar\n2026-03-28,Test,100.00,expense,Main,Savings\n"), nil
}

func (m *mockReportService) ExportTransactionsToCSVForUser(ctx context.Context, _ string, filter models.ReportFilter) ([]byte, error) {
	return m.ExportTransactionsToCSV(ctx, filter)
}

func TestExportReport(t *testing.T) {
	svc := &mockReportService{}
	h := NewReportHandler(svc)

	req := httptest.NewRequest("GET", "/api/v1/reports/export?start_date=2026-03-01&end_date=2026-03-31", nil)
	req = req.WithContext(auth.ContextWithUser(req.Context(), &models.User{ID: "user-1"}))
	w := httptest.NewRecorder()

	h.ExportReport(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/csv" {
		t.Errorf("Expected Content-Type text/csv, got %s", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Date,Description,Amount,Type,Wallet,Jar") {
		t.Errorf("Response body missing header, got: %s", body)
	}
	if !strings.Contains(body, "2026-03-28,Test,100.00,expense,Main,Savings") {
		t.Errorf("Response body missing data, got: %s", body)
	}
}
