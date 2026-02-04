package models

import "time"

// Core Domain Models for JarWise

type Wallet struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Currency string  `json:"currency"`
	Balance  float64 `json:"balance"`
	Type     string  `json:"type"` // e.g. "cash", "bank", "credit_card"
}

type Jar struct { // Category
	ID       string `json:"id"`
	Name     string `json:"name"`
	ParentID string `json:"parent_id,omitempty"`
	Type     string `json:"type"` // "income", "expense"
	Icon     string `json:"icon"`
	Color    string `json:"color"`
}

type Transaction struct {
	ID          string    `json:"id"`
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
	Type        string    `json:"type"` // "income", "expense", "transfer"

	WalletID string `json:"wallet_id"`
	JarID    string `json:"jar_id"`

	// For transfer
	ToWalletID string `json:"to_wallet_id,omitempty"`
}
