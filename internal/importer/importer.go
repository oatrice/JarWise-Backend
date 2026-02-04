package importer

import (
	"fmt"
	"jarwise-backend/internal/models"
	"time"
)

type Importer struct {
	// Add DB repository here
}

func NewImporter() *Importer {
	return &Importer{}
}

// ImportData converts MM data to JarWise domain models and persists them
func (i *Importer) ImportData(data *models.ParsedData) error {
	wallets := mapWallets(data.Accounts)
	jars := mapJars(data.Categories)
	transactions := mapTransactions(data.Transactions)

	// Mock Persistence
	fmt.Printf("--- Importing Data to JarWise DB ---\n")
	fmt.Printf("Saved %d Wallets\n", len(wallets))
	fmt.Printf("Saved %d Jars (Categories)\n", len(jars))
	fmt.Printf("Saved %d Transactions\n", len(transactions))

	// Print sample for verification
	if len(transactions) > 0 {
		t := transactions[0]
		fmt.Printf("Sample Tx: %s | %s | %.2f | %s\n", t.Date.Format("2006-01-02"), t.Description, t.Amount, t.Type)
	}

	return nil
}

// Mappers

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
			// Fallback for YYYY-MM-DD
			date, _ = time.Parse("2006-01-02", t.Date)
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
