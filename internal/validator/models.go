package validator

import "jarwise-backend/internal/models"

// ValidationResult holds the comparison result
type ValidationResult struct {
	IsValid  bool     `json:"is_valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`

	DBStats  models.MigrationStats `json:"db_stats"`
	XLSStats models.MigrationStats `json:"xls_stats"`

	DiffBalance float64 `json:"diff_balance"`
}
