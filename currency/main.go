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
	"github.com/cenkalti/backoff/v4"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func initConfig(logger *zap.Logger) *configParam {

	viper.SetConfigFile("./internal/config/config.example.yaml")
	if err := viper.ReadInConfig(); err != nil {
		err = fmt.Errorf("failed to set config file: %w", err)
		log.Fatal(err)
	}

	config := &configParam{
		currencyAPI: viper.GetString("currency_api_url"),
		grpcPort:    viper.GetString("grpc_port"),
		httpPort:    viper.GetString("http_port"),
	}

	logger.Info("Configuration loaded",
		zap.String("currency_api_url", config.currencyAPI),
		zap.String("grpc_port", config.grpcPort),
		zap.String("http_port", config.httpPort))

	return config
}

type App struct {
	Config *configParam
	Store  *Store
	Logger *zap.Logger
}

type server struct {
	pb.UnimplementedCurrencyServiceServer
	app *App
}

type configParam struct {
	currencyAPI string
	grpcPort    string
	httpPort    string
}

type Store struct {
	sync.RWMutex
	CurrentRateRUB float64
	RateHistoryRUB map[string]float64
}

type Currency struct {
	Date time.Time
	Val  map[string]float64 `json:"conversion_rates"`
}

func (s *Store) SetCurrentRateRUB(rate Currency, logger *zap.Logger) {
	s.Lock()
	defer s.Unlock()
	s.CurrentRateRUB = rate.Val["RUB"]
	s.RateHistoryRUB[rate.Date.Format("02.01.2006")] = rate.Val["RUB"]
	logger.Info("Periodic RUB rate updated",
		zap.Float64("rate_usd_rub", s.CurrentRateRUB))
}

func FoundUSDT(ctx context.Context, config *configParam, logger *zap.Logger) (Currency, error) {

	var resp *http.Response
	var err error

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = 1 * time.Minute

	err = backoff.Retry(func() error {

		resp, err = http.Get(config.currencyAPI)
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

func (s *server) GetCurrentRate(ctx context.Context, req *pb.GetCurrentRateRequest) (*pb.GetCurrentRateResponse, error) {
	rate, err := FoundUSDT(ctx, s.app.Config, s.app.Logger)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch rate: %v", err)
	}

	s.app.Store.SetCurrentRateRUB(rate, s.app.Logger)

	return &pb.GetCurrentRateResponse{
		Date: time.Now().Format("02.01.2006"),
		Rate: s.app.Store.CurrentRateRUB,
	}, nil

}

func (s *server) GetHistoryRate(ctx context.Context, req *pb.GetHistoryRateRequest) (*pb.GetHistoryRateResponse, error) {
	return &pb.GetHistoryRateResponse{
		History: s.app.Store.RateHistoryRUB,
	}, nil
}

func main() {

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to create zap logger: %v", err)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			log.Printf("Failed to sync logger: %v", err)
		}
	}()

	config := initConfig(logger)

	store := &Store{
		RateHistoryRUB: make(map[string]float64),
	}

	app := &App{
		Config: config,
		Store:  store,
		Logger: logger,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := cron.New()
	defer c.Stop()

	if _, err = c.AddFunc("0 0 * * *", func() {
		rate, err := FoundUSDT(ctx, config, logger)
		if err != nil {
			err = fmt.Errorf("failed to get start currency rate: %w", err)
			logger.Error("get current currency rate", zap.Error(err))
		} else {
			app.Store.SetCurrentRateRUB(rate, logger)
		}
	}); err != nil {
		logger.Fatal("Failed to add cron job", zap.Error(err))
	}

	c.Start()

	rate, err := FoundUSDT(ctx, app.Config, app.Logger)
	if err != nil {
		err = fmt.Errorf("failed to get start currency rate: %w", err)
		logger.Error("get start currency rate", zap.Error(err))
	} else {
		app.Store.SetCurrentRateRUB(rate, logger)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterCurrencyServiceServer(grpcServer, &server{
		app: app,
	})

	lis, err := net.Listen("tcp", config.grpcPort)
	if err != nil {
		err = fmt.Errorf("failed to listen: %w", err)
		logger.Fatal("listen on tcp: %v\n", zap.Error(err))
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
		_, err = w.Write([]byte("OK"))
		if err != nil {
			logger.Error("Failed to write HTTP response", zap.Error(err))

		}
	})

	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			err = fmt.Errorf("failed to listen and serve: %w", err)
			logger.Fatal("listen and serve http: %v\n", zap.Error(err))
		}
	}()

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-sigCtx.Done()
	logger.Info("shutting down...")

	cancel()

	c.Stop()

	grpcServer.GracefulStop()

}
