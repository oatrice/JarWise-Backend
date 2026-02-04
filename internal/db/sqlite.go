package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// InitDB initializes the SQLite database and runs migrations
func InitDB(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

func runMigrations(db *sql.DB) error {
	// Simple schema migration
	schema := `
	CREATE TABLE IF NOT EXISTS transactions (
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
	CREATE INDEX IF NOT EXISTS idx_related_transaction_id ON transactions(related_transaction_id);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	log.Println("Database migration completed successfully.")
	return nil
}
