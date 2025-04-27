package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Ivan-Lapin/DailyRate/currency/internal/config"
	"github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"
)

type Currency struct {
	Date time.Time
	Val  map[string]float64 `json:"conversion_rates"`
}

func FoundUSDT(ctx context.Context, config *config.ConfigParam, logger *zap.Logger) (Currency, error) {

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

	return *rate, nil
}
