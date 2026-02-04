package parser

import (
	"database/sql"
	"fmt"
	"jarwise-backend/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

type MmbakParser struct{}

func NewMmbakParser() *MmbakParser {
	return &MmbakParser{}
}

// Parse reads the SQLite file and extracts data
func (p *MmbakParser) Parse(filePath string) (*models.ParsedData, error) {
	// 1. Open Database
	db, err := sql.Open("sqlite3", filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	result := &models.ParsedData{
		Accounts:     []models.AccountDTO{},
		Categories:   []models.CategoryDTO{},
		Transactions: []models.TransactionDTO{},
	}

	// 2. Query Accounts (Table name usually 'assets' or 'account' in MM)
	// Query: SELECT uid, name FROM assets WHERE type = 1 (Cash/Bank) - logic may need adjustment based on real schema
	// For MVP, assuming a standard schema.
	// WARN: Schema names need verification from specific .mmbak version
	assetsRows, err := db.Query("SELECT uid, name FROM assets")
	if err != nil {
		// Fallback to 'accounts' if assets doesn't exist? or just return error
		return nil, fmt.Errorf("failed to query assets: %w", err)
	}
	defer assetsRows.Close()

	for assetsRows.Next() {
		var acc models.AccountDTO
		if err := assetsRows.Scan(&acc.ID, &acc.Name); err != nil {
			return nil, err
		}
		result.Accounts = append(result.Accounts, acc)
	}

	// 3. Query Categories
	// Table: category?
	// Columns: uid, name, type (0=Exp, 1=Inc)
	catRows, err := db.Query("SELECT uid, name, type FROM category")
	if err != nil {
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}
	defer catRows.Close()

	for catRows.Next() {
		var cat models.CategoryDTO
		var catType sql.NullInt64 // handle nulls if any
		if err := catRows.Scan(&cat.ID, &cat.Name, &catType); err != nil {
			return nil, err
		}
		cat.Type = int(catType.Int64)
		result.Categories = append(result.Categories, cat)
	}

	// 4. Query Transactions (and Calculate Totals)
	// Table: trans?
	// Columns: uid, datetime, money, type, note, categoryId, assetId
	// Note: MM schema usually stores amount as positive, type determines sign
	transRows, err := db.Query(`
        SELECT uid, datetime, money, type, note, categoryId, assetId 
        FROM trans 
        WHERE type IN (0, 1) -- 0=Exp, 1=Inc (Transfer=2 excluded for totals usually)
    `)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer transRows.Close()

	for transRows.Next() {
		var t models.TransactionDTO
		var note sql.NullString
		if err := transRows.Scan(&t.ID, &t.Date, &t.Amount, &t.Type, &note, &t.CategoryID, &t.AccountID); err != nil {
			return nil, err
		}
		t.Note = note.String

		result.Transactions = append(result.Transactions, t)

		// Aggregate Totals
		if t.Type == 1 { // Income
			result.TotalIncome += t.Amount
		} else if t.Type == 0 { // Expense
			result.TotalExpense += t.Amount // Assume stored as positive
		}
	}

	return result, nil
}
