package handler

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/Ivan-Lapin/DailyRate/currency/internal/service"
	"github.com/Ivan-Lapin/DailyRate/proto/currency/pb"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrDateRateNotFound = fmt.Errorf("failed to find rate for the date")

type Server struct {
	pb.UnimplementedCurrencyServiceServer
	App *service.App
}

func (s *Server) GetCurrentRate(ctx context.Context, req *pb.GetCurrentRateRequest) (*pb.GetCurrentRateResponse, error) {
	rate, err := s.App.CS.Fetch(ctx, s.App.Config, s.App.Logger)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch rate: %v", err)
	}

	return &pb.GetCurrentRateResponse{
		Date: time.Now().Format("02.01.2006"),
		Rate: rate.Val["RUB"],
	}, nil

}

func (s *Server) GetRateDate(ctx context.Context, req *pb.GetRateDateRequest) (*pb.GetRateDateResponse, error) {
	rateForDate, exist, err := s.App.CS.GetRateForDate(req.Date, s.App.Logger)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get rate for the date: %v", err)
	} else if !exist {
		return nil, ErrDateRateNotFound
	}
	return &pb.GetRateDateResponse{
		Rate: float64(rateForDate),
	}, err
}

func (s *Server) GetHistoryRate(ctx context.Context, req *pb.GetHistoryRateRequest) (*pb.GetHistoryRateResponse, error) {
	historyRate, err := s.App.CS.GetAllHistory(s.App.Logger)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get history: %v", err)
	}

	// Преобразуем storage.HistoryRate в []*pb.HistoryRateItem
	historyItems := make([]*pb.HistoryRateItem, 0, len(historyRate))
	for _, item := range historyRate {
		// Валидация формата даты
		if !regexp.MustCompile(`^\d{2}\.\d{2}\.\d{4}$`).MatchString(item.Date) {
			s.App.Logger.Warn("invalid date format in history", zap.String("date", item.Date))
			continue // Пропускаем некорректные даты
		}
		historyItems = append(historyItems, &pb.HistoryRateItem{
			Date: item.Date,
			Rate: float64(item.Value),
		})
	}

	return &pb.GetHistoryRateResponse{
		History: historyItems,
	}, nil
}
