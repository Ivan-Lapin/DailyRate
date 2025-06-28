package db

//not used

import (
	"database/sql"
	"fmt"

	"go.uber.org/zap"
)

type DailyRateDB struct {
	DB *sql.DB
}

type Rate float64

type HistoryRate struct {
	Data  string
	Value Rate
}

func NewDailyRateDB(db *sql.DB) *DailyRateDB {
	return &DailyRateDB{
		DB: db,
	}
}

func (drdb *DailyRateDB) SaveRate(date string, rate float64, logger *zap.Logger) error {
	query := `INSERT INTO dailyrate (date, value) VALUES ($1, $2)`
	_, err := drdb.DB.Exec(query, date, rate)
	if err != nil {
		logger.Error("failed to insert data", zap.Error(err))
	}
	return err
}

func (drdb *DailyRateDB) GetRate(date string, logger *zap.Logger) (Rate, error) {
	query := `SELECT value FROM dailyrate WHERE date == $1`
	rows, err := drdb.DB.Query(query, date)
	if err != nil {
		logger.Error("failed to get rate from date", zap.Error(err))
	}

	var rate Rate
	for rows.Next() {

		if err := rows.Scan(&rate); err != nil {
			logger.Error("failed to read row", zap.Error(err))
			return 0.0, err
		}
	}

	if err := rows.Err(); err != nil {
		return 0.0, fmt.Errorf("ошибка обработки строк: %w", err)
	}

	return rate, err
}

func (drdb *DailyRateDB) GetHistory(logger *zap.Logger) (HistoryRate, error) {
	query := `SELECT date, value FROM dailyrate`
	rows, err := drdb.DB.Query(query)
	if err != nil {
		logger.Error("failed to get history rate", zap.Error(err))
	}

	hr := HistoryRate{}
	for rows.Next() {

		if err := rows.Scan(&hr.Data, &hr.Value); err != nil {
			logger.Error("failed to read row", zap.Error(err))
			return HistoryRate{}, err
		}
	}

	if err := rows.Err(); err != nil {
		return HistoryRate{}, fmt.Errorf("filed parsing of the rows %w", err)
	}

	return hr, err
}
