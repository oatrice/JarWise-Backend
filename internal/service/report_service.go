package service

import (
	"context"
	"jarwise-backend/internal/models"
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
	transactions, err := s.repo.ListByDateRange(filter.StartDate, filter.EndDate)
	if err != nil {
		return nil, err
	}

	filtered := applyReportFilters(transactions, filter)

	var total float64
	for _, tx := range filtered {
		total += tx.Amount
	}

	return &models.Report{
		TotalAmount:      total,
		TransactionCount: len(filtered),
		Transactions:     filtered,
		FilterUsed:       filter,
	}, nil
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
