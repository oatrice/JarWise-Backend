package repository

import (
	"database/sql"
	"fmt"
	"jarwise-backend/internal/models"
)

type WalletRepository interface {
	Create(wallet *models.Wallet) error
	Get(id string) (*models.Wallet, error)
	GetForUser(userID, id string) (*models.Wallet, error)
	Delete(id string) error
	DeleteForUser(userID, id string) error
	// To satisfy Data Integrity Requirement
	DeleteWithReplacement(id string, replacementWalletID string) error
	DeleteWithReplacementForUser(userID, id string, replacementWalletID string) error
	DeleteCascade(id string) error
	DeleteCascadeForUser(userID, id string) error
	ListAll() ([]models.Wallet, error)
	ListAllForUser(userID string) ([]models.Wallet, error)
}

type sqliteWalletRepository struct {
	db *sql.DB
}

func NewSQLiteWalletRepository(db *sql.DB) WalletRepository {
	return &sqliteWalletRepository{db: db}
}

func (r *sqliteWalletRepository) Create(w *models.Wallet) error {
	w.UserID = normalizedUserID(w.UserID)
	query := `INSERT INTO wallets (id, user_id, name, currency, balance, type) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, w.ID, w.UserID, w.Name, w.Currency, w.Balance, w.Type)
	return err
}

func (r *sqliteWalletRepository) Get(id string) (*models.Wallet, error) {
	return r.getByQuery(`SELECT id, user_id, name, currency, balance, type FROM wallets WHERE id = ?`, id)
}

func (r *sqliteWalletRepository) GetForUser(userID, id string) (*models.Wallet, error) {
	return r.getByQuery(`SELECT id, user_id, name, currency, balance, type FROM wallets WHERE user_id = ? AND id = ?`, normalizedUserID(userID), id)
}

func (r *sqliteWalletRepository) getByQuery(query string, args ...interface{}) (*models.Wallet, error) {
	w := &models.Wallet{}
	err := r.db.QueryRow(query, args...).Scan(&w.ID, &w.UserID, &w.Name, &w.Currency, &w.Balance, &w.Type)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return w, err
}

// Initial Delete implementation (will fail integrity check in TDD Red)
func (r *sqliteWalletRepository) Delete(id string) error {
	_, err := r.db.Exec("DELETE FROM wallets WHERE id = ?", id)
	return err
}

func (r *sqliteWalletRepository) DeleteForUser(userID, id string) error {
	_, err := r.db.Exec("DELETE FROM wallets WHERE user_id = ? AND id = ?", normalizedUserID(userID), id)
	return err
}

func (r *sqliteWalletRepository) DeleteWithReplacement(id string, replacementWalletID string) error {
	return r.deleteWithReplacement("", id, replacementWalletID, false)
}

func (r *sqliteWalletRepository) DeleteWithReplacementForUser(userID, id string, replacementWalletID string) error {
	return r.deleteWithReplacement(normalizedUserID(userID), id, replacementWalletID, true)
}

func (r *sqliteWalletRepository) deleteWithReplacement(userID, id string, replacementWalletID string, scoped bool) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Move Jars to the replacement wallet
	queryJars := "UPDATE jars SET wallet_id = ? WHERE wallet_id = ?"
	argsJars := []interface{}{replacementWalletID, id}
	if scoped {
		queryJars += " AND user_id = ?"
		argsJars = append(argsJars, userID)
	}
	_, err = tx.Exec(queryJars, argsJars...)
	if err != nil {
		return fmt.Errorf("failed to re-assign jars: %w", err)
	}

	// 2. Move Transactions to the replacement wallet
	queryTransactions := "UPDATE transactions SET wallet_id = ? WHERE wallet_id = ?"
	argsTransactions := []interface{}{replacementWalletID, id}
	if scoped {
		queryTransactions += " AND user_id = ?"
		argsTransactions = append(argsTransactions, userID)
	}
	_, err = tx.Exec(queryTransactions, argsTransactions...)
	if err != nil {
		return fmt.Errorf("failed to re-assign transactions: %w", err)
	}

	// 3. Delete the original wallet
	deleteQuery := "DELETE FROM wallets WHERE id = ?"
	deleteArgs := []interface{}{id}
	if scoped {
		deleteQuery += " AND user_id = ?"
		deleteArgs = append(deleteArgs, userID)
	}
	_, err = tx.Exec(deleteQuery, deleteArgs...)
	if err != nil {
		return fmt.Errorf("failed to delete wallet: %w", err)
	}

	return tx.Commit()
}

func (r *sqliteWalletRepository) DeleteCascade(id string) error {
	return r.deleteCascade("", id, false)
}

func (r *sqliteWalletRepository) DeleteCascadeForUser(userID, id string) error {
	return r.deleteCascade(normalizedUserID(userID), id, true)
}

func (r *sqliteWalletRepository) deleteCascade(userID, id string, scoped bool) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Delete associated Transactions
	txQuery := "DELETE FROM transactions WHERE wallet_id = ?"
	txArgs := []interface{}{id}
	if scoped {
		txQuery += " AND user_id = ?"
		txArgs = append(txArgs, userID)
	}
	_, err = tx.Exec(txQuery, txArgs...)
	if err != nil {
		return fmt.Errorf("failed to cascade delete transactions: %w", err)
	}

	// 2. Delete associated Jars
	jarQuery := "DELETE FROM jars WHERE wallet_id = ?"
	jarArgs := []interface{}{id}
	if scoped {
		jarQuery += " AND user_id = ?"
		jarArgs = append(jarArgs, userID)
	}
	_, err = tx.Exec(jarQuery, jarArgs...)
	if err != nil {
		return fmt.Errorf("failed to cascade delete jars: %w", err)
	}

	// 3. Delete the Wallet
	walletQuery := "DELETE FROM wallets WHERE id = ?"
	walletArgs := []interface{}{id}
	if scoped {
		walletQuery += " AND user_id = ?"
		walletArgs = append(walletArgs, userID)
	}
	_, err = tx.Exec(walletQuery, walletArgs...)
	if err != nil {
		return fmt.Errorf("failed to delete wallet: %w", err)
	}

	return tx.Commit()
}
func (r *sqliteWalletRepository) ListAll() ([]models.Wallet, error) {
	query := `SELECT id, user_id, name, currency, balance, type FROM wallets`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wallets []models.Wallet
	for rows.Next() {
		var w models.Wallet
		if err := rows.Scan(&w.ID, &w.UserID, &w.Name, &w.Currency, &w.Balance, &w.Type); err != nil {
			return nil, err
		}
		wallets = append(wallets, w)
	}
	return wallets, nil
}

func (r *sqliteWalletRepository) ListAllForUser(userID string) ([]models.Wallet, error) {
	query := `SELECT id, user_id, name, currency, balance, type FROM wallets WHERE user_id = ?`
	rows, err := r.db.Query(query, normalizedUserID(userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wallets []models.Wallet
	for rows.Next() {
		var w models.Wallet
		if err := rows.Scan(&w.ID, &w.UserID, &w.Name, &w.Currency, &w.Balance, &w.Type); err != nil {
			return nil, err
		}
		wallets = append(wallets, w)
	}
	return wallets, nil
}
