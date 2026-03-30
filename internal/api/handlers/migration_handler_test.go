package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"jarwise-backend/internal/db"
	"jarwise-backend/internal/models"
	"jarwise-backend/internal/service"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestHandleUpload_PersistsMigratedData(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "migration-test.db")
	dbConn, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("failed to initialize database: %v", err)
	}

	handler := NewMigrationHandler(service.NewMigrationService(dbConn))

	requestBody, contentType := buildMigrationMultipartBody(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/migrations/money-manager", requestBody)
	req.Header.Set("Content-Type", contentType)

	recorder := httptest.NewRecorder()
	handler.HandleUpload(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body: %s", recorder.Code, recorder.Body.String())
	}

	var response models.MigrationResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Fatalf("expected success status, got %+v", response)
	}

	if response.JobID == "" {
		t.Fatal("expected non-empty job ID")
	}

	assertTableCount(t, dbConn, "wallets", 2)
	assertTableCount(t, dbConn, "jars", 3)
	assertTableCount(t, dbConn, "transactions", 4)
}

func buildMigrationMultipartBody(t *testing.T) (io.Reader, string) {
	t.Helper()

	mmbakBytes, err := os.ReadFile(validMmbakPath(t))
	if err != nil {
		t.Fatalf("failed to read mmbak fixture: %v", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	mmbakWriter, err := writer.CreateFormFile("mmbak_file", "valid.mmbak")
	if err != nil {
		t.Fatalf("failed to create mmbak form field: %v", err)
	}
	if _, err := mmbakWriter.Write(mmbakBytes); err != nil {
		t.Fatalf("failed to write mmbak form field: %v", err)
	}

	xlsWriter, err := writer.CreateFormFile("xls_file", "valid.xls")
	if err != nil {
		t.Fatalf("failed to create xls form field: %v", err)
	}
	if _, err := xlsWriter.Write([]byte(validXlsFixture())); err != nil {
		t.Fatalf("failed to write xls form field: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	return &body, writer.FormDataContentType()
}

func validMmbakPath(t *testing.T) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve current file path")
	}

	return filepath.Join(filepath.Dir(currentFile), "..", "..", "parser", "testdata", "valid.mmbak")
}

func validXlsFixture() string {
	return `<html><body><table>
<tr><td>2025-01-15</td><td>Cash Wallet</td><td>Food</td><td>Lunch</td><td>-100.50</td></tr>
<tr><td>2025-01-20</td><td>Bank Account</td><td>Salary</td><td>Monthly Salary</td><td>50000.00</td></tr>
<tr><td>2025-01-22</td><td>Cash Wallet</td><td>Transport</td><td>Bus fare</td><td>-35.00</td></tr>
</table></body></html>`
}

func assertTableCount(t *testing.T, dbConn *sql.DB, table string, expected int) {
	t.Helper()

	var actual int
	if err := dbConn.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&actual); err != nil {
		t.Fatalf("failed to query %s count: %v", table, err)
	}

	if actual != expected {
		t.Fatalf("expected %d rows in %s, got %d", expected, table, actual)
	}
}
