package cron

import (
	"context"
	"fmt"

	"github.com/Ivan-Lapin/DailyRate/currency/internal/config"
	"github.com/Ivan-Lapin/DailyRate/currency/internal/service"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type Scheduler struct {
	cron            *cron.Cron
	ctx             context.Context
	logger          *zap.Logger
	config          *config.ConfigParam
	currencyService service.CurrencyService
}

func NewScheduler(ctx context.Context, logger *zap.Logger, config *config.ConfigParam, currencyService service.CurrencyService) *Scheduler {
	return &Scheduler{
		cron:            cron.New(),
		ctx:             ctx,
		logger:          logger,
		config:          config,
		currencyService: currencyService,
	}
}

func (s *Scheduler) AddCurrencyFetchJob() error {
	if _, err := s.cron.AddFunc("0 0 * * *", func() {
		_, err := s.currencyService.Fetch(s.ctx, s.config, s.logger)
		if err != nil {
			err = fmt.Errorf("failed to get start currency rate: %w", err)
			s.logger.Error("get current currency rate", zap.Error(err))
		}
	}); err != nil {
		s.logger.Fatal("Failed to add cron job", zap.Error(err))
	}

	return nil
}

func (s *Scheduler) Start() {
	s.cron.Start()
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}
