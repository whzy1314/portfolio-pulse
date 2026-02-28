package db

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, err
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS holdings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ticker TEXT NOT NULL,
		asset_type TEXT NOT NULL,
		quantity REAL NOT NULL,
		avg_cost REAL NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS price_alerts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ticker TEXT NOT NULL,
		asset_type TEXT NOT NULL,
		direction TEXT NOT NULL,
		threshold REAL NOT NULL,
		triggered INTEGER NOT NULL DEFAULT 0,
		triggered_at DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("migrate sqlite: %w", err)
	}
	return nil
}
