package models

import "mime/multipart"

// MigrationResponse represents the standard API response for migration status
type MigrationResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	JobID   string `json:"job_id,omitempty"`
}

// MigrationStats holds counts of imported items
type MigrationStats struct {
	Wallets      int     `json:"wallets"`
	Jars         int     `json:"jars"`
	Transactions int     `json:"transactions"`
	TotalIncome  float64 `json:"total_income"`
	TotalExpense float64 `json:"total_expense"`
}

// MigrationRequests holds the uploaded files
type MigrationUploadRequest struct {
	MmbakFile *multipart.FileHeader `form:"mmbak_file"`
	XlsFile   *multipart.FileHeader `form:"xls_file"`
}
