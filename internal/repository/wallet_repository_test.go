package repository

import (
	"database/sql"
	"jarwise-backend/internal/db"
	"jarwise-backend/internal/models"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	// Use in-memory DB for tests
	dbConn, err := db.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	// Enable Foreign Keys for SQLite
	_, err = dbConn.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}
	return dbConn
}

func TestWalletDeletionIntegrity(t *testing.T) {
	dbConn := setupTestDB(t)
	defer dbConn.Close()

	repo := NewSQLiteWalletRepository(dbConn)
	txRepo := NewSQLiteTransactionRepository(dbConn)

	// 1. Setup: Create Wallet A and a Jar and a Transaction
	wA := &models.Wallet{ID: "wallet-a", Name: "Wallet A", Currency: "THB", Balance: 100}
	if err := repo.Create(wA); err != nil {
		t.Fatalf("Failed to create wallet A: %v", err)
	}

	// Create a jar to avoid FK error
	_, err := dbConn.Exec("INSERT INTO jars (id, name, type) VALUES (?, ?, ?)", "jar-1", "Food", "expense")
	if err != nil {
		t.Fatalf("Failed to create jar: %v", err)
	}

	tx := &models.Transaction{
		ID:          "tx-1",
		Amount:      50,
		Description: "Test Tx",
		Date:        time.Now(),
		Type:        "expense",
		WalletID:    "wallet-a",
		JarID:       "jar-1",
	}
	if err := txRepo.Create(tx); err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}

	// 2. Action: Attempt to delete Wallet A (Initial implementation)
	// This should FAIL if we want to protect integrity via Foreign Keys
	err = repo.Delete("wallet-a")

	// 3. Assert: Deletion should be prevented if transactions exist
	if err == nil {
		t.Errorf("Expected error when deleting wallet with transactions, but got nil")
	}
}

func TestDeleteWithReplacement(t *testing.T) {
	dbConn := setupTestDB(t)
	defer dbConn.Close()

	repo := NewSQLiteWalletRepository(dbConn)
	txRepo := NewSQLiteTransactionRepository(dbConn)

	// 1. Setup: Create Wallet A, Wallet B, a Jar for A, and a Transaction for A
	wA := &models.Wallet{ID: "wallet-a", Name: "Wallet A", Currency: "THB"}
	wB := &models.Wallet{ID: "wallet-b", Name: "Wallet B", Currency: "THB"}
	if err := repo.Create(wA); err != nil {
		t.Fatalf("Failed to create wallet A: %v", err)
	}
	if err := repo.Create(wB); err != nil {
		t.Fatalf("Failed to create wallet B: %v", err)
	}

	// Create a jar and link it to Wallet A
	_, err := dbConn.Exec("INSERT INTO jars (id, name, type, wallet_id) VALUES (?, ?, ?, ?)", "jar-1", "Food", "expense", "wallet-a")
	if err != nil {
		t.Fatalf("Failed to create jar: %v", err)
	}

	tx := &models.Transaction{
		ID: "tx-a", Amount: 10, WalletID: "wallet-a", JarID: "jar-1", Date: time.Now(), Type: "expense",
	}
	if err := txRepo.Create(tx); err != nil {
		t.Fatalf("Failed to create transaction: %v", err)
	}

	// 2. Action: Delete A and replace with B
	err = repo.DeleteWithReplacement("wallet-a", "wallet-b")
	
	// 3. Assert
	if err != nil {
		t.Fatalf("DeleteWithReplacement failed: %v", err)
	}

	// Verify transaction is now linked to Wallet B
	updatedTx, err := txRepo.GetByID("tx-a")
	if err != nil {
		t.Fatalf("Failed to get transaction: %v", err)
	}
	if updatedTx.WalletID != "wallet-b" {
		t.Errorf("Expected transaction to be moved to wallet-b, but got %s", updatedTx.WalletID)
	}

	// Verify jar is now linked to Wallet B
	var updatedWalletID string
	err = dbConn.QueryRow("SELECT wallet_id FROM jars WHERE id = ?", "jar-1").Scan(&updatedWalletID)
	if err != nil {
		t.Fatalf("Failed to get jar: %v", err)
	}
	if updatedWalletID != "wallet-b" {
		t.Errorf("Expected jar to be moved to wallet-b, but got %s", updatedWalletID)
	}
}

func TestDeleteCascade(t *testing.T) {
	dbConn := setupTestDB(t)
	defer dbConn.Close()

	repo := NewSQLiteWalletRepository(dbConn)
	txRepo := NewSQLiteTransactionRepository(dbConn)

	// 1. Setup: Create Wallet A, a Jar for A, and a Transaction for A
	wA := &models.Wallet{ID: "wallet-a", Name: "Wallet A", Currency: "THB"}
	repo.Create(wA)

	// Create a jar and link it to Wallet A
	_, err := dbConn.Exec("INSERT INTO jars (id, name, type, wallet_id) VALUES (?, ?, ?, ?)", "jar-1", "Food", "expense", "wallet-a")
	if err != nil {
		t.Fatalf("Failed to create jar: %v", err)
	}

	tx := &models.Transaction{
		ID: "tx-a", Amount: 10, WalletID: "wallet-a", JarID: "jar-1", Date: time.Now(), Type: "expense",
	}
	txRepo.Create(tx)

	// 2. Action: Delete A with CASCADE option
	err = repo.DeleteCascade("wallet-a")
	
	// 3. Assert (Expected to fail in RED phase)
	if err != nil {
		t.Fatalf("DeleteCascade failed: %v", err)
	}

	// Verify wallet is deleted
	w, _ := repo.Get("wallet-a")
	if w != nil {
		t.Errorf("Expected wallet-a to be deleted, but it still exists")
	}

	// Verify jar is deleted
	var jarCount int
	dbConn.QueryRow("SELECT COUNT(*) FROM jars WHERE id = ?", "jar-1").Scan(&jarCount)
	if jarCount != 0 {
		t.Errorf("Expected jar-1 to be deleted via cascade, but it still exists")
	}

	// Verify transaction is deleted
	var txCount int
	dbConn.QueryRow("SELECT COUNT(*) FROM transactions WHERE id = ?", "tx-a").Scan(&txCount)
	if txCount != 0 {
		t.Errorf("Expected tx-a to be deleted via cascade, but it still exists")
	}
}
