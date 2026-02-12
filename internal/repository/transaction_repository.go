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
	ListByDateRange(start, end time.Time) ([]models.Transaction, error)
	Delete(id string) error
	Unlink(id1, id2 string) error
}

type sqliteTransactionRepository struct {
	db *sql.DB
}

func NewSQLiteTransactionRepository(db *sql.DB) TransactionRepository {
	return &sqliteTransactionRepository{db: db}
}

func (r *sqliteTransactionRepository) Create(tx *models.Transaction) error {
	query := `INSERT INTO transactions 
		(id, amount, description, date, type, wallet_id, jar_id, related_transaction_id) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query,
		tx.ID, tx.Amount, tx.Description, tx.Date, tx.Type,
		tx.WalletID, tx.JarID, tx.RelatedTransactionID)
	return err
}

func (r *sqliteTransactionRepository) CreateTransfer(expense, income *models.Transaction) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `INSERT INTO transactions 
		(id, amount, description, date, type, wallet_id, jar_id, related_transaction_id) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	// Insert Expense
	_, err = tx.Exec(query,
		expense.ID, expense.Amount, expense.Description, expense.Date, expense.Type,
		expense.WalletID, expense.JarID, expense.RelatedTransactionID)
	if err != nil {
		return fmt.Errorf("failed to insert expense: %w", err)
	}

	// Insert Income
	_, err = tx.Exec(query,
		income.ID, income.Amount, income.Description, income.Date, income.Type,
		income.WalletID, income.JarID, income.RelatedTransactionID)
	if err != nil {
		return fmt.Errorf("failed to insert income: %w", err)
	}

	return tx.Commit()
}

func (r *sqliteTransactionRepository) GetByID(id string) (*models.Transaction, error) {
	query := `SELECT id, amount, description, date, type, wallet_id, jar_id, related_transaction_id 
		FROM transactions WHERE id = ?`

	row := r.db.QueryRow(query, id)

	var tx models.Transaction
	var relatedID sql.NullString
	var jarID sql.NullString

	err := row.Scan(&tx.ID, &tx.Amount, &tx.Description, &tx.Date, &tx.Type,
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
	query := `SELECT id, amount, description, date, type, wallet_id, jar_id, related_transaction_id
		FROM transactions
		WHERE date >= ? AND date <= ?
		ORDER BY date DESC`

	rows, err := r.db.Query(query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.Transaction
	for rows.Next() {
		var tx models.Transaction
		var relatedID sql.NullString
		var jarID sql.NullString

		if err := rows.Scan(&tx.ID, &tx.Amount, &tx.Description, &tx.Date, &tx.Type, &tx.WalletID, &jarID, &relatedID); err != nil {
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
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Check for link
	var relatedID sql.NullString
	err = tx.QueryRow("SELECT related_transaction_id FROM transactions WHERE id = ?", id).Scan(&relatedID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil // Already deleted?
		}
		return err
	}

	// 2. Unlink pair if exists
	if relatedID.Valid {
		_, err = tx.Exec("UPDATE transactions SET related_transaction_id = NULL WHERE id = ?", relatedID.String)
		if err != nil {
			return err
		}
	}

	// 3. Delete
	_, err = tx.Exec("DELETE FROM transactions WHERE id = ?", id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *sqliteTransactionRepository) Unlink(id1, id2 string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := "UPDATE transactions SET related_transaction_id = NULL WHERE id = ?"

	if _, err := tx.Exec(query, id1); err != nil {
		return err
	}
	if _, err := tx.Exec(query, id2); err != nil {
		return err
	}

	return tx.Commit()
}
