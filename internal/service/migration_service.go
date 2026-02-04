package service

import (
	"context"
	"fmt"
	"io"
	"jarwise-backend/internal/models"
	"jarwise-backend/internal/parser"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"
)

// MigrationService defines the interface for handling migration logic
type MigrationService interface {
	ProcessUpload(ctx context.Context, mmbak, xls *multipart.FileHeader) (*models.MigrationResponse, error)
}

type migrationService struct {
	// Add repositories or parsers here later
}

// NewMigrationService creates a new instance of the migration service
func NewMigrationService() MigrationService {
	return &migrationService{}
}

// ProcessUpload handles the uploaded files, validates them, and starts the migration process
func (s *migrationService) ProcessUpload(ctx context.Context, mmbak, xls *multipart.FileHeader) (*models.MigrationResponse, error) {
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
		return &models.MigrationResponse{
			Status:  "error",
			Message: fmt.Sprintf("Failed to parse database: %v", err),
		}, nil // Return 200 with error status for UI handling? Or actual error
	}

	fmt.Printf("Parsed Data: %d Accounts, %d Categories, %d Transactions\n",
		len(parsedData.Accounts), len(parsedData.Categories), len(parsedData.Transactions))

	// Mock response for now (until Validation step)
	return &models.MigrationResponse{
		Status:  "success", // Changed to success for testing parser
		Message: fmt.Sprintf("Parsed %d transactions. Total Income: %.2f", len(parsedData.Transactions), parsedData.TotalIncome),
		JobID:   "job-uuid-123",
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
