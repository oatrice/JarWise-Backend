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

type jarRepository interface {
	ListAll(ctx context.Context) ([]models.Jar, error)
}

type ReportService interface {
	GenerateReport(ctx context.Context, filter models.ReportFilter) (*models.Report, error)
}

type reportService struct {
	repo    reportTransactionRepository
	jarRepo jarRepository
}

func NewReportService(repo reportTransactionRepository, jarRepo jarRepository) ReportService {
	return &reportService{repo: repo, jarRepo: jarRepo}
}

func (s *reportService) GenerateReport(ctx context.Context, filter models.ReportFilter) (*models.Report, error) {
	// 1. Fetch jars for name mapping
	jars, err := s.jarRepo.ListAll(ctx)
	if err != nil {
		// Log error but continue with IDs if jars can't be fetched
		jars = []models.Jar{}
	}
	jarNameMap := make(map[string]string)
	for _, j := range jars {
		jarNameMap[j.ID] = j.Name
	}

	// 2. Fetch transactions for the requested period
	transactions, err := s.repo.ListByDateRange(filter.StartDate, filter.EndDate)
	if err != nil {
		return nil, err
	}

	// 3. Apply filters (Jar/Wallet)
	filtered := applyReportFilters(transactions, filter)

	// 4. Aggregate Current Period
	report := s.aggregate(filtered, filter, jarNameMap)

	// 5. Calculate Comparison and Category comparisons
	prevStart := filter.StartDate.AddDate(0, -1, 0)
	duration := filter.EndDate.Sub(filter.StartDate)
	prevEnd := prevStart.Add(duration)

	// Cap prevEnd to not overlap with the current period
	if prevEnd.After(filter.StartDate) {
		prevEnd = filter.StartDate.Add(-time.Nanosecond)
	}

	prevTransactions, err := s.repo.ListByDateRange(prevStart, prevEnd)
	if err == nil {
		prevFiltered := applyReportFilters(prevTransactions, models.ReportFilter{
			StartDate: prevStart,
			EndDate:   prevEnd,
			JarIDs:    filter.JarIDs,
			WalletIDs: filter.WalletIDs,
		})
		prevReport := s.aggregate(prevFiltered, filter, jarNameMap)

		// Merge Previous Expense into Categories
		catPrevMap := make(map[string]float64)
		for _, cat := range prevReport.ByCategory {
			catPrevMap[cat.ID] = cat.Expense
		}
		for i := range report.ByCategory {
			report.ByCategory[i].PrevExpense = catPrevMap[report.ByCategory[i].ID]
		}

		jarPrevMap := make(map[string]float64)
		for _, j := range prevReport.ByJar {
			jarPrevMap[j.ID] = j.Expense
		}
		for i := range report.ByJar {
			report.ByJar[i].PrevExpense = jarPrevMap[report.ByJar[i].ID]
		}

		report.Comparison = &models.ComparisonData{
			Current:  report.Summary,
			Previous: prevReport.Summary,
		}
	} else {
		report.Comparison = &models.ComparisonData{Current: report.Summary}
	}

	// 6. Populate legacy fields
	report.FilterUsed = filter
	report.Transactions = filtered
	report.TransactionCount = len(filtered)
	report.TotalAmount = report.Summary.Net

	return report, nil
}

func (s *reportService) aggregate(transactions []models.Transaction, filter models.ReportFilter, jarNames map[string]string) *models.Report {
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
				name := tx.JarID
				if n, exists := jarNames[tx.JarID]; exists {
					name = n
				}
				categoryMap[tx.JarID] = &models.CategoryAmount{ID: tx.JarID, Name: name}
			}
			if tx.Type == "income" {
				categoryMap[tx.JarID].Income += tx.Amount
			} else if tx.Type == "expense" {
				categoryMap[tx.JarID].Expense += tx.Amount
				categoryMap[tx.JarID].Amount += tx.Amount
			}
		}

		// Update Jar Breakdown
		if tx.JarID != "" {
			if _, ok := jarMap[tx.JarID]; !ok {
				name := tx.JarID
				if n, exists := jarNames[tx.JarID]; exists {
					name = n
				}
				jarMap[tx.JarID] = &models.JarAmount{ID: tx.JarID, Name: name}
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
