# [Feature] Migrate Data from Money Manager App (.mmbak)

<!-- Paste your generated PR description here -->
Resolves #65

## Summary

This PR implements a complete data migration pipeline for importing user financial data from Money Manager app (.mmbak format) into JarWise. The migration process includes parsing SQLite database files, validating data integrity against Excel reports, and transforming Money Manager's data structure into JarWise's domain models.

## Key Changes

### 📊 Data Parser (`internal/parser/mmbak_parser.go`)
- Enhanced SQLite parsing to correctly handle Money Manager's database schema
- Fixed transaction type mapping with proper support for Income (type 1), Expense (type 0/2), and Transfer (type 2/3)
- Improved field ID handling by switching from `int` to `string` for better UUID compatibility
- Added transfer exclusion logic from income/expense totals to prevent double-counting
- Implemented robust error handling for malformed database queries

### ✅ Data Validator (`internal/validator/`)
- Created validation framework comparing parsed database data against Excel reports
- Implemented tolerance-based float comparison (epsilon: 0.01) for financial accuracy
- Added comprehensive mismatch detection for:
  - Transaction counts (with 100+ diff threshold for errors)
  - Total income/expense amounts
  - Balance calculations
- Generates detailed validation results with errors, warnings, and statistics

### 💾 Data Importer (`internal/importer/importer.go`)
- Built data transformation layer mapping Money Manager models to JarWise domain:
  - `AccountDTO` → `Wallet` (with currency and balance preservation)
  - `CategoryDTO` → `Jar` (with parent-child hierarchy support)
  - `TransactionDTO` → `Transaction` (with proper date parsing and type conversion)
- Implemented mock persistence layer ready for database integration
- Added sample data verification for debugging

### 🔧 Migration Service (`internal/service/migration_service.go`)
- Orchestrated end-to-end migration workflow: Upload → Parse → Validate → Import
- Added `BypassValidation` toggle for development flexibility
- Improved error handling with proper error propagation instead of status wrapping
- Enhanced response messaging based on validation results

### 📦 Domain Models (`internal/models/`)
- Created core JarWise domain models: `Wallet`, `Jar`, `Transaction`
- Updated Money Manager DTOs to use `string` IDs throughout for consistency
- Added support for transfer transactions with `ToWalletID` field

## Migration Pipeline Flow

```
Upload .mmbak + .xls
       ↓
Parse SQLite DB ──→ ParsedData (Accounts, Categories, Transactions)
       ↓
Parse Excel Report ──→ ParsedData (Reference totals)
       ↓
Validate ──→ Compare counts & totals ──→ ValidationResult
       ↓
Import ──→ Transform to domain models ──→ Persist to DB
       ↓
Response (success/error with stats)
```

## Technical Improvements

- **Type Safety**: Changed all ID fields from `int` to `string` for UUID support
- **Data Integrity**: Added validation layer preventing silent data corruption
- **Error Handling**: Proper error propagation with descriptive messages
- **Flexibility**: Configurable validation bypass for testing scenarios
- **Extensibility**: Clean separation of concerns (Parser → Validator → Importer)

## Testing Notes

- Successfully handles large datasets (9000+ transactions tested)
- Properly identifies discrepancies between database and Excel reports
- Transfer transactions correctly excluded from income/expense aggregates
- Date parsing supports multiple format fallbacks (YYYY-MM-DD HH:MM:SS and YYYY-MM-DD)

## Statistics

- **12 files changed**: 601 insertions(+), 44 deletions(-)
- **New packages**: `importer`, `validator`
- **Updated packages**: `parser`, `service`, `models`

## Next Steps

- [ ] Integrate actual database persistence layer
- [ ] Add unit tests for mappers and validators
- [ ] Implement preview endpoint for user data verification
- [ ] Add progress tracking for large imports
- [ ] Support batch processing for multiple migration jobs