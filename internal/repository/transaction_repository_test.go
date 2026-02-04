package repository

import (
	"jarwise-backend/internal/db"
	"jarwise-backend/internal/models"
	"testing"
	"time"
)

func TestCreateTransfer_Atomic(t *testing.T) {
	// 1. Setup
	database, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	repo := NewSQLiteTransactionRepository(database)

	// 2. Prepare Data
	linkID2 := "tx2"
	txn1 := &models.Transaction{
		ID:                   "tx1",
		Amount:               -100.0,
		Type:                 "expense",
		WalletID:             "w1",
		Date:                 time.Now(),
		RelatedTransactionID: &linkID2,
		Description:          "Transfer to w2",
	}

	linkID1 := "tx1"
	txn2 := &models.Transaction{
		ID:                   "tx2",
		Amount:               100.0,
		Type:                 "income",
		WalletID:             "w2",
		Date:                 time.Now(),
		RelatedTransactionID: &linkID1,
		Description:          "Transfer from w1",
	}

	// 3. Execute
	// 3. Execute
	err = repo.CreateTransfer(txn1, txn2)

	if err != nil {
		t.Fatalf("CreateTransfer failed: %v", err)
	}

	// 4. Verify
	savedTx1, err := repo.GetByID("tx1")
	if err != nil {
		t.Errorf("GetByID tx1 failed: %v", err)
	}
	if savedTx1 == nil || savedTx1.RelatedTransactionID == nil || *savedTx1.RelatedTransactionID != "tx2" {
		t.Errorf("Tx1 not saved correctly or link missing")
	}

	savedTx2, err := repo.GetByID("tx2")
	if err != nil {
		t.Errorf("GetByID tx2 failed: %v", err)
	}
	if savedTx2 == nil || savedTx2.RelatedTransactionID == nil || *savedTx2.RelatedTransactionID != "tx1" {
		t.Errorf("Tx2 not saved correctly or link missing")
	}
}
