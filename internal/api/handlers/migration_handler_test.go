package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"jarwise-backend/internal/auth"
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
	"time"
)

func TestMigrationJobWorkflow_PersistsMigratedData(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "migration-test.db")
	dbConn, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("failed to initialize database: %v", err)
	}
	seedTestUser(t, dbConn, "user-1")

	handler := NewMigrationHandler(service.NewMigrationService(dbConn))

	createBody, contentType := buildMigrationMultipartBody(t)
	createReq := withAuthenticatedUser(
		httptest.NewRequest(http.MethodPost, "/api/v1/migrations/money-manager/jobs", createBody),
		"user-1",
	)
	createReq.Header.Set("Content-Type", contentType)

	createRecorder := httptest.NewRecorder()
	handler.CreateJob(createRecorder, createReq)

	if createRecorder.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d with body: %s", createRecorder.Code, createRecorder.Body.String())
	}

	var createResponse models.MigrationJobStatusResponse
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &createResponse); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	if createResponse.JobID == "" {
		t.Fatal("expected non-empty job ID")
	}

	preview := waitForMigrationPhase(t, handler, "user-1", createResponse.JobID, models.MigrationPhasePreviewReady)
	if preview.Counts == nil {
		t.Fatal("expected preview counts")
	}
	if preview.Counts.Wallets != 2 || preview.Counts.Jars != 3 || preview.Counts.Transactions != 4 {
		t.Fatalf("unexpected preview counts: %+v", preview.Counts)
	}
	if !preview.CanConfirmImport {
		t.Fatal("expected preview to allow confirm")
	}

	confirmReq := withAuthenticatedUser(
		httptest.NewRequest(http.MethodPost, "/api/v1/migrations/money-manager/jobs/"+createResponse.JobID+"/confirm", nil),
		"user-1",
	)
	confirmRecorder := httptest.NewRecorder()
	handler.ConfirmJob(confirmRecorder, confirmReq)

	if confirmRecorder.Code != http.StatusOK {
		t.Fatalf("expected confirm status 200, got %d with body: %s", confirmRecorder.Code, confirmRecorder.Body.String())
	}

	completed := waitForMigrationPhase(t, handler, "user-1", createResponse.JobID, models.MigrationPhaseCompleted)
	if completed.Counts == nil || completed.Counts.Transactions != 4 {
		t.Fatalf("unexpected completed counts: %+v", completed.Counts)
	}

	assertTableCountForUser(t, dbConn, "wallets", "user-1", 2)
	assertTableCountForUser(t, dbConn, "jars", "user-1", 3)
	assertTableCountForUser(t, dbConn, "transactions", "user-1", 4)
	assertSourceRefCount(t, dbConn, "user-1", 9)
}

func waitForMigrationPhase(t *testing.T, handler *MigrationHandler, userID, jobID string, target models.MigrationPhase) models.MigrationJobStatusResponse {
	t.Helper()

	var last models.MigrationJobStatusResponse
	for range 100 {
		req := withAuthenticatedUser(
			httptest.NewRequest(http.MethodGet, "/api/v1/migrations/money-manager/jobs/"+jobID, nil),
			userID,
		)
		recorder := httptest.NewRecorder()
		handler.GetJob(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status 200 while polling, got %d with body: %s", recorder.Code, recorder.Body.String())
		}

		if err := json.Unmarshal(recorder.Body.Bytes(), &last); err != nil {
			t.Fatalf("failed to decode polling response: %v", err)
		}

		if last.Phase == target {
			return last
		}

		if last.Phase == models.MigrationPhaseFailed || last.Phase == models.MigrationPhaseDuplicateBlocked || last.Phase == models.MigrationPhaseExpired {
			t.Fatalf("migration entered unexpected terminal phase %s: %+v", last.Phase, last)
		}

		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("migration job %s did not reach phase %s, last response: %+v", jobID, target, last)
	return last
}

func withAuthenticatedUser(req *http.Request, userID string) *http.Request {
	user := &models.User{
		ID:    userID,
		Email: userID + "@example.com",
		Name:  "Test User",
	}
	return req.WithContext(auth.ContextWithUser(req.Context(), user))
}

func seedTestUser(t *testing.T, dbConn *sql.DB, userID string) {
	t.Helper()

	now := time.Now().UTC()
	_, err := dbConn.Exec(`
		INSERT INTO users (id, google_sub, email, name, avatar_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, userID, userID, userID+"@example.com", "Test User", "", now, now)
	if err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}
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
<tr><th>Date</th><th>Account</th><th>Category</th><th>Subcategory</th><th>Note</th><th>THB</th><th>Income/Expense</th><th>Description</th><th>Amount</th><th>Currency</th><th>Account</th></tr>
<tr><td>01/22/2025 08:15:00</td><td>Cash Wallet</td><td>Food</td><td></td><td>Lunch</td><td>100.50</td><td>Expense</td><td></td><td>100.50</td><td>THB</td><td>100.50</td></tr>
<tr><td>01/20/2025 09:00:00</td><td>Bank Account</td><td>Salary</td><td></td><td>Monthly Salary</td><td>50000.00</td><td>Income</td><td></td><td>50000.00</td><td>THB</td><td>50000.00</td></tr>
<tr><td>01/15/2025 18:00:00</td><td>Cash Wallet</td><td>Transport</td><td></td><td>Bus fare</td><td>35.00</td><td>Expense</td><td></td><td>35.00</td><td>THB</td><td>35.00</td></tr>
<tr><td>01/25/2025 12:00:00</td><td>Cash Wallet</td><td>Bank Account</td><td></td><td>Transfer to savings</td><td>5000.00</td><td>Transfer-Out</td><td></td><td>5000.00</td><td>THB</td><td>5000.00</td></tr>
</table></body></html>`
}

func assertTableCountForUser(t *testing.T, dbConn *sql.DB, table, userID string, expected int) {
	t.Helper()

	var actual int
	if err := dbConn.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE user_id = ?", userID).Scan(&actual); err != nil {
		t.Fatalf("failed to query %s count: %v", table, err)
	}

	if actual != expected {
		t.Fatalf("expected %d rows in %s for user %s, got %d", expected, table, userID, actual)
	}
}

func assertSourceRefCount(t *testing.T, dbConn *sql.DB, userID string, expected int) {
	t.Helper()

	var actual int
	if err := dbConn.QueryRow("SELECT COUNT(*) FROM migration_source_refs WHERE user_id = ?", userID).Scan(&actual); err != nil {
		t.Fatalf("failed to query migration_source_refs count: %v", err)
	}
	if actual != expected {
		t.Fatalf("expected %d source refs, got %d", expected, actual)
	}
}
