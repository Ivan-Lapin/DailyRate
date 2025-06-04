package storage

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type Storage struct {
	db *sql.DB
}

type Rate float64

type HistoryRateItem struct {
	Date  string
	Value Rate
}

type HistoryRate []HistoryRateItem

func New(storagePath string, logger *zap.Logger) (*Storage, error) {
	const op = "storage.postgres.New"

	db, err := sql.Open("postgres", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if err := db.Ping(); err != nil {
		logger.Error("failed to ping database", zap.Error(err))
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	stmt, err := db.Prepare(`
	CREATE TABLE IF NOT EXISTS dailyRate (
	id SERIAL PRIMARY KEY,
	date VARCHAR(20) NOT NULL,
	value float NOT NULL);
	`)

	if err != nil {
		return nil, fmt.Errorf("#{op}: #{err}")
	}

	_, err = stmt.Exec()

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (st *Storage) SaveRate(date string, rate float64, logger *zap.Logger) error {
	query := `INSERT INTO dailyrate (date, value) VALUES ($1, $2)`
	_, err := st.db.Exec(query, date, rate)
	if err != nil {
		logger.Error("failed to insert data", zap.Error(err))
	}
	return err
}

func (st *Storage) GetRate(date string, logger *zap.Logger) (Rate, bool, error) {
	query := `SELECT value FROM dailyrate WHERE date == $1`
	rows, err := st.db.Query(query, date)
	if err != nil {
		logger.Error("failed to get rate from date", zap.Error(err))
	}

	var rate Rate
	for rows.Next() {

		if err := rows.Scan(&rate); err != nil {
			logger.Error("failed to read row", zap.Error(err))
			return 0.0, false, err
		}
	}

	if err := rows.Err(); err != nil {
		return 0.0, false, fmt.Errorf("string processing error: %w", err)
	}

	if rate == 0.0 {
		return rate, false, err
	}

	return rate, true, err
}

func (st *Storage) GetHistory(logger *zap.Logger) (HistoryRate, error) {
	query := `SELECT date, value FROM dailyrate`
	rows, err := st.db.Query(query)
	if err != nil {
		logger.Error("failed to get history rate", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var history HistoryRate
	for rows.Next() {
		var item HistoryRateItem
		if err := rows.Scan(&item.Date, &item.Value); err != nil {
			logger.Error("failed to read row", zap.Error(err))
			return nil, err
		}
		history = append(history, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed parsing rows: %w", err)
	}

	return history, nil
}
