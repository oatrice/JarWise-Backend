package models

import (
	"mime/multipart"
	"time"
)

type MigrationPhase string

const (
	MigrationPhaseValidating       MigrationPhase = "validating"
	MigrationPhasePreviewReady     MigrationPhase = "preview_ready"
	MigrationPhaseDuplicateBlocked MigrationPhase = "duplicate_blocked"
	MigrationPhaseImporting        MigrationPhase = "importing"
	MigrationPhaseCompleted        MigrationPhase = "completed"
	MigrationPhaseFailed           MigrationPhase = "failed"
	MigrationPhaseExpired          MigrationPhase = "expired"
)

type MigrationValidationError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type MigrationDuplicateItem struct {
	SourceID    string `json:"sourceId,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	MatchedBy   string `json:"matchedBy,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

type MigrationDuplicateSummary struct {
	Wallets      []MigrationDuplicateItem `json:"wallets"`
	Jars         []MigrationDuplicateItem `json:"jars"`
	Transactions []MigrationDuplicateItem `json:"transactions"`
}

type MigrationJobCounts struct {
	Wallets      int     `json:"wallets"`
	Jars         int     `json:"jars"`
	Transactions int     `json:"transactions"`
	TotalIncome  float64 `json:"totalIncome"`
	TotalExpense float64 `json:"totalExpense"`
}

type MigrationJobStatusResponse struct {
	JobID            string                     `json:"jobId"`
	Phase            MigrationPhase             `json:"phase"`
	Message          string                     `json:"message,omitempty"`
	Counts           *MigrationJobCounts        `json:"counts,omitempty"`
	ValidationErrors []MigrationValidationError `json:"validationErrors,omitempty"`
	DuplicateSummary *MigrationDuplicateSummary `json:"duplicateSummary,omitempty"`
	CanConfirmImport bool                       `json:"canConfirmImport"`
	ExpiresAt        *time.Time                 `json:"expiresAt,omitempty"`
}

type MigrationJob struct {
	ID               string
	UserID           string
	Phase            MigrationPhase
	Message          string
	MmbakPath        string
	XlsPath          string
	Counts           *MigrationJobCounts
	ValidationErrors []MigrationValidationError
	DuplicateSummary *MigrationDuplicateSummary
	CanConfirmImport bool
	ExpiresAt        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
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
