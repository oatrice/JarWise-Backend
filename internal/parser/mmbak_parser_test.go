package parser

import (
	"path/filepath"
	"runtime"
	"testing"
)

// getTestdataPath returns the absolute path to the testdata directory
func getTestdataPath(filename string) string {
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	return filepath.Join(basepath, "testdata", filename)
}

// =============================================================================
// 🟢 Valid File Tests
// =============================================================================

func TestParse_ValidFile(t *testing.T) {
	parser := NewMmbakParser()
	result, err := parser.Parse(getTestdataPath("valid.mmbak"))

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify accounts
	if len(result.Accounts) != 2 {
		t.Errorf("Expected 2 accounts, got %d", len(result.Accounts))
	}

	// Verify categories
	if len(result.Categories) != 3 {
		t.Errorf("Expected 3 categories, got %d", len(result.Categories))
	}

	// Verify transactions (2 expense + 1 income + 1 transfer)
	if len(result.Transactions) != 4 {
		t.Errorf("Expected 4 transactions, got %d", len(result.Transactions))
	}

	// Verify totals
	expectedIncome := 50000.0
	expectedExpense := 135.50 // 100.50 + 35.00

	if result.TotalIncome != expectedIncome {
		t.Errorf("Expected TotalIncome %.2f, got %.2f", expectedIncome, result.TotalIncome)
	}

	if result.TotalExpense != expectedExpense {
		t.Errorf("Expected TotalExpense %.2f, got %.2f", expectedExpense, result.TotalExpense)
	}
}

func TestParse_ValidFile_AccountDetails(t *testing.T) {
	parser := NewMmbakParser()
	result, err := parser.Parse(getTestdataPath("valid.mmbak"))

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check specific account
	accountNames := make(map[string]bool)
	for _, acc := range result.Accounts {
		accountNames[acc.Name] = true
	}

	if !accountNames["Cash Wallet"] {
		t.Error("Expected 'Cash Wallet' account to exist")
	}

	if !accountNames["Bank Account"] {
		t.Error("Expected 'Bank Account' account to exist")
	}
}

// =============================================================================
// 🟥 Bad Dates Tests
// =============================================================================

func TestParse_BadDates_DoesNotCrash(t *testing.T) {
	parser := NewMmbakParser()
	result, err := parser.Parse(getTestdataPath("bad_dates.mmbak"))

	// The parser should NOT crash on bad dates
	// It should either return an error OR handle them gracefully
	if err != nil {
		t.Logf("Parser returned error for bad dates (acceptable): %v", err)
		return
	}

	// If no error, verify we got transactions (even with bad dates)
	if result == nil {
		t.Fatal("Expected result to not be nil")
	}

	t.Logf("Parsed %d transactions with bad dates", len(result.Transactions))
}

func TestParse_BadDates_TransactionCount(t *testing.T) {
	parser := NewMmbakParser()
	result, err := parser.Parse(getTestdataPath("bad_dates.mmbak"))

	if err != nil {
		t.Skipf("Skipping: parser returned error (may be expected): %v", err)
	}

	// We inserted 10 transactions with various bad dates
	expectedTxCount := 10
	if len(result.Transactions) != expectedTxCount {
		t.Errorf("Expected %d transactions, got %d", expectedTxCount, len(result.Transactions))
	}
}

func TestParse_BadDates_NullDate(t *testing.T) {
	parser := NewMmbakParser()
	result, err := parser.Parse(getTestdataPath("bad_dates.mmbak"))

	if err != nil {
		t.Skipf("Skipping: parser returned error: %v", err)
	}

	// Find the null date transaction
	for _, tx := range result.Transactions {
		if tx.ID == "tx_null_date" {
			// Date should be empty or handled gracefully
			t.Logf("Null date transaction date value: '%s'", tx.Date)
			return
		}
	}

	t.Error("Expected to find transaction with null date (tx_null_date)")
}

func TestParse_BadDates_EmptyStringDate(t *testing.T) {
	parser := NewMmbakParser()
	result, err := parser.Parse(getTestdataPath("bad_dates.mmbak"))

	if err != nil {
		t.Skipf("Skipping: parser returned error: %v", err)
	}

	for _, tx := range result.Transactions {
		if tx.ID == "tx_empty_date" {
			t.Logf("Empty date transaction date value: '%s'", tx.Date)
			return
		}
	}

	t.Error("Expected to find transaction with empty date (tx_empty_date)")
}

func TestParse_BadDates_InvalidString(t *testing.T) {
	parser := NewMmbakParser()
	result, err := parser.Parse(getTestdataPath("bad_dates.mmbak"))

	if err != nil {
		t.Skipf("Skipping: parser returned error: %v", err)
	}

	for _, tx := range result.Transactions {
		if tx.ID == "tx_invalid_str" {
			// The date should be the raw invalid string or empty
			t.Logf("Invalid string date value: '%s'", tx.Date)
			return
		}
	}

	t.Error("Expected to find transaction with invalid string date (tx_invalid_str)")
}

