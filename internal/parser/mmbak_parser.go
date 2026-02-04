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

	// 2. Query Accounts (Table 'ASSETS')
	// Schema: uid (TEXT), NIC_NAME (TEXT), TYPE (INT)
	assetsRows, err := db.Query("SELECT uid, NIC_NAME FROM ASSETS")
	if err != nil {
		return nil, fmt.Errorf("failed to query ASSETS: %w", err)
	}
	defer assetsRows.Close()

	for assetsRows.Next() {
		var acc models.AccountDTO
		if err := assetsRows.Scan(&acc.ID, &acc.Name); err != nil {
			return nil, err
		}
		result.Accounts = append(result.Accounts, acc)
	}

	// 3. Query Categories (Table 'ZCATEGORY')
	catRows, err := db.Query("SELECT uid, NAME, TYPE FROM ZCATEGORY")
	if err != nil {
		return nil, fmt.Errorf("failed to query ZCATEGORY: %w", err)
	}
	defer catRows.Close()

	for catRows.Next() {
		var cat models.CategoryDTO
		var catType sql.NullInt64
		if err := catRows.Scan(&cat.ID, &cat.Name, &catType); err != nil {
			return nil, err
		}
		cat.Type = int(catType.Int64)
		result.Categories = append(result.Categories, cat)
	}

	// 4. Query Transactions (Table 'INOUTCOME')
	// Columns: uid, ZDATE (date), ZMONEY (amount), DO_TYPE (type? or maybe just check ZMONEY sign?), ZCONTENT (note)
	// categoryUid (Category), assetUid (Account)
	// Note: DO_TYPE needs verification aka '1' or '2'.
	// Usually Money Manager uses: 1=Income, 2=Expense, 3=Transfer (or 0 index?)
	// Let's inspect data later if needed, assuming logic:
	// We select raw columns and map
	transRows, err := db.Query(`
        SELECT uid, ZDATE, ZMONEY, DO_TYPE, ZCONTENT, categoryUid, assetUid 
        FROM INOUTCOME 
        WHERE DO_TYPE IN ('0', '1', '2') OR DO_TYPE IS NULL
    `)
	// WARN: DO_TYPE might be varchar based on schema.
	if err != nil {
		return nil, fmt.Errorf("failed to query INOUTCOME: %w", err)
	}
	defer transRows.Close()

	for transRows.Next() {
		var t models.TransactionDTO
		var note, doType, catID, assetID sql.NullString
		var money sql.NullFloat64

		if err := transRows.Scan(&t.ID, &t.Date, &money, &doType, &note, &catID, &assetID); err != nil {
			return nil, err
		}
		t.Amount = money.Float64
		t.Note = note.String
		t.CategoryID = catID.String
		t.AccountID = assetID.String

		// Map Type
		// If logic is unclear, we might logging raw DO_TYPE values helps.
		// Standard guess: 'Inc' or 'Exp'? Or '1'/'0'?
		// Schema said DO_TYPE is VARCHAR due to index creation? "CREATE INDEX ... ON INOUTCOME (DO_TYPE)"
		// Let's assume numeric string for now.
		dt := doType.String
		if dt == "1" { // Income?
			t.Type = 1
		} else {
			t.Type = 0 // Expense
			// Fix negative amount if needed?
		}

		result.Transactions = append(result.Transactions, t)

		// Aggregate Totals
		if t.Type == 1 { // Income
			result.TotalIncome += t.Amount
		} else { // Expense
			result.TotalExpense += t.Amount
		}
	}

	return result, nil
}
