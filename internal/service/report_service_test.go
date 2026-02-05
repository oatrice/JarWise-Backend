package service

import (
	"context"
	"jarwise-backend/internal/models"
	"testing"
	"time"
)

type fakeReportRepo struct {
	transactions []models.Transaction
}

func (f *fakeReportRepo) ListByDateRange(start, end time.Time) ([]models.Transaction, error) {
	var results []models.Transaction
	for _, tx := range f.transactions {
		if !tx.Date.Before(start) && !tx.Date.After(end) {
			results = append(results, tx)
		}
	}
	return results, nil
}

func TestGenerateReport_NoFilters(t *testing.T) {
	service := NewReportService(&fakeReportRepo{transactions: seedReportTransactions()})
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC)

	report, err := service.GenerateReport(context.Background(), models.ReportFilter{
		StartDate: start,
		EndDate:   end,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.TransactionCount != 4 {
		t.Fatalf("expected 4 transactions, got %d", report.TransactionCount)
	}
}

func TestGenerateReport_FilterByJar(t *testing.T) {
	service := NewReportService(&fakeReportRepo{transactions: seedReportTransactions()})
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC)

	report, err := service.GenerateReport(context.Background(), models.ReportFilter{
		StartDate: start,
		EndDate:   end,
		JarIDs:    []string{"jar-1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.TransactionCount != 2 {
		t.Fatalf("expected 2 transactions, got %d", report.TransactionCount)
	}
	for _, tx := range report.Transactions {
		if tx.JarID != "jar-1" {
			t.Fatalf("unexpected jar id: %s", tx.JarID)
		}
	}
}

func TestGenerateReport_FilterByWallet(t *testing.T) {
	service := NewReportService(&fakeReportRepo{transactions: seedReportTransactions()})
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC)

	report, err := service.GenerateReport(context.Background(), models.ReportFilter{
		StartDate: start,
		EndDate:   end,
		WalletIDs: []string{"wallet-2"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.TransactionCount != 2 {
		t.Fatalf("expected 2 transactions, got %d", report.TransactionCount)
	}
	for _, tx := range report.Transactions {
		if tx.WalletID != "wallet-2" {
			t.Fatalf("unexpected wallet id: %s", tx.WalletID)
		}
	}
}

func TestGenerateReport_FilterByJarAndWallet(t *testing.T) {
	service := NewReportService(&fakeReportRepo{transactions: seedReportTransactions()})
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC)

	report, err := service.GenerateReport(context.Background(), models.ReportFilter{
		StartDate: start,
		EndDate:   end,
		JarIDs:    []string{"jar-1"},
		WalletIDs: []string{"wallet-2"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.TransactionCount != 1 {
		t.Fatalf("expected 1 transaction, got %d", report.TransactionCount)
	}
	if report.Transactions[0].JarID != "jar-1" || report.Transactions[0].WalletID != "wallet-2" {
		t.Fatalf("unexpected filter result: jar=%s wallet=%s", report.Transactions[0].JarID, report.Transactions[0].WalletID)
	}
}

func TestGenerateReport_NoResults(t *testing.T) {
	service := NewReportService(&fakeReportRepo{transactions: seedReportTransactions()})
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC)

	report, err := service.GenerateReport(context.Background(), models.ReportFilter{
		StartDate: start,
		EndDate:   end,
		JarIDs:    []string{"missing"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.TransactionCount != 0 {
		t.Fatalf("expected 0 transactions, got %d", report.TransactionCount)
	}
}

func seedReportTransactions() []models.Transaction {
	return []models.Transaction{
		{
			ID:       "tx-1",
			Amount:   120.0,
			Date:     time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC),
			Type:     "expense",
			JarID:    "jar-1",
			WalletID: "wallet-1",
		},
		{
			ID:       "tx-2",
			Amount:   80.0,
			Date:     time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC),
			Type:     "expense",
			JarID:    "jar-2",
			WalletID: "wallet-2",
		},
		{
			ID:       "tx-3",
			Amount:   42.5,
			Date:     time.Date(2026, 1, 8, 12, 0, 0, 0, time.UTC),
			Type:     "income",
			JarID:    "jar-1",
			WalletID: "wallet-2",
		},
		{
			ID:       "tx-4",
			Amount:   200.0,
			Date:     time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC),
			Type:     "expense",
			JarID:    "jar-3",
			WalletID: "wallet-3",
		},
	}
}
