package service

import (
	"context"
	"jarwise-backend/internal/models"
	"testing"
	"time"
)

// fakeChartRepo จำลอง repository สำหรับทดสอบ
type fakeChartRepo struct {
	transactions []models.Transaction
}

func (f *fakeChartRepo) ListByDateRange(start, end time.Time) ([]models.Transaction, error) {
	var results []models.Transaction
	for _, tx := range f.transactions {
		if !tx.Date.Before(start) && !tx.Date.After(end) {
			results = append(results, tx)
		}
	}
	return results, nil
}

func seedChartTransactions() []models.Transaction {
	return []models.Transaction{
		{
			ID: "tx-1", Amount: 5000.0, Date: time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC),
			Type: "income", JarID: "jar-income", WalletID: "wallet-1",
		},
		{
			ID: "tx-2", Amount: 1200.0, Date: time.Date(2026, 1, 8, 12, 0, 0, 0, time.UTC),
			Type: "expense", JarID: "jar-food", WalletID: "wallet-1",
		},
		{
			ID: "tx-3", Amount: 800.0, Date: time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC),
			Type: "expense", JarID: "jar-transport", WalletID: "wallet-2",
		},
		{
			ID: "tx-4", Amount: 3000.0, Date: time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC),
			Type: "income", JarID: "jar-income", WalletID: "wallet-1",
		},
		{
			ID: "tx-5", Amount: 500.0, Date: time.Date(2026, 1, 25, 12, 0, 0, 0, time.UTC),
			Type: "expense", JarID: "jar-food", WalletID: "wallet-1",
		},
	}
}

// --- 🟥 RED: Failing Tests ---

func TestChartService_Summary(t *testing.T) {
	svc := NewChartService(&fakeChartRepo{transactions: seedChartTransactions()})
	filter := models.ReportFilter{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	}

	chart, err := svc.GetChartData(context.Background(), filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// income: 5000 + 3000 = 8000
	if chart.Summary.Income != 8000 {
		t.Errorf("expected income 8000, got %f", chart.Summary.Income)
	}
	// expense: 1200 + 800 + 500 = 2500
	if chart.Summary.Expense != 2500 {
		t.Errorf("expected expense 2500, got %f", chart.Summary.Expense)
	}
	// net: 8000 - 2500 = 5500
	if chart.Summary.Net != 5500 {
		t.Errorf("expected net 5500, got %f", chart.Summary.Net)
	}
}

func TestChartService_Trend(t *testing.T) {
	svc := NewChartService(&fakeChartRepo{transactions: seedChartTransactions()})
	filter := models.ReportFilter{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	}

	chart, err := svc.GetChartData(context.Background(), filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// ข้อมูลควรจะ group by เดือน → ได้ 1 เดือน (2026-01)
	if len(chart.Trend) != 1 {
		t.Fatalf("expected 1 trend point, got %d", len(chart.Trend))
	}

	if chart.Trend[0].Date != "2026-01" {
		t.Errorf("expected date '2026-01', got '%s'", chart.Trend[0].Date)
	}
	if chart.Trend[0].Income != 8000 {
		t.Errorf("expected trend income 8000, got %f", chart.Trend[0].Income)
	}
	if chart.Trend[0].Expense != 2500 {
		t.Errorf("expected trend expense 2500, got %f", chart.Trend[0].Expense)
	}
}

func TestChartService_ByJar(t *testing.T) {
	svc := NewChartService(&fakeChartRepo{transactions: seedChartTransactions()})
	filter := models.ReportFilter{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	}

	chart, err := svc.GetChartData(context.Background(), filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// expense jars: jar-food (1200+500=1700), jar-transport (800)
	if len(chart.ByJar) != 2 {
		t.Fatalf("expected 2 jar entries, got %d", len(chart.ByJar))
	}

	jarMap := make(map[string]float64)
	for _, j := range chart.ByJar {
		jarMap[j.ID] = j.Amount
	}

	if jarMap["jar-food"] != 1700 {
		t.Errorf("expected jar-food amount 1700, got %f", jarMap["jar-food"])
	}
	if jarMap["jar-transport"] != 800 {
		t.Errorf("expected jar-transport amount 800, got %f", jarMap["jar-transport"])
	}
}

func TestChartService_EmptyData(t *testing.T) {
	svc := NewChartService(&fakeChartRepo{transactions: []models.Transaction{}})
	filter := models.ReportFilter{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	}

	chart, err := svc.GetChartData(context.Background(), filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if chart.Summary.Income != 0 || chart.Summary.Expense != 0 || chart.Summary.Net != 0 {
		t.Error("expected all zero summary for empty data")
	}
	if len(chart.Trend) != 0 {
		t.Errorf("expected 0 trend points, got %d", len(chart.Trend))
	}
	if len(chart.ByJar) != 0 {
		t.Errorf("expected 0 jar entries, got %d", len(chart.ByJar))
	}
}

func TestChartService_FilterByJar(t *testing.T) {
	svc := NewChartService(&fakeChartRepo{transactions: seedChartTransactions()})
	filter := models.ReportFilter{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
		JarIDs:    []string{"jar-food"},
	}

	chart, err := svc.GetChartData(context.Background(), filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// expense ของ jar-food: 1200 + 500 = 1700, income = 0
	if chart.Summary.Expense != 1700 {
		t.Errorf("expected expense 1700, got %f", chart.Summary.Expense)
	}
	if chart.Summary.Income != 0 {
		t.Errorf("expected income 0, got %f", chart.Summary.Income)
	}
}

func TestChartService_Comparison(t *testing.T) {
	// ข้อมูลคาบเกี่ยว 2 เดือน
	txns := []models.Transaction{
		// ธ.ค. 2025 (previous)
		{ID: "p1", Amount: 3000, Date: time.Date(2025, 12, 10, 0, 0, 0, 0, time.UTC), Type: "income", JarID: "j1", WalletID: "w1"},
		{ID: "p2", Amount: 1000, Date: time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC), Type: "expense", JarID: "j2", WalletID: "w1"},
		// ม.ค. 2026 (current)
		{ID: "c1", Amount: 5000, Date: time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC), Type: "income", JarID: "j1", WalletID: "w1"},
		{ID: "c2", Amount: 2000, Date: time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC), Type: "expense", JarID: "j2", WalletID: "w1"},
	}

	svc := NewChartService(&fakeChartRepo{transactions: txns})
	// filter ม.ค. 2026
	filter := models.ReportFilter{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	}

	chart, err := svc.GetChartData(context.Background(), filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if chart.Comparison == nil {
		t.Fatal("expected comparison data, got nil")
	}

	// current: income=5000, expense=2000
	if chart.Comparison.Current.Income != 5000 {
		t.Errorf("expected current income 5000, got %f", chart.Comparison.Current.Income)
	}
	if chart.Comparison.Current.Expense != 2000 {
		t.Errorf("expected current expense 2000, got %f", chart.Comparison.Current.Expense)
	}

	// previous: income=3000, expense=1000
	if chart.Comparison.Previous.Income != 3000 {
		t.Errorf("expected previous income 3000, got %f", chart.Comparison.Previous.Income)
	}
	if chart.Comparison.Previous.Expense != 1000 {
		t.Errorf("expected previous expense 1000, got %f", chart.Comparison.Previous.Expense)
	}
}
