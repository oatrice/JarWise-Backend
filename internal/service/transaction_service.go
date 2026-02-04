package service

import (
	"fmt"
	"jarwise-backend/internal/models"
	"jarwise-backend/internal/repository"
	"time"

	"github.com/google/uuid"
)

type TransactionService interface {
	CreateTransfer(fromWalletID, toWalletID string, amount float64, date time.Time, notes string) (*models.Transaction, *models.Transaction, error)
}

type transactionService struct {
	repo repository.TransactionRepository
}

func NewTransactionService(repo repository.TransactionRepository) TransactionService {
	return &transactionService{repo: repo}
}

func (s *transactionService) CreateTransfer(fromWalletID, toWalletID string, amount float64, date time.Time, notes string) (*models.Transaction, *models.Transaction, error) {
	// 1. Generate IDs
	expenseID := uuid.New().String()
	incomeID := uuid.New().String()

	// 2. Create Objects
	expense := &models.Transaction{
		ID:                   expenseID,
		Amount:               -amount, // Expense is negative
		Type:                 "expense",
		WalletID:             fromWalletID,
		Date:                 date,
		Description:          notes,
		RelatedTransactionID: &incomeID,
	}

	income := &models.Transaction{
		ID:                   incomeID,
		Amount:               amount, // Income is positive
		Type:                 "income",
		WalletID:             toWalletID,
		Date:                 date,
		Description:          notes,
		RelatedTransactionID: &expenseID,
	}

	// 3. Persist Atomic
	err := s.repo.CreateTransfer(expense, income)
	if err != nil {
		return nil, nil, fmt.Errorf("service: failed to create transfer: %w", err)
	}

	return expense, income, nil
}
