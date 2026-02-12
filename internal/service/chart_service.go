package service

import (
	"context"
	"jarwise-backend/internal/models"
	"sort"
	"time"
)

// chartTransactionRepository interface สำหรับ dependency injection
type chartTransactionRepository interface {
	ListByDateRange(start, end time.Time) ([]models.Transaction, error)
}

// ChartService interface สำหรับ chart data aggregation
type ChartService interface {
	GetChartData(ctx context.Context, filter models.ReportFilter) (*models.ChartData, error)
}

type chartService struct {
	repo chartTransactionRepository
}

// NewChartService สร้าง ChartService instance ใหม่
func NewChartService(repo chartTransactionRepository) ChartService {
	return &chartService{repo: repo}
}

// GetChartData ดึงข้อมูล transactions แล้ว aggregate เป็น chart data ทั้งหมดในรอบเดียว
func (s *chartService) GetChartData(ctx context.Context, filter models.ReportFilter) (*models.ChartData, error) {
	// 1. ดึง transactions ตาม date range
	transactions, err := s.repo.ListByDateRange(filter.StartDate, filter.EndDate)
	if err != nil {
		return nil, err
	}

	// 2. Apply jar/wallet filters
	filtered := applyReportFilters(transactions, filter)

	// 3. Aggregate ข้อมูลทั้งหมดใน single pass
	chart := s.aggregate(filtered)

	// 4. Comparison: ดึงข้อมูล previous period
	comparison, err := s.buildComparison(ctx, filter, chart.Summary)
	if err != nil {
		return nil, err
	}
	chart.Comparison = comparison

	return chart, nil
}

// aggregate รวมข้อมูล transactions เป็น summary, trend, byJar ในรอบเดียว
func (s *chartService) aggregate(transactions []models.Transaction) *models.ChartData {
	var totalIncome, totalExpense float64

	// Maps สำหรับ grouping
	trendMap := make(map[string]*models.TrendPoint) // key: "2026-01"
	jarMap := make(map[string]*models.JarAmount)    // key: jar_id

	for _, tx := range transactions {
		switch tx.Type {
		case "income":
			totalIncome += tx.Amount
		case "expense":
			totalExpense += tx.Amount
		}

		// Trend: group by เดือน (YYYY-MM)
		monthKey := tx.Date.Format("2006-01")
		if _, ok := trendMap[monthKey]; !ok {
			trendMap[monthKey] = &models.TrendPoint{Date: monthKey}
		}
		switch tx.Type {
		case "income":
			trendMap[monthKey].Income += tx.Amount
		case "expense":
			trendMap[monthKey].Expense += tx.Amount
		}

		// ByJar: เฉพาะ expense
		if tx.Type == "expense" && tx.JarID != "" {
			if _, ok := jarMap[tx.JarID]; !ok {
				jarMap[tx.JarID] = &models.JarAmount{ID: tx.JarID, Name: tx.JarID}
			}
			jarMap[tx.JarID].Amount += tx.Amount
		}
	}

	// Convert maps to sorted slices
	trend := make([]models.TrendPoint, 0, len(trendMap))
	for _, tp := range trendMap {
		trend = append(trend, *tp)
	}
	sort.Slice(trend, func(i, j int) bool { return trend[i].Date < trend[j].Date })

	byJar := make([]models.JarAmount, 0, len(jarMap))
	for _, ja := range jarMap {
		byJar = append(byJar, *ja)
	}
	sort.Slice(byJar, func(i, j int) bool { return byJar[i].Amount > byJar[j].Amount })

	return &models.ChartData{
		Summary: models.ChartSummary{
			Income:  totalIncome,
			Expense: totalExpense,
			Net:     totalIncome - totalExpense,
		},
		Trend: trend,
		ByJar: byJar,
	}
}

// buildComparison คำนวณ previous period แล้วเปรียบเทียบ
func (s *chartService) buildComparison(ctx context.Context, filter models.ReportFilter, currentSummary models.ChartSummary) (*models.ComparisonData, error) {
	duration := filter.EndDate.Sub(filter.StartDate)
	prevStart := filter.StartDate.Add(-duration - time.Nanosecond)
	prevEnd := filter.StartDate.Add(-time.Nanosecond)

	prevTransactions, err := s.repo.ListByDateRange(prevStart, prevEnd)
	if err != nil {
		return nil, err
	}

	prevFiltered := applyReportFilters(prevTransactions, models.ReportFilter{
		StartDate: prevStart,
		EndDate:   prevEnd,
		JarIDs:    filter.JarIDs,
		WalletIDs: filter.WalletIDs,
	})

	var prevIncome, prevExpense float64
	for _, tx := range prevFiltered {
		switch tx.Type {
		case "income":
			prevIncome += tx.Amount
		case "expense":
			prevExpense += tx.Amount
		}
	}

	return &models.ComparisonData{
		Current: currentSummary,
		Previous: models.ChartSummary{
			Income:  prevIncome,
			Expense: prevExpense,
			Net:     prevIncome - prevExpense,
		},
	}, nil
}
