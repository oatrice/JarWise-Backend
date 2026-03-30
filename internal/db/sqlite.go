package db

import (
	"database/sql"
	"fmt"
	"jarwise-backend/internal/models"
	"log"
	"strings"
	"time"

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
	schema := `
	CREATE TABLE IF NOT EXISTS users (
	        id TEXT PRIMARY KEY,
	        google_sub TEXT NOT NULL UNIQUE,
	        email TEXT NOT NULL,
	        name TEXT NOT NULL,
	        avatar_url TEXT,
	        created_at DATETIME NOT NULL,
	        updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS user_sessions (
	        id TEXT PRIMARY KEY,
	        user_id TEXT NOT NULL,
	        token_hash TEXT NOT NULL UNIQUE,
	        expires_at DATETIME NOT NULL,
	        created_at DATETIME NOT NULL,
	        last_seen_at DATETIME NOT NULL,
	        FOREIGN KEY(user_id) REFERENCES users(id)
	);
	CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);

	CREATE TABLE IF NOT EXISTS wallets (
	        id TEXT PRIMARY KEY,
	        user_id TEXT NOT NULL DEFAULT 'legacy-local-user',
	        name TEXT NOT NULL,
	        currency TEXT NOT NULL,
	        balance REAL DEFAULT 0.0,
	        type TEXT
	);

	CREATE TABLE IF NOT EXISTS jars (
	        id TEXT PRIMARY KEY,
	        user_id TEXT NOT NULL DEFAULT 'legacy-local-user',
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
	        user_id TEXT NOT NULL DEFAULT 'legacy-local-user',
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

	CREATE TABLE IF NOT EXISTS migration_jobs (
	        id TEXT PRIMARY KEY,
	        user_id TEXT NOT NULL,
	        phase TEXT NOT NULL,
	        message TEXT,
	        mmbak_path TEXT,
	        xls_path TEXT,
	        counts_json TEXT,
	        validation_errors_json TEXT,
	        duplicate_summary_json TEXT,
	        can_confirm_import INTEGER NOT NULL DEFAULT 0,
	        expires_at DATETIME,
	        created_at DATETIME NOT NULL,
	        updated_at DATETIME NOT NULL,
	        FOREIGN KEY(user_id) REFERENCES users(id)
	);
	CREATE INDEX IF NOT EXISTS idx_migration_jobs_user_id ON migration_jobs(user_id);
	CREATE INDEX IF NOT EXISTS idx_migration_jobs_phase ON migration_jobs(phase);

	CREATE TABLE IF NOT EXISTS migration_source_refs (
	        id TEXT PRIMARY KEY,
	        user_id TEXT NOT NULL,
	        source_system TEXT NOT NULL,
	        entity_type TEXT NOT NULL,
	        source_id TEXT NOT NULL,
	        fingerprint TEXT NOT NULL,
	        display_name TEXT,
	        imported_record_id TEXT NOT NULL,
	        created_at DATETIME NOT NULL,
	        FOREIGN KEY(user_id) REFERENCES users(id)
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_migration_source_refs_source
	        ON migration_source_refs(user_id, source_system, entity_type, source_id);
	CREATE INDEX IF NOT EXISTS idx_migration_source_refs_fingerprint
	        ON migration_source_refs(user_id, entity_type, fingerprint);
	`
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	if err := ensureLegacyUser(db); err != nil {
		return fmt.Errorf("failed to ensure legacy user: %w", err)
	}

	if err := ensureUserOwnershipColumns(db); err != nil {
		return fmt.Errorf("failed to ensure user ownership columns: %w", err)
	}

	if err := ensureUserOwnershipIndexes(db); err != nil {
		return fmt.Errorf("failed to ensure user ownership indexes: %w", err)
	}

	log.Println("Database migration completed successfully.")
	return nil
}

func ensureLegacyUser(db *sql.DB) error {
	now := time.Now().UTC()
	_, err := db.Exec(`
		INSERT INTO users (id, google_sub, email, name, avatar_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			email = excluded.email,
			name = excluded.name,
			avatar_url = excluded.avatar_url,
			updated_at = excluded.updated_at
	`, models.DefaultLocalUserID, "legacy-local-user", "legacy@jarwise.local", "Legacy Local User", "", now, now)
	return err
}

func ensureUserOwnershipColumns(db *sql.DB) error {
	tables := []string{"wallets", "jars", "transactions"}
	for _, table := range tables {
		hasColumn, err := tableHasColumn(db, table, "user_id")
		if err != nil {
			return err
		}
		if !hasColumn {
			if _, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN user_id TEXT", table)); err != nil {
				return fmt.Errorf("failed to add user_id to %s: %w", table, err)
			}
		}

		if _, err := db.Exec(fmt.Sprintf(
			"UPDATE %s SET user_id = ? WHERE user_id IS NULL OR user_id = ''",
			table,
		), models.DefaultLocalUserID); err != nil {
			return fmt.Errorf("failed to backfill user_id on %s: %w", table, err)
		}
	}

	return nil
}

func ensureUserOwnershipIndexes(db *sql.DB) error {
	statements := []string{
		`CREATE INDEX IF NOT EXISTS idx_wallets_user_id ON wallets(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_jars_user_id ON jars(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func tableHasColumn(db *sql.DB, tableName, columnName string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if name == columnName {
			return true, nil
		}
	}

	return false, rows.Err()
}
