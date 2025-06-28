package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func LoadConfig(logger *zap.Logger) (*ConfigParam, error) {
	exePath, err := os.Executable()
	if err != nil {
		err := fmt.Errorf("cannot get executable path: %w", err)
		logger.Fatal("fatal in Executable", zap.Error(err))
	}
	configPath := filepath.Join(filepath.Dir(exePath), "configs", "config.example.yaml")
	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		err = fmt.Errorf("failed to load the configuration file from disk: %w", err)
		logger.Fatal("fatal in Read In Config", zap.Error(err))
	}

	config := &ConfigParam{
		CurrencyAPI: viper.GetString("currency_api_url"),
		GRPCPort:    viper.GetString("grpc_port"),
		HTTPPort:    viper.GetString("http_port"),
		ConnDB:      viper.GetString("connectDB"),
		NameDB:      viper.GetString("nameDB"),
	}

	logger.Info("Configuration loaded", zap.Any("config", config))

	return config, nil
}

type ConfigParam struct {
	CurrencyAPI string
	GRPCPort    string
	HTTPPort    string
	ConnDB      string
	NameDB      string
}
