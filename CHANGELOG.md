# Changelog

## [0.3.0] - 2026-02-11

### Added
- **API**: Introduced a new transaction report endpoint (`/api/v1/reports/transactions`) with support for date-range filtering.
- **Testing**: Added a comprehensive test suite for the `.mmbak` parser, covering edge cases such as corrupted data, empty files, and invalid dates to improve import reliability.

### Fixed
- **Data Parsing**: Resolved an issue where null or invalid dates in `.mmbak` files would cause the import process to fail.
- **API**: Corrected JSON struct tags in the report model to ensure proper API response formatting.

All notable changes to the JarWise Backend project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-02-04

### Added
- **Transaction Management**: Implemented core functionality for creating and managing financial transactions, including transfers between linked wallets.
- **Data Persistence**: Introduced a `TransactionRepository` for robust data access and persistence in the SQLite database.
- **Data Migration**: Completed the end-to-end data migration pipeline from Money Manager backups (`.mmbak`).
- **API**:
  - Added a new mock wallet endpoint (`/api/v1/wallets/mock`) for development and manual verification.

### Fixed
- **Data Parsing**: Corrected JSON struct tags and improved error handling for date parsing to prevent data import failures.

## [0.1.0] - 2026-02-04

### Added
- **Migration Service**: Initial core logic for migrating data from Money Manager app.
  - SQLite Parser (`.mmbak`) with support for Assets, Categories, and Transactions.
  - XLS Parser (HTML Table) for validating transaction reports.
  - Validation Logic to compare discrepancies between DB and Report.
  - Mock Importer for mapping data to JarWise domain models.
- **API**:
  - `POST /api/v1/migrations/money-manager`: Endpoint to handle file uploads and return migration status.
- **Infrastructure**:
  - Basic Go server setup with `net/http`.
  - Clean Architecture structure (`internal/api`, `internal/service`, `internal/models`).
