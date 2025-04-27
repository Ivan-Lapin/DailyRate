package handler

import (
	"context"
	"sync"
	"time"

	"github.com/Ivan-Lapin/DailyRate/currency/internal/config"
	"github.com/Ivan-Lapin/DailyRate/currency/internal/service"
	"github.com/Ivan-Lapin/DailyRate/proto/currency/pb"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Store struct {
	sync.RWMutex
	CurrentRateRUB float64
	RateHistoryRUB map[string]float64
}

type App struct {
	Config *config.ConfigParam
	Store  *Store
	Logger *zap.Logger
}

type Server struct {
	pb.UnimplementedCurrencyServiceServer
	App *App
}

type Currency struct {
	Date time.Time
	Val  map[string]float64 `json:"conversion_rates"`
}

func NewApp(config *config.ConfigParam, store *Store, logger *zap.Logger) *App {
	return &App{
		Config: config,
		Store:  store,
		Logger: logger,
	}
}

func (s *Store) SetCurrentRateRUB(rate service.Currency, logger *zap.Logger) {
	s.Lock()
	defer s.Unlock()
	s.CurrentRateRUB = rate.Val["RUB"]
	s.RateHistoryRUB[rate.Date.Format("02.01.2006")] = rate.Val["RUB"]
	logger.Info("Periodic RUB rate updated",
		zap.Float64("rate_usd_rub", s.CurrentRateRUB))
}

func NewStore() *Store {
	return &Store{
		RateHistoryRUB: make(map[string]float64),
	}
}

func (s *Server) GetCurrentRate(ctx context.Context, req *pb.GetCurrentRateRequest) (*pb.GetCurrentRateResponse, error) {
	rate, err := service.FoundUSDT(ctx, s.App.Config, s.App.Logger)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch rate: %v", err)
	}

	s.App.Store.SetCurrentRateRUB(rate, s.App.Logger)

	return &pb.GetCurrentRateResponse{
		Date: time.Now().Format("02.01.2006"),
		Rate: s.App.Store.CurrentRateRUB,
	}, nil

}

func (s *Server) GetHistoryRate(ctx context.Context, req *pb.GetHistoryRateRequest) (*pb.GetHistoryRateResponse, error) {
	return &pb.GetHistoryRateResponse{
		History: s.App.Store.RateHistoryRUB,
	}, nil
}
