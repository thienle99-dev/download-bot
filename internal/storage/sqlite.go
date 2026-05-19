package storage

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func InitDB(dbPath string) (*DB, error) {
	// Open the SQLite database using modernc.org/sqlite (no-CGO driver)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Configure SQLite for production-level concurrency and safety
	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA busy_timeout=5000;",
		"PRAGMA foreign_keys=ON;",
		"PRAGMA synchronous=NORMAL;",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return nil, fmt.Errorf("failed to execute pragma (%s): %w", pragma, err)
		}
	}

	// Create tables
	if err := createSchema(db); err != nil {
		return nil, fmt.Errorf("failed to create database schema: %w", err)
	}

	log.Printf("Database initialized successfully at %s (WAL mode enabled)", dbPath)
	return &DB{db}, nil
}

func createSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS download_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		chat_id INTEGER NOT NULL,
		url TEXT NOT NULL,
		platform TEXT NOT NULL,
		title TEXT NOT NULL,
		format TEXT NOT NULL,
		file_size INTEGER NOT NULL,
		file_path TEXT NOT NULL,
		file_id TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_history_user ON download_history(user_id);
	CREATE INDEX IF NOT EXISTS idx_history_url ON download_history(url);

	CREATE TABLE IF NOT EXISTS sticker_sets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		name TEXT UNIQUE NOT NULL,
		title TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_stickers_user ON sticker_sets(user_id);
	`
	_, err := db.Exec(schema)
	return err
}
