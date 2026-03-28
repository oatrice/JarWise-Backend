# Implementation Plan - Financial Reports Improvements & Audit

This plan addresses several issues identified in the Financial Reports feature across Android, Backend, and Web repositories.

## Phase 1: Android Stability (Completed)
*Done: Fixed concurrency in ViewModel, improved error handling, and added UI safety (division-by-zero).*

## Phase 2: Reports Audit & Stability (Backend & Web)

This phase addresses logic errors, potential memory leaks, and performance issues identified in the audit of the code changes in the failed prompt logs (#59).

### User Review Required

> [!IMPORTANT]
> **Comparison Period Logic Change:**
> The current Backend logic uses `AddDate(0, -1, 0)` which is simplistic. I will refactor it to use a "duration-based offset". If a user views the last 7 days, it will compare against the *previous* 7 days, avoiding calendar mismatch issues.

> [!WARNING]
> **Web Performance:**
> The `metrics` array in `ReportsPage.tsx` currently resets the `setInterval` on every render. I will memoize it with `useMemo`.

### Proposed Changes

#### [Component] Backend (Go)

##### [MODIFY] [report_service.go](file:///Users/oatrice/Software-projects/JarWise/Backend/internal/service/report_service.go)
- Refactor `GenerateReport` comparison logic to handle arbitrary durations correctly.
- Add safety checks for empty datasets in aggregation.

##### [MODIFY] [transaction_repository.go](file:///Users/oatrice/Software-projects/JarWise/Backend/internal/repository/transaction_repository.go)
- Ensure consistent `rows.Close()` usage across all query methods.

#### [Component] Web (React)

##### [MODIFY] [ReportsPage.tsx](file:///Users/oatrice/Software-projects/JarWise/Web/src/pages/ReportsPage.tsx)
- Memoize `metrics` array to fix interval reset issue.
- Add error boundary or try/catch feedback in the UI.

## Open Questions

- Should the backend comparison logic automatically shift to "Year-over-Year" if the range is 1 year? (Assuming "Previous Period" of same duration for now).

## Verification Plan

### Automated Tests
- **Backend**: Update `report_service_test.go` with a "Yearly Range" comparison test.
- **Backend**: Run `go test ./internal/service/...`.

### Manual Verification
- **Web UI**: Verify the "Summary Badge" rotation in the top right is smooth and doesn't flicker/reset on data changes.
- **Web UI**: Verify that the "Previous Period" correctly aligns when using a custom range of 10 days.
