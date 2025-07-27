package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "net/http/pprof"

	"github.com/Ivan-Lapin/DailyRate/currency/cmd/cron"
	"github.com/Ivan-Lapin/DailyRate/currency/internal/config"
	"github.com/Ivan-Lapin/DailyRate/currency/internal/handler"
	"github.com/Ivan-Lapin/DailyRate/currency/internal/service"
	"github.com/Ivan-Lapin/DailyRate/currency/internal/storage"
	"github.com/Ivan-Lapin/DailyRate/proto/currency/pb"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	// сгенерированные доки, важно: путь по модулю
	httpSwagger "github.com/swaggo/http-swagger" // подключаем swagger UI
)

//go:generate swag init --output docs

func main() {

	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.TimeKey = "" // убрать timestamp
	logger, err := cfg.Build()
	if err != nil {
		log.Fatal("failed to build constructs a logger from the config and options", zap.Error(err))
	}

	config, err := config.LoadConfig(logger)
	if err != nil {
		log.Fatal("failed to load config %w", zap.Error(err))
	}

	db_postgreSQL, err := storage.New(config.ConnDB, logger)
	if err != nil {
		logger.Fatal("failed to create/connect to DB: ", zap.Error(err))
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
		logger.Info("gRPC server started", zap.String("port", config.GRPCPort))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("failed to serve grpc: %v\n", zap.Error(err))
		}
	}()

	httpServer := &http.Server{
		Addr: config.HTTPPort,
	}

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		// @Summary Проверка здоровья сервиса
		// @Description Возвращает 200 OK, если сервис работает
		// @Tags health
		// @Success 200 {string} string "OK"
		// @Router /healthz [get]
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("OK"))
		if err != nil {
			logger.Error("Failed to write HTTP response: %v/n", zap.Error(err))

		}
	})
	http.Handle("/swagger/", httpSwagger.WrapHandler)

	go func() {
		logger.Info("http server started", zap.String("port", config.HTTPPort))
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
