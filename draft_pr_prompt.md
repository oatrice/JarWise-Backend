# PR Draft Prompt

You are an AI assistant helping to create a Pull Request description.
    
TASK: [Feature] Report Filter: Multi-Select Categories & Accounts
ISSUE: {
  "title": "[Feature] Report Filter: Multi-Select Categories & Accounts",
  "number": 68,
  "body": "# \ud83c\udfaf Objective\nEnable users to filter reports and charts by selecting specific categories (Jars) and accounts (Wallets) via multi-select checkboxes.\n\n## \ud83d\udcdd Specifications\n\n### UI Components\n- [ ] **Filter Panel**: Collapsible sidebar or modal with checkbox tree\n- [ ] **Category Checkboxes**: Select/deselect individual Jars (including sub-jars if HIER-01 is done)\n- [ ] **Account Checkboxes**: Select/deselect individual Wallets (including sub-wallets if HIER-01 is done)\n- [ ] **Select All / Clear All**: Quick actions\n- [ ] **Remember Selection**: Persist filter state per session or per report type\n\n### Behavior\n- [ ] **Real-time Update**: Charts/reports update as checkboxes change (or Apply button)\n- [ ] **Count Display**: Show number of transactions matching current filter\n- [ ] **Visual Indicator**: Badge showing active filter count\n\n## \ud83d\udd17 References\n- Depends on #67 (Hierarchical Accounts & Categories) for sub-item support\n- Related to #59 (Reports & Data Export)\n- Feature ID: `REPORT-02`\n\n## \ud83c\udfd7\ufe0f Technical Notes\n- Use bitmasking or array-based filtering on transaction queries\n- Consider performance with large transaction sets (pagination/lazy load)"
}

GIT CONTEXT:
COMMITS:
c8c98e2 feat(reports): add transaction report API and improve mmbak parser reliability
30841b8 Highlight report struct tag bug
658e925 Add report endpoint and date filter
0417914 test(backend): add mmbak parser test suite and fix null date handling

STATS:
CHANGELOG.md                                    |  10 +
 README.md                                       |  10 +
 VERSION                                         |   2 +-
 code_review.md                                  |  95 +++----
 internal/api/handlers/report_handler.go         | 110 ++++++++
 internal/api/router.go                          |   3 +
 internal/models/report.go                       |  19 ++
 internal/parser/mmbak_parser.go                 |   7 +-
 internal/parser/mmbak_parser_test.go            | 349 ++++++++++++++++++++++++
 internal/parser/testdata/bad_dates.mmbak        | Bin 0 -> 28672 bytes
 internal/parser/testdata/corrupt.mmbak          |   1 +
 internal/parser/testdata/empty.mmbak            | Bin 0 -> 28672 bytes
 internal/parser/testdata/generate_test_files.py | 161 +++++++++++
 internal/parser/testdata/missing_tables.mmbak   | Bin 0 -> 12288 bytes
 internal/parser/testdata/non_existent.mmbak     |   0
 internal/parser/testdata/valid.mmbak            | Bin 0 -> 28672 bytes
 internal/repository/transaction_repository.go   |  39 +++
 internal/service/report_service.go              |  76 ++++++
 internal/service/report_service_test.go         | 220 +++++++++++++++
 19 files changed, 1034 insertions(+), 68 deletions(-)

