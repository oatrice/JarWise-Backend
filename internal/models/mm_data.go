package models

// AccountDTO represents a wallet/account in Money Manager
type AccountDTO struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Currency string  `json:"currency"`
	Balance  float64 `json:"balance"` // Initial or calculated
}

// CategoryDTO represents a category in Money Manager
type CategoryDTO struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     int    `json:"type"` // 0=Expense, 1=Income, 2=Transfer? (check schema)
	ParentID string `json:"parent_id"`
}

// TransactionDTO represents a transaction record
type TransactionDTO struct {
	ID          string  `json:"id"`
	Date        string  `json:"date"` // YYYY-MM-DD
	Amount      float64 `json:"amount"`
	Type        int     `json:"type"`
	CategoryID  string  `json:"category_id"`
	AccountID   string  `json:"account_id"`
	ToAccountID string  `json:"to_account_id"` // For transfers
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
