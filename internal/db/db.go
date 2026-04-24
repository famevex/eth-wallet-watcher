package db

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)

func Connect(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("Fail open db: %w", err)
	}

	// checking connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("Fail connection db: %w", err)
	}

	return db, nil
}

func RunMigrations(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS subscriptions (
		id        SERIAL PRIMARY KEY,
		chat_id   BIGINT NOT NULL,
		address   TEXT NOT NULL,
		UNIQUE(chat_id, address)
	);
	`
	_, err := db.Exec(query)
	return err
}