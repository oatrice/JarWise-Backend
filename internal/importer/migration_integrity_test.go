package importer

import (
	"jarwise-backend/internal/models"
	"testing"
)

func TestImportData_IncompleteData(t *testing.T) {
	imp := NewImporter()

	// 1. Setup: สร้างข้อมูลที่ "ไม่สมบูรณ์" 
	// มี 1 Transaction แต่อ้างถึง Account ID 'acc-missing' ที่ไม่มีอยู่ในรายชื่อ Accounts
	incompleteData := &models.ParsedData{
		Accounts: []models.AccountDTO{
			{ID: "acc-1", Name: "Valid Account", Currency: "THB", Balance: 100},
		},
		Categories: []models.CategoryDTO{
			{ID: "cat-1", Name: "Food", Type: 0},
		},
		Transactions: []models.TransactionDTO{
			{ID: "tx-1", Amount: 50, Note: "Valid Tx", Date: "2026-03-10 10:00:00", AccountID: "acc-1", CategoryID: "cat-1", Type: 0},
			{ID: "tx-2", Amount: 20, Note: "Orphaned Tx", Date: "2026-03-10 11:00:00", AccountID: "acc-missing", CategoryID: "cat-1", Type: 0},
		},
	}

	// 2. Action: รันการนำเข้าข้อมูล
	// ในขั้นตอนนี้ เราคาดหวังว่าระบบจะ "ยกเลิก (Stop)" การนำเข้าทันที
	err := imp.ImportData(incompleteData)

	// 3. Assert
	if err == nil {
		t.Errorf("Expected error for incomplete data (stop on error), but got nil")
	}

	// คาดหวังว่า Error message จะระบุว่ายกเลิก
	expectedMsg := "import aborted"
	if err != nil && !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain '%s', but got: %v", expectedMsg, err)
	}
}

// Helper for tests
func contains(s, substr string) bool {
	return s != "" && (s == substr || (len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr)))
}

