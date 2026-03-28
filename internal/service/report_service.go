package service

import (
	"context"
	"jarwise-backend/internal/models"
	"sort"
	"time"
)

type reportTransactionRepository interface {
	ListByDateRange(start, end time.Time) ([]models.Transaction, error)
}

type ReportService interface {
	GenerateReport(ctx context.Context, filter models.ReportFilter) (*models.Report, error)
}

type reportService struct {
	repo reportTransactionRepository
}

func NewReportService(repo reportTransactionRepository) ReportService {
	return &reportService{repo: repo}
}

func (s *reportService) GenerateReport(ctx context.Context, filter models.ReportFilter) (*models.Report, error) {
	// 1. Fetch transactions for the requested period
	transactions, err := s.repo.ListByDateRange(filter.StartDate, filter.EndDate)
	if err != nil {
		return nil, err
	}

	// 2. Apply filters (Jar/Wallet)
	filtered := applyReportFilters(transactions, filter)

	// 3. Aggregate Current Period
	report := s.aggregate(filtered, filter)

	// 4. Calculate Comparison (Previous Period)
	comparison, err := s.buildComparison(ctx, filter, report.Summary)
	if err != nil {
		// Log error but don't fail the whole report if comparison fails
		report.Comparison = &models.ComparisonData{Current: report.Summary}
	} else {
		report.Comparison = comparison
	}

	// 5. Populate legacy fields for compatibility
	report.FilterUsed = filter
	report.Transactions = filtered
	report.TransactionCount = len(filtered)
	report.TotalAmount = report.Summary.Net

	return report, nil
}

func (s *reportService) aggregate(transactions []models.Transaction, filter models.ReportFilter) *models.Report {
	var summary models.ChartSummary
	trendMap := make(map[string]*models.TrendPoint)
	categoryMap := make(map[string]*models.CategoryAmount)
	jarMap := make(map[string]*models.JarAmount)

	// Determine bucket format (Daily vs Monthly)
	bucketFormat := "2006-01"
	if filter.EndDate.Sub(filter.StartDate) <= 31*24*time.Hour {
		bucketFormat = "2006-01-02"
	}

	for _, tx := range transactions {
		// Update Summary
		switch tx.Type {
		case "income":
			summary.Income += tx.Amount
		case "expense":
			summary.Expense += tx.Amount
		}
		summary.Net = summary.Income - summary.Expense

		// Update Trend
		dateKey := tx.Date.Format(bucketFormat)
		if _, ok := trendMap[dateKey]; !ok {
			trendMap[dateKey] = &models.TrendPoint{Date: dateKey}
		}
		if tx.Type == "income" {
			trendMap[dateKey].Income += tx.Amount
		} else if tx.Type == "expense" {
			trendMap[dateKey].Expense += tx.Amount
		}

		// Update Category Breakdown (Both Income & Expense)
		if tx.JarID != "" {
			if _, ok := categoryMap[tx.JarID]; !ok {
				// We use JarID as Name for now, in a real DB we'd fetch the Name
				categoryMap[tx.JarID] = &models.CategoryAmount{ID: tx.JarID, Name: tx.JarID}
			}
			if tx.Type == "income" {
				categoryMap[tx.JarID].Income += tx.Amount
			} else if tx.Type == "expense" {
				categoryMap[tx.JarID].Expense += tx.Amount
				categoryMap[tx.JarID].Amount += tx.Amount
			}
		}

		// Update Jar Breakdown (Simplified version of Category for this domain)
		if tx.JarID != "" {
			if _, ok := jarMap[tx.JarID]; !ok {
				jarMap[tx.JarID] = &models.JarAmount{ID: tx.JarID, Name: tx.JarID}
			}
			if tx.Type == "income" {
				jarMap[tx.JarID].Income += tx.Amount
			} else if tx.Type == "expense" {
				jarMap[tx.JarID].Expense += tx.Amount
				jarMap[tx.JarID].Amount += tx.Amount
			}
		}
	}

	// Convert maps to slices
	return &models.Report{
		Summary:    summary,
		Trend:      sortTrend(trendMap),
		ByCategory: sortCategories(categoryMap),
		ByJar:      sortJars(jarMap),
	}
}

func (s *reportService) buildComparison(ctx context.Context, filter models.ReportFilter, currentSummary models.ChartSummary) (*models.ComparisonData, error) {
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

// Helpers for sorting

func sortTrend(m map[string]*models.TrendPoint) []models.TrendPoint {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	res := make([]models.TrendPoint, 0, len(keys))
	for _, k := range keys {
		res = append(res, *m[k])
	}
	return res
}

func sortCategories(m map[string]*models.CategoryAmount) []models.CategoryAmount {
	res := make([]models.CategoryAmount, 0, len(m))
	for _, v := range m {
		res = append(res, *v)
	}
	sort.Slice(res, func(i, j int) bool { return res[i].Amount > res[j].Amount })
	return res
}

func sortJars(m map[string]*models.JarAmount) []models.JarAmount {
	res := make([]models.JarAmount, 0, len(m))
	for _, v := range m {
		res = append(res, *v)
	}
	sort.Slice(res, func(i, j int) bool { return res[i].Amount > res[j].Amount })
	return res
}

func applyReportFilters(transactions []models.Transaction, filter models.ReportFilter) []models.Transaction {
	if len(filter.JarIDs) == 0 && len(filter.WalletIDs) == 0 {
		return transactions
	}

	jarSet := make(map[string]struct{}, len(filter.JarIDs))
	for _, id := range filter.JarIDs {
		jarSet[id] = struct{}{}
	}

	walletSet := make(map[string]struct{}, len(filter.WalletIDs))
	for _, id := range filter.WalletIDs {
		walletSet[id] = struct{}{}
	}

	var results []models.Transaction
	for _, tx := range transactions {
		jarMatch := len(jarSet) == 0 || (tx.JarID != "" && containsKey(jarSet, tx.JarID))
		walletMatch := len(walletSet) == 0 || containsKey(walletSet, tx.WalletID)
		if jarMatch && walletMatch {
			results = append(results, tx)
		}
	}

	return results
}

func containsKey(set map[string]struct{}, key string) bool {
	_, ok := set[key]
	return ok
}
