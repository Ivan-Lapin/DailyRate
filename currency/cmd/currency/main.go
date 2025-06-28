package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Ivan-Lapin/DailyRate/currency/cmd/cron"
	"github.com/Ivan-Lapin/DailyRate/currency/internal/config"
	"github.com/Ivan-Lapin/DailyRate/currency/internal/handler"
	"github.com/Ivan-Lapin/DailyRate/currency/internal/service"
	"github.com/Ivan-Lapin/DailyRate/currency/internal/storage"
	"github.com/Ivan-Lapin/DailyRate/proto/currency/pb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to create zap logger: %v", err)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			log.Printf("Failed to sync logger: %v", err)
		}
	}()

	config, err := config.LoadConfig(logger)
	if err != nil {
		logger.Fatal("failed to load config %w", zap.Error(err))
	}

	db_postgreSQL, err := storage.New(config.ConnDB, logger)
	if err != nil {
		logger.Fatal("failed to create/connect to DB: %v\n", zap.Error(err))
	}

	// repo := repository.NewInMemory()

	currencyService := service.NewCurrencyService(*db_postgreSQL)

	app := service.NewApp(config, logger, currencyService)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cronScheduler := cron.NewScheduler(ctx, logger, config, currencyService)
	defer cronScheduler.Stop()

	if err = cronScheduler.AddCurrencyFetchJob(); err != nil {
		logger.Fatal("failed to add cron job: %v\n", zap.Error(err))
	}

	cronScheduler.Start()

	_, err = currencyService.Fetch(ctx, app.Config, app.Logger)
	if err != nil {
		logger.Error("failed to get start currency rate: %v/n", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	pb.RegisterCurrencyServiceServer(grpcServer, &handler.Server{
		App: app,
	})

	lis, err := net.Listen("tcp", config.GRPCPort)
	if err != nil {
		logger.Fatal("failed listen on tcp: %v\n", zap.Error(err))
		return
	}

	go func() {
		logger.Info("gRPC server started", zap.String("port", config.HTTPPort))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("failed to serve grpc: %v\n", zap.Error(err))
		}
	}()

	httpServer := &http.Server{
		Addr: config.HTTPPort,
	}

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("OK"))
		if err != nil {
			logger.Error("Failed to write HTTP response: %v/n", zap.Error(err))

		}
	})

	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			logger.Fatal("listen and serve http: %v\n", zap.Error(err))
		}
	}()

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-sigCtx.Done()
	logger.Info("shutting down...")

	grpcServer.GracefulStop()

}
