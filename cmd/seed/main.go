package main

import (
	"database/sql"
	"fmt"
	"jarwise-backend/internal/db"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type jar struct {
	ID    string
	Name  string
	Icon  string
	Color string
}

type transaction struct {
	ID              string
	Amount          float64
	Description     string
	Category        string
	Date            string
	IsTaxDeductible bool
}

func main() {
	dbConn, err := db.InitDB("transactions.db")
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer dbConn.Close()

	// 1. Clear existing data
	fmt.Println("Wiping existing data...")
	tables := []string{"transactions", "jars", "wallets"}
	for _, table := range tables {
		_, err := dbConn.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			log.Fatalf("Failed to clear table %s: %v", table, err)
		}
	}

	// 2. Insert Default Wallet
	fmt.Println("Seeding default wallet...")
	walletID := "wallet-1"
	_, err = dbConn.Exec(`INSERT INTO wallets (id, name, currency, balance, type) VALUES (?, ?, ?, ?, ?)`,
		walletID, "Main Wallet", "THB", 50000.0, "checking")
	if err != nil {
		log.Fatalf("Failed to insert wallet: %v", err)
	}

	// 3. Seed Jars (Initial Jars from feat/48)
	fmt.Println("Seeding jars...")
	initialJars := []jar{
		{ID: "1", Name: "Necessities", Icon: "Home", Color: "text-blue-400"},
		{ID: "2", Name: "Play", Icon: "Gamepad2", Color: "text-pink-400"},
		{ID: "3", Name: "Education", Icon: "GraduationCap", Color: "text-purple-400"},
		{ID: "4", Name: "Long Term", Icon: "Plane", Color: "text-green-400"},
		{ID: "5", Name: "Freedom", Icon: "DollarSign", Color: "text-yellow-400"},
		{ID: "6", Name: "Give", Icon: "Heart", Color: "text-red-400"},
	}

	for _, j := range initialJars {
		_, err = dbConn.Exec(`INSERT INTO jars (id, name, type, wallet_id, icon, color) VALUES (?, ?, ?, ?, ?, ?)`,
			j.ID, j.Name, "jar", walletID, j.Icon, j.Color)
		if err != nil {
			log.Fatalf("Failed to insert jar %s: %v", j.Name, err)
		}
	}

	// 4. Seed Transactions (Initial Transactions from feat/48 + More for better reports)
	fmt.Println("Seeding transactions...")
	now := time.Now()
	initialTransactions := []transaction{
		// Original Mock Data
		{ID: "t1", Amount: 12.99, Description: "Spotify Premium", Category: "Play", Date: "Today, 2:30 PM", IsTaxDeductible: false},
		{ID: "t2", Amount: 86.42, Description: "Whole Foods Market", Category: "Necessities", Date: "Yesterday, 6:15 PM", IsTaxDeductible: true},
		{ID: "t3", Amount: 6.50, Description: "Starbucks Coffee", Category: "Play", Date: "Yesterday, 8:00 AM", IsTaxDeductible: false},
		{ID: "t4", Amount: 999.00, Description: "Apple Store", Category: "Necessities", Date: "3 days ago", IsTaxDeductible: true},

		// Additional Expenses
		{ID: "t5", Amount: 45.00, Description: "Shell Petrol", Category: "Necessities", Date: "2 days ago", IsTaxDeductible: false},
		{ID: "t6", Amount: 120.00, Description: "Udemy Course", Category: "Education", Date: "4 days ago", IsTaxDeductible: true},
		{ID: "t7", Amount: 15.00, Description: "Netflix", Category: "Play", Date: "5 days ago", IsTaxDeductible: false},
		{ID: "t8", Amount: 200.00, Description: "Charity Donation", Category: "Give", Date: "6 days ago", IsTaxDeductible: true},
		{ID: "t9", Amount: 30.00, Description: "Amazon Kindle Book", Category: "Education", Date: "Today, 10:00 AM", IsTaxDeductible: false},

		// Income (Directly to Wallet for now, or to a Jar if needed)
		{ID: "inc1", Amount: 5000.00, Description: "Monthly Salary", Category: "Income", Date: "1 day ago", IsTaxDeductible: false},
		{ID: "inc2", Amount: 200.00, Description: "Freelance Project", Category: "Income", Date: "4 days ago", IsTaxDeductible: false},

		// Previous Month (February) Data for Comparison
		{ID: "feb1", Amount: 1500.00, Description: "Rent (Feb)", Category: "Necessities", Date: "30 days ago", IsTaxDeductible: false},
		{ID: "feb2", Amount: 800.00, Description: "Groceries (Feb)", Category: "Necessities", Date: "35 days ago", IsTaxDeductible: true},
		{ID: "feb3", Amount: 300.00, Description: "Dinner Out (Feb)", Category: "Play", Date: "40 days ago", IsTaxDeductible: false},
		{ID: "incFeb", Amount: 5000.00, Description: "Salary (Feb)", Category: "Income", Date: "30 days ago", IsTaxDeductible: false},
	}

	for _, tx := range initialTransactions {
		var jarID sql.NullString
		txType := "expense"
		if tx.Category == "Income" {
			txType = "income"
		} else {
			// Map category to Jar ID
			for _, j := range initialJars {
				if j.Name == tx.Category {
					jarID.String = j.ID
					jarID.Valid = true
					break
				}
			}
		}

		// Parse date
		txDate := parseDate(tx.Date, now)

		_, err = dbConn.Exec(`INSERT INTO transactions (id, amount, description, date, type, wallet_id, jar_id) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			tx.ID, tx.Amount, tx.Description, txDate, txType, walletID, jarID)
		if err != nil {
			log.Fatalf("Failed to insert transaction %s: %v", tx.Description, err)
		}
	}

	fmt.Println("Database seeded successfully!")
}

func parseDate(dateStr string, now time.Time) time.Time {
	if dateStr == "Today, 2:30 PM" {
		return time.Date(now.Year(), now.Month(), now.Day(), 14, 30, 0, 0, now.Location())
	}
	if dateStr == "Today, 10:00 AM" {
		return time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())
	}
	if dateStr == "Yesterday, 6:15 PM" {
		yesterday := now.AddDate(0, 0, -1)
		return time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 18, 15, 0, 0, now.Location())
	}
	if dateStr == "Yesterday, 8:00 AM" {
		yesterday := now.AddDate(0, 0, -1)
		return time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 8, 0, 0, 0, now.Location())
	}
	if dateStr == "1 day ago" {
		ago := now.AddDate(0, 0, -1)
		return time.Date(ago.Year(), ago.Month(), ago.Day(), 12, 0, 0, 0, now.Location())
	}
	if dateStr == "2 days ago" {
		ago := now.AddDate(0, 0, -2)
		return time.Date(ago.Year(), ago.Month(), ago.Day(), 12, 0, 0, 0, now.Location())
	}
	if dateStr == "3 days ago" {
		ago := now.AddDate(0, 0, -3)
		return time.Date(ago.Year(), ago.Month(), ago.Day(), 12, 0, 0, 0, now.Location())
	}
	if dateStr == "4 days ago" {
		ago := now.AddDate(0, 0, -4)
		return time.Date(ago.Year(), ago.Month(), ago.Day(), 12, 0, 0, 0, now.Location())
	}
	if dateStr == "5 days ago" {
		ago := now.AddDate(0, 0, -5)
		return time.Date(ago.Year(), ago.Month(), ago.Day(), 12, 0, 0, 0, now.Location())
	}
	if dateStr == "6 days ago" {
		ago := now.AddDate(0, 0, -6)
		return time.Date(ago.Year(), ago.Month(), ago.Day(), 12, 0, 0, 0, now.Location())
	}
	if dateStr == "30 days ago" {
		ago := now.AddDate(0, 0, -30)
		return time.Date(ago.Year(), ago.Month(), ago.Day(), 12, 0, 0, 0, now.Location())
	}
	if dateStr == "35 days ago" {
		ago := now.AddDate(0, 0, -35)
		return time.Date(ago.Year(), ago.Month(), ago.Day(), 12, 0, 0, 0, now.Location())
	}
	if dateStr == "40 days ago" {
		ago := now.AddDate(0, 0, -40)
		return time.Date(ago.Year(), ago.Month(), ago.Day(), 12, 0, 0, 0, now.Location())
	}
	return now
}
