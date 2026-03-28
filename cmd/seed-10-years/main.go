package main

import (
	"fmt"
	"jarwise-backend/internal/db"
	"log"
	"math/rand"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type jar struct {
	ID    string
	Name  string
}

func main() {
	// Initialize DB
	dbConn, err := db.InitDB("transactions.db")
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer dbConn.Close()

	fmt.Println("Wiping existing data for 10-year seed...")
	tables := []string{"transactions", "jars", "wallets"}
	for _, table := range tables {
		_, err := dbConn.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			log.Fatalf("Failed to clear table %s: %v", table, err)
		}
	}

	// 1. Seed Wallet
	walletID := "wallet-1"
	_, err = dbConn.Exec(`INSERT INTO wallets (id, name, currency, balance, type) VALUES (?, ?, ?, ?, ?)`,
		walletID, "Main Wallet", "THB", 100000.0, "checking")
	if err != nil {
		log.Fatalf("Failed to insert wallet: %v", err)
	}

	// 2. Seed Jars
	jars := []jar{
		{ID: "1", Name: "Necessities"},
		{ID: "2", Name: "Play"},
		{ID: "3", Name: "Education"},
		{ID: "4", Name: "Long Term"},
		{ID: "5", Name: "Freedom"},
		{ID: "6", Name: "Give"},
	}

	for _, j := range jars {
		_, err = dbConn.Exec(`INSERT INTO jars (id, name, type, wallet_id, icon, color) VALUES (?, ?, ?, ?, ?, ?)`,
			j.ID, j.Name, "jar", walletID, "Home", "text-blue-400")
		if err != nil {
			log.Fatalf("Failed to insert jar %s: %v", j.Name, err)
		}
	}

	// 3. Generate 10 Years of Data (120 months)
	fmt.Println("Generating 10 years of transactions...")
	now := time.Now()
	rand.Seed(time.Now().UnixNano())

	txCount := 0
	for m := 0; m < 120; m++ {
		// Calculate the target month
		targetMonth := now.AddDate(0, -m, 0)
		
		// Monthly Income
		incomeID := fmt.Sprintf("inc-%d", m)
		incomeAmount := 50000.0 + (rand.Float64() * 10000) // 50k - 60k salary
		_, err = dbConn.Exec(`INSERT INTO transactions (id, amount, description, date, type, wallet_id) VALUES (?, ?, ?, ?, ?, ?)`,
			incomeID, incomeAmount, "Monthly Salary", targetMonth.Format(time.RFC3339), "income", walletID)
		if err != nil {
			log.Fatalf("Failed to insert income for month %d: %v", m, err)
		}
		txCount++

		// Monthly Expenses (15-20 transactions per month)
		numExpenses := 15 + rand.Intn(10)
		for e := 0; e < numExpenses; e++ {
			txID := fmt.Sprintf("tx-%d-%d", m, e)
			selectedJar := jars[rand.Intn(len(jars))]
			amount := 100.0 + (rand.Float64() * 2000) // 100 - 2100 per expense
			
			// Slightly randomize the day within the month
			dayOffset := rand.Intn(28)
			txDate := time.Date(targetMonth.Year(), targetMonth.Month(), dayOffset+1, 12, 0, 0, 0, time.UTC)

			_, err = dbConn.Exec(`INSERT INTO transactions (id, amount, description, date, type, wallet_id, jar_id) VALUES (?, ?, ?, ?, ?, ?, ?)`,
				txID, amount, fmt.Sprintf("Expense %s #%d", selectedJar.Name, e), txDate.Format(time.RFC3339), "expense", walletID, selectedJar.ID)
			if err != nil {
				log.Fatalf("Failed to insert expense for month %d, tx %d: %v", m, e, err)
			}
			txCount++
		}
	}

	fmt.Printf("Successfully seeded %d transactions over 10 years!\n", txCount)
}
