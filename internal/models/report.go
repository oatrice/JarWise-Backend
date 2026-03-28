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
// It reuses Common chart models (TrendPoint, CategoryAmount, etc.) from chart.go
type Report struct {
	Summary    ChartSummary     `json:"summary"`
	Trend      []TrendPoint     `json:"trend"`
	ByCategory []CategoryAmount `json:"by_category"`
	ByJar      []JarAmount      `json:"by_jar"`
	Comparison *ComparisonData  `json:"comparison,omitempty"`
	FilterUsed ReportFilter     `json:"filter_used"`

	// Keep for backward compatibility during transition
	TotalAmount      float64       `json:"total_amount"`
	TransactionCount int           `json:"transaction_count"`
	Transactions     []Transaction `json:"transactions"`
}
