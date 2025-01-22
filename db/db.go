package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

func ConnectDB(host, port, user, password, dbname string) *sql.DB {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Помилка підключення до бази даних: %v", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("База даних недоступна: %v", err)
	}

	log.Println("Підключення до PostgreSQL успішне!")
	return db
}

func AddUser(db *sql.DB, telegramID int64) error {
	query := `
		INSERT INTO users (telegram_id, created_at)
		VALUES ($1, $2)
		ON CONFLICT (telegram_id) DO NOTHING;
	`
	_, err := db.Exec(query, telegramID, time.Now())
	return err
}

func SetUserCurrency(db *sql.DB, telegramID int64, currency string) error {
	query := `
		UPDATE users
		SET currency = $1
		WHERE telegram_id = $2;
	`
	_, err := db.Exec(query, currency, telegramID)
	return err
}

func SetUserMonthlyBudget(db *sql.DB, telegramID int64, budget float64) error {
	query := `
		UPDATE users
		SET monthly_budget = $1
		WHERE telegram_id = $2;
	`
	_, err := db.Exec(query, budget, telegramID)
	return err
}

