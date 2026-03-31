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

// Parse reads the SQLite file and extracts data.
func (p *MmbakParser) Parse(filePath string) (*models.ParsedData, error) {
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

	accountIDs := make(map[string]struct{})
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
		accountIDs[acc.ID] = struct{}{}
		result.Accounts = append(result.Accounts, acc)
	}
	if err := assetsRows.Err(); err != nil {
		return nil, err
	}

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
	if err := catRows.Err(); err != nil {
		return nil, err
	}

	inoutcomeColumns, err := loadSQLiteColumnSet(db, "INOUTCOME")
	if err != nil {
		return nil, err
	}

	toAssetExpr := "''"
	if _, ok := inoutcomeColumns["toAssetUid"]; ok {
		toAssetExpr = "COALESCE(toAssetUid, '')"
	}

	transRows, err := db.Query(fmt.Sprintf(`
		SELECT uid, ZDATE, ZMONEY, DO_TYPE, ZCONTENT, categoryUid, assetUid, %s AS toAssetUid
		FROM INOUTCOME
		WHERE DO_TYPE IN ('0', '1', '2', '3') OR DO_TYPE IS NULL
	`, toAssetExpr))
	if err != nil {
		return nil, fmt.Errorf("failed to query INOUTCOME: %w", err)
	}
	defer transRows.Close()

	for transRows.Next() {
		var (
			t                            models.TransactionDTO
			dateStr, note, doType        sql.NullString
			categoryID, assetID, toAsset sql.NullString
			money                        sql.NullFloat64
		)

		if err := transRows.Scan(&t.ID, &dateStr, &money, &doType, &note, &categoryID, &assetID, &toAsset); err != nil {
			return nil, err
		}

		t.Date = dateStr.String
		t.Amount = math.Abs(money.Float64)
		t.Note = note.String
		t.AccountID = assetID.String

		txType, isTransfer := mapMoneyManagerTransactionType(doType.String, money.Float64)
		t.Type = txType

		if isTransfer {
			t.CategoryID = ""
			destinationAccountID := firstKnownAccount(accountIDs, toAsset.String, categoryID.String)
			t.ToAccountID = destinationAccountID
		} else {
			t.CategoryID = normalizeMoneyManagerCategory(categoryID.String)
		}

		result.Transactions = append(result.Transactions, t)

		if !isTransfer {
			if t.Type == 1 {
				result.TotalIncome += t.Amount
			} else {
				result.TotalExpense += t.Amount
			}
		}
	}
	if err := transRows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func loadSQLiteColumnSet(db *sql.DB, tableName string) (map[string]struct{}, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := make(map[string]struct{})
	for rows.Next() {
		var (
			cid          int
			name         string
			columnType   string
			notNull      int
			defaultValue sql.NullString
			primaryKey   int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return nil, err
		}
		columns[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("failed to query %s: table not found or empty schema", tableName)
	}

	return columns, nil
}

func mapMoneyManagerTransactionType(rawType string, amount float64) (int, bool) {
	switch rawType {
	case "1":
		return 1, false
	case "2", "3":
		return 2, true
	case "0":
		return 0, false
	default:
		if amount < 0 {
			return 0, false
		}
		return 0, false
	}
}

func normalizeMoneyManagerCategory(categoryID string) string {
	switch categoryID {
	case "", "0", "-4":
		return ""
	default:
		return categoryID
	}
}

func firstKnownAccount(accounts map[string]struct{}, candidates ...string) string {
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, ok := accounts[candidate]; ok {
			return candidate
		}
	}
	return ""
}
