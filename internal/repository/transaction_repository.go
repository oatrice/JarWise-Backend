package repository

import (
	"database/sql"
	"fmt"
	"jarwise-backend/internal/models"
	"time"
)

type TransactionRepository interface {
	Create(tx *models.Transaction) error
	CreateTransfer(expense, income *models.Transaction) error
	GetByID(id string) (*models.Transaction, error)
	GetByIDForUser(userID, id string) (*models.Transaction, error)
	ListByDateRange(start, end time.Time) ([]models.Transaction, error)
	ListByDateRangeForUser(userID string, start, end time.Time) ([]models.Transaction, error)
	Delete(id string) error
	DeleteForUser(userID, id string) error
	Unlink(id1, id2 string) error
	UnlinkForUser(userID, id1, id2 string) error
	GetExpenseGraphData(jarID, period string) ([]models.GraphDataPoint, error)
	GetExpenseGraphDataForUser(userID, jarID, period string) ([]models.GraphDataPoint, error)
}

type sqliteTransactionRepository struct {
	db *sql.DB
}

func NewSQLiteTransactionRepository(db *sql.DB) TransactionRepository {
	return &sqliteTransactionRepository{db: db}
}

func (r *sqliteTransactionRepository) Create(tx *models.Transaction) error {
	tx.UserID = normalizedUserID(tx.UserID)
	query := `INSERT INTO transactions 
		(id, user_id, amount, description, date, type, wallet_id, jar_id, related_transaction_id) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query,
		tx.ID, tx.UserID, tx.Amount, tx.Description, tx.Date.UTC(), tx.Type,
		tx.WalletID, tx.JarID, tx.RelatedTransactionID)
	return err
}

func (r *sqliteTransactionRepository) CreateTransfer(expense, income *models.Transaction) error {
	expense.UserID = normalizedUserID(expense.UserID)
	income.UserID = normalizedUserID(income.UserID)
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `INSERT INTO transactions 
		(id, user_id, amount, description, date, type, wallet_id, jar_id, related_transaction_id) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	// Insert Expense
	_, err = tx.Exec(query,
		expense.ID, expense.UserID, expense.Amount, expense.Description, expense.Date.UTC(), expense.Type,
		expense.WalletID, expense.JarID, expense.RelatedTransactionID)
	if err != nil {
		return fmt.Errorf("failed to insert expense: %w", err)
	}

	// Insert Income
	_, err = tx.Exec(query,
		income.ID, income.UserID, income.Amount, income.Description, income.Date.UTC(), income.Type,
		income.WalletID, income.JarID, income.RelatedTransactionID)
	if err != nil {
		return fmt.Errorf("failed to insert income: %w", err)
	}

	return tx.Commit()
}

func (r *sqliteTransactionRepository) GetByID(id string) (*models.Transaction, error) {
	query := `SELECT id, user_id, amount, description, date, type, wallet_id, jar_id, related_transaction_id 
		FROM transactions WHERE id = ?`

	return r.getByQuery(query, id)
}

func (r *sqliteTransactionRepository) GetByIDForUser(userID, id string) (*models.Transaction, error) {
	query := `SELECT id, user_id, amount, description, date, type, wallet_id, jar_id, related_transaction_id 
		FROM transactions WHERE user_id = ? AND id = ?`
	return r.getByQuery(query, normalizedUserID(userID), id)
}

