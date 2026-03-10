package repository

import (
	"database/sql"
	"fmt"
	"jarwise-backend/internal/models"
)

type WalletRepository interface {
	Create(wallet *models.Wallet) error
	Get(id string) (*models.Wallet, error)
	Delete(id string) error
	// To satisfy Data Integrity Requirement
	DeleteWithReplacement(id string, replacementWalletID string) error
}

type sqliteWalletRepository struct {
	db *sql.DB
}

func NewSQLiteWalletRepository(db *sql.DB) WalletRepository {
	return &sqliteWalletRepository{db: db}
}

func (r *sqliteWalletRepository) Create(w *models.Wallet) error {
	query := `INSERT INTO wallets (id, name, currency, balance, type) VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, w.ID, w.Name, w.Currency, w.Balance, w.Type)
	return err
}

func (r *sqliteWalletRepository) Get(id string) (*models.Wallet, error) {
	w := &models.Wallet{}
	query := `SELECT id, name, currency, balance, type FROM wallets WHERE id = ?`
	err := r.db.QueryRow(query, id).Scan(&w.ID, &w.Name, &w.Currency, &w.Balance, &w.Type)
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

func (r *sqliteWalletRepository) DeleteWithReplacement(id string, replacementWalletID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Move Jars to the replacement wallet
	_, err = tx.Exec("UPDATE jars SET wallet_id = ? WHERE wallet_id = ?", replacementWalletID, id)
	if err != nil {
		return fmt.Errorf("failed to re-assign jars: %w", err)
	}

	// 2. Move Transactions to the replacement wallet
	_, err = tx.Exec("UPDATE transactions SET wallet_id = ? WHERE wallet_id = ?", replacementWalletID, id)
	if err != nil {
		return fmt.Errorf("failed to re-assign transactions: %w", err)
	}

	// 3. Delete the original wallet
	_, err = tx.Exec("DELETE FROM wallets WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete wallet: %w", err)
	}

	return tx.Commit()
}
