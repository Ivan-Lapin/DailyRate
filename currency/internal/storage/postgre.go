package storage

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

var (
	dbQueriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_queries_total",
			Help: "Total number of DB queries executed",
		},
		[]string{"query"},
	)

	dbQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Duration of DB queries",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // от 1мс
		},
		[]string{"query"},
	)

	dbQueryErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_query_errors_total",
			Help: "Total number of DB query errors",
		},
		[]string{"query"},
	)
)

func init() {
	prometheus.MustRegister(dbQueriesTotal, dbQueryDuration, dbQueryErrors)
}

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
		logger.Error("fatal to ping to database", zap.Error(err))
		db.Close()
		return nil, fmt.Errorf("failed to ping to database: %w", err)
	}

	stmt, err := db.Prepare(`
	CREATE TABLE IF NOT EXISTS dailyRate (
	id SERIAL PRIMARY KEY,
	date VARCHAR(20) NOT NULL UNIQUE,
	value float NOT NULL);
	`)

	if err != nil {
		return nil, fmt.Errorf("failed to prepare request with creating table: %w", err)
	}

	_, err = stmt.Exec()

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (st *Storage) execWithMetric(queryName string, execFunc func() error) error {
	dbQueriesTotal.WithLabelValues(queryName).Inc()

	timer := prometheus.NewTimer(dbQueryDuration.WithLabelValues(queryName))
	err := execFunc()
	timer.ObserveDuration()

	if err != nil {
		dbQueryErrors.WithLabelValues(queryName)
	}

	return err
}

func (st *Storage) SaveRate(date string, rate float64, logger *zap.Logger) error {
	err := st.execWithMetric("Save Rate", func() error {
		query := `INSERT INTO dailyrate (date, value) VALUES ($1, $2)`
		_, err := st.db.Exec(query, date, rate)
		if err != nil {
			logger.Error("failed to insert data", zap.Error(err))
			return fmt.Errorf("failed to insert data: %w", err)
		}
		return err
	})

	if err != nil {
		return fmt.Errorf("failed method Svare Rate: %w", err)
	}

	return err
}

func (st *Storage) GetRate(date string, logger *zap.Logger) (Rate, bool, error) {

	var rate Rate

	err := st.execWithMetric("Get Rate", func() error {
		query := `SELECT value FROM dailyrate WHERE date = $1 LIMIT 1`
		rows, err := st.db.Query(query, date)
		if err != nil {
			logger.Error("failed to do request with getting rate from the date", zap.Error(err))
			return fmt.Errorf("failed to do request with the date:%w", err)
		}

		for rows.Next() {
			if err := rows.Scan(&rate); err != nil {
				logger.Error("failed to scan row", zap.Error(err))
				return fmt.Errorf("failed to scan row: %w", err)
			}
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("for getting rate after iterating through error: %w", err)
		}

		return nil

	})

	if err != nil {
		return 0.0, false, fmt.Errorf("failed method Gate Rate: %w", err)
	}

	if rate == 0.0 {
		return rate, false, err
	}

	return rate, true, err

}

func (st *Storage) GetHistory(logger *zap.Logger) (HistoryRate, error) {

	var history HistoryRate

	err := st.execWithMetric("Get History", func() error {
		query := `SELECT date, value FROM dailyrate`
		rows, err := st.db.Query(query)
		if err != nil {
			logger.Error("failed to get history rate", zap.Error(err))
			return fmt.Errorf("failed to get history rate: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var item HistoryRateItem
			if err := rows.Scan(&item.Date, &item.Value); err != nil {
				logger.Error("failed to scan row", zap.Error(err))
				return fmt.Errorf("failed to scan row: %w", err)
			}
			history = append(history, item)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("for getting hostory after iterating through error: %w", err)
		}

		return nil
	})

	if err != nil {
		return HistoryRate{}, fmt.Errorf("failed method Get History: %w", err)
	}

	return history, nil
}
