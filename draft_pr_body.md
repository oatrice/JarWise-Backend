Closes: https://github.com/owner/repo/issues/68

## 📝 Description

This pull request introduces the backend foundation for the new report filtering feature, as outlined in issue #68. It adds a new API endpoint that allows users to generate transaction reports based on a flexible set of filters, including date ranges, multiple categories (Jars), and multiple accounts (Wallets).

To support this new feature and improve overall data integrity, this PR also significantly enhances the reliability of the `.mmbak` file parser. It now gracefully handles malformed date entries and correctly includes transfer transactions in the dataset. A comprehensive test suite has been added for both the new reporting service and the improved parser to ensure stability and correctness.

**Note:** This PR implements the backend API and logic. The UI components (filter panel, checkboxes, etc.) will be handled in a separate, subsequent PR.

## ✨ Key Changes

### 🚀 Feature Implementation

- **New Report Endpoint:** Added a `GET /api/v1/reports` endpoint to fetch filtered transaction data.
- **Multi-Select Filtering:** The endpoint supports the following query parameters:
    - `start_date` & `end_date`: Filter by a specific date range (defaults to the current month).
    - `jar_ids` (alias `category_ids`): A comma-separated list of category IDs to include.
    - `wallet_ids` (alias `account_ids`): A comma-separated list of account IDs to include.
- **Backend Architecture:** Implemented a clean `Handler -> Service -> Repository` pattern for the reporting feature.
    - `ReportHandler`: Parses API requests and validates parameters.
    - `ReportService`: Contains the core business logic for generating reports.
    - `TransactionRepository`: Extended with a new method to query the database using the `ReportFilter` model.

### 🛠️ Parser Improvements & Testing

- **Improved Parser Reliability:** The `mmbak` parser now handles `NULL` or invalid date strings in the database without crashing, ensuring more robust data ingestion.
- **Inclusion of Transfers:** The parser now correctly identifies and includes transfer transactions (`DO_TYPE = '3'`), making the imported dataset more complete.
- **Comprehensive Parser Test Suite:** A new, extensive test suite (`mmbak_parser_test.go`) has been added to cover various scenarios:
    - Valid files
    - Files with malformed or `NULL` dates
    - Corrupt or empty database files
    - Missing tables
- **Test Data Generation:** A Python script (`generate_test_files.py`) is included to programmatically create the `.mmbak` test files, ensuring tests are reproducible and easy to extend.

## 🧪 How to Test

After running the backend service, you can test the new endpoint using `curl` or any API client.

1.  **Get a report for the current month (default behavior):**
    ```bash
    curl "http://localhost:8080/api/v1/reports"
    ```

2.  **Filter by a specific date range:**
    ```bash
    curl "http://localhost:8080/api/v1/reports?start_date=2024-01-01&end_date=2024-03-31"
    ```

3.  **Filter by multiple categories (Jars):**
    ```bash
    # Assuming category IDs 'cat1' and 'cat3' exist
    curl "http://localhost:8080/api/v1/reports?jar_ids=cat1,cat3"
    ```

4.  **Filter by a single account (Wallet):**
    ```bash
    # Assuming account ID 'acc2' exists
    curl "http://localhost:8080/api/v1/reports?wallet_ids=acc2"
    ```

5.  **Combine all filters:**
    ```bash
    curl "http://localhost:8080/api/v1/reports?start_date=2024-01-01&end_date=2024-01-31&jar_ids=cat1&wallet_ids=acc1,acc2"
    ```