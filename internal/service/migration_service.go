package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"jarwise-backend/internal/models"
	"jarwise-backend/internal/parser"
	"jarwise-backend/internal/validator"
	"log"
	"math"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const migrationJobTTL = 24 * time.Hour

var (
	ErrMigrationJobNotFound = errors.New("migration job not found")
	ErrMigrationJobConflict = errors.New("migration job is not in a confirmable state")
)

type MigrationService interface {
	CreateJob(ctx context.Context, userID string, mmbak, xls *multipart.FileHeader) (*models.MigrationJobStatusResponse, error)
	GetJob(ctx context.Context, userID, jobID string) (*models.MigrationJobStatusResponse, error)
	ConfirmJob(ctx context.Context, userID, jobID string) (*models.MigrationJobStatusResponse, error)
}

type migrationService struct {
	db        *sql.DB
	validator *validator.Validator
	clock     func() time.Time
}

func NewMigrationService(db *sql.DB) MigrationService {
	return &migrationService{
		db:        db,
		validator: validator.NewValidator(),
		clock: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *migrationService) CreateJob(ctx context.Context, userID string, mmbak, xls *multipart.FileHeader) (*models.MigrationJobStatusResponse, error) {
	if err := s.cleanupExpiredJobs(ctx); err != nil {
		return nil, err
	}

	userID = normalizedServiceUserID(userID)
	jobID := uuid.NewString()
	now := s.clock()
	expiresAt := now.Add(migrationJobTTL)

	mmbakPath, xlsPath, err := saveJobFiles(jobID, mmbak, xls)
	if err != nil {
		return nil, err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO migration_jobs (
			id, user_id, phase, message, mmbak_path, xls_path,
			counts_json, validation_errors_json, duplicate_summary_json,
			can_confirm_import, expires_at, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		jobID,
		userID,
		models.MigrationPhaseValidating,
		"Files uploaded. Validation is in progress.",
		mmbakPath,
		xlsPath,
		nil,
		nil,
		nil,
		false,
		expiresAt,
		now,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create migration job: %w", err)
	}

	log.Printf("[migration:%s] created validation job for user=%s", jobID, userID)

	go s.runValidation(jobID, userID)

	return &models.MigrationJobStatusResponse{
		JobID:            jobID,
		Phase:            models.MigrationPhaseValidating,
		Message:          "Files uploaded. Validation is in progress.",
		CanConfirmImport: false,
		ExpiresAt:        &expiresAt,
	}, nil
}

func (s *migrationService) GetJob(ctx context.Context, userID, jobID string) (*models.MigrationJobStatusResponse, error) {
	if err := s.cleanupExpiredJobs(ctx); err != nil {
		return nil, err
	}

	job, err := s.loadJob(ctx, normalizedServiceUserID(userID), jobID)
	if err != nil {
		return nil, err
	}

	return migrationJobToStatus(job), nil
}

func (s *migrationService) ConfirmJob(ctx context.Context, userID, jobID string) (*models.MigrationJobStatusResponse, error) {
	if err := s.cleanupExpiredJobs(ctx); err != nil {
		return nil, err
	}

	userID = normalizedServiceUserID(userID)
	job, err := s.loadJob(ctx, userID, jobID)
	if err != nil {
		return nil, err
	}

	if job.ExpiresAt != nil && job.ExpiresAt.Before(s.clock()) {
		if err := s.markJobExpired(ctx, job); err != nil {
			return nil, err
		}
		return nil, ErrMigrationJobConflict
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE migration_jobs
		SET phase = ?, message = ?, can_confirm_import = ?, updated_at = ?
		WHERE id = ? AND user_id = ? AND phase = ?
	`,
		models.MigrationPhaseImporting,
		"Import is in progress.",
		false,
		s.clock(),
		jobID,
		userID,
		models.MigrationPhasePreviewReady,
	)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, ErrMigrationJobConflict
	}

	go s.runImport(jobID, userID)

	job.Phase = models.MigrationPhaseImporting
	job.Message = "Import is in progress."
	job.CanConfirmImport = false

	return migrationJobToStatus(job), nil
}

func (s *migrationService) runValidation(jobID, userID string) {
	ctx := context.Background()
	job, err := s.loadJob(ctx, userID, jobID)
	if err != nil {
		log.Printf("[migration:%s] failed to load job for validation: %v", jobID, err)
		return
	}

	parsedData, validationErrors, duplicateSummary, counts, err := s.validateJobFiles(job)
	if err != nil {
		log.Printf("[migration:%s] validation failed: %v", jobID, err)
		_ = s.persistJobState(ctx, jobID, userID, models.MigrationPhaseFailed, "Validation failed.", counts, append(validationErrors, models.MigrationValidationError{
			Code:    "validation_failed",
			Message: err.Error(),
		}), duplicateSummary, false)
		return
	}

	if len(validationErrors) > 0 {
		log.Printf("[migration:%s] validation failed with %d errors", jobID, len(validationErrors))
		_ = s.persistJobState(ctx, jobID, userID, models.MigrationPhaseFailed, "Validation failed.", counts, validationErrors, duplicateSummary, false)
		return
	}

	if hasDuplicates(duplicateSummary) {
		log.Printf("[migration:%s] duplicate block detected for user=%s", jobID, userID)
		_ = s.persistJobState(ctx, jobID, userID, models.MigrationPhaseDuplicateBlocked, "Duplicate data was detected for this account.", counts, nil, duplicateSummary, false)
		return
	}

	log.Printf("[migration:%s] validation preview ready accounts=%d categories=%d transactions=%d", jobID, len(parsedData.Accounts), len(parsedData.Categories), len(parsedData.Transactions))
	_ = s.persistJobState(ctx, jobID, userID, models.MigrationPhasePreviewReady, "Validation complete. Ready to import.", counts, nil, duplicateSummary, true)
}

func (s *migrationService) runImport(jobID, userID string) {
	ctx := context.Background()
	job, err := s.loadJob(ctx, userID, jobID)
	if err != nil {
		log.Printf("[migration:%s] failed to load job for import: %v", jobID, err)
		return
	}

	parsedData, validationErrors, duplicateSummary, counts, err := s.validateJobFiles(job)
	if err != nil {
		log.Printf("[migration:%s] import validation failed: %v", jobID, err)
		_ = s.persistJobState(ctx, jobID, userID, models.MigrationPhaseFailed, "Import failed during validation.", counts, append(validationErrors, models.MigrationValidationError{
			Code:    "validation_failed",
			Message: err.Error(),
		}), duplicateSummary, false)
		return
	}

	if len(validationErrors) > 0 {
		_ = s.persistJobState(ctx, jobID, userID, models.MigrationPhaseFailed, "Import failed during validation.", counts, validationErrors, duplicateSummary, false)
		return
	}

	if hasDuplicates(duplicateSummary) {
		_ = s.persistJobState(ctx, jobID, userID, models.MigrationPhaseDuplicateBlocked, "Duplicate data was detected for this account.", counts, nil, duplicateSummary, false)
		return
	}

	if err := s.importParsedData(ctx, userID, parsedData, counts); err != nil {
		log.Printf("[migration:%s] import failed: %v", jobID, err)
		_ = s.persistJobState(ctx, jobID, userID, models.MigrationPhaseFailed, "Import failed.", counts, []models.MigrationValidationError{{
			Code:    "import_failed",
			Message: err.Error(),
		}}, nil, false)
		return
	}

	log.Printf("[migration:%s] import completed successfully for user=%s", jobID, userID)
	_ = s.persistJobState(ctx, jobID, userID, models.MigrationPhaseCompleted, "Import completed successfully.", counts, nil, nil, false)
	_ = cleanupJobFiles(job)
}

func (s *migrationService) validateJobFiles(job *models.MigrationJob) (*models.ParsedData, []models.MigrationValidationError, *models.MigrationDuplicateSummary, *models.MigrationJobCounts, error) {
	mmParser := parser.NewMmbakParser()
	parsedData, err := mmParser.Parse(job.MmbakPath)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to parse mmbak: %w", err)
	}

	log.Printf(
		"[migration:%s] parsed mmbak accounts=%d categories=%d transactions=%d total_income=%.2f total_expense=%.2f",
		job.ID,
		len(parsedData.Accounts),
		len(parsedData.Categories),
		len(parsedData.Transactions),
		parsedData.TotalIncome,
		parsedData.TotalExpense,
	)

	xlsParser := parser.NewXlsParser()
	xlsData, err := xlsParser.Parse(job.XlsPath)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to parse xls: %w", err)
	}

	log.Printf(
		"[migration:%s] parsed xls transactions=%d total_income=%.2f total_expense=%.2f",
		job.ID,
		len(xlsData.Transactions),
		xlsData.TotalIncome,
		xlsData.TotalExpense,
	)

	counts := &models.MigrationJobCounts{
		Wallets:      len(parsedData.Accounts),
		Jars:         len(parsedData.Categories),
		Transactions: len(parsedData.Transactions),
		TotalIncome:  parsedData.TotalIncome,
		TotalExpense: parsedData.TotalExpense,
	}

	validationResult := s.validator.Validate(parsedData, xlsData)
	validationErrors := make([]models.MigrationValidationError, 0, len(validationResult.Errors))
	for _, message := range validationResult.Errors {
		validationErrors = append(validationErrors, models.MigrationValidationError{
			Code:    "validation_error",
			Message: message,
		})
	}

	integrityErrors := s.validator.ValidateIntegrity(parsedData)
	for _, message := range integrityErrors {
		validationErrors = append(validationErrors, models.MigrationValidationError{
			Code:    "integrity_error",
			Message: message,
		})
	}

	duplicateSummary, err := s.detectDuplicates(context.Background(), job.UserID, parsedData)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return parsedData, validationErrors, duplicateSummary, counts, nil
}

func (s *migrationService) detectDuplicates(ctx context.Context, userID string, data *models.ParsedData) (*models.MigrationDuplicateSummary, error) {
	summary := &models.MigrationDuplicateSummary{
		Wallets:      []models.MigrationDuplicateItem{},
		Jars:         []models.MigrationDuplicateItem{},
		Transactions: []models.MigrationDuplicateItem{},
	}

	seen := make(map[string]struct{})

	for _, account := range data.Accounts {
		item, duplicate, err := s.lookupDuplicate(ctx, userID, "wallet", account.ID, fingerprintWallet(account), account.Name)
		if err != nil {
			return nil, err
		}
		if duplicate {
			key := "wallet:" + item.SourceID + ":" + item.MatchedBy
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				summary.Wallets = append(summary.Wallets, item)
			}
		}
	}

	for _, category := range data.Categories {
		item, duplicate, err := s.lookupDuplicate(ctx, userID, "jar", category.ID, fingerprintJar(category), category.Name)
		if err != nil {
			return nil, err
		}
		if duplicate {
			key := "jar:" + item.SourceID + ":" + item.MatchedBy
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				summary.Jars = append(summary.Jars, item)
			}
		}
	}

	for _, transaction := range data.Transactions {
		displayName := transaction.Note
		if displayName == "" {
			displayName = fmt.Sprintf("%s %.2f", transaction.Date, transaction.Amount)
		}

		item, duplicate, err := s.lookupDuplicate(ctx, userID, "transaction", transaction.ID, fingerprintTransaction(transaction), displayName)
		if err != nil {
			return nil, err
		}
		if duplicate {
			key := "transaction:" + item.SourceID + ":" + item.MatchedBy
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				summary.Transactions = append(summary.Transactions, item)
			}
		}
	}

	return summary, nil
}

func (s *migrationService) lookupDuplicate(ctx context.Context, userID, entityType, sourceID, fingerprint, displayName string) (models.MigrationDuplicateItem, bool, error) {
	item := models.MigrationDuplicateItem{
		SourceID:    sourceID,
		DisplayName: displayName,
		Fingerprint: fingerprint,
	}

	var matchedSourceID string
	err := s.db.QueryRowContext(ctx, `
		SELECT source_id
		FROM migration_source_refs
		WHERE user_id = ? AND source_system = 'money_manager' AND entity_type = ? AND source_id = ?
		LIMIT 1
	`, userID, entityType, sourceID).Scan(&matchedSourceID)
	if err == nil {
		item.MatchedBy = "source_id"
		return item, true, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return item, false, err
	}

	var matchedFingerprint string
	err = s.db.QueryRowContext(ctx, `
		SELECT fingerprint
		FROM migration_source_refs
		WHERE user_id = ? AND entity_type = ? AND fingerprint = ?
		LIMIT 1
	`, userID, entityType, fingerprint).Scan(&matchedFingerprint)
	if err == nil {
		item.MatchedBy = "fingerprint"
		return item, true, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return item, false, err
	}

	return item, false, nil
}

func (s *migrationService) importParsedData(ctx context.Context, userID string, data *models.ParsedData, counts *models.MigrationJobCounts) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	walletIDs := make(map[string]string, len(data.Accounts))
	for _, account := range data.Accounts {
		newID := uuid.NewString()
		walletIDs[account.ID] = newID
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO wallets (id, user_id, name, currency, balance, type)
			VALUES (?, ?, ?, ?, ?, ?)
		`, newID, userID, account.Name, account.Currency, account.Balance, "general"); err != nil {
			return fmt.Errorf("failed to insert wallet %s: %w", account.ID, err)
		}
		if err := s.insertSourceRefTx(ctx, tx, userID, "wallet", account.ID, fingerprintWallet(account), account.Name, newID); err != nil {
			return err
		}
	}

	jarIDs := make(map[string]string, len(data.Categories))
	for _, category := range data.Categories {
		jarIDs[category.ID] = uuid.NewString()
	}
	for _, category := range data.Categories {
		parentID := ""
		if category.ParentID != "" {
			parentID = jarIDs[category.ParentID]
		}
		jarType := "expense"
		if category.Type == 1 {
			jarType = "income"
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO jars (id, user_id, name, type, parent_id, wallet_id, icon, color)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, jarIDs[category.ID], userID, category.Name, jarType, nullableString(parentID), nil, "", ""); err != nil {
			return fmt.Errorf("failed to insert jar %s: %w", category.ID, err)
		}
		if err := s.insertSourceRefTx(ctx, tx, userID, "jar", category.ID, fingerprintJar(category), category.Name, jarIDs[category.ID]); err != nil {
			return err
		}
	}

	for _, mmTx := range data.Transactions {
		date, err := parseMMTransactionDate(mmTx.Date)
		if err != nil {
			return fmt.Errorf("failed to parse transaction date for %s: %w", mmTx.ID, err)
		}

		txType := "expense"
		if mmTx.Type == 1 {
			txType = "income"
		} else if mmTx.Type == 2 {
			txType = "transfer"
		}

		newID := uuid.NewString()
		walletID, ok := walletIDs[mmTx.AccountID]
		if !ok {
			return fmt.Errorf("unknown wallet source id %s", mmTx.AccountID)
		}

		var jarID interface{}
		if mmTx.CategoryID != "" {
			mappedJarID, ok := jarIDs[mmTx.CategoryID]
			if !ok {
				return fmt.Errorf("unknown category source id %s", mmTx.CategoryID)
			}
			jarID = mappedJarID
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO transactions (id, user_id, amount, description, date, type, wallet_id, jar_id, related_transaction_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, newID, userID, mmTx.Amount, mmTx.Note, date.UTC(), txType, walletID, jarID, nil); err != nil {
			return fmt.Errorf("failed to insert transaction %s: %w", mmTx.ID, err)
		}

		if err := s.insertSourceRefTx(ctx, tx, userID, "transaction", mmTx.ID, fingerprintTransaction(mmTx), mmTx.Note, newID); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Printf(
		"[migration-import] committed wallets=%d jars=%d transactions=%d skipped_transactions=%d user=%s",
		counts.Wallets,
		counts.Jars,
		counts.Transactions,
		0,
		userID,
	)

	return nil
}

func (s *migrationService) insertSourceRefTx(ctx context.Context, tx *sql.Tx, userID, entityType, sourceID, fingerprint, displayName, importedRecordID string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO migration_source_refs (
			id, user_id, source_system, entity_type, source_id, fingerprint, display_name, imported_record_id, created_at
		)
		VALUES (?, ?, 'money_manager', ?, ?, ?, ?, ?, ?)
	`, uuid.NewString(), userID, entityType, sourceID, fingerprint, displayName, importedRecordID, s.clock())
	if err != nil {
		return fmt.Errorf("failed to insert source ref for %s %s: %w", entityType, sourceID, err)
	}
	return nil
}

func (s *migrationService) persistJobState(ctx context.Context, jobID, userID string, phase models.MigrationPhase, message string, counts *models.MigrationJobCounts, validationErrors []models.MigrationValidationError, duplicateSummary *models.MigrationDuplicateSummary, canConfirmImport bool) error {
	countsJSON, err := marshalJSONText(counts)
	if err != nil {
		return err
	}
	validationErrorsJSON, err := marshalJSONText(validationErrors)
	if err != nil {
		return err
	}
	duplicateSummaryJSON, err := marshalJSONText(duplicateSummary)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE migration_jobs
		SET phase = ?, message = ?, counts_json = ?, validation_errors_json = ?, duplicate_summary_json = ?, can_confirm_import = ?, updated_at = ?
		WHERE id = ? AND user_id = ?
	`, phase, message, countsJSON, validationErrorsJSON, duplicateSummaryJSON, canConfirmImport, s.clock(), jobID, userID)
	return err
}

func (s *migrationService) loadJob(ctx context.Context, userID, jobID string) (*models.MigrationJob, error) {
	job := &models.MigrationJob{}
	var (
		countsJSON           sql.NullString
		validationErrorsJSON sql.NullString
		duplicateSummaryJSON sql.NullString
		expiresAt            sql.NullTime
	)

	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, phase, message, mmbak_path, xls_path, counts_json, validation_errors_json, duplicate_summary_json, can_confirm_import, expires_at, created_at, updated_at
		FROM migration_jobs
		WHERE id = ? AND user_id = ?
	`, jobID, userID).Scan(
		&job.ID,
		&job.UserID,
		&job.Phase,
		&job.Message,
		&job.MmbakPath,
		&job.XlsPath,
		&countsJSON,
		&validationErrorsJSON,
		&duplicateSummaryJSON,
		&job.CanConfirmImport,
		&expiresAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMigrationJobNotFound
		}
		return nil, err
	}

	if countsJSON.Valid && countsJSON.String != "" {
		job.Counts = &models.MigrationJobCounts{}
		if err := json.Unmarshal([]byte(countsJSON.String), job.Counts); err != nil {
			return nil, err
		}
	}
	if validationErrorsJSON.Valid && validationErrorsJSON.String != "" {
		if err := json.Unmarshal([]byte(validationErrorsJSON.String), &job.ValidationErrors); err != nil {
			return nil, err
		}
	}
	if duplicateSummaryJSON.Valid && duplicateSummaryJSON.String != "" {
		job.DuplicateSummary = &models.MigrationDuplicateSummary{}
		if err := json.Unmarshal([]byte(duplicateSummaryJSON.String), job.DuplicateSummary); err != nil {
			return nil, err
		}
	}
	if expiresAt.Valid {
		value := expiresAt.Time.UTC()
		job.ExpiresAt = &value
	}

	return job, nil
}

func (s *migrationService) cleanupExpiredJobs(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, phase, message, mmbak_path, xls_path, counts_json, validation_errors_json, duplicate_summary_json, can_confirm_import, expires_at, created_at, updated_at
		FROM migration_jobs
		WHERE expires_at IS NOT NULL AND expires_at < ? AND phase NOT IN (?, ?)
	`, s.clock(), models.MigrationPhaseCompleted, models.MigrationPhaseExpired)
	if err != nil {
		return err
	}
	defer rows.Close()

	var jobs []*models.MigrationJob
	for rows.Next() {
		job := &models.MigrationJob{}
		var (
			countsJSON           sql.NullString
			validationErrorsJSON sql.NullString
			duplicateSummaryJSON sql.NullString
			expiresAt            sql.NullTime
		)
		if err := rows.Scan(
			&job.ID,
			&job.UserID,
			&job.Phase,
			&job.Message,
			&job.MmbakPath,
			&job.XlsPath,
			&countsJSON,
			&validationErrorsJSON,
			&duplicateSummaryJSON,
			&job.CanConfirmImport,
			&expiresAt,
			&job.CreatedAt,
			&job.UpdatedAt,
		); err != nil {
			return err
		}
		if expiresAt.Valid {
			value := expiresAt.Time.UTC()
			job.ExpiresAt = &value
		}
		jobs = append(jobs, job)
	}

	for _, job := range jobs {
		if err := s.markJobExpired(ctx, job); err != nil {
			return err
		}
	}

	return rows.Err()
}

func (s *migrationService) markJobExpired(ctx context.Context, job *models.MigrationJob) error {
	if err := cleanupJobFiles(job); err != nil {
		return err
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE migration_jobs
		SET phase = ?, message = ?, mmbak_path = '', xls_path = '', can_confirm_import = ?, updated_at = ?
		WHERE id = ? AND user_id = ?
	`, models.MigrationPhaseExpired, "Migration preview expired.", false, s.clock(), job.ID, job.UserID)
	return err
}

func saveJobFiles(jobID string, mmbak, xls *multipart.FileHeader) (string, string, error) {
	jobDir := filepath.Join(os.TempDir(), "jarwise-migration-jobs", jobID)
	if err := os.MkdirAll(jobDir, 0o755); err != nil {
		return "", "", err
	}

	mmbakPath := filepath.Join(jobDir, sanitizeFileName(mmbak.Filename))
	if err := saveMultipartFile(mmbak, mmbakPath); err != nil {
		return "", "", err
	}

	xlsPath := filepath.Join(jobDir, sanitizeFileName(xls.Filename))
	if err := saveMultipartFile(xls, xlsPath); err != nil {
		return "", "", err
	}

	return mmbakPath, xlsPath, nil
}

func cleanupJobFiles(job *models.MigrationJob) error {
	paths := []string{job.MmbakPath, job.XlsPath}
	seenDirs := make(map[string]struct{})
	for _, path := range paths {
		if path == "" {
			continue
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		dir := filepath.Dir(path)
		if _, ok := seenDirs[dir]; ok {
			continue
		}
		seenDirs[dir] = struct{}{}
		if err := os.Remove(dir); err != nil && !errors.Is(err, os.ErrNotExist) {
			if !errors.Is(err, os.ErrPermission) {
				return err
			}
		}
	}
	return nil
}

func saveMultipartFile(fileHeader *multipart.FileHeader, destPath string) error {
	src, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func migrationJobToStatus(job *models.MigrationJob) *models.MigrationJobStatusResponse {
	return &models.MigrationJobStatusResponse{
		JobID:            job.ID,
		Phase:            job.Phase,
		Message:          job.Message,
		Counts:           job.Counts,
		ValidationErrors: job.ValidationErrors,
		DuplicateSummary: job.DuplicateSummary,
		CanConfirmImport: job.CanConfirmImport,
		ExpiresAt:        job.ExpiresAt,
	}
}

func parseMMTransactionDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("unsupported date format %q", value)
	}

	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02",
		time.RFC3339,
		time.RFC3339Nano,
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, nil
		}
	}

	if unixTime, ok := parseUnixLikeTimestamp(value); ok {
		return unixTime, nil
	}

	return time.Time{}, fmt.Errorf("unsupported date format %q", value)
}

func parseUnixLikeTimestamp(value string) (time.Time, bool) {
	raw, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}, false
	}

	absValue := math.Abs(float64(raw))
	switch {
	case absValue >= 1e18:
		return time.Unix(0, raw).UTC(), true
	case absValue >= 1e15:
		return time.UnixMicro(raw).UTC(), true
	case absValue >= 1e12:
		return time.UnixMilli(raw).UTC(), true
	default:
		return time.Unix(raw, 0).UTC(), true
	}
}

func fingerprintWallet(account models.AccountDTO) string {
	return fingerprintStrings(normalize(account.Name), normalize(account.Currency), fmt.Sprintf("%.2f", account.Balance))
}

func fingerprintJar(category models.CategoryDTO) string {
	return fingerprintStrings(normalize(category.Name), fmt.Sprintf("%d", category.Type), normalize(category.ParentID))
}

func fingerprintTransaction(transaction models.TransactionDTO) string {
	return fingerprintStrings(
		normalize(transaction.Date),
		fmt.Sprintf("%.2f", transaction.Amount),
		fmt.Sprintf("%d", transaction.Type),
		normalize(transaction.CategoryID),
		normalize(transaction.AccountID),
		normalize(transaction.ToAccountID),
		normalize(transaction.Note),
	)
}

func fingerprintStrings(parts ...string) string {
	hash := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(hash[:])
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func nullableString(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}

func marshalJSONText(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	bytes, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if string(bytes) == "null" {
		return nil, nil
	}
	return string(bytes), nil
}

func hasDuplicates(summary *models.MigrationDuplicateSummary) bool {
	if summary == nil {
		return false
	}
	return len(summary.Wallets) > 0 || len(summary.Jars) > 0 || len(summary.Transactions) > 0
}

func sanitizeFileName(name string) string {
	name = filepath.Base(name)
	if name == "." || name == string(filepath.Separator) {
		return uuid.NewString()
	}
	return name
}

func normalizedServiceUserID(userID string) string {
	if userID == "" {
		return models.DefaultLocalUserID
	}
	return userID
}
