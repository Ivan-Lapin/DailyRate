package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Ivan-Lapin/DailyRate/currency/internal/config"
	"github.com/Ivan-Lapin/DailyRate/currency/internal/storage"
	"github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"
)

type CurrencyService interface {
	Fetch(ctx context.Context, config *config.ConfigParam, logger *zap.Logger) (Currency, error)
	GetRateForDate(date string, logger *zap.Logger) (storage.Rate, bool, error)
	GetAllHistory(logger *zap.Logger) (storage.HistoryRate, error)
}

type currencyService struct {
	// repo    repository.Repository
	storage *storage.Storage
	client  *http.Client
}

type App struct {
	Config *config.ConfigParam
	Logger *zap.Logger
	CS     CurrencyService
}

type Currency struct {
	Date string             `json:"time_last_update_utc"`
	Val  map[string]float64 `json:"conversion_rates"`
}

func (cs currencyService) NewRetryGet(b *backoff.ExponentialBackOff, logger *zap.Logger, config *config.ConfigParam, resp *http.Response) (*http.Response, error) {
	err_retry := backoff.Retry(func() error {
		var err error
		resp, err = cs.client.Get(config.CurrencyAPI)
		if err != nil {
			err = fmt.Errorf("failed to fetch currency data: %w", err)
			logger.Error("found usdt", zap.Error(err))
			return err
		}

		return nil
	}, b)

	return resp, err_retry
}

func (cs currencyService) Fetch(ctx context.Context, config *config.ConfigParam, logger *zap.Logger) (Currency, error) {
	var resp *http.Response
	var err error

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = 1 * time.Minute

	resp, err = cs.NewRetryGet(b, logger, config, resp)
	if err != nil {
		logger.Error("failed to fetch currency data", zap.Error(err))
		return Currency{}, fmt.Errorf("failed to fetch currency data: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("failed to read response from API", zap.Error(err))
		return Currency{}, fmt.Errorf("failed to read response from API: %w", err)
	}

	// Декодируем ответ API
	var apiResponse struct {
		Result             string             `json:"result"`
		TimeLastUpdateUTC  string             `json:"time_last_update_utc"`
		TimeLastUpdateUnix int64              `json:"time_last_update_unix"`
		BaseCode           string             `json:"base_code"`
		ConversionRates    map[string]float64 `json:"conversion_rates"`
	}

	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		logger.Error("failed to Unmarshal JSON", zap.Error(err))
		return Currency{}, fmt.Errorf("failed to Unmarshal JSON data: %w", err)
	}

	// Парсим дату из API
	apiDate, err := time.Parse(time.RFC1123, apiResponse.TimeLastUpdateUTC)
	if err != nil {
		logger.Error("failed to parse API date", zap.String("api_date", apiResponse.TimeLastUpdateUTC), zap.Error(err))
		return Currency{}, fmt.Errorf("failed to parse API date: %w", err)
	}
	formattedDate := apiDate.Format("02.01.2006")

	// Проверяем наличие курса RUB
	rateVal, ok := apiResponse.ConversionRates["RUB"]
	if !ok {
		logger.Error("RUB rate not found in API response")
		return Currency{}, fmt.Errorf("RUB rate not found in API response")
	}

	currency := Currency{
		Date: formattedDate,
		Val:  apiResponse.ConversionRates,
	}

	// cs.repo.Save(rate.Date, rate.Val["RUB"])
	if err := cs.storage.SaveRate(formattedDate, rateVal, logger); err != nil {
		logger.Error("failed to save rate", zap.String("date", formattedDate), zap.Float64("rate", rateVal), zap.Error(err))
		return currency, nil
	}

	return currency, nil
}

func (cs currencyService) GetAllHistory(logger *zap.Logger) (storage.HistoryRate, error) {
	return cs.storage.GetHistory(logger)
	// return cs.repo.All(), historyRate
}

func (cs currencyService) GetRateForDate(date string, logger *zap.Logger) (storage.Rate, bool, error) {
	return cs.storage.GetRate(date, logger)
}

func NewCurrencyService(storage storage.Storage) CurrencyService {
	return currencyService{storage: &storage, client: &http.Client{
		Timeout: 10 * time.Second,
	}}
}

func NewApp(config *config.ConfigParam, logger *zap.Logger, cs CurrencyService) *App {
	return &App{
		Config: config,
		Logger: logger,
		CS:     cs,
	}
}
