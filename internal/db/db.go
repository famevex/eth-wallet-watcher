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

func AddSubscription(db *sql.DB, chatID int64, address string) error {
	query := `
	INSERT INTO subscriptions (chat_id, address)
	VALUES ($1, $2)
	ON CONFLICT (chat_id, address) DO NOTHING;
	`
	result, err := db.Exec(query, chatID, address)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("already exists")
	}
	return nil
}

func RemoveSubscription(db *sql.DB, chatID int64, address string) error {
	query := `
	DELETE FROM subscriptions
	WHERE chat_id = $1 AND address = $2
	`

	_, err := db.Exec(query, chatID, address)
	return err
}

type Subscription struct {
    ChatID  int64
    Address string
}

func GetAllSubscriptions(db *sql.DB) ([]Subscription, error) {
	query := `
	SELECT chat_id, address FROM subscriptions
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		var s Subscription
		if err := rows.Scan(&s.ChatID, &s.Address); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}