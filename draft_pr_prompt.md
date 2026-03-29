# PR Draft Prompt

You are an AI assistant helping to create a Pull Request description.
    
TASK: [Web | Android] Financial Reports & Data Export
ISSUE: {
  "title": "[Web | Android] Financial Reports & Data Export",
  "number": 59,
  "body": "# \ud83c\udfaf Objective\nImplement comprehensive financial reporting with charts, graphs, and data export capabilities.\n\n## \ud83e\udde0 AI Brain Context\n- [task.md](https://raw.githubusercontent.com/oatrice/JarWise-Backend/feat/59-financial-reports-export/docs/features/59_issue-59/ai_brain/task.md)\n- [walkthrough.md](https://raw.githubusercontent.com/oatrice/JarWise-Backend/feat/59-financial-reports-export/docs/features/59_issue-59/ai_brain/walkthrough.md)\n- [implementation_plan.md](https://raw.githubusercontent.com/oatrice/JarWise-Backend/feat/59-financial-reports-export/docs/features/59_issue-59/ai_brain/implementation_plan.md)\n\n\nCloses #59",
  "url": "https://github.com/oatrice/JarWise-Root/issues/59"
}

GIT CONTEXT:
COMMITS:
ec6428b docs: sync AI brain artifacts
8cd3627 ✨ feat(release): Upgrade to version 0.5.0
63d9c48 fix: correct yearly comparison period calculation in report generation
84b7663 ✨ feat(seeder): Enhance income generation and report comparison
eea1ada ✨ feat(seed): Add 10-year data seeding script
2bef597 ✨ feat(report): Add CSV export functionality
032f619 ✨ feat(report): Enhance report generation with jar data and comparison
451a101 feat(seed): expand transaction seed data for better reports
b54f982 feat: add database seed command for wallets, jars, and transactions
ac91291 ✨ feat(api): Add CORS middleware
baf8e5b refactor: use string concatenation for SQL date format
19992f4 ✨ feat(report): Enhance report generation with detailed breakdowns

STATS:
.gitignore                                         |   1 +
 CHANGELOG.md                                       |  15 ++
 VERSION                                            |   2 +-
 cmd/seed-10-years/main.go                          | 129 ++++++++++
 cmd/seed/main.go                                   | 185 ++++++++++++++
 .../59_issue-59/ai_brain/implementation_plan.md    |  51 ++++
 docs/features/59_issue-59/ai_brain/task.md         |  13 +
 docs/features/59_issue-59/ai_brain/walkthrough.md  |  42 ++++
 internal/api/cors_test.go                          |  25 ++
 internal/api/handlers/report_handler.go            |  74 ++++--
 internal/api/handlers/report_handler_test.go       |  46 ++++
 internal/api/router.go                             |  21 +-
 internal/models/chart.go                           |  20 +-
 internal/models/report.go                          |  10 +-
 internal/repository/jar_repository.go              |  46 ++++
 internal/repository/transaction_repository.go      |  14 +-
 internal/repository/wallet_repository.go           |  19 ++
 internal/service/report_service.go                 | 267 +++++++++++++++++++-
 internal/service/report_service_test.go            | 273 ++++++++++++++++++---
 19 files changed, 1178 insertions(+), 75 deletions(-)

