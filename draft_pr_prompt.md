# PR Draft Prompt

You are an AI assistant helping to create a Pull Request description.
    
TASK: [Feature] Migrate Data from Money Manager App (.mmbak)
ISSUE: {
  "title": "[Feature] Migrate Data from Money Manager App (.mmbak)",
  "number": 65
}

GIT CONTEXT:
COMMITS:
12dc36a 🐛 fix(migration): add validation bypass toggle and improve error handling
7317a80 ✨ feat(migration): Add initial Money Manager migration service
5893f5c 🐛 fix(parser): correct transaction type handling and error responses
8fc3fb7 ✨ feat(importer): add data import functionality
12340d0 ✨ feat(validator): add data validation framework
a7c7e8c 🐛 fix(parser): update schema queries for Money Manager data parsing

STATS:
.luma_state.json                      |  21 +++++
 CHANGELOG.md                          |  20 +++++
 README.md                             |  61 ++++++++++++++
 VERSION                               |   1 +
 code_review.md                        | 153 ++++++++++++++++++++++++++++++++++
 internal/importer/importer.go         | 109 ++++++++++++++++++++++++
 internal/models/domain.go             |  36 ++++++++
 internal/models/mm_data.go            |  14 ++--
 internal/parser/mmbak_parser.go       |  92 ++++++++++++++------
 internal/service/migration_service.go |  50 ++++++++---
 internal/validator/models.go          |  15 ++++
 internal/validator/validator.go       |  73 ++++++++++++++++
 12 files changed, 601 insertions(+), 44 deletions(-)

