# Luma Code Review Report

**Date:** 2026-02-04 19:12:32
**Files Reviewed:** ['go.mod', 'draft_pr_prompt.md', 'internal/db/sqlite.go', 'internal/api/router.go', 'draft_pr_body.md', 'internal/models/domain.go', '.gitignore', 'internal/repository/transaction_repository.go', 'internal/service/transaction_service.go', 'internal/repository/transaction_repository_test.go', '.luma_state.json', 'internal/api/handlers/transaction_handler.go', 'go.sum']

## 📝 Reviewer Feedback

There are two issues in the provided code that need to be addressed.

### 1. Critical: Incorrect JSON Struct Tags

**File:** `internal/models/domain.go` (and other model files like `internal/validator/models.go`)

**Problem:** The JSON struct tags are formatted with a space after the colon, for example, `` `json: "id"` ``. The standard Go `encoding/json` package will not parse this tag correctly. The correct format has no space: `` `json:"id"` ``. This will cause JSON serialization and deserialization to fail or produce unexpected results (e.g., using the Go field names like `ID` instead of the specified lowercase names like `id`).

**Fix:** Remove the space after the colon in all JSON struct tags across all model files.

**Example:**

```go
// In internal/models/domain.go

// Incorrect:
type Wallet struct {
	ID       string  `json: "id"`
	Name     string  `json: "name"`
	// ...
}

// Correct:
type Wallet struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	// ...
}
```

This correction needs to be applied to `Wallet`, `Jar`, and `Transaction` structs in `domain.go`, as well as the `ValidationResult` struct in `validator/models.go`.

### 2. Bug: Unhandled Error in Date Parsing Can Lead to Data Corruption

**File:** `internal/importer/importer.go`

**Problem:** In the `mapTransactions` function, the fallback logic for parsing dates ignores the error returned by `time.Parse`. If a date string does not match either of the expected formats, the error is discarded (`_`), and the `date` variable will contain the zero value for `time.Time` (`0001-01-01 00:00:00 UTC`). This incorrect date will be silently saved to the database, corrupting the transaction data.

**Code with issue:**
```go
// ...
date, err := time.Parse(layout, t.Date)
if err != nil {
    // Fallback for YYYY-MM-DD
    date, _ = time.Parse("2006-01-02", t.Date) // Error is ignored here
}
// ...
```

**Fix:** The error from the fallback parse must be handled. If parsing fails, you should at least log a warning and skip the problematic transaction, or return an error to stop the import process entirely.

**Example Fix (Skipping the invalid record):**
```go
// ...
for _, t := range mmTrans {
    date, err := time.Parse(layout, t.Date)
    if err != nil {
        // Fallback for YYYY-MM-DD
        var errFallback error
        date, errFallback = time.Parse("2006-01-02", t.Date)
        if errFallback != nil {
            // Log the failure and skip this transaction to prevent importing bad data
            fmt.Printf("WARN: Could not parse date string '%s' for transaction ID %s. Skipping transaction.\n", t.Date, t.ID)
            continue 
        }
    }
    // ... rest of the loop
}
// ...
```

## 🧪 Test Suggestions

*   **Empty `.mmbak` File:** Test the migration with a structurally valid `.mmbak` file that contains no accounts, categories, or transactions. The system should handle this gracefully, importing zero records without crashing or producing errors.
*   **Data with Referential Integrity Issues:** Test with a `.mmbak` file where a transaction record refers to an account ID or category ID that does not exist in the corresponding accounts or categories tables. The import process should either skip the invalid transaction with a clear log/error message or halt the entire process, but it must not crash.
*   **Validation Bypass with Invalid Data:** Create a file with data that would normally fail validation (e.g., an expense transaction with a positive amount, a transaction with a missing date). First, verify that the import is rejected when validation is enabled. Second, verify that the import proceeds (and potentially imports the malformed data) when the new validation bypass toggle is activated.