func (r *sqliteTransactionRepository) getByQuery(query string, args ...interface{}) (*models.Transaction, error) {
	row := r.db.QueryRow(query, args...)
	var tx models.Transaction
	var relatedID sql.NullString
	var jarID sql.NullString

	err := row.Scan(&tx.ID, &tx.UserID, &tx.Amount, &tx.Description, &tx.Date, &tx.Type,
		&tx.WalletID, &jarID, &relatedID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if relatedID.Valid {
		tx.RelatedTransactionID = &relatedID.String
	}
	if jarID.Valid {
		tx.JarID = jarID.String
	}

	return &tx, nil
}

func (r *sqliteTransactionRepository) ListByDateRange(start, end time.Time) ([]models.Transaction, error) {
	query := `SELECT id, user_id, amount, description, date, type, wallet_id, jar_id, related_transaction_id
		FROM transactions
		WHERE date >= ? AND date <= ?
		ORDER BY date DESC`

	return r.listByDateRangeQuery(query, start.UTC(), end.UTC())
}

func (r *sqliteTransactionRepository) ListByDateRangeForUser(userID string, start, end time.Time) ([]models.Transaction, error) {
	query := `SELECT id, user_id, amount, description, date, type, wallet_id, jar_id, related_transaction_id
		FROM transactions
		WHERE user_id = ? AND date >= ? AND date <= ?
		ORDER BY date DESC`
	return r.listByDateRangeQuery(query, normalizedUserID(userID), start.UTC(), end.UTC())
}

func (r *sqliteTransactionRepository) listByDateRangeQuery(query string, args ...interface{}) ([]models.Transaction, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.Transaction
	for rows.Next() {
		var tx models.Transaction
		var relatedID sql.NullString
		var jarID sql.NullString

		if err := rows.Scan(&tx.ID, &tx.UserID, &tx.Amount, &tx.Description, &tx.Date, &tx.Type, &tx.WalletID, &jarID, &relatedID); err != nil {
			return nil, err
		}
		if relatedID.Valid {
			tx.RelatedTransactionID = &relatedID.String
		}
		if jarID.Valid {
			tx.JarID = jarID.String
		}
		results = append(results, tx)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func (r *sqliteTransactionRepository) Delete(id string) error {
	return r.deleteByQuery("SELECT related_transaction_id FROM transactions WHERE id = ?", "DELETE FROM transactions WHERE id = ?", []interface{}{id}, []interface{}{id}, false)
}

func (r *sqliteTransactionRepository) DeleteForUser(userID, id string) error {
	normalized := normalizedUserID(userID)
	return r.deleteByQuery(
		"SELECT related_transaction_id FROM transactions WHERE user_id = ? AND id = ?",
		"DELETE FROM transactions WHERE user_id = ? AND id = ?",
		[]interface{}{normalized, id},
		[]interface{}{normalized, id},
		true,
	)
}

func (r *sqliteTransactionRepository) deleteByQuery(selectQuery, deleteQuery string, selectArgs, deleteArgs []interface{}, scoped bool) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Check for link
	var relatedID sql.NullString
	err = tx.QueryRow(selectQuery, selectArgs...).Scan(&relatedID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil // Already deleted?
		}
		return err
	}

	// 2. Unlink pair if exists
	if relatedID.Valid {
		if scoped {
			_, err = tx.Exec("UPDATE transactions SET related_transaction_id = NULL WHERE user_id = ? AND id = ?", selectArgs[0], relatedID.String)
		} else {
			_, err = tx.Exec("UPDATE transactions SET related_transaction_id = NULL WHERE id = ?", relatedID.String)
		}
		if err != nil {
			return err
		}
	}

	// 3. Delete
	_, err = tx.Exec(deleteQuery, deleteArgs...)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *sqliteTransactionRepository) Unlink(id1, id2 string) error {
	return r.unlinkByQuery("UPDATE transactions SET related_transaction_id = NULL WHERE id = ?", []interface{}{id1}, []interface{}{id2})
}

func (r *sqliteTransactionRepository) UnlinkForUser(userID, id1, id2 string) error {
	normalized := normalizedUserID(userID)
	return r.unlinkByQuery(
		"UPDATE transactions SET related_transaction_id = NULL WHERE user_id = ? AND id = ?",
		[]interface{}{normalized, id1},
		[]interface{}{normalized, id2},
	)
}

func (r *sqliteTransactionRepository) unlinkByQuery(query string, args1, args2 []interface{}) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(query, args1...); err != nil {
		return err
	}
	if _, err := tx.Exec(query, args2...); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *sqliteTransactionRepository) GetExpenseGraphData(jarID, period string) ([]models.GraphDataPoint, error) {
	return r.getExpenseGraphDataQuery("", jarID, period)
}

func (r *sqliteTransactionRepository) GetExpenseGraphDataForUser(userID, jarID, period string) ([]models.GraphDataPoint, error) {
	return r.getExpenseGraphDataQuery(normalizedUserID(userID), jarID, period)
}

func (r *sqliteTransactionRepository) getExpenseGraphDataQuery(userID, jarID, period string) ([]models.GraphDataPoint, error) {
	var dateFormat string
	switch period {
	case "weekly":
		dateFormat = "%Y-%W" // Standard ISO week number might be %Y-%V or similar, but %W is week of year (00-53) starting Monday
	case "monthly":
		dateFormat = "%Y-%m"
	case "yearly":
		dateFormat = "%Y"
	default:
		return nil, fmt.Errorf("invalid period: %s", period)
	}

	query := `
		SELECT 
			strftime('` + dateFormat + `', date) as period_label, 
			ABS(SUM(amount)) as total_amount
		FROM transactions 
		WHERE 
			jar_id = ? 
			AND type = 'expense'
	`
	args := []interface{}{jarID}
	if userID != "" {
		query += ` AND user_id = ?`
		args = append(args, userID)
	}
	query += `
		GROUP BY period_label
		ORDER BY period_label ASC
	`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dataPoints []models.GraphDataPoint
	for rows.Next() {
		var label string
		var amount float64
		if err := rows.Scan(&label, &amount); err != nil {
			return nil, err
		}
		dataPoints = append(dataPoints, models.GraphDataPoint{
			Label:  label,
			Amount: amount,
		})
	}

	return dataPoints, nil
}
