# Luma Code Review Report

**Date:** 2026-02-04 10:20:12
**Files Reviewed:** ['internal/validator/models.go', 'internal/models/mm_data.go', 'internal/service/migration_service.go', 'internal/validator/validator.go', 'internal/parser/mmbak_parser.go', 'internal/models/domain.go', 'internal/importer/importer.go', '.luma_state.json']

## 📝 Reviewer Feedback

There are several critical issues in the provided code, ranging from logic errors that will lead to incorrect data migration to inconsistencies and typos.

### 1. Critical Logic Error: Incorrect Transaction Type Handling

The parser, validator, and importer are misaligned on how transaction types (Income, Expense, Transfer) are handled. This will lead to incorrect calculations and data mapping.

**File:** `internal/parser/mmbak_parser.go`

**Problem:** The code incorrectly classifies all non-income transactions as expenses. It queries for `DO_TYPE IN ('0', '1', '2')` but only has logic for `DO_TYPE == "1"`. Everything else becomes an expense (`t.Type = 0`). This means transfers are likely being misclassified as expenses, which will incorrectly inflate the `TotalExpense` and cause the validation against the XLS file to fail.

The importer (`internal/importer/importer.go`) expects `t.Type == 2` for transfers, but the parser **never** sets this value.

**Fix:** The parser logic must be updated to correctly identify and classify all transaction types, especially transfers. You need to confirm the exact values for `DO_TYPE` from the database schema, but based on the comments, a likely fix is:

```go
// In internal/parser/mmbak_parser.go, inside the transaction query loop

// ... (previous scan logic) ...

dt := doType.String
isTransfer := false

// NOTE: The DO_TYPE values ('0', '1', '2', '3'?) need to be confirmed from the DB schema.
// This is a likely mapping based on common patterns and your code comments.
switch dt {
case "1": // Income
    t.Type = 1
case "0", "2": // Assuming '0' and '2' are Expense types
    t.Type = 0
case "3": // Assuming '3' is Transfer
    t.Type = 2
    isTransfer = true
default:
    // Default to expense for safety, but you should log this case
    // to find any unhandled transaction types.
    fmt.Printf("Warning: Unknown DO_TYPE '%s' found for transaction %s. Classifying as expense.\n", dt, t.ID)
    t.Type = 0
}

result.Transactions = append(result.Transactions, t)

// Aggregate Totals, making sure to exclude transfers
if !isTransfer {
    if t.Type == 1 { // Income
        result.TotalIncome += t.Amount
    } else { // Expense
        // Also, ensure expense amounts are positive before adding.
        // If ZMONEY is negative for expenses, you must use math.Abs().
        result.TotalExpense += math.Abs(t.Amount)
    }
}
```
You also need to update the SQL query to include the `DO_TYPE` for transfers if it's not already included (e.g., if transfers are `DO_TYPE = '3'`).

---

### 2. Potential Logic Error: Expense Calculation

**File:** `internal/parser/mmbak_parser.go`

**Problem:** The line `result.TotalExpense += t.Amount` assumes that the `ZMONEY` column for expenses is a positive value. In many accounting systems, expenses are stored as negative numbers. If that's the case here, your `TotalExpense` will be calculated incorrectly (it will be subtracted from, not added to).

**Fix:** You must ensure you are summing the absolute values of expenses.

```go
// In internal/parser/mmbak_parser.go

import "math" // Add this import

// ... inside the transaction loop ...
} else { // Expense
    result.TotalExpense += math.Abs(t.Amount)
}
```

---

### 3. Inconsistent Error Handling

**File:** `internal/service/migration_service.go`

**Problem:** The `ProcessUpload` function handles errors inconsistently. For file-saving errors, it returns a proper Go error (`return nil, err`), which would typically result in a 500 Internal Server Error. However, for parsing errors, it returns a `200 OK` response with an error message in the JSON body (`return &models.MigrationResponse{...}, nil`). This makes the API's behavior unpredictable.

**Fix:** Standard practice is to always return an error from the service layer and let the HTTP handler layer decide on the response code and format.

```go
// In internal/service/migration_service.go

// Change this:
if err != nil {
    return &models.MigrationResponse{
        Status:  "error",
        Message: fmt.Sprintf("Failed to parse database: %v", err),
    }, nil
}

// To this (for both mmbak and xls parsing):
if err != nil {
    return nil, fmt.Errorf("failed to parse mmbak file: %w", err)
}
```

---

### 4. Typo: Incorrect JSON Struct Tags

**Problem:** Across all model files (`models.go`, `mm_data.go`, `domain.go`), the JSON struct tags are syntactically incorrect. You are using a colon (`:`) instead of quotes and a colon (`":"`). For example, `json:\"id\"` should be `json:"id"`. This will prevent the `encoding/json` package from correctly marshaling the structs into JSON with the desired field names.

**Fix:** Correct the struct tags in all model files.

**Example:**
```go
// In internal/validator/models.go

// Incorrect:
type ValidationResult struct {
	IsValid  bool     `json:\"is_valid\"`
	Errors   []string `json:\"errors\"`
    // ...
}

// Correct:
type ValidationResult struct {
	IsValid  bool     `json:"is_valid"`
	Errors   []string `json:"errors"`
    // ...
}
```
This fix needs to be applied to **all structs** in `internal/validator/models.go`, `internal/models/mm_data.go`, and `internal/models/domain.go`.

## 🧪 Test Suggestions

Here are 3 critical, edge-case test cases that should be added or verified for the given code changes:

*   **Test Case: Zero-Balance Transfer Validation.**
    *   **Scenario:** Provide an `.mmbak` and a matching `.xls` file that contain several "transfer" type transactions between two internal accounts, but no external income or expense transactions. The initial and final total balance across all accounts should remain the same.
    *   **Rationale:** This is a critical edge case because transfer transactions involve both a debit and a credit within the system. A naive validation summing all "expense-type" and "income-type" transactions would incorrectly flag this as a mismatch. This test ensures the logic correctly identifies and nets out internal transfers, preventing false negatives in the validation result.

*   **Test Case: Floating-Point Precision with High Transaction Volume.**
    *   **Scenario:** Use input files containing thousands of transactions with small, repeating fractional values (e.g., amounts like `0.10` or `1/3` which cannot be perfectly represented in binary floating-point). The total balance in both the `.mmbak` and `.xls` files should be mathematically identical but computationally prone to minute differences when summed using `float64`.
    *   **Rationale:** The use of `float64` for monetary values can lead to precision errors. A direct `==` comparison of the final calculated balances might fail due to these tiny discrepancies. This test verifies that the balance comparison logic is robust, likely by checking if the absolute difference is within a very small, acceptable tolerance (an epsilon) rather than being exactly zero.

*   **Test Case: Malformed or Empty Input File.**
    *   **Scenario:** Attempt to process an upload where the `.mmbak` file is empty (0 bytes), corrupted, or not a valid database file.
    *   **Rationale:** This tests the system's fundamental robustness and error handling at the file I/O and parsing stage. The service must not crash or enter an unrecoverable state. It needs to gracefully handle invalid input and provide a clear, user-facing error message within the `ValidationResult`, ensuring the system is resilient against user error or data corruption.

