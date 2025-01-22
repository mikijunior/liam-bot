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

func AddExpense(db *sql.DB, userID int, amount float64, category, note string) error {
	query := `
		INSERT INTO expenses (user_id, amount, category, note, created_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP);
	`
	_, err := db.Exec(query, userID, amount, category, note)
	return err
}

func GetMonthlyExpenses(db *sql.DB, userID int) (float64, error) {
	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM expenses
		WHERE user_id = $1 AND date_trunc('month', created_at) = date_trunc('month', CURRENT_TIMESTAMP);
	`
	var total float64
	err := db.QueryRow(query, userID).Scan(&total)
	return total, err
}

func GetUserMonthlyBudget(db *sql.DB, userID int64) (float64, error) {
	query := `
		SELECT monthly_budget
		FROM users
		WHERE telegram_id = $1;
	`
	var budget float64
	err := db.QueryRow(query, userID).Scan(&budget)
	return budget, err
}

func GetUserID(db *sql.DB, telegramID int64) (int, error) {
	query := `
		SELECT id
		FROM users
		WHERE telegram_id = $1;
	`
	var userID int
	err := db.QueryRow(query, telegramID).Scan(&userID)
	return userID, err
}

func GetUsersWithBudget(db *sql.DB) ([]int64, error) {
	query := `
		SELECT telegram_id
		FROM users
		WHERE monthly_budget IS NOT NULL AND monthly_budget > 0;
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIDs []int64
	for rows.Next() {
		var telegramID int64
		if err := rows.Scan(&telegramID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, telegramID)
	}

	return userIDs, nil
}

func GetBudgetStatusWithNotification(db *sql.DB, userID int64) (float64, float64, time.Time, error) {
	query := `
		SELECT COALESCE(monthly_budget, 0), COALESCE(SUM(amount), 0), COALESCE(last_notification, '1970-01-01'::timestamp)
		FROM users
		LEFT JOIN expenses ON users.id = expenses.user_id
		WHERE users.telegram_id = $1
		GROUP BY users.id;
	`
	var budget, totalExpenses float64
	var lastNotification time.Time
	err := db.QueryRow(query, userID).Scan(&budget, &totalExpenses, &lastNotification)
	if err != nil {
		return 0, 0, time.Time{}, err
	}

	remaining := budget - totalExpenses
	percentageSpent := (totalExpenses / budget) * 100
	return remaining, percentageSpent, lastNotification, nil
}

func UpdateLastNotification(db *sql.DB, telegramID int64) error {
	query := `
		UPDATE users
		SET last_notification = $1
		WHERE telegram_id = $2;
	`
	_, err := db.Exec(query, time.Now(), telegramID)
	return err
}
