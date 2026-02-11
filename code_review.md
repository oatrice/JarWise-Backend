# Luma Code Review Report

**Date:** 2026-02-11 22:55:09
**Files Reviewed:** ['internal/service/report_service_test.go', 'internal/parser/testdata/non_existent.mmbak', 'internal/repository/transaction_repository.go', 'internal/parser/mmbak_parser.go', 'code_review.md', 'internal/parser/testdata/corrupt.mmbak', 'internal/api/handlers/report_handler.go', 'internal/api/router.go', 'internal/service/report_service.go', 'internal/parser/testdata/generate_test_files.py', 'internal/models/report.go', 'internal/parser/mmbak_parser_test.go']

## 📝 Reviewer Feedback

There is a logic error in the `mmbak_parser.go` file.

**File:** `internal/parser/mmbak_parser.go`

**Issue:**
The SQL query used to fetch transactions from the `INOUTCOME` table is inconsistent with the Go code that processes the transaction types. The query only selects transactions where `DO_TYPE` is '0', '1', or '2', but the `switch` statement that follows has a case to handle `DO_TYPE` '3' (for transfers).

Because of this discrepancy, any transaction marked as a transfer with `DO_TYPE = '3'` in the source `.mmbak` file will be ignored by the query and never parsed, leading to incomplete data processing.

**Fix:**
Update the SQL query to include `'3'` in the `WHERE` clause to ensure transfer transactions are selected from the database.

**Change this:**
```go
// in internal/parser/mmbak_parser.go, line 79

	transRows, err := db.Query(`
        SELECT uid, ZDATE, ZMONEY, DO_TYPE, ZCONTENT, categoryUid, assetUid 
        FROM INOUTCOME 
        WHERE DO_TYPE IN ('0', '1', '2') OR DO_TYPE IS NULL
    `)
```

**To this:**
```go
// in internal/parser/mmbak_parser.go, line 79

	transRows, err := db.Query(`
        SELECT uid, ZDATE, ZMONEY, DO_TYPE, ZCONTENT, categoryUid, assetUid 
        FROM INOUTCOME 
        WHERE DO_TYPE IN ('0', '1', '2', '3') OR DO_TYPE IS NULL
    `)
```

## 🧪 Test Suggestions

Here are 3 critical, edge-case test cases that should be added or verified for the `ReportService`:

*   **Test with an invalid date range where the start date is after the end date.** The current tests all use valid date ranges. This edge case checks for proper input validation. The service should gracefully handle this by returning an error, rather than an empty or incorrect report.

*   **Test with filters that result in zero transactions.** This scenario tests the filtering logic when there is no data to return. For example, use a valid date range but filter by a `JarID` and a `WalletID` that never appear together on the same transaction. The expected outcome is a valid, empty report (e.g., `TransactionCount: 0`), not an error.

*   **Test date range boundaries.** Create a test where the `StartDate` and `EndDate` are set to the exact timestamp of a single known transaction. This verifies that the date range filtering is inclusive (`>= start` and `<= end`) and correctly handles transactions that fall precisely on the boundary, which can be a common source of off-by-one errors.

