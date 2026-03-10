package service

import (
	"jarwise-backend/internal/db"
	"jarwise-backend/internal/models"
	"jarwise-backend/internal/repository"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestSyncConflict_TransferToDeletedWallet(t *testing.T) {
	// 1. Setup DB and Repos
	dbConn, _ := db.InitDB(":memory:")
	defer dbConn.Close()
	dbConn.Exec("PRAGMA foreign_keys = ON")

	txRepo := repository.NewSQLiteTransactionRepository(dbConn)
	walletRepo := repository.NewSQLiteWalletRepository(dbConn)
	svc := NewTransactionService(txRepo, walletRepo)

	// Create Wallets
	wA := &models.Wallet{ID: "wallet-a", Name: "Wallet A", Currency: "THB"}
	wB := &models.Wallet{ID: "wallet-b", Name: "Wallet B", Currency: "THB"}
	walletRepo.Create(wA)
	walletRepo.Create(wB)

	// 2. Simulation: Wallet B is DELETED on another device (Mocked here)
	walletRepo.Delete("wallet-b")

	// 3. Action: Device 1 tries to Transfer to B
	// This should fail because Wallet B no longer exists!
	_, _, err := svc.CreateTransfer("wallet-a", "wallet-b", 100, time.Now(), "Conflict Test")

	// 4. Assert: Expecting an error due to missing target wallet (Sync Conflict)
	if err == nil {
		t.Errorf("Expected error when transferring to a deleted wallet, but got nil")
	}
}
