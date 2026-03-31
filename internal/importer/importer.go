package importer

import (
	"database/sql"
	"fmt"
	"jarwise-backend/internal/models"
	"jarwise-backend/internal/validator"
	"log"
	"time"
)

type Importer struct {
	db        *sql.DB
	validator *validator.Validator
}

func NewImporter(db *sql.DB) *Importer {
	return &Importer{
		db:        db,
		validator: validator.NewValidator(),
	}
}

// ImportData converts MM data to JarWise domain models and persists them
func (i *Importer) ImportData(data *models.ParsedData) error {
	if i.db == nil {
		return fmt.Errorf("importer database is not configured")
	}

	// Validate Integrity
	validationErrors := i.validator.ValidateIntegrity(data)
	if len(validationErrors) > 0 {
		return fmt.Errorf("import aborted due to %d validation errors: %v", len(validationErrors), validationErrors)
	}

	// Prepare IDs for filtering
	walletMap := make(map[string]bool)
	for _, acc := range data.Accounts {
		walletMap[acc.ID] = true
	}
	jarMap := make(map[string]bool)
	for _, cat := range data.Categories {
		jarMap[cat.ID] = true
	}

	wallets := mapWallets(data.Accounts)
	jars := mapJars(data.Categories)

	// Filter out transactions that would fail FK constraints
	var validTxDTOs []models.TransactionDTO
	for _, tx := range data.Transactions {
		if walletMap[tx.AccountID] && (tx.CategoryID == "" || jarMap[tx.CategoryID]) {
			validTxDTOs = append(validTxDTOs, tx)
		}
	}

	transactions := mapTransactions(validTxDTOs)

	tx, err := i.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start import transaction: %w", err)
	}
	defer tx.Rollback()

	for _, wallet := range wallets {
		wallet.UserID = models.DefaultLocalUserID
		if _, err := tx.Exec(
			`INSERT INTO wallets (id, user_id, name, currency, balance, type) VALUES (?, ?, ?, ?, ?, ?)`,
			wallet.ID, wallet.UserID, wallet.Name, wallet.Currency, wallet.Balance, wallet.Type,
		); err != nil {
			return fmt.Errorf("failed to insert wallet %s: %w", wallet.ID, err)
		}
	}

	for _, jar := range jars {
		jar.UserID = models.DefaultLocalUserID
		if _, err := tx.Exec(
			`INSERT INTO jars (id, user_id, name, type, parent_id, wallet_id, icon, color) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			jar.ID, jar.UserID, jar.Name, jar.Type, nullableString(jar.ParentID), nullableString(jar.WalletID), jar.Icon, jar.Color,
		); err != nil {
			return fmt.Errorf("failed to insert jar %s: %w", jar.ID, err)
		}
	}

	for _, transaction := range transactions {
		transaction.UserID = models.DefaultLocalUserID
		if _, err := tx.Exec(
			`INSERT INTO transactions (id, user_id, amount, description, date, type, wallet_id, jar_id, related_transaction_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			transaction.ID,
			transaction.UserID,
			transaction.Amount,
			transaction.Description,
			transaction.Date.UTC(),
			transaction.Type,
			transaction.WalletID,
			nullableString(transaction.JarID),
			nullableRelatedID(transaction.RelatedTransactionID),
		); err != nil {
			return fmt.Errorf("failed to insert transaction %s: %w", transaction.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit import transaction: %w", err)
	}

	log.Printf(
		"[migration-import] committed wallets=%d jars=%d transactions=%d skipped_transactions=%d",
		len(wallets),
		len(jars),
		len(transactions),
		len(data.Transactions)-len(transactions),
	)

	if len(validationErrors) > 0 {
		return fmt.Errorf("import completed with %d validation errors: %v", len(validationErrors), validationErrors)
	}

	return nil
} // Mappers

func nullableString(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}

func nullableRelatedID(value *string) interface{} {
	if value == nil || *value == "" {
		return nil
	}
	return *value
}

func mapWallets(mmAccounts []models.AccountDTO) []models.Wallet {
	var result []models.Wallet
	for _, acc := range mmAccounts {
		result = append(result, models.Wallet{
			ID:       acc.ID, // Keep original ID for mapping logic? Or generate new UUID?
			Name:     acc.Name,
			Currency: acc.Currency,
			Balance:  acc.Balance,
			Type:     "general", // Default
		})
	}
	return result
}

func mapJars(mmCategories []models.CategoryDTO) []models.Jar {
	var result []models.Jar
	for _, cat := range mmCategories {
		t := "expense"
		if cat.Type == 1 {
			t = "income"
		}

		result = append(result, models.Jar{
			ID:       cat.ID,
			Name:     cat.Name,
			Type:     t,
			ParentID: cat.ParentID,
		})
	}
	return result
}

func mapTransactions(mmTrans []models.TransactionDTO) []models.Transaction {
	var result []models.Transaction
	layout := "2006-01-02 15:04:05" // Check MM date format!
	// MM format might be just YYYY-MM-DD or float timestamp?
	// In parser we scanned it as string. Let's assume standard SQL string for now.
	// Parser output usually: 'YYYY-MM-DD HH:MM:SS' or similar.

	// Heuristic for format:
	// If MM stores as timestamp (REAL/INTEGER), we need to handle that in parser.
	// In mmbak_parser.go we scanned ZDATE into string. SQLite often stores as YYYY-MM-DD HH:MM:SS

	for _, t := range mmTrans {
		date, err := time.Parse(layout, t.Date)
		if err != nil {
			var errFallback error
			date, errFallback = time.Parse("2006-01-02", t.Date)
			if errFallback != nil {
				fmt.Printf("WARN: Could not parse date string '%s' for transaction ID %s. Skipping.\n", t.Date, t.ID)
				continue
			}
		}

		txType := "expense"
		if t.Type == 1 {
			txType = "income"
		} else if t.Type == 2 { // Assuming 2 is transfer
			txType = "transfer"
		}

		result = append(result, models.Transaction{
			ID:          t.ID,
			Amount:      t.Amount,
			Description: t.Note,
			Date:        date,
			Type:        txType,
			WalletID:    t.AccountID,
			JarID:       t.CategoryID,
			ToWalletID:  t.ToAccountID,
		})
	}
	return result
}
