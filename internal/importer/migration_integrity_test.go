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
	// ในขั้นตอนนี้ เราคาดหวังว่าระบบควรจะทำงานต่อได้โดยข้ามรายการที่ผิดพลาด 
	// หรือรายงานข้อผิดพลาดออกมา แทนที่จะปล่อยให้ Transaction ที่ไม่มี Wallet หลุดเข้าไปใน DB
	err := imp.ImportData(incompleteData)

	// 3. Assert: สำหรับ TDD Red เราจะเขียนให้มันตรวจจับว่า 
	// หากเรายังไม่แก้ไขโค้ด โค้ดควรจะคืนค่าความเสี่ยงหรือ Error ออกมา
	if err == nil {
		t.Errorf("Expected error or warning for incomplete data, but got nil")
	}
}
