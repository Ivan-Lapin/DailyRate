package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Ivan-Lapin/DailyRate/currency/internal/config"
	"github.com/Ivan-Lapin/DailyRate/currency/internal/repository"
	"github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"
)

type CurrencyService interface {
	Fetch(ctx context.Context, config *config.ConfigParam, logger *zap.Logger) (Currency, error)
	GetRateForDate(date string) (float64, bool)
	GetAllHistory() map[string]float64
}

type currencyService struct {
	repo repository.Repository
}

func (cs currencyService) Fetch(ctx context.Context, config *config.ConfigParam, logger *zap.Logger) (Currency, error) {
	var resp *http.Response
	var err error

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = 1 * time.Minute

	err = backoff.Retry(func() error {

		resp, err = http.Get(config.CurrencyAPI)
		if err != nil {
			err = fmt.Errorf("failed to fetch currency data: %w", err)
			logger.Error("found usdt", zap.Error(err))
			return err
		}

		return nil
	}, b)

	if err != nil {
		err = fmt.Errorf("failed to fetch currency data: %w", err)
		logger.Error("Failed to fetch currency data", zap.Error(err))
		return Currency{}, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("failed to read response from API: %w", err)
		logger.Error("read response from API", zap.Error(err))
		return Currency{}, err
	}

	rate := &Currency{}
	rate.Date = time.Now()
	err = json.Unmarshal(body, rate)
	if err != nil {
		err = fmt.Errorf("failed to Unmarshal JSON data: %w", err)
		logger.Error("Unmarshal JSON", zap.Error(err))
		return Currency{}, err
	}

	cs.repo.Save(rate.Date.Format("02.10.2024"), rate.Val["RUB"])

	return *rate, nil
}

func (cs currencyService) GetAllHistory() map[string]float64 {
	return cs.repo.All()
}

func (cs currencyService) GetRateForDate(date string) (float64, bool) {
	return cs.repo.Get(date)
}

func NewCurrencyService(repo repository.Repository) CurrencyService {
	return currencyService{repo: repo}
}

type App struct {
	Config *config.ConfigParam
	Logger *zap.Logger
	CS     CurrencyService
}

type Currency struct {
	Date time.Time
	Val  map[string]float64 `json:"conversion_rates"`
}

func NewApp(config *config.ConfigParam, logger *zap.Logger, cs CurrencyService) *App {
	return &App{
		Config: config,
		Logger: logger,
		CS:     cs,
	}
}
