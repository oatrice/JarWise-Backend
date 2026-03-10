package validator

import (
	"fmt"
	"jarwise-backend/internal/models"
	"math"
)

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

// ValidateIntegrity checks internal consistency of parsed data (e.g. FK references)
func (v *Validator) ValidateIntegrity(data *models.ParsedData) []string {
	var errors []string

	walletIDs := make(map[string]bool)
	for _, acc := range data.Accounts {
		walletIDs[acc.ID] = true
	}

	jarIDs := make(map[string]bool)
	for _, cat := range data.Categories {
		jarIDs[cat.ID] = true
	}

	for _, tx := range data.Transactions {
		if !walletIDs[tx.AccountID] {
			errors = append(errors, fmt.Sprintf("Tx %s (%.2f) references unknown Account %s", tx.ID, tx.Amount, tx.AccountID))
		}
		if tx.CategoryID != "" && !jarIDs[tx.CategoryID] {
			errors = append(errors, fmt.Sprintf("Tx %s (%.2f) references unknown Category %s", tx.ID, tx.Amount, tx.CategoryID))
		}
		// Check transfer target
		if tx.ToAccountID != "" && !walletIDs[tx.ToAccountID] {
			errors = append(errors, fmt.Sprintf("Transfer %s (%.2f) references unknown ToAccount %s", tx.ID, tx.Amount, tx.ToAccountID))
		}
	}

	return errors
}

// Validate compares parsed data from both sources
func (v *Validator) Validate(dbData, xlsData *models.ParsedData) *ValidationResult {
	result := &ValidationResult{
		IsValid:  true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// 1. Calculate Stats
	result.DBStats = calculateStats(dbData)
	result.XLSStats = calculateStats(xlsData)

	// 2. Compare Transaction Counts
	// Allow small discrepancy? Or must be exact?
	// Given user data showed 9475 vs 10414 (diff ~1000), this is significant.
	// Likely Transfers are missing in DB query or included in XLS.
	diffCount := result.DBStats.Transactions - result.XLSStats.Transactions
	if diffCount != 0 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Transaction count mismatch: DB=%d, XLS=%d (Diff: %d)",
			result.DBStats.Transactions, result.XLSStats.Transactions, diffCount))

		// If diff is huge, maybe error?
		if math.Abs(float64(diffCount)) > 100 {
			result.IsValid = false
			result.Errors = append(result.Errors, "Significant transaction count mismatch. Check if Transfer handling differs.")
		}
	}

	// 3. Compare Totals (Income)
	// Using epsilon for float comparison
	epsilon := 0.01
	if math.Abs(result.DBStats.TotalIncome-result.XLSStats.TotalIncome) > epsilon {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Total Income mismatch: DB=%.2f, XLS=%.2f",
			result.DBStats.TotalIncome, result.XLSStats.TotalIncome))
	}

	// 4. Compare Totals (Expense)
	if math.Abs(result.DBStats.TotalExpense-result.XLSStats.TotalExpense) > epsilon {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Total Expense mismatch: DB=%.2f, XLS=%.2f",
			result.DBStats.TotalExpense, result.XLSStats.TotalExpense))
	}

	result.DiffBalance = (result.DBStats.TotalIncome - result.DBStats.TotalExpense) -
		(result.XLSStats.TotalIncome - result.XLSStats.TotalExpense)

	return result
}

func calculateStats(data *models.ParsedData) models.MigrationStats {
	return models.MigrationStats{
		Wallets:      len(data.Accounts),
		Jars:         len(data.Categories),
		Transactions: len(data.Transactions),
		TotalIncome:  data.TotalIncome,
		TotalExpense: data.TotalExpense,
	}
}
