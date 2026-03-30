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
	CreateTransferForUser(userID, fromWalletID, toWalletID string, amount float64, date time.Time, notes string) (*models.Transaction, *models.Transaction, error)
}

type transactionService struct {
	repo       repository.TransactionRepository
	walletRepo repository.WalletRepository
}

func NewTransactionService(repo repository.TransactionRepository, walletRepo repository.WalletRepository) TransactionService {
	return &transactionService{
		repo:       repo,
		walletRepo: walletRepo,
	}
}

func (s *transactionService) CreateTransfer(fromWalletID, toWalletID string, amount float64, date time.Time, notes string) (*models.Transaction, *models.Transaction, error) {
	return s.CreateTransferForUser("", fromWalletID, toWalletID, amount, date, notes)
}

func (s *transactionService) CreateTransferForUser(userID, fromWalletID, toWalletID string, amount float64, date time.Time, notes string) (*models.Transaction, *models.Transaction, error) {
	// 0. Verify Wallets exist (Handling Sync Conflicts)
	var (
		fromWallet *models.Wallet
		toWallet   *models.Wallet
		err        error
	)

	if userID != "" {
		fromWallet, err = s.walletRepo.GetForUser(userID, fromWalletID)
	} else {
		fromWallet, err = s.walletRepo.Get(fromWalletID)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("service: failed to check source wallet: %w", err)
	}
	if fromWallet == nil {
		return nil, nil, fmt.Errorf("source wallet %s does not exist", fromWalletID)
	}

	if userID != "" {
		toWallet, err = s.walletRepo.GetForUser(userID, toWalletID)
	} else {
		toWallet, err = s.walletRepo.Get(toWalletID)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("service: failed to check target wallet: %w", err)
	}
	if toWallet == nil {
		return nil, nil, fmt.Errorf("target wallet %s does not exist (it might have been deleted on another device)", toWalletID)
	}

	// 1. Generate IDs
	expenseID := uuid.New().String()
	incomeID := uuid.New().String()

	// 2. Create Objects
	expense := &models.Transaction{
		ID:                   expenseID,
		UserID:               userID,
		Amount:               -amount, // Expense is negative
		Type:                 "expense",
		WalletID:             fromWalletID,
		Date:                 date,
		Description:          notes,
		RelatedTransactionID: &incomeID,
	}

	income := &models.Transaction{
		ID:                   incomeID,
		UserID:               userID,
		Amount:               amount, // Income is positive
		Type:                 "income",
		WalletID:             toWalletID,
		Date:                 date,
		Description:          notes,
		RelatedTransactionID: &expenseID,
	}

	// 3. Persist Atomic
	err = s.repo.CreateTransfer(expense, income)
	if err != nil {
		return nil, nil, fmt.Errorf("service: failed to create transfer: %w", err)
	}

	return expense, income, nil
}
