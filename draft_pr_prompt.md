# PR Draft Prompt

You are an AI assistant helping to create a Pull Request description.
    
TASK: [Feature] Transaction Linking & Transfers
ISSUE: {
  "title": "[Feature] Transaction Linking & Transfers",
  "number": 71
}

GIT CONTEXT:
COMMITS:
8d8bcc4 ✨ feat(transactions): Add core transaction management and data migration
b8d96a7 ✨ feat(api): add mock wallet endpoint for manual verification
37b1958 🐛 fix(json): correct struct tags and handle date parsing errors
aa009d1 ✨ feat(transactions): add transfer functionality with wallet linking
57db634 ✨ feat(db): implement transaction repository and related models
a1a54a5 ✨ feat(migration): implement complete data migration pipeline

STATS:
.gitignore                                         |   1 +
 .luma_state.json                                   |  18 +-
 CHANGELOG.md                                       |  12 +
 README.md                                          |   1 +
 VERSION                                            |   2 +-
 code_review.md                                     | 167 ++----
 draft_pr_body.md                                   |  90 ++++
 draft_pr_prompt.md                                 | 580 +++++++++++++++++++++
 go.mod                                             |   2 +
 go.sum                                             |   2 +
 internal/api/handlers/transaction_handler.go       |  71 +++
 internal/api/router.go                             |  22 +
 internal/db/sqlite.go                              |  53 ++
 internal/importer/importer.go                      |   8 +-
 internal/models/domain.go                          |   3 +
 internal/repository/transaction_repository.go      | 145 ++++++
 internal/repository/transaction_repository_test.go | 121 +++++
 internal/service/transaction_service.go            |  57 ++
 18 files changed, 1221 insertions(+), 134 deletions(-)

KEY FILE DIFFS:
diff --git a/internal/api/handlers/transaction_handler.go b/internal/api/handlers/transaction_handler.go
new file mode 100644
index 0000000..1c9591b
--- /dev/null
+++ b/internal/api/handlers/transaction_handler.go
@@ -0,0 +1,71 @@
+package handlers
+
+import (
+	"encoding/json"
+	"jarwise-backend/internal/models"
+	"jarwise-backend/internal/service"
+	"net/http"
+	"time"
+)
+
+type TransactionHandler struct {
+	service service.TransactionService
+}
+
+func NewTransactionHandler(service service.TransactionService) *TransactionHandler {
+	return &TransactionHandler{service: service}
+}
+
+type CreateTransferRequest struct {
+	FromWalletID string  `json:"from_wallet_id"`
+	ToWalletID   string  `json:"to_wallet_id"`
+	Amount       float64 `json:"amount"`
+	Date         string  `json:"date"` // IOS8601
+	Notes        string  `json:"notes"`
+}
+
+type CreateTransferResponse struct {
+	ExpenseTransaction *models.Transaction `json:"expense_transaction"`
+	IncomeTransaction  *models.Transaction `json:"income_transaction"`
+}
+
+func (h *TransactionHandler) CreateTransfer(w http.ResponseWriter, r *http.Request) {
+	if r.Method != http.MethodPost {
+		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
+		return
+	}
+
+	var req CreateTransferRequest
+	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
+		http.Error(w, "Invalid request body", http.StatusBadRequest)
+		return
+	}
+
+	// Basic validation
+	if req.FromWalletID == "" || req.ToWalletID == "" || req.Amount <= 0 {
+		http.Error(w, "Invalid input parameters", http.StatusBadRequest)
+		return
+	}
+
+	date, err := time.Parse(time.RFC3339, req.Date)
+	if err != nil {
+		// Try short date
+		date, err = time.Parse("2006-01-02", req.Date)
+		if err != nil {
+			http.Error(w, "Invalid date format", http.StatusBadRequest)
+			return
+		}
+	}
+
+	expense, income, err := h.service.CreateTransfer(req.FromWalletID, req.ToWalletID, req.Amount, date, req.Notes)
+	if err != nil {
+		http.Error(w, err.Error(), http.StatusInternalServerError)
+		return
+	}
+
+	w.WriteHeader(http.StatusCreated)
+	json.NewEncoder(w).Encode(CreateTransferResponse{
+		ExpenseTransaction: expense,
+		IncomeTransaction:  income,
+	})
+}
diff --git a/internal/api/router.go b/internal/api/router.go
index 9c6a2fe..b49f000 100644
--- a/internal/api/router.go
+++ b/internal/api/router.go
@@ -2,6 +2,8 @@ package api
 
 import (
 	"jarwise-backend/internal/api/handlers"
+	"jarwise-backend/internal/db"
+	"jarwise-backend/internal/repository"
 	"jarwise-backend/internal/service"
 	"net/http"
 )