KEY FILE DIFFS:
diff --git a/internal/api/handlers/report_handler.go b/internal/api/handlers/report_handler.go
new file mode 100644
index 0000000..d346a86
--- /dev/null
+++ b/internal/api/handlers/report_handler.go
@@ -0,0 +1,110 @@
+package handlers
+
+import (
+	"encoding/json"
+	"jarwise-backend/internal/models"
+	"jarwise-backend/internal/service"
+	"net/http"
+	"strings"
+	"time"
+)
+
+type ReportHandler struct {
+	service service.ReportService
+}
+
+func NewReportHandler(service service.ReportService) *ReportHandler {
+	return &ReportHandler{service: service}
+}
+
+func (h *ReportHandler) GetReport(w http.ResponseWriter, r *http.Request) {
+	if r.Method != http.MethodGet {
+		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
+		return
+	}
+
+	now := time.Now().UTC()
+	defaultStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
+	defaultEnd := defaultStart.AddDate(0, 1, 0).Add(-time.Nanosecond)
+
+	startDate, err := parseDateParam(r.URL.Query().Get("start_date"), defaultStart, false)
+	if err != nil {
+		http.Error(w, "Invalid start_date format. Use YYYY-MM-DD or RFC3339.", http.StatusBadRequest)
+		return
+	}
+	endDate, err := parseDateParam(r.URL.Query().Get("end_date"), defaultEnd, true)
+	if err != nil {
+		http.Error(w, "Invalid end_date format. Use YYYY-MM-DD or RFC3339.", http.StatusBadRequest)
+		return
+	}
+	if endDate.Before(startDate) {
+		http.Error(w, "end_date must be after start_date", http.StatusBadRequest)
+		return
+	}
+
+	jarIDs := parseIDsParam(r, "jar_ids", "category_ids")
+	walletIDs := parseIDsParam(r, "wallet_ids", "account_ids")
+
+	filter := models.ReportFilter{
+		StartDate: startDate,
+		EndDate:   endDate,
+		JarIDs:    jarIDs,
+		WalletIDs: walletIDs,
+	}
+
+	report, err := h.service.GenerateReport(r.Context(), filter)
+	if err != nil {
+		http.Error(w, "Failed to generate report", http.StatusInternalServerError)
+		return
+	}
+
+	w.Header().Set("Content-Type", "application/json")
+	json.NewEncoder(w).Encode(report)
+}
+
+func parseIDsParam(r *http.Request, keys ...string) []string {
+	for _, key := range keys {
+		raw := strings.TrimSpace(r.URL.Query().Get(key))
+		if raw == "" {
+			continue
+		}
+		return splitCommaSeparated(raw)
+	}
+	return []string{}
+}
+
+func splitCommaSeparated(value string) []string {
+	parts := strings.Split(value, ",")
+	results := make([]string, 0, len(parts))
+	for _, part := range parts {
+		trimmed := strings.TrimSpace(part)
+		if trimmed == "" {
+			continue
+		}
+		results = append(results, trimmed)
+	}
+	return results
+}
+
+func parseDateParam(value string, defaultValue time.Time, isEnd bool) (time.Time, error) {
+	if value == "" {
+		return defaultValue, nil
+	}
+
+	if len(value) == len("2006-01-02") {
+		parsed, err := time.ParseInLocation("2006-01-02", value, time.UTC)
+		if err != nil {
+			return time.Time{}, err
+		}
+		if isEnd {
+			return parsed.AddDate(0, 0, 1).Add(-time.Nanosecond), nil
+		}
+		return parsed, nil
+	}
+
+	parsed, err := time.Parse(time.RFC3339, value)
+	if err != nil {
+		return time.Time{}, err
+	}
+	return parsed, nil
+}
diff --git a/internal/api/router.go b/internal/api/router.go
index b49f000..630564a 100644
--- a/internal/api/router.go
+++ b/internal/api/router.go
@@ -25,6 +25,8 @@ func NewRouter() http.Handler {
 	txRepo := repository.NewSQLiteTransactionRepository(dbConn)
 	txService := service.NewTransactionService(txRepo)
 	txHandler := handlers.NewTransactionHandler(txService)
+	reportService := service.NewReportService(txRepo)
+	reportHandler := handlers.NewReportHandler(reportService)
 
 	// Routes
 	mux.HandleFunc("/api/v1/migrations/money-manager", func(w http.ResponseWriter, r *http.Request) {
@@ -36,6 +38,7 @@ func NewRouter() http.Handler {
 	})
 
 	mux.HandleFunc("/api/v1/transfers", txHandler.CreateTransfer)
+	mux.HandleFunc("/api/v1/reports", reportHandler.GetReport)
 
 	// Wallets (Mock for Manual Verification)
 	mux.HandleFunc("/api/wallets", func(w http.ResponseWriter, r *http.Request) {
diff --git a/internal/models/report.go b/internal/models/report.go
new file mode 100644
index 0000000..6c931fc
--- /dev/null
+++ b/internal/models/report.go
@@ -0,0 +1,19 @@
+package models
+
+import "time"
+
+// ReportFilter defines multi-select filters for reports.
+type ReportFilter struct {
+	StartDate time.Time `json:"start_date"`
+	EndDate   time.Time `json:"end_date"`
+	JarIDs    []string  `json:"jar_ids"`
+	WalletIDs []string  `json:"wallet_ids"`
+}
+
+// Report represents aggregated report data with the applied filter.
+type Report struct {
+	TotalAmount      float64     `json:"total_amount"`
+	TransactionCount int         `json:"transaction_count"`
+	Transactions     []Transaction `json:"transactions"`
+	FilterUsed       ReportFilter `json:"filter_used"`
+}
diff --git a/internal/parser/mmbak_parser.go b/internal/parser/mmbak_parser.go
index 095739f..a71b39d 100644
--- a/internal/parser/mmbak_parser.go
+++ b/internal/parser/mmbak_parser.go
@@ -77,7 +77,7 @@ func (p *MmbakParser) Parse(filePath string) (*models.ParsedData, error) {
 	transRows, err := db.Query(`
         SELECT uid, ZDATE, ZMONEY, DO_TYPE, ZCONTENT, categoryUid, assetUid 
         FROM INOUTCOME 
-        WHERE DO_TYPE IN ('0', '1', '2') OR DO_TYPE IS NULL
+        WHERE DO_TYPE IN ('0', '1', '2', '3') OR DO_TYPE IS NULL
     `)
 	// WARN: DO_TYPE might be varchar based on schema.
 	if err != nil {
@@ -87,12 +87,13 @@ func (p *MmbakParser) Parse(filePath string) (*models.ParsedData, error) {
 
 	for transRows.Next() {
 		var t models.TransactionDTO
-		var note, doType, catID, assetID sql.NullString
+		var note, doType, catID, assetID, dateStr sql.NullString
 		var money sql.NullFloat64
 
-		if err := transRows.Scan(&t.ID, &t.Date, &money, &doType, &note, &catID, &assetID); err != nil {
+		if err := transRows.Scan(&t.ID, &dateStr, &money, &doType, &note, &catID, &assetID); err != nil {
 			return nil, err
 		}
+		t.Date = dateStr.String
 		t.Amount = money.Float64
 		t.Note = note.String
 		t.CategoryID = catID.String
diff --git a/internal/parser/mmbak_parser_test.go b/internal/parser/mmbak_parser_test.go
new file mode 100644
index 0000000..ab5549a
--- /dev/null
+++ b/internal/parser/mmbak_parser_test.go
@@ -0,0 +1,349 @@
+package parser
+
+import (
+	"path/filepath"
+	"runtime"
+	"testing"
+)
+
+// getTestdataPath returns the absolute path to the testdata directory
+func getTestdataPath(filename string) string {
+	_, b, _, _ := runtime.Caller(0)
+	basepath := filepath.Dir(b)
+	return filepath.Join(basepath, "testdata", filename)
+}
+
+// =============================================================================
+// 🟢 Valid File Tests
+// =============================================================================
+
+func TestParse_ValidFile(t *testing.T) {
+	parser := NewMmbakParser()
+	result, err := parser.Parse(getTestdataPath("valid.mmbak"))
+
+	if err != nil {
+		t.Fatalf("Expected no error, got: %v", err)
+	}
+
+	// Verify accounts
+	if len(result.Accounts) != 2 {
+		t.Errorf("Expected 2 accounts, got %d", len(result.Accounts))
+	}
+
+	// Verify categories
+	if len(result.Categories) != 3 {
+		t.Errorf("Expected 3 categories, got %d", len(result.Categories))
+	}
+
+	// Verify transactions (2 expense + 1 income + 1 transfer)
+	if len(result.Transactions) != 4 {
+		t.Errorf("Expected 4 transactions, got %d", len(result.Transactions))
+	}
+
+	// Verify totals
+	expectedIncome := 50000.0
+	expectedExpense := 135.50 // 100.50 + 35.00
+
+	if result.TotalIncome != expectedIncome {
+		t.Errorf("Expected TotalIncome %.2f, got %.2f", expectedIncome, result.TotalIncome)
+	}
+
+	if result.TotalExpense != expectedExpense {
+		t.Errorf("Expected TotalExpense %.2f, got %.2f", expectedExpense, result.TotalExpense)
+	}
+}
+
+func TestParse_ValidFile_AccountDetails(t *testing.T) {
+	parser := NewMmbakParser()
+	result, err := parser.Parse(getTestdataPath("valid.mmbak"))
+
+	if err != nil {
+		t.Fatalf("Expected no error, got: %v", err)
+	}
+
+	// Check specific account
+	accountNames := make(map[string]bool)
+	for _, acc := range result.Accounts {
+		accountNames[acc.Name] = true
+	}
+
+	if !accountNames["Cash Wallet"] {
+		t.Error("Expected 'Cash Wallet' account to exist")
+	}
+
+	if !accountNames["Bank Account"] {
+		t.Error("Expected 'Bank Account' account to exist")
+	}
+}
+
+// =============================================================================
+// 🟥 Bad Dates Tests
+// =============================================================================
+
+func TestParse_BadDates_DoesNotCrash(t *testing.T) {
+	parser := NewMmbakParser()
+	result, err := parser.Parse(getTestdataPath("bad_dates.mmbak"))
+
+	// The parser should NOT crash on bad dates
+	// It should either return an error OR handle them gracefully
+	if err != nil {
+		t.Logf("Parser returned error for bad dates (acceptable): %v", err)
+		return
+	}
+
+	// If no error, verify we got transactions (even with bad dates)
+	if result == nil {
+		t.Fatal("Expected result to not be nil")
+	}
+
+	t.Logf("Parsed %d transactions with bad dates", len(result.Transactions))
+}
+
+func TestParse_BadDates_TransactionCount(t *testing.T) {
+	parser := NewMmbakParser()
+	result, err := parser.Parse(getTestdataPath("bad_dates.mmbak"))
+
+	if err != nil {
+		t.Skipf("Skipping: parser returned error (may be expected): %v", err)
+	}
+
+	// We inserted 10 transactions with various bad dates
+	expectedTxCount := 10
+	if len(result.Transactions) != expectedTxCount {
+		t.Errorf("Expected %d transactions, got %d", expectedTxCount, len(result.Transactions))
+	}
+}
+
+func TestParse_BadDates_NullDate(t *testing.T) {
+	parser := NewMmbakParser()
+	result, err := parser.Parse(getTestdataPath("bad_dates.mmbak"))
+
+	if err != nil {
+		t.Skipf("Skipping: parser returned error: %v", err)
+	}
+
+	// Find the null date transaction
+	for _, tx := range result.Transactions {
+		if tx.ID == "tx_null_date" {
+			// Date should be empty or handled gracefully
+			t.Logf("Null date transaction date value: '%s'", tx.Date)
+			return
+		}
+	}
+
+	t.Error("Expected to find transaction with null date (tx_null_date)")
+}
+
+func TestParse_BadDates_EmptyStringDate(t *testing.T) {
+	parser := NewMmbakParser()
+	result, err := parser.Parse(getTestdataPath("bad_dates.mmbak"))
+
+	if err != nil {
+		t.Skipf("Skipping: parser returned error: %v", err)
+	}
+
+	for _, tx := range result.Transactions {
+		if tx.ID == "tx_empty_date" {
+			t.Logf("Empty date transaction date value: '%s'", tx.Date)
+			return
+		}
+	}
+
+	t.Error("Expected to find transaction with empty date (tx_empty_date)")
+}
+
+func TestParse_BadDates_InvalidString(t *testing.T) {
+	parser := NewMmbakParser()
+	result, err := parser.Parse(getTestdataPath("bad_dates.mmbak"))
+
+	if err != nil {
+		t.Skipf("Skipping: parser returned error: %v", err)
+	}
+
+	for _, tx := range result.Transactions {
+		if tx.ID == "tx_invalid_str" {
+			// The date should be the raw invalid string or empty
+			t.Logf("Invalid string date value: '%s'", tx.Date)
+			return
+		}
+	}
+
+	t.Error("Expected to find transaction with invalid string date (tx_invalid_str)")
+}
+
+func TestParse_BadDates_WrongFormat(t *testing.T) {
+	parser := NewMmbakParser()
+	result, err := parser.Parse(getTestdataPath("bad_dates.mmbak"))
+
+	if err != nil {
+		t.Skipf("Skipping: parser returned error: %v", err)
+	}
+
+	for _, tx := range result.Transactions {
+		if tx.ID == "tx_wrong_format" {
+			// Date in DD/MM/YYYY format instead of YYYY-MM-DD
+			t.Logf("Wrong format date value: '%s'", tx.Date)
+			return
+		}
+	}
+
+	t.Error("Expected to find transaction with wrong format date (tx_wrong_format)")
+}
+
+// =============================================================================
+// 🟥 Error Handling Tests
+// =============================================================================
+
+func TestParse_MissingTables_ReturnsError(t *testing.T) {
+	parser := NewMmbakParser()
+	_, err := parser.Parse(getTestdataPath("missing_tables.mmbak"))
+
+	// Should return an error because ZCATEGORY and INOUTCOME tables are missing
+	if err == nil {
+		t.Error("Expected error for missing tables, got nil")
+	} else {
+		t.Logf("Got expected error: %v", err)
+	}
+}
+
+func TestParse_CorruptFile_ReturnsError(t *testing.T) {
+	parser := NewMmbakParser()
+	_, err := parser.Parse(getTestdataPath("corrupt.mmbak"))
+
+	// Should return an error because the file is not a valid SQLite database
+	if err == nil {
+		t.Error("Expected error for corrupt file, got nil")
+	} else {
+		t.Logf("Got expected error: %v", err)
+	}
+}
+
+func TestParse_NonExistentFile_ReturnsError(t *testing.T) {
+	parser := NewMmbakParser()
+	_, err := parser.Parse(getTestdataPath("non_existent.mmbak"))
+
+	if err == nil {
+		t.Error("Expected error for non-existent file, got nil")
+	}
+}
+
+// =============================================================================
+// 🟢 Empty File Tests
+// =============================================================================
+
+func TestParse_EmptyFile_NoTransactions(t *testing.T) {
+	parser := NewMmbakParser()
+	result, err := parser.Parse(getTestdataPath("empty.mmbak"))
+
+	if err != nil {
+		t.Fatalf("Expected no error for empty file, got: %v", err)
+	}
+
+	if len(result.Accounts) != 0 {
+		t.Errorf("Expected 0 accounts, got %d", len(result.Accounts))
+	}
+
+	if len(result.Categories) != 0 {
+		t.Errorf("Expected 0 categories, got %d", len(result.Categories))
+	}
+
+	if len(result.Transactions) != 0 {
+		t.Errorf("Expected 0 transactions, got %d", len(result.Transactions))
+	}
+
+	if result.TotalIncome != 0 {
+		t.Errorf("Expected TotalIncome 0, got %.2f", result.TotalIncome)
+	}
+
+	if result.TotalExpense != 0 {
+		t.Errorf("Expected TotalExpense 0, got %.2f", result.TotalExpense)
+	}
+}
+
+// =============================================================================
+// 🟢 Transaction Type Tests
+// =============================================================================
+
+func TestParse_TransactionTypes(t *testing.T) {
+	parser := NewMmbakParser()
+	result, err := parser.Parse(getTestdataPath("valid.mmbak"))
+
+	if err != nil {
+		t.Fatalf("Expected no error, got: %v", err)
+	}
+
+	incomeCount := 0
+	expenseCount := 0
+
+	for _, tx := range result.Transactions {
+		switch tx.Type {
+		case 0: // Expense
+			expenseCount++
+		case 1: // Income
+			incomeCount++
+		}
+	}
+
+	if incomeCount != 1 {
+		t.Errorf("Expected 1 income transaction, got %d", incomeCount)
+	}
+
+	if expenseCount != 2 {
+		t.Errorf("Expected 2 expense transactions, got %d", expenseCount)
+	}
+}
+
+// =============================================================================
+// 🟥 Transfer Transaction Tests (DO_TYPE = '3')
+// =============================================================================
+
+func TestParse_TransferTransaction_IsIncluded(t *testing.T) {
+	parser := NewMmbakParser()
+	result, err := parser.Parse(getTestdataPath("valid.mmbak"))
+
+	if err != nil {
+		t.Fatalf("Expected no error, got: %v", err)
+	}
+
+	// valid.mmbak now has 4 transactions: 2 expense, 1 income, 1 transfer
+	if len(result.Transactions) != 4 {
+		t.Errorf("Expected 4 transactions (including transfer), got %d", len(result.Transactions))
+	}
+
+	// Find the transfer transaction (tx4)
+	found := false
+	for _, tx := range result.Transactions {
+		if tx.ID == "tx4" {
+			found = true
+			if tx.Type != 2 {
+				t.Errorf("Expected transfer Type=2, got %d", tx.Type)
+			}
+			break
+		}
+	}
+
+	if !found {
+		t.Error("Expected to find transfer transaction 'tx4' but it was missing")
+	}
+}
+
+func TestParse_TransferTransaction_ExcludedFromTotals(t *testing.T) {
+	parser := NewMmbakParser()
+	result, err := parser.Parse(getTestdataPath("valid.mmbak"))
+
+	if err != nil {
+		t.Fatalf("Expected no error, got: %v", err)
+	}
+
+	// Transfer amount (5000) should NOT be in TotalIncome or TotalExpense
+	expectedIncome := 50000.0
+	expectedExpense := 135.50 // 100.50 + 35.00
+
+	if result.TotalIncome != expectedIncome {
+		t.Errorf("Expected TotalIncome %.2f (transfer excluded), got %.2f", expectedIncome, result.TotalIncome)
+	}
+
+	if result.TotalExpense != expectedExpense {
+		t.Errorf("Expected TotalExpense %.2f (transfer excluded), got %.2f", expectedExpense, result.TotalExpense)
+	}
+}
diff --git a/internal/parser/testdata/generate_test_files.py b/internal/parser/testdata/generate_test_files.py
new file mode 100644
index 0000000..a695201
--- /dev/null
+++ b/internal/parser/testdata/generate_test_files.py
@@ -0,0 +1,161 @@
+#!/usr/bin/env python3
+"""
+Script to generate sample .mmbak test files for MmbakParser tests.
+These are SQLite databases mimicking Money Manager's schema.
+"""
+import sqlite3
+import os
+
+TESTDATA_DIR = "/Users/oatrice/Software-projects/JarWise/Backend/internal/parser/testdata"
+
+def create_schema(conn):
+    """Create Money Manager-like schema."""
+    cursor = conn.cursor()
+    cursor.execute("""
+        CREATE TABLE IF NOT EXISTS ASSETS (
+            uid TEXT PRIMARY KEY,
+            NIC_NAME TEXT,
+            TYPE INTEGER
+        )
+    """)
+    cursor.execute("""
+        CREATE TABLE IF NOT EXISTS ZCATEGORY (
+            uid TEXT PRIMARY KEY,
+            NAME TEXT,
+            TYPE INTEGER
+        )
+    """)
+    cursor.execute("""
+        CREATE TABLE IF NOT EXISTS INOUTCOME (
+            uid TEXT PRIMARY KEY,
+            ZDATE TEXT,
+            ZMONEY REAL,
+            DO_TYPE TEXT,
+            ZCONTENT TEXT,
+            categoryUid TEXT,
+            assetUid TEXT
+        )
+    """)
+    conn.commit()
+
+def create_valid_mmbak():
+    """Create a valid .mmbak file with correct data."""
+    path = os.path.join(TESTDATA_DIR, "valid.mmbak")
+    conn = sqlite3.connect(path)
+    create_schema(conn)
+    cursor = conn.cursor()
+    
+    # Insert accounts
+    cursor.execute("INSERT INTO ASSETS VALUES ('acc1', 'Cash Wallet', 1)")
+    cursor.execute("INSERT INTO ASSETS VALUES ('acc2', 'Bank Account', 2)")
+    
+    # Insert categories
+    cursor.execute("INSERT INTO ZCATEGORY VALUES ('cat1', 'Food', 0)")
+    cursor.execute("INSERT INTO ZCATEGORY VALUES ('cat2', 'Salary', 1)")
+    cursor.execute("INSERT INTO ZCATEGORY VALUES ('cat3', 'Transport', 0)")
+    
+    # Insert transactions with valid dates
+    cursor.execute("INSERT INTO INOUTCOME VALUES ('tx1', '2025-01-15', 100.50, '0', 'Lunch', 'cat1', 'acc1')")
+    cursor.execute("INSERT INTO INOUTCOME VALUES ('tx2', '2025-01-20', 50000.00, '1', 'Monthly Salary', 'cat2', 'acc2')")
+    cursor.execute("INSERT INTO INOUTCOME VALUES ('tx3', '2025-01-22', 35.00, '0', 'Bus fare', 'cat3', 'acc1')")
+    # Transfer transaction (DO_TYPE = '3')
+    cursor.execute("INSERT INTO INOUTCOME VALUES ('tx4', '2025-01-25', 5000.00, '3', 'Transfer to savings', 'cat1', 'acc1')")
+    
+    conn.commit()
+    conn.close()
+    print(f"Created: {path}")
+
+def create_bad_dates_mmbak():
+    """Create a .mmbak file with various bad date formats."""
+    path = os.path.join(TESTDATA_DIR, "bad_dates.mmbak")
+    conn = sqlite3.connect(path)
+    create_schema(conn)
+    cursor = conn.cursor()
+    
+    # Insert accounts
+    cursor.execute("INSERT INTO ASSETS VALUES ('acc1', 'Wallet', 1)")
+    
+    # Insert categories
+    cursor.execute("INSERT INTO ZCATEGORY VALUES ('cat1', 'Food', 0)")
+    
+    # Insert transactions with BAD dates
+    # 1. NULL date
+    cursor.execute("INSERT INTO INOUTCOME VALUES ('tx_null_date', NULL, 100.00, '0', 'Null date tx', 'cat1', 'acc1')")
+    
+    # 2. Empty string date
+    cursor.execute("INSERT INTO INOUTCOME VALUES ('tx_empty_date', '', 200.00, '0', 'Empty date tx', 'cat1', 'acc1')")
+    
+    # 3. Invalid date string
+    cursor.execute("INSERT INTO INOUTCOME VALUES ('tx_invalid_str', 'not-a-date', 300.00, '0', 'Invalid string', 'cat1', 'acc1')")
+    
+    # 4. Invalid date format (DD/MM/YYYY instead of YYYY-MM-DD)
+    cursor.execute("INSERT INTO INOUTCOME VALUES ('tx_wrong_format', '32/1
... (Diff truncated for size) ...

PR TEMPLATE:


INSTRUCTIONS:
1. Generate a comprehensive PR description in Markdown format.
2. If a template is provided, fill it out intelligently.
3. If no template, use a standard structure: Summary, Changes, Impact.
4. Focus on 'Why' and 'What'.
5. Do not include 'Here is the PR description' preamble. Just the body.
6. IMPORTANT: Always use FULL URLs for links to issues and other PRs (e.g., https://github.com/owner/repo/issues/123), do NOT use short syntax (e.g., #123) to ensuring proper linking across platforms.
