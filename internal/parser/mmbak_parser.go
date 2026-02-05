package parser

import (
	"database/sql"
	"fmt"
	"jarwise-backend/internal/models"
	"math"

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
		var note, doType, catID, assetID, dateStr sql.NullString
		var money sql.NullFloat64

		if err := transRows.Scan(&t.ID, &dateStr, &money, &doType, &note, &catID, &assetID); err != nil {
			return nil, err
		}
		t.Date = dateStr.String
		t.Amount = money.Float64
		t.Note = note.String
		t.CategoryID = catID.String
		t.AccountID = assetID.String

		// Map Type
		// DO_TYPE values: '1'=Income, '0' or '2'=Expense, '3'=Transfer?
		// Need to confirm exact mapping. Assuming:
		// 1 = Income
		// 2 = Transfer? (Or 0?)
		// Let's refine based on review suggestion:
		dt := doType.String
		isTransfer := false

		switch dt {
		case "1": // Income
			t.Type = 1
		case "0", "2": // Expense (generic guess, adjust if 2 is transfer)
			// Wait, if 2 is transfer, we should handle it.
			// Let's assume standard:
			// 0=Expense, 1=Income, 2=Transfer
			if dt == "2" {
				t.Type = 2
				isTransfer = true
			} else {
				t.Type = 0
			}
		case "3": // Some versions use 3 for transfer
			t.Type = 2
			isTransfer = true
		default:
			// Default to expense
			t.Type = 0
		}

		result.Transactions = append(result.Transactions, t)

		// Aggregate Totals
		// Exclude transfers from Income/Expense totals for now (or handle them separately)
		if !isTransfer {
			if t.Type == 1 { // Income
				result.TotalIncome += t.Amount
			} else { // Expense
				result.TotalExpense += math.Abs(t.Amount)
			}
		}
	}

	return result, nil
}