@@ -9,10 +11,21 @@ import (
 func NewRouter() http.Handler {
 	mux := http.NewServeMux()
 
+	// Infrastructure
+	dbConn, err := db.InitDB("transactions.db")
+	if err != nil {
+		// In a real app we might panic or handle differently
+		panic(err)
+	}
+
 	// Dependencies
 	migrationSvc := service.NewMigrationService()
 	migrationHandler := handlers.NewMigrationHandler(migrationSvc)
 
+	txRepo := repository.NewSQLiteTransactionRepository(dbConn)
+	txService := service.NewTransactionService(txRepo)
+	txHandler := handlers.NewTransactionHandler(txService)
+
 	// Routes
 	mux.HandleFunc("/api/v1/migrations/money-manager", func(w http.ResponseWriter, r *http.Request) {
 		if r.Method != http.MethodPost {
@@ -22,6 +35,15 @@ func NewRouter() http.Handler {
 		migrationHandler.HandleUpload(w, r)
 	})
 
+	mux.HandleFunc("/api/v1/transfers", txHandler.CreateTransfer)
+
+	// Wallets (Mock for Manual Verification)
+	mux.HandleFunc("/api/wallets", func(w http.ResponseWriter, r *http.Request) {
+		w.Header().Set("Content-Type", "application/json")
+		w.WriteHeader(http.StatusOK)
+		w.Write([]byte(`[{"id":"1","name":"Cash","balance":100.0},{"id":"2","name":"Bank","balance":5000.0}]`))
+	})
+
 	// Health Check
 	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
 		w.WriteHeader(http.StatusOK)
diff --git a/internal/db/sqlite.go b/internal/db/sqlite.go
new file mode 100644
index 0000000..6f9d227
--- /dev/null
+++ b/internal/db/sqlite.go
@@ -0,0 +1,53 @@
+package db
+
+import (
+	"database/sql"
+	"fmt"
+	"log"
+
+	_ "github.com/mattn/go-sqlite3"
+)
+
+// InitDB initializes the SQLite database and runs migrations
+func InitDB(dataSourceName string) (*sql.DB, error) {
+	db, err := sql.Open("sqlite3", dataSourceName)
+	if err != nil {
+		return nil, fmt.Errorf("failed to open database: %w", err)
+	}
+
+	if err := db.Ping(); err != nil {
+		return nil, fmt.Errorf("failed to ping database: %w", err)
+	}
+
+	if err := runMigrations(db); err != nil {
+		return nil, fmt.Errorf("failed to run migrations: %w", err)
+	}
+
+	return db, nil
+}
+
+func runMigrations(db *sql.DB) error {
+	// Simple schema migration
+	schema := `
+	CREATE TABLE IF NOT EXISTS transactions (
+		id TEXT PRIMARY KEY,
+		amount REAL NOT NULL,
+		description TEXT,
+		date DATETIME NOT NULL,
+		type TEXT NOT NULL,
+		wallet_id TEXT NOT NULL,
+		jar_id TEXT,
+		related_transaction_id TEXT,
+		FOREIGN KEY(related_transaction_id) REFERENCES transactions(id)
+	);
+	CREATE INDEX IF NOT EXISTS idx_related_transaction_id ON transactions(related_transaction_id);
+	`
+
+	_, err := db.Exec(schema)
+	if err != nil {
+		return fmt.Errorf("failed to create schema: %w", err)
+	}
+
+	log.Println("Database migration completed successfully.")
+	return nil
+}
diff --git a/internal/importer/importer.go b/internal/importer/importer.go
index 2e9e9d7..f266110 100644
--- a/internal/importer/importer.go
+++ b/internal/importer/importer.go
@@ -83,8 +83,12 @@ func mapTransactions(mmTrans []models.TransactionDTO) []models.Transaction {
 	for _, t := range mmTrans {
 		date, err := time.Parse(layout, t.Date)
 		if err != nil {
-			// Fallback for YYYY-MM-DD
-			date, _ = time.Parse("2006-01-02", t.Date)
+        var errFallback error
+			date, errFallback = time.Parse("2006-01-02", t.Date)
+			if errFallback != nil {
+				fmt.Printf("WARN: Could not parse date string '%s' for transaction ID %s. Skipping.\n", t.Date, t.ID)
+				continue
+			}
 		}
 
 		txType := "expense"
diff --git a/internal/models/domain.go b/internal/models/domain.go
index 79d402f..cb26209 100644
--- a/internal/models/domain.go
+++ b/internal/models/domain.go
@@ -33,4 +33,7 @@ type Transaction struct {
 
 	// For transfer
 	ToWalletID string `json:"to_wallet_id,omitempty"`
+
+	// New field for linking (e.g., Transfer, Refund)
+	RelatedTransactionID *string `json:"related_transaction_id,omitempty"`
 }
diff --git a/internal/repository/transaction_repository.go b/internal/repository/transaction_repository.go
new file mode 100644
index 0000000..791f7f6
--- /dev/null
+++ b/internal/repository/transaction_repository.go
@@ -0,0 +1,145 @@
+package repository
+
+import (
+	"database/sql"
+	"fmt"
+	"jarwise-backend/internal/models"
+)
+
+type TransactionRepository interface {
+	Create(tx *models.Transaction) error
+	CreateTransfer(expense, income *models.Transaction) error
+	GetByID(id string) (*models.Transaction, error)
+	Delete(id string) error
+	Unlink(id1, id2 string) error
+}
+
+type sqliteTransactionRepository struct {
+	db *sql.DB
+}
+
+func NewSQLiteTransactionRepository(db *sql.DB) TransactionRepository {
+	return &sqliteTransactionRepository{db: db}
+}
+
+func (r *sqliteTransactionRepository) Create(tx *models.Transaction) error {
+	query := `INSERT INTO transactions 
+		(id, amount, description, date, type, wallet_id, jar_id, related_transaction_id) 
+		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
+	_, err := r.db.Exec(query,
+		tx.ID, tx.Amount, tx.Description, tx.Date, tx.Type,
+		tx.WalletID, tx.JarID, tx.RelatedTransactionID)
+	return err
+}
+
+func (r *sqliteTransactionRepository) CreateTransfer(expense, income *models.Transaction) error {
+	tx, err := r.db.Begin()
+	if err != nil {
+		return err
+	}
+	defer tx.Rollback()
+
+	query := `INSERT INTO transactions 
+		(id, amount, description, date, type, wallet_id, jar_id, related_transaction_id) 
+		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
+
+	// Insert Expense
+	_, err = tx.Exec(query,
+		expense.ID, expense.Amount, expense.Description, expense.Date, expense.Type,
+		expense.WalletID, expense.JarID, expense.RelatedTransactionID)
+	if err != nil {
+		return fmt.Errorf("failed to insert expense: %w", err)
+	}
+
+	// Insert Income
+	_, err = tx.Exec(query,
+		income.ID, income.Amount, income.Description, income.Date, income.Type,
+		income.WalletID, income.JarID, income.RelatedTransactionID)
+	if err != nil {
+		return fmt.Errorf("failed to insert income: %w", err)
+	}
+
+	return tx.Commit()
+}
+
+func (r *sqliteTransactionRepository) GetByID(id string) (*models.Transaction, error) {
+	query := `SELECT id, amount, description, date, type, wallet_id, jar_id, related_transaction_id 
+		FROM transactions WHERE id = ?`
+
+	row := r.db.QueryRow(query, id)
+
+	var tx models.Transaction
+	var relatedID sql.NullString
+	var jarID sql.NullString
+
+	err := row.Scan(&tx.ID, &tx.Amount, &tx.Description, &tx.Date, &tx.Type,
+		&tx.WalletID, &jarID, &relatedID)
+	if err != nil {
+		if err == sql.ErrNoRows {
+			return nil, nil
+		}
+		return nil, err
+	}
+
+	if relatedID.Valid {
+		tx.RelatedTransactionID = &relatedID.String
+	}
+	if jarID.Valid {
+		tx.JarID = jarID.String
+	}
+
+	return &tx, nil
+}
+
+func (r *sqliteTransactionRepository) Delete(id string) error {
+	tx, err := r.db.Begin()
+	if err != nil {
+		return err
+	}
+	defer tx.Rollback()
+
+	// 1. Check for link
+	var relatedID sql.NullString
+	err = tx.QueryRow("SELECT related_transaction_id FROM transactions WHERE id = ?", id).Scan(&relatedID)
+	if err != nil {
+		if err == sql.ErrNoRows {
+			return nil // Already deleted?
+		}
+		return err
+	}
+
+	// 2. Unlink pair if exists
+	if relatedID.Valid {
+		_, err = tx.Exec("UPDATE transactions SET related_transaction_id = NULL WHERE id = ?", relatedID.String)
+		if err != nil {
+			return err
+		}
+	}
+
+	// 3. Delete
+	_, err = tx.Exec("DELETE FROM transactions WHERE id = ?", id)
+	if err != nil {
+		return err
+	}
+
+	return tx.Commit()
+}
+
+func (r *sqliteTransactionRepository) Unlink(id1, id2 string) error {
+	tx, err := r.db.Begin()
+	if err != nil {
+		return err
+	}
+	defer tx.Rollback()
+
+	query := "UPDATE transactions SET related_transaction_id = NULL WHERE id = ?"
+
+	if _, err := tx.Exec(query, id1); err != nil {
+		return err
+	}
+	if _, err := tx.Exec(query, id2); err != nil {
+		return err
+	}
+
+	return tx.Commit()
+}
diff --git a/internal/repository/transaction_repository_test.go b/internal/repository/transaction_repository_test.go
new file mode 100644
index 0000000..7157b15
--- /dev/null
+++ b/internal/repository/transaction_repository_test.go
@@ -0,0 +1,121 @@
+package repository
+
+import (
+	"jarwise-backend/internal/db"
+	"jarwise-backend/internal/models"
+	"testing"
+	"time"
+)
+
+func TestCreateTransfer_Atomic(t *testing.T) {
+	// 1. Setup
+	database, err := db.InitDB(":memory:")
+	if err != nil {
+		t.Fatalf("Failed to init DB: %v", err)
+	}
+	repo := NewSQLiteTransactionRepository(database)
+
+	// 2. Prepare Data
+	linkID2 := "tx2"
+	txn1 := &models.Transaction{
+		ID:                   "tx1",
+		Amount:               -100.0,
+		Type:                 "expense",
+		WalletID:             "w1",
+		Date:                 time.Now(),
+		RelatedTransactionID: &linkID2,
+		Description:          "Transfer to w2",
+	}
+
+	linkID1 := "tx1"
+	txn2 := &models.Transaction{
+		ID:                   "tx2",
+		Amount:               100.0,
+		Type:                 "income",
+		WalletID:             "w2",
+		Date:                 time.Now(),
+		RelatedTransactionID: &linkID1,
+		Description:          "Transfer from w1",
+	}
+
+	// 3. Execute
+	// 3. Execute
+	err = repo.CreateTransfer(txn1, txn2)
+
+	if err != nil {
+		t.Fatalf("CreateTransfer failed: %v", err)
+	}
+
+	// 4. Verify
+	savedTx1, err := repo.GetByID("tx1")
+	if err != nil {
+		t.Errorf("GetByID tx1 failed: %v", err)
+	}
+	if savedTx1 == nil || savedTx1.RelatedTransactionID == nil || *savedTx1.RelatedTransactionID != "tx2" {
+		t.Errorf("Tx1 not saved correctly or link missing")
+	}
+
+	savedTx2, err := repo.GetByID("tx2")
+	if err != nil {
+		t.Errorf("GetByID tx2 failed: %v", err)
+	}
+	if savedTx2 == nil || savedTx2.RelatedTransactionID == nil || *savedTx2.RelatedTransactionID != "tx1" {
+		t.Errorf("Tx2 not saved correctly or link missing")
+	}
+}
+
+func TestDeleteTransaction_UnlinksRelated(t *testing.T) {
+	// 1. Setup & Seeding
+	database, err := db.InitDB(":memory:")
+	if err != nil {
+		t.Fatalf("Failed to init DB: %v", err)
+	}
+	repo := NewSQLiteTransactionRepository(database)
+
+	linkID2 := "tx2"
+	txn1 := &models.Transaction{
+		ID:                   "tx1",
+		Amount:               -100.0,
+		Type:                 "expense",
+		WalletID:             "w1",
+		Date:                 time.Now(),
+		RelatedTransactionID: &linkID2,
+	}
+
+	linkID1 := "tx1"
+	txn2 := &models.Transaction{
+		ID:                   "tx2",
+		Amount:               100.0,
+		Type:                 "income",
+		WalletID:             "w2",
+		Date:                 time.Now(),
+		RelatedTransactionID: &linkID1,
+	}
+	_ = repo.CreateTransfer(txn1, txn2)
+
+	// 2. Execute Delete on Tx1
+	err = repo.Delete("tx1")
+	if err != nil {
+		t.Fatalf("Delete failed: %v", err)
+	}
+
+	// 3. Verify Tx1 is gone
+	deletedTx, err := repo.GetByID("tx1")
+	if err != nil {
+		t.Errorf("GetByID failed: %v", err)
+	}
+	if deletedTx != nil {
+		t.Errorf("Tx1 should be deleted")
+	}
+
+	// 4. Verify Tx2 still exists but RelatedID is NULL
+	remainingTx, err := repo.GetByID("tx2")
+	if err != nil {
+		t.Errorf("GetByID tx2 failed: %v", err)
+	}
+	if remainingTx == nil {
+		t.Errorf("Tx2 should still exist")
+	} else if remainingTx.RelatedTransactionID != nil {
+		t.Errorf("Tx2 should be unlinked (RelatedID should be nil), got: %v", *remainingTx.RelatedTransactionID)
+	}
+}
diff --git a/internal/service/transaction_service.go b/internal/service/transaction_service.go
new file mode 100644
index 0000000..4a05d98
--- /dev/null
+++ b/internal/service/transaction_service.go
@@ -0,0 +1,57 @@
+package service
+
+import (
+	"fmt"
+	"jarwise-backend/internal/models"
+	"jarwise-backend/internal/repository"
+	"time"
+
+	"github.com/google/uuid"
+)
+
+type TransactionService interface {
+	CreateTransfer(fromWalletID, toWalletID string, amount float64, date time.Time, notes string) (*models.Transaction, *models.Transaction, error)
+}
+
+type transactionService struct {
+	repo repository.TransactionRepository
+}
+
+func NewTransactionService(repo repository.TransactionRepository) TransactionService {
+	return &transactionService{repo: repo}
+}
+
+func (s *transactionService) CreateTransfer(fromWalletID, toWalletID string, amount float64, date time.Time, notes string) (*models.Transaction, *models.Transaction, error) {
+	// 1. Generate IDs
+	expenseID := uuid.New().String()
+	incomeID := uuid.New().String()
+
+	// 2. Create Objects
+	expense := &models.Transaction{
+		ID:                   expenseID,
+		Amount:               -amount, // Expense is negative
+		Type:                 "expense",
+		WalletID:             fromWalletID,
+		Date:                 date,
+		Description:          notes,
+		RelatedTransactionID: &incomeID,
+	}
+
+	income := &models.Transaction{
+		ID:                   incomeID,
+		Amount:               amount, // Income is positive
+		Type:                 "income",
+		WalletID:             toWalletID,
+		Date:                 date,
+		Description:          notes,
+		RelatedTransactionID: &expenseID,
+	}
+
+	// 3. Persist Atomic
+	err := s.repo.CreateTransfer(expense, income)
+	if err != nil {
+		return nil, nil, fmt.Errorf("service: failed to create transfer: %w", err)
+	}
+
+	return expense, income, nil
+}

PR TEMPLATE:


INSTRUCTIONS:
1. Generate a comprehensive PR description in Markdown format.
2. If a template is provided, fill it out intelligently.
3. If no template, use a standard structure: Summary, Changes, Impact.
4. Focus on 'Why' and 'What'.
5. Do not include 'Here is the PR description' preamble. Just the body.
6. IMPORTANT: Always use FULL URLs for links to issues and other PRs (e.g., https://github.com/owner/repo/issues/123), do NOT use short syntax (e.g., #123) to ensuring proper linking across platforms.