KEY FILE DIFFS:
diff --git a/cmd/seed-10-years/main.go b/cmd/seed-10-years/main.go
new file mode 100644
index 0000000..6838196
--- /dev/null
+++ b/cmd/seed-10-years/main.go
@@ -0,0 +1,129 @@
+package main
+
+import (
+	"fmt"
+	"jarwise-backend/internal/db"
+	"log"
+	"math/rand"
+	"time"
+
+	_ "github.com/mattn/go-sqlite3"
+)
+
+type jar struct {
+	ID    string
+	Name  string
+}
+
+func main() {
+	// Initialize DB
+	dbConn, err := db.InitDB("transactions.db")
+	if err != nil {
+		log.Fatalf("Failed to connect to DB: %v", err)
+	}
+	defer dbConn.Close()
+
+	fmt.Println("Wiping existing data for 10-year seed...")
+	tables := []string{"transactions", "jars", "wallets"}
+	for _, table := range tables {
+		_, err := dbConn.Exec(fmt.Sprintf("DELETE FROM %s", table))
+		if err != nil {
+			log.Fatalf("Failed to clear table %s: %v", table, err)
+		}
+	}
+
+	// 1. Seed Wallet
+	walletID := "wallet-1"
+	_, err = dbConn.Exec(`INSERT INTO wallets (id, name, currency, balance, type) VALUES (?, ?, ?, ?, ?)`,
+		walletID, "Main Wallet", "THB", 100000.0, "checking")
+	if err != nil {
+		log.Fatalf("Failed to insert wallet: %v", err)
+	}
+
+	// 2. Seed Jars
+	jars := []jar{
+		{ID: "1", Name: "Necessities"},
+		{ID: "2", Name: "Play"},
+		{ID: "3", Name: "Education"},
+		{ID: "4", Name: "Long Term"},
+		{ID: "5", Name: "Freedom"},
+		{ID: "6", Name: "Give"},
+	}
+
+	for _, j := range jars {
+		_, err = dbConn.Exec(`INSERT INTO jars (id, name, type, wallet_id, icon, color) VALUES (?, ?, ?, ?, ?, ?)`,
+			j.ID, j.Name, "jar", walletID, "Home", "text-blue-400")
+		if err != nil {
+			log.Fatalf("Failed to insert jar %s: %v", j.Name, err)
+		}
+	}
+
+	// 3. Generate 10 Years of Data (120 months)
+	fmt.Println("Generating 10 years of transactions...")
+	now := time.Now()
+	rand.Seed(time.Now().UnixNano())
+
+	txCount := 0
+	for m := 0; m < 120; m++ {
+		// Calculate the target month
+		targetMonth := now.AddDate(0, -m, 0)
+		
+		// Rich Monthly Income (2-5 sources)
+		numIncomes := 2 + rand.Intn(4)
+		incomeSources := []struct {
+			name string
+			jar  string
+			min  float64
+			max  float64
+		}{
+			{"Main Salary", "1", 45000, 55000},    // Necessities
+			{"Project Fee", "3", 8000, 25000},     // Education
+			{"Consulting", "5", 5000, 15000},      // Freedom
+			{"Dividends", "5", 1000, 4000},        // Freedom
+			{"Freelance Task", "2", 3000, 12000},  // Play
+			{"Gift Received", "6", 500, 3000},     // Give
+			{"Annual Bonus", "4", 10000, 50000},   // Long Term (rare, we'll randomize)
+		}
+
+		for i := 0; i < numIncomes; i++ {
+			inc := incomeSources[rand.Intn(len(incomeSources))]
+			// Monthly Bonus check (only Dec/Jan)
+			if inc.name == "Annual Bonus" && targetMonth.Month() != time.December && targetMonth.Month() != time.January {
+				continue
+			}
+
+			incomeID := fmt.Sprintf("inc-%d-%d", m, i)
+			amount := inc.min + (rand.Float64() * (inc.max - inc.min))
+			dayOffset := rand.Intn(28)
+			txDate := time.Date(targetMonth.Year(), targetMonth.Month(), dayOffset+1, 10, 0, 0, 0, time.UTC)
+
+			_, err = dbConn.Exec(`INSERT INTO transactions (id, amount, description, date, type, wallet_id, jar_id) VALUES (?, ?, ?, ?, ?, ?, ?)`,
+				incomeID, amount, inc.name, txDate.Format(time.RFC3339), "income", walletID, inc.jar)
+			if err != nil {
+				log.Fatalf("Failed to insert income %s for month %d: %v", inc.name, m, err)
+			}
+			txCount++
+		}
+
+		// Monthly Expenses (15-20 transactions per month)
+		numExpenses := 15 + rand.Intn(10)
+		for e := 0; e < numExpenses; e++ {
+			txID := fmt.Sprintf("tx-%d-%d", m, e)
+			selectedJar := jars[rand.Intn(len(jars))]
+			amount := 100.0 + (rand.Float64() * 2000) // 100 - 2100 per expense
+			
+			// Slightly randomize the day within the month
+			dayOffset := rand.Intn(28)
+			txDate := time.Date(targetMonth.Year(), targetMonth.Month(), dayOffset+1, 12, 0, 0, 0, time.UTC)
+
+			_, err = dbConn.Exec(`INSERT INTO transactions (id, amount, description, date, type, wallet_id, jar_id) VALUES (?, ?, ?, ?, ?, ?, ?)`,
+				txID, amount, fmt.Sprintf("Expense %s #%d", selectedJar.Name, e), txDate.Format(time.RFC3339), "expense", walletID, selectedJar.ID)
+			if err != nil {
+				log.Fatalf("Failed to insert expense for month %d, tx %d: %v", m, e, err)
+			}
+			txCount++
+		}
+	}
+
+	fmt.Printf("Successfully seeded %d transactions over 10 years!\n", txCount)
+}
diff --git a/cmd/seed/main.go b/cmd/seed/main.go
new file mode 100644
index 0000000..e15bf3f
--- /dev/null
+++ b/cmd/seed/main.go
@@ -0,0 +1,185 @@
+package main
+
+import (
+
+	"database/sql"
+	"fmt"
+	"jarwise-backend/internal/db"
+	"log"
+	"time"
+
+	_ "github.com/mattn/go-sqlite3"
+)
+
+type jar struct {
+	ID    string
+	Name  string
+	Icon  string
+	Color string
+}
+
+type transaction struct {
+	ID              string
+	Amount          float64
+	Description     string
+	Category        string
+	Date            string
+	IsTaxDeductible bool
+}
+
+func main() {
+	dbConn, err := db.InitDB("transactions.db")
+	if err != nil {
+		log.Fatalf("Failed to connect to DB: %v", err)
+	}
+	defer dbConn.Close()
+
+
+	// 1. Clear existing data
+	fmt.Println("Wiping existing data...")
+	tables := []string{"transactions", "jars", "wallets"}
+	for _, table := range tables {
+		_, err := dbConn.Exec(fmt.Sprintf("DELETE FROM %s", table))
+		if err != nil {
+			log.Fatalf("Failed to clear table %s: %v", table, err)
+		}
+	}
+
+	// 2. Insert Default Wallet
+	fmt.Println("Seeding default wallet...")
+	walletID := "wallet-1"
+	_, err = dbConn.Exec(`INSERT INTO wallets (id, name, currency, balance, type) VALUES (?, ?, ?, ?, ?)`,
+		walletID, "Main Wallet", "THB", 50000.0, "checking")
+	if err != nil {
+		log.Fatalf("Failed to insert wallet: %v", err)
+	}
+
+	// 3. Seed Jars (Initial Jars from feat/48)
+	fmt.Println("Seeding jars...")
+	initialJars := []jar{
+		{ID: "1", Name: "Necessities", Icon: "Home", Color: "text-blue-400"},
+		{ID: "2", Name: "Play", Icon: "Gamepad2", Color: "text-pink-400"},
+		{ID: "3", Name: "Education", Icon: "GraduationCap", Color: "text-purple-400"},
+		{ID: "4", Name: "Long Term", Icon: "Plane", Color: "text-green-400"},
+		{ID: "5", Name: "Freedom", Icon: "DollarSign", Color: "text-yellow-400"},
+		{ID: "6", Name: "Give", Icon: "Heart", Color: "text-red-400"},
+	}
+
+	for _, j := range initialJars {
+		_, err = dbConn.Exec(`INSERT INTO jars (id, name, type, wallet_id, icon, color) VALUES (?, ?, ?, ?, ?, ?)`,
+			j.ID, j.Name, "jar", walletID, j.Icon, j.Color)
+		if err != nil {
+			log.Fatalf("Failed to insert jar %s: %v", j.Name, err)
+		}
+	}
+
+	// 4. Seed Transactions (Initial Transactions from feat/48 + More for better reports)
+	fmt.Println("Seeding transactions...")
+	now := time.Now()
+	initialTransactions := []transaction{
+		// Original Mock Data
+		{ID: "t1", Amount: 12.99, Description: "Spotify Premium", Category: "Play", Date: "Today, 2:30 PM", IsTaxDeductible: false},
+		{ID: "t2", Amount: 86.42, Description: "Whole Foods Market", Category: "Necessities", Date: "Yesterday, 6:15 PM", IsTaxDeductible: true},
+		{ID: "t3", Amount: 6.50, Description: "Starbucks Coffee", Category: "Play", Date: "Yesterday, 8:00 AM", IsTaxDeductible: false},
+		{ID: "t4", Amount: 999.00, Description: "Apple Store", Category: "Necessities", Date: "3 days ago", IsTaxDeductible: true},
+
+		// Additional Expenses
+		{ID: "t5", Amount: 45.00, Description: "Shell Petrol", Category: "Necessities", Date: "2 days ago", IsTaxDeductible: false},
+		{ID: "t6", Amount: 120.00, Description: "Udemy Course", Category: "Education", Date: "4 days ago", IsTaxDeductible: true},
+		{ID: "t7", Amount: 15.00, Description: "Netflix", Category: "Play", Date: "5 days ago", IsTaxDeductible: false},
+		{ID: "t8", Amount: 200.00, Description: "Charity Donation", Category: "Give", Date: "6 days ago", IsTaxDeductible: true},
+		{ID: "t9", Amount: 30.00, Description: "Amazon Kindle Book", Category: "Education", Date: "Today, 10:00 AM", IsTaxDeductible: false},
+
+		// Income (Directly to Wallet for now, or to a Jar if needed)
+		{ID: "inc1", Amount: 5000.00, Description: "Monthly Salary", Category: "Income", Date: "1 day ago", IsTaxDeductible: false},
+		{ID: "inc2", Amount: 200.00, Description: "Freelance Project", Category: "Income", Date: "4 days ago", IsTaxDeductible: false},
+
+		// Previous Month (February) Data for Comparison
+		{ID: "feb1", Amount: 1500.00, Description: "Rent (Feb)", Category: "Necessities", Date: "30 days ago", IsTaxDeductible: false},
+		{ID: "feb2", Amount: 800.00, Description: "Groceries (Feb)", Category: "Necessities", Date: "35 days ago", IsTaxDeductible: true},
+		{ID: "feb3", Amount: 300.00, Description: "Dinner Out (Feb)", Category: "Play", Date: "40 days ago", IsTaxDeductible: false},
+		{ID: "incFeb", Amount: 5000.00, Description: "Salary (Feb)", Category: "Income", Date: "30 days ago", IsTaxDeductible: false},
+	}
+
+	for _, tx := range initialTransactions {
+		var jarID sql.NullString
+		txType := "expense"
+		if tx.Category == "Income" {
+			txType = "income"
+		} else {
+			// Map category to Jar ID
+			for _, j := range initialJars {
+				if j.Name == tx.Category {
+					jarID.String = j.ID
+					jarID.Valid = true
+					break
+				}
+			}
+		}
+
+		// Parse date
+		txDate := parseDate(tx.Date, now)
+
+		_, err = dbConn.Exec(`INSERT INTO transactions (id, amount, description, date, type, wallet_id, jar_id) VALUES (?, ?, ?, ?, ?, ?, ?)`,
+			tx.ID, tx.Amount, tx.Description, txDate, txType, walletID, jarID)
+		if err != nil {
+			log.Fatalf("Failed to insert transaction %s: %v", tx.Description, err)
+		}
+	}
+
+	fmt.Println("Database seeded successfully!")
+}
+
+func parseDate(dateStr string, now time.Time) time.Time {
+	if dateStr == "Today, 2:30 PM" {
+		return time.Date(now.Year(), now.Month(), now.Day(), 14, 30, 0, 0, now.Location())
+	}
+	if dateStr == "Today, 10:00 AM" {
+		return time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())
+	}
+	if dateStr == "Yesterday, 6:15 PM" {
+		yesterday := now.AddDate(0, 0, -1)
+		return time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 18, 15, 0, 0, now.Location())
+	}
+	if dateStr == "Yesterday, 8:00 AM" {
+		yesterday := now.AddDate(0, 0, -1)
+		return time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 8, 0, 0, 0, now.Location())
+	}
+	if dateStr == "1 day ago" {
+		ago := now.AddDate(0, 0, -1)
+		return time.Date(ago.Year(), ago.Month(), ago.Day(), 12, 0, 0, 0, now.Location())
+	}
+	if dateStr == "2 days ago" {
+		ago := now.AddDate(0, 0, -2)
+		return time.Date(ago.Year(), ago.Month(), ago.Day(), 12, 0, 0, 0, now.Location())
+	}
+	if dateStr == "3 days ago" {
+		ago := now.AddDate(0, 0, -3)
+		return time.Date(ago.Year(), ago.Month(), ago.Day(), 12, 0, 0, 0, now.Location())
+	}
+	if dateStr == "4 days ago" {
+		ago := now.AddDate(0, 0, -4)
+		return time.Date(ago.Year(), ago.Month(), ago.Day(), 12, 0, 0, 0, now.Location())
+	}
+	if dateStr == "5 days ago" {
+		ago := now.AddDate(0, 0, -5)
+		return time.Date(ago.Year(), ago.Month(), ago.Day(), 12, 0, 0, 0, now.Location())
+	}
+	if dateStr == "6 days ago" {
+		ago := now.AddDate(0, 0, -6)
+		return time.Date(ago.Year(), ago.Month(), ago.Day(), 12, 0, 0, 0, now.Location())
+	}
+	if dateStr == "30 days ago" {
+		ago := now.AddDate(0, 0, -30)
+		return time.Date(ago.Year(), ago.Month(), ago.Day(), 12, 0, 0, 0, now.Location())
+	}
+	if dateStr == "35 days ago" {
+		ago := now.AddDate(0, 0, -35)
+		return time.Date(ago.Year(), ago.Month(), ago.Day(), 12, 0, 0, 0, now.Location())
+	}
+	if dateStr == "40 days ago" {
+		ago := now.AddDate(0, 0, -40)
+		return time.Date(ago.Year(), ago.Month(), ago.Day(), 12, 0, 0, 0, now.Location())
+	}
+	return now
+}
diff --git a/internal/api/cors_test.go b/internal/api/cors_test.go
new file mode 100644
index 0000000..dfa0d5f
--- /dev/null
+++ b/internal/api/cors_test.go
@@ -0,0 +1,25 @@
+package api
+
+import (
+	"net/http"
+	"net/http/httptest"
+	"testing"
+)
+
+func TestCORSHandler(t *testing.T) {
+	mux := NewRouter()
+	
+	// Create a mock request with the origin header
+	req, _ := http.NewRequest("OPTIONS", "/api/v1/reports", nil)
+	req.Header.Set("Origin", "http://localhost:5173")
+	req.Header.Set("Access-Control-Request-Method", "GET")
+	
+	rr := httptest.NewRecorder()
+	mux.ServeHTTP(rr, req)
+	
+	// เราคาดหวังว่าพอร์ตจะเป็น 8081 และต้องมี CORS Header
+	if rr.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
+		t.Errorf("Expected Access-Control-Allow-Origin to be http://localhost:5173, got %s", 
+			rr.Header().Get("Access-Control-Allow-Origin"))
+	}
+}
diff --git a/internal/api/handlers/report_handler.go b/internal/api/handlers/report_handler.go
index d346a86..9cd0cbc 100644
--- a/internal/api/handlers/report_handler.go
+++ b/internal/api/handlers/report_handler.go
@@ -2,8 +2,10 @@ package handlers
 
 import (
 	"encoding/json"
+	"fmt"
 	"jarwise-backend/internal/models"
 	"jarwise-backend/internal/service"
+	"log"
 	"net/http"
 	"strings"
 	"time"
@@ -23,43 +25,75 @@ func (h *ReportHandler) GetReport(w http.ResponseWriter, r *http.Request) {
 		return
 	}
 
+	filter, err := h.parseFilter(r)
+	if err != nil {
+		http.Error(w, err.Error(), http.StatusBadRequest)
+		return
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
+func (h *ReportHandler) ExportReport(w http.ResponseWriter, r *http.Request) {
+	if r.Method != http.MethodGet {
+		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
+		return
+	}
+
+	filter, err := h.parseFilter(r)
+	if err != nil {
+		http.Error(w, err.Error(), http.StatusBadRequest)
+		return
+	}
+
+	csvData, err := h.service.ExportTransactionsToCSV(r.Context(), filter)
+	if err != nil {
+		log.Printf("Export error: %v", err)
+		http.Error(w, "Failed to export report: "+err.Error(), http.StatusInternalServerError)
+		return
+	}
+
+	log.Printf("Exporting CSV: %d bytes (Filter: %v - %v)", len(csvData), filter.StartDate, filter.EndDate)
+
+	filename := "jarwise-report-" + time.Now().Format("2006-01-02-150405") + ".csv"
+	w.Header().Set("Content-Type", "text/csv")
+	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
+	w.Write(csvData)
+}
+
+func (h *ReportHandler) parseFilter(r *http.Request) (models.ReportFilter, error) {
 	now := time.Now().UTC()
 	defaultStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
 	defaultEnd := defaultStart.AddDate(0, 1, 0).Add(-time.Nanosecond)
 
 	startDate, err := parseDateParam(r.URL.Query().Get("start_date"), defaultStart, false)
 	if err != nil {
-		http.Error(w, "Invalid start_date format. Use YYYY-MM-DD or RFC3339.", http.StatusBadRequest)
-		return
+		return models.ReportFilter{}, fmt.Errorf("invalid start_date format. Use YYYY-MM-DD or RFC3339")
 	}
 	endDate, err := parseDateParam(r.URL.Query().Get("end_date"), defaultEnd, true)
 	if err != nil {
-		http.Error(w, "Invalid end_date format. Use YYYY-MM-DD or RFC3339.", http.StatusBadRequest)
-		return
+		return models.ReportFilter{}, fmt.Errorf("invalid end_date format. Use YYYY-MM-DD or RFC3339")
 	}
 	if endDate.Before(startDate) {
-		http.Error(w, "end_date must be after start_date", http.StatusBadRequest)
-		return
+		return models.ReportFilter{}, fmt.Errorf("end_date must be after start_date")
 	}
 
 	jarIDs := parseIDsParam(r, "jar_ids", "category_ids")
 	walletIDs := parseIDsParam(r, "wallet_ids", "account_ids")
 
-	filter := models.ReportFilter{
+	return models.ReportFilter{
 		StartDate: startDate,
 		EndDate:   endDate,
 		JarIDs:    jarIDs,
 		WalletIDs: walletIDs,
-	}
-
-	report, err := h.service.GenerateReport(r.Context(), filter)
-	if err != nil {
-		http.Error(w, "Failed to generate report", http.StatusInternalServerError)
-		return
-	}
-
-	w.Header().Set("Content-Type", "application/json")
-	json.NewEncoder(w).Encode(report)
+	}, nil
 }
 
 func parseIDsParam(r *http.Request, keys ...string) []string {
@@ -104,7 +138,11 @@ func parseDateParam(value string, defaultValue time.Time, isEnd bool) (time.Time
 
 	parsed, err := time.Parse(time.RFC3339, value)
 	if err != nil {
-		return time.Time{}, err
+		// Fallback for fractional seconds if RFC3339 is strict (Go 1.x behavior varies)
+		parsed, err = time.Parse(time.RFC3339Nano, value)
+		if err != nil {
+			return time.Time{}, err
+		}
 	}
 	return parsed, nil
 }
diff --git a/internal/api/handlers/report_handler_test.go b/internal/api/handlers/report_handler_test.go
new file mode 100644
index 0000000..6ff8de5
--- /dev/null
+++ b/internal/api/handlers/report_handler_test.go
@@ -0,0 +1,46 @@
+package handlers
+
+import (
+	"context"
+	"jarwise-backend/internal/models"
+	"net/http"
+	"net/http/httptest"
+	"strings"
+	"testing"
+)
+
+type mockReportService struct{}
+
+func (m *mockReportService) GenerateReport(ctx context.Context, filter models.ReportFilter) (*models.Report, error) {
+	return nil, nil
+}
+func (m *mockReportService) ExportTransactionsToCSV(ctx context.Context, filter models.ReportFilter) ([]byte, error) {
+	return []byte("Date,Description,Amount,Type,Wallet,Jar\n2026-03-28,Test,100.00,expense,Main,Savings\n"), nil
+}
+
+func TestExportReport(t *testing.T) {
+	svc := &mockReportService{}
+	h := NewReportHandler(svc)
+
+	req := httptest.NewRequest("GET", "/api/v1/reports/export?start_date=2026-03-01&end_date=2026-03-31", nil)
+	w := httptest.NewRecorder()
+
+	h.ExportReport(w, req)
+
+	if w.Code != http.StatusOK {
+		t.Errorf("Expected status 200, got %d", w.Code)
+	}
+
+	contentType := w.Header().Get("Content-Type")
+	if contentType != "text/csv" {
+		t.Errorf("Expected Content-Type text/csv, got %s", contentType)
+	}
+
+	body := w.Body.String()
+	if !strings.Contains(body, "Date,Description,Amount,Type,Wallet,Jar") {
+		t.Errorf("Response body missing header, got: %s", body)
+	}
+	if !strings.Contains(body, "2026-03-28,Test,100.00,expense,Main,Savings") {
+		t.Errorf("Response body missing data, got: %s", body)
+	}
+}
diff --git a/internal/api/router.go b/internal/api/router.go
index 8b3d742..cdd3aee 100644
--- a/internal/api/router.go
+++ b/internal/api/router.go
@@ -28,7 +28,8 @@ func NewRouter() http.Handler {
 	txRepo := repository.NewSQLiteTransactionRepository(dbConn)
 	txService := service.NewTransactionService(txRepo, walletRepo)
 	txHandler := handlers.NewTransactionHandler(txService)
-	reportService := service.NewReportService(txRepo)
+	jarRepo := repository.NewSQLiteJarRepository(dbConn)
+	reportService := service.NewReportService(txRepo, jarRepo, walletRepo)
 	reportHandler := handlers.NewReportHandler(reportService)
 
 	graphService := service.NewGraphService(txRepo)
@@ -48,6 +49,7 @@ func NewRouter() http.Handler {
 
 	mux.HandleFunc("/api/v1/transfers", txHandler.CreateTransfer)
 	mux.HandleFunc("/api/v1/reports", reportHandler.GetReport)
+	mux.HandleFunc("/api/v1/reports/export", reportHandler.ExportReport)
 	mux.HandleFunc("/api/v1/graph/expenses", graphHandler.GetExpenseGraphData)
 	mux.HandleFunc("/api/v1/charts", chartHandler.GetChartData)
 
@@ -60,5 +62,20 @@ func NewRouter() http.Handler {
 		w.Write([]byte("OK"))
 	})
 
-	return mux
+	return CORSMiddleware(mux)
+}
+
+func CORSMiddleware(next http.Handler) http.Handler {
+	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
+		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
+		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
+		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
+
+		if r.Method == "OPTIONS" {
+			w.WriteHeader(http.StatusNoContent)
+			return
+		}
+
+		next.ServeHTTP(w, r)
+	})
 }
diff --git a/internal/models/chart.go b/internal/models/chart.go
index 0dc1b96..99d5170 100644
--- a/internal/models/chart.go
++
... (Diff truncated for size) ...


PR TEMPLATE:
# 📋 Backend Update Summary
<!-- 
Brief description of the changes in this PR.
-->

## ✅ Checklist
- [ ] 🏗️ I have moved the related issue to "In Progress" on the Kanban board
- [ ] 🧪 Tests added/updated and verified locally
- [ ] 🔄 All CI checks passed

## 🎯 Type
- [ ] ✨ New Feature
- [ ] 🐛 Bug Fix
- [ ] 🛠️ Refactoring
- [ ] 📄 Documentation
- [ ] 🔄 CI/Workflow update
- [ ] 💥 Breaking change

## 📝 Detailed Changes
<!-- Describe the purpose and implementation details of this PR -->

## 🧪 Testing Results
<!-- Provide test logs or descriptions of testing performed -->

## 🚀 Migration/Database Changes
- [ ] Database schema updated
- [ ] Environment variables updated

```sql
-- SQL Migration if applicable
```

## 🔗 Related Issues
<!-- 
Use 'Resolves' keyword with FULL repo reference for auto-linking.
Example: Resolves oatrice/JarWise-Backend#16
-->
- Resolves oatrice/JarWise-Backend#

**Breaking Changes**: <!-- Yes/No -->
**Migration Required**: <!-- Yes/No -->


INSTRUCTIONS:
1. Generate a comprehensive PR description in Markdown format.
2. If a template is provided, fill it out intelligently.
3. If no template, use a standard structure: Summary, Changes, Impact.
4. Focus on 'Why' and 'What'.
5. Do not include 'Here is the PR description' preamble. Just the body.
6. IMPORTANT: Always use the exact FULL URL for closing issues. You must write `Closes https://github.com/oatrice/JarWise-Root/issues/59`. Do NOT use short syntax (e.g., #123) and do not invent an owner/repo.
