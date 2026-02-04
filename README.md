# JarWise Backend

The backend service for JarWise, built with Go. Currently focuses on data migration and core logic processing.

## 🚀 Getting Started

### Prerequisites
- Go 1.21 or higher
- SQLite3 (for `go-sqlite3` driver)

### Installation

```bash
cd backend
go mod tidy
```

### Running the Server

```bash
go run cmd/server/main.go
```
The server will start on `http://localhost:8080`.

## 📡 API Endpoints

### Data Migration

**POST** `/api/v1/migrations/money-manager`

Uploads Money Manager backup files for migration.

- **Content-Type**: `multipart/form-data`
- **Params**:
  - `mmbak_file`: The SQLite backup file (`.mmbak`).
  - `xls_file`: The Excel export file (`.xls` / HTML format).

**Response:**
```json
{
  "status": "success",
  "message": "Validation successful",
  "job_id": "job-uuid-123"
}
```

## 📂 Project Structure

- `cmd/server`: Application entry point.
- `internal/api`: HTTP handlers and routing.
- `internal/service`: Business logic orchestration.
- `internal/parser`: File parsing logic (SQLite, HTML/XLS).
- `internal/validator`: Data validation and comparison logic.
- `internal/importer`: Domain mapping and persistence steps.
- `internal/models`: Data structures and DTOs.

## 🛠 Features

- **Robust Parsing**: Handles specific behaviors of Money Manager's SQLite schema and HTML-based XLS exports.
- **Data Validation**: Automatically detects discrepancies between raw DB data and reported totals.
- **Domain Mapping**: Transformers to convert external data formats into JarWise's standard Wallet/Transaction models.
