package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Ivan-Lapin/DailyRate/proto/currency/pb"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	config   *configParam
	TIME_DAY = 24 * time.Hour
	store    *Store
	logger   *zap.Logger
)

type server struct {
	pb.UnimplementedCurrencyServiceServer
}

type Store struct {
	sync.RWMutex
	CurrentRateRUB float64
	RateHistoryRUB map[string]float64
}

type configParam struct {
	currencyAPI string
	grpcPort    string
	httpPort    string
}

type Currency struct {
	Date time.Time
	Val  map[string]float64 `json:"conversion_rates"`
}

func init() {
	store = &Store{
		RateHistoryRUB: make(map[string]float64),
	}
}

func initLogger() {
	logger, _ = zap.NewProduction()
	defer logger.Sync()
}

func initConfig() {
	viper.SetConfigFile("./internal/config/config.example.yaml")
	if err := viper.ReadInConfig(); err != nil {
		err = fmt.Errorf("failed to set config file: %w", err)
		log.Fatal("config error %v", err)
	}

	config = &configParam{}

	config.currencyAPI = viper.GetString("currency_api_url")
	log.Printf("currency API URL: %s", config.currencyAPI)
	config.grpcPort = viper.GetString("grpc_port")
	config.httpPort = viper.GetString("http_port")
}

func FoundUSDT() (Currency, error) {
	resp, err := http.Get(config.currencyAPI)
	if err != nil {
		err = fmt.Errorf("failed to fetch currency data: %w", err)
		logger.Error("found usdt", zap.Error(err))
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

func (s *server) GetCurrentRate(ctx context.Context, req *pb.GetCurrentRateRequest) (*pb.GetCurrentRateResponse, error) {
	rate, err := FoundUSDT()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch rate: %v", err)
	}

	store.Lock()
	store.CurrentRateRUB = rate.Val["RUB"]
	store.RateHistoryRUB[rate.Date.Format("02.01.2006")] = rate.Val["RUB"]
	store.Unlock()

	return &pb.GetCurrentRateResponse{
		Date: time.Now().Format("02.01.2006"),
		Rate: store.CurrentRateRUB,
	}, nil

}

func (s *server) GetHistoryRate(ctx context.Context, req *pb.GetHistoryRateRequest) (*pb.GetHistoryRateResponse, error) {
	return &pb.GetHistoryRateResponse{
		History: store.RateHistoryRUB,
	}, nil
}

func main() {

	initConfig()
	initLogger()

	ticker := time.NewTicker(TIME_DAY)

	defer ticker.Stop()

	rate, err := FoundUSDT()

	if err != nil {
		err = fmt.Errorf("failed to get start currency rate: %w", err)
		logger.Error("get start currency rate", zap.Error(err))
	} else {
		store.Lock()
		store.CurrentRateRUB = rate.Val["RUB"]
		store.Unlock()
	}

	go func() {
		for range ticker.C {
			rate, err := FoundUSDT()
			if err != nil {
				err = fmt.Errorf("failed to get start currency rate: %w", err)
				logger.Error("get current currency rate", zap.Error(err))
			} else {
				store.Lock()
				store.CurrentRateRUB = rate.Val["RUB"]
				store.Unlock()
			}
		}
	}()

	grpcServer := grpc.NewServer()
	pb.RegisterCurrencyServiceServer(grpcServer, &server{})

	lis, err := net.Listen("tcp", config.grpcPort)
	if err != nil {
		err = fmt.Errorf("failed to listen: %w", err)
		logger.Error("listen on tcp: %v\n", zap.Error(err))
		return
	}

	go func() {
		logger.Info("gRPC server started", zap.String("port", "8082"))
		if err := grpcServer.Serve(lis); err != nil {
			err = fmt.Errorf("failed to serve: %w", err)
			logger.Fatal("serve grpc: %v\n", zap.Error(err))
		}
	}()

	httpServer := &http.Server{
		Addr: config.httpPort,
	}

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			err = fmt.Errorf("failed to listen and serve: %w", err)
			logger.Fatal("listen and serve http: %v\n", zap.Error(err))
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	logger.Info("shutting down...")
	grpcServer.GracefulStop()

}
