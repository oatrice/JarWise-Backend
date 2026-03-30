package service

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"jarwise-backend/internal/importer"
	"jarwise-backend/internal/models"
	"jarwise-backend/internal/parser"
	"jarwise-backend/internal/validator"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// TOGGLE: Set to true to allow import even if validation fails
const BypassValidation = false

// MigrationService defines the interface for handling migration logic
type MigrationService interface {
	ProcessUpload(ctx context.Context, mmbak, xls *multipart.FileHeader) (*models.MigrationResponse, error)
}

type migrationService struct {
	db *sql.DB
}

// NewMigrationService creates a new instance of the migration service
func NewMigrationService(db *sql.DB) MigrationService {
	return &migrationService{db: db}
}

// ProcessUpload handles the uploaded files, validates them, and starts the migration process
func (s *migrationService) ProcessUpload(ctx context.Context, mmbak, xls *multipart.FileHeader) (*models.MigrationResponse, error) {
	if s.db == nil {
		return nil, fmt.Errorf("migration database is not configured")
	}

	jobID := uuid.NewString()
	log.Printf(
		"[migration:%s] upload received mmbak=%s(%d bytes) xls=%s(%d bytes)",
		jobID,
		mmbak.Filename,
		mmbak.Size,
		xls.Filename,
		xls.Size,
	)

	// 1. Save .mmbak to temp file
	tempDir := os.TempDir()
	mmbakPath := filepath.Join(tempDir, fmt.Sprintf("upload-%d.mmbak", time.Now().UnixNano()))

	if err := saveMultipartFile(mmbak, mmbakPath); err != nil {
		return nil, fmt.Errorf("failed to save temp mmbak file: %w", err)
	}
	defer os.Remove(mmbakPath) // Clean up

	// 2. Parse .mmbak
	mmParser := parser.NewMmbakParser()
	parsedData, err := mmParser.Parse(mmbakPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database: %w", err)
	}

	log.Printf(
		"[migration:%s] parsed mmbak accounts=%d categories=%d transactions=%d total_income=%.2f total_expense=%.2f",
		jobID,
		len(parsedData.Accounts),
		len(parsedData.Categories),
		len(parsedData.Transactions),
		parsedData.TotalIncome,
		parsedData.TotalExpense,
	)

	// 3. Parse .xls (HTML)
	// Save .xls to temp
	xlsPath := filepath.Join(tempDir, fmt.Sprintf("upload-%d.xls", time.Now().UnixNano()))
	if err := saveMultipartFile(xls, xlsPath); err != nil {
		return nil, fmt.Errorf("failed to save temp xls file: %w", err)
	}
	defer os.Remove(xlsPath)

	xlsParser := parser.NewXlsParser()
	xlsData, err := xlsParser.Parse(xlsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse XLS report: %w", err)
	}

	log.Printf(
		"[migration:%s] parsed xls transactions=%d total_income=%.2f total_expense=%.2f",
		jobID,
		len(xlsData.Transactions),
		xlsData.TotalIncome,
		xlsData.TotalExpense,
	)

	// 4. Validate
	v := validator.NewValidator()
	validationResult := v.Validate(parsedData, xlsData)

	status := "preview" // Ready for preview if valid
	msg := "Validation successful"

	if !validationResult.IsValid {
		if BypassValidation {
			log.Printf("[migration:%s] validation failed but continuing because bypass is enabled errors=%v warnings=%v", jobID, validationResult.Errors, validationResult.Warnings)
			msg = "Import successful (with validation warnings)"
		} else {
			log.Printf("[migration:%s] validation failed errors=%v warnings=%v", jobID, validationResult.Errors, validationResult.Warnings)
			return &models.MigrationResponse{
				Status:  "error",
				Message: "Validation failed. Discrepancies found.",
				JobID:   jobID,
			}, nil
		}
	} else {
		log.Printf("[migration:%s] validation passed warnings=%v", jobID, validationResult.Warnings)
		msg = "Import successful!"
	}

	// 5. Import (Only if valid or bypassed)
	importer := importer.NewImporter(s.db)
	if err := importer.ImportData(parsedData); err != nil {
		log.Printf("[migration:%s] import failed: %v", jobID, err)
		return &models.MigrationResponse{
			Status:  "error",
			Message: fmt.Sprintf("Import failed: %v", err),
			JobID:   jobID,
		}, nil
	}

	status = "success"
	log.Printf("[migration:%s] import completed successfully", jobID)

	return &models.MigrationResponse{
		Status:  status,
		Message: msg,
		JobID:   jobID,
	}, nil
}

// Helper to save multipart file
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
