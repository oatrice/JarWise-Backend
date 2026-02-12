package models

import "time"

// ReportFilter defines multi-select filters for reports.
type ReportFilter struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	JarIDs    []string  `json:"jar_ids"`
	WalletIDs []string  `json:"wallet_ids"`
}

// Report represents aggregated report data with the applied filter.
type Report struct {
	TotalAmount      float64     `json:"total_amount"`
	TransactionCount int         `json:"transaction_count"`
	Transactions     []Transaction `json:"transactions"`
	FilterUsed       ReportFilter `json:"filter_used"`
}