func TestParse_BadDates_WrongFormat(t *testing.T) {
	parser := NewMmbakParser()
	result, err := parser.Parse(getTestdataPath("bad_dates.mmbak"))

	if err != nil {
		t.Skipf("Skipping: parser returned error: %v", err)
	}

	for _, tx := range result.Transactions {
		if tx.ID == "tx_wrong_format" {
			// Date in DD/MM/YYYY format instead of YYYY-MM-DD
			t.Logf("Wrong format date value: '%s'", tx.Date)
			return
		}
	}

	t.Error("Expected to find transaction with wrong format date (tx_wrong_format)")
}

// =============================================================================
// 🟥 Error Handling Tests
// =============================================================================

func TestParse_MissingTables_ReturnsError(t *testing.T) {
	parser := NewMmbakParser()
	_, err := parser.Parse(getTestdataPath("missing_tables.mmbak"))

	// Should return an error because ZCATEGORY and INOUTCOME tables are missing
	if err == nil {
		t.Error("Expected error for missing tables, got nil")
	} else {
		t.Logf("Got expected error: %v", err)
	}
}

func TestParse_CorruptFile_ReturnsError(t *testing.T) {
	parser := NewMmbakParser()
	_, err := parser.Parse(getTestdataPath("corrupt.mmbak"))

	// Should return an error because the file is not a valid SQLite database
	if err == nil {
		t.Error("Expected error for corrupt file, got nil")
	} else {
		t.Logf("Got expected error: %v", err)
	}
}

func TestParse_NonExistentFile_ReturnsError(t *testing.T) {
	parser := NewMmbakParser()
	_, err := parser.Parse(getTestdataPath("non_existent.mmbak"))

	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

// =============================================================================
// 🟢 Empty File Tests
// =============================================================================

func TestParse_EmptyFile_NoTransactions(t *testing.T) {
	parser := NewMmbakParser()
	result, err := parser.Parse(getTestdataPath("empty.mmbak"))

	if err != nil {
		t.Fatalf("Expected no error for empty file, got: %v", err)
	}

	if len(result.Accounts) != 0 {
		t.Errorf("Expected 0 accounts, got %d", len(result.Accounts))
	}

	if len(result.Categories) != 0 {
		t.Errorf("Expected 0 categories, got %d", len(result.Categories))
	}

	if len(result.Transactions) != 0 {
		t.Errorf("Expected 0 transactions, got %d", len(result.Transactions))
	}

	if result.TotalIncome != 0 {
		t.Errorf("Expected TotalIncome 0, got %.2f", result.TotalIncome)
	}

	if result.TotalExpense != 0 {
		t.Errorf("Expected TotalExpense 0, got %.2f", result.TotalExpense)
	}
}

// =============================================================================
// 🟢 Transaction Type Tests
// =============================================================================

func TestParse_TransactionTypes(t *testing.T) {
	parser := NewMmbakParser()
	result, err := parser.Parse(getTestdataPath("valid.mmbak"))

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	incomeCount := 0
	expenseCount := 0

	for _, tx := range result.Transactions {
		switch tx.Type {
		case 0: // Expense
			expenseCount++
		case 1: // Income
			incomeCount++
		}
	}

	if incomeCount != 1 {
		t.Errorf("Expected 1 income transaction, got %d", incomeCount)
	}

	if expenseCount != 2 {
		t.Errorf("Expected 2 expense transactions, got %d", expenseCount)
	}
}

// =============================================================================
// 🟥 Transfer Transaction Tests (DO_TYPE = '3')
// =============================================================================

func TestParse_TransferTransaction_IsIncluded(t *testing.T) {
	parser := NewMmbakParser()
	result, err := parser.Parse(getTestdataPath("valid.mmbak"))

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// valid.mmbak now has 4 transactions: 2 expense, 1 income, 1 transfer
	if len(result.Transactions) != 4 {
		t.Errorf("Expected 4 transactions (including transfer), got %d", len(result.Transactions))
	}

	// Find the transfer transaction (tx4)
	found := false
	for _, tx := range result.Transactions {
		if tx.ID == "tx4" {
			found = true
			if tx.Type != 2 {
				t.Errorf("Expected transfer Type=2, got %d", tx.Type)
			}
			break
		}
	}

	if !found {
		t.Error("Expected to find transfer transaction 'tx4' but it was missing")
	}
}

func TestParse_TransferTransaction_ExcludedFromTotals(t *testing.T) {
	parser := NewMmbakParser()
	result, err := parser.Parse(getTestdataPath("valid.mmbak"))

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Transfer amount (5000) should NOT be in TotalIncome or TotalExpense
	expectedIncome := 50000.0
	expectedExpense := 135.50 // 100.50 + 35.00

	if result.TotalIncome != expectedIncome {
		t.Errorf("Expected TotalIncome %.2f (transfer excluded), got %.2f", expectedIncome, result.TotalIncome)
	}

	if result.TotalExpense != expectedExpense {
		t.Errorf("Expected TotalExpense %.2f (transfer excluded), got %.2f", expectedExpense, result.TotalExpense)
	}
}
