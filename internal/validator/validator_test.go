package validator

import (
	"jarwise-backend/internal/models"
	"testing"
)

func TestValidateIntegrity_AllowsTransferRowsWithoutCategoryJar(t *testing.T) {
	v := NewValidator()
	data := &models.ParsedData{
		Accounts: []models.AccountDTO{
			{ID: "wallet-1", Name: "Cash"},
			{ID: "wallet-2", Name: "Bank"},
		},
		Categories: []models.CategoryDTO{
			{ID: "jar-1", Name: "Food", Type: 0},
		},
		Transactions: []models.TransactionDTO{
			{
				ID:          "tx-transfer",
				Type:        2,
				Amount:      250,
				AccountID:   "wallet-1",
				CategoryID:  "wallet-2",
				ToAccountID: "wallet-2",
			},
		},
	}

	errors := v.ValidateIntegrity(data)
	if len(errors) != 0 {
		t.Fatalf("expected no integrity errors, got %v", errors)
	}
}

func TestValidateIntegrity_AllowsKnownSystemCategories(t *testing.T) {
	v := NewValidator()
	data := &models.ParsedData{
		Accounts: []models.AccountDTO{{ID: "wallet-1", Name: "Cash"}},
		Transactions: []models.TransactionDTO{
			{ID: "tx-other", Type: 0, Amount: 10, AccountID: "wallet-1", CategoryID: "0"},
			{ID: "tx-adjust", Type: 1, Amount: 20, AccountID: "wallet-1", CategoryID: "-4"},
		},
	}

	errors := v.ValidateIntegrity(data)
	if len(errors) != 0 {
		t.Fatalf("expected no integrity errors, got %v", errors)
	}
}
