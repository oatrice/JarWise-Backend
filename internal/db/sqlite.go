package db

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// InitDB initializes the SQLite database and runs migrations
func InitDB(dataSourceName string) (*sql.DB, error) {
	// Add query parameter to enable foreign keys if not already there
	if dataSourceName != ":memory:" && !contains(dataSourceName, "_foreign_keys") {
		if contains(dataSourceName, "?") {
			dataSourceName += "&_foreign_keys=on"
		} else {
			dataSourceName += "?_foreign_keys=on"
		}
	}

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
	CREATE TABLE IF NOT EXISTS wallets (
	        id TEXT PRIMARY KEY,
	        name TEXT NOT NULL,
	        currency TEXT NOT NULL,
	        balance REAL DEFAULT 0.0,
	        type TEXT
	);

	CREATE TABLE IF NOT EXISTS jars (
	        id TEXT PRIMARY KEY,
	        name TEXT NOT NULL,
	        type TEXT NOT NULL,
	        parent_id TEXT,
	        wallet_id TEXT,
	        icon TEXT,
	        color TEXT,
	        FOREIGN KEY(parent_id) REFERENCES jars(id),
	        FOREIGN KEY(wallet_id) REFERENCES wallets(id)
	);
	CREATE TABLE IF NOT EXISTS transactions (
	        id TEXT PRIMARY KEY,
	        amount REAL NOT NULL,
	        description TEXT,
	        date DATETIME NOT NULL,
	        type TEXT NOT NULL,
	        wallet_id TEXT NOT NULL,
	        jar_id TEXT,
	        related_transaction_id TEXT,
	        FOREIGN KEY(wallet_id) REFERENCES wallets(id),
	        FOREIGN KEY(jar_id) REFERENCES jars(id),
	        FOREIGN KEY(related_transaction_id) REFERENCES transactions(id)
	);
	CREATE INDEX IF NOT EXISTS idx_related_transaction_id ON transactions(related_transaction_id);
	CREATE INDEX IF NOT EXISTS idx_wallet_id ON transactions(wallet_id);
	CREATE INDEX IF NOT EXISTS idx_jar_id ON transactions(jar_id);
	`
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	log.Println("Database migration completed successfully.")
	return nil
}
