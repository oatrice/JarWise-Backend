# Transaction Linking & Transfers

## Summary

This PR implements a complete transaction management system with support for wallet-to-wallet transfers and transaction linking. The feature enables users to create atomic transfer operations that automatically generate paired expense and income transactions, maintaining referential integrity through bidirectional linking.

## Changes

### Core Features

**Transaction Repository & Database Layer**
- Implemented SQLite-based transaction repository with CRUD operations
- Added `transactions` table schema with support for transaction linking via `related_transaction_id`
- Created atomic `CreateTransfer` operation using database transactions to ensure data consistency
- Implemented smart deletion logic that automatically unlinks related transactions
- Added comprehensive unit tests for repository operations

**Transfer Service**
- Built transfer service layer that orchestrates the creation of linked transaction pairs
- Generates expense (negative amount) and income (positive amount) transactions atomically
- Links transactions bidirectionally using UUIDs for referential integrity
- Handles business logic for wallet-to-wallet transfers with proper validation

**REST API Endpoints**
- Added `POST /api/v1/transfers` endpoint for creating wallet transfers
- Implemented request validation for wallet IDs, amounts, and date formats
- Returns both expense and income transactions in a structured response
- Added mock `/api/wallets` endpoint for manual testing and verification

**Data Migration Enhancements**
- Improved date parsing with better error handling for transaction imports
- Added fallback mechanisms for multiple date formats (RFC3339, YYYY-MM-DD)
- Enhanced error logging for debugging migration issues

### Technical Implementation

**Database Schema**
```sql
CREATE TABLE transactions (
    id TEXT PRIMARY KEY,
    amount REAL NOT NULL,
    description TEXT,
    date DATETIME NOT NULL,
    type TEXT NOT NULL,
    wallet_id TEXT NOT NULL,
    jar_id TEXT,
    related_transaction_id TEXT,
    FOREIGN KEY(related_transaction_id) REFERENCES transactions(id)
);
```

**Key Design Decisions**
- Used database transactions to ensure atomic creation of transfer pairs
- Implemented bidirectional linking to enable navigation in both directions
- Soft unlinking on deletion to maintain data integrity
- Pointer type for `RelatedTransactionID` to distinguish between null and empty string

## Files Changed

- **New Files**: 6 files added (handlers, repository, service, tests, database layer)
- **Modified Files**: 5 files updated (router, importer, domain models)
- **Total Changes**: +1,221 insertions, -134 deletions across 18 files

## Testing

- ✅ Unit tests for atomic transfer creation
- ✅ Unit tests for transaction deletion with unlinking
- ✅ In-memory database testing for repository operations
- ✅ Manual verification support via mock wallet endpoint

## Impact

### User Benefits
- Users can now transfer money between wallets with a single API call
- Transaction pairs are automatically linked, making it easy to trace transfers
- Deleting one transaction safely unlinks its pair, preventing orphaned references

### Developer Benefits
- Clean separation of concerns across layers (handler → service → repository)
- Reusable repository interface for future transaction operations
- Comprehensive test coverage for critical transfer logic
- Foundation for future features like refunds, reimbursements, and transaction reconciliation

## Related

Closes https://github.com/owner/repo/issues/71

## Migration Notes

- Database schema will be automatically applied on application startup
- Existing transactions remain unaffected
- New `related_transaction_id` field defaults to NULL for backward compatibility