# Luma Code Review Report

**Date:** 2026-02-12 22:56:44
**Files Reviewed:** ['internal/service/graph_service.go', 'internal/service/chart_service_test.go', 'internal/repository/transaction_repository.go', 'internal/service/chart_service.go', 'internal/models/graph.go', 'internal/api/handlers/graph_handler.go', 'internal/models/chart.go', 'internal/api/router.go', '.gitignore', 'internal/api/handlers/chart_handler.go', 'internal/models/errors.go', 'go.mod']

## 📝 Reviewer Feedback

There is a consistent bug across all model definitions and some handler-specific response structs regarding JSON field tagging. The backslashes are incorrectly included within the struct tag strings.

**Problem:**

The struct tags for JSON serialization are malformed. For example, in `internal/models/graph.go`, the tag is written as `` `json:\"label\"` ``. The Go compiler interprets this as a raw string literal containing the characters `j`, `s`, `o`, `n`, `:`, `\`, `"`, `l`, `a`, `b`, `e`, `l`, `\`, `"`.

The `encoding/json` package expects the tag format to be `json:"label"`, without the escaped quotes. Because of this error, the JSON marshaler will ignore the tags and use the default behavior, which is to use the struct's field names as-is (e.g., `Label`, `Amount`). This will result in an incorrect API response contract.

**Fix:**

Remove the backslashes from all JSON struct tags.

**Example:**

In `internal/models/graph.go`:

**Incorrect:**
```go
type GraphDataPoint struct {
	Label  string  `json:\"label\"`
	Amount float64 `json:\"amount\"`
}
```

**Correct:**
```go
type GraphDataPoint struct {
	Label  string  `json:"label"`
	Amount float64 `json:"amount"`
}
```

This correction needs to be applied to the following files:
*   `internal/models/graph.go`
*   `internal/models/chart.go`
*   `internal/api/handlers/graph_handler.go` (for the `GraphResponse` struct)

## 🧪 Test Suggestions

Here are 3 critical, edge-case test cases that should be added or verified for the new `GraphService`:

*   **Test with an invalid `period` parameter:** The service explicitly validates the `period` string. A test should pass an invalid value (e.g., `"daily"`, `"quarterly"`, or an empty string `""`) and assert that the function returns the expected `models.ErrInvalidPeriod` error and a `nil` slice. This directly tests the new validation logic.

*   **Test when the repository finds no matching transactions:** A test should be configured where the repository mock is called with a valid `jarID` and `period` but returns an empty slice (`[]models.GraphDataPoint{}`) and no error. The test should verify that the service correctly returns the empty slice and a `nil` error, rather than panicking or returning a `nil` slice.

*   **Test when the underlying repository returns an error:** The service layer is responsible for handling or propagating errors from the repository. A test should mock the `repo.GetExpenseGraphData` call to return a specific error (e.g., a database connection error). The test must then assert that the `graphService` correctly propagates this exact error back to the caller.