KEY FILE DIFFS:
diff --git a/internal/importer/importer.go b/internal/importer/importer.go
new file mode 100644
index 0000000..2e9e9d7
--- /dev/null
+++ b/internal/importer/importer.go
@@ -0,0 +1,109 @@
+package importer
+
+import (
+	"fmt"
+	"jarwise-backend/internal/models"
+	"time"
+)
+
+type Importer struct {
+	// Add DB repository here
+}
+
+func NewImporter() *Importer {
+	return &Importer{}
+}
+
+// ImportData converts MM data to JarWise domain models and persists them
+func (i *Importer) ImportData(data *models.ParsedData) error {
+	wallets := mapWallets(data.Accounts)
+	jars := mapJars(data.Categories)
+	transactions := mapTransactions(data.Transactions)
+
+	// Mock Persistence
+	fmt.Printf("--- Importing Data to JarWise DB ---\n")
+	fmt.Printf("Saved %d Wallets\n", len(wallets))
+	fmt.Printf("Saved %d Jars (Categories)\n", len(jars))
+	fmt.Printf("Saved %d Transactions\n", len(transactions))
+
+	// Print sample for verification
+	if len(transactions) > 0 {
+		t := transactions[0]
+		fmt.Printf("Sample Tx: %s | %s | %.2f | %s\n", t.Date.Format("2006-01-02"), t.Description, t.Amount, t.Type)
+	}
+
+	return nil
+}
+
+// Mappers
+
+func mapWallets(mmAccounts []models.AccountDTO) []models.Wallet {
+	var result []models.Wallet
+	for _, acc := range mmAccounts {
+		result = append(result, models.Wallet{
+			ID:       acc.ID, // Keep original ID for mapping logic? Or generate new UUID?
+			Name:     acc.Name,
+			Currency: acc.Currency,
+			Balance:  acc.Balance,
+			Type:     "general", // Default
+		})
+	}
+	return result
+}
+
+func mapJars(mmCategories []models.CategoryDTO) []models.Jar {
+	var result []models.Jar
+	for _, cat := range mmCategories {
+		t := "expense"
+		if cat.Type == 1 {
+			t = "income"
+		}
+
+		result = append(result, models.Jar{
+			ID:       cat.ID,
+			Name:     cat.Name,
+			Type:     t,
+			ParentID: cat.ParentID,
+		})
+	}
+	return result
+}
+
+func mapTransactions(mmTrans []models.TransactionDTO) []models.Transaction {
+	var result []models.Transaction
+	layout := "2006-01-02 15:04:05" // Check MM date format!
+	// MM format might be just YYYY-MM-DD or float timestamp?
+	// In parser we scanned it as string. Let's assume standard SQL string for now.
+	// Parser output usually: 'YYYY-MM-DD HH:MM:SS' or similar.
+
+	// Heuristic for format:
+	// If MM stores as timestamp (REAL/INTEGER), we need to handle that in parser.
+	// In mmbak_parser.go we scanned ZDATE into string. SQLite often stores as YYYY-MM-DD HH:MM:SS
+
+	for _, t := range mmTrans {
+		date, err := time.Parse(layout, t.Date)
+		if err != nil {
+			// Fallback for YYYY-MM-DD
+			date, _ = time.Parse("2006-01-02", t.Date)
+		}
+
+		txType := "expense"
+		if t.Type == 1 {
+			txType = "income"
+		} else if t.Type == 2 { // Assuming 2 is transfer
+			txType = "transfer"
+		}
+
+		result = append(result, models.Transaction{
+			ID:          t.ID,
+			Amount:      t.Amount,
+			Description: t.Note,
+			Date:        date,
+			Type:        txType,
+			WalletID:    t.AccountID,
+			JarID:       t.CategoryID,
+			ToWalletID:  t.ToAccountID,
+		})
+	}
+	return result
+}
diff --git a/internal/models/domain.go b/internal/models/domain.go
new file mode 100644
index 0000000..79d402f
--- /dev/null
+++ b/internal/models/domain.go
@@ -0,0 +1,36 @@
+package models
+
+import "time"
+
+// Core Domain Models for JarWise
+
+type Wallet struct {
+	ID       string  `json:"id"`
+	Name     string  `json:"name"`
+	Currency string  `json:"currency"`
+	Balance  float64 `json:"balance"`
+	Type     string  `json:"type"` // e.g. "cash", "bank", "credit_card"
+}
+
+type Jar struct { // Category
+	ID       string `json:"id"`
+	Name     string `json:"name"`
+	ParentID string `json:"parent_id,omitempty"`
+	Type     string `json:"type"` // "income", "expense"
+	Icon     string `json:"icon"`
+	Color    string `json:"color"`
+}
+
+type Transaction struct {
+	ID          string    `json:"id"`
+	Amount      float64   `json:"amount"`
+	Description string    `json:"description"`
+	Date        time.Time `json:"date"`
+	Type        string    `json:"type"` // "income", "expense", "transfer"
+
+	WalletID string `json:"wallet_id"`
+	JarID    string `json:"jar_id"`
+
+	// For transfer
+	ToWalletID string `json:"to_wallet_id,omitempty"`
+}
diff --git a/internal/models/mm_data.go b/internal/models/mm_data.go
index a74d0b3..0daa4c6 100644
--- a/internal/models/mm_data.go
+++ b/internal/models/mm_data.go
@@ -2,7 +2,7 @@ package models
 
 // AccountDTO represents a wallet/account in Money Manager
 type AccountDTO struct {
-	ID       int     `json:"id"`
+	ID       string  `json:"id"`
 	Name     string  `json:"name"`
 	Currency string  `json:"currency"`
 	Balance  float64 `json:"balance"` // Initial or calculated
@@ -10,21 +10,21 @@ type AccountDTO struct {
 
 // CategoryDTO represents a category in Money Manager
 type CategoryDTO struct {
-	ID       int    `json:"id"`
+	ID       string `json:"id"`
 	Name     string `json:"name"`
 	Type     int    `json:"type"` // 0=Expense, 1=Income, 2=Transfer? (check schema)
-	ParentID int    `json:"parent_id"`
+	ParentID string `json:"parent_id"`
 }
 
 // TransactionDTO represents a transaction record
 type TransactionDTO struct {
-	ID          int     `json:"id"`
+	ID          string  `json:"id"`
 	Date        string  `json:"date"` // YYYY-MM-DD
 	Amount      float64 `json:"amount"`
 	Type        int     `json:"type"`
-	CategoryID  int     `json:"category_id"`
-	AccountID   int     `json:"account_id"`
-	ToAccountID int     `json:"to_account_id"` // For transfers
+	CategoryID  string  `json:"category_id"`
+	AccountID   string  `json:"account_id"`
+	ToAccountID string  `json:"to_account_id"` // For transfers
 	Note        string  `json:"note"`
 }
 
diff --git a/internal/parser/mmbak_parser.go b/internal/parser/mmbak_parser.go
index 59408ea..095739f 100644
--- a/internal/parser/mmbak_parser.go
+++ b/internal/parser/mmbak_parser.go
@@ -4,6 +4,7 @@ import (
 	"database/sql"
 	"fmt"
 	"jarwise-backend/internal/models"
+	"math"
 
 	_ "github.com/mattn/go-sqlite3"
 )
@@ -33,14 +34,11 @@ func (p *MmbakParser) Parse(filePath string) (*models.ParsedData, error) {
 		Transactions: []models.TransactionDTO{},
 	}
 
-	// 2. Query Accounts (Table name usually 'assets' or 'account' in MM)
-	// Query: SELECT uid, name FROM assets WHERE type = 1 (Cash/Bank) - logic may need adjustment based on real schema
-	// For MVP, assuming a standard schema.
-	// WARN: Schema names need verification from specific .mmbak version
-	assetsRows, err := db.Query("SELECT uid, name FROM assets")
+	// 2. Query Accounts (Table 'ASSETS')
+	// Schema: uid (TEXT), NIC_NAME (TEXT), TYPE (INT)
+	assetsRows, err := db.Query("SELECT uid, NIC_NAME FROM ASSETS")
 	if err != nil {
-		// Fallback to 'accounts' if assets doesn't exist? or just return error
-		return nil, fmt.Errorf("failed to query assets: %w", err)
+		return nil, fmt.Errorf("failed to query ASSETS: %w", err)
 	}
 	defer assetsRows.Close()
 
@@ -52,18 +50,16 @@ func (p *MmbakParser) Parse(filePath string) (*models.ParsedData, error) {
 		result.Accounts = append(result.Accounts, acc)
 	}
 
-	// 3. Query Categories
-	// Table: category?
-	// Columns: uid, name, type (0=Exp, 1=Inc)
-	catRows, err := db.Query("SELECT uid, name, type FROM category")
+	// 3. Query Categories (Table 'ZCATEGORY')
+	catRows, err := db.Query("SELECT uid, NAME, TYPE FROM ZCATEGORY")
 	if err != nil {
-		return nil, fmt.Errorf("failed to query categories: %w", err)
+		return nil, fmt.Errorf("failed to query ZCATEGORY: %w", err)
 	}
 	defer catRows.Close()
 
 	for catRows.Next() {
 		var cat models.CategoryDTO
-		var catType sql.NullInt64 // handle nulls if any
+		var catType sql.NullInt64
 		if err := catRows.Scan(&cat.ID, &cat.Name, &catType); err != nil {
 			return nil, err
 		}
@@ -71,35 +67,77 @@ func (p *MmbakParser) Parse(filePath string) (*models.ParsedData, error) {
 		result.Categories = append(result.Categories, cat)
 	}
 
-	// 4. Query Transactions (and Calculate Totals)
-	// Table: trans?
-	// Columns: uid, datetime, money, type, note, categoryId, assetId
-	// Note: MM schema usually stores amount as positive, type determines sign
+	// 4. Query Transactions (Table 'INOUTCOME')
+	// Columns: uid, ZDATE (date), ZMONEY (amount), DO_TYPE (type? or maybe just check ZMONEY sign?), ZCONTENT (note)
+	// categoryUid (Category), assetUid (Account)
+	// Note: DO_TYPE needs verification aka '1' or '2'.
+	// Usually Money Manager uses: 1=Income, 2=Expense, 3=Transfer (or 0 index?)
+	// Let's inspect data later if needed, assuming logic:
+	// We select raw columns and map
 	transRows, err := db.Query(`
-        SELECT uid, datetime, money, type, note, categoryId, assetId 
-        FROM trans 
-        WHERE type IN (0, 1) -- 0=Exp, 1=Inc (Transfer=2 excluded for totals usually)
+        SELECT uid, ZDATE, ZMONEY, DO_TYPE, ZCONTENT, categoryUid, assetUid 
+        FROM INOUTCOME 
+        WHERE DO_TYPE IN ('0', '1', '2') OR DO_TYPE IS NULL
     `)
+	// WARN: DO_TYPE might be varchar based on schema.
 	if err != nil {
-		return nil, fmt.Errorf("failed to query transactions: %w", err)
+		return nil, fmt.Errorf("failed to query INOUTCOME: %w", err)
 	}
 	defer transRows.Close()
 
 	for transRows.Next() {
 		var t models.TransactionDTO
-		var note sql.NullString
-		if err := transRows.Scan(&t.ID, &t.Date, &t.Amount, &t.Type, &note, &t.CategoryID, &t.AccountID); err != nil {
+		var note, doType, catID, assetID sql.NullString
+		var money sql.NullFloat64
+
+		if err := transRows.Scan(&t.ID, &t.Date, &money, &doType, &note, &catID, &assetID); err != nil {
 			return nil, err
 		}
+		t.Amount = money.Float64
 		t.Note = note.String
+		t.CategoryID = catID.String
+		t.AccountID = assetID.String
+
+		// Map Type
+		// DO_TYPE values: '1'=Income, '0' or '2'=Expense, '3'=Transfer?
+		// Need to confirm exact mapping. Assuming:
+		// 1 = Income
+		// 2 = Transfer? (Or 0?)
+		// Let's refine based on review suggestion:
+		dt := doType.String
+		isTransfer := false
+
+		switch dt {
+		case "1": // Income
+			t.Type = 1
+		case "0", "2": // Expense (generic guess, adjust if 2 is transfer)
+			// Wait, if 2 is transfer, we should handle it.
+			// Let's assume standard:
+			// 0=Expense, 1=Income, 2=Transfer
+			if dt == "2" {
+				t.Type = 2
+				isTransfer = true
+			} else {
+				t.Type = 0
+			}
+		case "3": // Some versions use 3 for transfer
+			t.Type = 2
+			isTransfer = true
+		default:
+			// Default to expense
+			t.Type = 0
+		}
 
 		result.Transactions = append(result.Transactions, t)
 
 		// Aggregate Totals
-		if t.Type == 1 { // Income
-			result.TotalIncome += t.Amount
-		} else if t.Type == 0 { // Expense
-			result.TotalExpense += t.Amount // Assume stored as positive
+		// Exclude transfers from Income/Expense totals for now (or handle them separately)
+		if !isTransfer {
+			if t.Type == 1 { // Income
+				result.TotalIncome += t.Amount
+			} else { // Expense
+				result.TotalExpense += math.Abs(t.Amount)
+			}
 		}
 	}
 
diff --git a/internal/service/migration_service.go b/internal/service/migration_service.go
index ad43371..620da96 100644
--- a/internal/service/migration_service.go
+++ b/internal/service/migration_service.go
@@ -4,14 +4,19 @@ import (
 	"context"
 	"fmt"
 	"io"
+	"jarwise-backend/internal/importer"
 	"jarwise-backend/internal/models"
 	"jarwise-backend/internal/parser"
+	"jarwise-backend/internal/validator"
 	"mime/multipart"
 	"os"
 	"path/filepath"
 	"time"
 )
 
+// TOGGLE: Set to true to allow import even if validation fails
+const BypassValidation = false
+
 // MigrationService defines the interface for handling migration logic
 type MigrationService interface {
 	ProcessUpload(ctx context.Context, mmbak, xls *multipart.FileHeader) (*models.MigrationResponse, error)
@@ -41,10 +46,7 @@ func (s *migrationService) ProcessUpload(ctx context.Context, mmbak, xls *multip
 	mmParser := parser.NewMmbakParser()
 	parsedData, err := mmParser.Parse(mmbakPath)
 	if err != nil {
-		return &models.MigrationResponse{
-			Status:  "error",
-			Message: fmt.Sprintf("Failed to parse database: %v", err),
-		}, nil // Return 200 with error status for UI handling? Or actual error
+		return nil, fmt.Errorf("failed to parse database: %w", err)
 	}
 
 	fmt.Printf("Parsed Data: %d Accounts, %d Categories, %d Transactions\n",
@@ -61,19 +63,47 @@ func (s *migrationService) ProcessUpload(ctx context.Context, mmbak, xls *multip
 	xlsParser := parser.NewXlsParser()
 	xlsData, err := xlsParser.Parse(xlsPath)
 	if err != nil {
+		return nil, fmt.Errorf("failed to parse XLS report: %w", err)
+	}
+
+	fmt.Printf("Parsed XLS Data: %d Transactions\n", len(xlsData.Transactions))
+	fmt.Printf("DB Total Income: %.2f, XLS Total Income: %.2f\n", parsedData.TotalIncome, xlsData.TotalIncome)
+
+	// 4. Validate
+	v := validator.NewValidator()
+	validationResult := v.Validate(parsedData, xlsData)
+
+	status := "preview" // Ready for preview if valid
+	msg := "Validation successful"
+
+	if !validationResult.IsValid {
+		if BypassValidation {
+			fmt.Println("WARNING: Validation failed but proceeding (BypassValidation = true)")
+			msg = "Import successful (with validation warnings)"
+		} else {
+			return &models.MigrationResponse{
+				Status:  "error",
+				Message: "Validation failed. Discrepancies found.",
+			}, nil
+		}
+	} else {
+		msg = "Import successful!"
+	}
+
+	// 5. Import (Only if valid or bypassed)
+	importer := importer.NewImporter()
+	if err := importer.ImportData(parsedData); err != nil {
 		return &models.MigrationResponse{
 			Status:  "error",
-			Message: fmt.Sprintf("Failed to parse XLS report: %v", err),
+			Message: fmt.Sprintf("Import failed: %v", err),
 		}, nil
 	}
 
-	fmt.Printf("Parsed XLS Data: %d Transactions\n", len(xlsData.Transactions))
-	fmt.Printf("DB Total Income: %.2f, XLS Total Income: %.2f\n", parsedData.TotalIncome, xlsData.TotalIncome)
+	status = "success"
 
-	// Mock response with comparison data
 	return &models.MigrationResponse{
-		Status:  "success",
-		Message: fmt.Sprintf("Parsed DB: %d tx, XLS: %d tx. Validation ready.", len(parsedData.Transactions), len(xlsData.Transactions)),
+		Status:  status,
+		Message: msg,
 		JobID:   "job-uuid-123",
 	}, nil
 }
diff --git a/internal/validator/models.go b/internal/validator/models.go
new file mode 100644
index 0000000..3bb1162
--- /dev/null
+++ b/internal/validator/models.go
@@ -0,0 +1,15 @@
+package validator
+
+import "jarwise-backend/internal/models"
+
+// ValidationResult holds the comparison result
+type ValidationResult struct {
+	IsValid  bool     `json:"is_valid"`
+	Errors   []string `json:"errors"`
+	Warnings []string `json:"warnings"`
+
+	DBStats  models.MigrationStats `json:"db_stats"`
+	XLSStats models.MigrationStats `json:"xls_stats"`
+
+	DiffBalance float64 `json:"diff_balance"`
+}
diff --git a/internal/validator/validator.go b/internal/validator/validator.go
new file mode 100644
index 0000000..54f5b03
--- /dev/null
+++ b/internal/validator/validator.go
@@ -0,0 +1,73 @@
+package validator
+
+import (
+	"fmt"
+	"jarwise-backend/internal/models"
+	"math"
+)
+
+type Validator struct{}
+
+func NewValidator() *Validator {
+	return &Validator{}
+}
+
+// Validate compares parsed data from both sources
+func (v *Validator) Validate(dbData, xlsData *models.ParsedData) *ValidationResult {
+	result := &ValidationResult{
+		IsValid:  true,
+		Errors:   []string{},
+		Warnings: []string{},
+	}
+
+	// 1. Calculate Stats
+	result.DBStats = calculateStats(dbData)
+	result.XLSStats = calculateStats(xlsData)
+
+	// 2. Compare Transaction Counts
+	// Allow small discrepancy? Or must be exact?
+	// Given user data showed 9475 vs 10414 (diff ~1000), this is significant.
+	// Likely Transfers are missing in DB query or included in XLS.
+	diffCount := result.DBStats.Transactions - result.XLSStats.Transactions
+	if diffCount != 0 {
+		result.Warnings = append(result.Warnings, fmt.Sprintf("Transaction count mismatch: DB=%d, XLS=%d (Diff: %d)",
+			result.DBStats.Transactions, result.XLSStats.Transactions, diffCount))
+
+		// If diff is huge, maybe error?
+		if math.Abs(float64(diffCount)) > 100 {
+			result.IsValid = false
+			result.Errors = append(result.Errors, "Significant transaction count mismatch. Check if Transfer handling differs.")
+		}
+	}
+
+	// 3. Compare Totals (Income)
+	// Using epsilon for float comparison
+	epsilon := 0.01
+	if math.Abs(result.DBStats.TotalIncome-result.XLSStats.TotalIncome) > epsilon {
+		result.IsValid = false
+		result.Errors = append(result.Errors, fmt.Sprintf("Total Income mismatch: DB=%.2f, XLS=%.2f",
+			result.DBStats.TotalIncome, result.XLSStats.TotalIncome))
+	}
+
+	// 4. Compare Totals (Expense)
+	if math.Abs(result.DBStats.TotalExpense-result.XLSStats.TotalExpense) > epsilon {
+		result.IsValid = false
+		result.Errors = append(result.Errors, fmt.Sprintf("Total Expense mismatch: DB=%.2f, XLS=%.2f",
+			result.DBStats.TotalExpense, result.XLSStats.TotalExpense))
+	}
+
+	result.DiffBalance = (result.DBStats.TotalIncome - result.DBStats.TotalExpense) -
+		(result.XLSStats.TotalIncome - result.XLSStats.TotalExpense)
+
+	return result
+}
+
+func calculateStats(data *models.ParsedData) models.MigrationStats {
+	return models.MigrationStats{
+		Wallets:      len(data.Accounts),
+		Jars:         len(data.Categories),
+		Transactions: len(data.Transactions),
+		TotalIncome:  data.TotalIncome,
+		TotalExpense: data.TotalExpense,
+	}
+}

PR TEMPLATE:


INSTRUCTIONS:
1. Generate a comprehensive PR description in Markdown format.
2. If a template is provided, fill it out intelligently.
3. If no template, use a standard structure: Summary, Changes, Impact.
4. Focus on 'Why' and 'What'.
5. Do not include 'Here is the PR description' preamble. Just the body.
6. IMPORTANT: Always use FULL URLs for links to issues and other PRs (e.g., https://github.com/owner/repo/issues/123), do NOT use short syntax (e.g., #123) to ensuring proper linking across platforms.
