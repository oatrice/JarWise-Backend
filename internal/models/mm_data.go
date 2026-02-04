package models

// AccountDTO represents a wallet/account in Money Manager
type AccountDTO struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Currency string  `json:"currency"`
	Balance  float64 `json:"balance"` // Initial or calculated
}

// CategoryDTO represents a category in Money Manager
type CategoryDTO struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Type     int    `json:"type"` // 0=Expense, 1=Income, 2=Transfer? (check schema)
	ParentID int    `json:"parent_id"`
}

// TransactionDTO represents a transaction record
type TransactionDTO struct {
	ID          int     `json:"id"`
	Date        string  `json:"date"` // YYYY-MM-DD
	Amount      float64 `json:"amount"`
	Type        int     `json:"type"`
	CategoryID  int     `json:"category_id"`
	AccountID   int     `json:"account_id"`
	ToAccountID int     `json:"to_account_id"` // For transfers
	Note        string  `json:"note"`
}

// ParsedData holds all extracted data from the mmbak file
type ParsedData struct {
	Accounts     []AccountDTO     `json:"accounts"`
	Categories   []CategoryDTO    `json:"categories"`
	Transactions []TransactionDTO `json:"transactions"`
	TotalIncome  float64          `json:"total_income"`
	TotalExpense float64          `json:"total_expense"`
}
