package handler

import (
	"context"
	"time"

	"github.com/Ivan-Lapin/DailyRate/currency/internal/service"
	"github.com/Ivan-Lapin/DailyRate/proto/currency/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

func (s *Server) GetHistoryRate(ctx context.Context, req *pb.GetHistoryRateRequest) (*pb.GetHistoryRateResponse, error) {
	historyRate, err := s.App.CS.GetAllHistory(s.App.Logger)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get history: %v", err)
	}

	historyMap := make(map[string]float64)
	for _, item := range historyRate {
		historyMap[item.Date] = float64(item.Value)
	}

	return &pb.GetHistoryRateResponse{
		History: historyMap,
	}, nil
}
