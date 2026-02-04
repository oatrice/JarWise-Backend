# Changelog

All notable changes to the JarWise Backend project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
