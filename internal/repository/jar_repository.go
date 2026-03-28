package repository

import (
	"context"
	"database/sql"
	"jarwise-backend/internal/models"
)

type JarRepository interface {
	ListAll(ctx context.Context) ([]models.Jar, error)
}

type sqliteJarRepository struct {
	db *sql.DB
}

func NewSQLiteJarRepository(db *sql.DB) JarRepository {
	return &sqliteJarRepository{db: db}
}

func (r *sqliteJarRepository) ListAll(ctx context.Context) ([]models.Jar, error) {
	query := `SELECT id, name, type, parent_id, wallet_id, icon, color FROM jars`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jars []models.Jar
	for rows.Next() {
		var j models.Jar
		var parentID, walletID sql.NullString
		err := rows.Scan(&j.ID, &j.Name, &j.Type, &parentID, &walletID, &j.Icon, &j.Color)
		if err != nil {
			return nil, err
		}
		if parentID.Valid {
			j.ParentID = parentID.String
		}
		if walletID.Valid {
			j.WalletID = walletID.String
		}
		jars = append(jars, j)
	}
	return jars, nil
}
